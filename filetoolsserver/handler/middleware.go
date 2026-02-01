package handler

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WithRecovery wraps a tool handler with panic recovery.
// If a panic occurs, it returns an error result instead of crashing the server.
func WithRecovery[In, Out any](handler mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args In) (result *mcp.CallToolResult, output Out, err error) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				err = fmt.Errorf("panic recovered in tool handler: %v\n%s", r, stack)
				result = errorResult(fmt.Sprintf("internal error: panic in tool handler: %v", r))
			}
		}()
		return handler(ctx, req, args)
	}
}

// WithLogging wraps a tool handler with request/response logging.
// Logs tool name, duration, and any errors.
func WithLogging[In, Out any](logger *slog.Logger, toolName string, handler mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	if logger == nil {
		return handler
	}
	return func(ctx context.Context, req *mcp.CallToolRequest, args In) (*mcp.CallToolResult, Out, error) {
		logger.Debug("tool_call_start", "tool", toolName)

		result, output, err := handler(ctx, req, args)

		if err != nil {
			logger.Error("tool_call_error", "tool", toolName, "error", err)
		} else if result != nil && result.IsError {
			// Extract error message from result content
			var errMsg string
			if len(result.Content) > 0 {
				if tc, ok := result.Content[0].(*mcp.TextContent); ok {
					errMsg = tc.Text
				}
			}
			logger.Warn("tool_call_failed", "tool", toolName, "message", errMsg)
		} else {
			logger.Debug("tool_call_success", "tool", toolName)
		}

		return result, output, err
	}
}

// Wrap applies recovery and optional logging to a tool handler.
// This is the main entry point for wrapping handlers.
func Wrap[In, Out any](logger *slog.Logger, toolName string, handler mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	// Apply recovery first (outermost), then logging
	wrapped := WithRecovery(handler)
	if logger != nil {
		wrapped = WithLogging(logger, toolName, wrapped)
	}
	return wrapped
}
