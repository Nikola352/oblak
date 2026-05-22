package dast

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func createTarStream(localPath, fileNameInContainer string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	content, err := os.ReadFile(localPath)
	if err != nil {
		return nil, err
	}

	hdr := &tar.Header{
		Name: fileNameInContainer,
		Mode: 0644,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return nil, err
	}
	if _, err := tw.Write(content); err != nil {
		return nil, err
	}
	tw.Close()
	return buf, nil
}

// ---- Container helpers ----

func (b *GVisorBox) readFileFromContainer(ctx context.Context, containerID, path string) (string, error) {
	rc, _, err := b.cli.CopyFromContainer(ctx, containerID, path)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	if _, err := tr.Next(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, tr); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ---- Log file helpers ----

func snapshotLogFiles() (map[string]struct{}, error) {
	entries, err := os.ReadDir(runscLogDir)
	if err != nil {
		return nil, err
	}
	snap := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		snap[e.Name()] = struct{}{}
	}
	return snap, nil
}

func findNewBootLog(before map[string]struct{}, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(runscLogDir)
		if err != nil {
			return "", err
		}
		for _, e := range entries {
			name := e.Name()
			if _, existed := before[name]; existed {
				continue
			}
			if strings.HasSuffix(name, ".boot.txt") {
				return filepath.Join(runscLogDir, name), nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out waiting for new boot log in %s", runscLogDir)
}

// ---- Strace parser ----

// parseBehaviorReport extracts only security-relevant events from gVisor strace output.
//
// gVisor exit line format:
//
//	I0521 22:49:06.655741   1 strace.go:605] [   1:   1] python X open(...) = 3 (0x3) (10.94µs)
//
// We only process X (exit) lines — they contain the return value and are never duplicated.
func parseBehaviorReport(raw string) BehaviorReport {
	var report BehaviorReport

	fsOps := map[string]bool{
		"open": true, "openat": true, "read": true, "write": true,
		"stat": true, "fstat": true, "lstat": true,
		"rename": true, "unlink": true, "mkdir": true, "rmdir": true,
	}
	netOps := map[string]bool{
		"socket": true, "connect": true, "bind": true, "listen": true,
		"accept": true, "sendto": true, "recvfrom": true, "sendmsg": true,
		"recvmsg": true,
	}
	procOps := map[string]bool{
		"execve": true, "vfork": true, "clone": true, "kill": true,
		"exit_group": true,
	}

	for _, line := range splitLines(raw) {
		// Only exit lines contain the return value; skip enter lines to avoid duplicates
		if !containsStr(line, " X ") {
			continue
		}

		ts := extractTimestamp(line)
		body := extractBody(line)
		//log.Printf("body: %q", body)
		if body == "" {
			continue
		}

		parenIdx := indexByte(body, '(')
		if parenIdx == -1 {
			continue
		}
		syscallName := body[:parenIdx]
		argsSection := body[parenIdx:]

		retVal := ""
		if eqIdx := indexStr(body, " = "); eqIdx != -1 {
			retVal = body[eqIdx+3:]
			// Trim trailing timing annotation e.g. "(10.94µs)"
			if tiIdx := indexByte(retVal, '('); tiIdx != -1 {
				retVal = trimSpace(retVal[:tiIdx])
			}
		}

		switch {
		case fsOps[syscallName]:
			path := extractPath(argsSection)
			if isNoisyFSPath(path) {
				continue
			}
			report.FSAccess = append(report.FSAccess, FSEvent{
				Timestamp: ts,
				Op:        syscallName,
				Path:      path,
				Flags:     extractFlags(argsSection),
				Result:    retVal,
			})

		case netOps[syscallName]:
			report.NetEvents = append(report.NetEvents, NetEvent{
				Timestamp: ts,
				Op:        syscallName,
				Detail:    extractNetDetail(argsSection),
				Result:    retVal,
			})

		case procOps[syscallName]:
			report.ProcEvents = append(report.ProcEvents, ProcEvent{
				Timestamp: ts,
				Op:        syscallName,
				Args:      extractExecArgs(argsSection),
				Result:    retVal,
			})
		}
	}

	return report
}

// isNoisyFSPath filters out Python runtime internals we don't care about.
// isNoisyFSPath filters out Python runtime internals we don't care about.
func isNoisyFSPath(path string) bool {
	if path == "/tmp/script.py" || path == "script.py" {
		return true
	}
	if path == "" || path == "/" || path == "/tmp" || path == "/tmp/" {
		return true
	}

	// Wipes out entire directory hierarchies common to interpreter startups
	if strings.HasPrefix(path, "/usr/local/") ||
		strings.HasPrefix(path, "/usr/lib/") ||
		strings.HasPrefix(path, "/root/.local/") {
		return true
	}

	// Fallback catch-all strings
	noisySubstrings := []string{
		"__pycache__",
		".pyc",
		"pyvenv.cfg",
		"pybuilddir.txt",
		"Setup.local",
		"python311.zip",
		"/etc/localtime",
	}

	for _, sub := range noisySubstrings {
		if strings.Contains(path, sub) {
			return true
		}
	}
	return false
}

// extractTimestamp pulls "HH:MM:SS.ffffff" from a gVisor log line prefix.
// Format: "I0521 22:49:06.655741 ..."
func extractTimestamp(line string) string {
	for i := 0; i < len(line)-6; i++ {
		if line[i] == ' ' && i+9 < len(line) {
			candidate := line[i+1 : i+9]
			if len(candidate) == 8 && candidate[2] == ':' && candidate[5] == ':' {
				end := i + 1
				for end < len(line) && line[end] != ' ' {
					end++
				}
				return line[i+1 : end]
			}
		}
	}
	return ""
}

// extractBody returns the syscall call expression after "[pid:tid] procname X "
func extractBody(line string) string {
	// Find the last "] " which closes the "[tid:pid]" prefix
	// e.g. "...strace.go:605] [   1:   1] python X open(...)"
	//                                               ^-- we want from here
	lastBracket := -1
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == ']' && i+1 < len(line) && line[i+1] == ' ' {
			lastBracket = i
			break
		}
	}
	if lastBracket == -1 {
		return ""
	}
	rest := line[lastBracket+2:] // "python X open(...)"

	// Find " X " and skip past it — but keep the syscall name
	xIdx := indexStr(rest, " X ")
	if xIdx == -1 {
		return ""
	}
	return rest[xIdx+3:] // "open(...)"  ← this is correct, syscall IS here
}

// extractPath finds the first absolute path (starts with /) in a gVisor args string.
// gVisor annotates fd arguments like: "0x7ecc... /etc/passwd" — the path follows the pointer.
func extractPath(args string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == '/' {
			end := i
			for end < len(args) && args[end] != ',' && args[end] != ')' &&
				args[end] != ' ' && args[end] != '"' && args[end] != '}' {
				end++
			}
			return args[i:end]
		}
	}
	return ""
}

