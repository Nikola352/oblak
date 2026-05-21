package dast

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// GVisorBox struct holds our initialized Docker client
type GVisorBox struct {
	cli *client.Client
}

// NewGVisorBox initializes the Moby client using environment variables
// and automatically negotiates the correct API version with the daemon.
func NewGVisorBox() (*GVisorBox, error) {
	// NewClientWithOpts is the standard for versions v24 through v28+
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &GVisorBox{cli: cli}, nil
}

func createTarStream(localPath, fileNameInContainer string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	content, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}

	hdr := &tar.Header{
		Name: fileNameInContainer,
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	tw.Close()
	return buf, nil
}
func (b *GVisorBox) Detonate(ctx context.Context, localPath string) (string, error) {
	config := &container.Config{
		Image:           "python:3.11-alpine",
		Cmd:             []string{"python", "/tmp/malware.py"},
		Env:             []string{"PYTHONUNBUFFERED=1"},
		NetworkDisabled: true,
		User:            "nobody",
	}

	hostConfig := &container.HostConfig{
		Runtime: "runsc",
		Annotations: map[string]string{
			"dev.gvisor.spec.strace":          "true",
			"dev.gvisor.spec.strace.log_path": "/dev/stderr",
		},
		// 1. Disable AutoRemove to prevent the container from vanishing before we read logs
		AutoRemove: false,
		Resources: container.Resources{
			Memory: 256 * 1024 * 1024,
		},
	}

	resp, err := b.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", err
	}

	// 2. Ensure cleanup happens even if the function returns early
	defer b.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})

	tarStream, err := createTarStream(localPath, "malware.py")
	if err != nil {
		return "", err
	}

	err = b.cli.CopyToContainer(ctx, resp.ID, "/tmp", tarStream, container.CopyToContainerOptions{})
	if err != nil {
		return "", err
	}

	if err := b.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	// 3. Wait for execution to finish
	statusCh, errCh := b.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", err
		}
	case <-statusCh:
	}

	// 4. Collect logs (Application output + gVisor strace)
	out, err := b.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", err
	}
	defer out.Close()

	var logBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&logBuf, &logBuf, out)
	if err != nil {
		return "", err
	}

	return logBuf.String(), nil
}
