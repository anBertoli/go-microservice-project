package galleries

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The AuthMiddleware validates necessary authorizations for the galleries service
// public interface. Authentication data must be already present in the context
// argument. Each method that needs to perform some authorization checks extracts
// the auth data and validates the permissions. Note that teh permissions are tied
// to the auth key not directly to the user.
//
// If a method is publicly accessible, this middlewares operates as a no-op. Note
// also that some methods operates in dual-mode, that is, if the request is marked
// as public it will not check any permission. In these cases, further checks will
// be done inside the core service.
type AuthMiddleware struct {
	Next Service
}

func (am *AuthMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	return am.Next.ListAllPublic(ctx, filter)
}

func (am *AuthMiddleware) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListGalleries)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return am.Next.ListAllOwned(ctx, filter)
}

func (am *AuthMiddleware) Get(ctx context.Context, public bool, galleryID int64) (store.Gallery, error) {
	if !public {
		_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListGalleries)
		if err != nil {
			return store.Gallery{}, err
		}
	}
	return am.Next.Get(ctx, public, galleryID)
}

func (am *AuthMiddleware) Download(ctx context.Context, public bool, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	if !public {
		_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionDownloadGallery)
		if err != nil {
			return store.Gallery{}, nil, err
		}
	}
	return am.Next.Download(ctx, public, galleryID)
}

func (am *AuthMiddleware) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionCreateGallery)
	if err != nil {
		return store.Gallery{}, err
	}
	return am.Next.Insert(ctx, gallery)
}

func (am *AuthMiddleware) Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionUpdateGallery)
	if err != nil {
		return store.Gallery{}, err
	}
	return am.Next.Update(ctx, gallery)
}

func (am *AuthMiddleware) Delete(ctx context.Context, galleryID int64) error {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionDeleteGallery)
	if err != nil {
		return err
	}
	return am.Next.Delete(ctx, galleryID)
}
