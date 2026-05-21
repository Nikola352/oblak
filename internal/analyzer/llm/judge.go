package llm

type JudgeLLM interface {
	Ask(sastMessage string, codeContext string) (Verdict, error)
}
