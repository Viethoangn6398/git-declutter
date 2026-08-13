package exitcode

import "errors"

const (
	OK             = 0
	Execution      = 1
	InvalidArgs    = 2
	NotRepository  = 3
	ProviderAuth   = 4
	PartialCleanup = 5
)

type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(code int, err error) *Error {
	return &Error{Code: code, Err: err}
}

func Code(err error) int {
	if err == nil {
		return OK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return Execution
}
