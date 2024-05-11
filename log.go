package temple

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

var (
	slogCtxKey = ctxKey{}
)

func logger(ctx context.Context) *slog.Logger {
	val := ctx.Value(slogCtxKey)
	if val == nil {
		return slog.New(noopHandler{})
	}
	logger, ok := val.(*slog.Logger)
	if !ok {
		return slog.New(noopHandler{})
	}
	return logger
}

// LoggingContext returns a context.Context with the slog.Logger embedded in it
// in such a way that temple will be able to find it. Passing the returned
// context.Context to temple functions will let temple write its logging output
// to that logger.
func LoggingContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, slogCtxKey, logger)
}

type noopHandler struct{}

// Enabled always returns false for the noopHandler.
func (noopHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

// Handle always returns a nil error and does nothing else for the noopHandler.
func (noopHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

// WithAttrs always returns the noopHandler as it was before WithAttrs was
// called.
func (n noopHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return n
}

// WithGroup always returns the noopHandler as it was before WithGroup was
// called.
func (n noopHandler) WithGroup(_ string) slog.Handler {
	return n
}
