package main

import (
	"net/http"
)

// This file contains application methods which signature matches the HTTP handlerFunc one,
// so they can be registered as endpoints to our router. These methods act as wrappers
// around the 'core' services of the application. They are used to decouple transport
// dependent logic and issues from the business logic present in the services.

func (app *application) listUserKeysHandler(w http.ResponseWriter, r *http.Request) {
	keys, err := app.users.ListUserKeys(r.Context())
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": keys}, nil)
}

func (app *application) addUserKeyHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Permissions []string `json:"permissions"`
	}

	err := readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}

	keys, err := app.users.AddUserKey(r.Context(), input.Permissions)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": keys, "permissions": input.Permissions}, nil)
}

func (app *application) editKeyPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Permissions []string `json:"permissions"`
	}

	keyID, err := readUrlIntParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}

	keys, permissions, err := app.users.EditUserKey(r.Context(), keyID, input.Permissions)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": keys, "permissions": permissions}, nil)
}

func (app *application) deleteUserKeyHandler(w http.ResponseWriter, r *http.Request) {
	keyID, err := readUrlIntParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.users.DeleteUserKey(r.Context(), keyID)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"deleted_key_id": keyID}, nil)
}
