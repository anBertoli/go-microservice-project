package images

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

type AuthMiddleware struct {
	Next Service
}

func (am *AuthMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error) {
	return am.Next.ListAllPublic(ctx, filter)
}

func (am *AuthMiddleware) ListAllOwned(ctx context.Context, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListImages)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return am.Next.ListAllOwned(ctx, galleryID, filter)
}

func (am *AuthMiddleware) Download(ctx context.Context, imageID int64) (store.Image, io.ReadCloser, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionDownloadImage)
	if err != nil {
		return store.Image{}, nil, err
	}
	return am.Next.Download(ctx, imageID)
}

func (am *AuthMiddleware) DownloadPublic(ctx context.Context, imageID int64) (store.Image, io.ReadCloser, error) {
	return am.Next.DownloadPublic(ctx, imageID)
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
