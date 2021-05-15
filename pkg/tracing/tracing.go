package tracing

import (
	"context"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type privateKey string

const requestTraceKey privateKey = "requestTrace"

type RequestTrace struct {
	ID         string
	Start      time.Time
	HttpStatus int
	Message    interface{}
	Err        error
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

func LoggerWithRequestID(ctx context.Context, logger *zap.SugaredLogger) *zap.SugaredLogger {
	trace, ok := ctx.Value(requestTraceKey).(*RequestTrace)
	if !ok {
		trace = &RequestTrace{
			ID: "<no request id>",
		}
	}
	logger.With("id", trace.ID)
	return logger
}

func GenRequestID(length int) string {
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().UnixNano())
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
