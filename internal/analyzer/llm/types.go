package llm

type SastVerdict int

// 2. Create constants using iota (which auto-increments)
const (
	FAILURE   SastVerdict = iota // 0
	MALICIOUS                    // 1
	SAFE                         // 2
)
