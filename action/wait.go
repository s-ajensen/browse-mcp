package action

import (
	"context"
	"fmt"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type WaitParams struct {
	Selector    string
	XPath       string
	Text        string
	URLContains string
	Function    string
	TimeoutMs   int
	State       string
}

type WaitResult struct {
	ElapsedMs int64 `json:"elapsed_ms"`
}

type WaitFunc func(browserCtx context.Context, params WaitParams) (WaitResult, error)

func countWaitStrategies(params WaitParams) int {
	count := 0
	if params.Selector != "" {
		count++
	}
	if params.XPath != "" {
		count++
	}
	if params.Text != "" {
		count++
	}
	if params.URLContains != "" {
		count++
	}
	if params.Function != "" {
		count++
	}
	return count
}

func validateWaitStrategy(params WaitParams) error {
	count := countWaitStrategies(params)
	if count == 0 {
		return fmt.Errorf("exactly one wait strategy must be provided (selector, xpath, text, url_contains, or function)")
	}
	if count > 1 {
		return fmt.Errorf("only one wait strategy allowed, but %d were provided", count)
	}
	return nil
}

func WaitTool(manager *session.Manager, wait WaitFunc) server.ServerTool {
	tool := mcp.NewTool("browse_wait",
		mcp.WithDescription("Wait for a condition before proceeding"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Description("CSS selector to wait for")),
		mcp.WithString("xpath", mcp.Description("XPath expression to wait for")),
		mcp.WithString("text", mcp.Description("Text to appear on the page")),
		mcp.WithString("url_contains", mcp.Description("Substring the URL must contain")),
		mcp.WithString("function", mcp.Description("JS expression returning truthy")),
		mcp.WithNumber("timeout_ms", mcp.Description("Timeout in milliseconds")),
		mcp.WithString("state", mcp.Description("Element state to wait for (for selector strategy)"), mcp.DefaultString("visible")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		params := WaitParams{
			Selector:    request.GetString("selector", ""),
			XPath:       request.GetString("xpath", ""),
			Text:        request.GetString("text", ""),
			URLContains: request.GetString("url_contains", ""),
			Function:    request.GetString("function", ""),
			TimeoutMs:   int(floatOrDefault(request.GetArguments(), "timeout_ms", 30000)),
			State:       request.GetString("state", "visible"),
		}
		validationErr := validateWaitStrategy(params)
		if validationErr != nil {
			return mcputil.ErrorResult(validationErr.Error()), nil
		}
		result, err := wait(found.BrowserCtx, params)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(result)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
