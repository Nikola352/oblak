package main

import (
	"context"
	"fmt"
	"io"
	httpclient "oblak/internal/cli/client"
	cliconfig "oblak/internal/cli/config"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var executeCmd = &cobra.Command{
	Use:   "execute [function id]",
	Short: "Execute a function in Oblak",
	Long:  `Execute a function with the given function id in your Oblak instance.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		functionId, err := validateId(args[0])
		if err != nil {
			return fmt.Errorf("invalid id %q, %q", args[0], err)
		}

		cfg, err := cliconfig.Load()
		if err != nil {
			return fmt.Errorf("loading existing config: %w", err)
		}

		profile, err := cfg.ActiveProfile()
		if err != nil {
			return fmt.Errorf("loading profile: %w", err)
		}
		fmt.Printf("Executing lambda with %s to %s as user %s\n", functionId, profile.Endpoint, profile.AuthID)

		err = sendExecutionRequest(functionId, profile)
		if err != nil {
			return err
		}

		return nil
	},
}

func validateId(id string) (string, error) {
	err := uuid.Validate(id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func sendExecutionRequest(functionId string, profile cliconfig.Profile) error {
	client, err := httpclient.NewSignedClient(profile)
	if err != nil {
		return err
	}

	ctx := context.Background()

	response, err := client.ExecuteLambda(ctx, functionId, "")
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("loading existing config: %v", err)
		}
	}(response.Body)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	fmt.Println(string(body))

	return nil
}

func init() {
	rootCmd.AddCommand(executeCmd)
}
