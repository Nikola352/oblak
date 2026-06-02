package apperr

import "net/http"

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrNotFound     = &Error{Status: http.StatusNotFound, Message: "not found"}
	ErrUnauthorized = &Error{Status: http.StatusUnauthorized, Message: "unauthorized"}
)
