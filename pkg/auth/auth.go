package auth

import (
	"context"
	"errors"

	"github.com/anBertoli/snap-vault/pkg/store"
)

type Authenticator struct {
	Store store.Store
}

type Auth struct {
	store.User
	store.Keys
	store.Permissions
}

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrNotActivated    = errors.New("user not activated")
	ErrNoPermission    = errors.New("user not activated")
)

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

// Perform authentication, but extract the auth key from the passed in context.
func (a *Authenticator) AuthenticateFromCtx(ctx context.Context) (Auth, error) {
	plainKey, ok := ctx.Value(keyContextKey).(string)
	if !ok {
		return Auth{}, ErrUnauthenticated
	}
	return a.Authenticate(plainKey)
}

func (a *Authenticator) RequireAuthenticatedUser(ctx *context.Context) (Auth, error) {
	auth, err := a.AuthenticateFromCtx(*ctx)
	if err != nil {
		return Auth{}, err
	}
	// Replace the context in place.
	*ctx = context.WithValue(*ctx, authContextKey, auth)
	return auth, nil
}

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

// Helpers to manipulate the context.

// Declare a private type to be used in context to avoid key collision.
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

// Set the key string into the context.
func ContextSetKey(ctx context.Context, key string) context.Context {
	childCtx := context.WithValue(ctx, keyContextKey, key)
	return childCtx
}
