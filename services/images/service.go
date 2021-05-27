package images

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The GalleriesService retrieves and save gallery images metadata in a relation database
// and the actual image file in the file system.
type ImagesService struct {
	Store store.Store
}

// Returns a filtered and paginated list of public images.
func (is *ImagesService) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error) {
	images, metadata, err := is.Store.Images.GetAllPublic(filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return images, metadata, nil
}

// Returns a filtered and paginated list of images about a specific gallery
// owned by the authenticated user.
func (is *ImagesService) ListForGallery(ctx context.Context, public bool, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	authData := store.ContextGetAuth(ctx)

	gallery, err := is.Store.Galleries.Get(galleryID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return nil, filters.Meta{}, store.ErrRecordNotFound
		default:
			return nil, filters.Meta{}, err
		}
	}

	// Depending of the type of the request check if the request could be performed.
	// If it is a public request, check that the gallery is published, else check
	// the authenticated user is the owner of the gallery.
	if public {
		if !gallery.Published {
			return nil, filters.Meta{}, store.ErrForbidden
		}
	} else {
		if authData.User.ID != gallery.UserID {
			return nil, filters.Meta{}, store.ErrForbidden
		}
	}

	images, metadata, err := is.Store.Images.GetAllForGallery(galleryID, filter)
	if err != nil {
		return nil, filters.Meta{}, err
	}

	return images, metadata, nil
}

// Fetch the image data, the request could be public or authenticated.
func (is *ImagesService) Get(ctx context.Context, public bool, imageID int64) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	image, err := is.Store.Images.Get(imageID)
	if err != nil {
		return store.Image{}, err
	}

	// Depending of the type of the request check if the request could be performed.
	// If it is a public request, check that the gallery is published, else check
	// the authenticated user is the owner of the gallery.
	if public {
		if !image.Published {
			return store.Image{}, store.ErrForbidden
		}
	} else {
		if authData.User.ID != image.UserID {
			return store.Image{}, store.ErrForbidden
		}
	}

	return image, nil
}

// Download a specific image, the request could be public or authenticated. The images bytes are
// provided as a reader.
func (is *ImagesService) Download(ctx context.Context, public bool, imageID int64) (store.Image, io.ReadCloser, error) {
	image, err := is.Store.Images.Get(imageID)
	if err != nil {
		return store.Image{}, nil, err
	}

	// Depending of the type of the request check if the request could be performed.
	// If it is a public request, check that the gallery is published, else check
	// the authenticated user is the owner of the gallery.
	if public {
		if !image.Published {
			return store.Image{}, nil, store.ErrForbidden
		}
	} else {
		authData := store.ContextGetAuth(ctx)
		if authData.User.ID != image.UserID {
			return store.Image{}, nil, store.ErrForbidden
		}
	}

	readCloser, err := is.Store.Images.GetReader(imageID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, nil, store.ErrEditConflict
		default:
			return store.Image{}, nil, err
		}
	}

	return image, readCloser, nil
}

// Creates a new image for a specific gallery owned by the authenticated user. The actual image
// bytes are provided as a reader from the caller.
func (is *ImagesService) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	gallery, err := is.Store.Galleries.Get(image.GalleryID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrRecordNotFound
		default:
			return store.Image{}, err
		}
	}

	if gallery.UserID != authData.User.ID {
		return store.Image{}, store.ErrForbidden
	}

	image, err = is.Store.Images.Insert(reader, store.Image{
		Title:       image.Title,
		Caption:     image.Caption,
		UserID:      authData.User.ID,
		ContentType: image.ContentType,
		GalleryID:   image.GalleryID,
		Published:   gallery.Published,
	})
	if err != nil {
		switch {
		// The gallery was deleted concurrently during this request.
		case strings.Contains(err.Error(), "image_to_galleries_fk"):
			return store.Image{}, store.ErrEditConflict
		default:
			return store.Image{}, err
		}
	}

	return image, nil
}

// Updates an existing image with the data provided, the image gallery must be owned
// by the authenticated user.
func (is *ImagesService) Update(ctx context.Context, image store.Image) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	oldImage, err := is.Store.Images.Get(image.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrRecordNotFound
		default:
			return store.Image{}, err
		}
	}

	// Make sure that the authenticated user is the owner of the gallery.
	if oldImage.UserID != authData.User.ID {
		return store.Image{}, store.ErrForbidden
	}

	oldImage.Title = image.Title
	oldImage.Caption = image.Caption

	image, err = is.Store.Images.Update(oldImage)
	if err != nil {
		switch {
		// The gallery was deleted concurrently during this request.
		case strings.Contains(err.Error(), "image_to_galleries_fk"):
			return store.Image{}, store.ErrEditConflict
		default:
			return store.Image{}, err
		}
	}

	return image, nil
}

// Delete a specific image. The authenticated user must be the owner of the image gallery.
func (is *ImagesService) Delete(ctx context.Context, imageID int64) (store.Image, error) {
	authData := store.ContextGetAuth(ctx)

	image, err := is.Store.Images.Get(imageID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrRecordNotFound
		default:
			return store.Image{}, err
		}
	}

	// Make sure that the authenticated user is the owner of the gallery.
	if image.UserID != authData.User.ID {
		return store.Image{}, store.ErrForbidden
	}

	err = is.Store.Images.Delete(imageID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return store.Image{}, store.ErrEditConflict
		default:
			return store.Image{}, err
		}
	}

	return image, nil
}
