package main

import (
	"evo/internal/repo"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var initCmd = &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new Evo repository",
		Long: `Creates a .evo directory with default stream "main", config folder, index for stable file IDs,
and other structures needed for CRDT-based version control.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			if err := repo.InitRepo(path); err != nil {
				return err
			}
			fmt.Println("Initialized Evo repository at", path)
			return nil
		},
	}
	rootCmd.AddCommand(initCmd)
}
