package users

import (
	"context"

	"github.com/anBertoli/snap-vault/pkg/store"
)

type AuthMiddleware struct {
	Store store.Store
	Next  Service
}

func (am *AuthMiddleware) RegisterUser(ctx context.Context, name, email, password string) (store.User, store.Keys, string, error) {
	return am.Next.RegisterUser(ctx, name, email, password)
}

func (am *AuthMiddleware) ActivateUser(ctx context.Context, token string) (store.User, error) {
	return am.Next.ActivateUser(ctx, token)
}

func (am *AuthMiddleware) GenKeyRecoveryToken(ctx context.Context, email, password string) (string, error) {
	return am.Next.GenKeyRecoveryToken(ctx, email, password)
}
func (am *AuthMiddleware) RecoverMainKey(ctx context.Context, token string) (store.Keys, error) {
	return am.Next.RecoverMainKey(ctx, token)
}

func (am *AuthMiddleware) ListUserKeys(ctx context.Context) ([]KeysList, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionListKeys)
	if err != nil {
		return nil, err
	}
	return am.Next.ListUserKeys(ctx)
}

func (am *AuthMiddleware) AddUserKey(ctx context.Context, permissions store.Permissions) (store.Keys, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionCreateKeys)
	if err != nil {
		return store.Keys{}, err
	}
	return am.Next.AddUserKey(ctx, permissions)
}

func (am *AuthMiddleware) EditUserKey(ctx context.Context, keyID int64, permissions store.Permissions) (store.Keys, store.Permissions, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionUpdateKeys)
	if err != nil {
		return store.Keys{}, store.Permissions{}, err
	}
	return am.Next.EditUserKey(ctx, keyID, permissions)
}

func (am *AuthMiddleware) DeleteUserKey(ctx context.Context, keyID int64) error {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionDeleteKeys)
	if err != nil {
		return err
	}
	return am.Next.DeleteUserKey(ctx, keyID)
}

func (am *AuthMiddleware) GetStats(ctx context.Context) (store.Stats, error) {
	_, err := store.RequireUserPermissions(ctx, store.PermissionMain, store.PermissionGetStats)
	if err != nil {
		return store.Stats{}, err
	}
	return am.Next.GetStats(ctx)
}
