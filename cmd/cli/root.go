package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "oblak",
	Short: "Oblak CLI: Manage your oblak functions",
	Long:  `Oblak is a CLI tool for managing and deploying serverless functions to your Oblak instance.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
}
