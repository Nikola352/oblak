package vm

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/google/uuid"
)

const (
	vsockPort      = 6000
	connectRetries = 20
	retryInterval  = 500 * time.Millisecond
)

type MicroVM struct {
	Id          string
	Cid         uint32
	LogPath     string
	vsockPath   string
	controlPath string
	logFile     *os.File
	machine     *firecracker.Machine
}

func StartMachine(ctx context.Context, drives []DriveMount) (*MicroVM, error) {
	id := uuid.New().String()
	controlPath := fmt.Sprintf("/tmp/fc-%s.sock", id)
	vsockPath := fmt.Sprintf("/tmp/fc-vsock-%s.sock", id)

	logPath := fmt.Sprintf("/tmp/vm-%s.log", id)

	for _, p := range []string{controlPath, vsockPath} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove socket %s: %w", p, err)
		}
	}

	driveConfigs := make([]models.Drive, len(drives))
	for i, mount := range drives {
		cfg := mount.Config
		cfg.DriveID = firecracker.String(fmt.Sprintf("drive-%d", i))
		driveConfigs[i] = cfg
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	cid := allocateCid()

	firecrackerCfg := firecracker.Config{
		SocketPath:      controlPath,
		KernelImagePath: "./deployment/firecracker/vmlinux-6.1.155",
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off root=/dev/vda rootfstype=squashfs ro init=/usr/local/bin/agent-runner",
		VsockDevices: []firecracker.VsockDevice{
			{
				ID:   "1",
				Path: vsockPath,
				CID:  cid,
			},
		},
		Drives: driveConfigs,
		MachineCfg: models.MachineConfiguration{
			MemSizeMib: firecracker.Int64(512),
			VcpuCount:  firecracker.Int64(1),
		},
	}

	cmd := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(controlPath).
		WithStdout(logFile).
		WithStderr(logFile).
		Build(ctx)

	machine, err := firecracker.NewMachine(ctx, firecrackerCfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		_ = logFile.Close()
		return nil, err
	}

	if err := machine.Start(ctx); err != nil {
		_ = logFile.Close()
		return nil, err
	}

	return &MicroVM{
		Id:          id,
		Cid:         cid,
		LogPath:     logPath,
		vsockPath:   vsockPath,
		controlPath: controlPath,
		logFile:     logFile,
		machine:     machine,
	}, nil
}

func (vm *MicroVM) Connect() (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)
	for i := range connectRetries {
		conn, err = net.Dial("unix", vm.vsockPath)
		if err != nil {
			time.Sleep(retryInterval)
			continue
		}

		if _, err = fmt.Fprintf(conn, "CONNECT %d\n", vsockPort); err != nil {
			if err = conn.Close(); err != nil {
				return nil, err
			}
			time.Sleep(retryInterval)
			continue
		}

		ack, _ := bufio.NewReader(conn).ReadString('\n')
		ack = strings.TrimRight(ack, "\n")
		if !strings.HasPrefix(ack, "OK ") {
			if err = conn.Close(); err != nil {
				return nil, err
			}
			err = fmt.Errorf("vsock handshake failed: %q (attempt %d)", ack, i+1)
			time.Sleep(retryInterval)
			continue
		}

		return conn, nil
	}
	return nil, fmt.Errorf("failed to connect to VM after %d attempts: %w", connectRetries, err)
}

func (vm *MicroVM) Stop() error {
	err := vm.machine.StopVMM()
	_ = vm.logFile.Close()
	_ = os.Remove(vm.vsockPath)
	_ = os.Remove(vm.controlPath)
	return err
}
