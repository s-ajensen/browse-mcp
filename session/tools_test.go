package session

import (
	"context"
	"errors"
	"testing"

	"github.com/s-ajensen/browse-mcp/testutil"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestListSessionsTool_ReturnsToolNamedBrowseListSessions(t *testing.T) {
	manager := NewManager(5)

	serverTool := ListSessionsTool(manager)

	assert.Equal(t, "browse_list_sessions", serverTool.Tool.Name)
}

func TestListSessionsTool_Handler_ReturnsEmptyArrayWhenNoSessions(t *testing.T) {
	manager := NewManager(5)
	serverTool := ListSessionsTool(manager)
	request := mcp.CallToolRequest{}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var sessions []any
	testutil.UnmarshalToolResult(t, result, &sessions)
	assert.Equal(t, 0, len(sessions))
}

func TestListSessionsTool_Handler_ReturnsSessionData(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(Spawned)
	session.CurrentURL = "https://example.com"
	manager.Add(session)
	serverTool := ListSessionsTool(manager)
	request := mcp.CallToolRequest{}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var sessions []map[string]any
	testutil.UnmarshalToolResult(t, result, &sessions)
	assert.Equal(t, 1, len(sessions))
	assert.Equal(t, session.ID.String(), sessions[0]["session_id"])
	assert.Equal(t, "spawned", sessions[0]["type"])
	assert.NotEmpty(t, sessions[0]["created_at"])
	assert.NotEmpty(t, sessions[0]["last_active"])
	assert.Equal(t, "https://example.com", sessions[0]["current_url"])
}

func TestListSessionsTool_Handler_MapsProvenanceToCorrectTypeStrings(t *testing.T) {
	manager := NewManager(5)
	spawned := NewSession(Spawned)
	owned := NewSession(ConnectedOwned)
	attached := NewSession(ConnectedAttached)
	manager.Add(spawned)
	manager.Add(owned)
	manager.Add(attached)
	serverTool := ListSessionsTool(manager)
	request := mcp.CallToolRequest{}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var sessions []map[string]any
	testutil.UnmarshalToolResult(t, result, &sessions)
	assert.Equal(t, 3, len(sessions))
	typesByID := make(map[string]string)
	for _, entry := range sessions {
		typesByID[entry["session_id"].(string)] = entry["type"].(string)
	}
	assert.Equal(t, "spawned", typesByID[spawned.ID.String()])
	assert.Equal(t, "connected_owned", typesByID[owned.ID.String()])
	assert.Equal(t, "connected_attached", typesByID[attached.ID.String()])
}

func TestDisconnectTool_ReturnsToolNamedBrowseDisconnect(t *testing.T) {
	manager := NewManager(5)

	serverTool := DisconnectTool(manager)

	assert.Equal(t, "browse_disconnect", serverTool.Tool.Name)
}

func TestDisconnectTool_Handler_DisconnectsAndRemovesSession(t *testing.T) {
	manager := NewManager(5)
	sess := NewSession(Spawned)
	browserCancelCalled := false
	sess.browserCancel = func() { browserCancelCalled = true }
	manager.Add(sess)
	serverTool := DisconnectTool(manager)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": sess.ID.String(),
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, browserCancelCalled)
	_, getErr := manager.Get(sess.ID)
	assert.Error(t, getErr)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, true, parsed["success"])
}

func TestDisconnectTool_Handler_ReturnsErrorForUnknownSessionID(t *testing.T) {
	manager := NewManager(5)
	serverTool := DisconnectTool(manager)
	unknownID := uuid.New()
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": unknownID.String(),
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestDisconnectTool_Handler_ReturnsErrorForInvalidSessionIDFormat(t *testing.T) {
	manager := NewManager(5)
	serverTool := DisconnectTool(manager)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"session_id": "not-a-uuid",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

type spawnRecorder struct {
	headless bool
	width    int
	height   int
	session  *Session
}

func (recorder *spawnRecorder) spawn(ctx context.Context, headless bool, width, height int) (*Session, error) {
	recorder.headless = headless
	recorder.width = width
	recorder.height = height
	return recorder.session, nil
}

func fakeSpawnFunc(session *Session) SpawnFunc {
	return func(ctx context.Context, headless bool, width, height int) (*Session, error) {
		return session, nil
	}
}

func failingSpawnFunc(message string) SpawnFunc {
	return func(ctx context.Context, headless bool, width, height int) (*Session, error) {
		return nil, errors.New(message)
	}
}

func TestSpawnTool_ReturnsToolNamedBrowseSpawn(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(Spawned)

	serverTool := SpawnTool(manager, fakeSpawnFunc(session))

	assert.Equal(t, "browse_spawn", serverTool.Tool.Name)
}

func TestSpawnTool_Handler_UsesDefaultViewportWhenNoParams(t *testing.T) {
	manager := NewManager(5)
	recorder := &spawnRecorder{session: NewSession(Spawned)}
	serverTool := SpawnTool(manager, recorder.spawn)
	request := mcp.CallToolRequest{}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 1280, recorder.width)
	assert.Equal(t, 800, recorder.height)
}

func TestSpawnTool_Handler_UsesCustomViewportFromParams(t *testing.T) {
	manager := NewManager(5)
	recorder := &spawnRecorder{session: NewSession(Spawned)}
	serverTool := SpawnTool(manager, recorder.spawn)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"viewport_width":  1920,
				"viewport_height": 1080,
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, 1920, recorder.width)
	assert.Equal(t, 1080, recorder.height)
}

