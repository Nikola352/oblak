package vm

import "time"

const (
	CpuCountDefault = 1
	MemoryMbDefault = 512

	CpuCountHardLimit = 6
	MemoryMbHardLimit = 10 * 1024
)

// ResourceRequirements describes the CPU, memory, and time budget for a single VM operation.
type ResourceRequirements struct {
	CpuCount           int64
	MemoryMb           int64
	ExecutionTimeLimit time.Duration
}

// ResourcePolicy defines the default and maximum execution time for a class of VM operation.
type ResourcePolicy struct {
	defaultExecTime   time.Duration
	execTimeHardLimit time.Duration
}

// Defaults returns a ResourceRequirements populated with safe defaults for this policy.
func (p ResourcePolicy) Defaults() ResourceRequirements {
	return ResourceRequirements{
		CpuCount:           CpuCountDefault,
		MemoryMb:           MemoryMbDefault,
		ExecutionTimeLimit: p.defaultExecTime,
	}
}

// ApplyHardLimits clamps r's fields to the maximum values allowed by this policy.
func (p ResourcePolicy) ApplyHardLimits(r *ResourceRequirements) {
	if r.CpuCount > CpuCountHardLimit {
		r.CpuCount = CpuCountHardLimit
	}
	if r.MemoryMb > MemoryMbHardLimit {
		r.MemoryMb = MemoryMbHardLimit
	}
	if r.ExecutionTimeLimit > p.execTimeHardLimit {
		r.ExecutionTimeLimit = p.execTimeHardLimit
	}
}

var ExecutePolicy = ResourcePolicy{
	defaultExecTime:   5 * time.Minute,
	execTimeHardLimit: 15 * time.Minute,
}

var PreparePolicy = ResourcePolicy{
	defaultExecTime:   30 * time.Minute,
	execTimeHardLimit: 60 * time.Minute,
}
