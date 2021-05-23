package images

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

func (vm *ValidationMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Next.ListAllPublic(ctx, filter)
}

func (vm *ValidationMiddleware) ListAllOwned(ctx context.Context, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Next.ListAllOwned(ctx, galleryID, filter)
}

func (vm *ValidationMiddleware) DownloadPublic(ctx context.Context, imageID int64) (store.Image, io.ReadCloser, error) {
	return vm.Next.DownloadPublic(ctx, imageID)
}

func (vm *ValidationMiddleware) Download(ctx context.Context, imageID int64) (store.Image, io.ReadCloser, error) {
	return vm.Next.Download(ctx, imageID)
}

func (vm *ValidationMiddleware) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	v := validator.New()
	v.Check(image.Title != "", "title", "must be specified")
	if !v.Ok() {
		return store.Image{}, v
	}
	return vm.Next.Insert(ctx, reader, image)
}

func (vm *ValidationMiddleware) Update(ctx context.Context, image store.Image) (store.Image, error) {
	v := validator.New()
	v.Check(image.Title != "", "title", "must be specified")
	if !v.Ok() {
		return store.Image{}, v
	}
	return vm.Next.Update(ctx, image)
}

func (vm *ValidationMiddleware) Delete(ctx context.Context, imageID int64) (store.Image, error) {
	return vm.Next.Delete(ctx, imageID)
}
