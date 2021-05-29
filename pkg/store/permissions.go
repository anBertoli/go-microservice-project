package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// List of existing permissions, this will be stored into the database
// permissions table as is. They will be linked to auth keys, but they
// are 'constants' in our DB and are not editable from the application.
const (
	PermissionMain = "*:*" // non editable

	PermissionListKeys   = "keys:list"
	PermissionCreateKeys = "keys:create"
	PermissionUpdateKeys = "keys:update"
	PermissionDeleteKeys = "keys:delete"
	PermissionGetStats   = "users:stats"

	PermissionListGalleries   = "galleries:list"
	PermissionCreateGallery   = "galleries:create"
	PermissionUpdateGallery   = "galleries:update"
	PermissionDeleteGallery   = "galleries:delete"
	PermissionDownloadGallery = "galleries:download"

	PermissionListImages    = "images:list"
	PermissionCreateImage   = "images:create"
	PermissionUpdateImage   = "images:update"
	PermissionDeleteImage   = "images:delete"
	PermissionDownloadImage = "images:download"
)

// The list of permissions that could be linked or unlinked from auth keys.
// In other words, requests can only modify this permissions (the 'main
// permission' for a account is excluded).
var EditablePermissions = Permissions{
	PermissionListKeys,
	PermissionCreateKeys,
	PermissionUpdateKeys,
	PermissionDeleteKeys,
	PermissionListGalleries,
	PermissionCreateGallery,
	PermissionUpdateGallery,
	PermissionDeleteGallery,
	PermissionDownloadGallery,
	PermissionListImages,
	PermissionCreateImage,
	PermissionUpdateImage,
	PermissionDeleteImage,
	PermissionDownloadImage,
}

// Define a type to manipulate easily permissions.
type Permissions []string

// Check that at least one of the provided codes is included in the
// permissions receiver (p).
func (p Permissions) Include(codes ...string) bool {
	for i := range p {
		for _, c := range codes {
			if c == p[i] {
				return true
			}
		}
	}
	return false
}

// Filter and return invalids permissions.
func (p Permissions) Invalids() Permissions {
	var invalids Permissions
	for i := range p {
		if !EditablePermissions.Include(p[i]) {
			invalids = append(invalids, p[i])
		}
	}
	return invalids
}

// The store abstraction to manipulate permissions into the database. It holds a
// DB connection pool.
type PermissionsStore struct {
	DB *sqlx.DB
}

// Retrieve all permissions associated with a specified key. The provided key could be hashed
// or could be the plain version.
func (ps *PermissionsStore) GetAllForKey(key string, isKeyHashed bool) (Permissions, error) {
	var (
		keyHash     = key
		permissions = []string{}
	)
	if !isKeyHashed {
		keyHash = hashString(key)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ps.DB.SelectContext(ctx, &permissions, `
		SELECT permissions.code FROM permissions
		INNER JOIN auth_keys_permissions ON auth_keys_permissions.permission_id = permissions.id 
		INNER JOIN auth_keys ON auth_keys_permissions.auth_key_id = auth_keys.id
		WHERE auth_keys.auth_key_hash = $1
	`, keyHash)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return permissions, nil
		default:
			return nil, err
		}
	}

	return permissions, nil
}

// Replace associated permissions of an auth key with the provided permissions. The old
// permissions are deleted and the new permissions are inserted into a transaction.
func (ps *PermissionsStore) ReplaceForKey(keyID int64, codes ...string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Begin the transaction.
	tx, err := ps.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	// Delete the association between old permissions and the auth key.
	_, err = tx.ExecContext(ctx, `
		DELETE FROM auth_keys_permissions WHERE auth_keys_permissions.auth_key_id = $1
	`, keyID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	// Create the new associations.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO auth_keys_permissions
		SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
	`, keyID, pq.Array(codes))
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	// Commit the transaction.
	return tx.Commit()
}
