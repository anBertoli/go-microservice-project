package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/anBertoli/snap-vault/pkg/filters"
)

type Gallery struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Published   bool      `json:"published" db:"published"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type GalleriesStore struct {
	db *sqlx.DB
}

func NewGalleriesStore(db *sqlx.DB) GalleriesStore {
	return GalleriesStore{
		db: db,
	}
}

func (gs *GalleriesStore) Get(id int64) (Gallery, error) {
	var gallery Gallery
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.db.GetContext(ctx, &gallery, `SELECT * FROM GALLERIES WHERE id = $1`, id)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Gallery{}, ErrRecordNotFound
		default:
			return Gallery{}, err
		}
	}

	return gallery, nil
}

func (gs *GalleriesStore) GetAllPublic(filter filters.Input) ([]Gallery, filters.Meta, error) {
	var (
		galleries = []Gallery{}
		pagMeta   filters.Meta
		dbg       []struct {
			Gallery
			Count int64 `db:"count"`
		}
	)

	// The inclusion of the count(*) OVER() expression at the start
	// of the query will result in the filtered record count being
	// included as the first value in each row.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.db.SelectContext(ctx, &dbg, fmt.Sprintf(`
		SELECT count(*) OVER(), * FROM galleries
		WHERE ((LOWER(%s) LIKE LOWER('%%%s%%')) OR ($1 = '')) AND published = true
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filter.SearchCol, filter.Search, filter.SortColumn(), filter.SortDirection(),
	), filter.Search, filter.Limit(), filter.Offset())
	if err == sql.ErrNoRows {
		return nil, pagMeta, ErrRecordNotFound
	}
	if err != nil {
		return nil, pagMeta, err
	}

	for _, g := range dbg {
		galleries = append(galleries, g.Gallery)
	}
	if len(dbg) > 0 {
		pagMeta = filter.CalculateOutput(dbg[0].Count)
	} else {
		pagMeta = filter.CalculateOutput(0)
	}

	return galleries, pagMeta, nil
}

func (gs *GalleriesStore) GetAllForUser(userID int64, filter filters.Input) ([]Gallery, filters.Meta, error) {
	var (
		galleries = []Gallery{}
		pagMeta   filters.Meta
		dbg       []struct {
			Gallery
			Count int64 `db:"count"`
		}
	)

	// The inclusion of the count(*) OVER() expression at the start
	// of the query will result in the filtered record count being
	// included as the first value in each row.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.db.SelectContext(ctx, &dbg, fmt.Sprintf(`
		SELECT count(*) OVER(), * FROM galleries
		WHERE ((LOWER(%s) LIKE LOWER('%%%s%%')) OR ($1 = '')) AND user_id = $2
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`,
		filter.SearchCol, filter.Search, filter.SortColumn(), filter.SortDirection(),
	), filter.Search, userID, filter.Limit(), filter.Offset())
	if err == sql.ErrNoRows {
		return nil, pagMeta, ErrRecordNotFound
	}
	if err != nil {
		return nil, pagMeta, err
	}

	for _, g := range dbg {
		galleries = append(galleries, g.Gallery)
	}
	if len(dbg) > 0 {
		pagMeta = filter.CalculateOutput(dbg[0].Count)
	} else {
		pagMeta = filter.CalculateOutput(0)
	}

	return galleries, pagMeta, nil
}

func (gs *GalleriesStore) Insert(gallery Gallery) (Gallery, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.db.GetContext(ctx, &gallery, `
			INSERT
			INTO galleries (title, description, published, user_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, now(), now()) 
			RETURNING id, created_at, updated_at
	`, gallery.Title, gallery.Description, gallery.Published, gallery.UserID)
	if err == sql.ErrNoRows {
		return Gallery{}, ErrRecordNotFound
	}

	return gallery, err
}

func (gs *GalleriesStore) Update(gallery Gallery) (Gallery, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.db.GetContext(ctx, &gallery, `
			UPDATE galleries SET title = $1, description = $2, published = $3, updated_at = now()
			WHERE id = $4
			RETURNING user_id, created_at, updated_at
	`, gallery.Title, gallery.Description, gallery.Published, gallery.ID)
	if err == sql.ErrNoRows {
		return Gallery{}, ErrRecordNotFound
	}
	if err != nil {
		return Gallery{}, err
	}

	return gallery, err
}

func (gs *GalleriesStore) DeleteGallery(id int64) error {
	res, err := gs.db.Exec(`DELETE from galleries WHERE id=$1`, id)
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
