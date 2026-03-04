package action

import (
	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/mcp"
)

func lookupSession(manager *session.Manager, request mcp.CallToolRequest) (*session.Session, *mcp.CallToolResult) {
	sessionID, err := mcputil.ParseSessionID(request)
	if err != nil {
		return nil, mcputil.ErrorResult(err.Error())
	}
	found, err := manager.Get(sessionID)
	if err != nil {
		return nil, mcputil.ErrorResult(err.Error())
	}
	return found, nil
}
