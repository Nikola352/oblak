package orchestrator

type AnalysisVerdict int

// 2. Create constants using iota (which auto-increments)
const (
	FAILURE   AnalysisVerdict = iota // 0
	MALICIOUS                        // 1
	SAFE                             // 2
)

// Result mimics the C# Result<T> pattern using Go generics.
type Result[T any] struct {
	value interface{} // Can hold T or nil safely
	err   error
}

// Success creates a successful Result wrapping the value.
func Success[T any](val T) Result[T] {
	return Result[T]{value: val, err: nil}
}

// Failure creates a failed Result wrapping an error.
func Failure[T any](err error) Result[T] {
	return Result[T]{value: nil, err: err}
}

// IsSuccess returns true if the action executed without errors.
func (r Result[T]) IsSuccess() bool {
	return r.err == nil
}

// IsFailure returns true if the action encountered a failure condition.
func (r Result[T]) IsFailure() bool {
	return r.err != nil
}

// Error returns the underlying error object, or nil.
func (r Result[T]) Error() error {
	return r.err
}

// Value returns the wrapped value.
// Warning: Will panic if called on a Failure result, matching C# behavior.
func (r Result[T]) Value() T {
	if r.err != nil {
		panic("cannot retrieve value from a failed Result: " + r.err.Error())
	}
	return r.value.(T)
}

// ValueOrDefault returns the value if successful, or a zero-value fallback if failed.
func (r Result[T]) ValueOrDefault() T {
	if r.err != nil {
		var zero T
		return zero
	}
	return r.value.(T)
}
