package notify

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// we test errors first, then success.
var postTests = map[string]struct {
	d string        // description of test case
	t time.Duration // context timeout
	s int           // http status code of mock response
	u string        // url of mock server with the test case id as last segment
	r PostResult    // expected result
}{
	// errors
	"err1": { // 503 - EOF
		d: "expect the connection to get closed by the server unexpectedly",
		t: 10000 * time.Millisecond,
		s: http.StatusServiceUnavailable,
		r: PostResult{
			Err: errors.New("EOF"),
		},
	},
	"err2": { // 404 - not found
		d: "expect an error when 404 is returned",
		t: 10000 * time.Millisecond,
		s: http.StatusNotFound,
		r: PostResult{
			Body: []byte("not found"),
			Err:  errors.New("404: not found"),
		},
	},
	"err3": { // 504 - timeout
		d: "expect the context to be cancelled due to timeout",
		t: 200 * time.Millisecond,
		s: http.StatusGatewayTimeout,
		r: PostResult{
			Err: errors.New("context deadline exceeded"),
		},
	},
	"err4": { // malformed backend url
		d: "expect error due to wrong url",
		t: 10000 * time.Millisecond,
		r: PostResult{
			Err: errors.New("missing protocol scheme"),
		},
		u: "://127.0.0.1:8080",
	},
	// success
	"succ1": { // 200
		d: "expect success",
		t: 10000 * time.Millisecond,
		r: PostResult{
			Body: []byte("success"),
		},
		s: http.StatusOK,
	},
}

func TestPost(t *testing.T) {
	// mock target backend to receive POST requests
	targetSrvc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		segments := strings.Split(r.URL.Path, "/")
		if len(segments) != 2 {
			t.Fatalf("expect 2 path segments but got %d", len(segments))
		}
		// we use the last segment of the URL to get the test case id
		id := segments[1]
		// test case
		tt, ok := postTests[id]
		if !ok {
			t.Fatalf("missing test case for id: %s", id)
		}
		// we base the response on the expected
		// http status code of the test case
		switch tt.s {
		case http.StatusNotFound:
			w.WriteHeader(tt.s)
			w.Header().Set("Content-Type", "text/plain")
			w.Write(tt.r.Body)
		case http.StatusGatewayTimeout:
			time.Sleep(tt.t + 10*time.Millisecond)
		case http.StatusServiceUnavailable:
			c, _, _ := w.(http.Hijacker).Hijack()
			c.Close() // connection unexpectedly closed
		case http.StatusOK:
			w.WriteHeader(tt.s)
			w.Header().Set("Content-Type", "text/plain")
			w.Write(tt.r.Body)
		default:
			t.Fatalf("unsupported http status code in test case: %s", id)
		}
	}))
	defer targetSrvc.Close()

	// service to test with a HTTP test client
	ns := NotificationService{
		client: targetSrvc.Client(),
	}

	for id, tc := range postTests {
		tt := tc
		t.Run(tt.d, func(t *testing.T) {
			// we need to set the target URL, either the correct one of the
			// mock backend or we get it from the test case when testing errors
			targetURL := targetSrvc.URL
			if tt.u != "" {
				targetURL = tt.u
			}
			// append the test case id to lookup the test case in the mock backend
			targetURL = strings.TrimRight(targetURL, "/")
			ns.targetURL = fmt.Sprintf("%s/%s", targetURL, id)

			ctx, cancel := context.WithTimeout(context.Background(), tt.t)
			defer cancel()
			res := ns.Post(ctx, tt.r.Body)

			// unexpected errors
			if res.Err != nil && tt.r.Err == nil {
				t.Fatalf("unexpected err: %v", res.Err)
			}
			// expected errors
			if res.Err == nil && tt.r.Err != nil {
				t.Fatalf("expected err: %v", tt.r.Err)
			}
			if res.Err != nil && tt.r.Err != nil {
				// we need to match the error message against a regex since it
				// contains the url of the mock backend with a dynamically
				// assigned port for transport layer errors
				want, got := tt.r.Err.Error(), res.Err.Error()
				urlRegex := fmt.Sprintf(`(http:\/\/127.0.0.1:-?[0-9]*\/[.]+: )?%s`, want)
				match, err := regexp.MatchString(urlRegex, got)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !match {
					t.Errorf("want error\n%+v\ngot\n%+v", urlRegex, got)
				}
			} else {
				// expected response
				if want, got := tt.r.Body, res.Body; bytes.Compare(want, got) != 0 {
					t.Errorf("want body\n%+v\ngot\n%+v", want, got)
				}
			}
		})
	}
}
