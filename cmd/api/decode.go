package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	maxBytesBody = 1048576
)

// The readJSON helper is used to decode the request body into the target destination.
func readJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {

	// Limit the size of the request body to 1MB.
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytesBody))

	// Read all the request body in memory as raw bytes. The body cannot be empty.
	jsonBytes, err := io.ReadAll(r.Body)
	if err != nil {
		switch {
		// Body > 1MB in size.
		case err.Error() == "http: request body too large":
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesBody)
		default:
			return err
		}
	}

	if len(jsonBytes) == 0 {
		return errors.New("body must not be empty")
	}

	// Try to unmarshal the bytes into the destination. If there is an error
	// during un-marshaling, try to return an informative error, otherwise
	// the destination will store the decoded JSON values.
	err = json.Unmarshal(jsonBytes, dst)
	if err == nil {
		return nil
	}

	// This type of error checking is excessive for normal applications but it is
	// reported here with an illustrative purpose.
	var invalidUnmarshalError *json.InvalidUnmarshalError
	var unmarshalTypeError *json.UnmarshalTypeError
	var syntaxError *json.SyntaxError

	switch {
	// Use the errors.As() function to check whether the error has the type
	// *json.SyntaxError. If it does, then return a verbose error message
	// which includes the location of the problem.
	case errors.As(err, &syntaxError):
		return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

	// Likewise, catch any *json.UnmarshalTypeError errors. These occur when the
	// JSON value is the wrong type for the target destination. If the error relates
	// to a specific field, then we include that in our error message to make it
	// easier for the client to debug.
	case errors.As(err, &unmarshalTypeError):
		if unmarshalTypeError.Field != "" {
			return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
		}
		return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

	// A json.InvalidUnmarshalError error will be returned if we pass a nil pointer
	// to json.Unmarshal(). We catch this and panic, rather than returning an error,
	// because this is a developer error that must not happen.
	case errors.As(err, &invalidUnmarshalError):
		panic(err)

	// For anything else, return the error message as-is.
	default:
		return err
	}
}

// Extract a numeric value from the URL params provided by the router.
func readUrlIntParam(r *http.Request, param string) (int64, error) {
	params := mux.Vars(r)
	id, err := strconv.ParseInt(params[param], 10, 64)
	if err != nil || id < 1 {
		return 0, fmt.Errorf("invalid %s parameter", param)
	}
	return id, nil
}

// Extract the value for a given key from the query string. If no key exists this
// will default to the provided value.
func readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

// Extract the value for a given key from the query string. If no key exists, or the
// value is not a numeric value, the function will default to the provided value.
func readInt(qs url.Values, key string, defaultValue int) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return i
}

const (
	dataMode       = "data"
	attachmentMode = "attachment"
	viewMode       = "view"
)

// Extract the value for the image mode key from the query string and make sure it
// is one of the allowed modes.
func readMode(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	found := false
	for _, m := range []string{attachmentMode, viewMode, dataMode} {
		if m == s {
			found = true
		}
	}
	if !found {
		return defaultValue
	}
	return s
}
