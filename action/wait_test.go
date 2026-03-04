package action

import (
	"context"
	"errors"
	"testing"

	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

type waitRecorder struct {
	params WaitParams
	result WaitResult
	err    error
}

func (recorder *waitRecorder) wait(browserCtx context.Context, params WaitParams) (WaitResult, error) {
	recorder.params = params
	return recorder.result, recorder.err
}

func TestWaitTool_ReturnsToolNamedBrowseWait(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &waitRecorder{}

	serverTool := WaitTool(manager, recorder.wait)

	assert.Equal(t, "browse_wait", serverTool.Tool.Name)
}

func TestWaitTool_Handler_PassesSelectorAndStateToWaitFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 100}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#loading",
		"state":    "hidden",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "#loading", recorder.params.Selector)
	assert.Equal(t, "hidden", recorder.params.State)
}

func TestWaitTool_Handler_DefaultsTimeoutTo30000AndStateToVisible(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 50}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#loading",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 30000, recorder.params.TimeoutMs)
	assert.Equal(t, "visible", recorder.params.State)
}

func TestWaitTool_Handler_PassesXPathToWaitFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 75}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"xpath": "//div[@id='loaded']",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "//div[@id='loaded']", recorder.params.XPath)
}

func TestWaitTool_Handler_PassesTextToWaitFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 200}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "Loading complete",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "Loading complete", recorder.params.Text)
}

func TestWaitTool_Handler_PassesURLContainsToWaitFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 300}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"url_contains": "/dashboard",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "/dashboard", recorder.params.URLContains)
}

func TestWaitTool_Handler_PassesFunctionToWaitFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 500}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"function": "document.readyState === 'complete'",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "document.readyState === 'complete'", recorder.params.Function)
}

func TestWaitTool_Handler_PassesCustomTimeoutToWaitFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 100}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector":   "#loading",
		"timeout_ms": float64(5000),
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 5000, recorder.params.TimeoutMs)
}

func TestWaitTool_Handler_ReturnsElapsedMs(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{result: WaitResult{ElapsedMs: 150}}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#loading",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, float64(150), parsed["elapsed_ms"])
}

func TestWaitTool_Handler_ReturnsErrorWhenNoStrategyProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestWaitTool_Handler_ReturnsErrorWhenMultipleStrategiesProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#loading",
		"text":     "Loading complete",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestWaitTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &waitRecorder{}
	serverTool := WaitTool(manager, recorder.wait)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"selector": "#loading",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestWaitTool_Handler_ReturnsErrorWhenWaitFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &waitRecorder{err: errors.New("timeout waiting for selector")}
	serverTool := WaitTool(manager, recorder.wait)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#loading",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}
