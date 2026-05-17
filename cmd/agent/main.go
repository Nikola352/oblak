//go:build linux && amd64

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

const defaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

func main() {
	err := os.Setenv("PATH", defaultPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[agent] ERROR: cannot set PATH: %v\n", err)
	}

	mustMount("proc", "/proc", "proc", 0)
	mustMount("sysfs", "/sys", "sysfs", 0)
	mustMountIfNotMounted("devtmpfs", "/dev", "devtmpfs", syscall.MS_NOSUID)

	fmt.Println("[agent] Starting Python execution...")

	code := "print('hello world')"
	cmd := exec.Command("python3", "-c", code)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "[agent] ERROR: %v\n", err)
	}

	fmt.Println("[agent] Execution complete.")
	shutdown()
}

func mustMount(source, target, fsType string, flags uintptr) {
	os.MkdirAll(target, 0755)
	if err := syscall.Mount(source, target, fsType, flags, ""); err != nil {
		fmt.Printf("[agent] mount %s -> %s: %v\n", source, target, err)
	}
}

func mustMountIfNotMounted(source, target, fsType string, flags uintptr) {
	os.MkdirAll(target, 0755)
	if err := syscall.Mount(source, target, fsType, flags, ""); err != nil {
		if !errors.Is(err, syscall.EBUSY) {
			fmt.Printf("[agent] mount %s -> %s: %v\n", source, target, err)
		}
	}
}

// shutdown tells the kernel to halt.
// As PID 1, returning from main() would panic the kernel,
// so we use reboot(2) with LINUX_REBOOT_CMD_POWER_OFF.
func shutdown() {
	fmt.Println("[agent] Shutting down VM...")
	syscall.Sync()
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		fmt.Printf("[agent] reboot failed: %v\n", err)
	}
	os.Exit(0)
}
