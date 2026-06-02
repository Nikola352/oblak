package agent

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"oblak/internal/agentproto"
	"os"
	"os/exec"
	"syscall"
)

type ExecJob struct{}

func (j *ExecJob) Run(conn *agentproto.Conn, payload string) error {
	e := newEmitter(conn)
	e.emit("system", "Execution started")

	if err := j.mountDrives(); err != nil {
		return err
	}

	if err := validatePayload(payload); err != nil {
		WriteErr("the payload is not valid json", err)
		e.emit("system", "The payload is not valid json")
		j.unmountDrives()
		syscall.Sync()
		_ = conn.Send(agentproto.Done(0))
		return nil
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	code := "import base64,json,handler; handler.handle(json.loads(base64.b64decode('" + encoded + "')))"

	cmd := exec.Command("python3", "-c", code)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Credential: &syscall.Credential{
			Uid: 5000,
			Gid: 5000,
		},
	}
	cmd.Env = append(os.Environ(), "PYTHONPATH=/deps")
	cmd.Dir = "/app"

	if err := startWithStream(cmd, e.emit); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		e.emit("system", "Execution failed")
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

func validatePayload(payload string) error {
	if !json.Valid([]byte(payload)) {
		return errors.New("payload is not valid JSON")
	}
	return nil
}

func (j *ExecJob) mountDrives() error {
	if err := mount(
		"/dev/vdb", "/app", "squashfs",
		syscall.MS_RDONLY|syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV,
	); err != nil {
		return err
	}
	if err := mount(
		"/dev/vdc", "/deps", "ext4",
		syscall.MS_RDONLY|syscall.MS_NOSUID|syscall.MS_NODEV,
	); err != nil {
		return err
	}
	if err := mount(
		"/dev/vdd", "/tmp", "ext4",
		syscall.MS_NOSUID|syscall.MS_NOEXEC|syscall.MS_NODEV,
	); err != nil {
		return err
	}
	if err := os.Chown("/tmp", 5000, 5000); err != nil {
		return fmt.Errorf("chown /tmp: %w", err)
	}
	return nil
}

func (j *ExecJob) unmountDrives() {
	for _, target := range []string{"/app", "/deps", "/tmp"} {
		if err := syscall.Unmount(target, syscall.MNT_DETACH); err != nil {
			WriteErr("unmount "+target, err)
		}
	}
}

// cleanUp kills any processes spawned by user code, unmounts drives,
// and flushes filesystem buffers before the done signal is sent.
// This ensures the host sees a consistent drive image when it copies it out of the VM.
func (j *ExecJob) cleanUp(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	j.unmountDrives()
	syscall.Sync()
}
