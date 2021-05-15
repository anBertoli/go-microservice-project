package main

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/anBertoli/snap-vault/pkg/tracing"
)

type env map[string]interface{}

func (app *application) sendJSON(w http.ResponseWriter, r *http.Request, status int, data env, headers http.Header) {

	trace := tracing.TraceFromRequestCtx(r)
	trace.HttpStatus = status
	trace.Err = nil

	err := writeJSON(w, status, data, headers)
	if err != nil {
		app.logger.Errorw("sending json", "id", trace.ID, "err", err)
		trace.HttpStatus = http.StatusInternalServerError
		trace.Err = err
	}
}

// The sendJSONError() method is a generic helper for sending JSON-formatted error
// messages to the ipLimiter with a given status code. Note that we're using an interface{}
// type for the message parameter, rather than just a string type, as this gives callers
// more flexibility over the values that we can include in the response.
func (app *application) sendJSONError(w http.ResponseWriter, r *http.Request, resp errResponse) {

	trace := tracing.TraceFromRequestCtx(r)
	trace.HttpStatus = resp.status
	trace.Message = resp.message
	trace.Err = resp.err

	// Write the response using the sendJSON() helper. If this happens to return an
	// error then log it, and fall back to sending the ipLimiter an empty response with a
	// 500 Internal Server Error status code.
	err := writeJSON(w, resp.status, env{
		"status_code": resp.status,
		"error":       resp.message,
	}, nil)
	if err != nil {
		app.logger.Errorw("sending json", "id", trace.ID, "err", err)
		trace.HttpStatus = http.StatusInternalServerError
		trace.Err = err
	}
}

type errResponse struct {
	message interface{}
	status  int
	err     error
}

// Define a sendJSON() helper for sending responses. This takes the destination
// http.ResponseWriter, the HTTP status code to send, the data to encode to JSON, and a
// header map containing any additional HTTP headers we want to include in the response.
func writeJSON(w http.ResponseWriter, status int, data env, headers http.Header) error {
	// Encode the data to JSON, returning the error if there was one.
	// Avoid Indent in case of performance issues.
	js, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Append a newline to make it easier to view in terminal applications.
	js = append(js, '\n')

	// At this point, we know that we won't encounter any more errors before writing the
	// response, so it's safe to add any headers that we want to include. We loop
	// through the header map and add each header to the http.ResponseWriter header map.
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Add the "Content-Type: application/json" header, then write the status code and
	// JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(js)
	return err
}

func (app *application) streamBytes(w http.ResponseWriter, r *http.Request, reader io.Reader, headers http.Header) {

	trace := tracing.TraceFromRequestCtx(r)
	trace.HttpStatus = http.StatusOK
	trace.Err = nil

	logger := app.logger.With("id", trace.ID)

	if rc, ok := reader.(io.ReadCloser); ok {
		defer func() {
			// If the reader is also a readCloser then defer a function that will close it.
			// When need to close the reader because if a network issue happens, we need to
			// signal to the application internals that the streaming must be exited. If the
			// reader has been completely drained this operation is a no-op.
			err := rc.Close()
			if err != nil {
				logger.Errorw("error closing file reader", "err", err)
			}
		}()
	}

	for key, value := range headers {
		w.Header()[key] = value
	}

	_, err := io.Copy(w, reader)
	if err != nil {
		var netErr *net.OpError
		switch {
		case errors.As(err, &netErr):
			// This is a network/client issue. We cannot do nothing here so simply
			// log the error. This is not an HTTP error, but still report it with
			// the status 500.
			logger.Errorw("network/client issue streaming file", "err", err)
		default:
			// The error is originated internally and in this case the error is returned
			// from the readCloser which will be the appropriate one. Here the status code
			// on the response is already set and we cannot modify it, but we cann report
			// this internally.
			logger.Errorw("internal error streaming file", "err", err)
		}

		trace.HttpStatus = http.StatusInternalServerError
		trace.Err = err
	}
}
