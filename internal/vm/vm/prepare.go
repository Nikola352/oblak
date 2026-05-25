package vm

import (
	"context"
	"fmt"
	"log"
	"oblak/internal/agentproto"
	"oblak/internal/server/filestore"

	"github.com/minio/minio-go/v7"
)

type EnvironmentPrepareRunner struct {
	filestore *minio.Client
}

func NewEnvironmentPrepareRunner(filestore *minio.Client) *EnvironmentPrepareRunner {
	return &EnvironmentPrepareRunner{filestore}
}

func (ep *EnvironmentPrepareRunner) PrepareEnvironment(ctx context.Context, codeObjectName, depsObjectName string) (err error) {
	var pendingCleanups []func() error
	defer func() {
		if err != nil {
			for _, c := range pendingCleanups {
				_ = c()
			}
		}
	}()

	codeBucket := filestore.NewBucketRegistry().Name(filestore.FunctionsBucket)
	archiveDrive, err := ArchiveDrive(ctx, ep.filestore, codeBucket, codeObjectName)
	if err != nil {
		return fmt.Errorf("failed to prepare code drive: %w", err)
	}
	pendingCleanups = append(pendingCleanups, archiveDrive.Cleanup)

	drivesBucket := filestore.NewBucketRegistry().Name(filestore.DrivesBucket)
	depsDrive, err := MinioUploadDrive(ep.filestore, drivesBucket, depsObjectName)
	if err != nil {
		return fmt.Errorf("failed to prepare deps drive: %w", err)
	}
	pendingCleanups = append(pendingCleanups, depsDrive.Cleanup)

	drives := []DriveMount{RootDrive(), archiveDrive, depsDrive}

	m, err := StartMachine(ctx, drives)
	if err != nil {
		return fmt.Errorf("failed to start build machine: %w", err)
	}
	pendingCleanups = nil
	defer m.CleanUpDrives()

	log.Printf("Log available at: %s\n", m.LogPath)

	rawConn, err := m.Connect()
	if err != nil {
		return err
	}
	conn := agentproto.NewConn(rawConn)
	defer func(conn *agentproto.Conn) {
		_ = conn.Close()
	}(conn)

	if err = conn.Send(agentproto.Build()); err != nil {
		return err
	}

	var isSuccessful = false
	for {
		msg, err := conn.Receive()
		if err != nil {
			return err
		}
		log.Println(msg) // TODO: stream outputs to minio
		if msg.Type == agentproto.TypeDone || msg.Type == agentproto.TypeError {
			isSuccessful = msg.Type == agentproto.TypeDone && *msg.ExitCode == 0
			break
		}
	}

	if err = m.Stop(); err != nil {
		log.Printf("failed to stop vm %s: %v", m.Id, err)
	}

	if isSuccessful {
		err = depsDrive.Save()
		if err != nil {
			return fmt.Errorf("failed to upload prepared env to minio: %v\n", err)
		}
	} else {
		return fmt.Errorf("failed to install dependencies")
	}

	return nil
}
