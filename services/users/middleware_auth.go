package users

import (
	"context"

	"github.com/anBertoli/snap-vault/pkg/auth"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// The AuthMiddleware performs authentication and validates necessary authorizations
// for the users service. Authentication is performed starting from the auth key
// eventually present in the context passed in.
//
// Note that if an API of the service doesn't require authentication, the request
// is handled automatically since the AuthMiddleware embeds a service interface.
type AuthMiddleware struct {
	Auth auth.Authenticator
	Service
}

// Perform authentication and check that permissions to list user keys are present.
func (am *AuthMiddleware) ListUserKeys(ctx context.Context) ([]KeysList, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionListKeys)
	if err != nil {
		return nil, err
	}
	return am.Service.ListUserKeys(ctx)
}

// Perform authentication and check that permissions to create user keys are present.
func (am *AuthMiddleware) AddUserKey(ctx context.Context, permissions store.Permissions) (store.Keys, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionCreateKeys)
	if err != nil {
		return store.Keys{}, err
	}
	return am.Service.AddUserKey(ctx, permissions)
}

// Perform authentication and check that permissions to edit user keys are present.
func (am *AuthMiddleware) EditUserKey(ctx context.Context, keyID int64, permissions store.Permissions) (store.Keys, store.Permissions, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionUpdateKeys)
	if err != nil {
		return store.Keys{}, store.Permissions{}, err
	}
	return am.Service.EditUserKey(ctx, keyID, permissions)
}

// Perform authentication and check that permissions to delete user keys are present.
func (am *AuthMiddleware) DeleteUserKey(ctx context.Context, keyID int64) error {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionDeleteKeys)
	if err != nil {
		return err
	}
	return am.Service.DeleteUserKey(ctx, keyID)
}

// Perform authentication. It is needed to return data about the authenticated user.
func (am *AuthMiddleware) GetMe(ctx context.Context) (auth.Auth, error) {
	_, err := am.Auth.RequireAuthenticatedUser(&ctx)
	if err != nil {
		return auth.Auth{}, err
	}
	return am.Service.GetMe(ctx)
}

// Perform authentication and check that permissions to retrieve user keys are present.
func (am *AuthMiddleware) GetStats(ctx context.Context) (store.Stats, error) {
	_, err := am.Auth.RequireUserPermissions(&ctx, store.PermissionMain, store.PermissionGetStats)
	if err != nil {
		return store.Stats{}, err
	}
	return am.Service.GetStats(ctx)
}
