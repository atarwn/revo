package main

import (
	"evo/internal/repo"
	"evo/internal/streams"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var streamCmd = &cobra.Command{
		Use:   "stream",
		Short: "Manage named streams (like branches)",
		Long:  "Create, switch, list, merge, or cherry-pick commits in named streams.",
	}

	var createCmd = &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: evo stream create <name>")
			}
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			if err := streams.CreateStream(rp, args[0]); err != nil {
				return err
			}
			fmt.Println("Created stream:", args[0])
			return nil
		},
	}

	var switchCmd = &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch to another stream locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: evo stream switch <name>")
			}
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			if err := streams.SwitchStream(rp, args[0]); err != nil {
				return err
			}
			fmt.Println("Switched to stream:", args[0])
			return nil
		},
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List named streams",
		RunE: func(cmd *cobra.Command, args []string) error {
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			ss, err := streams.ListStreams(rp)
			if err != nil {
				return err
			}
			cur, _ := streams.CurrentStream(rp)
			for _, s := range ss {
				prefix := "  "
				if s == cur {
					prefix = "* "
				}
				fmt.Println(prefix + s)
			}
			return nil
		},
	}

	var mergeCmd = &cobra.Command{
		Use:   "merge <source> <target>",
		Short: "Merge all commits from source stream into target stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: evo stream merge <source> <target>")
			}
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			if err := streams.MergeStreams(rp, args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Merged all missing commits from '%s' into '%s'\n", args[0], args[1])
			return nil
		},
	}

	var cherryPickCmd = &cobra.Command{
		Use:   "cherry-pick <commit-id> <target-stream>",
		Short: "Replicate only one commit's ops into the target stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: evo stream cherry-pick <commit-id> <target-stream>")
			}
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			if err := streams.CherryPick(rp, args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Cherry-picked commit %s into stream %s\n", args[0], args[1])
			return nil
		},
	}

	streamCmd.AddCommand(createCmd, switchCmd, listCmd, mergeCmd, cherryPickCmd)
	rootCmd.AddCommand(streamCmd)
}
