package main

import (
	"evo/internal/commits"
	"evo/internal/config"
	"evo/internal/repo"
	"evo/internal/signing"
	"evo/internal/streams"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	var logCmd = &cobra.Command{
		Use:   "log",
		Short: "Show commit history for the current stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				return err
			}
			stream, err := streams.CurrentStream(rp)
			if err != nil {
				return err
			}
			verifyStr, _ := config.GetConfigValue(rp, "verifySignatures")
			doVerify := (verifyStr == "true")

			cc, err := commits.ListCommits(rp, stream)
			if err != nil {
				return err
			}
			if len(cc) == 0 {
				fmt.Println("No commits found in this stream.")
				return nil
			}
			for _, c := range cc {
				ver := ""
				if c.Signature != "" && doVerify {
					valid, err := signing.VerifyCommit(&c, rp)
					if err != nil {
						ver = " (error: " + err.Error() + ")"
					} else if valid {
						ver = " (verified)"
					} else {
						ver = " (INVALID!)"
					}
				}
				fmt.Printf("commit %s%s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n\n",
					c.ID, ver, c.AuthorName, c.AuthorEmail, c.Timestamp.Local(), c.Message)
			}
			return nil
		},
	}
	rootCmd.AddCommand(logCmd)
}
