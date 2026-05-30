package main

import (
	"bytes"
	"context"
	"encoding/json"
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

		response, err := client.GetUserFunctionsWithResponse(ctx)
		if err != nil {
			return err
		}

		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, response.Body, "", "    ") // 4 spaces indentation
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}

		// Print the indented string
		fmt.Println(prettyJSON.String())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
