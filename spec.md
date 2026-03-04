Checkpoint: browse MCP Server Spec
What This Is
An MCP server (stdio transport) written in Go that gives agents the ability to browse the web. Uses Chrome DevTools Protocol via chromedp. Two modes: spawn a headless browser, or connect to an existing Chrome with remote debugging enabled — including attaching to tabs the user already has open.
Core Architecture Decisions
| Decision | Choice | Why |
|---|---|---|
| Language | Go | User preference. CDP via chromedp handles both modes. |
| MCP SDK | github.com/mark3labs/mcp-go | Most mature Go MCP SDK. |
| CDP library | github.com/chromedp/chromedp | De facto Go CDP library. |
| Transport | stdio | OpenCode spawns MCP servers as child processes. |
| Session model | Session = single tab | Simpler mental model. New tab = new session. |
| Persistence | None | Server restart kills all sessions. Ephemeral. |
| Screenshots | Base64 in MCP response | MCP supports image content type natively. |
| Element targeting | CSS, XPath, text content, coordinates | All four. Tool descriptions advise when to use each. |
| Default viewport | 1280×800 | Reasonable for screenshots without being enormous. |
| Idle timeout | 30 minutes | Background goroutine reaps idle sessions. |
Session Lifecycle & Multi-Client Safety
The problem: One OpenCode server process → one MCP server process → N agent clients. Agents must not share browser state, and abandoned sessions must not leak.
The solution:
- Every session gets a UUID.
- Every tool call requires a session_id parameter (except browse_spawn, browse_connect, browse_list_sessions).
- Sessions are stored in a mutex-protected map in the session manager.
- A background goroutine runs every 60 seconds, reaping sessions whose last_active exceeds 30 minutes.
- On server shutdown (SIGINT/SIGTERM), all sessions are closed and all spawned Chrome processes are killed.
Session provenance — three origins, three disconnect behaviors:
| Origin | How created | Disconnect behavior |
|---|---|---|
| spawned | browse_spawn | Kill the browser process |
| connected_owned | browse_connect without tab_url | Close our tab, leave browser running |
| connected_attached | browse_connect with tab_url | Detach only — leave the tab open and untouched |
The session struct tracks this provenance as an enum. Disconnect dispatches on it.
Spawned sessions: chromedp.NewExecAllocator → we own the process.
Connected sessions: Hit http://{debug_url}/json/version to discover the WebSocket endpoint, then chromedp.NewRemoteAllocator. If tab_url is provided, hit http://{debug_url}/json/list, find the target whose URL contains the substring, and attach to it. If no match, return an error that lists all available tabs (URLs + titles) so the agent can self-correct.
Project Structure
browse/
  main.go              # MCP server setup, tool registration, signal handling
  session/
    manager.go         # Session pool, cleanup goroutine, lookup by ID
    session.go         # Session struct, spawn/connect/disconnect
  action/
    navigate.go        # navigate, back, forward, reload
    interact.go        # click, type, scroll, hover, select, key
    observe.go         # screenshot, get_text, get_html, get_url
    execute.go         # eval, wait, batch
  selector/
    resolve.go         # Unified selector resolution → chromedp actions
