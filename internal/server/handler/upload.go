package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"oblak/internal/api"
	"oblak/internal/function"
	"oblak/internal/server/events"
	"oblak/internal/server/filestore"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type ParsedFile struct {
	Reader   io.Reader
	Filename string
	Size     int64
}

func (h *Handler) UploadLambda(ctx context.Context, request api.UploadLambdaRequestObject) (api.UploadLambdaResponseObject, error) {
	ginCtx, ok := ctx.(*gin.Context)
	if !ok {
		return nil, fmt.Errorf("failed to get gin context")
	}
	userID, exists := ginCtx.Get("user_id")
	if !exists {
		return nil, fmt.Errorf("user_id not found in context")
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("user_id is not a string")
	}

	// upload to minIO
	bucketName := filestore.NewBucketRegistry().Name(filestore.QuarantineBucket)
	file, err := ParseUploadedFile(request.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading the file: %w", err)
	}

	objectName, err := UploadFile(ctx, h.filestore, file, bucketName)
	if err != nil {
		return nil, fmt.Errorf("error uploading file: %w", err)
	}

	// upload to DB
	f, err := h.functionStore.CreateFunction(ctx, userUUID, function.StatusQuarantined, bucketName, objectName)
	if err != nil {
		return nil, fmt.Errorf("error saving function to db: %w", err)
	}
	fmt.Println(f)

	//upload to rabbitMQ or channel event
	h.quarantineBus.Publish(events.QuarantineEvent{
		FunctionID: f.FunctionId,
		UserID:     f.UserId,
		Bucket:     f.Bucket,
		Path:       f.Path,
		Timestamp:  time.Now(),
	})

	return api.UploadLambda200JSONResponse{Status: "ok"}, nil
}

func UploadFile(ctx context.Context, filestore *minio.Client, file *ParsedFile, bucketName string) (string, error) {
	objectName := fmt.Sprintf("%s-%s", uuid.New().String(), file.Filename)

	info, err := filestore.PutObject(ctx,
		bucketName,
		objectName,
		file.Reader,
		file.Size,
		minio.PutObjectOptions{},
	)

	if err != nil {
		return "", fmt.Errorf("uploading to quarantine: %w", err)
	}
	log.Printf("Uploaded to quarantine: %s (size: %d)\n\n", info.Key, info.Size)
	return objectName, nil
}

func ParseUploadedFile(reader *multipart.Reader) (*ParsedFile, error) {
	fmt.Println("ParseUploadedFile: Starting to parse")

	partCount := 0
	for {
		part, err := reader.NextPart()
		partCount++
		fmt.Printf("ParseUploadedFile: Part %d, err: %v\n", partCount, err)

		if err == io.EOF {
			fmt.Println("ParseUploadedFile: Reached EOF")
			break
		}
		if err != nil {
			fmt.Printf("ParseUploadedFile: Error reading part: %v\n", err)
			return nil, fmt.Errorf("reading multipart part: %w", err)
		}

		fmt.Printf("ParseUploadedFile: Found part - FormName: %q, FileName: %q\n", part.FormName(), part.FileName())

		if part.FormName() == "function" {
			filename := part.FileName()
			if filename == "" {
				filename = "unknown.tar.gz"
			}

			fmt.Printf("ParseUploadedFile: Found file! Filename: %s\n", filename)

			return &ParsedFile{
				Reader:   part,
				Filename: filename,
				Size:     -1,
			}, nil
		}
	}

	fmt.Println("ParseUploadedFile: No file found")
	return nil, fmt.Errorf("no file found with field name 'function'")
}
