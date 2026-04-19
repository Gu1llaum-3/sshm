package ui

import (
	"os/exec"

	keypkg "github.com/Gu1llaum-3/sshm/internal/key"

	tea "github.com/charmbracelet/bubbletea"
)

func runUIDeployCommand(opts keypkg.DeployOptions, onDone func(error) tea.Msg) tea.Cmd {
	plan, err := keypkg.BuildDeployPlan(opts)
	if err != nil {
		return func() tea.Msg { return onDone(err) }
	}

	cmd := exec.Command(plan.Command, plan.Args...)
	return tea.ExecProcess(cmd, onDone)
}
