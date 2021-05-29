package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type Keys struct {
	ID          int64     `db:"id" json:"id"`
	AuthKey     string    `db:"-" json:"auth_key,omitempty"`
	AuthKeyHash string    `db:"auth_key_hash" json:"-"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UserID      int64     `db:"user_id" json:"-"`
}

// The store abstraction to manipulate user auth keys into the database. It holds a
// DB connection pool. Only the hashed version of the keys are saved into the db.
type KeysStore struct {
	DB *sqlx.DB
}

// Creates a new auth key and saves the hash into the database. The plain text version
// of the key is returned and not viewable/recoverable anymore.
func (ks *KeysStore) New(userID int64) (Keys, error) {
	authKey, authKeyHash, err := generateToken()
	if err != nil {
		return Keys{}, err
	}
	keys := Keys{
		AuthKey:     authKey,
		AuthKeyHash: authKeyHash,
		UserID:      userID,
	}
	keys, err = ks.Insert(keys)
	if err != nil {
		return Keys{}, err
	}
	return keys, nil
}

// Retrieve auth key data starting from the plain text version of the key.
func (ks *KeysStore) GetForPlainKey(key string) (Keys, error) {
	var (
		keys    Keys
		keyHash = hashString(key)
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ks.DB.GetContext(ctx, &keys, `
		SELECT id, auth_key_hash, created_at, user_id  
		FROM auth_keys WHERE auth_key_hash = $1
	`, keyHash)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Keys{}, ErrRecordNotFound
		default:
			return Keys{}, err
		}
	}

	return keys, nil
}

// Retrieve all auth keys for a specific user.
func (ks *KeysStore) GetAllForUser(userID int64) ([]Keys, error) {
	keys := []Keys{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ks.DB.SelectContext(ctx, &keys, `
		SELECT id, auth_key_hash, created_at, user_id  
		FROM auth_keys WHERE user_id = $1
	`, userID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return []Keys{}, nil
		default:
			return []Keys{}, err
		}
	}

	return keys, err
}

// Insert a new auth key into the database. Only the hashed version is saved.
func (ks *KeysStore) Insert(keys Keys) (Keys, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ks.DB.GetContext(ctx, &keys, `
		INSERT INTO auth_keys (auth_key_hash, user_id)
			VALUES ($1, $2)
			RETURNING id, created_at
	`, keys.AuthKeyHash, keys.UserID)

	return keys, err
}

// Delete an auth key specified via the key ID and the owner ID.
func (ks *KeysStore) DeleteKey(keyID, userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := ks.DB.ExecContext(ctx, `
		DELETE FROM auth_keys WHERE auth_keys.id = $1 AND auth_keys.user_id = $2
	`, keyID, userID)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func generateToken() (string, string, error) {
	// Use the Read() function from the crypto/rand package to fill the byte slice
	// with random bytes from your operating system's CSPRNG.
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", "", err
	}

	// Encode the byte slice to a base-64-encoded string.
	keyPlain := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(randomBytes)

	// Generate a SHA-256 hash of the token string. This will be the value
	// that we store in the `hash` field of our database tables.
	keyHash := sha256.Sum256([]byte(keyPlain))
	keyHashStr := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(keyHash[:])
	return keyPlain, keyHashStr, nil
}

func hashString(s string) string {
	keyHash := sha256.Sum256([]byte(s))
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(keyHash[:])
}
