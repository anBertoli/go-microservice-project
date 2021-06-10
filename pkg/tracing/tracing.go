package tracing

import (
	"context"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// The tracing package provides request tracing via a RequestTrace struct shared via
// contexts. Note that by using the function in this package, other parts of the
// application could safely retrieve an anonymous trace from a context even if
// it wasn't set before.

// Define a private type to avoid collisions in the context values...
type privateKey string

// ...and declare a const of that type.
const requestTraceKey privateKey = "requestTrace"

// Contains several data about the request lifecycle.
type RequestTrace struct {
	ID         string
	Start      time.Time
	HttpCode   int
	PublicErr  interface{}
	PrivateErr error
}

// Enrich the HTTP request with a newly initialized trace.
func NewRequestWithTrace(r *http.Request) *http.Request {
	trace := RequestTrace{
		ID:    genRequestID(25),
		Start: time.Now().UTC(),
	}
	return TraceToRequestCtx(r, &trace)
}

// Put a trace into the HTTP request.
func TraceToRequestCtx(r *http.Request, tr *RequestTrace) *http.Request {
	ctx := r.Context()
	childCtx := context.WithValue(ctx, requestTraceKey, tr)
	return r.WithContext(childCtx)
}

// Get a request trace from the HTTP request. If the request context doesn't have any
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

// Retrieve a trace from a context object. If the context doesn't have any trace
// return a default request trace with no ID.
func TraceFromCtx(ctx context.Context) *RequestTrace {
	if trace, ok := ctx.Value(requestTraceKey).(*RequestTrace); ok {
		return trace
	}
	return &RequestTrace{
		ID: "<no request id>",
	}
}

// Generate a random string of the indicated length.
func genRequestID(length int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
