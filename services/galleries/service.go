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

	"github.com/anBertoli/snap-vault/pkg/auth"
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

// The GalleriesService retrieves and save galleries data in a relation database.
type GalleriesService struct {
	logger *zap.SugaredLogger
	store  store.Store
	sema   chan struct{}
}

// Returns a filtered and paginated list of public galleries.
func (gs *GalleriesService) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	galleries, metadata, err := gs.store.Galleries.GetAllPublic(filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return galleries, metadata, nil
}

// Returns a filtered and paginated list of galleries owned by the authenticated user.
func (gs *GalleriesService) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	authData := auth.MustContextGetAuth(ctx)
	galleries, metadata, err := gs.store.Galleries.GetAllForUser(authData.User.ID, filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return galleries, metadata, nil
}

// Fetch the gallery data, the request could be public or authenticated.
func (gs *GalleriesService) Get(ctx context.Context, public bool, galleryID int64) (store.Gallery, error) {

	gallery, err := gs.store.Galleries.Get(galleryID)
	if err != nil {
		return store.Gallery{}, err
	}

	// Depending of the type of the request check if the request could be performed.
	// If it is a public request, check that the gallery is published, else check
	// the authenticated user is the owner of the gallery.
	if public {
		if !gallery.Published {
			return store.Gallery{}, store.ErrForbidden
		}
	} else {
		authData, err := auth.ContextGetAuth(ctx)
		if err != nil {
			return store.Gallery{}, err
		}
		if authData.User.ID != gallery.UserID {
			return store.Gallery{}, store.ErrForbidden
		}
	}

	return gallery, nil
}

// Download all gallery images as a compressed tar archive (tar.gz), the request could
// be public or authenticated.
func (gs *GalleriesService) Download(ctx context.Context, public bool, galleryID int64) (store.Gallery, io.ReadCloser, error) {

	gallery, err := gs.store.Galleries.Get(galleryID)
	if err != nil {
		return store.Gallery{}, nil, err
	}

	// Depending of the type of the request check if the request could be performed.
	// If it is a public request, check that the gallery is published, else check
	// the authenticated user is the owner of the gallery.
	if public {
		if !gallery.Published {
			return store.Gallery{}, nil, store.ErrForbidden
		}
	} else {
		authData, err := auth.ContextGetAuth(ctx)
		if err != nil {
			return store.Gallery{}, nil, err
		}
		if authData.User.ID != gallery.UserID {
			return store.Gallery{}, nil, store.ErrForbidden
		}
	}

	// Try to acquire a token in the semaphore and continue in case of success. If the
	// the current concurrency is reached, return an explicative error to inform the
	// caller that the service is currently too busy.
	select {
	case gs.sema <- struct{}{}:
	default:
		return store.Gallery{}, nil, ErrBusy
	}

	// Start a goroutine in charge of streaming the compressed tar archive to the provided
	// writer. The writer is an io.Pipe, which is necessary since the caller expects a reader.
	// The io.Pipe matches reads and writes one to one.
	logger := gs.logger.With("id", tracing.TraceFromCtx(ctx).ID)
	r, w := io.Pipe()

	go func() {
		// It's vital to release the token of the semaphore in any case to release
		// acquired resources. We must close the writer so the caller knows (while
		// reading from the reader) that the bytes are ended.
		defer func() {
			w.Close()
			<-gs.sema
		}()

		// Start the helper function that will write the newly generated archive
		// into the writer passed in.
		err := gs.streamGallery(ctx, w, galleryID)
		if err != nil {
			switch {
			// This error is originated from the consumer side and we cannot do anything
			// about that, simply drop the job and don't return any error.
			case errors.Is(err, io.ErrClosedPipe):
			// Real error coming from the internal streaming function. Log the error and
			// store the error into the pipe, in order to inform the caller about it.
			default:
				logger.Errorw("streaming gallery archive", "err", err)
				w.CloseWithError(err)
			}
		}
	}()

	return gallery, r, nil
}

