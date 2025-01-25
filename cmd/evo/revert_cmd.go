package main

import (
	"evo/internal/commits"
	"evo/internal/repo"
	"evo/internal/streams"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var revertCmd = &cobra.Command{
		Use:   "revert <commit-id>",
		Short: "Revert the specified commit by generating inverse ops",
		Long:  `This properly restores old lines if the commit performed updates, removing inserted lines, etc.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: evo revert <commit-id>")
			}
			commitID := args[0]
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			str, err := streams.CurrentStream(rp)
			if err != nil {
				return err
			}
			newC, err := commits.RevertCommit(rp, str, commitID)
			if err != nil {
				return fmt.Errorf("failed to revert commit: %w", err)
			}
			fmt.Printf("Created revert commit %s\n", newC.ID)
			return nil
		},
	}
	rootCmd.AddCommand(revertCmd)
}
