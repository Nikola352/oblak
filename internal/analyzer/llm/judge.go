package llm

type JudgeLLM interface {
	AskForSAST(sastMessage string, codeContext string) (SastVerdict, error)
}
