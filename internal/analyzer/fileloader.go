package analyzer

import (
	"context"
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileLoader struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	client     *minio.Client
}

func NewFileLoader(endpoint, accessKey, secretKey, bucketName string) *FileLoader {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // Set to true if using HTTPS
	})
	if err != nil {
		log.Fatalln(err)
	}
	return &FileLoader{
		endpoint:   endpoint,
		accessKey:  accessKey,
		secretKey:  secretKey,
		client:     client,
		bucketName: bucketName,
	}
}

func (fl *FileLoader) Download(ctx context.Context, fileName string) (string, error) {
	localPath := "/tmp/quarantine/" + fileName

	err := fl.client.FGetObject(ctx, fl.bucketName, fileName, localPath, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("minio download failed: %w", err)
	}

	return localPath, nil
}
