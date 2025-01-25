package main

import (
	"evo/internal/commits"
	"evo/internal/config"
	"evo/internal/index"
	"evo/internal/repo"
	"evo/internal/streams"
	"evo/internal/types"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	commitMsg  string
	commitSign bool
)

func init() {
	var commitCmd = &cobra.Command{
		Use:   "commit",
		Short: "Group new CRDT ops into a commit, optionally signed",
		Long: `Collect newly added CRDT ops (including old content for updates) into a single commit
with a message and optional Ed25519 signature, if configured.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if commitMsg == "" {
				return fmt.Errorf("use -m to specify a commit message")
			}
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			stream, err := streams.CurrentStream(rp)
			if err != nil {
				return err
			}
			// update index
			if err := index.UpdateIndex(rp); err != nil {
				return err
			}
			name, _ := config.GetConfigValue(rp, "user.name")
			email, _ := config.GetConfigValue(rp, "user.email")
			if name == "" {
				name = "EvoUser"
			}
			if email == "" {
				email = "user@evo"
			}
			cid, err := commits.CreateCommit(rp, stream, commitMsg, name, email, []types.ExtendedOp{}, commitSign)
			if err != nil {
				return err
			}
			fmt.Printf("Created commit %s in stream %s\n", cid.ID, stream)
			return nil
		},
	}
	commitCmd.Flags().StringVarP(&commitMsg, "message", "m", "", "Commit message")
	commitCmd.Flags().BoolVar(&commitSign, "sign", false, "Sign commit using Ed25519 if configured")
	rootCmd.AddCommand(commitCmd)
}
