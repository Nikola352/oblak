package util

import (
	"context"
	"encoding/json"
	"fmt"
	httpclient "oblak/internal/cli/client"
	cliconfig "oblak/internal/cli/config"
	"os"
	"time"

	"github.com/google/uuid"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBlue   = "\033[34m"
	clearLine   = "\033[2K\r"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type GetFunctionStatusResponse struct {
	Status string `json:"status"`
}

func PollFunctionStatus(functionId string, profile cliconfig.Profile) {

	statusColor := func(status string) string {
		switch status {
		case "READY":
			return colorGreen
		case "FAILED":
			return colorRed
		case "PENDING":
			return colorYellow
		default:
			return colorCyan
		}
	}

	parsedId, err := uuid.Parse(functionId)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "%s✗ Invalid function ID %q: %v%s\n", colorRed, functionId, err, colorReset)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	client, err := httpclient.NewSignedClient(profile)
	if err != nil {
		_, err := fmt.Fprintf(os.Stderr, "%s✗ Failed to create client: %v%s\n", colorRed, err, colorReset)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	fmt.Printf("%s%s Polling function%s %s%s%s\n",
		colorBold, colorBlue, colorReset,
		colorDim, functionId, colorReset,
	)
	fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n\n", colorBold, colorBlue, colorReset)

	ctx := context.Background()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	start := time.Now()
	frame := 0
	var lastStatus string

	for {
		select {
		case <-ticker.C:
			response, err := client.GetFunctionStatusWithResponse(ctx, parsedId)
			if err != nil {
				fmt.Printf("\n%s✗ Request error: %v%s\n", colorRed, err, colorReset)
				os.Exit(1)
			}
			if response.StatusCode() >= 300 {
				fmt.Printf("\n%s✗ Unexpected response: HTTP %d%s\n", colorRed, response.StatusCode(), colorReset)
				os.Exit(1)
			}

			var result GetFunctionStatusResponse
			if err = json.Unmarshal(response.Body, &result); err != nil {
				fmt.Printf("\n%s✗ Failed to parse response: %v%s\n", colorRed, err, colorReset)
				os.Exit(1)
			}

			elapsed := time.Since(start).Round(time.Second)
			spin := spinnerFrames[frame%len(spinnerFrames)]
			frame++

			sc := statusColor(result.Status)
			fmt.Printf("%s%s%s %sStatus:%s %s%s%-12s%s  %s%s elapsed%s",
				clearLine,
				colorCyan, spin,
				colorBold, colorReset,
				colorBold, sc, result.Status, colorReset,
				colorDim, elapsed, colorReset,
			)

			if result.Status != lastStatus && lastStatus != "" {
				fmt.Printf("\n%s↳ Status changed: %s%s%s → %s%s%s\n",
					colorDim,
					statusColor(lastStatus), lastStatus, colorReset,
					sc, result.Status, colorReset,
				)
			}
			lastStatus = result.Status

			if result.Status == "FAILED" || result.Status == "READY" {
				fmt.Println() // end spinner line
				fmt.Printf("\n%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorBold, sc, colorReset)

				icon := "✓"
				if result.Status == "FAILED" {
					icon = "✗"
				}
				fmt.Printf("%s%s%s %sFinal status:%s %s%s%s%s  %s(took %s)%s\n",
					colorBold, sc, icon, colorReset,
					colorReset, colorBold, sc, result.Status, colorReset,
					colorDim, elapsed, colorReset,
				)
				fmt.Printf("%s%s━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorBold, sc, colorReset)
				return
			}
		}
	}
}
