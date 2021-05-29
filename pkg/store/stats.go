package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

type Stats struct {
	Galleries int       `db:"n_galleries" json:"n_galleries"`
	Images    int       `db:"n_images" json:"n_images"`
	Space     int64     `db:"n_bytes" json:"n_bytes"`
	UserID    int64     `db:"user_id" json:"user_id"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Version   int       `db:"version" json:"-"`
}

// The store abstraction to manipulate permissions into the database. It holds a
// DB connection pool.
type StatsStore struct {
	DB *sqlx.DB
}

// Retrieve statistics about a specific user.
func (ss *StatsStore) GetForUser(userID int64) (Stats, error) {
	var stats Stats
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := ss.DB.GetContext(ctx, &stats, `SELECT * FROM stats WHERE user_id = $1`, userID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Stats{}, ErrRecordNotFound
		default:
			return Stats{}, err
		}
	}

	return stats, nil
}

// Initialize a statistics row into the database for a specific user.
func (ss *StatsStore) InitStatsForUser(userID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := ss.DB.ExecContext(ctx, `
		INSERT INTO stats (n_galleries, n_images, n_bytes, user_id, updated_at)
		VALUES (0, 0, 0, $1, $2)
	`, userID, time.Now().UTC())

	return err
}

// Increment the images counter statistic for a specific user.
func (ss *StatsStore) IncrementImages(userID int64, n int) error {
	stats, err := ss.GetForUser(userID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := ss.DB.ExecContext(ctx, ` 
		UPDATE stats
		SET n_images = n_images + $1, updated_at = $2, version = version + 1 WHERE user_id = $3 AND version = $4
		RETURNING version, updated_at
	`, n, time.Now().UTC(), userID, stats.Version)
	if err != nil {
		return err
	}

	rn, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rn == 0 {
		return ErrEditConflict
	}
	return nil
}

// Increment the space used statistic (in bytes) for a specific user.
func (ss *StatsStore) IncrementBytes(userID, n int64) error {
	stats, err := ss.GetForUser(userID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := ss.DB.ExecContext(ctx, ` 
		UPDATE stats
		SET n_bytes = n_bytes + $1, updated_at = $2, version = version + 1 WHERE user_id = $3 AND version = $4
		RETURNING version, updated_at
	`, n, time.Now().UTC(), userID, stats.Version)
	if err != nil {
		return err
	}

	rn, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rn == 0 {
		return ErrEditConflict
	}
	return nil
}

// Increment the galleries counter statistic for a specific user.
func (ss *StatsStore) IncrementGalleries(userID int64, n int) error {
	stats, err := ss.GetForUser(userID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := ss.DB.ExecContext(ctx, ` 
		UPDATE stats
		SET n_galleries = n_galleries + $1, updated_at = $2, version = version + 1 WHERE user_id = $3 AND version = $4
		RETURNING version, updated_at
	`, n, time.Now().UTC(), userID, stats.Version)
	if err != nil {
		return err
	}

	rn, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rn == 0 {
		return ErrEditConflict
	}
	return nil
}
