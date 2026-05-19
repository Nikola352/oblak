package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/minio/minio-go/v7"
)

const driveSizeMB = 512

type DriveMount struct {
	Config  models.Drive
	Cleanup func() error
}

// RootDrive returns the shared read-only root filesystem drive.
func RootDrive() DriveMount {
	cfg := models.Drive{
		PathOnHost:   firecracker.String("./deployment/firecracker/rootfs.squashfs"),
		IsRootDevice: firecracker.Bool(true),
		IsReadOnly:   firecracker.Bool(true),
	}
	return DriveMount{cfg, func() error { return nil }}
}

// MinioDrive downloads objectName from Minio into a temp file and returns a read-only drive.
// Cleanup deletes the temp file.
func MinioDrive(ctx context.Context, filestore *minio.Client, bucket, objectName string) (DriveMount, error) {
	file, err := os.CreateTemp("", "drive-*.ext4")
	if err != nil {
		return DriveMount{}, err
	}
	if err = file.Close(); err != nil {
		return DriveMount{}, err
	}

	if err := filestore.FGetObject(ctx, bucket, objectName, file.Name(), minio.GetObjectOptions{}); err != nil {
		_ = os.Remove(file.Name())
		return DriveMount{}, err
	}

	cfg := models.Drive{
		PathOnHost:   firecracker.String(file.Name()),
		IsRootDevice: firecracker.Bool(false),
		IsReadOnly:   firecracker.Bool(true),
	}

	cleanup := func() error {
		return os.Remove(file.Name())
	}

	return DriveMount{cfg, cleanup}, nil
}

// MinioUploadDrive creates a blank writable drive. Cleanup uploads the drive to objectName in Minio, then deletes the local file.
func MinioUploadDrive(filestore *minio.Client, bucket, objectName string) (DriveMount, error) {
	path, err := createBlankExt4(driveSizeMB)
	if err != nil {
		return DriveMount{}, err
	}

	cfg := models.Drive{
		PathOnHost:   firecracker.String(path),
		IsRootDevice: firecracker.Bool(false),
		IsReadOnly:   firecracker.Bool(false),
	}

	cleanup := func() error {
		_, err := filestore.FPutObject(
			context.Background(),
			bucket,
			objectName,
			path,
			minio.PutObjectOptions{},
		)
		if err != nil {
			return fmt.Errorf("upload drive to minio: %w", err)
		}
		return os.Remove(path)
	}

	return DriveMount{cfg, cleanup}, nil
}

// EphemeralDrive creates a blank writable drive. Cleanup discards it.
func EphemeralDrive() (DriveMount, error) {
	path, err := createBlankExt4(driveSizeMB)
	if err != nil {
		return DriveMount{}, err
	}

	cfg := models.Drive{
		PathOnHost:   firecracker.String(path),
		IsRootDevice: firecracker.Bool(false),
		IsReadOnly:   firecracker.Bool(false),
	}

	cleanup := func() error {
		return os.Remove(path)
	}

	return DriveMount{cfg, cleanup}, nil
}

// ArchiveDrive downloads a tar.gz from Minio, extracts it, and packs it into a read-only squashfs drive.
// Cleanup deletes the squashfs file.
func ArchiveDrive(ctx context.Context, filestore *minio.Client, bucket, objectName string) (DriveMount, error) {
	tgz, err := os.CreateTemp("", "archive-*.tar.gz")
	if err != nil {
		return DriveMount{}, err
	}
	tgzPath := tgz.Name()
	defer func(name string) {
		_ = os.Remove(name)
	}(tgzPath)

	if err = tgz.Close(); err != nil {
		return DriveMount{}, err
	}
	if err = filestore.FGetObject(ctx, bucket, objectName, tgzPath, minio.GetObjectOptions{}); err != nil {
		return DriveMount{}, err
	}

	extractDir, err := os.MkdirTemp("", "archive-extract-*")
	if err != nil {
		return DriveMount{}, err
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(extractDir)

	if err = exec.CommandContext(ctx, "tar", "-xzf", tgzPath, "-C", extractDir).Run(); err != nil {
		return DriveMount{}, err
	}

	squashfs, err := os.CreateTemp("", "drive-*.sqfs")
	if err != nil {
		return DriveMount{}, err
	}
	squashfsPath := squashfs.Name()
	if err = squashfs.Close(); err != nil {
		_ = os.Remove(squashfsPath)
		return DriveMount{}, err
	}

	// -noappend overwrites the pre-created temp file instead of appending to it
	if err = exec.CommandContext(ctx, "mksquashfs", extractDir, squashfsPath, "-noappend").Run(); err != nil {
		_ = os.Remove(squashfsPath)
		return DriveMount{}, err
	}

	cfg := models.Drive{
		PathOnHost:   firecracker.String(squashfsPath),
		IsRootDevice: firecracker.Bool(false),
		IsReadOnly:   firecracker.Bool(true),
	}

	cleanup := func() error {
		return os.Remove(squashfsPath)
	}

	return DriveMount{cfg, cleanup}, nil
}

// createBlankExt4 creates a temp file of sizeMB formatted as ext4 and returns its path.
func createBlankExt4(sizeMB int64) (string, error) {
	file, err := os.CreateTemp("", "drive-*.ext4")
	if err != nil {
		return "", err
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}

	path := file.Name()

	if err = os.Truncate(path, sizeMB*1024*1024); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	if err = exec.Command("mkfs.ext4", "-F", path).Run(); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	return path, nil
}
