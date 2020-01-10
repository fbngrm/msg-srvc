package scan

import (
	"bufio"
	"io"

	"github.com/rs/zerolog"
)

type Scanner struct {
	in     io.Reader
	queue  *Queue
	quit   chan struct{}
	logger zerolog.Logger
}

func NewScanner(in io.Reader, logger zerolog.Logger) *Scanner {
	return &Scanner{
		in:     in,
		queue:  NewQueue(),
		quit:   make(chan struct{}, 2),
		logger: logger,
	}
}

// Create a scanner to read each map entry line by line
// Note: We assume the line entry can fit into the scanner's buffer
// Non-blocking until buffer size is exceeded.
// default scan token size of the Scanner is 64*1024 bytes
func (s *Scanner) Run() (*Queue, chan error) {
	s.logger.Info().Msg("start scanner")
	scanner := bufio.NewScanner(s.in)
	errC := make(chan error)
	go func() {
		defer func() { errC <- scanner.Err() }()
		for {
			select {
			case <-s.quit:
				s.queue.setDone()
				s.logger.Info().Str("status", "SIGTERM").Msg("stop scanner")
				return
			default:
				if scanner.Scan() {
					msg := scanner.Text()
					if len(msg) == 0 {
						continue
					}
					s.queue.Push(msg)
				} else {
					s.queue.setDone()
					s.logger.Info().Str("status", "EOF").Msg("stop scanner")
					return
				}
			}
		}
	}()
	return s.queue, errC
}

func (s *Scanner) Stop() {
	s.quit <- struct{}{}
	close(s.quit)
}
