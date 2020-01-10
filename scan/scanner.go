package scan

import (
	"bufio"
	"io"

	"github.com/rs/zerolog"
)

// Scanner reads lines from an io.Reader.
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

// Run reades from the Scanners io.Reader until it reaches EOF or a quit signal.
// Note: We assume a line can fit into the scanner's buffer/token-size (64*1024B).
func (s *Scanner) Run() (*Queue, chan error) {
	s.logger.Info().Msg("start scanner")
	scanner := bufio.NewScanner(s.in)
	errC := make(chan error)
	go func() {
		defer func() { errC <- scanner.Err() }()
		for {
			select {
			case <-s.quit:
				s.queue.setReady()
				s.logger.Info().Str("term", "SIGTERM").Msg("stop scanner")
				return
			default:
				if scanner.Scan() {
					msg := scanner.Text()
					if len(msg) == 0 {
						continue
					}
					s.queue.Push(msg)
				} else {
					s.queue.setReady()
					s.logger.Info().Str("term", "EOF").Msg("stop scanner")
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
