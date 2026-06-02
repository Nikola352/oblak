package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"oblak/cmd/cli/util"
	httpclient "oblak/internal/cli/client"
	cliconfig "oblak/internal/cli/config"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

type UploadLambdaResponse struct {
	FunctionID string `json:"functionId"`
	Status     string `json:"status"`
}

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

		functionId, err := uploadFile(path, profile)
		if err != nil {
			return err
		}
		util.PollFunctionStatus(functionId, profile)

		return nil
	},
}

func validatePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error resolving path: %s\n", err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %w", err)
	}

	if !fileInfo.IsDir() {
		return "", fmt.Errorf("file path is not a directory")
	}
	return absPath, nil
}

func uploadFile(path string, profile cliconfig.Profile) (string, error) {
	body, err := zipFolder(path)
	if err != nil {
		return "", fmt.Errorf("creating request body: %w", err)
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("function", "function.tar.gz")
	if err != nil {
		return "", err
	}

	_, err = part.Write(body.Bytes())
	if err != nil {
		return "", err
	}

	err = writer.Close()
	if err != nil {
		return "", err
	}
	client, err := httpclient.NewSignedClient(profile)
	if err != nil {
		return "", err
	}

	ctx := context.Background()

	response, err := client.UploadLambdaWithBodyWithResponse(ctx, writer.FormDataContentType(), &buf)
	if err != nil {
		return "", err
	}
	if response.StatusCode() >= 300 {
		return "", fmt.Errorf("upload failed: %s", string(response.Body))
	}

	var result UploadLambdaResponse

	err = json.Unmarshal(response.Body, &result)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Function with functionId: %s uploaded with status: %s \n", result.FunctionID, result.Status)

	return result.FunctionID, nil
}

func zipFolder(source string) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(buffer)

	defer func(gzipWriter *gzip.Writer) {
		err := gzipWriter.Close()
		if err != nil {
			fmt.Printf("closing gzip writer: %v", err)
		}
	}(gzipWriter)

	tarWriter := tar.NewWriter(gzipWriter)
	defer func(tarWriter *tar.Writer) {
		err := tarWriter.Close()
		if err != nil {
			fmt.Printf("closing tar writer: %v", err)
		}
	}(tarWriter)

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// skip root
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = relPath

		// write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// directories have no content
		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})

	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func init() {
	rootCmd.AddCommand(uploadCmd)
}
