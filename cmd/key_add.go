package cmd

import (
	"fmt"

	keycmd "github.com/Gu1llaum-3/sshm/internal/key"

	"github.com/spf13/cobra"
)

var (
	keyAddAppleKeychain bool
	keyAddDryRun        bool
)

var keyAddCmd = &cobra.Command{
	Use:   "add <private-key>",
	Short: "Add a private key to the local ssh-agent",
	Long: `Add a private key to the local ssh-agent with ssh-add.

SSHM leaves passphrase prompts and agent handling to the system ssh-add binary.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := keycmd.AddOptions{
			Path:          args[0],
			AppleKeychain: keyAddAppleKeychain,
			DryRun:        keyAddDryRun,
		}

		path, addArgs, err := keycmd.Add(cmd.Context(), keycmd.ExecRunner{}, opts)
		if err != nil {
			return err
		}

		if keyAddDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "Would run: ssh-add %s\n", shellJoin(addArgs))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Added to ssh-agent: %s\n", path)
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyAddCmd)

	keyAddCmd.Flags().BoolVar(&keyAddAppleKeychain, "apple-keychain", false, "Use Apple's ssh-add keychain support when available")
	keyAddCmd.Flags().BoolVar(&keyAddDryRun, "dry-run", false, "Show the ssh-add command without running it")
}
