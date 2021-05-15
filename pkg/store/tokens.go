package store

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

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

type TokenStore struct {
	db *sqlx.DB
}

func NewTokensStore(db *sqlx.DB) TokenStore {
	return TokenStore{
		db: db,
	}
}

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

func (m *TokenStore) Insert(token Token) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.db.GetContext(ctx, &token, `
		INSERT INTO tokens (hash, user_id, expiry, scope) 
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`, token.Hash, token.UserID, token.Expiry, token.Scope)

	return err
}

func (m *TokenStore) DeleteAllForUser(scope string, userID int64) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := m.db.ExecContext(ctx, `
		DELETE FROM tokens WHERE scope = $1 AND user_id = $2
	`, scope, userID)
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
