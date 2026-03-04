package action

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/s-ajensen/browse-mcp/session"
	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

type navigateRecorder struct {
	url       string
	waitUntil string
	result    NavigateResult
	err       error
}

func (recorder *navigateRecorder) navigate(browserCtx context.Context, url string, waitUntil string) (NavigateResult, error) {
	recorder.url = url
	recorder.waitUntil = waitUntil
	return recorder.result, recorder.err
}

func fakeNavigateFunc(result NavigateResult) NavigateFunc {
	return func(browserCtx context.Context, url string, waitUntil string) (NavigateResult, error) {
		return result, nil
	}
}

func failingNavigateFunc(message string) NavigateFunc {
	return func(browserCtx context.Context, url string, waitUntil string) (NavigateResult, error) {
		return NavigateResult{}, errors.New(message)
	}
}

func TestNavigateTool_ReturnsToolNamedBrowseNavigate(t *testing.T) {
	manager := session.NewManager(5)
	navigate := fakeNavigateFunc(NavigateResult{})

	serverTool := NavigateTool(manager, navigate, 5*time.Second)

	assert.Equal(t, "browse_navigate", serverTool.Tool.Name)
}

func TestNavigateTool_Handler_PassesURLAndWaitUntilToNavigateFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &navigateRecorder{result: NavigateResult{URL: "https://example.com", Title: "Example"}}
	serverTool := NavigateTool(manager, recorder.navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"url":        "https://example.com",
				"wait_until": "domcontentloaded",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", recorder.url)
	assert.Equal(t, "domcontentloaded", recorder.waitUntil)
}

func TestNavigateTool_Handler_DefaultsWaitUntilToLoad(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &navigateRecorder{result: NavigateResult{URL: "https://example.com", Title: "Example"}}
	serverTool := NavigateTool(manager, recorder.navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"url":        "https://example.com",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "load", recorder.waitUntil)
}

func TestNavigateTool_Handler_ReturnsURLAndTitleFromNavigateResult(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	navigate := fakeNavigateFunc(NavigateResult{URL: "https://example.com/page", Title: "Example Page"})
	serverTool := NavigateTool(manager, navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"url":        "https://example.com/page",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "https://example.com/page", parsed["url"])
	assert.Equal(t, "Example Page", parsed["title"])
}

func TestNavigateTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	navigate := fakeNavigateFunc(NavigateResult{})
	serverTool := NavigateTool(manager, navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"url": "https://example.com",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNavigateTool_Handler_ReturnsErrorForUnknownSessionID(t *testing.T) {
	manager := session.NewManager(5)
	navigate := fakeNavigateFunc(NavigateResult{})
	serverTool := NavigateTool(manager, navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": uuid.New().String(),
				"url":        "https://example.com",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNavigateTool_Handler_ReturnsErrorForMissingURL(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	navigate := fakeNavigateFunc(NavigateResult{})
	serverTool := NavigateTool(manager, navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNavigateTool_Handler_ReturnsErrorWhenNavigateFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	serverTool := NavigateTool(manager, failingNavigateFunc("navigation timeout"), 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"url":        "https://example.com",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNavigateTool_Handler_UpdatesSessionCurrentURL(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	navigate := fakeNavigateFunc(NavigateResult{URL: "https://example.com/final", Title: "Final Page"})
	serverTool := NavigateTool(manager, navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"url":        "https://example.com/final",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	found, getErr := manager.Get(sess.ID)
	assert.NoError(t, getErr)
	assert.Equal(t, "https://example.com/final", found.CurrentURL)
}

type historyNavRecorder struct {
	called bool
	result NavigateResult
	err    error
}

func (recorder *historyNavRecorder) navigate(browserCtx context.Context) (NavigateResult, error) {
	recorder.called = true
	return recorder.result, recorder.err
}

func TestBackTool_ReturnsToolNamedBrowseBack(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &historyNavRecorder{}

	serverTool := BackTool(manager, recorder.navigate, 5*time.Second)

	assert.Equal(t, "browse_back", serverTool.Tool.Name)
}

func TestBackTool_Handler_ReturnsURLAndTitle(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &historyNavRecorder{result: NavigateResult{URL: "https://example.com/previous", Title: "Previous Page"}}
	serverTool := BackTool(manager, recorder.navigate, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "https://example.com/previous", parsed["url"])
	assert.Equal(t, "Previous Page", parsed["title"])
}

func TestBackTool_Handler_UpdatesSessionCurrentURL(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &historyNavRecorder{result: NavigateResult{URL: "https://example.com/previous", Title: "Previous Page"}}
	serverTool := BackTool(manager, recorder.navigate, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	found, getErr := manager.Get(sess.ID)
	assert.NoError(t, getErr)
	assert.Equal(t, "https://example.com/previous", found.CurrentURL)
}

func TestBackTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &historyNavRecorder{}
	serverTool := BackTool(manager, recorder.navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestBackTool_Handler_ReturnsErrorWhenBackFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &historyNavRecorder{err: errors.New("cannot go back")}
	serverTool := BackTool(manager, recorder.navigate, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestForwardTool_ReturnsToolNamedBrowseForward(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &historyNavRecorder{}

	serverTool := ForwardTool(manager, recorder.navigate, 5*time.Second)

	assert.Equal(t, "browse_forward", serverTool.Tool.Name)
}

func TestForwardTool_Handler_ReturnsURLAndTitle(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &historyNavRecorder{result: NavigateResult{URL: "https://example.com/next", Title: "Next Page"}}
	serverTool := ForwardTool(manager, recorder.navigate, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "https://example.com/next", parsed["url"])
	assert.Equal(t, "Next Page", parsed["title"])
}

func TestForwardTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &historyNavRecorder{}
	serverTool := ForwardTool(manager, recorder.navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestReloadTool_ReturnsToolNamedBrowseReload(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &historyNavRecorder{}

	serverTool := ReloadTool(manager, recorder.navigate, 5*time.Second)

	assert.Equal(t, "browse_reload", serverTool.Tool.Name)
}

func TestReloadTool_Handler_ReturnsURLAndTitle(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &historyNavRecorder{result: NavigateResult{URL: "https://example.com/current", Title: "Current Page"}}
	serverTool := ReloadTool(manager, recorder.navigate, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "https://example.com/current", parsed["url"])
	assert.Equal(t, "Current Page", parsed["title"])
}

func TestReloadTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &historyNavRecorder{}
	serverTool := ReloadTool(manager, recorder.navigate, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestNavigateTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingNavigate := func(browserCtx context.Context, url string, waitUntil string) (NavigateResult, error) {
		<-browserCtx.Done()
		return NavigateResult{}, browserCtx.Err()
	}
	serverTool := NavigateTool(manager, blockingNavigate, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{"url": "https://example.com"})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}
