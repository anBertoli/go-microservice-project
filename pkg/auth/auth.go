package auth

import (
	"context"
	"errors"

	"github.com/anBertoli/snap-vault/pkg/store"
)

// Complete data obtained from the authentication process. Note: a user could have
// more auth keys, each with different permissions.
type Auth struct {
	store.User
	store.Keys
	store.Permissions
}

// This struct will appropriately query the underlying data source to authenticate
// the user behind the request.
type Authenticator struct {
	Store store.Store
}

// Perform authentication using the provided plain text auth key. Note that the returned
// error in case of invalid key is uniformed to a generic ErrUnauthenticated.
func (a *Authenticator) Authenticate(plainKey string) (Auth, error) {

	// Retrieve the user using the auth key provided.
	user, err := a.Store.Users.GetForKey(plainKey)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return Auth{}, ErrUnauthenticated
		default:
			return Auth{}, err
		}
	}

	// Retrieve the auth key data from the plain text key provided.
	keys, err := a.Store.Keys.GetForPlainKey(plainKey)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrRecordNotFound):
			return Auth{}, ErrUnauthenticated
		default:
			return Auth{}, err
		}
	}

	// Retrieve all associated permissions for this auth key.
	permissions, err := a.Store.Permissions.GetAllForKey(plainKey, false)
	if err != nil {
		return Auth{}, err
	}

	return Auth{
		User:        user,
		Keys:        keys,
		Permissions: permissions,
	}, nil
}

// Perform authentication, but extract the plain text auth key from the context passed in.
// Just a wrapper over the authentication method above.
func (a *Authenticator) AuthenticateFromCtx(ctx context.Context) (Auth, error) {
	plainKey, ok := ctx.Value(keyContextKey).(string)
	if !ok {
		return Auth{}, ErrUnauthenticated
	}
	return a.Authenticate(plainKey)
}

// Perform authentication extracting the key from the context. Replace the context
// pointed by the ctx pointer with a new one containing the auth data.
func (a *Authenticator) RequireAuthenticatedUser(ctx *context.Context) (Auth, error) {
	auth, err := a.AuthenticateFromCtx(*ctx)
	if err != nil {
		return Auth{}, err
	}

	// Replace the context in place.
	*ctx = context.WithValue(*ctx, authContextKey, auth)

	return auth, nil
}

// Perform authentication and require an activated user. The authentication logic is
// delegated to the RequireAuthenticatedUser method so the context will be modified
// with the obtained auth data.
func (a *Authenticator) RequireActivatedUser(ctx *context.Context) (Auth, error) {
	auth, err := a.RequireAuthenticatedUser(ctx)
	if err != nil {
		return Auth{}, err
	}
	if !auth.User.Activated {
		return Auth{}, ErrNotActivated
	}
	return auth, nil
}

// Perform authentication and require both an activated user and a user with at least one of the
// provided permissions. The authentication logic is delegated to the RequireActivatedUser
// method so the context will be modified with the obtained auth data.
func (a *Authenticator) RequireUserPermissions(ctx *context.Context, permissions ...string) (Auth, error) {
	auth, err := a.RequireActivatedUser(ctx)
	if err != nil {
		return Auth{}, err
	}
	if !auth.Permissions.Include(permissions...) {
		return Auth{}, ErrNoPermission
	}
	return auth, nil
}

// Declare a private type to be used in context to avoid key collision
// and define the keys to be used with contexts.
type privateKey string

const (
	authContextKey privateKey = "auth"
	keyContextKey  privateKey = "key"
)

// Retrieve the auth struct from a context.
func ContextGetAuth(ctx context.Context) (Auth, error) {
	authData, ok := ctx.Value(authContextKey).(Auth)
	if !ok {
		return Auth{}, ErrUnauthenticated
	}
	return authData, nil
}

// Retrieve the auth struct from a context, panicking if not found.
func MustContextGetAuth(ctx context.Context) Auth {
	authData, ok := ctx.Value(authContextKey).(Auth)
	if !ok {
		panic("cannot retrieve auth data from context")
	}
	return authData
}

// Set the auth key into the context.
func ContextSetKey(ctx context.Context, key string) context.Context {
	childCtx := context.WithValue(ctx, keyContextKey, key)
	return childCtx
}

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrNotActivated    = errors.New("user not activated")
	ErrNoPermission    = errors.New("user not activated")
)
