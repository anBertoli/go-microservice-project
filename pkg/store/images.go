package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/anBertoli/snap-vault/pkg/filters"
)

type Image struct {
	ID          int64     `json:"id" db:"id"`
	Path        string    `json:"-" db:"filepath"`
	Title       string    `json:"title" db:"title"`
	Caption     string    `json:"caption" db:"caption"`
	Size        int64     `json:"size" db:"size"`
	ContentType string    `json:"content_type" db:"content_type"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	GalleryID   int64     `json:"gallery_id" db:"gallery_id"`
	Published   bool      `json:"published" db:"published"`
	UserID      int64     `json:"user_id" db:"user_id"`
}

type ImagesStore struct {
	db     *sqlx.DB
	fsRoot string
}

func NewImagesStore(db *sqlx.DB, path string) (ImagesStore, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ImagesStore{}, err
	}
	return ImagesStore{
		db:     db,
		fsRoot: absPath,
	}, nil
}

func (is *ImagesStore) Get(imageID int64) (Image, error) {
	var image Image
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.GetContext(ctx, &image, `
		SELECT 
			images.id, images.filepath, images.title, images.size, images.caption, images.content_type, images.created_at, 
  			images.updated_at, images.gallery_id, users.id as user_id, galleries.published
		FROM images 
			LEFT JOIN galleries on images.gallery_id = galleries.id
			LEFT JOIN users on users.id = galleries.user_id
		WHERE images.id = $1
	`, imageID)
	if err == sql.ErrNoRows {
		return Image{}, ErrRecordNotFound
	}
	if err != nil {
		return Image{}, err
	}

	return image, nil
}

func (is *ImagesStore) GetReader(imageID int64) (io.ReadCloser, error) {
	var image Image
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.GetContext(ctx, &image, `
		SELECT 
			images.id, images.filepath, images.title, images.size, images.content_type, images.caption, images.created_at, 
  			images.updated_at, images.gallery_id, galleries.user_id as user_id
		FROM images 
		LEFT JOIN galleries on images.gallery_id = galleries.id
		WHERE images.id = $1
	`, imageID)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}

	path, err := filepath.Abs(filepath.Join(is.fsRoot, image.Path))
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (is *ImagesStore) GetAllPublic(filter filters.Input) ([]Image, filters.Meta, error) {
	var (
		images   = []Image{}
		metadata filters.Meta
		tmp      []struct {
			Count int64 `db:"count"`
			Image
		}
	)

	// The inclusion of the count(*) OVER() expression at the start
	// of the query will result in the filtered record count being
	// included as the first value in each row.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.SelectContext(ctx, &tmp, fmt.Sprintf(`
		SELECT count(*) OVER(), 
			images.id, images.filepath, images.title, images.size, images.content_type, images.caption, images.created_at, 
			images.updated_at, images.gallery_id, galleries.user_id as user_id, galleries.published
		FROM images 
		LEFT JOIN galleries on images.gallery_id = galleries.id
		WHERE ((LOWER(images.%s) LIKE LOWER('%%%s%%')) OR ($1 = '')) AND published = true
		ORDER BY images.%s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filter.SearchCol, filter.Search, filter.SortColumn(), filter.SortDirection(),
	), filter.Search, filter.Limit(), filter.Offset())
	if err == sql.ErrNoRows {
		return nil, filters.Meta{}, ErrRecordNotFound
	}
	if err != nil {
		return nil, metadata, err
	}

	for _, i := range tmp {
		images = append(images, i.Image)
	}
	if len(tmp) > 0 {
		metadata = filter.CalculateMetadata(tmp[0].Count)
	} else {
		metadata = filter.CalculateMetadata(0)
	}

	return images, metadata, nil
}

