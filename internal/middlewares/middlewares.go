package middlewares

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/cyclimse/mcp-scaleway-functions/pkg/slogctx"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewInjectLogger returns a middleware that injects the provided slog.Logger into the context of each request.
func NewInjectLogger(logger *slog.Logger) mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			ctx = slogctx.Inject(ctx, logger.With(slog.String("request_id", uuid.NewString())))

			return next(ctx, method, req)
		}
	}
}

// NewLogging returns a middleware that logs the beginning and end of each request,
// along with any error that may have occurred.
// It fetches the logger from the context, if available, otherwise it uses a default logger.
func NewLogging() mcp.Middleware {
	return func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			logger := slogctx.FromContext(ctx).With(slogAttrFromRequest(method, req)...)

			defer func() {
				if r := recover(); r != nil {
					logger.Error("Request panicked", slog.Any("error", r))
				}
			}()

			startedAt := time.Now()

			logger.Info("Starting request")

			result, err := next(ctx, method, req)

			duration := time.Since(startedAt)
			logger = logger.With(slog.Duration("duration", duration))
			logger = logger.With(slogAttrFromResult(result)...)

			if err != nil {
				logger.Error("Request failed", slog.String("error", err.Error()))
			} else {
				logger.Info("Request succeeded")
			}

			return result, err
		}
	}
}

func slogAttrFromRequest(method string, req mcp.Request) []any {
	attrs := []any{
		slog.String("method", method),
	}

	if callReq, ok := req.(*mcp.CallToolRequest); ok {
		attrs = append(attrs,
			slog.String("tool_name", callReq.Params.Name),
			slogJSON("tool_input", callReq.Params.Arguments),
		)
	}

	return attrs
}

func slogAttrFromResult(result mcp.Result) []any {
	attrs := []any{}

	if callResult, ok := result.(*mcp.CallToolResult); ok {
		attrs = append(attrs, slogJSON("tool_result", callResult.StructuredContent))
	}

	return attrs
}

// slogJSON marshals the given value to JSON and returns a slog.Attr with the given key and the JSON value.
func slogJSON(k string, v any) slog.Attr {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return slog.String(k, "<error marshaling to JSON>")
	}

	return slog.String(k, string(jsonBytes))
}
