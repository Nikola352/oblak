package vm

import (
	"context"
	"fmt"
	"log"
	"oblak/internal/agentproto"
	"oblak/internal/server/filestore"

	"github.com/minio/minio-go/v7"
)

type ExecutionRunner struct {
	filestore   *minio.Client
	logStreamer *InvocationLogStreamer
}

func NewExecutionRunner(minioClient *minio.Client) *ExecutionRunner {
	bucketRegistry := filestore.NewBucketRegistry()
	bucketName := bucketRegistry.Name(filestore.LogsBucket)
	return &ExecutionRunner{
		filestore:   minioClient,
		logStreamer: NewInvocationLogStreamer(minioClient, bucketName),
	}
}

func (e *ExecutionRunner) Execute(ctx context.Context, codeObjectName, depsObjectName, logsObjectName string) (err error) {
	var pendingCleanups []func() error
	defer func() {
		if err != nil {
			for _, c := range pendingCleanups {
				_ = c()
			}
		}
	}()

	codeBucket := filestore.NewBucketRegistry().Name(filestore.FunctionsBucket)
	codeDrive, err := ArchiveDrive(ctx, e.filestore, codeBucket, codeObjectName)
	if err != nil {
		return fmt.Errorf("failed to prepare code drive: %w", err)
	}
	pendingCleanups = append(pendingCleanups, codeDrive.Cleanup)

	drivesBucket := filestore.NewBucketRegistry().Name(filestore.DrivesBucket)
	depsDrive, err := MinioDrive(ctx, e.filestore, drivesBucket, depsObjectName)
	if err != nil {
		return fmt.Errorf("failed to prepare deps drive: %w", err)
	}
	pendingCleanups = append(pendingCleanups, depsDrive.Cleanup)

	tmpDrive, err := EphemeralDrive()
	if err != nil {
		return fmt.Errorf("failed to prepare tmp drive: %w", err)
	}
	pendingCleanups = append(pendingCleanups, tmpDrive.Cleanup)

	drives := []DriveMount{RootDrive(), codeDrive, depsDrive, tmpDrive}

	m, err := StartMachine(ctx, drives)
	if err != nil {
		return fmt.Errorf("failed to start build machine: %w", err)
	}
	pendingCleanups = nil
	defer m.CleanUpDrives()

	log.Printf("Log available at: %s\n", m.LogPath)
	log.Printf("Log about to start 1")

	rawConn, err := m.Connect()
	if err != nil {
		return err
	}
	conn := agentproto.NewConn(rawConn)
	defer func(conn *agentproto.Conn) {
		_ = conn.Close()
	}(conn)
	log.Printf("Log about to start")

	if err = conn.Send(agentproto.Exec()); err != nil {
		return err
	}

	if _, err = e.logStreamer.Stream(ctx, logsObjectName, conn); err != nil {
		return fmt.Errorf("failed during log streaming: %w", err)
	}
	log.Printf("Log streaming completed")

	if err = m.Stop(); err != nil {
		log.Printf("failed to stop vm %s: %v", m.Id, err)
	}

	return nil
}
