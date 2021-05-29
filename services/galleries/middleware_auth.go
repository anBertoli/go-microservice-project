package galleries

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The AuthMiddleware performs authentication and validates necessary authorizations
// for the galleries service. Authentication is performed starting from the auth key
// eventually present in the context passed in.
//
// Note that if an API of the service doesn't require authentication, the request
// is handled automatically since the AuthMiddleware embeds a service interface.
type AuthMiddleware struct {
	Auth auth.Authenticator
	Service
}

// Perform authentication and check that appropriate listing permissions are present.
func (am *AuthMiddleware) ListAllOwned(ctx context.Context, filter filters.Input) ([]store.Gallery, filters.Meta, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionListGalleries)
	if err != nil {
		return nil, filters.Meta{}, err
	}
	return am.Service.ListAllOwned(ctx, filter)
}

// Perform authentication and check that appropriate get permissions are present.
func (am *AuthMiddleware) Get(ctx context.Context, public bool, galleryID int64) (store.Gallery, error) {
	if !public {
		_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionListGalleries)
		if err != nil {
			return store.Gallery{}, err
		}
	}
	return am.Service.Get(ctx, public, galleryID)
}

// Perform authentication and check that appropriate download permissions are present.
func (am *AuthMiddleware) Download(ctx context.Context, public bool, galleryID int64) (store.Gallery, io.ReadCloser, error) {
	if !public {
		_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionDownloadGallery)
		if err != nil {
			return store.Gallery{}, nil, err
		}
	}
	return am.Service.Download(ctx, public, galleryID)
}

// Perform authentication and check that appropriate insert permissions are present.
func (am *AuthMiddleware) Insert(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionCreateGallery)
	if err != nil {
		return store.Gallery{}, err
	}
	return am.Service.Insert(ctx, gallery)
}

// Perform authentication and check that appropriate update permissions are present.
func (am *AuthMiddleware) Update(ctx context.Context, gallery store.Gallery) (store.Gallery, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionUpdateGallery)
	if err != nil {
		return store.Gallery{}, err
	}
	return am.Service.Update(ctx, gallery)
}

// Perform authentication and check that appropriate delete permissions are present.
func (am *AuthMiddleware) Delete(ctx context.Context, galleryID int64) error {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionDeleteGallery)
	if err != nil {
		return err
	}
	return am.Service.Delete(ctx, galleryID)
}