Tool Catalog
Every tool returns structured JSON (except browse_screenshot which returns MCP image content). Every tool that operates on a session takes session_id as its first required parameter and updates last_active on the session.
---
Session Lifecycle
browse_spawn — Start a new headless Chrome and return a session.
- Params: viewport_width (opt, 1280), viewport_height (opt, 800), headless (opt, true)
- Returns: { session_id, viewport: { width, height } }
browse_connect — Connect to an existing Chrome DevTools instance.
- Params: debug_url (required, e.g. http://localhost:9222), tab_url (opt — URL substring to match an existing tab)
- Behavior:
  - tab_url provided → find tab whose URL contains the substring, attach to it. Session provenance: connected_attached.
  - tab_url omitted → open a new tab. Session provenance: connected_owned.
  - No tab matches tab_url → error listing all available tabs (URL + title) so the agent can self-correct.
- Returns: { session_id, url, title }
browse_disconnect — Close a session and free resources.
- Params: session_id
- Returns: { success: true }
- Behavior: Dispatches on session provenance (see table above).
browse_list_sessions — List all active sessions.
- Params: none
- Returns: [{ session_id, type, created_at, last_active, current_url }]
---
Navigation
browse_navigate — Go to a URL.
- Params: session_id, url, wait_until (opt: "load" | "domcontentloaded", default "load")
- Returns: { url, title }
browse_back / browse_forward / browse_reload — History navigation.
- Params: session_id
- Returns: { url, title }
---
Observation
browse_screenshot — Capture the visible page or a specific element.
- Params: session_id, selector (opt — screenshot specific element), full_page (opt, false)
- Returns: MCP image content (PNG, base64)
browse_get_text — Extract text content.
- Params: session_id, selector (opt, defaults to body)
- Returns: { text }
- Tool description hint: "Use this to read page content. Returns visible text, stripped of HTML. For structured data, use browse_get_html or browse_eval."
browse_get_html — Get HTML of page or element.
- Params: session_id, selector (opt), outer (opt, true)
- Returns: { html }
browse_get_url — Get current page URL and title.
- Params: session_id
- Returns: { url, title }
---
Interaction
browse_click — Click an element.
- Params: session_id, then one of: selector (CSS), xpath, text (visible text match), or x+y (coordinates). Plus: button (opt: "left"/"right"/"middle", default "left"), click_count (opt, 1; use 2 for double-click)
- Returns: { success: true }
- Tool description hint: "Use CSS selectors when you know the DOM structure. Use text matching when you can see a button label in a screenshot. Use coordinates as a last resort."
browse_type — Type text into a field.
- Params: session_id, selector (opt — omit to type into focused element), text, clear (opt, false — clear field first), submit (opt, false — press Enter after)
- Returns: { success: true }
browse_scroll — Scroll the page.
- Params: session_id, then one of: direction ("up"/"down"/"left"/"right") with amount (opt, pixels, defaults to one viewport height), selector (scroll element into view), or delta_x+delta_y (pixel deltas)
- Returns: { scroll_x, scroll_y } (current scroll position after)
browse_hover — Hover over an element.
- Params: session_id, then one of: selector, xpath, text, or x+y
- Returns: { success: true }
browse_select — Choose from a <select> dropdown.
- Params: session_id, selector (required — the select element), then one of: value or label
- Returns: { selected_value, selected_label }
browse_key — Press a key or key combination.
- Params: session_id, key (e.g. "Enter", "Escape", "Tab", "ArrowDown", "Control+a")
- Returns: { success: true }
- Note: This is how "paste" works — agent uses browse_eval to write to clipboard, then browse_key with Control+v.
---
Advanced
browse_eval — Execute JavaScript in the page.
- Params: session_id, expression (JS code)
- Returns: { result } (JSON-serialized return value)
- Tool description hint: "Power tool. Use for clipboard operations, complex DOM queries, or anything the other tools can't do. Expression is evaluated as the body of an async function."
browse_wait — Wait for a condition before proceeding.
- Params: session_id, then one of: selector (wait for element), xpath, text (wait for text to appear), url_contains, or function (JS expression that returns truthy). Plus: timeout_ms (opt, default 30000), state (opt for selector: "visible"/"hidden"/"attached"/"detached", default "visible")
- Returns: { elapsed_ms }
browse_batch — Execute multiple actions sequentially in one round-trip.
- Params: session_id, actions (array of { tool, params } where tool is any tool name minus the browse_ prefix and params excludes session_id), stop_on_error (opt, true)
- Returns: { results: [{ tool, success, result?, error? }], completed, total }
- Rationale: MCP round-trips are expensive. An agent should be able to say "navigate, wait, screenshot" in one call.
---
Selector Resolution (selector/resolve.go)
All interaction tools accept multiple targeting strategies. The resolve package provides a unified function that maps params to chromedp actions.
Priority when multiple are present: error. Exactly one strategy must be provided.
| Strategy | Parameter(s) | chromedp mapping |
|---|---|---|
| Coordinates | x + y | chromedp.MouseClickXY / dispatched input event |
| XPath | xpath | chromedp.BySearch |
| Text | text | Build XPath: //*[contains(normalize-space(text()), "...")] |
| CSS | selector | chromedp.ByQuery |
Error Handling
- Invalid session_id → "session not found: {id}. Use browse_spawn or browse_connect to create a session."
- Element not found → include the selector/strategy used in the error message
- Navigation timeout → include URL and timeout value
- No tab matches tab_url → list all available tabs (URL + title) in the error
- Chrome crashed → mark session as dead, clean up, tell the agent to spawn a new one
Configuration (Environment Variables)
| Var | Default | Description |
|---|---|---|
| BROWSE_CHROME_PATH | auto-detect | Path to Chrome/Chromium binary |
| BROWSE_IDLE_TIMEOUT | 30m | Session idle timeout |
| BROWSE_MAX_SESSIONS | 10 | Max concurrent sessions |
| BROWSE_DEFAULT_VIEWPORT | 1280x800 | Default viewport for spawned sessions |
Open Questions (for implementer to decide)
1. networkidle in chromedp: Intentionally dropped from the spec. chromedp doesn't have native network idle detection. wait_until supports "load" and "domcontentloaded" only. Agents can use browse_wait with a JS function for finer-grained readiness checks.
2. Screenshot format: PNG for v1. JPEG option can come later.
3. browse_get_text truncation: A full page's text could be enormous. Consider a max_length param or automatic truncation with a note indicating content was cut
