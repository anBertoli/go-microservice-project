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

type KeysStore struct {
	db *sqlx.DB
}

func NewKeysStore(db *sqlx.DB) KeysStore {
	return KeysStore{
		db: db,
	}
}

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

func (ks *KeysStore) GetForPlainKey(key string) (Keys, error) {
	var (
		keys    Keys
		keyHash = hashString(key)
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ks.db.GetContext(ctx, &keys, `
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

func (ks *KeysStore) GetAllForUser(userID int64) ([]Keys, error) {
	keys := []Keys{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ks.db.SelectContext(ctx, &keys, `
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

func (ks *KeysStore) Insert(keys Keys) (Keys, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ks.db.GetContext(ctx, &keys, `
		INSERT INTO auth_keys (auth_key_hash, user_id)
			VALUES ($1, $2)
			RETURNING id, created_at
	`, keys.AuthKeyHash, keys.UserID)

	return keys, err
}

func (ks *KeysStore) DeleteKey(keyID, userID int64) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := ks.db.ExecContext(ctx, `
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
	// Use the Read() function from the crypto/rand package to fill the byte slice with
	// random bytes from your operating system's CSPRNG. This will return an error if
	// the CSPRNG fails to function correctly.
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", "", err
	}

	// Encode the byte slice to a base-32-encoded string and assign it to the token
	// Plaintext field. This will be the token string that we send to the user in their
	// welcome email. They will look similar to this:
	//
	// Y3QMGX3PJ3WLRL2YRTQGQ6KRHU
	//
	// Note that by default base-32 strings may be padded at the end with the =
	// character. We don't need this padding character for the purpose of our tokens, so
	// we use the WithPadding(base32.NoPadding) method in the line below to omit them.
	keyPlain := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(randomBytes)

	// Generate a SHA-256 hash of the plaintext token string. This will be the value
	// that we store in the `hash` field of our database table. Note that the
	// sha256.Sum256() function returns an *array* of length 32, so to make it easier to
	// work with we convert it to a slice using the [:] operator before storing it.
	keyHash := sha256.Sum256([]byte(keyPlain))
	keyHashStr := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(keyHash[:])
	return keyPlain, keyHashStr, nil
}

func hashString(s string) string {
	keyHash := sha256.Sum256([]byte(s))
	return base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(keyHash[:])
}
