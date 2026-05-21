package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type QwenJudge struct {
	baseUrl string
}

func NewQwenJudge(bUrl string) *QwenJudge {
	return &QwenJudge{baseUrl: bUrl}
}

func (j *QwenJudge) Ask(sastMessage string, codeContext string) (string, error) {
	url := j.baseUrl + "/api/generate"
	if j.baseUrl == "" {
		url = "http://localhost:11434/api/generate"
	}
	prompt := "Rule: %s\nCode: %s\nVerdict (MALICIOUS/SAFE):"
	payload := map[string]interface{}{
		"model":  "qwen2.5-coder:7b",
		"prompt": fmt.Sprintf(prompt, sastMessage, codeContext),
		"stream": false,
	}

	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res struct {
		Response string `json:"response"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	return res.Response, nil
}
