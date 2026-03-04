package action

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

type scrollRecorder struct {
	params ScrollParams
	result ScrollResult
	err    error
}

func (recorder *scrollRecorder) scroll(browserCtx context.Context, params ScrollParams) (ScrollResult, error) {
	recorder.params = params
	return recorder.result, recorder.err
}

func TestScrollTool_ReturnsToolNamedBrowseScroll(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &scrollRecorder{}

	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)

	assert.Equal(t, "browse_scroll", serverTool.Tool.Name)
}

func TestScrollTool_Handler_PassesDirectionAndAmountToScrollFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{result: ScrollResult{ScrollX: 0, ScrollY: 500}}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"direction": "down",
		"amount":    float64(500),
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "down", recorder.params.Direction)
	assert.Equal(t, float64(500), recorder.params.Amount)
}

func TestScrollTool_Handler_DefaultsAmountToZeroWhenDirectionProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{result: ScrollResult{}}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"direction": "down",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "down", recorder.params.Direction)
	assert.Equal(t, float64(0), recorder.params.Amount)
}

func TestScrollTool_Handler_PassesSelectorToScrollFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{result: ScrollResult{}}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#footer",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "#footer", recorder.params.Selector)
}

func TestScrollTool_Handler_PassesDeltasToScrollFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{result: ScrollResult{}}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"delta_x": float64(100),
		"delta_y": float64(200),
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, float64(100), recorder.params.DeltaX)
	assert.Equal(t, float64(200), recorder.params.DeltaY)
}

func TestScrollTool_Handler_ReturnsScrollPosition(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{result: ScrollResult{ScrollX: 0, ScrollY: 500}}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"direction": "down",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, float64(0), parsed["scroll_x"])
	assert.Equal(t, float64(500), parsed["scroll_y"])
}

func TestScrollTool_Handler_ReturnsErrorWhenNoStrategyProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestScrollTool_Handler_ReturnsErrorWhenMultipleStrategiesProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"direction": "down",
		"selector":  "#footer",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestScrollTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &scrollRecorder{}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"direction": "down",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestScrollTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingScroll := func(browserCtx context.Context, params ScrollParams) (ScrollResult, error) {
		<-browserCtx.Done()
		return ScrollResult{}, browserCtx.Err()
	}
	serverTool := ScrollTool(manager, blockingScroll, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{
		"direction": "down",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}

func TestScrollTool_Handler_ReturnsErrorWhenScrollFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &scrollRecorder{err: errors.New("scroll failed")}
	serverTool := ScrollTool(manager, recorder.scroll, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"direction": "down",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}
