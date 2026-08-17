// Package logger provides structured logging. Every call site logs a
// static message with dynamic data passed as key-value pairs, so log lines
// stay aggregable across occurrences - see go-coding-standards.
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/Laaaaksh/gohighlevel-round1/internal/constants/contextkeys"
)

const fieldRequestID = "requestID"

var base = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// New returns the base structured logger, for call sites without a request
// context (e.g. startup, shutdown, cmd/seed).
func New() *slog.Logger {
	return base
}

// Ctx returns a logger enriched with the request id carried on ctx, if any,
// so every log line from one request can be correlated.
func Ctx(ctx context.Context) *slog.Logger {
	requestID, ok := ctx.Value(contextkeys.RequestIDKey).(string)
	if !ok || requestID == "" {
		return base
	}
	return base.With(slog.String(fieldRequestID, requestID))
}
