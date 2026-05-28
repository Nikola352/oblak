package agent

import (
	"errors"
	"oblak/internal/agentproto"
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

	cmd := exec.Command(
		"python3", "-m", "pip", "install",
		"-r", "/app/requirements.txt",
		"--target", "/deps",
		"--ignore-installed",
		"--root-user-action=ignore",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
		if err := syscall.Unmount(target, syscall.MNT_DETACH); err != nil {
			WriteErr("unmount "+target, err)
		}
	}
}

func (j *BuildJob) cleanUp(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	j.unmountDrives()
	syscall.Sync()
}
