package action

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/s-ajensen/browse-mcp/mcputil"
	"github.com/mark3labs/mcp-go/mcp"
)

func actionContext(browserCtx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if browserCtx == nil {
		browserCtx = context.Background()
	}
	return context.WithTimeout(browserCtx, timeout)
}

func timeoutError(err error, timeout time.Duration) *mcp.CallToolResult {
	if errors.Is(err, context.DeadlineExceeded) {
		return mcputil.ErrorResult(fmt.Sprintf("action timed out after %s", timeout))
	}
	return nil
}
