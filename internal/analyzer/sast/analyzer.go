package sast

type StaticAnalyzer interface {
	Run(targetPath string) (*SemgrepOutput, error)
}
