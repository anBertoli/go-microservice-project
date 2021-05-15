package main

import (
	"fmt"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

func (app *application) listPublicImagesHandler(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	filter := filters.Input{
		Page:                 readInt(queryString, "page", 1),
		PageSize:             readInt(queryString, "page_size", 20),
		SortCol:              readString(queryString, "sort", "id"),
		SortSafeList:         []string{"id", "title", "created_at", "-id", "-title", "-created_at"},
		Search:               readString(queryString, "search", ""),
		SearchCol:            readString(queryString, "search_field", "title"),
		SearchColumnSafeList: []string{"title", "caption"},
	}

	images, metadata, err := app.images.ListAllPublic(r.Context(), filter)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"public_images": images, "filter": metadata}, nil)
}

func (app *application) listImagesHandler(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	filter := filters.Input{
		Page:                 readInt(queryString, "page", 1),
		PageSize:             readInt(queryString, "page_size", 20),
		SortCol:              readString(queryString, "sort", "id"),
		SortSafeList:         []string{"id", "title", "created_at", "-id", "-title", "-created_at"},
		Search:               readString(queryString, "search", ""),
		SearchCol:            readString(queryString, "search_field", "title"),
		SearchColumnSafeList: []string{"title", "caption"},
	}

	galleryID, err := readIDParam(r, "gallery-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	images, metadata, err := app.images.ListAllOwned(r.Context(), galleryID, filter)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"images": images, "filter": metadata}, nil)
}

func (app *application) downloadImageHandler(w http.ResponseWriter, r *http.Request) {
	imageID, err := readIDParam(r, "image-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	image, readCloser, err := app.images.Download(r.Context(), imageID)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	headers := http.Header{
		"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"%s\"", image.Title)},
	}
	app.streamBytes(w, r, readCloser, headers)
}

const maxBodyBytes = 1024 * 1024 * 20

func (app *application) createImageHandler(w http.ResponseWriter, r *http.Request) {
	galleryID, err := readIDParam(r, "gallery-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	title := r.URL.Query().Get("title")

	reader := http.MaxBytesReader(w, r.Body, maxBodyBytes)

	image, err := app.images.Insert(r.Context(), reader, store.Image{
		GalleryID: galleryID,
		Title:     title,
	})
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"image": image}, nil)
}

func (app *application) editImageHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string `json:"title"`
		Caption string `json:"caption"`
	}

	err := readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}

	imageID, err := readIDParam(r, "image-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	image, err := app.images.Update(r.Context(), store.Image{
		ID:      imageID,
		Title:   input.Title,
		Caption: input.Caption,
	})
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"image": image}, nil)
}

func (app *application) deleteImageHandler(w http.ResponseWriter, r *http.Request) {
	imageID, err := readIDParam(r, "image-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	_, err = app.images.Delete(r.Context(), imageID)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"deleted_image_id": imageID}, nil)
}
