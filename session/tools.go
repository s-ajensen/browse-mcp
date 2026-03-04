package session

import (
	"context"
	"time"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var provenanceLabels = map[Provenance]string{
	Spawned:           "spawned",
	ConnectedOwned:    "connected_owned",
	ConnectedAttached: "connected_attached",
}

type SpawnFunc func(ctx context.Context, headless bool, width, height int) (*Session, error)

type ConnectFunc func(ctx context.Context, debugURL string, tabURL string) (*Session, error)

func serializeSession(session *Session) map[string]string {
	return map[string]string{
		"session_id":  session.ID.String(),
		"type":        provenanceLabels[session.Provenance],
		"created_at":  session.CreatedAt.Format(time.RFC3339),
		"last_active": session.LastActive.Format(time.RFC3339),
		"current_url": session.CurrentURL,
	}
}

func ListSessionsTool(manager *Manager) server.ServerTool {
	tool := mcp.NewTool("browse_list_sessions",
		mcp.WithDescription("List all active browser sessions"),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessions := manager.List()
		serialized := make([]map[string]string, len(sessions))
		for index, session := range sessions {
			serialized[index] = serializeSession(session)
		}
		return mcputil.JSONToolResult(serialized)
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func DisconnectTool(manager *Manager) server.ServerTool {
	tool := mcp.NewTool("browse_disconnect",
		mcp.WithDescription("Disconnect and remove a browser session"),
		mcp.WithString("session_id", mcp.Required(), mcp.Description("The session ID to disconnect")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID, err := mcputil.ParseSessionID(request)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		session, err := manager.Get(sessionID)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		session.Disconnect()
		manager.Remove(sessionID)
		return mcputil.SuccessResult()
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func SpawnTool(manager *Manager, spawn SpawnFunc) server.ServerTool {
	tool := mcp.NewTool("browse_spawn",
		mcp.WithDescription("Spawn a new browser session"),
		mcp.WithNumber("viewport_width", mcp.Description("Viewport width in pixels"), mcp.DefaultNumber(1280)),
		mcp.WithNumber("viewport_height", mcp.Description("Viewport height in pixels"), mcp.DefaultNumber(800)),
		mcp.WithBoolean("headless", mcp.Description("Run browser in headless mode"), mcp.DefaultBool(true)),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		width := request.GetInt("viewport_width", 1280)
		height := request.GetInt("viewport_height", 800)
		headless := request.GetBool("headless", true)
		session, err := spawn(ctx, headless, width, height)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		err = manager.Add(session)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(map[string]any{
			"session_id": session.ID.String(),
			"viewport": map[string]int{
				"width":  width,
				"height": height,
			},
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}

func ConnectTool(manager *Manager, connect ConnectFunc) server.ServerTool {
	tool := mcp.NewTool("browse_connect",
		mcp.WithDescription("Connect to an existing browser via DevTools debug URL"),
		mcp.WithString("debug_url", mcp.Required(), mcp.Description("The Chrome DevTools debug URL")),
		mcp.WithString("tab_url", mcp.Description("URL of the tab to connect to")),
	)
	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		debugURL := request.GetString("debug_url", "")
		if debugURL == "" {
			return mcputil.ErrorResult("debug_url is required"), nil
		}
		tabURL := request.GetString("tab_url", "")
		session, err := connect(ctx, debugURL, tabURL)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		err = manager.Add(session)
		if err != nil {
			return mcputil.ErrorResult(err.Error()), nil
		}
		return mcputil.JSONToolResult(map[string]string{
			"session_id": session.ID.String(),
			"url":        session.CurrentURL,
			"title":      session.Title,
		})
	}
	return server.ServerTool{Tool: tool, Handler: handler}
}
