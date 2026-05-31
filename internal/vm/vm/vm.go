package vm

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"oblak/internal/agentproto"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

const (
	jailerBin      = "/usr/local/bin/jailer"
	jailerBase     = "/srv/jailer"
	firecrackerBin = "/usr/local/bin/firecracker"
	imageDir       = "/srv/firecracker"
	kernelImage    = "vmlinux-6.1.155"

	firecrackerUID = 900
	firecrackerGID = 900

	vsockPort      = 6000
	connectRetries = 20
	retryInterval  = 500 * time.Millisecond
)

// MicroVM is a handle to a running Firecracker microVM.
type MicroVM struct {
	Id            string
	Cid           uint32
	LogPath       string
	vsockPath     string
	jailerDir     string
	logFile       *os.File
	machine       *firecracker.Machine
	drives        []DriveMount
	cancelMachine context.CancelFunc
	stopOnce      sync.Once
}

// StartMachine launches a new Firecracker VM with the given drives and resource limits.
// The VM is automatically stopped when ctx is cancelled.
func StartMachine(ctx context.Context, drives []DriveMount, resources ResourceRequirements) (*MicroVM, error) {
	id := uuid.New().String()
	jailerDir := filepath.Join(jailerBase, "firecracker", id)

	if err := os.RemoveAll(jailerDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to remove stale jailer dir %s: %w", jailerDir, err)
	}

	logPath := fmt.Sprintf("/tmp/vm-%s.log", id)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	driveConfigs := make([]models.Drive, len(drives))
	for i, mount := range drives {
		cfg := mount.Config
		cfg.DriveID = firecracker.String(fmt.Sprintf("drive%d", i))
		driveConfigs[i] = cfg
	}

	cid := allocateCid()

	netnsPath := filepath.Join("/var/run/netns", id)

	uid := firecrackerUID
	gid := firecrackerGID
	numaNode := 0

	firecrackerCfg := firecracker.Config{
		SocketPath:      "api.socket",
		KernelImagePath: filepath.Join(imageDir, kernelImage),
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off root=/dev/vda rootfstype=squashfs ro init=/usr/local/bin/agent-runner",
		VsockDevices: []firecracker.VsockDevice{
			{
				ID:   "1",
				Path: "vsock.sock",
				CID:  cid,
			},
		},
		Drives: driveConfigs,
		MachineCfg: models.MachineConfiguration{
			MemSizeMib: firecracker.Int64(resources.MemoryMb),
			VcpuCount:  firecracker.Int64(resources.CpuCount),
		},
		NetworkInterfaces: firecracker.NetworkInterfaces{
			{
				CNIConfiguration: &firecracker.CNIConfiguration{
					NetworkName: "oblak",
					IfName:      "eth0",
				},
			},
		},
		NetNS: netnsPath,
		JailerCfg: &firecracker.JailerConfig{
			UID:           &uid,
			GID:           &gid,
			ID:            id,
			NumaNode:      &numaNode,
			ExecFile:      firecrackerBin,
			JailerBinary:  jailerBin,
			ChrootBaseDir: jailerBase,
			CgroupVersion: "2",
			ChrootStrategy: firecracker.NewNaiveChrootStrategy(
				filepath.Join(imageDir, kernelImage),
			),
			Stdout: logFile,
			Stderr: logFile,
		},
	}

	machineCtx, machineCancel := context.WithCancel(context.Background())

	machine, err := firecracker.NewMachine(machineCtx, firecrackerCfg)
	if err != nil {
		machineCancel()
		_ = logFile.Close()
		return nil, err
	}

	machine.Handlers.FcInit = machine.Handlers.FcInit.AppendAfter(
		firecracker.SetupNetworkHandlerName,
		firecracker.Handler{
			Name: "fixTapOwner",
			Fn: func(_ context.Context, _ *firecracker.Machine) error {
				return fixTapOwner(netnsPath)
			},
		},
	)

	if err := machine.Start(machineCtx); err != nil {
		machineCancel()
		_ = logFile.Close()
		return nil, err
	}

	vm := &MicroVM{
		Id:            id,
		Cid:           cid,
		LogPath:       logPath,
		vsockPath:     filepath.Join(jailerDir, "root", "vsock.sock"),
		jailerDir:     jailerDir,
		logFile:       logFile,
		machine:       machine,
		drives:        drives,
		cancelMachine: machineCancel,
	}

	// Safety net: if the caller's context is cancelled before Stop() is called,
	// run the full teardown so the VM doesn't run indefinitely.
	go func() {
		select {
		case <-ctx.Done():
			_ = vm.Stop()
		case <-machineCtx.Done():
		}
	}()

	return vm, nil
}

