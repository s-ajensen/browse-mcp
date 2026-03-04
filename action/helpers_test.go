package action

import (
	"testing"

	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
)

func setupManagerWithSession(t *testing.T) (*session.Manager, *session.Session) {
	t.Helper()
	manager := session.NewManager(5)
	sess := session.NewSession(session.Spawned)
	manager.Add(sess)
	return manager, sess
}

func toolRequest(sessionID string, args map[string]any) mcp.CallToolRequest {
	args["session_id"] = sessionID
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}
