package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/store"
	"github.com/anBertoli/snap-vault/pkg/validator"
	"github.com/anBertoli/snap-vault/services/galleries"
	"github.com/anBertoli/snap-vault/services/images"
	"github.com/anBertoli/snap-vault/services/users"
)

func (app *application) encodeError(w http.ResponseWriter, r *http.Request, err error) {
	v := validator.New()

	switch {
	case errors.As(err, &v):
		app.failedValidationResponse(w, r, v)

	// store errors
	case errors.Is(err, store.ErrDuplicateEmail):
		app.emailTakenResponse(w, r)
	case errors.Is(err, store.ErrRecordNotFound):
		app.notFoundResponse(w, r)
	case errors.Is(err, store.ErrEditConflict):
		app.editConflictResponse(w, r)
	case errors.Is(err, store.ErrUnauthenticated):
		app.unauthenticatedResponse(w, r)
	case errors.Is(err, store.ErrForbidden):
		app.forbiddenResponse(w, r)
	case errors.Is(err, store.ErrNotActivated):
		app.inactiveAccountResponse(w, r)
	case errors.Is(err, store.ErrNoPermission):
		app.wrongPermissionsResponse(w, r)

	// users service errors
	case errors.Is(err, users.ErrMainKeysEdit):
		app.notEditableKeysResponse(w, r)

	// galleries service errors
	case errors.Is(err, galleries.ErrBusy):
		app.tooBusyResponse(w, r)

	// images service errors
	case errors.Is(err, images.ErrMaxSpaceReached):
		app.maxSpaceReachedResponse(w, r)

	// default to 500 errors
	default:
		app.serverErrorResponse(w, r, err)
	}
}

// These are generic responses given back to the user. Below there are more specific
// error responses that may utilize the same HTTP code but differ for the returned message.
func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.sendJSONError(w, r, errResponse{
		message: "the server encountered a problem and could not process your request",
		status:  http.StatusInternalServerError,
		err:     err,
	})
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("the requested resource could not be found")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusNotFound,
		err:     err,
	})
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusBadRequest,
		err:     err,
	})
}

func (app *application) forbiddenResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("you don't have rights to perform this action")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusForbidden,
		err:     err,
	})
}

func (app *application) unauthenticatedResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("you must be authenticated to access this resource")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusUnauthorized,
		err:     err,
	})
}

func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("unable to update the resource due to a conflict, please try again")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusConflict,
		err:     err,
	})
}

func (app *application) tooBusyResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("the server is currently too busy to process your request")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusTooManyRequests,
		err:     err,
	})
}

// Errors responses used by the router.
func (app *application) routeNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	err := errors.New("the requested API endpoint doesn't exist")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusNotFound,
		err:     err,
	})
}

func (app *application) methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	err := fmt.Errorf("the %s method is not supported for this endpoint", r.Method)
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusMethodNotAllowed,
		err:     err,
	})
}

// More specific error responses.
func (app *application) malformedJSONResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusBadRequest,
		err:     err,
	})
}

func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors validator.Validator) {
	app.sendJSONError(w, r, errResponse{
		message: errors,
		status:  http.StatusUnprocessableEntity,
		err:     errors,
	})
}

func (app *application) emailTakenResponse(w http.ResponseWriter, r *http.Request) {
	app.sendJSONError(w, r, errResponse{
		message: map[string]string{"email": "a user with this email address already exists"},
		status:  http.StatusUnprocessableEntity,
		err:     store.ErrDuplicateEmail,
	})
}

func (app *application) notEditableKeysResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("main keys cannot be edited or deleted")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusForbidden,
		err:     err,
	})
}

func (app *application) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("your user account must be activated to access this resource")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusForbidden,
		err:     err,
	})
}

func (app *application) wrongPermissionsResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("you don't have the right permission to perform this action")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusForbidden,
		err:     err,
	})
}

func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	err := errors.New("the provided authentication token is invalid")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusUnauthorized,
		err:     err,
	})
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("rate limit exceeded")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusTooManyRequests,
		err:     err,
	})
}

func (app *application) maxSpaceReachedResponse(w http.ResponseWriter, r *http.Request) {
	err := errors.New("max space reached, delete some images and retry")
	app.sendJSONError(w, r, errResponse{
		message: err.Error(),
		status:  http.StatusNotAcceptable,
		err:     err,
	})
}
