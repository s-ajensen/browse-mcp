package action

import (
	"context"
	"fmt"
	"time"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/selector"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func resolveTargetFromRequest(request mcp.CallToolRequest) (selector.Resolved, error) {
	args := request.GetArguments()
	params := selector.Params{
		Selector: request.GetString("selector", ""),
		XPath:    request.GetString("xpath", ""),
		Text:     request.GetString("text", ""),
		X:        optionalFloat(args, "x"),
		Y:        optionalFloat(args, "y"),
	}
	return selector.Resolve(params)
}

type ClickParams struct {
	Target     selector.Resolved
	Button     string
	ClickCount int
}

type ClickFunc func(browserCtx context.Context, params ClickParams) error

func parseClickCount(request mcp.CallToolRequest) int {
	args := request.GetArguments()
	return int(floatOrDefault(args, "click_count", 1))
}

func ClickTool(manager *session.Manager, click ClickFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_click",
		mcp.WithDescription("Click an element. Use CSS selectors when you know the DOM structure. Use text matching when you can see a button label in a screenshot. Use coordinates as a last resort."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Description("CSS selector")),
		mcp.WithString("xpath", mcp.Description("XPath selector")),
		mcp.WithString("text", mcp.Description("Text to match")),
		mcp.WithNumber("x", mcp.Description("X coordinate")),
		mcp.WithNumber("y", mcp.Description("Y coordinate")),
		mcp.WithString("button", mcp.Description("Mouse button"), mcp.DefaultString("left")),
		mcp.WithNumber("click_count", mcp.Description("Number of clicks"), mcp.DefaultNumber(1)),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		target, err := resolveTargetFromRequest(request)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		params := ClickParams{
			Target:     target,
			Button:     request.GetString("button", "left"),
			ClickCount: parseClickCount(request),
		}
		err = click(actionCtx, params)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.SuccessResult()
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

type HoverFunc func(browserCtx context.Context, target selector.Resolved) error

func HoverTool(manager *session.Manager, hover HoverFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_hover",
		mcp.WithDescription("Hover over an element"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Description("CSS selector")),
		mcp.WithString("xpath", mcp.Description("XPath selector")),
		mcp.WithString("text", mcp.Description("Text to match")),
		mcp.WithNumber("x", mcp.Description("X coordinate")),
		mcp.WithNumber("y", mcp.Description("Y coordinate")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		target, err := resolveTargetFromRequest(request)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		err = hover(actionCtx, target)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.SuccessResult()
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

type TypeParams struct {
	Selector string
	Text     string
	Clear    bool
	Submit   bool
}

type TypeFunc func(browserCtx context.Context, params TypeParams) error

func TypeTool(manager *session.Manager, typeText TypeFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_type",
		mcp.WithDescription("Type text into a field"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Description("CSS selector")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to type")),
		mcp.WithBoolean("clear", mcp.Description("Clear field before typing"), mcp.DefaultBool(false)),
		mcp.WithBoolean("submit", mcp.Description("Submit after typing"), mcp.DefaultBool(false)),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		text := request.GetString("text", "")
		if text == "" {
			return mcputil.ErrorResult("text is required"), nil
		}
		params := TypeParams{
			Selector: request.GetString("selector", ""),
			Text:     text,
			Clear:    request.GetBool("clear", false),
			Submit:   request.GetBool("submit", false),
		}
		err := typeText(actionCtx, params)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.SuccessResult()
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

type KeyFunc func(browserCtx context.Context, key string) error

func KeyTool(manager *session.Manager, pressKey KeyFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_key",
		mcp.WithDescription("Press a key or key combination"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("key", mcp.Required(), mcp.Description("Key or key combination (e.g. Enter, Escape, Control+a)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		key := request.GetString("key", "")
		if key == "" {
			return mcputil.ErrorResult("key is required"), nil
		}
		err := pressKey(actionCtx, key)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.SuccessResult()
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

type SelectParams struct {
	Selector string
	Value    string
	Label    string
}

type SelectResult struct {
	SelectedValue string `json:"selected_value"`
	SelectedLabel string `json:"selected_label"`
}

type SelectFunc func(browserCtx context.Context, params SelectParams) (SelectResult, error)

func validateSelectStrategy(value string, label string) error {
	if value == "" && label == "" {
		return fmt.Errorf("exactly one of value or label must be provided")
	}
	if value != "" && label != "" {
		return fmt.Errorf("exactly one of value or label must be provided, not both")
	}
	return nil
}

func SelectTool(manager *session.Manager, selectOption SelectFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_select",
		mcp.WithDescription("Choose an option from a select dropdown"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("selector", mcp.Required(), mcp.Description("CSS selector for the select element")),
		mcp.WithString("value", mcp.Description("Select by value attribute")),
		mcp.WithString("label", mcp.Description("Select by visible label text")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actionCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		selectorVal := request.GetString("selector", "")
		if selectorVal == "" {
			return mcputil.ErrorResult("selector is required"), nil
		}
		value := request.GetString("value", "")
		label := request.GetString("label", "")
		err := validateSelectStrategy(value, label)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		params := SelectParams{
			Selector: selectorVal,
			Value:    value,
			Label:    label,
		}
		selectResult, err := selectOption(actionCtx, params)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(selectResult)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
