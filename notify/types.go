package notify

import (
	"fmt"
	"net/http"
)

type PostErr struct {
	Err      string
	Response *http.Response
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
	Body []byte
	Err  error
}
