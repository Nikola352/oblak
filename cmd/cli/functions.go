package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	httpclient "oblak/internal/cli/client"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// Declare a variable to store the flag value
var functionIDFlag string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List uploaded functions or function invocations",
	Long:  `List all functions uploaded to your Oblak instance, or look up invocation history for a specific function via ID.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := globalCfg.ActiveProfile()
		if err != nil {
			return err
		}
		client, err := httpclient.NewSignedClient(profile)
		if err != nil {
			return err
		}

		ctx := context.Background()
		var rawBody []byte

		// Check if the user passed the function flag
		if functionIDFlag != "" {
			// Call the new invocations history endpoint
			parsedId, err := uuid.Parse(functionIDFlag)
			if err != nil {
				return err
			}

			response, err := client.GetFunctionInvocationsWithResponse(ctx, parsedId)
			if err != nil {
				return err
			}

			if response.JSON200 == nil {
				return fmt.Errorf("request failed with status %d: %s", response.StatusCode(), string(response.Body))
			}
			rawBody = response.Body
		} else {
			// Fallback to the default listing endpoint
			response, err := client.GetUserFunctionsWithResponse(ctx)
			if err != nil {
				return err
			}

			if response.JSON200 == nil {
				return fmt.Errorf("request failed with status %d: %s", response.StatusCode(), string(response.Body))
			}
			rawBody = response.Body
		}

		// Indent and print the JSON payload
		var prettyJSON bytes.Buffer
		err = json.Indent(&prettyJSON, rawBody, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}

		fmt.Println(prettyJSON.String())
		return nil
	},
}

func init() {
	// Register the local string flag `--function_id` (or `-f` shorthand)
	listCmd.Flags().StringVarP(
		&functionIDFlag,
		"function-id",
		"f",
		"",
		"Filter history logs by passing a specific function UUID",
	)

	rootCmd.AddCommand(listCmd)
}
