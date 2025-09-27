package slogctx

import (
	"context"
	"log/slog"
)

type ctxKeyInjectedLogger struct{}

// Inject returns a new context with the provided slog.Logger injected.
func Inject(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKeyInjectedLogger{}, logger)
}

// FromContext retrieves the slog.Logger from the context, if available.
// If not, it returns the default slog logger.
func FromContext(ctx context.Context) *slog.Logger {
	logger, ok := ctx.Value(ctxKeyInjectedLogger{}).(*slog.Logger)
	if !ok || logger == nil {
		return slog.Default()
	}

	return logger
}
