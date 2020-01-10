package scan

import (
	"bufio"
	"io"

	"github.com/rs/zerolog"
)

type Scanner struct {
	in     io.Reader
	queue  *Queue
	logger zerolog.Logger
}

func NewScanner(in io.Reader, logger zerolog.Logger) *Scanner {
	return &Scanner{
		in:     in,
		queue:  NewQueue(),
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
	errC := make(chan error, 1)
	go func() {
		for scanner.Scan() {
			msg := scanner.Text()
			if len(msg) == 0 {
				continue
			}
			s.queue.Push(msg)
		}
		s.queue.setDone()
		errC <- scanner.Err()
		s.logger.Info().Str("status", "EOF").Msg("stop scanner")
	}()
	return s.queue, errC
}
