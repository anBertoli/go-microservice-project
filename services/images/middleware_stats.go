package images

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

const (
	maxBytes = 1024 * 1024 * 5 // 5 mb
)

type StatsMiddleware struct {
	Next     Service
	Store    store.StatsStore
	MaxBytes int64
}

func (sm *StatsMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error) {
	return sm.Next.ListAllPublic(ctx, filter)
}

func (sm *StatsMiddleware) ListForGallery(ctx context.Context, public bool, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	return sm.Next.ListForGallery(ctx, public, galleryID, filter)
}

func (sm *StatsMiddleware) Get(ctx context.Context, public bool, imageID int64) (store.Image, error) {
	return sm.Next.Get(ctx, public, imageID)
}

func (sm *StatsMiddleware) Download(ctx context.Context, public bool, imageID int64) (store.Image, io.ReadCloser, error) {
	return sm.Next.Download(ctx, public, imageID)
}

func (sm *StatsMiddleware) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	auth := store.ContextGetAuth(ctx)

	stats, err := sm.Store.GetForUser(auth.User.ID)
	if err != nil {
		return store.Image{}, err
	}
	if stats.Space >= sm.MaxBytes {
		return store.Image{}, ErrMaxSpaceReached
	}

	image, err = sm.Next.Insert(ctx, reader, image)
	if err != nil {
		return image, err
	}

	err = sm.Store.IncrementImages(image.UserID, 1)
	if err != nil {
		return store.Image{}, err
	}
	err = sm.Store.IncrementBytes(image.UserID, image.Size)
	if err != nil {
		return store.Image{}, err
	}
	return image, nil
}

func (sm *StatsMiddleware) Update(ctx context.Context, image store.Image) (store.Image, error) {
	return sm.Next.Update(ctx, image)
}

func (sm *StatsMiddleware) Delete(ctx context.Context, imageID int64) (store.Image, error) {
	image, err := sm.Next.Delete(ctx, imageID)
	if err != nil {
		return image, err
	}

	err = sm.Store.IncrementImages(image.UserID, -1)
	if err != nil {
		return store.Image{}, err
	}
	err = sm.Store.IncrementBytes(image.UserID, -image.Size)
	if err != nil {
		return store.Image{}, err
	}

	return image, nil
}
