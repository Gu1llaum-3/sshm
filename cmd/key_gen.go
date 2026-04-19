package cmd

import (
	"fmt"
	"strings"

	keycmd "github.com/Gu1llaum-3/sshm/internal/key"

	"github.com/spf13/cobra"
)

var (
	keyGenName      string
	keyGenAlgorithm string
	keyGenComment   string
	keyGenDirectory string
	keyGenKDFRounds int
	keyGenDryRun    bool
)

var keyGenCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate an SSH key pair with ssh-keygen",
	Long: `Generate an SSH key pair with ssh-keygen.

SSHM uses the system ssh-keygen binary and never handles passphrases itself.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := keycmd.GenerateOptions{
			Name:      keyGenName,
			Algorithm: keyGenAlgorithm,
			Comment:   keyGenComment,
			Directory: keyGenDirectory,
			KDFRounds: keyGenKDFRounds,
			DryRun:    keyGenDryRun,
		}

		result, genArgs, err := keycmd.BuildGenerateArgs(opts)
		if err != nil {
			return err
		}

		if keyGenDryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "Would run: ssh-keygen %s\n", shellJoin(genArgs))
			fmt.Fprintf(cmd.OutOrStdout(), "Private: %s\n", result.PrivateKeyPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Public: %s\n", result.PublicKeyPath)
			return nil
		}

		result, err = keycmd.Generate(cmd.Context(), keycmd.ExecRunner{}, opts)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Generated:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Private: %s\n", result.PrivateKeyPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  Public: %s\n", result.PublicKeyPath)
		return nil
	},
}

func shellJoin(args []string) string {
	var quoted []string
	for _, arg := range args {
		if strings.ContainsAny(arg, " \t\n\"'") {
			quoted = append(quoted, fmt.Sprintf("%q", arg))
			continue
		}
		quoted = append(quoted, arg)
	}
	return strings.Join(quoted, " ")
}

func init() {
	keyCmd.AddCommand(keyGenCmd)

	keyGenCmd.Flags().StringVar(&keyGenName, "name", "", "Key file name without .pub")
	keyGenCmd.Flags().StringVar(&keyGenAlgorithm, "algo", "ed25519", fmt.Sprintf("Key algorithm passed to ssh-keygen (%s)", strings.Join(keycmd.AllowedGenerateAlgorithms(), ", ")))
	keyGenCmd.Flags().StringVar(&keyGenComment, "comment", "", "Key comment")
	keyGenCmd.Flags().StringVar(&keyGenDirectory, "path", "", "Directory for the new key pair (default: ~/.ssh)")
	keyGenCmd.Flags().IntVar(&keyGenKDFRounds, "kdf-rounds", 100, "KDF rounds passed to ssh-keygen -a")
	keyGenCmd.Flags().BoolVar(&keyGenDryRun, "dry-run", false, "Show the ssh-keygen command without running it")
	_ = keyGenCmd.MarkFlagRequired("name")
}
