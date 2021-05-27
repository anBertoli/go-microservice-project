package users

import (
	"context"

	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

// The ValidationMiddleware validates incoming data of each request, rejecting them if
// some pieces of needed information are missing or malformed. The middleware makes
// sure the next service in the chain will receive valid data.
type ValidationMiddleware struct {
	Next Service
}

func (vm *ValidationMiddleware) RegisterUser(ctx context.Context, name, email, password string) (store.User, store.Keys, string, error) {
	v := validator.New()
	validator.ValidateUser(v, store.User{
		Name:     name,
		Email:    email,
		Password: password,
	})
	if !v.Ok() {
		return store.User{}, store.Keys{}, "", v
	}
	return vm.Next.RegisterUser(ctx, name, email, password)
}

func (vm *ValidationMiddleware) ActivateUser(ctx context.Context, token string) (store.User, error) {
	v := validator.New()
	v.Check(token != "", "token", "must be provided")
	if !v.Ok() {
		return store.User{}, v
	}
	return vm.Next.ActivateUser(ctx, token)
}

func (vm *ValidationMiddleware) GenKeyRecoveryToken(ctx context.Context, email, password string) (string, error) {
	v := validator.New()
	validator.ValidateEmail(v, email)
	validator.ValidatePassword(v, password)
	if !v.Ok() {
		return "", v
	}
	return vm.Next.GenKeyRecoveryToken(ctx, email, password)
}
func (vm *ValidationMiddleware) RecoverMainKey(ctx context.Context, token string) (store.Keys, error) {
	v := validator.New()
	v.Check(token != "", "token", "must be provided")
	if !v.Ok() {
		return store.Keys{}, v
	}
	return vm.Next.RecoverMainKey(ctx, token)
}

func (vm *ValidationMiddleware) ListUserKeys(ctx context.Context) ([]KeysList, error) {
	return vm.Next.ListUserKeys(ctx)
}

func (vm *ValidationMiddleware) AddUserKey(ctx context.Context, permissions store.Permissions) (store.Keys, error) {
	v := validator.New()
	validator.ValidatePermissions(v, permissions)
	if !v.Ok() {
		return store.Keys{}, v
	}
	return vm.Next.AddUserKey(ctx, permissions)
}

func (vm *ValidationMiddleware) EditUserKey(ctx context.Context, keyID int64, permissions store.Permissions) (store.Keys, store.Permissions, error) {
	v := validator.New()
	validator.ValidatePermissions(v, permissions)
	if !v.Ok() {
		return store.Keys{}, store.Permissions{}, v
	}
	return vm.Next.EditUserKey(ctx, keyID, permissions)
}

func (vm *ValidationMiddleware) DeleteUserKey(ctx context.Context, keyID int64) error {
	return vm.Next.DeleteUserKey(ctx, keyID)
}

func (vm *ValidationMiddleware) GetStats(ctx context.Context) (store.Stats, error) {
	return vm.Next.GetStats(ctx)
}
