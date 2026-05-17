package main

import (
	"context"
	"fmt"
	httpclient "oblak/internal/cli/client"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List uploaded functions",
	Long:  `List all functions uploaded to your Oblak instance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: change from health endpoint to list endpoint

		profile, err := globalCfg.ActiveProfile()
		if err != nil {
			return err
		}
		client, err := httpclient.NewSignedClient(profile)
		if err != nil {
			return err
		}

		ctx := context.Background()

		response, err := client.GetHealthWithResponse(ctx)
		if err != nil {
			return err
		}

		fmt.Println(string(response.Body))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
