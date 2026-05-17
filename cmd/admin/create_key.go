package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var createKeyCmd = &cobra.Command{
	Use:   "create-key",
	Short: "Create an auth key for a user",
	RunE: func(cmd *cobra.Command, args []string) error {
		rawID, _ := cmd.Flags().GetString("user-id")
		userID, err := uuid.Parse(rawID)
		if err != nil {
			return fmt.Errorf("invalid user-id: %w", err)
		}

		key, err := authKeyStore.CreateAuthKey(context.Background(), userID)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "auth_id:    %s\nsecret_key: %s\n", key.AuthId, key.SecretKey)
		return nil
	},
}

func init() {
	createKeyCmd.Flags().String("user-id", "", "User UUID (required)")
	createKeyCmd.MarkFlagRequired("user-id")
	rootCmd.AddCommand(createKeyCmd)
}
