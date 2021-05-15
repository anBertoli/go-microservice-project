package main

import (
	"fmt"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

func (app *application) listPublicGalleriesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		filter filters.Input
	}

	queryString := r.URL.Query()
	input.filter = filters.Input{
		Page:                 readInt(queryString, "page", 1),
		PageSize:             readInt(queryString, "page_size", 20),
		SortCol:              readString(queryString, "sort", "id"),
		SortSafeList:         []string{"id", "filter", "created_at", "-id", "-filter", "-created_at"},
		Search:               readString(queryString, "search", ""),
		SearchCol:            readString(queryString, "search_field", "title"),
		SearchColumnSafeList: []string{"title", "description"},
	}

	galleries, metadata, err := app.galleries.ListAllPublic(r.Context(), input.filter)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"public_galleries": galleries, "filter": metadata}, nil)
}

func (app *application) listGalleriesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		filter filters.Input
	}

	queryString := r.URL.Query()
	input.filter = filters.Input{
		Page:                 readInt(queryString, "page", 1),
		PageSize:             readInt(queryString, "page_size", 20),
		SortCol:              readString(queryString, "sort", "id"),
		SortSafeList:         []string{"id", "filter", "created_at", "-id", "-filter", "-created_at"},
		Search:               readString(queryString, "search", ""),
		SearchCol:            readString(queryString, "search_field", "title"),
		SearchColumnSafeList: []string{"title", "description"},
	}

	galleries, metadata, err := app.galleries.ListAllOwned(r.Context(), input.filter)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"galleries": galleries, "filter": metadata}, nil)
}

func (app *application) downloadGalleryHandler(w http.ResponseWriter, r *http.Request) {
	galleryID, err := readIDParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	gallery, readCloser, err := app.galleries.Download(r.Context(), galleryID)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	headers := http.Header{
		"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"gallery_%s.tar.gz\"", gallery.Title)},
	}
	app.streamBytes(w, r, readCloser, headers)
}

func (app *application) createGalleriesHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Published   bool   `json:"published"`
	}

	err := readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}

	gallery, err := app.galleries.Insert(r.Context(), store.Gallery{
		Title:       input.Title,
		Description: input.Description,
		Published:   input.Published,
	})
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
}

func (app *application) updateGalleryHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Published   bool   `json:"published"`
	}

	err := readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}
	id, err := readIDParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	gallery, err := app.galleries.Update(r.Context(), store.Gallery{
		ID:          id,
		Title:       input.Title,
		Description: input.Description,
		Published:   input.Published,
	})
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
}

func (app *application) deleteGalleryHandler(w http.ResponseWriter, r *http.Request) {
	id, err := readIDParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.galleries.Delete(r.Context(), id)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"deleted_gallery_id": id}, nil)
}
