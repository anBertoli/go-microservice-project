package galleries

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/tracing"
)

func NewGalleriesService(store store.Store, logger *zap.SugaredLogger, concurrency uint) *GalleriesService {
	return &GalleriesService{
		logger: logger,
		sema:   make(chan struct{}, concurrency),
		store:  store,
	}
}

type GalleriesService struct {
	logger *zap.SugaredLogger
	store  store.Store
	sema   chan struct{}
}

func (gs *GalleriesService) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	galleries, metadata, err := gs.store.Galleries.GetAllPublic(filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return galleries, metadata, nil
}

func (gs *GalleriesService) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	authData := store.ContextGetAuth(ctx)

	galleries, metadata, err := gs.store.Galleries.GetAllForUser(authData.User.ID, filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return galleries, metadata, nil
}

func (gs *GalleriesService) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	authData := store.ContextGetAuth(ctx)

	gallery, err := gs.store.Galleries.Insert(store.Gallery{
		UserID:      authData.User.ID,
		Title:       gallery.Title,
		Description: gallery.Description,
		Published:   gallery.Published,
	})
	if err != nil {
		return store.Gallery{}, err
	}
	return gallery, nil
}

func (gs *GalleriesService) Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	authData := store.ContextGetAuth(ctx)

	galleryToUpdate, err := gs.store.Galleries.Get(gallery.ID)
	if err != nil {
		return store.Gallery{}, err
	}
	if authData.User.ID != galleryToUpdate.UserID {
		return store.Gallery{}, ErrForbidden
	}

	gallery, err = gs.store.Galleries.Update(store.Gallery{
		ID:          gallery.ID,
		Title:       gallery.Title,
		Description: gallery.Description,
		Published:   gallery.Published,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Gallery{}, ErrConflict
		default:
			return store.Gallery{}, err
		}
	}

	return gallery, nil
}

func (gs *GalleriesService) Delete(ctx context.Context, galleryID int64) error {
	authData := store.ContextGetAuth(ctx)

	galleryToDelete, err := gs.store.Galleries.Get(galleryID)
	if err != nil {
		return err
	}
	if authData.User.ID != galleryToDelete.UserID {
		return ErrForbidden
	}

	err = gs.store.Galleries.DeleteGallery(galleryID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return ErrConflict
		default:
			return err
		}
	}

	return nil
}

func (gs *GalleriesService) Download(ctx context.Context, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	authData := store.ContextGetAuth(ctx)

	// Get the gallery from the database. If the gallery is public everyone could download
	// it, otherwise the authenticated user must be the owner of the gallery.
	gallery, err := gs.store.Galleries.Get(galleryID)
	if err != nil {
		return store.Gallery{}, nil, err
	}
	if !gallery.Published {
		if authData.User.ID != gallery.UserID {
			return store.Gallery{}, nil, ErrForbidden
		}
	}

	// Iterate over subsequent pages of images collecting all of them. The resulting list of
	// images is needed since we must include them one by one in the archive.
	var images []store.Image
	var page = 1
	for {
		pagImages, pagOut, err := gs.store.Images.GetAllForGallery(galleryID, filters.Input{
			Page:         page,
			PageSize:     100,
			SortCol:      "id",
			SortSafeList: []string{"id"},
			SearchCol:    "title",
		})
		if err != nil {
			return store.Gallery{}, nil, err
		}
		images = append(images, pagImages...)
		if pagOut.CurrentPage == pagOut.LastPage {
			break
		}
		page++
	}

	// Try to acquire a token in the semaphore and continue in case of success. If the operation
	// fails, that is, the current concurrency is reached, return an explicative error that is
	// used to inform the client that the service is currently too busy.
	select {
	case gs.sema <- struct{}{}:
	default:
		return store.Gallery{}, nil, ErrBusy
	}

	// Start a goroutine in charge of streaming the compressed tar archive to the returned reader.
	// It's vital to release the token of the semaphore in any case to release acquired resources.
	// The io.Pipe is necessary to connect the streamImagesArchive func (which needs a writer) to
	// the caller which expects a reader.
	logger := tracing.LoggerWithRequestID(ctx, gs.logger)
	r, w := io.Pipe()

	go func() {
		// Here the deferred function is used to make sue that, independently from
		// the outcome of the job, the writer will be closed and the semaphore is
		// updated.
		defer func() {
			_ = w.Close()
			<-gs.sema
		}()

		err := gs.streamImagesArchive(w, images)

		// This error is originated from the consumer side and we cannot do
		// anything about that, simply drop the job. The deferred func will
		// act normally freeing a token in the semaphore.
		if errors.Is(err, io.ErrClosedPipe) {
			return
		}

		// Real error coming from the internal streaming function. We log the error
		// and we close the write end of the pipe, so that the consumer will receive
		// the error.
		if err != nil {
			logger.Errorw("streaming archive", "err", err)
			_ = w.CloseWithError(err)
		}
	}()

	return gallery, r, nil
}

func (gs *GalleriesService) streamImagesArchive(w io.WriteCloser, images []store.Image) error {
	gzipWriter := gzip.NewWriter(w)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, image := range images {
		readCloser, err := gs.store.Images.GetReader(image.ID)
		if err != nil {
			return err
		}

		imageBytes, err := io.ReadAll(readCloser)
		if err != nil {
			return err
		}
		err = readCloser.Close()
		if err != nil {
			return err
		}
		imageName := image.Title
		if image.Title == "" {
			imageName = filepath.Base(image.Path)
		}

		err = tarWriter.WriteHeader(&tar.Header{
			Size: int64(len(imageBytes)),
			Name: imageName,
			Mode: 0666,
		})
		_, err = io.Copy(tarWriter, bytes.NewReader(imageBytes))
		if err != nil {
			return err
		}
	}

	err := tarWriter.Flush()
	if err != nil {
		return err
	}
	err = tarWriter.Close()
	if err != nil {
		return err
	}
	return gzipWriter.Close()
}
