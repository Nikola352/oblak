package analyzer

import (
	"context"
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

func (fl *FileLoader) download(fileName string) error {
	err := fl.client.FGetObject(context.Background(), fl.bucketName, fileName, "/tmp/quarantine/"+fileName, minio.GetObjectOptions{})
	if err != nil {
		log.Fatalln(err)
	}
	return nil
}
