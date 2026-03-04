package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type devtoolsVersion struct {
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

type devtoolsTarget struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  string `json:"type"`
}

func fetchJSON(url string, dest any) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func normalizeDebugURL(debugURL string) string {
	debugURL = strings.TrimRight(debugURL, "/")
	if !strings.HasPrefix(debugURL, "http://") && !strings.HasPrefix(debugURL, "https://") {
		return "http://" + debugURL
	}
	return debugURL
}

func fetchDevtoolsVersion(debugURL string) (devtoolsVersion, error) {
	var version devtoolsVersion
	err := fetchJSON(normalizeDebugURL(debugURL)+"/json/version", &version)
	return version, err
}

func fetchDevtoolsTargets(debugURL string) ([]devtoolsTarget, error) {
	var targets []devtoolsTarget
	err := fetchJSON(normalizeDebugURL(debugURL)+"/json/list", &targets)
	return targets, err
}

func findTargetByURL(targets []devtoolsTarget, urlSubstring string) (devtoolsTarget, error) {
	for _, entry := range targets {
		if strings.Contains(entry.URL, urlSubstring) {
			return entry, nil
		}
	}
	var available strings.Builder
	for _, entry := range targets {
		fmt.Fprintf(&available, "  - %s (%s)\n", entry.URL, entry.Title)
	}
	return devtoolsTarget{}, fmt.Errorf("no tab found matching %q. Available tabs:\n%s", urlSubstring, available.String())
}

func fetchPageInfo(browserCtx context.Context) (string, string, error) {
	var url, title string
	err := chromedp.Run(browserCtx,
		chromedp.Location(&url),
		chromedp.Title(&title),
	)
	return url, title, err
}

func ChromedpSpawn(ctx context.Context, headless bool, width, height int) (*Session, error) {
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.WindowSize(width, height),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	err := chromedp.Run(browserCtx)
	if err != nil {
		browserCancel()
		allocCancel()
		return nil, fmt.Errorf("failed to start browser: %w", err)
	}
	spawned := NewSession(Spawned)
	spawned.BrowserCtx = browserCtx
	spawned.allocCancel = allocCancel
	spawned.browserCancel = browserCancel
	return spawned, nil
}

func connectOwned(allocCtx context.Context) (*Session, context.CancelFunc, error) {
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	err := chromedp.Run(browserCtx)
	if err != nil {
		browserCancel()
		return nil, nil, fmt.Errorf("failed to connect to browser: %w", err)
	}
	url, title, err := fetchPageInfo(browserCtx)
	if err != nil {
		browserCancel()
		return nil, nil, fmt.Errorf("failed to fetch page info: %w", err)
	}
	owned := NewSession(ConnectedOwned)
	owned.CurrentURL = url
	owned.Title = title
	owned.BrowserCtx = browserCtx
	return owned, browserCancel, nil
}

func connectAttached(allocCtx context.Context, debugURL string, tabURL string) (*Session, context.CancelFunc, error) {
	targets, err := fetchDevtoolsTargets(debugURL)
	if err != nil {
		return nil, nil, err
	}
	matched, err := findTargetByURL(targets, tabURL)
	if err != nil {
		return nil, nil, err
	}
	browserCtx, browserCancel := chromedp.NewContext(allocCtx, chromedp.WithTargetID(target.ID(matched.ID)))
	err = chromedp.Run(browserCtx)
	if err != nil {
		browserCancel()
		return nil, nil, fmt.Errorf("failed to attach to tab: %w", err)
	}
	url, title, err := fetchPageInfo(browserCtx)
	if err != nil {
		browserCancel()
		return nil, nil, fmt.Errorf("failed to fetch page info: %w", err)
	}
	attached := NewSession(ConnectedAttached)
	attached.CurrentURL = url
	attached.Title = title
	attached.BrowserCtx = browserCtx
	return attached, browserCancel, nil
}

func connectToTab(allocCtx context.Context, debugURL string, tabURL string) (*Session, context.CancelFunc, error) {
	if tabURL == "" {
		return connectOwned(allocCtx)
	}
	return connectAttached(allocCtx, debugURL, tabURL)
}

func ChromedpConnect(ctx context.Context, debugURL string, tabURL string) (*Session, error) {
	version, err := fetchDevtoolsVersion(debugURL)
	if err != nil {
		return nil, err
	}
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, version.WebSocketDebuggerURL)
	connected, browserCancel, connectErr := connectToTab(allocCtx, debugURL, tabURL)
	if connectErr != nil {
		allocCancel()
		return nil, connectErr
	}
	connected.browserCancel = browserCancel
	return connected, nil
}