// Create a new gallery with the provided data, owned by the authenticated user.
func (gs *GalleriesService) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	authData := auth.MustContextGetAuth(ctx)

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

// Updates an existing gallery with the data provided, the gallery must be owned
// by the authenticated user.
func (gs *GalleriesService) Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	authData := auth.MustContextGetAuth(ctx)

	galleryToUpdate, err := gs.store.Galleries.Get(gallery.ID)
	if err != nil {
		return store.Gallery{}, err
	}

	// Make sure that the authenticated user is the owner of the gallery.
	if authData.User.ID != galleryToUpdate.UserID {
		return store.Gallery{}, store.ErrForbidden
	}

	gallery, err = gs.store.Galleries.Update(store.Gallery{
		ID:          gallery.ID,
		Title:       gallery.Title,
		Description: gallery.Description,
		Published:   gallery.Published,
		UserID:      authData.UserID,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			// The gallery is not more present, this is a concurrency issue, that is while
			// this request is being processed another request deleted the gallery.
			return store.Gallery{}, store.ErrEditConflict
		default:
			return store.Gallery{}, err
		}
	}

	return gallery, nil
}

// Delete a gallery and all related images. The authenticated user must be
// the owner of the gallery.
func (gs *GalleriesService) Delete(ctx context.Context, galleryID int64) error {
	authData := auth.MustContextGetAuth(ctx)

	galleryToDelete, err := gs.store.Galleries.Get(galleryID)
	if err != nil {
		return err
	}

	// Make sure that the authenticated user is the owner of the gallery.
	if authData.User.ID != galleryToDelete.UserID {
		return store.ErrForbidden
	}

	// Retrieves all the images of the gallery of the authenticated user and
	// delete all of them. Break the loop while all images are processed.
	var page = 1
	for {
		pagImages, meta, err := gs.store.Images.GetAllForGallery(galleryID, filters.Input{
			Page:         page,
			PageSize:     100,
			SortCol:      "id",
			SortSafeList: []string{"id"},
		})
		if err != nil {
			return err
		}
		for _, image := range pagImages {
			err := gs.store.Images.Delete(image.ID)
			if err != nil {
				return err
			}
		}
		if meta.LastPage == meta.CurrentPage {
			break
		}
		page++
	}

	err = gs.store.Galleries.DeleteGallery(galleryID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			// Edit conflict, but still ok in this case,
			// someone else already deleted the gallery.
		default:
			return err
		}
	}

	return nil
}

// The streamGallery function is a helper that writes a compressed tar archive to the
// provided writer argument. The writer could be a file or a network connection, or
// alternatively it could be a write end of a pipe. In the last case, this function
// is typically called in a separate goroutine.
func (gs *GalleriesService) streamGallery(ctx context.Context, w io.Writer, galleryID int64) error {

	// Iterate over subsequent pages of images collecting all of them.
	var images []store.Image
	var page = 1
	for {
		pagImages, pagOut, err := gs.store.Images.GetAllForGallery(galleryID, filters.Input{
			Page:         page,
			PageSize:     100,
			SortCol:      "id",
			SortSafeList: []string{"id"},
		})
		if err != nil {
			return err
		}
		images = append(images, pagImages...)
		if pagOut.CurrentPage == pagOut.LastPage {
			break
		}
		page++
	}

	// Build a writer that, in order, writes files to the tar archive, compress the data and
	// re-writes resulting bytes to the provided writer argument. To obtain this we chain
	// different types of writers, possible due to the fact that both the tar and the gzip
	// writers need a writer interface and not a concrete type.
	gzipWriter := gzip.NewWriter(w)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, image := range images {
		readCloser, err := gs.store.Images.GetReader(image.ID)
		if err != nil {
			return err
		}

		// We must cache all the image bytes in memory since the file size of the archive
		// entries must be actually written before the data itself.
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

	// Close the writers to flush all the data to the writer provided
	// as argument. Here the order matters.
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
