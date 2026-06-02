//go:build linux && amd64

package main

import (
	"oblak/internal/agent"
)

func main() {
	agent.SetPath()
	agent.MountSystemDirs()

	agent.WriteLog("Listening for commands...")

	err := agent.ListenForCommands()
	if err != nil {
		agent.WriteErr("", err)
	}

	agent.WriteLog("Execution complete.")
	agent.Shutdown()
}
