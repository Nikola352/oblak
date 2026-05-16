package main

import (
	"os"

	cliconfig "oblak/internal/cli/config"

	"github.com/spf13/cobra"
)

var globalCfg cliconfig.Config

var rootCmd = &cobra.Command{
	Use:   "oblak",
	Short: "Oblak CLI: Manage your oblak functions",
	Long:  `Oblak is a CLI tool for managing and deploying serverless functions to your Oblak instance.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "configure" {
			return nil
		}
		cfg, err := cliconfig.Load()
		if err != nil {
			return err
		}
		globalCfg = cfg
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
