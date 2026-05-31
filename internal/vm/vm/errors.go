package vm

// UserError represents a failure caused by user-provided code or configuration.
type UserError struct {
	reason string
}

func (e *UserError) Error() string {
	return e.reason
}
