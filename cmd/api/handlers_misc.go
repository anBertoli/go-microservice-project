package main

import (
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/store"
)

// Simple healthcheck handler that returns info about the app.
func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	env := env{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.Env,
			"version":     version,
		},
	}
	app.sendJSON(w, r, http.StatusOK, env, nil)
}

// Documentation handler that list all editable permissions.
func (app *application) listPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	env := env{
		"permissions": store.EditablePermissions,
	}
	app.sendJSON(w, r, http.StatusOK, env, nil)
}
