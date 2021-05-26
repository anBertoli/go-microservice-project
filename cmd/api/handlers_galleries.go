package main

import (
	"fmt"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// This file contains application methods which signature matches the HTTP handlerFunc one,
// so they can be registered as endpoints to our router. These methods act as wrappers
// around the 'core' services of the application. They are used to decouple transport
// dependent logic and issues from the business logic present in the services.

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
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"galleries": galleries, "filter": metadata}, nil)
}

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
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"galleries": galleries, "filter": metadata}, nil)
}

// Several responses are supported for this endpoint. The gallery could be visualized
// as a JSON-formatted record or downloaded as a tar archive.
func (app *application) getPublicGalleryHandler(w http.ResponseWriter, r *http.Request) {
	galleryMode := readMode(r.URL.Query(), "mode", dataMode)
	galleryID, err := readUrlIntParam(r, "gallery-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	switch galleryMode {
	case attachmentMode, viewMode:
		gallery, readCloser, err := app.galleries.Download(r.Context(), true, galleryID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.streamBytes(w, r, readCloser, http.Header{
			"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"gallery_%s.tar.gz\"", gallery.Title)},
		})
	case dataMode:
		gallery, err := app.galleries.Get(r.Context(), true, galleryID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
	}
}

// Several responses are supported for this endpoint. The gallery could be visualized
// as a JSON-formatted record or downloaded as a tar archive.
func (app *application) getGalleryHandler(w http.ResponseWriter, r *http.Request) {
	galleryMode := readMode(r.URL.Query(), "mode", dataMode)
	galleryID, err := readUrlIntParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	switch galleryMode {
	case viewMode, attachmentMode:
		gallery, readCloser, err := app.galleries.Download(r.Context(), false, galleryID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.streamBytes(w, r, readCloser, http.Header{
			"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"gallery_%s.tar.gz\"", gallery.Title)},
		})
	case dataMode:
		gallery, err := app.galleries.Get(r.Context(), false, galleryID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
	}
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
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"gallery": gallery}, nil)
}

func (app *application) deleteGalleryHandler(w http.ResponseWriter, r *http.Request) {
	id, err := readUrlIntParam(r, "id")
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
