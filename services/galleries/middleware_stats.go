package galleries

import (
	"context"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The StatsMiddleware updates the user stats about the number of galleries. Some methods
// are no-ops since they don't need to modify the stats of a user (the calls are handled
// directly from the embedded Service interface).
type StatsMiddleware struct {
	Store store.StatsStore
	Service
}

// Increment the galleries counter for the user if a new gallery is successfully created.
func (sm *StatsMiddleware) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	gallery, err := sm.Service.Insert(ctx, gallery)
	if err != nil {
		return gallery, err
	}

	err = sm.Store.IncrementGalleries(gallery.UserID, 1)
	if err != nil {
		return store.Gallery{}, err
	}
	return gallery, nil
}

// Decrement the galleries counter for the user if the gallery is deleted.
func (sm *StatsMiddleware) Delete(ctx context.Context, galleryID int64) error {
	authData := auth.MustContextGetAuth(ctx)

	err := sm.Service.Delete(ctx, galleryID)
	if err != nil {
		return err
	}
	return sm.Store.IncrementGalleries(authData.User.ID, -1)
}
