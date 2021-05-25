package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// Decode the request body into the target destination.
func readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1048576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Initialize the json.Decoder, and call the DisallowUnknownFields() method on it
	// before decoding. This means that if the JSON from the ipLimiter now includes any
	// field which cannot be mapped to the target destination, the decoder will return
	// an error instead of just ignoring the field.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err == nil {
		// Call Decode() again, using a pointer to an empty anonymous struct as the
		// destination. If the request body only contained a single JSON value this will
		// return an io.EOF error. So if we get anything else, we know that there is
		// additional data in the request body and we return our own custom error message.
		err = dec.Decode(&struct{}{})
		if err != io.EOF {
			return errors.New("body must only contain a single JSON value")
		}
		return nil
	}

	// If there is an error during decoding, start the triage...
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var invalidUnmarshalError *json.InvalidUnmarshalError
	switch {
	// Use the errors.As() function to check whether the error has the type
	// *json.SyntaxError. If it does, then return a plain-english error message
	// which includes the location of the problem.
	case errors.As(err, &syntaxError):
		return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
	// In some circumstances Decode() may also return an io.ErrUnexpectedEOF error
	// for syntax errors in the JSON. So we check for this using errors.Is() and
	// return a generic error message. There is an open issue regarding this at
	// https://github.com/golang/go/issues/25956.
	case errors.Is(err, io.ErrUnexpectedEOF):
		return errors.New("body contains badly-formed JSON")
	// Likewise, catch any *json.UnmarshalTypeError errors. These occur when the
	// JSON value is the wrong type for the target destination. If the error relates
	// to a specific field, then we include that in our error message to make it
	// easier for the ipLimiter to debug.
	case errors.As(err, &unmarshalTypeError):
		if unmarshalTypeError.Field != "" {
			return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
		}
		return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
	// An io.EOF error will be returned by Decode() if the request body is empty. We
	// check for this with errors.Is() and return a plain-english error message
	// instead.
	case errors.Is(err, io.EOF):
		return errors.New("body must not be empty")
	// A json.InvalidUnmarshalError error will be returned if we pass a non-nil
	// pointer to Decode(). We catch this and panic, rather than returning an error
	// to our handler. At the end of this chapter we'll talk about panicking
	// versus returning errors, and discuss why it's an appropriate thing to do in
	// this specific situation.
	case errors.As(err, &invalidUnmarshalError):
		panic(err)
	// If the JSON contains a field which cannot be mapped to the target destination
	// then Decode() will now return an error message in the format "json: unknown
	// field "<name>"". We check for this, extract the field name from the error,
	// and interpolate it into our custom error message. Note that there's an open
	// issue at https://github.com/golang/go/issues/29035 regarding turning this
	// into a distinct error type in the future.
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
		return fmt.Errorf("body contains unknown key %s", fieldName)
	// If the request body exceeds 1MB in size the decode will now fail with the
	// error "http: request body too large". There is an open issue about turning
	// this into a distinct error type at https://github.com/golang/go/issues/30715.
	case err.Error() == "http: request body too large":
		return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
	// For anything else, return the error message as-is.
	default:
		return err
	}

}

func readIDParam(r *http.Request, param string) (int64, error) {
	params := mux.Vars(r)
	id, err := strconv.ParseInt(params[param], 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid %s parameter", param)
	}
	return id, nil
}

func readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value for a given key from the query string. If no key exists this
	// will return the empty string "".
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Otherwise return the string.
	return s
}

// The readInt() helper reads a string value from the query string and converts it to an
// integer before returning. If no matching key count be found it returns the provided
// default value. If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func readInt(qs url.Values, key string, defaultValue int) int {
	// Extract the value from the query string.
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to an int. If this fails, add an error message to the
	// validator instance and return the default value.
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	// Otherwise, return the converted integer value.
	return i
}

const (
	dataMode       = "data"
	attachmentMode = "download"
	downloadMode   = "view"
)

func readImageMode(qs url.Values, key string, defaultValue string) string {
	// Extract the value from the query string.
	s := qs.Get(key)
	// If no key exists (or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to an int. If this fails, add an error message to the
	// validator instance and return the default value.
	found := false
	for _, m := range []string{attachmentMode, downloadMode, dataMode} {
		if m == s {
			found = true
		}
	}
	if !found {
		return defaultValue
	}
	return s
}
