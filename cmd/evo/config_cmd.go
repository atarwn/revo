package main

import (
	"evo/internal/config"
	"evo/internal/repo"
	"fmt"

	"github.com/spf13/cobra"
)

var cfgGlobal bool

func init() {
	var setCmd = &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key (repo-level by default, or --global)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: evo config set <key> <value>")
			}
			key, val := args[0], args[1]

			if cfgGlobal {
				return config.SetGlobalConfigValue(key, val)
			}
			rp, err := repo.FindRepoRoot(".")
			if err != nil {
				// fallback to global
				return config.SetGlobalConfigValue(key, val)
			}
			return config.SetRepoConfigValue(rp, key, val)
		},
	}
	setCmd.Flags().BoolVar(&cfgGlobal, "global", false, "Set global config instead of repo-level")

	var getCmd = &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value (repo-level overrides global)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: evo config get <key>")
			}
			key := args[0]
			rp, err := repo.FindRepoRoot(".")
			var val string
			if err != nil {
				// fallback global
				val, err = config.GetConfigValue("", key)
			} else {
				val, err = config.GetConfigValue(rp, key)
			}
			if err != nil {
				fmt.Println("Error:", err)
				return nil
			}
			if val == "" {
				fmt.Printf("No value found for key: %s\n", key)
			} else {
				fmt.Println(val)
			}
			return nil
		},
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage Evo configuration",
	}

	configCmd.AddCommand(setCmd, getCmd)
	rootCmd.AddCommand(configCmd)
}