func (is *ImagesStore) GetAllForGallery(galleryID int64, filter filters.Input) ([]Image, filters.Meta, error) {
	var (
		images = []Image{}
		meta   filters.Meta
		tmp    []struct {
			Count int64 `db:"count"`
			Image
		}
	)

	// The inclusion of the count(*) OVER() expression at the start
	// of the query will result in the filtered record count being
	// included as the first value in each row.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.SelectContext(ctx, &tmp, fmt.Sprintf(`
		SELECT count(*) OVER(), 
                images.id, images.filepath, images.title, images.size, images.content_type, images.caption, images.created_at, 
				images.updated_at, images.gallery_id, users.id as user_id, galleries.published
		FROM images 
			LEFT JOIN galleries on images.gallery_id = galleries.id
			LEFT JOIN users on users.id = galleries.user_id
		WHERE ((LOWER(images.%s) LIKE LOWER('%%%s%%')) OR ($1 = '')) AND gallery_id = $2
		ORDER BY images.%s %s, id ASC
		LIMIT $3 OFFSET $4`,
		filter.SearchCol, filter.Search, filter.SortColumn(), filter.SortDirection(),
	), filter.Search, galleryID, filter.Limit(), filter.Offset())
	if err == sql.ErrNoRows {
		return nil, meta, ErrRecordNotFound
	}
	if err != nil {
		return nil, meta, err
	}

	for _, i := range tmp {
		images = append(images, i.Image)
	}

	if len(tmp) > 0 {
		meta = filter.CalculateMetadata(tmp[0].Count)
	} else {
		meta = filter.CalculateMetadata(0)
	}

	return images, meta, nil
}

func (is *ImagesStore) Insert(r io.Reader, image Image) (Image, error) {
	relPath := filepath.Join(
		fmt.Sprintf("gallery_%d", image.GalleryID),
		fmt.Sprintf("%s_%s", image.Title, randString(15)),
	)
	path, err := filepath.Abs(filepath.Join(is.fsRoot, relPath))
	if err != nil {
		return Image{}, err
	}
	n, err := is.writeImage(r, path)
	if err != nil {
		return Image{}, err
	}

	image.Path = relPath
	image.Size = n

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	now := time.Now().UTC()
	err = is.db.GetContext(ctx, &image, `
		INSERT
			INTO images (filepath, title, caption, created_at, updated_at, size, content_type, gallery_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) 
			RETURNING id, created_at, updated_at
	`, image.Path, image.Title, image.Caption, now, now, n, image.ContentType, image.GalleryID)
	if err != nil {
		return Image{}, err
	}

	return image, nil
}

func (is *ImagesStore) writeImage(r io.Reader, path string) (int64, error) {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return 0, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if os.IsExist(err) {
		return 0, ErrFileAlreadyExists
	}
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(file, r)
	if err != nil {
		return n, err
	}
	err = file.Close()
	if err != nil {
		return n, err
	}
	return n, err
}

func (is *ImagesStore) Update(image Image) (Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.GetContext(ctx, &image, `
		UPDATE images SET title = $1, caption = $2, updated_at = $3 
		WHERE id = $4
		RETURNING updated_at
	`, image.Title, image.Caption, time.Now().UTC(), image.ID)
	if err == sql.ErrNoRows {
		return Image{}, ErrRecordNotFound
	}
	if err != nil {
		return Image{}, err
	}

	return image, nil
}

func (is *ImagesStore) Delete(imageID int64) error {
	var image Image

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.GetContext(ctx, &image, `SELECT * FROM images WHERE id = $1`, imageID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	path, err := filepath.Abs(filepath.Join(is.fsRoot, image.Path))
	if err != nil {
		return err
	}
	err = os.RemoveAll(path)
	if err != nil {
		return err
	}

	ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	res, err := is.db.ExecContext(ctx, `DELETE FROM images WHERE id = $1`, imageID)
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

func randString(n int) string {
	runes := []rune("123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]rune, n)
	for i := range b {
		b[i] = runes[r.Intn(len(runes))]
	}
	return string(b)
}