func TestSpawnTool_Handler_UsesDefaultHeadlessTrue(t *testing.T) {
	manager := NewManager(5)
	recorder := &spawnRecorder{session: NewSession(Spawned)}
	serverTool := SpawnTool(manager, recorder.spawn)
	request := mcp.CallToolRequest{}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, recorder.headless)
}

func TestSpawnTool_Handler_AddsSessionToManager(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(Spawned)
	serverTool := SpawnTool(manager, fakeSpawnFunc(session))
	request := mcp.CallToolRequest{}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	found, getErr := manager.Get(session.ID)
	assert.NoError(t, getErr)
	assert.Equal(t, session.ID, found.ID)
}

func TestSpawnTool_Handler_ReturnsSessionIDAndViewport(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(Spawned)
	serverTool := SpawnTool(manager, fakeSpawnFunc(session))
	request := mcp.CallToolRequest{}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, session.ID.String(), parsed["session_id"])
	viewport := parsed["viewport"].(map[string]any)
	assert.Equal(t, float64(1280), viewport["width"])
	assert.Equal(t, float64(800), viewport["height"])
}

func TestSpawnTool_Handler_ReturnsErrorWhenSpawnFails(t *testing.T) {
	manager := NewManager(5)
	serverTool := SpawnTool(manager, failingSpawnFunc("chrome not found"))
	request := mcp.CallToolRequest{}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestSpawnTool_Handler_ReturnsErrorWhenMaxSessionsReached(t *testing.T) {
	manager := NewManager(0)
	session := NewSession(Spawned)
	serverTool := SpawnTool(manager, fakeSpawnFunc(session))
	request := mcp.CallToolRequest{}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

type connectRecorder struct {
	debugURL string
	tabURL   string
	session  *Session
}

func (recorder *connectRecorder) connect(ctx context.Context, debugURL, tabURL string) (*Session, error) {
	recorder.debugURL = debugURL
	recorder.tabURL = tabURL
	return recorder.session, nil
}

func fakeConnectFunc(session *Session) ConnectFunc {
	return func(ctx context.Context, debugURL, tabURL string) (*Session, error) {
		return session, nil
	}
}

func failingConnectFunc(message string) ConnectFunc {
	return func(ctx context.Context, debugURL, tabURL string) (*Session, error) {
		return nil, errors.New(message)
	}
}

func TestConnectTool_ReturnsToolNamedBrowseConnect(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(ConnectedOwned)

	serverTool := ConnectTool(manager, fakeConnectFunc(session))

	assert.Equal(t, "browse_connect", serverTool.Tool.Name)
}

func TestConnectTool_Handler_PassesDebugURLAndTabURLToConnectFunc(t *testing.T) {
	manager := NewManager(5)
	recorder := &connectRecorder{session: NewSession(ConnectedAttached)}
	serverTool := ConnectTool(manager, recorder.connect)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"debug_url": "http://localhost:9222",
				"tab_url":   "github.com",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:9222", recorder.debugURL)
	assert.Equal(t, "github.com", recorder.tabURL)
}

func TestConnectTool_Handler_PassesEmptyTabURLWhenNotProvided(t *testing.T) {
	manager := NewManager(5)
	recorder := &connectRecorder{session: NewSession(ConnectedOwned)}
	serverTool := ConnectTool(manager, recorder.connect)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"debug_url": "http://localhost:9222",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, "", recorder.tabURL)
}

func TestConnectTool_Handler_ReturnsErrorWhenDebugURLMissing(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(ConnectedOwned)
	serverTool := ConnectTool(manager, fakeConnectFunc(session))
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestConnectTool_Handler_AddsSessionToManager(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(ConnectedOwned)
	serverTool := ConnectTool(manager, fakeConnectFunc(session))
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"debug_url": "http://localhost:9222",
			},
		},
	}

	_, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	found, getErr := manager.Get(session.ID)
	assert.NoError(t, getErr)
	assert.Equal(t, session.ID, found.ID)
}

func TestConnectTool_Handler_ReturnsSessionIDURLAndTitle(t *testing.T) {
	manager := NewManager(5)
	session := NewSession(ConnectedOwned)
	session.CurrentURL = "https://example.com"
	session.Title = "Example Domain"
	serverTool := ConnectTool(manager, fakeConnectFunc(session))
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"debug_url": "http://localhost:9222",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	var parsed map[string]any
	testutil.UnmarshalToolResult(t, result, &parsed)
	assert.Equal(t, session.ID.String(), parsed["session_id"])
	assert.Equal(t, "https://example.com", parsed["url"])
	assert.Equal(t, "Example Domain", parsed["title"])
}

func TestConnectTool_Handler_ReturnsErrorWhenConnectFails(t *testing.T) {
	manager := NewManager(5)
	serverTool := ConnectTool(manager, failingConnectFunc("connection refused"))
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"debug_url": "http://localhost:9222",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestConnectTool_Handler_ReturnsErrorWhenMaxSessionsReached(t *testing.T) {
	manager := NewManager(0)
	session := NewSession(ConnectedOwned)
	serverTool := ConnectTool(manager, fakeConnectFunc(session))
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"debug_url": "http://localhost:9222",
			},
		},
	}

	result, err := serverTool.Handler(context.Background(), request)

	assert.NoError(t, err)
	assert.True(t, result.IsError)
}
