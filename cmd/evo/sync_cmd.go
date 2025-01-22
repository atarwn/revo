package main

import (
	"evo/internal/repo"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var syncCmd = &cobra.Command{
		Use:   "sync <remote-url>",
		Short: "Synchronize CRDT logs with remote (not fully implemented)",
		Long: `Pull missing ops from remote for the current stream and push local ops
to the remote. Requires a future Evo server implementation for full functionality.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: evo sync <remote-url>")
			}
			remote := args[0]
			_, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			fmt.Printf("Sync with %s is not yet implemented.\n", remote)
			return nil
		},
	}
	rootCmd.AddCommand(syncCmd)
}
