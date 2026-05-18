package agent

import (
	"oblak/internal/agentproto"

	"github.com/mdlayher/vsock"
)

const vsockPort = 6000

func ListenForCommands() error {
	listener, err := vsock.Listen(vsockPort, nil)
	if err != nil {
		return err
	}
	defer func(listener *vsock.Listener) {
		_ = listener.Close()
	}(listener)

	rawConn, err := listener.Accept()
	if err != nil {
		return err
	}
	conn := agentproto.NewConn(rawConn)
	defer func(conn *agentproto.Conn) {
		_ = conn.Close()
	}(conn)

	msg, err := conn.Receive()
	if err != nil {
		return err
	}

	switch msg.Type {
	case "exec":
		WriteLog("Starting execution...")
		if err = ExecuteCode(conn); err != nil {
			WriteErr("execution failed", err)
		}
	}

	// Wait for host to close the connection before shutting down.
	// This ensures all buffered data (including the done message) has been
	// proxied through Firecracker before the VM powers off.
	_, _ = conn.Receive()

	return err
}
