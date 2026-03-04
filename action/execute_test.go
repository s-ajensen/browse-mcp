package action

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

type evalRecorder struct {
	expression string
	result     any
	err        error
}

func (recorder *evalRecorder) eval(browserCtx context.Context, expression string) (any, error) {
	recorder.expression = expression
	return recorder.result, recorder.err
}

func TestEvalTool_ReturnsToolNamedBrowseEval(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &evalRecorder{}

	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)

	assert.Equal(t, "browse_eval", serverTool.Tool.Name)
}

func TestEvalTool_Handler_PassesExpressionToEvalFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &evalRecorder{result: "ok"}
	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"expression": "document.title",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "document.title", recorder.expression)
}

func TestEvalTool_Handler_ReturnsEvalResult(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &evalRecorder{result: map[string]any{"count": float64(42)}}
	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"expression": "({count: 42})",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed EvalResult
	testutil.UnmarshalToolResult(t, result, &parsed)
	resultMap, ok := parsed.Result.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, float64(42), resultMap["count"])
}

func TestEvalTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &evalRecorder{}
	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"expression": "document.title",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEvalTool_Handler_ReturnsErrorForMissingExpression(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &evalRecorder{}
	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEvalTool_Handler_ReturnsErrorWhenEvalFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &evalRecorder{err: errors.New("eval failed")}
	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"expression": "badCode()",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEvalTool_Handler_HandlesStringResult(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &evalRecorder{result: "hello"}
	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"expression": "'hello'",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed EvalResult
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "hello", parsed.Result)
}

func TestEvalTool_Handler_HandlesNullResult(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &evalRecorder{result: nil}
	serverTool := EvalTool(manager, recorder.eval, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"expression": "void 0",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	rawResult, exists := parsed["result"]
	assert.True(t, exists)
	assert.Equal(t, json.RawMessage("null"), toRawJSON(t, rawResult))
}

func TestEvalTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingEval := func(browserCtx context.Context, expression string) (any, error) {
		<-browserCtx.Done()
		return nil, browserCtx.Err()
	}
	serverTool := EvalTool(manager, blockingEval, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{
		"expression": "document.title",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}

func toRawJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	assert.NoError(t, err)
	return data
}
