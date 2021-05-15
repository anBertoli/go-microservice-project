package main

import (
	"net/http"
)

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

	keyID, err := readIDParam(r, "id")
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
	keyID, err := readIDParam(r, "id")
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
