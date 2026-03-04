package action

import (
	"context"
	"testing"

	"github.com/s-ajensen/browse-mcp/session"
	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
)

func fakeHandler(result *mcp.CallToolResult) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return result, nil
	}
}

func failingHandler(message string) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: message}},
			IsError: true,
		}, nil
	}
}

func recordingHandler(result *mcp.CallToolResult, received *mcp.CallToolRequest) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		*received = request
		return result, nil
	}
}

func successResult(text string) *mcp.CallToolResult {
	return mcp.NewToolResultText(text)
}

func TestBatchTool_ReturnsToolNamedBrowseBatch(t *testing.T) {
	manager := session.NewManager(5)
	registry := ToolRegistry{}

	serverTool := BatchTool(manager, registry)

	assert.Equal(t, "browse_batch", serverTool.Tool.Name)
}

func TestBatchTool_Handler_ExecutesActionsSequentially(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	registry := ToolRegistry{
		"navigate": fakeHandler(successResult(`{"url":"https://example.com","title":"Example"}`)),
		"get_url":  fakeHandler(successResult(`{"url":"https://example.com"}`)),
	}
	serverTool := BatchTool(manager, registry)
	request := toolRequest(sess.ID.String(), map[string]any{
		"actions": []any{
			map[string]any{"tool": "navigate", "params": map[string]any{"url": "https://example.com"}},
			map[string]any{"tool": "get_url", "params": map[string]any{}},
		},
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var batch BatchResult
	testutil.UnmarshalToolResult(t, result, &batch)
	assert.Equal(t, 2, len(batch.Results))
	assert.True(t, batch.Results[0].Success)
	assert.True(t, batch.Results[1].Success)
	assert.Equal(t, 2, batch.Completed)
	assert.Equal(t, 2, batch.Total)
}

func TestBatchTool_Handler_StopsOnErrorByDefault(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	registry := ToolRegistry{
		"navigate":   fakeHandler(successResult(`{"url":"https://example.com"}`)),
		"wait":       failingHandler("element not found"),
		"screenshot": fakeHandler(successResult(`{"data":"base64..."}`)),
	}
	serverTool := BatchTool(manager, registry)
	request := toolRequest(sess.ID.String(), map[string]any{
		"actions": []any{
			map[string]any{"tool": "navigate", "params": map[string]any{"url": "https://example.com"}},
			map[string]any{"tool": "wait", "params": map[string]any{"selector": "#loaded"}},
			map[string]any{"tool": "screenshot", "params": map[string]any{}},
		},
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var batch BatchResult
	testutil.UnmarshalToolResult(t, result, &batch)
	assert.Equal(t, 2, len(batch.Results))
	assert.True(t, batch.Results[0].Success)
	assert.False(t, batch.Results[1].Success)
	assert.Equal(t, 1, batch.Completed)
	assert.Equal(t, 3, batch.Total)
}

func TestBatchTool_Handler_ContinuesOnErrorWhenStopOnErrorFalse(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	registry := ToolRegistry{
		"navigate":   fakeHandler(successResult(`{"url":"https://example.com"}`)),
		"wait":       failingHandler("element not found"),
		"screenshot": fakeHandler(successResult(`{"data":"base64..."}`)),
	}
	serverTool := BatchTool(manager, registry)
	request := toolRequest(sess.ID.String(), map[string]any{
		"stop_on_error": false,
		"actions": []any{
			map[string]any{"tool": "navigate", "params": map[string]any{"url": "https://example.com"}},
			map[string]any{"tool": "wait", "params": map[string]any{"selector": "#loaded"}},
			map[string]any{"tool": "screenshot", "params": map[string]any{}},
		},
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var batch BatchResult
	testutil.UnmarshalToolResult(t, result, &batch)
	assert.Equal(t, 3, len(batch.Results))
	assert.True(t, batch.Results[0].Success)
	assert.False(t, batch.Results[1].Success)
	assert.True(t, batch.Results[2].Success)
	assert.Equal(t, 2, batch.Completed)
	assert.Equal(t, 3, batch.Total)
}

func TestBatchTool_Handler_PassesSessionIDToSubActions(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	var received mcp.CallToolRequest
	registry := ToolRegistry{
		"navigate": recordingHandler(successResult(`{"url":"https://example.com"}`), &received),
	}
	serverTool := BatchTool(manager, registry)
	request := toolRequest(sess.ID.String(), map[string]any{
		"actions": []any{
			map[string]any{"tool": "navigate", "params": map[string]any{"url": "https://example.com"}},
		},
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, sess.ID.String(), received.GetString("session_id", ""))
}

func TestBatchTool_Handler_ReturnsErrorForUnknownTool(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	registry := ToolRegistry{}
	serverTool := BatchTool(manager, registry)
	request := toolRequest(sess.ID.String(), map[string]any{
		"actions": []any{
			map[string]any{"tool": "nonexistent", "params": map[string]any{}},
		},
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var batch BatchResult
	testutil.UnmarshalToolResult(t, result, &batch)
	assert.Equal(t, 1, len(batch.Results))
	assert.False(t, batch.Results[0].Success)
	assert.Contains(t, batch.Results[0].Error, "unknown")
}

func TestBatchTool_Handler_ReturnsErrorForMissingActions(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	registry := ToolRegistry{}
	serverTool := BatchTool(manager, registry)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBatchTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	registry := ToolRegistry{}
	serverTool := BatchTool(manager, registry)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"actions": []any{
					map[string]any{"tool": "navigate", "params": map[string]any{}},
				},
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBatchTool_Handler_ReturnsEmptyResultsForEmptyActions(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	registry := ToolRegistry{}
	serverTool := BatchTool(manager, registry)
	request := toolRequest(sess.ID.String(), map[string]any{
		"actions": []any{},
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var batch BatchResult
	testutil.UnmarshalToolResult(t, result, &batch)
	assert.Equal(t, 0, len(batch.Results))
	assert.Equal(t, 0, batch.Completed)
	assert.Equal(t, 0, batch.Total)
}
