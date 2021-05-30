package main

import (
	"net/http"
)

// List the user keys. Only requests authenticated with main keys will
// obtain the complete list of keys.
func (app *application) listUserKeysHandler(w http.ResponseWriter, r *http.Request) {
	keys, err := app.users.ListUserKeys(r.Context())
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": keys}, nil)
}

// Add a new auth key for the user. Permissions are read from the JSON-formatted body.
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
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": keys, "permissions": input.Permissions}, nil)
}

// Edit an existing auth key of the user. Permissions are read from the JSON-formatted body,
// while the ID of the auth key is parsed from URL parameters.
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
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": keys, "permissions": permissions}, nil)
}

// Delete an existing auth key of the user. The ID of the auth key is parsed from URL parameters.
func (app *application) deleteUserKeyHandler(w http.ResponseWriter, r *http.Request) {
	keyID, err := readUrlIntParam(r, "id")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.users.DeleteUserKey(r.Context(), keyID)
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"deleted_key_id": keyID}, nil)
}
