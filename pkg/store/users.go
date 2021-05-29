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

// The store abstraction used to manipulate users into our postgres database.
// It holds a DB connection pool.
type UsersStore struct {
	DB *sqlx.DB
}

// Retrieve a user using its email.
func (us *UsersStore) GetForEmail(email string) (User, error) {
	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.DB.GetContext(ctx, &user, `SELECT * FROM users WHERE email = $1`, email)
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

// Retrieve a user from one of its auth keys. The provided key argument is
// the plain text version of the auth key and it's hashed before searching
// the user into the database.
func (us *UsersStore) GetForKey(key string) (User, error) {
	var (
		user    User
		keyHash = hashString(key)
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.DB.GetContext(ctx, &user, `
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

// Retrieve the user that has the associated token. Both the token hash and the
// token scope must match. The token argument is the plain text version.
// The token must not be expired.
func (us *UsersStore) GetForToken(tokenScope, tokenPlain string) (User, error) {
	tokenHash := hashString(tokenPlain)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user User
	err := us.DB.GetContext(ctx, &user, `
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

// Create a new user into the database. The password is hashed before inserting it into
// the db. Note that some values are populated directly by the db.
func (us *UsersStore) Insert(user User) (User, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.DB.GetContext(ctx, &user, `
		INSERT INTO users (name, email, password_hash, activated) VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, version
	`, user.Name, user.Email, user.PasswordHash, user.Activated)

	if err != nil {
		switch {
		// We can detect if a user with the same
		// email already exists in our DB.
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return User{}, ErrDuplicateEmail
		default:
			return User{}, err
		}
	}

	return user, nil
}

// Update an existing user. The new data is provided in the passed in User struct.
func (us *UsersStore) Update(user User) (User, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := us.DB.GetContext(ctx, &user, ` 
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, updated_at = now(), version = version + 1 WHERE id = $5 AND version = $6
		RETURNING version, updated_at
	`, user.Name, user.Email, user.PasswordHash, user.Activated, user.ID, user.Version)
	if err != nil {
		switch {
		// We can detect if a user with the same
		// email already exists in our DB.
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
