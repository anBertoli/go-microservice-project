package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	PermissionAdmin = "admin" // non editable
	PermissionMain  = "*:*"   // non editable

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

type Permissions []string

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

func (p Permissions) Invalids() Permissions {
	var invalids Permissions
	for i := range p {
		if !EditablePermissions.Include(p[i]) {
			invalids = append(invalids, p[i])
		}
	}
	return invalids
}

type PermissionsStore struct {
	db *sqlx.DB
}

func NewPermissionsStore(db *sqlx.DB) PermissionsStore {
	return PermissionsStore{
		db: db,
	}
}

func (ps *PermissionsStore) GetAllForKey(key string, isKeyHashed bool) (Permissions, error) {

	var (
		keyHash       = key
		permissions   = []string{}
		dbPermissions []struct {
			Id   int    `db:"id"`
			Code string `db:"code"`
		}
	)
	if !isKeyHashed {
		keyHash = hashString(key)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ps.db.SelectContext(ctx, &dbPermissions, `
		SELECT permissions.code FROM permissions
		INNER JOIN auth_keys_permissions ON auth_keys_permissions.permission_id = permissions.id 
		INNER JOIN auth_keys ON auth_keys_permissions.auth_key_id = auth_keys.id
		WHERE auth_keys.auth_key_hash = $1
	`, keyHash)
	if err == sql.ErrNoRows {
		return permissions, nil
	}
	if err != nil {
		return nil, err
	}

	for _, p := range dbPermissions {
		permissions = append(permissions, p.Code)
	}
	return permissions, nil
}

func (ps *PermissionsStore) ReplaceForKey(keyID int64, codes ...string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := ps.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM auth_keys_permissions WHERE auth_keys_permissions.auth_key_id = $1
	`, keyID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO auth_keys_permissions
		SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)
	`, keyID, pq.Array(codes))
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
