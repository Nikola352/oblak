package main

import (
	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [path]",
	Short: "Upload a function to Oblak",
	Long:  `Upload a function from the given path to your Oblak instance.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}
