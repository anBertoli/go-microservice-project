package tracing

import (
	"context"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// The tracing package provides request tracing via a RequestTrace struct shared via
// contexts. Note that by using the function of this package, other parts of the
// application could safely retrieve an anonymous trace from a context even if
// was not set before.

// Define a private type to avoid collisions in the context values...
type privateKey string

// ...and declare a const of that type.
const requestTraceKey privateKey = "requestTrace"

// Will contain several data about the request lifecycle.
type RequestTrace struct {
	ID         string
	Start      time.Time
	HttpStatus int
	PubMessage interface{}
	PrivateErr error
}

// Enrich the HTTP request with a newly initialized trace.
func NewTraceToRequest(r *http.Request) *http.Request {
	trace := RequestTrace{
		ID:    genRequestID(25),
		Start: time.Now().UTC(),
	}
	return TraceToRequestCtx(r, &trace)
}

// Put a trace into an HTTP request.
func TraceToRequestCtx(r *http.Request, tr *RequestTrace) *http.Request {
	ctx := r.Context()
	childCtx := context.WithValue(ctx, requestTraceKey, tr)
	return r.WithContext(childCtx)
}

// Get a trace into an HTTP request. If the request context doesn't have any
// trace return a default trace with no ID.
func TraceFromRequestCtx(r *http.Request) *RequestTrace {
	ctx := r.Context()
	if trace, ok := ctx.Value(requestTraceKey).(*RequestTrace); ok {
		return trace
	}
	return &RequestTrace{
		ID: "<no request id>",
	}
}

// Put a trace into a context object.
func TraceToCtx(ctx context.Context, tr *RequestTrace) context.Context {
	return context.WithValue(ctx, requestTraceKey, tr)
}

// Retrieve a trace from a context object. If the context doesn't have any trace
// return a default trace with no ID.
func TraceFromCtx(ctx context.Context) *RequestTrace {
	if trace, ok := ctx.Value(requestTraceKey).(*RequestTrace); ok {
		return trace
	}
	return &RequestTrace{
		ID: "<no request id>",
	}
}

// Generate a random string of the requested length.
func genRequestID(length int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
