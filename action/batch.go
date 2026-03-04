package action

import (
	"context"
	"encoding/json"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolRegistry map[string]server.ToolHandlerFunc

type BatchActionResult struct {
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
}

type BatchResult struct {
	Results   []BatchActionResult `json:"results"`
	Completed int                 `json:"completed"`
	Total     int                 `json:"total"`
}

func parseActions(request mcp.CallToolRequest) ([]any, bool) {
	args := request.GetArguments()
	raw, exists := args["actions"]
	if !exists {
		return nil, false
	}
	actions, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	return actions, true
}

func parseStopOnError(request mcp.CallToolRequest) bool {
	args := request.GetArguments()
	raw, exists := args["stop_on_error"]
	if !exists {
		return true
	}
	val, ok := raw.(bool)
	if !ok {
		return true
	}
	return val
}

func extractTextFromResult(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return ""
	}
	return textContent.Text
}

func parseResultBody(text string) any {
	var parsed any
	if json.Unmarshal([]byte(text), &parsed) != nil {
		return text
	}
	return parsed
}

func buildSubRequest(actionParams map[string]any, sessionID string) mcp.CallToolRequest {
	merged := make(map[string]any, len(actionParams)+1)
	for key, val := range actionParams {
		merged[key] = val
	}
	merged["session_id"] = sessionID
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: merged,
		},
	}
}

func executeAction(ctx context.Context, registry ToolRegistry, toolName string, actionParams map[string]any, sessionID string) BatchActionResult {
	handler, found := registry[toolName]
	if !found {
		return BatchActionResult{Tool: toolName, Success: false, Error: "unknown tool: " + toolName}
	}
	subRequest := buildSubRequest(actionParams, sessionID)
	result, err := handler(ctx, subRequest)
	if err != nil {
		return BatchActionResult{Tool: toolName, Success: false, Error: err.Error()}
	}
	text := extractTextFromResult(result)
	if result.IsError {
		return BatchActionResult{Tool: toolName, Success: false, Error: text}
	}
	return BatchActionResult{Tool: toolName, Success: true, Result: parseResultBody(text)}
}

type actionEntry struct {
	tool   string
	params map[string]any
}

func parseActionEntry(raw any) (actionEntry, bool) {
	entry, ok := raw.(map[string]any)
	if !ok {
		return actionEntry{}, false
	}
	toolName, _ := entry["tool"].(string)
	params, _ := entry["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
	}
	return actionEntry{tool: toolName, params: params}, true
}

func executeActions(ctx context.Context, registry ToolRegistry, actions []any, sessionID string, stopOnError bool) BatchResult {
	results := make([]BatchActionResult, 0, len(actions))
	completed := 0
	for _, raw := range actions {
		parsed, ok := parseActionEntry(raw)
		if !ok {
			continue
		}
		actionResult := executeAction(ctx, registry, parsed.tool, parsed.params, sessionID)
		results = append(results, actionResult)
		if actionResult.Success {
			completed++
		}
		if !actionResult.Success && stopOnError {
			break
		}
	}
	return BatchResult{
		Results:   results,
		Completed: completed,
		Total:     len(actions),
	}
}

func BatchTool(manager *session.Manager, registry ToolRegistry) server.ServerTool {
	tool := mcp.NewTool("browse_batch",
		mcp.WithDescription("Execute multiple actions sequentially in one round-trip"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID")),
		mcp.WithArray("actions", mcp.Required(), mcp.Description("Array of actions to execute")),
		mcp.WithBoolean("stop_on_error", mcp.Description("Stop executing on first error (default: true)")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		_, toolErr := lookupSession(manager, request)
		if toolErr != nil {
			return toolErr, nil
		}
		actions, actionsExist := parseActions(request)
		if !actionsExist {
			return mcputil.ErrorResult("actions is required"), nil
		}
		stopOnError := parseStopOnError(request)
		sessionID := request.GetString("session_id", "")
		batch := executeActions(ctx, registry, actions, sessionID, stopOnError)
		return mcputil.JSONToolResult(batch)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
