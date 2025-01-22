package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "evo",
	Short: "Evo (🌿) - next-generation CRDT-based version control",
	Long: `Evo is a production-ready version control system that uses named streams,
line-based CRDT (with RGA for reordering), stable file IDs, commit signing, and large file support.`,
}

// Execute runs the CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
