package galleries

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

type AuthMiddleware struct {
	Store store.Store
	Next  Service
}

func (am *AuthMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListPublicGalleries)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return am.Next.ListAllPublic(ctx, filter)
}

func (am *AuthMiddleware) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListGalleries)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return am.Next.ListAllOwned(ctx, filter)
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

func (am *AuthMiddleware) Download(ctx context.Context, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionDownloadGallery)
	if err != nil {
		return store.Gallery{}, nil, err
	}
	return am.Next.Download(ctx, galleryID)
}
