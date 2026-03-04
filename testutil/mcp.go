package testutil

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func UnmarshalToolResult(t *testing.T, result *mcp.CallToolResult, target any) {
	t.Helper()
	textContent := result.Content[0].(mcp.TextContent)
	err := json.Unmarshal([]byte(textContent.Text), target)
	assert.NoError(t, err)
}
