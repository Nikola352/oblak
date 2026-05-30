package extractor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"oblak/internal/server/events"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
)

type Extractor struct {
	filestore *minio.Client
}

const MaxTotalSize = 1024 * 1024
const MaxFiles = 10

func NewExtractor(filestore *minio.Client) *Extractor {
	return &Extractor{
		filestore: filestore,
	}
}

func (e *Extractor) ExtractFromQuarantine(event events.QuarantineEvent) error {
	ctx := context.Background()

	obj, err := e.filestore.GetObject(ctx, event.Bucket, event.Path, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object from quarantine: %w", err)
	}
	defer func(obj *minio.Object) {
		_ = obj.Close()
	}(obj)

	// unzip and save
	err = e.Unzip(obj)
	if err != nil {
		return fmt.Errorf("failed to unzip object from quarantine: %w", err)
	}
	// try to extract the zip if no error save extracted to the bucket

	return nil
}

func (e *Extractor) Unzip(obj *minio.Object) error {
	gzReader, err := gzip.NewReader(obj)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func(gzReader *gzip.Reader) {
		_ = gzReader.Close()
	}(gzReader)

	tarReader := tar.NewReader(gzReader)

	//  iterate through tar entries
	var totalExtractedSize int64 = 0
	var totalExtractedFiles int64 = 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		cleanName, err := validateTarHeader(header, totalExtractedSize, totalExtractedFiles)
		if err != nil {
			return fmt.Errorf("invalid tar entry: %w", err)
		}

		totalExtractedSize += header.Size
		totalExtractedFiles++

		log.Printf("Safely extracted: %s file.\n", cleanName)
	}
	return nil
}

func validateTarHeader(header *tar.Header, totalExtractedSize int64, totalExtractedFiles int64) (cleanName string, err error) {
	if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
		return "", fmt.Errorf("non-regular file type %c: %s", header.Typeflag, header.Name)
	}

	cleanName = filepath.Clean(header.Name)

	if cleanName == "." || cleanName == "/" || cleanName == "" {
		return "", fmt.Errorf("invalid empty filename")
	}
	if strings.Contains(cleanName, "..") || filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("path traversal detected: %s", cleanName)
	}
	if strings.ContainsRune(cleanName, 0) {
		return "", fmt.Errorf("null byte in filename: %s", cleanName)
	}

	if header.Size > MaxTotalSize {
		return "", fmt.Errorf("file too large (%d > %d)", header.Size, MaxTotalSize)
	}
	newTotal := totalExtractedSize + header.Size
	if newTotal > MaxTotalSize {
		return "", fmt.Errorf("total extracted size would exceed %d bytes", MaxTotalSize)
	}
	if totalExtractedFiles+1 > MaxFiles {
		return "", fmt.Errorf("too many files (max %d)", MaxFiles)
	}

	return cleanName, nil
}
