package util

import "github.com/ringclaw/ringclaw/config"

// Truncate shortens s to n characters, appending "..." if truncated.
// In debug mode, returns the full string without truncation.
func Truncate(s string, n int) string {
	if config.IsDebug() || len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
