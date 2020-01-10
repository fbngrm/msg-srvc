## Notify
This program reads messages from stdin and send them as POST requests to a target URL.

### Setup
This section assumes there is a go, make and git installation available on the system.

### Build
A Makefile is provided which should be used to test, build and run the program.
Builds will be placed in the `/bin` directory.
Binaries use the latest git commit hash or tag as a version.

```bash
make build # build
```

### Run
Read messages from a file and write results to stdout.
```
make build
./bin/notify --url=http://localhost:8080 < messages.txt
```

Read messages from a file and discard results.
```
make build
./bin/notify --url=http://localhost:8080 < messages.txt > /dev/null
```

### Tests
There are several targets available to run tests.

```bash
make test # runs tests
make test-cover # creates a coverage profile
make test-race # tests service for race conditions
```

### Lint
There is a lint target which runs [golangci-lint](https://github.com/golangci/golangci-lint) in a docker container.

```bash
make lint
```

## Architecture
The program consists of three libraries which are used to build a pipeline consisting of three stages.
`scan` reads lines from an io.Reader in a non-blocking manner into a queue.
`schedule` reads from a queue and sends the messages to an outbound channel, one per time interval.
`notify` posts HTTP requests to a target URL.
Requests are sent concurrently, results are returned via a channel.
Note, since the program posts to the same host always, it tries to keep TCP connections open to save handshake time.

In general, stages close their outbound channels when all the send operations are done.
Stages keep receiving values from inbound channels until those channels are closed or the senders are unblocked.

> Note, the queue can potentially grow until the machine runs out of memory.

### Termination
The program terminates gracefully always.
In other words, it waits until all requests have returned and have been logged before it shuts down.
If a interrupt signal is caught, all stages of the pipeline are stopped and requests are canceled via their context.
If no interrupt is send, it terminates after reading an EOF and all requests have returned.
It is possible that a request timeout occurs, which leads to a canceled request.
Request errors are logged.

Note, for sending a EOF manually, on UNIX systems Ctrl+D is used.

### Logs
The program logs to stderr in a structured and human readable format.
Stages include the cause termination in the log message, where SIGTERM means a cancellation by interrupt and EOF|FIN means no messages left to process.
Results of POST requests are logged to stdout in machine readable format (JSON).

### Configuration
```bash
  -c int
        max number of concurrent POST requests (default 100)
  -i duration
        notification interval in milliseconds (default 10ms)
  -t duration
        request timeout in milliseconds (default 500ms)
  -url string
        target URL
  -v    print version
```

### Dependency management
For handling dependencies, go modules are used.
This requires to have a go version > 1.11 installed and setting `GO111MODULE=1`.
If the go version is >= 1.13, modules are enabled by default.

