package schedule

import (
	"time"

	"github.com/rs/zerolog"
)

type queue interface {
	IsExhausted() bool
	Pop() string
}

// Scheduler schedules send operations to a queue.
type Scheduler struct {
	interval time.Duration
	quit     chan struct{}
	logger   zerolog.Logger
}

func NewScheduler(interval time.Duration, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		interval: interval,
		quit:     make(chan struct{}, 2),
		logger:   logger,
	}
}

// Run reads from q and sends to an outbound channel once per interval until the
// queue is exhausted or a quit signal is received. It closes the outbound channel
// when the read loop terminates.
func (s *Scheduler) Run(q queue) chan string {
	ticker := time.NewTicker(s.interval)
	out := make(chan string)
	s.logger.Info().Msg("start scheduler")
	go func() {
		defer close(out)
		for {
			select {
			case <-s.quit:
				ticker.Stop() // does not close the tick channel
				s.logger.Info().Str("term", "SIGTERM").Msg("stop scheduler")
				return
			case <-ticker.C:
				if q.IsExhausted() {
					s.logger.Info().Str("term", "FIN").Msg("stop scheduler")
					return
				}
				msg := q.Pop()
				if len(msg) > 0 {
					out <- msg
				}
			}
		}
	}()
	return out
}

func (s *Scheduler) Stop() {
	s.quit <- struct{}{}
	defer close(s.quit)
}
