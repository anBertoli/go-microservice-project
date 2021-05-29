package images

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The StatsMiddleware updates the user stats about the number of images and the total
// stored bytes of a user. Additionally it check if the user has exceeded the space
// it can use to store data. Some methods are no-ops since they don't need to modify
// the stats of a user (the calls are handled directly from the embedded Service interface).
type StatsMiddleware struct {
	Store    store.StatsStore
	MaxBytes int64
	Service
}

func (sm *StatsMiddleware) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	authData := auth.MustContextGetAuth(ctx)

	stats, err := sm.Store.GetForUser(authData.User.ID)
	if err != nil {
		return store.Image{}, err
	}

	// If the total space in use by a user will exceed the threshold after
	// inserting the current image reject the request.
	if stats.Space >= sm.MaxBytes {
		return store.Image{}, ErrMaxSpaceReached
	}

	// Insert the image, then increment related counters for the user.
	image, err = sm.Service.Insert(ctx, reader, image)
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

func (sm *StatsMiddleware) Delete(ctx context.Context, imageID int64) (store.Image, error) {
	image, err := sm.Service.Delete(ctx, imageID)
	if err != nil {
		return image, err
	}

	// Decrement images counters for the user.
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
