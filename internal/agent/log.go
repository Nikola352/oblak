package agent

import "fmt"

func WriteLog(msg string) {
	fmt.Println("[agent]: " + msg)
}

func WriteErr(msg string, err error) {
	fmt.Printf("[agent] ERROR: %s: %v\n", msg, err)
}
