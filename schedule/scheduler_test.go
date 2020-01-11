package schedule_test

import (
	"io/ioutil"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/fgrimme/refurbed/scan"
	"github.com/fgrimme/refurbed/schedule"
	"github.com/rs/zerolog"
	"go.uber.org/goleak"
)

var schedulerTests = []string{
	"foo 1",
	"foo 2",
	"foo 3",
	"bar 1",
	"bar 2",
	"bar 3",
}

func TestRun(t *testing.T) {
	in := `
foo 1
foo 2
foo 3

bar 1
bar 2
bar 3
`
	// mute logger in tests
	l := zerolog.New(ioutil.Discard)
	log.SetOutput(l)

	r := strings.NewReader(in)
	s := scan.NewScanner(r, l)
	q, errc := s.Run()
	if err := <-errc; err != nil {
		t.Errorf("unexpected err: %v\n", err)
	}
	s.Stop()

	sc := schedule.NewScheduler(10*time.Millisecond, l)
	out := sc.Run(q)
	for _, tc := range schedulerTests {
		if want, got := tc, <-out; want != got {
			t.Errorf("expected: %s got: %s\n", want, got)
		}
	}
	sc.Stop()
}

// we test for leaking go routines
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
