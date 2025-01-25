package main

import (
	"evo/internal/repo"
	"evo/internal/status"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show the working tree status",
		Long: `Shows the status of files in the working directory:
- New (untracked) files
- Modified files
- Deleted files
- Renamed files
Respects .evo-ignore patterns for excluding files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}

			st, err := status.GetStatus(rp)
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}

			fmt.Print(status.FormatStatus(st))
			return nil
		},
	}
	rootCmd.AddCommand(statusCmd)
}
