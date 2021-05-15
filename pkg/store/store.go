package store

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrEmptyBytes        = errors.New("no bytes")
	ErrForbidden         = errors.New("forbidden")
	ErrUnauthenticated   = errors.New("unauthenticated")
	ErrNotActivated      = errors.New("user not activated")
	ErrNoPermission      = errors.New("wrong permissions")
)

type Store struct {
	Users       UsersStore
	Keys        KeysStore
	Permissions PermissionsStore
	Tokens      TokenStore
	Galleries   GalleriesStore
	Images      ImagesStore
	Stats       StatsStore
}

func New(db *sqlx.DB, storePath string) (Store, error) {
	imagesStore, err := NewImagesStore(db, storePath)
	if err != nil {
		return Store{}, err
	}
	return Store{
		Users:       NewUsersStore(db),
		Keys:        NewKeysStore(db),
		Permissions: NewPermissionsStore(db),
		Tokens:      NewTokensStore(db),
		Galleries:   NewGalleriesStore(db),
		Images:      imagesStore,
		Stats:       NewStatsStore(db),
	}, nil
}

type Auth struct {
	User
	Keys
	Permissions
}

type privateKey string

const authDataContextKey privateKey = "auth_data"

func ContextSetAuth(ctx context.Context, auth *Auth) context.Context {
	childCtx := context.WithValue(ctx, authDataContextKey, auth)
	return childCtx
}

func ContextGetAuth(ctx context.Context) *Auth {
	authData, ok := ctx.Value(authDataContextKey).(*Auth)
	if !ok {
		return nil
	}
	return authData
}

func RequireAuthenticatedUser(ctx context.Context) (*Auth, error) {
	auth := ContextGetAuth(ctx)
	if auth == nil {
		return nil, ErrUnauthenticated
	}
	return auth, nil
}

func RequireActivatedUser(ctx context.Context) (*Auth, error) {
	auth, err := RequireAuthenticatedUser(ctx)
	if err != nil {
		return nil, err
	}
	if !auth.User.Activated {
		return nil, ErrNotActivated
	}
	return auth, nil
}

func RequireUserPermissions(ctx context.Context, permissions ...string) (*Auth, error) {
	auth, err := RequireActivatedUser(ctx)
	if err != nil {
		return nil, err
	}
	if !auth.Permissions.Include(permissions...) {
		return nil, ErrNoPermission
	}
	return auth, nil
}
