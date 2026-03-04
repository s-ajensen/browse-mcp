package action

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/s-ajensen/browse-mcp/session"
	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func fakeGetURLFunc(result GetURLResult) GetURLFunc {
	return func(browserCtx context.Context) (GetURLResult, error) {
		return result, nil
	}
}

func failingGetURLFunc(message string) GetURLFunc {
	return func(browserCtx context.Context) (GetURLResult, error) {
		return GetURLResult{}, errors.New(message)
	}
}

func TestGetURLTool_ReturnsToolNamedBrowseGetURL(t *testing.T) {
	manager := session.NewManager(5)
	getURL := fakeGetURLFunc(GetURLResult{})

	serverTool := GetURLTool(manager, getURL, 5*time.Second)

	assert.Equal(t, "browse_get_url", serverTool.Tool.Name)
}

func TestGetURLTool_Handler_ReturnsURLAndTitle(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	getURL := fakeGetURLFunc(GetURLResult{URL: "https://example.com/page", Title: "Example Page"})
	serverTool := GetURLTool(manager, getURL, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
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

func TestGetURLTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	getURL := fakeGetURLFunc(GetURLResult{})
	serverTool := GetURLTool(manager, getURL, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetURLTool_Handler_ReturnsErrorForUnknownSessionID(t *testing.T) {
	manager := session.NewManager(5)
	getURL := fakeGetURLFunc(GetURLResult{})
	serverTool := GetURLTool(manager, getURL, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": uuid.New().String(),
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetURLTool_Handler_ReturnsErrorWhenGetURLFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	serverTool := GetURLTool(manager, failingGetURLFunc("page not available"), 5*time.Second)
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

type getTextRecorder struct {
	selector string
	result   GetTextResult
}

func (recorder *getTextRecorder) getText(browserCtx context.Context, selector string) (GetTextResult, error) {
	recorder.selector = selector
	return recorder.result, nil
}

func fakeGetTextFunc(result GetTextResult) GetTextFunc {
	return func(browserCtx context.Context, selector string) (GetTextResult, error) {
		return result, nil
	}
}

func failingGetTextFunc(message string) GetTextFunc {
	return func(browserCtx context.Context, selector string) (GetTextResult, error) {
		return GetTextResult{}, errors.New(message)
	}
}

func TestGetTextTool_ReturnsToolNamedBrowseGetText(t *testing.T) {
	manager := session.NewManager(5)
	getText := fakeGetTextFunc(GetTextResult{})

	serverTool := GetTextTool(manager, getText, 5*time.Second)

	assert.Equal(t, "browse_get_text", serverTool.Tool.Name)
}

func TestGetTextTool_Handler_PassesSelectorToGetTextFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &getTextRecorder{result: GetTextResult{Text: "hello"}}
	serverTool := GetTextTool(manager, recorder.getText, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"selector":   "#main-content",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "#main-content", recorder.selector)
}

func TestGetTextTool_Handler_DefaultsSelectorToBody(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &getTextRecorder{result: GetTextResult{Text: "page text"}}
	serverTool := GetTextTool(manager, recorder.getText, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "body", recorder.selector)
}

func TestGetTextTool_Handler_ReturnsTextFromResult(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	getText := fakeGetTextFunc(GetTextResult{Text: "Welcome to the page"})
	serverTool := GetTextTool(manager, getText, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "Welcome to the page", parsed["text"])
}

func TestGetTextTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	getText := fakeGetTextFunc(GetTextResult{})
	serverTool := GetTextTool(manager, getText, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetTextTool_Handler_ReturnsErrorWhenGetTextFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	serverTool := GetTextTool(manager, failingGetTextFunc("text extraction failed"), 5*time.Second)
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

type screenshotRecorder struct {
	selector string
	fullPage bool
	data     []byte
	err      error
}

func (recorder *screenshotRecorder) screenshot(browserCtx context.Context, params ScreenshotParams) ([]byte, error) {
	recorder.selector = params.Selector
	recorder.fullPage = params.FullPage
	return recorder.data, recorder.err
}

func TestScreenshotTool_ReturnsToolNamedBrowseScreenshot(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &screenshotRecorder{}

	serverTool := ScreenshotTool(manager, recorder.screenshot, 5*time.Second)

	assert.Equal(t, "browse_screenshot", serverTool.Tool.Name)
}

func TestScreenshotTool_Handler_ReturnsImageContentWithBase64PNG(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	expectedBytes := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	recorder := &screenshotRecorder{data: expectedBytes}
	serverTool := ScreenshotTool(manager, recorder.screenshot, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Len(t, result.Content, 1)
	imageContent, ok := result.Content[0].(mcp.ImageContent)
	assert.True(t, ok)
	assert.Equal(t, "image", imageContent.Type)
	assert.Equal(t, "image/png", imageContent.MIMEType)
	assert.Equal(t, base64.StdEncoding.EncodeToString(expectedBytes), imageContent.Data)
}

func TestScreenshotTool_Handler_PassesSelectorToScreenshotFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &screenshotRecorder{data: []byte{1, 2, 3}}
	serverTool := ScreenshotTool(manager, recorder.screenshot, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"selector":   "#hero-banner",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "#hero-banner", recorder.selector)
}

func TestScreenshotTool_Handler_PassesFullPageToScreenshotFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &screenshotRecorder{data: []byte{1, 2, 3}}
	serverTool := ScreenshotTool(manager, recorder.screenshot, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
				"full_page":  true,
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, recorder.fullPage)
}

func TestScreenshotTool_Handler_DefaultsSelectorToEmptyAndFullPageToFalse(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &screenshotRecorder{data: []byte{1, 2, 3}}
	serverTool := ScreenshotTool(manager, recorder.screenshot, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "", recorder.selector)
	assert.False(t, recorder.fullPage)
}

func TestScreenshotTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &screenshotRecorder{}
	serverTool := ScreenshotTool(manager, recorder.screenshot, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestScreenshotTool_Handler_ReturnsErrorWhenScreenshotFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &screenshotRecorder{err: errors.New("capture failed")}
	serverTool := ScreenshotTool(manager, recorder.screenshot, 5*time.Second)
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

type getHTMLRecorder struct {
	selector string
	outer    bool
	result   GetHTMLResult
	err      error
}

func (recorder *getHTMLRecorder) getHTML(browserCtx context.Context, params GetHTMLParams) (GetHTMLResult, error) {
	recorder.selector = params.Selector
	recorder.outer = params.Outer
	return recorder.result, recorder.err
}

func TestGetHTMLTool_ReturnsToolNamedBrowseGetHTML(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &getHTMLRecorder{}

	serverTool := GetHTMLTool(manager, recorder.getHTML, 5*time.Second)

	assert.Equal(t, "browse_get_html", serverTool.Tool.Name)
}

func TestGetHTMLTool_Handler_PassesSelectorAndOuterToGetHTMLFunc(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &getHTMLRecorder{result: GetHTMLResult{HTML: "<div>content</div>"}}
	serverTool := GetHTMLTool(manager, recorder.getHTML, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{
		"selector": "#content",
		"outer":    false,
	})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "#content", recorder.selector)
	assert.False(t, recorder.outer)
}

func TestGetHTMLTool_Handler_DefaultsSelectorToEmptyAndOuterToTrue(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &getHTMLRecorder{result: GetHTMLResult{HTML: "<html></html>"}}
	serverTool := GetHTMLTool(manager, recorder.getHTML, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "", recorder.selector)
	assert.True(t, recorder.outer)
}

func TestGetHTMLTool_Handler_ReturnsHTMLFromResult(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &getHTMLRecorder{result: GetHTMLResult{HTML: "<div>content</div>"}}
	serverTool := GetHTMLTool(manager, recorder.getHTML, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, "<div>content</div>", parsed["html"])
}

func TestGetHTMLTool_Handler_ReturnsErrorForMissingSessionID(t *testing.T) {
	manager := session.NewManager(5)
	recorder := &getHTMLRecorder{}
	serverTool := GetHTMLTool(manager, recorder.getHTML, 5*time.Second)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetHTMLTool_Handler_ReturnsErrorWhenGetHTMLFails(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	recorder := &getHTMLRecorder{err: errors.New("html extraction failed")}
	serverTool := GetHTMLTool(manager, recorder.getHTML, 5*time.Second)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestGetURLTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingGetURL := func(browserCtx context.Context) (GetURLResult, error) {
		<-browserCtx.Done()
		return GetURLResult{}, browserCtx.Err()
	}
	serverTool := GetURLTool(manager, blockingGetURL, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}

func TestGetHTMLTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingGetHTML := func(browserCtx context.Context, params GetHTMLParams) (GetHTMLResult, error) {
		<-browserCtx.Done()
		return GetHTMLResult{}, browserCtx.Err()
	}
	serverTool := GetHTMLTool(manager, blockingGetHTML, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}

func TestScreenshotTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingScreenshot := func(browserCtx context.Context, params ScreenshotParams) ([]byte, error) {
		<-browserCtx.Done()
		return nil, browserCtx.Err()
	}
	serverTool := ScreenshotTool(manager, blockingScreenshot, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}

func TestGetTextTool_Handler_ReturnsTimeoutErrorWhenActionExceedsDeadline(t *testing.T) {
	manager, sess := setupManagerWithSession(t)
	blockingGetText := func(browserCtx context.Context, selector string) (GetTextResult, error) {
		<-browserCtx.Done()
		return GetTextResult{}, browserCtx.Err()
	}
	serverTool := GetTextTool(manager, blockingGetText, 1*time.Millisecond)
	request := toolRequest(sess.ID.String(), map[string]any{})

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "timed out")
}