// Connect dials the VM's vsock port and returns a message connection to the in-VM agent.
// It retries up to connectRetries times before giving up.
func (vm *MicroVM) Connect() (*agentproto.Conn, error) {
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

		return agentproto.NewConn(conn), nil
	}
	return nil, fmt.Errorf("failed to connect to VM after %d attempts: %w", connectRetries, err)
}

// Stop shuts down the VM and frees all the resources it used.
func (vm *MicroVM) Stop() error {
	var err error
	vm.stopOnce.Do(func() {
		err = vm.machine.StopVMM()

		waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer waitCancel()
		_ = vm.machine.Wait(waitCtx)
		vm.cancelMachine()

		_ = vm.logFile.Close()
		_ = os.RemoveAll(vm.jailerDir)
	})
	return err
}

// CleanUpDrives calls the Cleanup function on each drive, logging any errors.
func (vm *MicroVM) CleanUpDrives() {
	for _, drive := range vm.drives {
		if drive.Cleanup == nil {
			continue
		}
		err := drive.Cleanup()
		if err != nil {
			log.Printf("error on drive cleanup: %v\n", err)
		}
	}
}

// fixTapOwner enters the VM's dedicated network namespace and transfers ownership  of tap0 to firecrackerUID/GID.
//
// CNI (tc-redirect-tap) creates the tap owned by the vm process user, but Firecracker runs as uid 900
// inside the jailer with no capabilities, so it cannot TUNSETIFF a tap it doesn't own.
func fixTapOwner(netnsPath string) error {
	// unix.Setns only affects the current thread
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	currNs, err := os.Open("/proc/self/ns/net")
	if err != nil {
		return fmt.Errorf("open current netns: %w", err)
	}
	defer func() { _ = currNs.Close() }()

	vmNs, err := os.Open(netnsPath)
	if err != nil {
		return fmt.Errorf("open VM netns %s: %w", netnsPath, err)
	}
	defer func() { _ = vmNs.Close() }()

	if err := unix.Setns(int(vmNs.Fd()), unix.CLONE_NEWNET); err != nil {
		return fmt.Errorf("setns into VM netns: %w", err)
	}
	defer func() {
		if err := unix.Setns(int(currNs.Fd()), unix.CLONE_NEWNET); err != nil {
			log.Printf("warning: failed to restore network namespace: %v", err)
		}
	}()

	tunFd, err := unix.Open("/dev/net/tun", unix.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open /dev/net/tun: %w", err)
	}
	defer func() { _ = unix.Close(tunFd) }()

	// TUNSETIFF: attach to the existing tap device.
	// ifreq layout: name[16] + flags[2] + padding[22]
	var ifr [40]byte
	copy(ifr[:unix.IFNAMSIZ], "tap0")
	*(*uint16)(unsafe.Pointer(&ifr[unix.IFNAMSIZ])) = unix.IFF_TAP | unix.IFF_NO_PI
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(tunFd), unix.TUNSETIFF, uintptr(unsafe.Pointer(&ifr[0]))); errno != 0 {
		return fmt.Errorf("TUNSETIFF tap0: %w", errno)
	}

	if err := unix.IoctlSetInt(tunFd, unix.TUNSETOWNER, firecrackerUID); err != nil {
		return fmt.Errorf("TUNSETOWNER %d: %w", firecrackerUID, err)
	}
	if err := unix.IoctlSetInt(tunFd, unix.TUNSETGROUP, firecrackerGID); err != nil {
		return fmt.Errorf("TUNSETGROUP %d: %w", firecrackerGID, err)
	}

	return nil
}
