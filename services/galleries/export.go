package galleries

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
	ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error)
	ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error)
	Get(ctx context.Context, public bool, galleryID int64) (store.Gallery, error)
	Download(ctx context.Context, public bool, galleryID int64) (store.Gallery, io.ReadCloser, error)
	Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error)
	Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error)
	Delete(ctx context.Context, galleryID int64) error
}

var (
	ErrBusy = errors.New("busy")
)

// This checks makes sure that all service implementation remain
// valid while we refactor our code.
var _ Service = &GalleriesService{}
var _ Service = &AuthMiddleware{}
var _ Service = &ValidationMiddleware{}
var _ Service = &StatsMiddleware{}
