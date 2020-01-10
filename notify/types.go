package notify

import (
	"fmt"
	"net/http"
)

type PostErr struct {
	Err      string         `json:"error"`
	Response *http.Response `json:"-"` // Will not be marshalled
}

func (e PostErr) Error() string {
	if e.Response == nil {
		return e.Err
	}
	return fmt.Sprintf("%v %v: %v",
		e.Response.Request.URL,
		e.Response.StatusCode,
		e.Err)
}

// PostResult wraps the result and error of a Post request.
type PostResult struct {
	Msg  string `json:"message"`
	Body string `json:"response_body"`
	Err  error  `json:"error"`
}
