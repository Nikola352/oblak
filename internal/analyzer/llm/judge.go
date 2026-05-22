package llm

import "context"

type JudgeLLM interface {
	AskForSAST(sastMessage string, codeContext string) (SastVerdict, error)
	AskForLogs(ctx context.Context, logPath string) (*JudgeVerdict, error)
}
