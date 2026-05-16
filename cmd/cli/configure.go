package main

import (
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure the Oblak CLI",
	Long:  `Configure the Oblak CLI with your server URL and authentication credentials.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
