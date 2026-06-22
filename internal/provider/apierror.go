package provider

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// apiError builds an error from a non-success HTTP response, surfacing the response
// body for diagnostics. When the status equals conflictStatus and the body contains
// conflictMsg, it wraps ErrRepoExists so callers can errors.Is it.
func apiError(prefix string, resp *http.Response, conflictStatus int, conflictMsg string) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10)) //nolint:gosec // best-effort read of the error body for diagnostics
	msg := strings.TrimSpace(string(body))
	if resp.StatusCode == conflictStatus && conflictMsg != "" && strings.Contains(msg, conflictMsg) {
		return fmt.Errorf("%s: %w", prefix, ErrRepoExists)
	}
	if msg == "" {
		return fmt.Errorf("%s: status %d", prefix, resp.StatusCode)
	}
	return fmt.Errorf("%s: status %d: %s", prefix, resp.StatusCode, msg)
}
