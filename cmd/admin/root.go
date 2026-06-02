package main

import (
	"context"
	"os"

	"oblak/internal/server/authkey"
	"oblak/internal/server/config"
	"oblak/internal/server/database"
	"oblak/internal/server/user"

	"github.com/spf13/cobra"
)

var (
	userStore    *user.Store
	authKeyStore *authkey.Store
)

var rootCmd = &cobra.Command{
	Use:   "oblak-admin",
	Short: "Oblak admin utility for managing users and auth keys",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		db, err := database.Connect(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return err
		}
		userStore = user.NewStore(db)
		authKeyStore = authkey.NewStore(db, cfg.KeyEncryptionKey)
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
