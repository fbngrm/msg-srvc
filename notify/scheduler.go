package notify

import (
	"container/list"
	"context"
	"errors"
	"time"
)

type PostClient interface {
	post(ctx context.Context, msg []byte) PostResult
}

type Scheduler struct {
	client      PostClient
	queue       *list.List
	tick        time.Duration
	timeout     time.Duration
	concurrency int
	done        chan struct{}
}

func NewScheduler(c PostClient, queue *list.List, timeout, tick time.Duration, concurrency int) (*Scheduler, error) {
	if concurrency < 1 {
		return nil, errors.New("concurrency must be > 0")
	}
	return &Scheduler{
		client:      c,
		queue:       queue,
		tick:        tick,
		concurrency: concurrency,
		done:        make(chan struct{}, 1),
	}, nil
}

// Run starts the event loop or the Scheduler, listening for incoming messages
// to be posted or a shutdown signal.
func (s *Scheduler) Run() chan PostResult {
	prCh := make(chan PostResult)
	limit := make(chan struct{}, s.concurrency)
	ctx, cancelFunc := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-s.done:
				// cancel requests
				cancelFunc()
				return
			default:
				e := s.queue.Front()
				if e == nil { // we might operate on a temporarily empty queue
					continue
				}
				// we need to assert the correct message type
				msg, ok := e.Value.([]byte)
				if !ok {
					// throttled logging err
					continue
				}
				// we can post a request
				limit <- struct{}{} // we limit the concurrency here
				ctx, _ := context.WithTimeout(ctx, s.timeout)
				// as a matter of good practice, we explicitly pass the args
				// here to avoid shadowing, even if is not required in this case
				go func(ctx context.Context, msg []byte) {
					prCh <- s.client.post(ctx, msg)
					<-limit
				}(ctx, msg)
				s.queue.Remove(e)
			}
		}
	}()
	return prCh
}

func (s *Scheduler) Stop() {
	s.done <- struct{}{}
}
