package users

import (
	"context"

	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

// The ValidationMiddleware validates incoming data of each request, rejecting them if some
// pieces of needed information are missing or malformed. The middleware makes sure the next
// service in the chain will receive valid data. Some methods are no-ops since there it isn't
// needed to validate data (the calls are handled directly from the embedded Service interface).
type ValidationMiddleware struct {
	Service
}

// Validate user data before actually registering a new user.
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
	return vm.Service.RegisterUser(ctx, name, email, password)
}

// Validate the activation token before activating the user.
func (vm *ValidationMiddleware) ActivateUser(ctx context.Context, token string) (store.User, error) {
	v := validator.New()
	v.Check(token != "", "token", "must be provided")
	if !v.Ok() {
		return store.User{}, v
	}
	return vm.Service.ActivateUser(ctx, token)
}

// Validate email and password before generating a new key recover token.
func (vm *ValidationMiddleware) GenKeyRecoveryToken(ctx context.Context, email, password string) (string, error) {
	v := validator.New()
	validator.ValidateEmail(v, email)
	validator.ValidatePassword(v, password)
	if !v.Ok() {
		return "", v
	}
	return vm.Service.GenKeyRecoveryToken(ctx, email, password)
}

// Validate the token before trying to regenerate the main auth key.
func (vm *ValidationMiddleware) RegenerateMainKey(ctx context.Context, token string) (store.Keys, error) {
	v := validator.New()
	v.Check(token != "", "token", "must be provided")
	if !v.Ok() {
		return store.Keys{}, v
	}
	return vm.Service.RegenerateMainKey(ctx, token)
}

// Validate the provided permissions before using them to add a new auth key.
func (vm *ValidationMiddleware) AddUserKey(ctx context.Context, permissions store.Permissions) (store.Keys, error) {
	v := validator.New()
	validator.ValidatePermissions(v, permissions)
	if !v.Ok() {
		return store.Keys{}, v
	}
	return vm.Service.AddUserKey(ctx, permissions)
}

// Validate the provided permissions before using them to edit an existing auth key.
func (vm *ValidationMiddleware) EditUserKey(ctx context.Context, keyID int64, permissions store.Permissions) (store.Keys, store.Permissions, error) {
	v := validator.New()
	validator.ValidatePermissions(v, permissions)
	if !v.Ok() {
		return store.Keys{}, store.Permissions{}, v
	}
	return vm.Service.EditUserKey(ctx, keyID, permissions)
}
