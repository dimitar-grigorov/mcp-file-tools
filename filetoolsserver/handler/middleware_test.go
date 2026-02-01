package handler

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type testInput struct {
	Value string `json:"value"`
}

type testOutput struct {
	Result string `json:"result"`
}

func TestWithRecovery_NoPanic(t *testing.T) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, testOutput, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "success"}},
		}, testOutput{Result: "ok"}, nil
	}

	wrapped := WithRecovery(handler)
	result, output, err := wrapped(context.Background(), &mcp.CallToolRequest{}, testInput{Value: "test"})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result.IsError {
		t.Error("expected non-error result")
	}
	if output.Result != "ok" {
		t.Errorf("expected output 'ok', got %q", output.Result)
	}
}

func TestWithRecovery_Panic(t *testing.T) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, testOutput, error) {
		panic("test panic")
	}

	wrapped := WithRecovery(handler)
	result, _, err := wrapped(context.Background(), &mcp.CallToolRequest{}, testInput{Value: "test"})

	if err == nil {
		t.Error("expected error from panic recovery")
	}
	if !strings.Contains(err.Error(), "panic recovered") {
		t.Errorf("expected panic recovery message, got %v", err)
	}
	if !strings.Contains(err.Error(), "test panic") {
		t.Errorf("expected original panic message, got %v", err)
	}
	if result == nil || !result.IsError {
		t.Error("expected error result")
	}
}

func TestWithRecovery_PanicWithNilValue(t *testing.T) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, testOutput, error) {
		panic(nil)
	}

	wrapped := WithRecovery(handler)
	result, _, err := wrapped(context.Background(), &mcp.CallToolRequest{}, testInput{Value: "test"})

	// panic(nil) should still be recovered
	if err == nil {
		t.Error("expected error from panic recovery")
	}
	if result == nil || !result.IsError {
		t.Error("expected error result")
	}
}

func TestWithLogging_Success(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, testOutput, error) {
		return &mcp.CallToolResult{}, testOutput{Result: "ok"}, nil
	}

	wrapped := WithLogging(logger, "test_tool", handler)
	_, _, _ = wrapped(context.Background(), &mcp.CallToolRequest{}, testInput{})

	logOutput := buf.String()
	if !strings.Contains(logOutput, "tool_call_start") {
		t.Error("expected tool_call_start log")
	}
	if !strings.Contains(logOutput, "tool_call_success") {
		t.Error("expected tool_call_success log")
	}
	if !strings.Contains(logOutput, "test_tool") {
		t.Error("expected tool name in log")
	}
}

func TestWithLogging_ToolError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, testOutput, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "something went wrong"}},
			IsError: true,
		}, testOutput{}, nil
	}

	wrapped := WithLogging(logger, "test_tool", handler)
	_, _, _ = wrapped(context.Background(), &mcp.CallToolRequest{}, testInput{})

	logOutput := buf.String()
	if !strings.Contains(logOutput, "tool_call_failed") {
		t.Error("expected tool_call_failed log")
	}
	if !strings.Contains(logOutput, "something went wrong") {
		t.Error("expected error message in log")
	}
}

func TestWithLogging_NilLogger(t *testing.T) {
	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, testOutput, error) {
		return &mcp.CallToolResult{}, testOutput{Result: "ok"}, nil
	}

	// Should not panic with nil logger
	wrapped := WithLogging(nil, "test_tool", handler)
	result, output, err := wrapped(context.Background(), &mcp.CallToolRequest{}, testInput{})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if output.Result != "ok" {
		t.Errorf("expected output 'ok', got %q", output.Result)
	}
	if result == nil {
		t.Error("expected non-nil result")
	}
}

func TestWrap_CombinesMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := func(ctx context.Context, req *mcp.CallToolRequest, input testInput) (*mcp.CallToolResult, testOutput, error) {
		panic("test panic in wrapped handler")
	}

	wrapped := Wrap(logger, "test_tool", handler)
	result, _, err := wrapped(context.Background(), &mcp.CallToolRequest{}, testInput{})

	// Should recover from panic
	if err == nil {
		t.Error("expected error from panic recovery")
	}
	if result == nil || !result.IsError {
		t.Error("expected error result")
	}

	// Should log the error (panic causes error return which triggers Error log)
	logOutput := buf.String()
	if !strings.Contains(logOutput, "tool_call_start") {
		t.Error("expected tool_call_start log")
	}
	if !strings.Contains(logOutput, "tool_call_error") {
		t.Error("expected tool_call_error log")
	}
}
