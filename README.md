# browse

MCP server that gives agents browser control via Chrome DevTools Protocol.

## Start Chrome with Remote Debugging

```bash
# macOS
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222

# Linux
google-chrome --remote-debugging-port=9222
```

Then connect from the MCP server using `browse_connect` with `debug_url` set to `http://localhost:9222`.

## Build & Run

```bash
go build -o browse .
./browse
```
