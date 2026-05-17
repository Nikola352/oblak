//go:build linux && amd64

package main

import (
	"oblak/internal/agent"
	"os"
	"os/exec"
)

func main() {
	agent.SetPath()
	agent.MountSystemDirs()

	agent.WriteLog("Starting Python execution...")

	code := "print('hello world')"
	cmd := exec.Command("python3", "-c", code)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		agent.WriteErr("python execution failed", err)
	}

	agent.WriteLog("Execution complete.")
	agent.Shutdown()
}
