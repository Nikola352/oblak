package agent

import (
	"bufio"
	"io"
	"oblak/internal/agentproto"
	"os/exec"
	"sync"
)

type emitter struct {
	mu   sync.Mutex
	conn *agentproto.Conn
}

func newEmitter(conn *agentproto.Conn) *emitter {
	return &emitter{conn: conn}
}

func (e *emitter) emit(channel, data string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	_ = e.conn.Send(agentproto.Output(channel, data))
}

// startWithStream attaches stdout/stderr pipes to cmd, starts it, and fans both
// streams to emit. The caller is responsible for calling cmd.Wait().
func startWithStream(cmd *exec.Cmd, emit func(channel, data string)) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	pipe := func(r io.Reader, channel string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			emit(channel, scanner.Text()+"\n")
		}
	}

	wg.Add(2)
	go pipe(stdout, "stdout")
	go pipe(stderr, "stderr")
	wg.Wait()

	return nil
}
