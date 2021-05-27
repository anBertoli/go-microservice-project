package tracing

import (
	"context"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// The tracing package provides request tracing via a RequestTrace struct shared via
// contexts. The functions present here could be used to create, set and retrieve
// request traces data from contexts objects.
//
// Note that by using the function of this package, other parts of the application
// could retrieve an anonymous but safe trace from a context even if was not set
// before.

type privateKey string

const requestTraceKey privateKey = "requestTrace"

type RequestTrace struct {
	ID         string
	Start      time.Time
	HttpStatus int
	PubMessage interface{}
	PrivateErr error
}

func NewTraceToRequest(r *http.Request) *http.Request {
	trace := RequestTrace{
		ID:    genRequestID(25),
		Start: time.Now().UTC(),
	}
	return TraceToRequestCtx(r, &trace)
}

func TraceToRequestCtx(r *http.Request, tr *RequestTrace) *http.Request {
	ctx := r.Context()
	childCtx := context.WithValue(ctx, requestTraceKey, tr)
	return r.WithContext(childCtx)
}

func TraceFromRequestCtx(r *http.Request) *RequestTrace {
	ctx := r.Context()
	if trace, ok := ctx.Value(requestTraceKey).(*RequestTrace); ok {
		return trace
	}
	return &RequestTrace{
		ID: "<no request id>",
	}
}

func TraceToCtx(ctx context.Context, tr *RequestTrace) context.Context {
	return context.WithValue(ctx, requestTraceKey, tr)
}

func TraceFromCtx(ctx context.Context) *RequestTrace {
	if trace, ok := ctx.Value(requestTraceKey).(*RequestTrace); ok {
		return trace
	}
	return &RequestTrace{
		ID: "<no request id>",
	}
}

func genRequestID(length int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
