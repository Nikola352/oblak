package invocations

import (
	"bufio"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

type LogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"`
	Message   string    `json:"message"`
}

type InvocationLogStore struct {
	client     *minio.Client
	bucketName string
}

func NewInvocationLogStore(client *minio.Client, bucketName string) *InvocationLogStore {
	return &InvocationLogStore{
		client:     client,
		bucketName: bucketName,
	}
}

var logRegex = regexp.MustCompile(`channel:([a-zA-Z0-9_-]+)\s+data:(.*)`)

func (s *InvocationLogStore) GetLogsByInvocationId(ctx context.Context, logPath string) ([]LogLine, error) {
	object, err := s.client.GetObject(ctx, s.bucketName, logPath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to reach minio object: %w", err)
	}
	defer object.Close()

	var logs []LogLine
	scanner := bufio.NewScanner(object)

	var currentChannel = "stdout" // Fallback default

	for scanner.Scan() {
		lineText := strings.TrimSpace(scanner.Text())
		if lineText == "" {
			continue
		}

		if lineText == "}" {
			continue // Safely drop the orphan brace, the payload message is already captured
		}

		if strings.Contains(lineText, "type:done") {
			logs = append(logs, LogLine{
				Timestamp: time.Now().UTC(),
				Stream:    "system",
				Message:   "Execution finished successfully.",
			})
			continue
		}

		matches := logRegex.FindStringSubmatch(lineText)
		if len(matches) >= 3 {
			channel := matches[1] // e.g., "system", "stdout", "stderr"
			message := matches[2] // The raw message block text

			message = strings.TrimSuffix(message, "}")

			streamTarget := "stdout"
			if channel == "stderr" || channel == "system" {
				streamTarget = channel
			}

			logs = append(logs, LogLine{
				Timestamp: time.Now().UTC(),
				Stream:    streamTarget,
				Message:   strings.TrimSpace(message),
			})
		} else {
			logs = append(logs, LogLine{
				Timestamp: time.Now().UTC(),
				Stream:    currentChannel,
				Message:   lineText,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed during log stream scanning: %w", err)
	}

	return logs, nil
}
