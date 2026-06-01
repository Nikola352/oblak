package dast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

func createContainerConfig() *container.Config {
	return &container.Config{
		Image: "python:3.11-alpine",
		User:  "nobody",
		Cmd: []string{"sh", "-c", `
            if [ -f /sample/requirements.txt ]; then
                pip install -q --no-user --target=/tmp/pypackages -r /sample/requirements.txt
            fi
            python -c "import sys; sys.path.insert(0, '/tmp/pypackages'); sys.path.insert(0, '/sample'); import handler; handler.handle('')"
        `},
		Env: []string{
			"PYTHONUNBUFFERED=1",
			"PYTHONPATH=/tmp/pypackages:/sample",
			"PIP_CACHE_DIR=/tmp/pip-cache",
			"HOME=/tmp",
		},
		WorkingDir:      "/sample",
		NetworkDisabled: false,
	}
}

func createHostConfig(dirPath string) *container.HostConfig {
	pidsLimit := int64(64)
	return &container.HostConfig{
		Runtime:    "runsc",
		AutoRemove: false,
		// Sample mounted at /sample — outside /tmp so the tmpfs doesn't shadow it
		Binds: []string{dirPath + ":/sample:ro"},
		Resources: container.Resources{
			Memory:    256 * 1024 * 1024,
			CPUQuota:  50000,
			CPUPeriod: 100000,
			PidsLimit: &pidsLimit,
		},
		CapDrop:        []string{"ALL"},
		SecurityOpt:    []string{"no-new-privileges"},
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			// Single /tmp tmpfs — pip uses /tmp/pypackages and /tmp/pip-cache
			// Both are subdirs of this mount, no nested mounts needed
			"/tmp": "rw,noexec,nosuid,size=256m,uid=65534,gid=65534",
		},
	}
}

func (b *GVisorBox) Detonate(ctx context.Context, dirPath string) (*ExecutionResult, error) {
	// Timeout governs the entire detonation, not just log fetching
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	before, err := snapshotLogFiles()
	if err != nil {
		before = map[string]struct{}{}
	}

	config := createContainerConfig()
	hostConfig := createHostConfig(dirPath)

	resp, err := b.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return nil, err
	}

	// Cleanup uses a fresh context so a cancelled ctx doesn't block removal
	defer func() {
		removeCtx, removeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer removeCancel()
		if err := b.cli.ContainerRemove(removeCtx, resp.ID, container.RemoveOptions{Force: true}); err != nil {
			log.Printf("failed to remove container %s: %v", resp.ID, err)
		}
	}()

	// Files are bind-mounted at /sample; no tar copy needed
	if err := b.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, err
	}

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
	if _, err := stdcopy.StdCopy(
		io.MultiWriter(&stdoutBuf, os.Stdout),
		io.MultiWriter(&stderrBuf, os.Stderr),
		dockerOut,
	); err != nil {
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

func (*GVisorBox) WriteJSONReport(result *ExecutionResult, outputPath string) error {
	log.Println("Writing JSON report")

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory path: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			log.Printf("failed to close report file: %v", err)
		}
	}(f)

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
