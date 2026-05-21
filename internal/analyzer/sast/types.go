package sast

type SemgrepOutput struct {
	Results []SemgrepResult `json:"results"`
}

type SemgrepResult struct {
	CheckID string    `json:"check_id"`
	Path    string    `json:"path"`
	Start   Position  `json:"start"`
	End     Position  `json:"end"`
	Extra   ExtraData `json:"extra"`
}

type Position struct {
	Line int `json:"line"`
	Col  int `json:"col"`
}

type ExtraData struct {
	Message  string `json:"message"`
	Lines    string `json:"lines"` // This is the actual code snippet Semgrep found
	Severity string `json:"severity"`
}
