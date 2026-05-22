package orchestrator

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type UnzipperService struct {
	MaxAllowedBytes int64
}

func NewUnzipper(maxSizeBytes int64) *UnzipperService {
	if maxSizeBytes <= 0 {
		maxSizeBytes = 50 * 1024 * 1024 // Fallback default: 50MB
	}
	return &UnzipperService{
		MaxAllowedBytes: maxSizeBytes,
	}
}
func (te *UnzipperService) Extract(sourceTarGz, targetDestinationDir string) error {
	fileReader, err := os.Open(sourceTarGz)
	if err != nil {
		return fmt.Errorf("failed opening archive file: %w", err)
	}
	defer fileReader.Close()

	gzipReader, err := gzip.NewReader(fileReader)
	if err != nil {
		return fmt.Errorf("invalid gzip compression layer: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	cleanBaseTarget := filepath.Clean(targetDestinationDir)
	if err := os.MkdirAll(cleanBaseTarget, 0755); err != nil {
		return fmt.Errorf("failed creating root staging folder: %w", err)
	}

	var totalWrittenBytes int64

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("malformed tar stream metadata: %w", err)
		}

		// -----------------------------------------------------------------
		// DYNAMIC STRIPPER: Removes the top-level wrapping folder if it exists
		// -----------------------------------------------------------------
		cleanedHeaderName := filepath.Clean(header.Name)
		parts := strings.Split(cleanedHeaderName, string(os.PathSeparator))

		// If parts length is 1 and it's a directory, this is the root wrapper folder entry itself.
		// Skip it so we don't create it on the host.
		if len(parts) == 1 && header.Typeflag == tar.TypeDir {
			continue
		}

		// If it's deeper, slice off the first element (the parent folder name)
		var targetRelPath string
		if len(parts) > 1 {
			targetRelPath = filepath.Join(parts[1:]...)
		} else {
			targetRelPath = cleanedHeaderName
		}

		// Construct final destination using the stripped relative path
		destinationPath := filepath.Join(cleanBaseTarget, targetRelPath)
		// -----------------------------------------------------------------

		// --- PATH TRAVERSAL MITIGATION ---
		if !strings.HasPrefix(destinationPath, cleanBaseTarget+string(os.PathSeparator)) && destinationPath != cleanBaseTarget {
			return fmt.Errorf("security rule exception: directory traversal path blocked on '%s'", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destinationPath, 0755); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
				return err
			}

			dstFile, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed creating host target file placeholder '%s': %w", header.Name, err)
			}

			// --- DECOMPRESSION BOMB MITIGATION ---
			remainingCap := te.MaxAllowedBytes - totalWrittenBytes
			limitedStream := io.LimitReader(tarReader, remainingCap)

			written, copyErr := io.Copy(dstFile, limitedStream)
			dstFile.Close()

			if copyErr != nil {
				return fmt.Errorf("write error processing file '%s': %w", header.Name, copyErr)
			}

			totalWrittenBytes += written
			if totalWrittenBytes >= te.MaxAllowedBytes {
				return fmt.Errorf("resource exhaustion boundary triggered: uncompressed footprint exceeds safe limits")
			}
		}
	}

	return nil
}
