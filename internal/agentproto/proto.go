package agentproto

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

type Message struct {
	Type     string `json:"type"`
	Channel  string `json:"channel,omitempty"`
	Data     string `json:"data,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

func Build() Message {
	return Message{Type: "build"}
}

func Exec() Message {
	return Message{Type: "exec"}
}

func Output(channel, data string) Message {
	return Message{Type: "output", Channel: channel, Data: data}
}

func Done(exitCode int) Message {
	return Message{Type: "done", ExitCode: &exitCode}
}

func AgentError(msg string) Message {
	return Message{Type: "error", Error: msg}
}

type Conn struct {
	conn net.Conn
	r    *bufio.Reader
}

func NewConn(c net.Conn) *Conn {
	return &Conn{conn: c, r: bufio.NewReader(c)}
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
	line, err := c.r.ReadBytes('\n')
	if err != nil {
		return Message{}, err
	}
	var m Message
	return m, json.Unmarshal(line, &m)
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
