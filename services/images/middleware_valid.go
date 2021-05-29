package images

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/gabriel-vasile/mimetype"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
)

// The ValidationMiddleware validates incoming data of each request, rejecting them if
// some pieces of needed information are missing or malformed. The middleware makes
// sure the next service in the chain will receive valid data and computes the
// MIME type of the image.
type ValidationMiddleware struct {
	Service
}

func (vm *ValidationMiddleware) ListAllPublic(ctx context.Context, filter filters.Input) ([]store.Image, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Service.ListAllPublic(ctx, filter)
}

func (vm *ValidationMiddleware) ListForGallery(ctx context.Context, public bool, galleryID int64, filter filters.Input) ([]store.Image, filters.Meta, error) {
	err := filter.Validate()
	if err != nil {
		v := validator.New()
		v.AddError("pagination", err.Error())
		return nil, filters.Meta{}, v
	}
	return vm.Service.ListForGallery(ctx, public, galleryID, filter)
}

func (vm *ValidationMiddleware) Insert(ctx context.Context, reader io.Reader, image store.Image) (store.Image, error) {
	v := validator.New()

	// Read at most 512 bytes from the reader, that is, the image. If err is io.EOF the
	// reader is empty, if err is io.ErrUnexpectedEOF the body has less than 512 bytes.
	// In the last case we we are still good since we can try to extract the MIME type
	// from the extracted bytes. So, in this case we will carry on nonetheless.
	buf := make([]byte, 512)
	n, err := io.ReadAtLeast(reader, buf, 512)
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
			return store.Image{}, store.ErrEmptyBytes
		case errors.Is(err, io.ErrUnexpectedEOF):
		default:
			return store.Image{}, err
		}
	}

	// Re-slice the byte to include only the portion of bytes filled by the read function
	// above (n could also be 512). Then try to guess the MIME type.
	buf = buf[:n]
	image.ContentType = mimetype.Detect(buf).String()

	v.Check(image.Title != "", "title", "must be specified")
	v.Check(strings.HasPrefix(image.ContentType, "image/"), "image", "not in supported format")
	if !v.Ok() {
		return store.Image{}, v
	}

	// Before going ahead with the next.Insert() call we must reform the reader since we
	// have consumed the first 512 bytes of it. The io.MultiReader returns a new reader
	// that will read sequentially from the provided readers.
	reader = io.MultiReader(bytes.NewReader(buf), reader)

	return vm.Service.Insert(ctx, reader, image)
}

func (vm *ValidationMiddleware) Update(ctx context.Context, image store.Image) (store.Image, error) {
	v := validator.New()
	v.Check(image.Title != "", "title", "must be specified")
	if !v.Ok() {
		return store.Image{}, v
	}
	return vm.Service.Update(ctx, image)
}
