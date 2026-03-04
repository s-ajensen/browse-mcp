package mcputil

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
)

func JSONToolResult(value any) (*mcp.CallToolResult, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize response: %w", err)
	}
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func SuccessResult() (*mcp.CallToolResult, error) {
	return JSONToolResult(map[string]bool{"success": true})
}

func ErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: message}},
		IsError: true,
	}
}

func ParseSessionID(request mcp.CallToolRequest) (uuid.UUID, error) {
	raw := request.GetString("session_id", "")
	if raw == "" {
		return uuid.UUID{}, fmt.Errorf("session_id is required")
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid session_id format: %w", err)
	}
	return parsed, nil
}
