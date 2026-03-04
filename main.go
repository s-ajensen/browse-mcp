package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/s-ajensen/browse-mcp/action"
	"github.com/s-ajensen/browse-mcp/config"
	"github.com/s-ajensen/browse-mcp/session"
	"github.com/mark3labs/mcp-go/server"
)

func buildRegistry(tools []server.ServerTool) action.ToolRegistry {
	registry := make(action.ToolRegistry, len(tools))
	for _, serverTool := range tools {
		shortName := strings.TrimPrefix(serverTool.Tool.Name, "browse_")
		registry[shortName] = serverTool.Handler
	}
	return registry
}

func startReaper(manager *session.Manager, idleTimeout time.Duration) *time.Ticker {
	ticker := time.NewTicker(60 * time.Second)
	go func() {
		for range ticker.C {
			manager.ReapIdle(idleTimeout)
		}
	}()
	return ticker
}

func handleSignals(manager *session.Manager, reaper *time.Ticker) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		reaper.Stop()
		manager.ShutdownAll()
		os.Exit(0)
	}()
}

func main() {
	cfg := config.Load()
	sessionManager := session.NewManager(cfg.MaxSessions)

	timeout := cfg.ActionTimeout
	actionTools := []server.ServerTool{
		action.NavigateTool(sessionManager, action.ChromedpNavigate, timeout),
		action.BackTool(sessionManager, action.ChromedpBack, timeout),
		action.ForwardTool(sessionManager, action.ChromedpForward, timeout),
		action.ReloadTool(sessionManager, action.ChromedpReload, timeout),
		action.GetURLTool(sessionManager, action.ChromedpGetURL, timeout),
		action.GetTextTool(sessionManager, action.ChromedpGetText, timeout),
		action.GetHTMLTool(sessionManager, action.ChromedpGetHTML, timeout),
		action.ScreenshotTool(sessionManager, action.ChromedpScreenshot, timeout),
		action.ClickTool(sessionManager, action.ChromedpClick, timeout),
		action.TypeTool(sessionManager, action.ChromedpType, timeout),
		action.HoverTool(sessionManager, action.ChromedpHover, timeout),
		action.SelectTool(sessionManager, action.ChromedpSelect, timeout),
		action.KeyTool(sessionManager, action.ChromedpKey, timeout),
		action.EvalTool(sessionManager, action.ChromedpEval, timeout),
		action.ScrollTool(sessionManager, action.ChromedpScroll, timeout),
		action.WaitTool(sessionManager, action.ChromedpWait),
	}

	registry := buildRegistry(actionTools)

	sessionTools := []server.ServerTool{
		session.ListSessionsTool(sessionManager),
		session.DisconnectTool(sessionManager),
		session.SpawnTool(sessionManager, session.ChromedpSpawn),
		session.ConnectTool(sessionManager, session.ChromedpConnect),
	}

	reaper := startReaper(sessionManager, cfg.IdleTimeout)
	handleSignals(sessionManager, reaper)

	mcpServer := server.NewMCPServer("browse", "0.1.0")
	mcpServer.AddTools(sessionTools...)
	mcpServer.AddTools(actionTools...)
	mcpServer.AddTools(action.BatchTool(sessionManager, registry))

	err := server.ServeStdio(mcpServer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
