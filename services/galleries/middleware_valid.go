package galleries

import (
	"context"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

// The ValidationMiddleware validates incoming data of each request, rejecting them if some
// pieces of needed information are missing or malformed. The middleware makes sure the next
// service in the chain will receive valid data. Some methods are no-ops since there it isn't
// needed to validate data (the calls are handled directly from the embedded Service interface).
type ValidationMiddleware struct {
	Service
}

// Validate the filtering and pagination parameters used in listing.
func (vm *ValidationMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Service.ListAllPublic(ctx, filter)
}

// Validate the filtering and pagination parameters used in listing.
func (vm *ValidationMiddleware) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Service.ListAllOwned(ctx, filter)
}

// Validate the title to be used to insert a new gallery.
func (vm *ValidationMiddleware) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	v := validator.New()
	v.Check(gallery.Title != "", "title", "must be provided")
	if !v.Ok() {
		return store.Gallery{}, v
	}
	return vm.Service.Insert(ctx, gallery)
}

// Validate the title to be used to update an existing gallery.
func (vm *ValidationMiddleware) Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	v := validator.New()
	v.Check(gallery.Title != "", "title", "must be provided")
	if !v.Ok() {
		return store.Gallery{}, v
	}
	return vm.Service.Update(ctx, gallery)
}
