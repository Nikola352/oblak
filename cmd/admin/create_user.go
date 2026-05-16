package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var createUserCmd = &cobra.Command{
	Use:   "create-user",
	Short: "Create a new user",
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")

		u, err := userStore.CreateUser(context.Background(), username, email)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "user_id:  %s\nusername: %s\nemail:    %s\n", u.UserId, u.Username, u.Email)
		return nil
	},
}

func init() {
	createUserCmd.Flags().String("username", "", "Username (required)")
	createUserCmd.Flags().String("email", "", "Email (required)")
	createUserCmd.MarkFlagRequired("username")
	createUserCmd.MarkFlagRequired("email")
	rootCmd.AddCommand(createUserCmd)
}
