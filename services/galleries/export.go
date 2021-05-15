package galleries

import (
	"context"
	"errors"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

type Service interface {
	ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error)
	ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error)
	Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error)
	Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error)
	Delete(ctx context.Context, galleryID int64) error
	Download(ctx context.Context, galleryID int64) (store.Gallery, io.ReadCloser, error)
}

var ErrBusy = errors.New("busy")

var _ Service = &GalleriesService{}
var _ Service = &AuthMiddleware{}
var _ Service = &ValidationMiddleware{}
var _ Service = &StatsMiddleware{}
