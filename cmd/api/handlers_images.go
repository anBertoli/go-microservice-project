package main

import (
	"fmt"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/filters"
	"github.com/anBertoli/snap-vault/pkg/store"
)

// This file contains application methods which signatures match the http.HandlerFunc so
// they can be registered as endpoints to our router. These methods act as wrappers
// around the 'core' services of the application. They are used to decouple transport
// dependent logic and issues from the business logic present in the services.

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

	app.sendJSON(w, r, http.StatusOK, env{"images": images, "filter": metadata}, nil)
}

func (app *application) listGalleryImagesHandler(w http.ResponseWriter, r *http.Request) {
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

	galleryID, err := readUrlIntParam(r, "gallery-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	images, metadata, err := app.images.ListForGallery(r.Context(), false, galleryID, filter)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"images": images, "filter": metadata}, nil)
}

func (app *application) listPublicGalleryImagesHandler(w http.ResponseWriter, r *http.Request) {
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

	galleryID, err := readUrlIntParam(r, "gallery-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	images, metadata, err := app.images.ListForGallery(r.Context(), true, galleryID, filter)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"images": images, "filter": metadata}, nil)
}

func (app *application) getPublicImageHandler(w http.ResponseWriter, r *http.Request) {
	imageMode := readMode(r.URL.Query(), "mode", dataMode)
	imageID, err := readUrlIntParam(r, "image-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	switch imageMode {
	case dataMode:
		image, err := app.images.Get(r.Context(), true, imageID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.sendJSON(w, r, http.StatusOK, env{"image": image}, nil)
	case viewMode:
		image, readCloser, err := app.images.Download(r.Context(), true, imageID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.streamBytes(w, r, readCloser, http.Header{
			"Content-Type": []string{image.ContentType},
		})
	case attachmentMode:
		image, readCloser, err := app.images.Download(r.Context(), true, imageID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.streamBytes(w, r, readCloser, http.Header{
			"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"%s\"", image.Title)},
			"Content-Type":        []string{image.ContentType},
		})
	}
}

func (app *application) getImageHandler(w http.ResponseWriter, r *http.Request) {
	imageMode := readMode(r.URL.Query(), "mode", dataMode)
	imageID, err := readUrlIntParam(r, "image-id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	switch imageMode {
	case dataMode:
		image, err := app.images.Get(r.Context(), false, imageID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.sendJSON(w, r, http.StatusOK, env{"image": image}, nil)
	case viewMode:
		image, readCloser, err := app.images.Download(r.Context(), false, imageID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.streamBytes(w, r, readCloser, http.Header{
			"Content-Type": []string{image.ContentType},
		})
	case attachmentMode:
		image, readCloser, err := app.images.Download(r.Context(), false, imageID)
		if err != nil {
			app.encodeError(w, r, err)
			return
		}
		app.streamBytes(w, r, readCloser, http.Header{
			"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"%s\"", image.Title)},
			"Content-Type":        []string{image.ContentType},
		})
	}
}

const maxBodyBytes = 1024 * 1024 * 50

func (app *application) createImageHandler(w http.ResponseWriter, r *http.Request) {
	galleryID, err := readUrlIntParam(r, "gallery-id")
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

	imageID, err := readUrlIntParam(r, "image-id")
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
	imageID, err := readUrlIntParam(r, "image-id")
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
