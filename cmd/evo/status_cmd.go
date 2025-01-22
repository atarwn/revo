package main

import (
	"evo/internal/index"
	"evo/internal/ops"
	"evo/internal/repo"
	"evo/internal/streams"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Detect changes and update CRDT logs in the current stream",
		Long: `Scans the working directory, updates .evo/index for new or renamed files,
and appends line-based CRDT ops for any changed files to the current stream.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			stream, err := streams.CurrentStream(rp)
			if err != nil {
				return err
			}
			// update stable file IDs first
			if err := index.UpdateIndex(rp); err != nil {
				return err
			}
			changed, err := ops.IngestLocalChanges(rp, stream)
			if err != nil {
				return err
			}
			if len(changed) == 0 {
				fmt.Println("No changes. Working directory is clean.")
				return nil
			}
			fmt.Println("Changes recorded in CRDT logs for files:")
			for _, c := range changed {
				fmt.Println("  ", c)
			}
			return nil
		},
	}
	rootCmd.AddCommand(statusCmd)
}
