package agent

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

const defaultPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

func SetPath() {
	err := os.Setenv("PATH", defaultPath)
	if err != nil {
		WriteErr("cannot set PATH", err)
	}
}

func MountSystemDirs() {
	mountIfNotMounted("proc", "/proc", "proc", 0)
	mountIfNotMounted("sysfs", "/sys", "sysfs", 0)
	mountIfNotMounted("devtmpfs", "/dev", "devtmpfs", syscall.MS_NOSUID)
}

func mountIfNotMounted(source, target, fsType string, flags uintptr) {
	err := os.MkdirAll(target, 0755)
	if err != nil {
		WriteErr("failed to create mount point", err)
	}
	if err = syscall.Mount(source, target, fsType, flags, ""); err != nil {
		if !errors.Is(err, syscall.EBUSY) {
			WriteLog(fmt.Sprintf("mount %s -> %s: %v\n", source, target, err))
		}
	}
}

func mount(source, target, fsType string, flags uintptr) error {
	err := os.MkdirAll(target, 0755)
	if err != nil {
		WriteErr("failed to create mount point", err)
		return err
	}
	if err = syscall.Mount(source, target, fsType, flags, ""); err != nil {
		WriteErr(fmt.Sprintf("mount %s -> %s", source, target), err)
		return err
	}
	return nil
}

// Shutdown tells the kernel to halt.
// As PID 1, returning from main() would panic the kernel,
// so we use reboot(2) with LINUX_REBOOT_CMD_POWER_OFF.
func Shutdown() {
	WriteLog("Shutting down VM...")
	syscall.Sync()
	if err := syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF); err != nil {
		WriteErr("reboot failed", err)
	}
	os.Exit(0)
}
