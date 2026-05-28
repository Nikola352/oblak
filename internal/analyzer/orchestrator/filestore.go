package orchestrator

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type FileStore struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string // Acts as the default source/quarantine bucket
	client     *minio.Client
}

func NewFileStore(endpoint, accessKey, secretKey, bucketName string) *FileStore {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false, // Set to true if using HTTPS
	})
	if err != nil {
		log.Fatalln(err)
	}
	return &FileStore{
		endpoint:   endpoint,
		accessKey:  accessKey,
		secretKey:  secretKey,
		client:     client,
		bucketName: bucketName,
	}
}

func (fl *FileStore) Download(ctx context.Context, fileName string, functionId uuid.UUID) (string, error) {
	localPath := "/tmp/quarantine/" + functionId.String() + "/" + fileName

	err := fl.client.FGetObject(ctx, fl.bucketName, fileName, localPath, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("minio download failed: %w", err)
	}

	return localPath, nil
}

func (fl *FileStore) Move(ctx context.Context, fileName, destBucket string) error {
	srcOpts := minio.CopySrcOptions{
		Bucket: fl.bucketName,
		Object: fileName,
	}

	destOpts := minio.CopyDestOptions{
		Bucket: destBucket,
		Object: fileName,
	}

	log.Printf("[MINIO] Moving object '%s' from bucket '%s' to '%s'", fileName, fl.bucketName, destBucket)

	_, err := fl.client.CopyObject(ctx, destOpts, srcOpts)
	if err != nil {
		return fmt.Errorf("failed copying file to target bucket %s: %w", destBucket, err)
	}

	err = fl.client.RemoveObject(ctx, fl.bucketName, fileName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("critical failure: object copied but failed to clear from source bucket: %w", err)
	}

	log.Printf("[MINIO] Object '%s' successfully moved to bucket '%s'", fileName, destBucket)
	return nil
}
