package llm

type Verdict int

// 2. Create constants using iota (which auto-increments)
const (
	FAILURE   Verdict = iota // 0
	MALICIOUS                // 1
	SAFE                     // 2
)
