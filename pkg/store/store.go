package store

import (
	"errors"

	"github.com/jmoiron/sqlx"
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

func New(db *sqlx.DB, storeRoot string) (Store, error) {
	imagesStore, err := NewImagesStore(db, storeRoot)
	if err != nil {
		return Store{}, err
	}
	return Store{
		Users:       UsersStore{db},
		Keys:        KeysStore{db},
		Permissions: PermissionsStore{db},
		Tokens:      TokenStore{db},
		Galleries:   GalleriesStore{db},
		Images:      imagesStore,
		Stats:       StatsStore{db},
	}, nil
}

var (
	ErrDuplicateEmail    = errors.New("duplicate email")
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
	ErrFileAlreadyExists = errors.New("file already exists")
	ErrEmptyBytes        = errors.New("no bytes")
	ErrForbidden         = errors.New("forbidden")
)
