package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	httpclient "oblak/internal/cli/client"
	cliconfig "oblak/internal/cli/config"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var uploadCmd = &cobra.Command{
	Use:   "upload [path]",
	Short: "Upload a function to Oblak",
	Long:  `Upload a function from the given path to your Oblak instance.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := validatePath(args[0])
		if err != nil {
			return fmt.Errorf("invalid path %q, %q", args[0], err)
		}
		cfg, err := cliconfig.Load()
		if err != nil {
			return fmt.Errorf("loading existing config: %w", err)
		}

		profile, err := cfg.ActiveProfile()
		if err != nil {
			return fmt.Errorf("loading profile: %w", err)
		}
		fmt.Printf("Uploading to %s as user %s\n", profile.Endpoint, profile.AuthID)

		err = uploadFile(path, profile)
		if err != nil {
			return err
		}

		return nil
	},
}

func validatePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error resolving path: %s\n", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	if !strings.HasSuffix(absPath, ".tar.gz") && !strings.HasSuffix(absPath, ".tgz") {
		return "", fmt.Errorf("file must be .tar.gz, .tgz, or format")
	}
	return absPath, nil
}

func uploadFile(path string, profile cliconfig.Profile) error {
	body, contentType, err := buildBody(path)
	if err != nil {
		return fmt.Errorf("creating request body: %w", err)
	}

	client, err := httpclient.NewSignedClient(profile)
	if err != nil {
		return err
	}

	ctx := context.Background()

	response, err := client.UploadLambdaWithBodyWithResponse(ctx, contentType, body)
	if err != nil {
		return err
	}

	fmt.Println(string(response.Body))

	return nil
}

func buildBody(path string) (*bytes.Buffer, string, error) {
	// Open the tar.gz file
	file, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("opening file: %w", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file to the form
	part, err := writer.CreateFormFile("function", filepath.Base(path))
	if err != nil {
		return nil, "", fmt.Errorf("creating form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return nil, "", fmt.Errorf("copying file content: %w", err)
	}

	// Close the writer to set the terminating boundary
	err = writer.Close()
	if err != nil {
		return nil, "", fmt.Errorf("closing writer: %w", err)
	}

	return body, writer.FormDataContentType(), nil
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}
