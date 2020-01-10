package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fgrimme/refurbed/notify"
	"github.com/fgrimme/refurbed/scan"
	"github.com/fgrimme/refurbed/schedule"
	"github.com/rs/zerolog"
)

var (
	version = "unknown" // will be compiled into the binary
	service = "notify"

	targetURL    string
	concurrency  int
	interval     time.Duration // milliseconds
	timeout      time.Duration // milliseconds
	printVersion bool
)

func main() {
	flag.StringVar(&targetURL, "url", "", "target URL")
	flag.IntVar(&concurrency, "c", 100, "max number of concurrent POST requests")
	flag.DurationVar(&interval, "i", time.Duration(10*time.Millisecond), "notification interval in milliseconds")
	flag.DurationVar(&timeout, "t", time.Duration(500*time.Millisecond), "request timeout in milliseconds")
	flag.BoolVar(&printVersion, "v", false, "print version")
	flag.Parse()

	if printVersion {
		fmt.Println(version)
		os.Exit(0)
	}
	if len(targetURL) == 0 {
		fmt.Println("no target URL specified")
		os.Exit(1)
	}

	// we use the default log level debug and write to stderr.
	// note, we log in (inefficient) human friendly format to console here since it
	// is a coding challenge. In a production environment we would prefer structured,
	// machine parsable format so we could make use of automated log analysis
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	logger = logger.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	// replace standard log
	log.SetFlags(0)
	log.SetOutput(logger)
	logger = logger.With().
		Interface("service", service).
		Interface("version", version).
		Logger()

	// Post messages using the provided PostClient.
	client := notify.NewHttpClient(targetURL)
	notifyService, err := notify.NewService(client, timeout, concurrency, logger)
	if err != nil {
		logger.Error().Err(err)
		os.Exit(1)
	}

	// send one message per interval
	scheduler := schedule.NewScheduler(interval, logger)

	// the scanner reads from stdin until it reaches EOF or it's Stop method is called.
	// note, this may consume a large amount of memory which can lead to a crash of the application
	scanner := scan.NewScanner(os.Stdin, logger)
	queue, errC := scanner.Run()
	defer close(errC)

	// context is used to cancel post requests
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// we catch interrupts to handle termination gracefully
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGINT)

		<-quit
		// stop reading from stdin
		scanner.Stop()
		// stop send messages to the notification service
		scheduler.Stop()
		// cancel POST requests
		cancel()
	}()

	// wait until all requests have returned, also in case of SIGINT
	// this way we ensure to shutdown gracefully always
	resCh := notifyService.Run(ctx, scheduler.Run(queue))
	for res := range resCh {
		// results will be logged to stdout
		if err := json.NewEncoder(os.Stdout).Encode(res); err != nil {
			logger.Error().Err(err)
			continue
		}
	}

	// check for scanner error
	if err := <-errC; err == nil {
		logger.Error().Err(err)
		// note, Exit does not run deferred functions
		// so we need to cancel the context
		cancel()
		close(errC)
		os.Exit(1)
	}
}
