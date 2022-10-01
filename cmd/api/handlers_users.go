package main

import (
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/tracing"
)

// Register a new user into the system. The user must be activated before using
// the newly generated auth key. User data must be provided in the JSON body.
// User activation is performed via a token sent via email.
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
		app.errorResponse(w, r, err)
		return
	}

	// Launch a background goroutine to send the activation email.
	logger := app.logger.With("id", tracing.TraceFromRequestCtx(r).ID)

	app.background(func() {
		mailData := map[string]interface{}{
			"activationToken": token,
			"hostName":        app.config.PublicHostname,
			"userID":          user.ID,
			"name":            user.Name,
		}
		err = app.mailer.Send(user.Email, "user_welcome.gohtml", mailData)
		if err != nil {
			logger.Errorw("sending activation mail", "err", err)
			return
		}
		logger.Infof("activation mail sent")
	})

	app.sendJSON(w, r, http.StatusOK, env{
		"message": "an email will be sent to you containing activation instructions",
		"user":    user,
		"keys":    keys,
	}, nil)
}

// Activate the user using the provided activation token.
func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	user, err := app.users.ActivateUser(r.Context(), token)
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"user": user}, nil)
}

// Regenerate the activation token for a not activated user. Useful if the first
// email was lost or incorrectly sent.
func (app *application) regenerateActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := readJSON(w, r, &input)
	if err != nil {
		app.malformedJSONResponse(w, r, err)
		return
	}

	user, token, err := app.users.RegenerateActivationToken(r.Context(), input.Email, input.Password)
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	// Launch a background goroutine to send the activation email.
	logger := app.logger.With("id", tracing.TraceFromRequestCtx(r).ID)

	app.background(func() {
		mailData := map[string]interface{}{
			"activationToken": token,
			"hostName":        app.config.PublicHostname,
			"userID":          user.ID,
			"name":            user.Name,
		}
		err = app.mailer.Send(user.Email, "user_welcome.gohtml", mailData)
		if err != nil {
			logger.Errorw("sending activation mail", "err", err)
			return
		}
		logger.Infof("activation mail sent")
	})

	app.sendJSON(w, r, http.StatusOK, env{
		"message": "an email will be sent to you containing activation instructions",
	}, nil)
}

// Start the process of main auth key recovery. The user is authenticated via email and
// password and a token is sent via email.
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
		app.errorResponse(w, r, err)
		return
	}

	// Launch a background goroutine to send the recover token email.
	logger := app.logger.With("id", tracing.TraceFromRequestCtx(r).ID)

	app.background(func() {
		mailData := map[string]interface{}{
			"recoverToken": plainToken,
			"hostName":     app.config.PublicHostname,
		}
		err = app.mailer.Send(input.Email, "recover_key.gohtml", mailData)
		if err != nil {
			logger.Errorw("sending recover key mail", "err", err)
			return
		}
		app.logger.Infof("recover key mail sent")
	})

	app.sendJSON(w, r, http.StatusOK, env{
		"message": "you will receive an email with instructions to regenerate the main auth key",
	}, nil)
}

// Regenerate the main auth key using the token provided to the user in the
// handler immediately above.
func (app *application) recoverKeyHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")

	key, err := app.users.RegenerateMainKey(r.Context(), token)
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"keys": key}, nil)
}

// Retrieve information about the user authenticated, namely, itself.
func (app *application) getUserAccountHandler(w http.ResponseWriter, r *http.Request) {
	authData, err := app.users.GetMe(r.Context())
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{
		"user":        authData.User,
		"keys":        authData.Keys,
		"permissions": authData.Perms,
	}, nil)
}

// Retrieve usage statistics about the user authenticated.
func (app *application) getUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	stats, err := app.users.GetStats(r.Context())
	if err != nil {
		app.errorResponse(w, r, err)
		return
	}

	app.sendJSON(w, r, http.StatusOK, env{"stats": stats}, nil)
}
