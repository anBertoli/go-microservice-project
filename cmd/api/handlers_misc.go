package main

import (
	"github.com/anBertoli/snap-vault/pkg/store"
	"net/http"
)

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

func (app *application) listPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	env := env{
		"permissions": store.EditablePermissions,
	}
	app.sendJSON(w, r, http.StatusOK, env, nil)
}
