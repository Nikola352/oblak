package agentproto

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const MaxMessageSize = 1 << 20 // 1 MiB

const (
	TypeBuild  = "build"
	TypeExec   = "exec"
	TypeOutput = "output"
	TypeDone   = "done"
	TypeError  = "error"
)

type Message struct {
	Type     string `json:"type"`
	Channel  string `json:"channel,omitempty"`
	Data     string `json:"data,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

func (m Message) String() string {
	if m.ExitCode != nil {
		return fmt.Sprintf("{type:%s exit_code:%d error:%s}", m.Type, *m.ExitCode, m.Error)
	}
	return fmt.Sprintf("{type:%s channel:%s data:%s}", m.Type, m.Channel, m.Data)
}

func Build() Message {
	return Message{Type: TypeBuild}
}

func Exec(payload string) Message {
	return Message{
		Type: TypeExec,
		Data: payload,
	}
}

func Output(channel, data string) Message {
	return Message{Type: TypeOutput, Channel: channel, Data: data}
}

func Done(exitCode int) Message {
	return Message{Type: TypeDone, ExitCode: &exitCode}
}

func AgentError(msg string) Message {
	return Message{Type: TypeError, Error: msg}
}

type Conn struct {
	conn    net.Conn
	scanner *bufio.Scanner
}

func NewConn(c net.Conn) *Conn {
	s := bufio.NewScanner(c)
	s.Buffer(make([]byte, MaxMessageSize), MaxMessageSize)
	return &Conn{conn: c, scanner: s}
}

func (c *Conn) Send(m Message) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.conn, "%s\n", data)
	return err
}

func (c *Conn) Receive() (Message, error) {
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return Message{}, err
		}
		return Message{}, io.EOF
	}
	var m Message
	return m, json.Unmarshal(c.scanner.Bytes(), &m)
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
