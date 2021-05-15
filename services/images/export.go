package images

import (
	"context"
	"errors"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

type Service interface {
	ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error)
	ListAllOwned(ctx context.Context, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error)
	Download(ctx context.Context, imageID int64) (store.Image, io.ReadCloser, error)
	Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error)
	Update(ctx context.Context, image store.Image) (store.Image, error)
	Delete(ctx context.Context, imageID int64) (store.Image, error)
}

var ErrMaxSpaceReached = errors.New("max space reached")

var _ Service = &ImagesService{}
var _ Service = &AuthMiddleware{}
var _ Service = &ValidationMiddleware{}
var _ Service = &StatsMiddleware{}
