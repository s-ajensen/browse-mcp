package action

import (
	"context"
	"fmt"
	"time"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ScrollParams struct {
	Direction string
	Amount    float64
	Selector  string
	DeltaX    float64
	DeltaY    float64
}

type ScrollResult struct {
	ScrollX float64 `json:"scroll_x"`
	ScrollY float64 `json:"scroll_y"`
}

type ScrollFunc func(browserCtx context.Context, params ScrollParams) (ScrollResult, error)

func validateScrollStrategy(args map[string]any) error {
	strategyCount := 0
	direction, _ := args["direction"].(string)
	if direction != "" {
		strategyCount++
	}
	selectorVal, _ := args["selector"].(string)
	if selectorVal != "" {
		strategyCount++
	}
	deltaX := optionalFloat(args, "delta_x")
	deltaY := optionalFloat(args, "delta_y")
	if deltaX != nil || deltaY != nil {
		strategyCount++
	}
	if strategyCount != 1 {
		return fmt.Errorf("exactly one scroll strategy required: provide direction, selector, or delta_x/delta_y")
	}
	return nil
}

func buildScrollParams(request mcp.CallToolRequest) ScrollParams {
	args := request.GetArguments()
	return ScrollParams{
		Direction: request.GetString("direction", ""),
		Amount:    floatOrDefault(args, "amount", 0),
		Selector:  request.GetString("selector", ""),
		DeltaX:    floatOrDefault(args, "delta_x", 0),
		DeltaY:    floatOrDefault(args, "delta_y", 0),
	}
}

func ScrollTool(manager *session.Manager, scroll ScrollFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_scroll",
		mcp.WithDescription("Scroll the page by direction, into an element's view, or by pixel deltas"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("direction", mcp.Description("Scroll direction: up, down, left, right")),
		mcp.WithNumber("amount", mcp.Description("Pixels to scroll")),
		mcp.WithString("selector", mcp.Description("CSS selector to scroll into view")),
		mcp.WithNumber("delta_x", mcp.Description("Horizontal pixel delta")),
		mcp.WithNumber("delta_y", mcp.Description("Vertical pixel delta")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		args := request.GetArguments()
		if err := validateScrollStrategy(args); err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		params := buildScrollParams(request)
		scrollCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		result, err := scroll(scrollCtx, params)
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
