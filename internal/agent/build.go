package agent

import (
	"errors"
	"fmt"
	"oblak/internal/agentproto"
	"os"
	"os/exec"
	"syscall"
)

type BuildJob struct{}

func (j *BuildJob) Run(conn *agentproto.Conn) error {
	e := newEmitter(conn)
	e.emit("system", "Preparing environment...")

	if err := j.mountDrives(); err != nil {
		return err
	}

	_, err := os.Stat("/app/requirements.txt")
	if err != nil && errors.Is(err, os.ErrNotExist) {
		e.emit("system", "No requirements.txt, skipping pip install...")
		j.unmountDrives()
		syscall.Sync()
		_ = conn.Send(agentproto.Done(0))
		return nil
	}

	cmd := exec.Command(
		"python3", "-m", "pip", "install",
		"-r", "/app/requirements.txt",
		"--target", "/deps",
		"--ignore-installed",
		"--root-user-action=ignore",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Credential: &syscall.Credential{
			Uid: 5000,
			Gid: 5000,
		},
	}

	if err := startWithStream(cmd, e.emit); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		e.emit("system", "Failed to install dependencies")
		j.cleanUp(cmd)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = conn.Send(agentproto.Done(exitErr.ExitCode()))
		}
		return err
	}

	j.cleanUp(cmd)
	_ = conn.Send(agentproto.Done(0))

	return nil
}

func (j *BuildJob) mountDrives() error {
	if err := mount(
		"/dev/vdb", "/app", "squashfs",
		syscall.MS_RDONLY|syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV,
	); err != nil {
		return err
	}
	if err := mount(
		"/dev/vdc", "/deps", "ext4",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV,
	); err != nil {
		return err
	}
	if err := os.Chown("/deps", 5000, 5000); err != nil {
		return fmt.Errorf("chown /deps: %w", err)
	}
	if err := mount(
		"tmpfs", "/tmp", "tmpfs",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV,
	); err != nil {
		return err
	}
	return nil
}

func (j *BuildJob) unmountDrives() {
	for _, target := range []string{"/app", "/deps"} {
		if err := syscall.Unmount(target, 0); err != nil {
			WriteErr("unmount "+target, err)
		}
	}
}

func (j *BuildJob) cleanUp(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	j.unmountDrives()
	syscall.Sync()
}
