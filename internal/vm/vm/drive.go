package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/minio/minio-go/v7"
)

const (
	driveSizeMB = 512
	driveDir    = jailerBase + "/drives"
)

type DriveMount struct {
	Config  models.Drive
	Save    func() error
	Cleanup func() error
}

// RootDrive returns the shared read-only root filesystem drive.
func RootDrive() DriveMount {
	return DriveMount{Config: models.Drive{
		PathOnHost:   firecracker.String(filepath.Join(imageDir, "rootfs.squashfs")),
		IsRootDevice: firecracker.Bool(true),
		IsReadOnly:   firecracker.Bool(true),
	}}
}

// MinioDrive downloads objectName from Minio into a temp file and returns a read-only drive.
// Cleanup deletes the temp file.
func MinioDrive(ctx context.Context, filestore *minio.Client, bucket, objectName string) (DriveMount, error) {
	file, err := os.CreateTemp(driveDir, "drive-*.ext4")
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

	if err := os.Chmod(file.Name(), 0o240); err != nil {
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

	return DriveMount{Config: cfg, Cleanup: cleanup}, nil
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

	save := func() error {
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
		return nil
	}

	cleanup := func() error {
		return os.Remove(path)
	}

	return DriveMount{cfg, save, cleanup}, nil
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

	return DriveMount{Config: cfg, Cleanup: cleanup}, nil
}

// ArchiveDrive downloads a tar.gz from Minio, extracts it, and packs it into a read-only squashfs drive.
// Cleanup deletes the squashfs file.
func ArchiveDrive(ctx context.Context, filestore *minio.Client, bucket, objectName string) (DriveMount, error) {
	tgz, err := os.CreateTemp("", "archive-*.tar.gz")
	if err != nil {
		return DriveMount{}, err
	}
	tgzPath := tgz.Name()
	defer func() {
		_ = os.Remove(tgzPath)
	}()

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
	defer func() {
		_ = os.RemoveAll(extractDir)
	}()

	if err = exec.CommandContext(ctx, "tar", "-xzf", tgzPath, "-C", extractDir).Run(); err != nil {
		return DriveMount{}, err
	}

	squashfs, err := os.CreateTemp(driveDir, "drive-*.sqfs")
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

	if err = os.Chmod(squashfsPath, 0o240); err != nil {
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

	return DriveMount{Config: cfg, Cleanup: cleanup}, nil
}

// createBlankExt4 creates a temp file of sizeMB formatted as ext4 and returns its path.
func createBlankExt4(sizeMB int64) (string, error) {
	file, err := os.CreateTemp(driveDir, "drive-*.ext4")
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

	if err = os.Chmod(path, 0o660); err != nil {
		_ = os.Remove(path)
		return "", err
	}

	return path, nil
}