// extractFlags finds open(2) flag names in the args string.
func extractFlags(args string) string {
	markers := []string{"O_RDONLY", "O_WRONLY", "O_RDWR", "O_CREAT", "O_EXCL", "O_TRUNC", "O_APPEND"}
	result := ""
	for _, m := range markers {
		if containsStr(args, m) {
			if result != "" {
				result += "|"
			}
			result += m
		}
	}
	return result
}

// extractNetDetail pulls address/port from connect/bind args.
// gVisor formats: "{Family: AF_INET, Addr: 8.8.8.8, Port: 53}"
func extractNetDetail(args string) string {
	addrIdx := indexStr(args, "Addr: ")
	if addrIdx == -1 {
		return ""
	}
	rest := args[addrIdx+6:]
	end := indexByte(rest, ',')
	if end == -1 {
		end = indexByte(rest, '}')
	}
	addr := rest
	if end != -1 {
		addr = trimSpace(rest[:end])
	}

	port := ""
	portIdx := indexStr(args, "Port: ")
	if portIdx != -1 {
		pr := args[portIdx+6:]
		pend := indexByte(pr, '}')
		if pend != -1 {
			port = ":" + trimSpace(pr[:pend])
		}
	}
	return addr + port
}

// extractExecArgs pulls the argv array from execve() args.
func extractExecArgs(args string) string {
	start := indexByte(args, '[')
	if start == -1 {
		return ""
	}
	end := indexByte(args[start:], ']')
	if end == -1 {
		return ""
	}
	return args[start : start+end+1]
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func indexStr(s, sub string) int {
	if len(sub) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func containsStr(s, sub string) bool {
	return indexStr(s, sub) != -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
