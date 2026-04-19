package cmd

import (
	"fmt"

	keycmd "github.com/Gu1llaum-3/sshm/internal/key"

	"github.com/spf13/cobra"
)

var (
	keyAttachIdentity string
	keyAttachDryRun   bool
)

var keyAttachCmd = &cobra.Command{
	Use:   "attach <host>",
	Short: "Attach a key to an existing SSH host",
	Long: `Attach a key to an existing SSH host by setting IdentityFile.

SSHM resolves the target host strictly. If the name is ambiguous or the block
cannot be edited safely, attach stops.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := keycmd.Attach(cmd.Context(), keycmd.AttachOptions{
			Host:       args[0],
			Identity:   keyAttachIdentity,
			ConfigPath: configFile,
			DryRun:     keyAttachDryRun,
		})
		if err != nil {
			return err
		}

		if result.AlreadyAttached {
			fmt.Fprintf(cmd.OutOrStdout(), "%s already uses %s\n", result.HostName, result.Identity)
			return nil
		}

		if keyAttachDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "Would attach %s to %s at %s:%d\n", result.Identity, result.HostName, result.SourceFile, result.Line)
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Attached %s to %s at %s:%d\n", result.Identity, result.HostName, result.SourceFile, result.Line)
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyAttachCmd)

	keyAttachCmd.Flags().StringVar(&keyAttachIdentity, "identity", "", "Private key path, or matching .pub path")
	keyAttachCmd.Flags().BoolVar(&keyAttachDryRun, "dry-run", false, "Show the resolved target without writing config")
	_ = keyAttachCmd.MarkFlagRequired("identity")
}
