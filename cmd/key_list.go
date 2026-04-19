package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	keycmd "github.com/Gu1llaum-3/sshm/internal/key"

	"github.com/spf13/cobra"
)

var keyListJSON bool

var keyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List local SSH keys and config references",
	Long: `List local SSH keys and explicit ssh_config references.

Only explicit IdentityFile entries are reported. SSHM does not resolve
effective identities via ssh -G.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := keycmd.Inventory(cmd.Context(), keycmd.ExecRunner{}, configFile)
		if err != nil {
			return err
		}

		if keyListJSON {
			data, err := json.MarshalIndent(items, "", "  ")
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return err
		}

		outputKeyInventoryTable(cmd, items)
		return nil
	},
}

func outputKeyInventoryTable(cmd *cobra.Command, items []keycmd.InventoryItem) {
	if len(items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No local SSH keys found.")
		return
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PATH\tMODE\tTYPE\tFINGERPRINT\tCONFIG HOSTS")
	for _, item := range items {
		hosts := "-"
		if len(item.References) > 0 {
			hostNames := make([]string, len(item.References))
			for i := range item.References {
				hostNames[i] = item.References[i].Host
			}
			hosts = strings.Join(hostNames, ", ")
		}

		keyType := item.Algorithm
		if keyType == "" {
			keyType = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", item.Path, item.Permissions, keyType, item.Fingerprint, hosts)
	}
	_ = w.Flush()

	label := "keys"
	if len(items) == 1 {
		label = "key"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\nFound %d %s\n", len(items), label)
}

func init() {
	keyCmd.AddCommand(keyListCmd)
	keyListCmd.Flags().BoolVar(&keyListJSON, "json", false, "Print key inventory as JSON")
}
