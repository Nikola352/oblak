package agent

import (
	"bufio"
	"errors"
	"io"
	"oblak/internal/agentproto"
	"os/exec"
	"sync"
	"syscall"
)

func ExecuteCode(conn *agentproto.Conn) error {
	var mu sync.Mutex
	emit := func(channel, data string) {
		mu.Lock()
		defer mu.Unlock()
		_ = conn.Send(agentproto.Output(channel, data))
	}

	emit("system", "Execution started")

	code := "print('hello world')"
	cmd := exec.Command("python3", "-c", code)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	stream := func(r io.Reader, channel string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			emit(channel, scanner.Text()+"\n")
		}
	}

	wg.Add(2)
	go stream(stdout, "stdout")
	go stream(stderr, "stderr")
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		emit("system", "Execution failed")
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
