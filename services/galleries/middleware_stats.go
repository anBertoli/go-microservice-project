package galleries

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

type StatsMiddleware struct {
	Next  Service
	Store store.StatsStore
}

func (sm *StatsMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	return sm.Next.ListAllPublic(ctx, filter)
}

func (sm *StatsMiddleware) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	return sm.Next.ListAllOwned(ctx, filter)
}

func (sm *StatsMiddleware) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	gallery, err := sm.Next.Insert(ctx, gallery)
	if err != nil {
		return gallery, err
	}

	err = sm.Store.IncrementGalleries(gallery.UserID, 1)
	if err != nil {
		return store.Gallery{}, err
	}
	return gallery, nil
}

func (sm *StatsMiddleware) Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	return sm.Next.Update(ctx, gallery)
}

func (sm *StatsMiddleware) Delete(ctx context.Context, galleryID int64) error {
	authData := store.ContextGetAuth(ctx)

	err := sm.Next.Delete(ctx, galleryID)
	if err != nil {
		return err
	}
	return sm.Store.IncrementGalleries(authData.User.ID, -1)
}

func (sm *StatsMiddleware) Download(ctx context.Context, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	return sm.Next.Download(ctx, galleryID)
}

func (sm *StatsMiddleware) DownloadPublic(ctx context.Context, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	return sm.Next.DownloadPublic(ctx, galleryID)
}
