package action

import (
	"context"
	"time"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type NavigateResult struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

type NavigateFunc func(browserCtx context.Context, url string, waitUntil string) (NavigateResult, error)

type HistoryNavFunc func(browserCtx context.Context) (NavigateResult, error)

func NavigateTool(manager *session.Manager, navigate NavigateFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_navigate",
		mcp.WithDescription("Navigate to a URL"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("url", mcp.Required(), mcp.Description("The URL to navigate to")),
		mcp.WithString("wait_until", mcp.Description("When to consider navigation complete"), mcp.DefaultString("load")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		url := request.GetString("url", "")
		if url == "" {
			return mcputil.ErrorResult("url is required"), nil
		}
		waitUntil := request.GetString("wait_until", "load")
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		result, err := navigate(actionCtx, url, waitUntil)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		found.CurrentURL = result.URL
		found.Title = result.Title
		return mcputil.JSONToolResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func historyNavTool(name string, description string, manager *session.Manager, navFunc HistoryNavFunc, timeout time.Duration) server.ServerTool {
	tool := mcp.NewTool(name,
		mcp.WithDescription(description),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, timeout)
		defer cancel()
		result, err := navFunc(actionCtx)
		if err != nil {
			if timeoutResult := timeoutError(err, timeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		found.CurrentURL = result.URL
		found.Title = result.Title
		return mcputil.JSONToolResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func BackTool(manager *session.Manager, goBack HistoryNavFunc, actionTimeout time.Duration) server.ServerTool {
	return historyNavTool("browse_back", "Go back in browser history", manager, goBack, actionTimeout)
}

func ForwardTool(manager *session.Manager, goForward HistoryNavFunc, actionTimeout time.Duration) server.ServerTool {
	return historyNavTool("browse_forward", "Go forward in browser history", manager, goForward, actionTimeout)
}

func ReloadTool(manager *session.Manager, reload HistoryNavFunc, actionTimeout time.Duration) server.ServerTool {
	return historyNavTool("browse_reload", "Reload the current page", manager, reload, actionTimeout)
}
