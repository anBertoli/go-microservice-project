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

// The store abstraction used to manipulate galleries into our postgres database.
// It holds a DB connection pool.
type GalleriesStore struct {
	DB *sqlx.DB
}

// Retrieve a specific gallery from the database.
func (gs *GalleriesStore) Get(id int64) (Gallery, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var gallery Gallery
	err := gs.DB.GetContext(ctx, &gallery, `SELECT * FROM GALLERIES WHERE id = $1`, id)
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

// Obtain a list of public galleries. This operation supports filtering and pagination
// so the method also returns pagination metadata.
func (gs *GalleriesStore) GetAllPublic(filter filters.Input) ([]Gallery, filters.Meta, error) {
	var (
		galleries = []Gallery{}
		meta      = filter.CalculateMetadata(0)
		// Use a tmp variable to scan also the count.
		tmp []struct {
			Gallery
			Count int64 `db:"count"`
		}
	)

	// The count(*) OVER() expression at the start of the query will result in the filtered
	// record count being included as the first value in each row. The query will filter
	// results based on the search col parameter but only if the value is populated. The
	// filtering is case-insensitive and the filter value must be a substring of the
	// related record field.
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), * FROM galleries
		WHERE ((LOWER(%s) LIKE LOWER('%%%s%%')) OR ($1 = '')) AND published = true
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filter.SearchCol, filter.Search, filter.SortColumn(), filter.SortDirection(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.DB.SelectContext(ctx, &tmp, query, filter.Search, filter.Limit(), filter.Offset())
	if err != nil {
		switch {
		// No records is not an error here, so
		// simply return an empty slice.
		case errors.Is(err, sql.ErrNoRows):
			return nil, meta, nil
		default:
			return nil, meta, err
		}
	}

	// Convert the results into a galleries slice, then calculate pagination metadata.
	for _, g := range tmp {
		galleries = append(galleries, g.Gallery)
	}
	if len(tmp) > 0 {
		meta = filter.CalculateMetadata(tmp[0].Count)
	}

	return galleries, meta, nil
}

// Obtain a list of galleries for the specified user. This operation supports filtering
// and pagination so the method also returns pagination metadata.
func (gs *GalleriesStore) GetAllForUser(userID int64, filter filters.Input) ([]Gallery, filters.Meta, error) {
	var (
		galleries = []Gallery{}
		pagMeta   = filter.CalculateMetadata(0)
		tmp       []struct {
			Gallery
			Count int64 `db:"count"`
		}
	)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.DB.SelectContext(ctx, &tmp, fmt.Sprintf(`
		SELECT count(*) OVER(), * FROM galleries
		WHERE ((LOWER(%s) LIKE LOWER('%%%s%%')) OR ($1 = '')) AND user_id = $2
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`,
		filter.SearchCol, filter.Search, filter.SortColumn(), filter.SortDirection(),
	), filter.Search, userID, filter.Limit(), filter.Offset())
	if err != nil {
		switch {
		// No records is not an error here, so
		// simply return an empty slice.
		case errors.Is(err, sql.ErrNoRows):
			return nil, pagMeta, nil
		default:
			return nil, pagMeta, err
		}
	}

	for _, g := range tmp {
		galleries = append(galleries, g.Gallery)
	}
	if len(tmp) > 0 {
		pagMeta = filter.CalculateMetadata(tmp[0].Count)
	}

	return galleries, pagMeta, nil
}

// Inserts a new gallery. The gallery struct passed in must contain the necessary information,
// but note that id, created_at and updated_at are set automatically by the database.
func (gs *GalleriesStore) Insert(gallery Gallery) (Gallery, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use the returning clause to collect values set by the database.
	err := gs.DB.GetContext(ctx, &gallery, `
			INSERT
			INTO galleries (title, description, published, user_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, now(), now()) 
			RETURNING id, created_at, updated_at
	`, gallery.Title, gallery.Description, gallery.Published, gallery.UserID)
	if err != nil {
		return Gallery{}, err
	}

	return gallery, nil
}

// Update an existing gallery. The gallery struct passed must contain the necessary information,
// but note that some fields are retrieved automatically from the database.
func (gs *GalleriesStore) Update(gallery Gallery) (Gallery, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := gs.DB.GetContext(ctx, &gallery, `
			UPDATE galleries SET title = $1, description = $2, published = $3, updated_at = now()
			WHERE id = $4
			RETURNING created_at, updated_at
	`, gallery.Title, gallery.Description, gallery.Published, gallery.ID)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Gallery{}, ErrRecordNotFound
		default:
			return Gallery{}, err
		}
	}

	return gallery, err
}

// Delete the specified gallery, note that deleting related images is a
// responsibility of the caller.
func (gs *GalleriesStore) DeleteGallery(id int64) error {
	res, err := gs.DB.Exec(`DELETE from galleries WHERE id=$1`, id)
	if err != nil {
		return err
	}
	// Check that the gallery is effectively deleted.
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRecordNotFound
	}
	return nil
}
