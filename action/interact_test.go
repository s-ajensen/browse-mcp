package action

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/s-ajensen/browse-mcp/selector"
	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

type clickRecorder struct {
	target     selector.Resolved
	button     string
	clickCount int
	err        error
}

func (recorder *clickRecorder) click(browserCtx context.Context, params ClickParams) error {
	recorder.target = params.Target
	recorder.button = params.Button
	recorder.clickCount = params.ClickCount
	return recorder.err
}

func TestClickTool_ReturnsToolNamedBrowseClick(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &clickRecorder{}

	serverTool := ClickTool(manager, recorder.click, 5*time.Second)

	assert.Equal(t, "browse_click", serverTool.Tool.Name)
}

func TestClickTool_Handler_ResolvesCSSSelector(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, selector.KindCSS, recorder.target.Kind)
	assert.Equal(t, "div.btn", recorder.target.Selector)
}

func TestClickTool_Handler_ResolvesTextSelector(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "Submit",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, selector.KindText, recorder.target.Kind)
}

func TestClickTool_Handler_ResolvesXPathSelector(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"xpath": "//button",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, selector.KindXPath, recorder.target.Kind)
	assert.Equal(t, "//button", recorder.target.Selector)
}

func TestClickTool_Handler_ResolvesCoordinates(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"x": float64(100),
		"y": float64(200),
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, selector.KindCoordinates, recorder.target.Kind)
	assert.Equal(t, float64(100), recorder.target.X)
	assert.Equal(t, float64(200), recorder.target.Y)
}

func TestClickTool_Handler_DefaultsButtonToLeft(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "left", recorder.button)
}

func TestClickTool_Handler_UsesCustomButton(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
		"button":   "right",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "right", recorder.button)
}

func TestClickTool_Handler_DefaultsClickCountToOne(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 1, recorder.clickCount)
}

func TestClickTool_Handler_ReturnsErrorWhenNoSelectorProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestClickTool_Handler_ReturnsErrorWhenMultipleSelectorsProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
		"xpath":    "//button",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestClickTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"selector": "div.btn",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestClickTool_Handler_ReturnsErrorWhenClickFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{err: errors.New("click failed")}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestClickTool_Handler_ReturnsSuccessJSON(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &clickRecorder{}
	serverTool := ClickTool(manager, recorder.click, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, true, parsed["success"])
}

func TestClickTool_DescriptionIncludesUsageHint(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &clickRecorder{}

	serverTool := ClickTool(manager, recorder.click, 5*time.Second)

	description := serverTool.Tool.Description
	assert.Contains(t, description, "CSS selectors")
	assert.Contains(t, description, "text matching")
	assert.Contains(t, description, "coordinates")
}

type typeRecorder struct {
	selector string
	text     string
	clear    bool
	submit   bool
	err      error
}

func (recorder *typeRecorder) typeText(browserCtx context.Context, params TypeParams) error {
	recorder.selector = params.Selector
	recorder.text = params.Text
	recorder.clear = params.Clear
	recorder.submit = params.Submit
	return recorder.err
}

func TestTypeTool_ReturnsToolNamedBrowseType(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &typeRecorder{}

	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)

	assert.Equal(t, "browse_type", serverTool.Tool.Name)
}

func TestTypeTool_Handler_PassesTextAndSelectorToTypeFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "input.name",
		"text":     "hello world",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "input.name", recorder.selector)
	assert.Equal(t, "hello world", recorder.text)
}

func TestTypeTool_Handler_AllowsEmptySelector(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "hello world",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "", recorder.selector)
}

func TestTypeTool_Handler_DefaultsClearToFalse(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "hello",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.False(t, recorder.clear)
}

func TestTypeTool_Handler_DefaultsSubmitToFalse(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "hello",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.False(t, recorder.submit)
}

func TestTypeTool_Handler_UsesCustomClearAndSubmit(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text":   "hello",
		"clear":  true,
		"submit": true,
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, recorder.clear)
	assert.True(t, recorder.submit)
}

func TestTypeTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"text": "hello",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTypeTool_Handler_ReturnsErrorForMissingText(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "input.name",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTypeTool_Handler_ReturnsErrorWhenTypeFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{err: errors.New("type failed")}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "hello",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestTypeTool_Handler_ReturnsSuccessJSON(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &typeRecorder{}
	serverTool := TypeTool(manager, recorder.typeText, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "hello",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, true, parsed["success"])
}

type hoverRecorder struct {
	target selector.Resolved
	err    error
}

func (recorder *hoverRecorder) hover(browserCtx context.Context, target selector.Resolved) error {
	recorder.target = target
	return recorder.err
}

func TestHoverTool_ReturnsToolNamedBrowseHover(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &hoverRecorder{}

	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)

	assert.Equal(t, "browse_hover", serverTool.Tool.Name)
}

func TestHoverTool_Handler_ResolvesCSSSelector(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &hoverRecorder{}
	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.menu",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, selector.KindCSS, recorder.target.Kind)
	assert.Equal(t, "div.menu", recorder.target.Selector)
}

