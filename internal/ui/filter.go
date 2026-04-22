package ui

import "github.com/Gu1llaum-3/sshm/internal/config"

// applySourceFileFilter returns only the hosts whose SourceFile equals
// selected. An empty selected string is treated as "no filter" and returns
// the slice unchanged.
func applySourceFileFilter(hosts []config.SSHHost, selected string) []config.SSHHost {
	if selected == "" {
		return hosts
	}
	out := make([]config.SSHHost, 0, len(hosts))
	for _, h := range hosts {
		if h.SourceFile == selected {
			out = append(out, h)
		}
	}
	return out
}
