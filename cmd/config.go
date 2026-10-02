package cmd

import (
	"fmt"
	"io"

	"github.com/Gu1llaum-3/sshm/internal/config"
)

// parseConfigForCLI emits each parser diagnostic once, independently of stdout.
// TUI and completion callers use the silent config APIs instead.
func parseConfigForCLI(path string, warningOut io.Writer) ([]config.SSHHost, error) {
	var hosts []config.SSHHost
	var warnings []string
	var err error
	if path != "" {
		hosts, warnings, err = config.ParseSSHConfigFileWithWarnings(path)
	} else {
		hosts, warnings, err = config.ParseSSHConfigWithWarnings()
	}
	for _, warning := range warnings {
		fmt.Fprintln(warningOut, "warning:", warning)
	}
	return hosts, err
}
