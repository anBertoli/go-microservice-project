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

// The store abstraction used to manipulate images into our postgres
// database and into the file system storage. It holds a DB
// connection pool.
type ImagesStore struct {
	db     *sqlx.DB
	fsRoot string
}

// Instantiate a new images store. The constructor is used
// to check if the provided store path is valid.
func NewImagesStore(db *sqlx.DB, path string) (ImagesStore, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ImagesStore{}, err
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		return ImagesStore{}, err
	}
	if !stat.IsDir() {
		return ImagesStore{}, fmt.Errorf("'%s' is not a dir", path)
	}
	return ImagesStore{
		db:     db,
		fsRoot: absPath,
	}, nil
}

// Retrieve a specific image data from the database.
func (is *ImagesStore) Get(imageID int64) (Image, error) {
	var image Image
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Join the galleries and users tables to provide more infos in
	// the returned image.
	err := is.db.GetContext(ctx, &image, `
		SELECT 
			images.id, images.filepath, images.title, images.size, images.caption, images.content_type, images.created_at, 
  			images.updated_at, images.gallery_id, users.id as user_id, galleries.published
		FROM images 
			LEFT JOIN galleries on images.gallery_id = galleries.id
			LEFT JOIN users on users.id = galleries.user_id
		WHERE images.id = $1
	`, imageID)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Image{}, ErrRecordNotFound
		default:
			return Image{}, err
		}
	}

	return image, nil
}

// Return a read-closer that provides the bytes content of a specific image. The
// returned read-closer must be closed by the caller, if not, file descriptors
// will be leaked.
func (is *ImagesStore) GetReader(imageID int64) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var image Image
	err := is.db.GetContext(ctx, &image, `
		SELECT 
			images.id, images.filepath, images.title, images.size, images.content_type, images.caption, images.created_at, 
  			images.updated_at, images.gallery_id, galleries.user_id as user_id
		FROM images 
		LEFT JOIN galleries on images.gallery_id = galleries.id
		WHERE images.id = $1
	`, imageID)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	// The image was saved into the file system. The relative path of the file
	// (relative to the 'root' of the store) was saved into the DB. Join the
	// root and the relative path to find the image location.
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

// Obtain a list of public images, that is, images belonging to a public gallery.
// This operation supports filtering and pagination so the method also returns
// pagination metadata.
func (is *ImagesStore) GetAllPublic(filter filters.Input) ([]Image, filters.Meta, error) {
	var (
		images   = []Image{}
		metadata = filter.CalculateMetadata(0)
		// Use a temporary variable to scan also the count.
		tmp []struct {
			Count int64 `db:"count"`
			Image
		}
	)

	// Like in the galleries listing operations we include the count and we provide
	// support for records filtering based on search col field.
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

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, metadata, nil
		default:
			return nil, metadata, err
		}
	}

	// Convert the results into a galleries slice, then calculate pagination metadata.
	for _, i := range tmp {
		images = append(images, i.Image)
	}
	if len(tmp) > 0 {
		metadata = filter.CalculateMetadata(tmp[0].Count)
	}

	return images, metadata, nil
}

// Obtain a list of images belonging to a specific gallery. This operation supports filtering and
// pagination so the method also returns pagination metadata.
func (is *ImagesStore) GetAllForGallery(galleryID int64, filter filters.Input) ([]Image, filters.Meta, error) {
	var (
		images   = []Image{}
		metadata = filter.CalculateMetadata(0)
		// Use a temporary variable to scan also the count.
		tmp []struct {
			Count int64 `db:"count"`
			Image
		}
	)

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

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, metadata, nil
		default:
			return nil, metadata, err
		}
	}

	// Convert the results into a galleries slice, then calculate pagination metadata.
	for _, i := range tmp {
		images = append(images, i.Image)
	}
	if len(tmp) > 0 {
		metadata = filter.CalculateMetadata(tmp[0].Count)
	}

	return images, metadata, nil
}

// Inserts a new image for a specific gallery into the database and save the image bytes
// into the file system. The image struct passed in must contain the necessary information,
// but note that id, created_at and updated_at are set automatically by the database.
func (is *ImagesStore) Insert(r io.Reader, image Image) (Image, error) {
	var (
		imageSize int64
		relPath   string
	)

	// Compute the path where the image will be saved, using a random string.
	// If a name collision occur, retry again with a different random string.
	for {
		relPath = filepath.Join(
			fmt.Sprintf("gallery_%d", image.GalleryID),
			fmt.Sprintf("%s_%s", image.Title, randString(25)),
		)
		path, err := filepath.Abs(filepath.Join(is.fsRoot, relPath))
		if err != nil {
			return Image{}, err
		}
		imageSize, err = is.writeImage(r, path)
		if errors.Is(err, ErrFileAlreadyExists) {
			continue
		}
		if err != nil {
			return Image{}, err
		}
		break
	}

	// Update relevant image fields then insert an image record into the db.
	image.Path = relPath
	image.Size = imageSize

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.GetContext(ctx, &image, `
		INSERT
			INTO images (filepath, title, caption, created_at, updated_at, size, content_type, gallery_id)
			VALUES ($1, $2, $3, now(), now(), $4, $5, $6) 
			RETURNING id, created_at, updated_at
	`, image.Path, image.Title, image.Caption, imageSize, image.ContentType, image.GalleryID)
	if err != nil {
		return Image{}, err
	}

	return image, nil
}

// Helper func used to write an image into the file system store. The file is
// created with O_EXCL mode, that is, it must not exist.
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

// Update data about a specific image into the database.
func (is *ImagesStore) Update(image Image) (Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := is.db.GetContext(ctx, &image, `
		UPDATE images SET title = $1, caption = $2, updated_at = $3 
		WHERE id = $4
		RETURNING updated_at
	`, image.Title, image.Caption, time.Now().UTC(), image.ID)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return Image{}, ErrRecordNotFound
		default:
			return Image{}, err
		}
	}

	return image, nil
}

// Delete the specified image both from the gallery and from the store.
func (is *ImagesStore) Delete(imageID int64) error {
	var image Image

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Retrieve the image to delete from the db. To know where it is stored
	// we must join the store root with the relative path of the image.
	err := is.db.GetContext(ctx, &image, `SELECT * FROM images WHERE id = $1`, imageID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	// Delete the image from the file system.
	path, err := filepath.Abs(filepath.Join(is.fsRoot, image.Path))
	if err != nil {
		return err
	}
	err = os.RemoveAll(path)
	if err != nil {
		return err
	}

	// Delete the image metadata from the database.
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
		return ErrEditConflict
	}

	return nil
}

// Generate a random string.
func randString(length int) string {
	runes := []rune("123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]rune, length)
	for i := range b {
		b[i] = runes[r.Intn(len(runes))]
	}
	return string(b)
}
