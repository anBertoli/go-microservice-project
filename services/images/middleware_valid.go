package images

import (
	"context"
	"io"
	"mime"

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

func (vm *ValidationMiddleware) ListForGallery(ctx context.Context, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Next.ListForGallery(ctx, galleryID, filter)
}

func (vm *ValidationMiddleware) Get(ctx context.Context, public bool, imageID int64) (store.Image, error) {
	return vm.Next.Get(ctx, public, imageID)
}

func (vm *ValidationMiddleware) Download(ctx context.Context, public bool, imageID int64) (store.Image, io.ReadCloser, error) {
	return vm.Next.Download(ctx, public, imageID)
}

func (vm *ValidationMiddleware) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	v := validator.New()
	if image.ContentType == "" {
		image.ContentType = "application/octet-stream"
	} else {
		_, _, err := mime.ParseMediaType(image.ContentType)
		if err != nil {
			v.AddError("content-type", "invalid value")
		}
	}
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
