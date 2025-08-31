package cmd

import (
	"fmt"
	"sshm/internal/ui"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <hostname>",
	Short: "Edit an existing SSH host configuration",
	Long:  `Edit an existing SSH host configuration with an interactive form.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hostname := args[0]

		err := ui.RunEditForm(hostname)
		if err != nil {
			fmt.Printf("Error editing host: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
