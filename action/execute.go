package action

import (
	"context"
	"time"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type EvalResult struct {
	Result any `json:"result"`
}

type EvalFunc func(browserCtx context.Context, expression string) (any, error)

func EvalTool(manager *session.Manager, eval EvalFunc, actionTimeout time.Duration) server.ServerTool {
	tool := mcp.NewTool("browse_eval",
		mcp.WithDescription("Execute JavaScript in the page. Power tool for clipboard operations, complex DOM queries, or anything other tools can't do. Expression is evaluated as the body of an async function."),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithString("expression", mcp.Required(), mcp.Description("The JavaScript expression to evaluate")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		found, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		expression := request.GetString("expression", "")
		if expression == "" {
			return mcputil.ErrorResult("expression is required"), nil
		}
		evalCtx, cancel := actionContext(found.BrowserCtx, actionTimeout)
		defer cancel()
		value, err := eval(evalCtx, expression)
		if err != nil {
			if timeoutResult := timeoutError(err, actionTimeout); timeoutResult != nil {
				return timeoutResult, nil
			}
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(EvalResult{Result: value})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
