package agent

import (
	"errors"
	"oblak/internal/agentproto"
	"os/exec"
	"syscall"
)

func ExecuteCode(conn *agentproto.Conn) error {
	e := newEmitter(conn)
	e.emit("system", "Execution started")

	code := "print('hello world')"
	cmd := exec.Command("python3", "-c", code)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := startWithStream(cmd, e.emit); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		e.emit("system", "Execution failed")
		cleanUp(cmd)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_ = conn.Send(agentproto.Done(exitErr.ExitCode()))
		}
		return err
	}

	cleanUp(cmd)
	_ = conn.Send(agentproto.Done(0))

	return nil
}

// cleanUp kills any processes spawned by user code and flushes filesystem buffers before the done signal is sent.
//
// This ensures the host sees a consistent drive image when it copies it out of the VM.
// In the future, if such drives are added, unmount them here.
func cleanUp(cmd *exec.Cmd) {
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	syscall.Sync()
}
