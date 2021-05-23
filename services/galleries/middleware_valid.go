package galleries

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

type ValidationMiddleware struct {
	Next Service
}

func (vm *ValidationMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Next.ListAllPublic(ctx, filter)
}

func (vm *ValidationMiddleware) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Next.ListAllOwned(ctx, filter)
}

func (vm *ValidationMiddleware) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	v := validator.New()
	v.Check(gallery.Title != "", "title", "must be provided")
	if !v.Ok() {
		return store.Gallery{}, v
	}
	return vm.Next.Insert(ctx, gallery)
}

func (vm *ValidationMiddleware) Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	v := validator.New()
	v.Check(gallery.Title != "", "title", "must be provided")
	if !v.Ok() {
		return store.Gallery{}, v
	}
	return vm.Next.Update(ctx, gallery)
}

func (vm *ValidationMiddleware) Delete(ctx context.Context, galleryID int64) error {
	return vm.Next.Delete(ctx, galleryID)
}

func (vm *ValidationMiddleware) Download(ctx context.Context, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	return vm.Next.Download(ctx, galleryID)
}

func (vm *ValidationMiddleware) DownloadPublic(ctx context.Context, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	return vm.Next.DownloadPublic(ctx, galleryID)
}
