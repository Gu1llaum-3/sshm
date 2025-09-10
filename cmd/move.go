package cmd

import (
	"fmt"

	"github.com/Gu1llaum-3/sshm/internal/ui"

	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move <hostname>",
	Short: "Move an existing SSH host configuration to another config file",
	Long:  `Move an existing SSH host configuration to another config file with an interactive file selector.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostname := args[0]

		err := ui.RunMoveForm(hostname, configFile)
		if err != nil {
			fmt.Printf("Error moving host: %v\n", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(moveCmd)
}
