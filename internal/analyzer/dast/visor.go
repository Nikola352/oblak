package dast

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

const runscLogDir = "/tmp/runsc-logs"

type GVisorBox struct {
	cli *client.Client
}

func NewGVisorBox() (*GVisorBox, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}
	return &GVisorBox{cli: cli}, nil
}

// ---- Detonate ----

func (b *GVisorBox) Detonate(ctx context.Context, localPath string) (*ExecutionResult, error) {
	before, err := snapshotLogFiles()
	if err != nil {
		before = map[string]struct{}{}
	}

	config := &container.Config{
		Image:           "python:3.11-alpine",
		Cmd:             []string{"python", "/tmp/script.py"},
		Env:             []string{"PYTHONUNBUFFERED=1"},
		NetworkDisabled: true,
	}

	hostConfig := &container.HostConfig{
		Runtime:    "runsc",
		AutoRemove: false,
		Resources: container.Resources{
			Memory: 256 * 1024 * 1024,
		},
	}

	resp, err := b.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return nil, err
	}
	defer func(cli *client.Client, ctx context.Context, containerID string, options container.RemoveOptions) {
		err := cli.ContainerRemove(ctx, containerID, options)
		if err != nil {

		}
	}(b.cli, ctx, resp.ID, container.RemoveOptions{Force: true})

	tarStream, err := createTarStream(localPath, "script.py")
	if err != nil {
		return nil, err
	}
	if err := b.cli.CopyToContainer(ctx, resp.ID, "/tmp", tarStream, container.CopyToContainerOptions{}); err != nil {
		return nil, err
	}

	if err := b.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

	// Find the boot log for this run — appears within milliseconds of ContainerStart
	bootLogPath, bootErr := findNewBootLog(before, 5*time.Second)

	statusCh, errCh := b.cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return nil, err
		}
	case <-statusCh:
	}

	dockerOut, err := b.cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return nil, err
	}
	defer dockerOut.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, dockerOut); err != nil {
		return nil, err
	}

	rawStrace := ""
	if bootErr != nil {
		rawStrace = "(strace unavailable: " + bootErr.Error() + ")"
	} else {
		data, err := os.ReadFile(bootLogPath)
		if err != nil {
			rawStrace = "(could not read boot log: " + err.Error() + ")"
		} else {
			rawStrace = string(data)
		}
	}

	return &ExecutionResult{
		Stdout:    stdoutBuf.String(),
		Stderr:    stderrBuf.String(),
		Behavior:  parseBehaviorReport(rawStrace),
		RawStrace: rawStrace,
	}, nil
}

func (_ *GVisorBox) WriteJSONReport(result *ExecutionResult, outputPath string) error {
	log.Println("Writing JSON report")
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(result)
}
