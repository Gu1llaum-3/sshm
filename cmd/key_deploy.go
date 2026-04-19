package cmd

import (
	"fmt"

	keycmd "github.com/Gu1llaum-3/sshm/internal/key"

	"github.com/spf13/cobra"
)

var (
	keyDeployIdentity string
	keyDeployUser     string
	keyDeployPort     string
	keyDeployDryRun   bool
)

var keyDeployCmd = &cobra.Command{
	Use:   "deploy <host-or-address>",
	Short: "Copy a public key to a remote host",
	Long: `Copy a public key to a remote host with OpenSSH tools.

SSHM uses ssh-copy-id when available and falls back to an idempotent ssh flow
when it is not.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := keycmd.DeployOptions{
			Target:     args[0],
			User:       keyDeployUser,
			Port:       keyDeployPort,
			Identity:   keyDeployIdentity,
			ConfigPath: configFile,
			DryRun:     keyDeployDryRun,
		}

		plan, err := keycmd.Deploy(cmd.Context(), keycmd.ExecRunner{}, opts)
		if err != nil {
			return err
		}

		if keyDeployDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "Would run: %s %s\n", plan.Command, shellJoin(plan.Args))
			fmt.Fprintf(cmd.OutOrStdout(), "Public key: %s\n", plan.PublicKeyPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Target: %s\n", plan.Target)
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deployed %s to %s\n", plan.PublicKeyPath, plan.Target)
		return nil
	},
}

func init() {
	keyCmd.AddCommand(keyDeployCmd)

	keyDeployCmd.Flags().StringVar(&keyDeployIdentity, "identity", "", "Private key path, or matching .pub path")
	keyDeployCmd.Flags().StringVar(&keyDeployUser, "user", "", "Remote SSH user (treat target as a direct host)")
	keyDeployCmd.Flags().StringVar(&keyDeployPort, "port", "", "Remote SSH port (treat target as a direct host)")
	keyDeployCmd.Flags().BoolVar(&keyDeployDryRun, "dry-run", false, "Show the deploy command without running it")
	_ = keyDeployCmd.MarkFlagRequired("identity")
}
