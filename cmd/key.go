package cmd

import "github.com/spf13/cobra"

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Manage SSH keys with local OpenSSH tools",
	Long: `Manage SSH keys with local OpenSSH tools.

SSHM never stores private keys or passphrases. It calls tools like
ssh-keygen and ssh-add.`,
}

func init() {
	RootCmd.AddCommand(keyCmd)
}
