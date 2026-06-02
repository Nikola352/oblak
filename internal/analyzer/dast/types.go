package dast

type ExecutionResult struct {
	Stdout    string         `json:"stdout"`
	Stderr    string         `json:"stderr"`
	Behavior  BehaviorReport `json:"behavior"`
	RawStrace string         `json:"-"` // available in memory, excluded from JSON
}

// SyscallEntry represents a single parsed strace event from gVisor
type SyscallEntry struct {
	Timestamp string `json:"ts"`
	Syscall   string `json:"syscall"`
	Args      string `json:"args"`
	Return    string `json:"return"`
	Category  string `json:"category"` // "fs", "net", "proc", etc.
}

// BehaviorReport holds only security-relevant observations extracted from strace.
type BehaviorReport struct {
	FSAccess    []FSEvent   `json:"fs_access"`
	NetEvents   []NetEvent  `json:"net_events"`
	ProcEvents  []ProcEvent `json:"proc_events"`
	RawSyscalls []string    `json:"raw_syscalls,omitempty"`
}

type FSEvent struct {
	Timestamp string `json:"ts"`
	Op        string `json:"op"` // open, read, write, stat, rename, unlink
	Path      string `json:"path"`
	Flags     string `json:"flags,omitempty"`
	Result    string `json:"result"` // fd number or errno
}

type NetEvent struct {
	Timestamp string `json:"ts"`
	Op        string `json:"op"`     // socket, connect, bind, sendto, recvfrom
	Detail    string `json:"detail"` // address/port if available
	Result    string `json:"result"`
}

type ProcEvent struct {
	Timestamp string `json:"ts"`
	Op        string `json:"op"`   // execve, vfork, clone, kill
	Args      string `json:"args"` // argv for execve, signal for kill
	Result    string `json:"result"`
}
