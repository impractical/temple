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

func LoggingContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, slogCtxKey, logger)
}

type noopHandler struct{}

func (noopHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (noopHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (n noopHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return n
}

func (n noopHandler) WithGroup(name string) slog.Handler {
	return n
}
