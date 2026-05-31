package vm

import (
	"context"
	"fmt"
	"io"
	"log"
	"oblak/internal/agentproto"

	"github.com/minio/minio-go/v7"
)

type InvocationLogStreamer struct {
	minioClient *minio.Client
	bucketName  string
}

func NewInvocationLogStreamer(client *minio.Client, bucketName string) *InvocationLogStreamer {
	return &InvocationLogStreamer{
		minioClient: client,
		bucketName:  bucketName,
	}
}
func (s *InvocationLogStreamer) Stream(ctx context.Context, objectName string, conn *agentproto.Conn) (*agentproto.Message, error) {
	pr, pw := io.Pipe()
	errChan := make(chan error, 1)

	go func() {
		defer func() { _ = pr.Close() }()
		_, err := s.minioClient.PutObject(ctx, s.bucketName, objectName, pr, -1, minio.PutObjectOptions{
			ContentType: "text/plain",
		})
		errChan <- err
	}()

	streamDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-streamDone:
		}
	}()

	var finalMsg *agentproto.Message
	var streamErr error

	for {
		msg, err := conn.Receive()
		if err != nil {
			streamErr = fmt.Errorf("connection receive failed: %w", err)
			break
		}

		// Write log line to MinIO stream
		_, writeErr := fmt.Fprintf(pw, "%v\n", msg)
		if writeErr != nil {
			log.Println("Failed to write log line to pipe:", writeErr)
		}

		// Track and break when complete
		if msg.Type == agentproto.TypeDone || msg.Type == agentproto.TypeError {
			finalMsg = &msg
			break
		}
	}

	close(streamDone)

	if streamErr != nil {
		_ = pw.CloseWithError(streamErr)
		<-errChan
		return nil, streamErr
	}

	_ = pw.Close()

	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("minio upload failed: %w", err)
	}

	return finalMsg, nil
}
