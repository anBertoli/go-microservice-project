package images

import (
	"context"
	"io"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The AuthMiddleware performs authentication and validates necessary authorizations
// for the images service. Authentication is performed starting from the auth key
// eventually present in the context passed in.
//
// Note that if an API of the service doesn't require authentication, the request
// is handled automatically since the AuthMiddleware embeds a service interface.
type AuthMiddleware struct {
	Auth auth.Authenticator
	Service
}

// Perform authentication and check that permissions to list images for a
// specific gallery are present.
func (am *AuthMiddleware) ListForGallery(ctx context.Context, public bool, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	if !public {
		_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionListImages)
		if err != nil {
			return nil, filters.Meta{}, err
		}
	}
	return am.Service.ListForGallery(ctx, public, galleryID, filter)
}

// Perform authentication and check that permissions to retrieve an image are present.
func (am *AuthMiddleware) Get(ctx context.Context, public bool, imageID int64) (store.Image, error) {
	if !public {
		_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionListImages)
		if err != nil {
			return store.Image{}, err
		}
	}
	return am.Service.Get(ctx, public, imageID)
}

// Perform authentication and check that permissions to download an image are present.
func (am *AuthMiddleware) Download(ctx context.Context, public bool, imageID int64) (store.Image, io.ReadCloser, error) {
	if !public {
		_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionDownloadImage)
		if err != nil {
			return store.Image{}, nil, err
		}
	}
	return am.Service.Download(ctx, public, imageID)
}

// Perform authentication and check that permissions to create a new image are present.
func (am *AuthMiddleware) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionCreateImage)
	if err != nil {
		return store.Image{}, err
	}
	return am.Service.Insert(ctx, reader, image)
}

// Perform authentication and check that permissions to edit an existing image are present.
func (am *AuthMiddleware) Update(ctx context.Context, image store.Image) (store.Image, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionUpdateImage)
	if err != nil {
		return store.Image{}, err
	}
	return am.Service.Update(ctx, image)
}

// Perform authentication and check that permissions to delete an existing image are present.
func (am *AuthMiddleware) Delete(ctx context.Context, imageID int64) (store.Image, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionDeleteImage)
	if err != nil {
		return store.Image{}, err
	}
	return am.Service.Delete(ctx, imageID)
}
