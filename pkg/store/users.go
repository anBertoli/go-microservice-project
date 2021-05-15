package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type User struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Email        string    `db:"email" json:"email"`
	Password     string    `db:"-" json:"-"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Activated    bool      `db:"activated" json:"activated"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
	Version      int       `db:"version" json:"-"`
}

type UsersStore struct {
	db *sqlx.DB
}

func NewUsersStore(db *sqlx.DB) UsersStore {
	return UsersStore{
		db: db,
	}
}

func (us *UsersStore) GetForEmail(email string) (User, error) {
	var user User

	// Execute the query, scanning the return values into a User struct. If no matching
	// record is found we return an ErrRecordNotFound error.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.db.GetContext(ctx, &user, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return User{}, ErrRecordNotFound
		default:
			return User{}, err
		}
	}

	return user, nil
}

func (us *UsersStore) GetForKey(key string) (User, error) {
	var (
		user    User
		keyHash = hashString(key)
	)

	// Execute the query, scanning the return values into a User struct. If no matching
	// record is found we return an ErrRecordNotFound error.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.db.GetContext(ctx, &user, `
		SELECT users.id, users.created_at, users.updated_at, users.name, users.email, users.password_hash, users.activated, users.version FROM users
		INNER JOIN auth_keys ON auth_keys.user_id = users.id
		WHERE auth_keys.auth_key_hash = $1
		`, keyHash,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return User{}, ErrRecordNotFound
		default:
			return User{}, err
		}
	}

	return user, nil
}

func (us *UsersStore) GetForToken(tokenScope, tokenPlain string) (User, error) {
	// Calculate the SHA-256 hash of the plaintext token provided by the client.
	tokenHash := hashString(tokenPlain)

	// Create a slice containing the query arguments. Notice how we use the [:] operator
	// to get a slice containing the token hash, rather than passing in the array (which
	// is not supported by the pq driver), and that we pass the current time as the
	// value to check against the token expiry.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the query, scanning the return values into a User struct. If no matching
	// record is found we return an ErrRecordNotFound error.
	var user User

	err := us.db.GetContext(ctx, &user, `
		SELECT users.id, users.created_at, users.updated_at, users.name, users.email, users.password_hash, users.activated, users.version FROM users
		INNER JOIN tokens ON users.id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3
		`, tokenHash, tokenScope, time.Now(),
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return User{}, ErrRecordNotFound
		default:
			return User{}, err
		}
	}

	return user, nil
}

func (us *UsersStore) Insert(user User) (User, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.db.GetContext(ctx, &user, `
		INSERT INTO users (name, email, password_hash, activated) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, version
	`, user.Name, user.Email, user.PasswordHash, user.Activated)

	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return User{}, ErrDuplicateEmail
		default:
			return User{}, err
		}
	}
	return user, nil
}

func (us *UsersStore) Update(user User) (User, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.db.GetContext(ctx, &user, ` 
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, updated_at = $5, version = version + 1 WHERE id = $6 AND version = $7
		RETURNING version, updated_at
	`, user.Name, user.Email, user.PasswordHash, user.Activated, time.Now().UTC(), user.ID, user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return User{}, ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return User{}, ErrRecordNotFound
		default:
			return User{}, err
		}
	}

	return user, nil
}
