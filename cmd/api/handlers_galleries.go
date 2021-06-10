package main

import (
	"fmt"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// List public galleries. Filtering and pagination is supported and specified via
// query parameters.
func (app *application) listPublicGalleriesHandler(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	filter := filters.Input{
		Page:                 readInt(queryString, "page", 1),
		PageSize:             readInt(queryString, "page_size", 20),
		SortCol:              readString(queryString, "sort", "id"),
		SortSafeList:         []string{"id", "filter", "created_at", "-id", "-filter", "-created_at"},
		Search:               readString(queryString, "search", ""),
		SearchCol:            readString(queryString, "search_field", "title"),
		SearchColumnSafeList: []string{"title", "description"},
	}

	galleries, metadata, err := app.galleries.ListAllPublic(r.Context(), filter)
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"galleries": galleries, "filter": metadata}, nil)
}

// List galleries owned by the authenticated user. Filtering and pagination is supported and
// specified via query parameters.
func (app *application) listGalleriesHandler(w http.ResponseWriter, r *http.Request) {
	queryString := r.URL.Query()
	filter := filters.Input{
		Page:                 readInt(queryString, "page", 1),
		PageSize:             readInt(queryString, "page_size", 20),
		SortCol:              readString(queryString, "sort", "id"),
		SortSafeList:         []string{"id", "filter", "created_at", "-id", "-filter", "-created_at"},
		Search:               readString(queryString, "search", ""),
		SearchCol:            readString(queryString, "search_field", "title"),
		SearchColumnSafeList: []string{"title", "description"},
	}

	galleries, metadata, err := app.galleries.ListAllOwned(r.Context(), filter)
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"galleries": galleries, "filter": metadata}, nil)
}

// Get a specific public gallery. The response mode is specified via the query string,
// while the gallery ID is specified in the URL parameters.
func (app *application) getPublicGalleryHandler(w http.ResponseWriter, r *http.Request) {

	// Parse the 'get' mode from the query string.
	galleryMode := readMode(r.URL.Query(), "mode", dataMode)

	galleryID, err := readUrlIntParam(r, "gallery-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Several responses are supported for this endpoint. The gallery could be visualized
	// as a JSON-formatted record or downloaded as a tar archive.
	switch galleryMode {
	case attachmentMode, viewMode:
		gallery, readCloser, err := app.galleries.Download(r.Context(), true, galleryID)
		if err != nil {
			app.errorResponse(w, r, err)
			return
		}
		app.streamBytes(w, r, http.StatusOK, readCloser, http.Header{
			"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"gallery_%s.tar.gz\"", gallery.Title)},
		})
	case dataMode:
		gallery, err := app.galleries.Get(r.Context(), true, galleryID)
		if err != nil {
			app.errorResponse(w, r, err)
			return
		}
		app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
	}
}

// Download a specific gallery owned by the authenticated user. The response mode is specified
// via the query string, while the gallery ID is specified in the URL parameters.
func (app *application) getGalleryHandler(w http.ResponseWriter, r *http.Request) {

	// Parse the 'get' mode from the query string.
	galleryMode := readMode(r.URL.Query(), "mode", dataMode)

	galleryID, err := readUrlIntParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Several responses are supported for this endpoint. The gallery could be visualized
	// as a JSON-formatted record or downloaded as a tar archive.
	switch galleryMode {
	case viewMode, attachmentMode:
		gallery, readCloser, err := app.galleries.Download(r.Context(), false, galleryID)
		if err != nil {
			app.errorResponse(w, r, err)
			return
		}
		app.streamBytes(w, r, http.StatusOK, readCloser, http.Header{
			"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"gallery_%s.tar.gz\"", gallery.Title)},
		})
	case dataMode:
		gallery, err := app.galleries.Get(r.Context(), false, galleryID)
		if err != nil {
			app.errorResponse(w, r, err)
			return
		}
		app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
	}
}

// Create a new gallery reading the mandatory data from the JSON-formatted body.
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
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
}

// Update an existing gallery reading the data to be used from the JSON-formatted body.
// The gallery to be updated is specified in the URL parameters.
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
	id, err := readUrlIntParam(r, "id")
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
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
}

// Delete an existing gallery. The gallery ID is parsed form the URL parameters.
func (app *application) deleteGalleryHandler(w http.ResponseWriter, r *http.Request) {
	id, err := readUrlIntParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.galleries.Delete(r.Context(), id)
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"deleted_gallery_id": id}, nil)
}
