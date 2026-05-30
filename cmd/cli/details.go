package main

import (
	"context"
	"fmt"
	httpclient "oblak/internal/cli/client"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const (
	colorRed   = "\033[31m"
	colorWhite = "\033[97m"
	colorReset = "\033[0m"
	colorCyan  = "\033[36m"
	colorBold  = "\033[1m"
)

var invocationIdFlag string

var detailsCmd = &cobra.Command{
	Use:   "details",
	Short: "Return a detailed execution of function run",
	Long:  `Present a execution of a function uploaded to your Oblak instance thru run ID.`,
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

		parsedId, err := uuid.Parse(invocationIdFlag)
		if err != nil {
			return err
		}

		response, err := client.GetInvocationDetailsWithResponse(ctx, parsedId)
		if err != nil {
			return err
		}
		if response.JSON200 == nil {
			return fmt.Errorf("request failed with status %d: %s", response.StatusCode(), string(response.Body))
		}

		inv := response.JSON200

		// Print basic details
		fmt.Printf("%s%sInvocation ID:%s  %s\n", colorBold, colorCyan, colorReset, inv.InvocationId)
		fmt.Printf("%s%sStatus:       %s  %s\n", colorBold, colorCyan, colorReset, inv.Status)
		fmt.Printf("%s%sStarted:      %s  %s\n", colorBold, colorCyan, colorReset, inv.InvocationTime.Format("2006-01-02 15:04:05"))
		if inv.EndTime != nil {
			fmt.Printf("%s%sEnded:        %s  %s\n", colorBold, colorCyan, colorReset, inv.EndTime.Format("2006-01-02 15:04:05"))
			duration := inv.EndTime.Sub(inv.InvocationTime)
			fmt.Printf("%s%sDuration:     %s  %s\n", colorBold, colorCyan, colorReset, duration)
		}

		// Separator
		fmt.Printf("\n%s%s--- Logs ---%s\n\n", colorBold, colorCyan, colorReset)

		// Print logs
		for _, log := range inv.Logs {
			color := colorWhite
			if log.Stream == "stderr" {
				color = colorRed
			}
			fmt.Printf("%s%s%s\n", color, log.Message, colorReset)
		}

		return nil
	},
}

func init() {
	detailsCmd.Flags().StringVarP(
		&invocationIdFlag,
		"invocation-id",
		"i",
		"",
		"Filter history logs by passing a specific function UUID",
	)
	rootCmd.AddCommand(detailsCmd)
}
