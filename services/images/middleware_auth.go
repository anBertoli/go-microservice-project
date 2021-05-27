package images

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The AuthMiddleware validates necessary authorizations for the images service
// public interface. For further information take a look at the same middleware
// of the gallery service.
//
// If a method is publicly accessible, this middlewares operates as a no-op. Note
// also that some methods operates in dual-mode, that is, if the request is marked
// as public it will not check any permission. In these cases, further checks will
// be done inside the core service.
type AuthMiddleware struct {
	Next Service
}

func (am *AuthMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error) {
	return am.Next.ListAllPublic(ctx, filter)
}

func (am *AuthMiddleware) ListForGallery(ctx context.Context, public bool, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	if !public {
		_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListImages)
		if err != nil {
			return nil, filters.Meta{}, err
		}
	}
	return am.Next.ListForGallery(ctx, public, galleryID, filter)
}

func (am *AuthMiddleware) Get(ctx context.Context, public bool, imageID int64) (store.Image, error) {
	if !public {
		_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListImages)
		if err != nil {
			return store.Image{}, err
		}
	}
	return am.Next.Get(ctx, public, imageID)
}

func (am *AuthMiddleware) Download(ctx context.Context, public bool, imageID int64) (store.Image, io.ReadCloser, error) {
	if !public {
		_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionDownloadImage)
		if err != nil {
			return store.Image{}, nil, err
		}
	}
	return am.Next.Download(ctx, public, imageID)
}

func (am *AuthMiddleware) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionCreateImage)
	if err != nil {
		return store.Image{}, err
	}
	return am.Next.Insert(ctx, reader, image)
}

func (am *AuthMiddleware) Update(ctx context.Context, image store.Image) (store.Image, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionUpdateImage)
	if err != nil {
		return store.Image{}, err
	}
	return am.Next.Update(ctx, image)
}

func (am *AuthMiddleware) Delete(ctx context.Context, imageID int64) (store.Image, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionDeleteImage)
	if err != nil {
		return store.Image{}, err
	}
	return am.Next.Delete(ctx, imageID)
}
