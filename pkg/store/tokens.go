package store

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// Define the scoped of the tokens used in the application.
const (
	ScopeActivation      = "activation"
	ScopeRecoverMainKeys = "recover-keys"
)

type Token struct {
	Plain     string    `db:"-" json:"-"`
	Hash      string    `db:"hash" json:"-"`
	Scope     string    `db:"scope" json:"-"`
	Expiry    time.Time `db:"expiry" json:"-"`
	UserID    int64     `db:"user_id" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"-"`
}

// The store abstraction to manipulate tokens into the database. It holds a DB
// connection pool. Only the hashed version of the token are saved into the db.
type TokenStore struct {
	DB *sqlx.DB
}

// Creates a new token with the given scope and ttl (time-to-live) and saves the hash into the
// database. The plain text version of the token is returned and not viewable/recoverable
// anymore.
func (m *TokenStore) New(userID int64, ttl time.Duration, scope string) (Token, error) {
	plainToken, tokenHash, err := generateToken()
	if err != nil {
		return Token{}, err
	}
	token := Token{
		Plain:  plainToken,
		Hash:   tokenHash,
		Scope:  scope,
		Expiry: time.Now().UTC().Add(ttl),
		UserID: userID,
	}

	err = m.Insert(token)
	if err != nil {
		return Token{}, err
	}
	return token, nil
}

// Insert a new token into the database. The plain text version of the token
// is not saved.
func (m *TokenStore) Insert(token Token) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.GetContext(ctx, &token, `
		INSERT INTO tokens (hash, user_id, expiry, scope) 
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`, token.Hash, token.UserID, token.Expiry, token.Scope)
}

// Delete all tokens with the given scope for the specified user.
func (m *TokenStore) DeleteAllForUser(scope string, userID int64) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := m.DB.ExecContext(ctx, `DELETE FROM tokens WHERE scope = $1 AND user_id = $2`, scope, userID)
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
