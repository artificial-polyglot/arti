package logger

import (
	"strings"
)

type Status struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
	Err     string `json:"error,omitempty"`
	Request string `json:"request"`
	Trace   string `json:"trace,omitempty"`
}

// Status implements the Error interface
func (e *Status) Error() string {
	return e.String()
}

// Status implements the Stringer interface
// Using fmt package here caused stack overflow
func (e *Status) String() string {
	var result = make([]string, 0)
	result = append(result, ` "message": "`+e.Message+`"`)
	result = append(result, ` "error": "`+e.Err+`" }`)
	return strings.Join(result, ",")
}
