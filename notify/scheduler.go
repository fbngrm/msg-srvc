package notify

import (
	"context"
	"errors"
	"time"
)

type PostClient interface {
	Post(ctx context.Context, msg []byte) PostResult
}

// Scheduler reads from an input queue and post messages to a PostClient.
// Post calls run in parallel, limited by the Schedulers concurrency setting.
type Scheduler struct {
	client      PostClient
	timeout     time.Duration
	concurrency int // must be greater than 0
}

// NewScheduler returns a reference to a Scheduler.
func NewScheduler(c PostClient, timeout time.Duration, concurrency int) (*Scheduler, error) {
	if concurrency < 1 {
		return nil, errors.New("concurrency must be > 0")
	}
	return &Scheduler{
		client:      c,
		concurrency: concurrency,
		timeout:     timeout,
	}, nil
}

// Run starts the event loop of the Scheduler.
// It reads messages from the provided inbound channel until it gets closed. The
// retrieved messages are posted to the Scheduler's PostClient. Results of the post
// calls get send to the outbound channel.
// When the inbound channel is closed, the function stops posting and waits until
// all post requests have returned before closing the outbound channel.
// Post calls can be canceled by the proviced Context. A derived Context is
// used to set a deadline to the post calls.
func (s *Scheduler) Run(ctx context.Context, queue chan []byte) chan PostResult {
	limit := make(chan struct{}, s.concurrency)
	out := make(chan PostResult)

	go func() {
		for msg := range queue {
			if msg == nil {
				continue
			}

			// limit concurrency
			limit <- struct{}{}

			// we explicitly pass the args here to avoid shadowing
			go func(ctx context.Context, msg []byte) {
				ctx, _ = context.WithTimeout(ctx, s.timeout)
				out <- s.client.Post(ctx, msg)
				<-limit
			}(ctx, msg)
		}

		// wait until all requests have returned
		// before closing the output channel
		for i := 0; i < s.concurrency; i++ {
			limit <- struct{}{}
		}
		close(out)
	}()

	return out
}