func TestHoverTool_Handler_ResolvesTextSelector(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &hoverRecorder{}
	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"text": "Menu Item",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, selector.KindText, recorder.target.Kind)
}

func TestHoverTool_Handler_ResolvesCoordinates(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &hoverRecorder{}
	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"x": float64(150),
		"y": float64(250),
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, selector.KindCoordinates, recorder.target.Kind)
	assert.Equal(t, float64(150), recorder.target.X)
	assert.Equal(t, float64(250), recorder.target.Y)
}

func TestHoverTool_Handler_ReturnsSuccessJSON(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &hoverRecorder{}
	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.menu",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, true, parsed["success"])
}

func TestHoverTool_Handler_ReturnsErrorWhenNoSelectorProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &hoverRecorder{}
	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHoverTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &hoverRecorder{}
	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"selector": "div.menu",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestHoverTool_Handler_ReturnsErrorWhenHoverFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &hoverRecorder{err: errors.New("hover failed")}
	serverTool := HoverTool(manager, recorder.hover, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.menu",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

type selectRecorder struct {
	params SelectParams
	result SelectResult
	err    error
}

func (recorder *selectRecorder) selectOption(browserCtx context.Context, params SelectParams) (SelectResult, error) {
	recorder.params = params
	return recorder.result, recorder.err
}

func TestSelectTool_ReturnsToolNamedBrowseSelect(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &selectRecorder{}

	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)

	assert.Equal(t, "browse_select", serverTool.Tool.Name)
}

func TestSelectTool_Handler_PassesSelectorAndValueToSelectFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &selectRecorder{result: SelectResult{SelectedValue: "us", SelectedLabel: "United States"}}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#country",
		"value":    "us",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "#country", recorder.params.Selector)
	assert.Equal(t, "us", recorder.params.Value)
}

func TestSelectTool_Handler_PassesSelectorAndLabelToSelectFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &selectRecorder{result: SelectResult{SelectedValue: "us", SelectedLabel: "United States"}}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#country",
		"label":    "United States",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "#country", recorder.params.Selector)
	assert.Equal(t, "United States", recorder.params.Label)
}

func TestSelectTool_Handler_ReturnsSelectedValueAndLabel(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &selectRecorder{result: SelectResult{SelectedValue: "us", SelectedLabel: "United States"}}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#country",
		"value":    "us",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "us", parsed["selected_value"])
	assert.Equal(t, "United States", parsed["selected_label"])
}

func TestSelectTool_Handler_ReturnsErrorForMissingSelector(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &selectRecorder{}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"value": "us",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSelectTool_Handler_ReturnsErrorWhenNeitherValueNorLabelProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &selectRecorder{}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#country",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSelectTool_Handler_ReturnsErrorWhenBothValueAndLabelProvided(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &selectRecorder{}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#country",
		"value":    "us",
		"label":    "United States",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSelectTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &selectRecorder{}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"selector": "#country",
				"value":    "us",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSelectTool_Handler_ReturnsErrorWhenSelectFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &selectRecorder{err: errors.New("select failed")}
	serverTool := SelectTool(manager, recorder.selectOption, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#country",
		"value":    "us",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

type keyRecorder struct {
	key string
	err error
}

func (recorder *keyRecorder) pressKey(browserCtx context.Context, key string) error {
	recorder.key = key
	return recorder.err
}

func TestKeyTool_ReturnsToolNamedBrowseKey(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &keyRecorder{}

	serverTool := KeyTool(manager, recorder.pressKey, 5*time.Second)

	assert.Equal(t, "browse_key", serverTool.Tool.Name)
}

func TestKeyTool_Handler_PassesKeyToKeyFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &keyRecorder{}
	serverTool := KeyTool(manager, recorder.pressKey, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"key": "Enter",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "Enter", recorder.key)
}

func TestKeyTool_Handler_PassesKeyCombinationToKeyFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &keyRecorder{}
	serverTool := KeyTool(manager, recorder.pressKey, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"key": "Control+a",
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "Control+a", recorder.key)
}

func TestKeyTool_Handler_ReturnsSuccessJSON(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &keyRecorder{}
	serverTool := KeyTool(manager, recorder.pressKey, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"key": "Enter",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, true, parsed["success"])
}

func TestKeyTool_Handler_ReturnsErrorForMissingKey(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &keyRecorder{}
	serverTool := KeyTool(manager, recorder.pressKey, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestKeyTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager, _ := setupManagerWithSession(t)
	recorder := &keyRecorder{}
	serverTool := KeyTool(manager, recorder.pressKey, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"key": "Enter",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestKeyTool_Handler_ReturnsErrorWhenKeyFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &keyRecorder{err: errors.New("key press failed")}
	serverTool := KeyTool(manager, recorder.pressKey, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"key": "Enter",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestClickTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingClick := func(browserCtx context.Context, params ClickParams) error {
		<-browserCtx.Done()
		return browserCtx.Err()
	}
	serverTool := ClickTool(manager, blockingClick, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "div.btn",
	})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}
