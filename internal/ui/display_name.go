package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

// sourceFileDisplayName returns a human-friendly label for a config file
// path, matching the convention used by the file selector:
//   - the main SSH config becomes "Main SSH Config (~/.ssh/config)"
//   - files under ~/.ssh become "~/.ssh/<relpath>"
//   - anything else is returned as-is
func sourceFileDisplayName(path string) string {
	if path == "" {
		return ""
	}
	if mainConfig, err := config.GetDefaultSSHConfigPath(); err == nil && path == mainConfig {
		return "Main SSH Config (~/.ssh/config)"
	}
	if sshDir, err := config.GetSSHDirectory(); err == nil && strings.HasPrefix(path, sshDir) {
		if rel, err := filepath.Rel(sshDir, path); err == nil {
			return fmt.Sprintf("~/.ssh/%s", rel)
		}
	}
	return path
}
