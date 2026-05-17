package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	cliconfig "oblak/internal/cli/config"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure the Oblak CLI",
	Long:  `Configure the Oblak CLI with your server URL and authentication credentials.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		scanner := bufio.NewScanner(os.Stdin)

		authID, err := prompt(scanner, "Auth ID", "", true)
		if err != nil {
			return err
		}

		secretKey, err := prompt(scanner, "Secret Key", "", true)
		if err != nil {
			return err
		}

		endpoint, err := prompt(scanner, "Endpoint", "localhost:8080", false)
		if err != nil {
			return err
		}

		cfg, err := cliconfig.Load()
		if err != nil {
			return fmt.Errorf("loading existing config: %w", err)
		}

		profileName := cliconfig.ActiveProfileName()
		cfg[profileName] = cliconfig.Profile{
			AuthID:    authID,
			SecretKey: secretKey,
			Endpoint:  endpoint,
		}

		if err := cfg.Save(); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		_, err = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q saved to %s\n", profileName, cliconfig.Path())
		return err
	},
}

func prompt(scanner *bufio.Scanner, label, defaultVal string, required bool) (string, error) {
	for {
		var err error
		if defaultVal != "" {
			_, err = fmt.Fprintf(os.Stderr, "%s [%s]: ", label, defaultVal)
		} else {
			_, err = fmt.Fprintf(os.Stderr, "%s: ", label)
		}
		if err != nil {
			return "", err
		}

		if !scanner.Scan() {
			return "", fmt.Errorf("input closed unexpectedly")
		}

		val := strings.TrimSpace(scanner.Text())
		if val == "" && defaultVal != "" {
			return defaultVal, nil
		}
		if val == "" && required {
			if _, err = fmt.Fprintln(os.Stderr, "  (required, please enter a value)"); err != nil {
				return "", err
			}
			continue
		}
		return val, nil
	}
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
