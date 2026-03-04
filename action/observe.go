package action

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type GetURLResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type GetURLFunc func(browserCtx context.Context) (GetURLResult, error)

type GetTextResult struct {
	Text string `json:"text"`
}

type GetTextFunc func(browserCtx context.Context, selector string) (GetTextResult, error)

type GetHTMLParams struct {
	Selector string
	Outer    bool
}

type GetHTMLResult struct {
	HTML string `json:"html"`
}

type GetHTMLFunc func(browserCtx context.Context, params GetHTMLParams) (GetHTMLResult, error)

type ScreenshotParams struct {
	Selector string
	FullPage bool
}

type ScreenshotFunc func(browserCtx context.Context, params ScreenshotParams) ([]byte, error)

func GetURLTool(manager *session.Manager, getURL GetURLFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_get_url",
		mcp.WithDescription("Get current page URL and title"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		result, err := getURL(actionCtx)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func GetTextTool(manager *session.Manager, getText GetTextFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_get_text",
		mcp.WithDescription("Extract text content from the page or a specific element. Returns visible text, stripped of HTML."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Description("CSS selector for the element to extract text from"), mcp.DefaultString("body")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		selector := request.GetString("selector", "body")
		result, err := getText(actionCtx, selector)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func GetHTMLTool(manager *session.Manager, getHTML GetHTMLFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_get_html",
		mcp.WithDescription("Get HTML content of the page or a specific element"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Description("CSS selector for the element to extract HTML from")),
		mcp.WithBoolean("outer", mcp.Description("Return outer HTML including the element itself"), mcp.DefaultBool(true)),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		params := GetHTMLParams{
			Selector: request.GetString("selector", ""),
			Outer:    request.GetBool("outer", true),
		}
		result, err := getHTML(actionCtx, params)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func ScreenshotTool(manager *session.Manager, screenshot ScreenshotFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_screenshot",
		mcp.WithDescription("Capture the visible page or a specific element as a PNG screenshot"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Description("CSS selector for the element to capture")),
		mcp.WithBoolean("full_page", mcp.Description("Capture the full scrollable page"), mcp.DefaultBool(false)),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		params := ScreenshotParams{
			Selector: request.GetString("selector", ""),
			FullPage: request.GetBool("full_page", false),
		}
		data, err := screenshot(actionCtx, params)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.ImageContent{
					Annotated: mcp.Annotated{},
					Type:      "image",
					MIMEType:  "image/png",
					Data:      base64.StdEncoding.EncodeToString(data),
				},
			},
		}, nil
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
