package notify_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/fgrimme/refurbed/notify"
)

const timeout = 100 * time.Millisecond // request timeout

// postClient is a mock client to control the
// PostResults received by the Scheduler.
type postClient struct{}

// we use the msg parameter to get the return value from the test cases.
func (pc *postClient) Post(ctx context.Context, msg []byte) notify.PostResult {
	tc := schedulerTests[string(msg)]
	if tc.t { // test timeout
		time.Sleep(timeout + 10*time.Millisecond)
	}
	// test context cancelation due to timeout or call to cancelFunc
	if err := ctx.Err(); err != nil {
		return notify.PostResult{
			Body: tc.r.Body,
			Err:  err,
		}
	}
	return notify.PostResult{
		Body: tc.r.Body,
	}
}

var schedulerTests = map[string]struct {
	d string            // description of test case
	r notify.PostResult // expected result, send by the mock PostClient
	t bool              // exceed context deadline
}{
	"timeout": {
		d: "expect context to exceed deadline",
		r: notify.PostResult{
			Body: []byte("timeout"), // test id
			Err:  context.DeadlineExceeded,
		},
		t: true,
	},
	"succ1": {
		d: "expect success",
		r: notify.PostResult{
			Body: []byte("succ1"), // test id
		},
	},
	"succ2": {
		d: "expect success",
		r: notify.PostResult{
			Body: []byte("succ2"), // test id
		},
	},
	"succ3": {
		d: "expect success",
		r: notify.PostResult{
			Body: []byte("succ3"), // test id
		},
	},
}

func TestRun(t *testing.T) {
	client := &postClient{}
	timeout := 100 * time.Millisecond
	concurrency := 2

	s, err := notify.NewScheduler(client, timeout, concurrency)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// we use the context to signal requests to return
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	queue := make(chan []byte, 10)
	out := s.Run(ctx, queue)

	// send the test messages to the queue
	for _, tc := range schedulerTests {
		queue <- tc.r.Body
	}
	// note, not closing the queue will result in an inifite loop
	close(queue)

	for res := range out {
		// we use the body as the id of the test case
		id := string(res.Body)
		tt := schedulerTests[id]
		t.Run(tt.d, func(t *testing.T) {
			// unexpected errors
			if res.Err != nil && tt.r.Err == nil {
				t.Fatalf("unexpected err: %v", res.Err)
			}
			// expected errors
			if res.Err == nil && tt.r.Err != nil {
				t.Fatalf("expected err: %v", tt.r.Err)
			}
			if res.Err != nil && tt.r.Err != nil {
				if want, got := tt.r.Err.Error(), res.Err.Error(); want != got {
					t.Errorf("want err\n%+v\ngot\n%+v", want, got)
				}
				return
			}
			// expected response
			if want, got := tt.r.Body, res.Body; bytes.Compare(want, got) != 0 {
				t.Errorf("want body\n%+v\ngot\n%+v", want, got)
			}
		})
	}
}
