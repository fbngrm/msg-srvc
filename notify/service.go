package notify

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
)

type PostClient interface {
	Post(ctx context.Context, msg string) PostResult
}

// Service reads from an input queue and post messages to a PostClient.
// Post calls run in parallel, limited by the Schedulers concurrency setting.
type Service struct {
	client      PostClient
	timeout     time.Duration
	concurrency int // must be greater than 0
	logger      zerolog.Logger
}

// NewService returns a reference to a Service.
func NewService(c PostClient, timeout time.Duration, concurrency int, logger zerolog.Logger) (*Service, error) {
	if concurrency < 1 {
		return nil, errors.New("concurrency must be > 0")
	}
	return &Service{
		client:      c,
		concurrency: concurrency,
		timeout:     timeout,
		logger:      logger,
	}, nil
}

// Run starts the event loop of the Service.
// It reads messages from the provided inbound channel until it gets closed. The
// retrieved messages are posted to the Service's PostClient. Results of the post
// calls get send to the outbound channel.
// When the inbound channel is closed, the function stops posting and waits until
// all post requests have returned before closing the outbound channel.
// Post calls can be canceled by the provided Context. A derived Context is used
// to set a deadline to the post calls.
func (s *Service) Run(ctx context.Context, queue chan string) chan PostResult {
	limit := make(chan struct{}, s.concurrency)
	out := make(chan PostResult)

	s.logger.Info().Msg("start notification service")
	go func() {
		for msg := range queue {
			if msg == "" {
				continue
			}

			// limit concurrency
			limit <- struct{}{}

			// we explicitly pass the args here to avoid shadowing
			go func(ctx context.Context, msg string) {
				ctx, cancel := context.WithTimeout(ctx, s.timeout)
				out <- s.client.Post(ctx, msg)
				<-limit
				cancel()
			}(ctx, msg)
		}

		s.logger.Info().Str("term", "FIN").Msg("stop notification service")

		// wait until all requests have returned
		// before closing the outbound channel
		for i := 0; i < s.concurrency; i++ {
			limit <- struct{}{}
		}
		close(out)
		close(limit)
	}()

	return out
}
