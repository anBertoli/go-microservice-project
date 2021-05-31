package images

import (
	"context"
	"errors"
	"io"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// Public interface for the gallery service. The service is exposed
// via transport-specific adapters, e.g. the JSON-HTTP api.
type Service interface {
	ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error)
	ListForGallery(ctx context.Context, public bool, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error)
	Get(ctx context.Context, public bool, imageID int64) (store.Image, error)
	Download(ctx context.Context, public bool, imageID int64) (store.Image, io.ReadCloser, error)
	Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error)
	Update(ctx context.Context, image store.Image) (store.Image, error)
	Delete(ctx context.Context, imageID int64) (store.Image, error)
}

var (
	ErrMaxSpaceReached = errors.New("max space reached")
)

// This checks makes sure that all service implementation remain
// valid while we refactor our code.
var _ Service = &ImagesService{}
var _ Service = &AuthMiddleware{}
var _ Service = &ValidationMiddleware{}
var _ Service = &StatsMiddleware{}
