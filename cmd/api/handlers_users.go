package main

import (
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/store"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}

	user, keys, token, err := app.users.RegisterUser(r.Context(), input.Name, input.Email, input.Password)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	// Launch a background goroutine to send the activation email.
	app.background(func() {
		mailData := map[string]interface{}{
			"activationToken": token,
			// TODO
			"hostName": "http://127.0.0.1:4000",
			"userID":   user.ID,
			"name":     user.Name,
		}
		err = app.mailer.Send(user.Email, "user_welcome.gohtml", mailData)
		if err != nil {
			app.logger.Errorf("sending activation mail", "err", err)
		}
	})

	app.sendJSON(w, r, http.StatusOK, env{"user": user, "keys": keys}, nil)
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	user, err := app.users.ActivateUser(r.Context(), token)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"user": user}, nil)
}

func (app *application) genKeyRecoveryTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}

	plainToken, err := app.users.GenKeyRecoveryToken(r.Context(), input.Email, input.Password)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.background(func() {
		mailData := map[string]interface{}{
			"recoverToken": plainToken,
			"hostName":     "http://127.0.0.1:4000",
		}
		err = app.mailer.Send(input.Email, "recover_key.gohtml", mailData)
		if err != nil {
			app.logger.Errorf("sending recover key mail", "err", err)
		} else {
			app.logger.Infof("recover key mail sent")
		}
	})

	app.sendJSON(w, r, http.StatusOK, env{
		"message": "you will receive an email with instructions to regenerate main keys",
	}, nil)
}

func (app *application) recoverKeyHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	key, err := app.users.RecoverMainKey(r.Context(), token)
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": key}, nil)
}

func (app *application) getUserAccountHandler(w http.ResponseWriter, r *http.Request) {
	authData := store.ContextGetAuth(r.Context())
	if authData == nil {
		app.unauthenticatedResponse(w, r)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{
		"user":        authData.User,
		"keys":        authData.Keys,
		"permissions": authData.Permissions,
	}, nil)
}

func (app *application) getUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := app.users.GetStats(r.Context())
	if err != nil {
		app.encodeError(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"stats": stats}, nil)
}
