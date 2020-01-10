package scan_test

import (
	"io/ioutil"
	"log"
	"strings"
	"testing"

	"github.com/fgrimme/refurbed/scan"
	"github.com/rs/zerolog"
	"go.uber.org/goleak"
)

var scanTests = []string{
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
	for _, tc := range scanTests {
		if want, got := tc, q.Pop(); want != got {
			t.Errorf("unexpected: %s got: %s\n", want, got)
		}
	}
	s.Stop()
	if !q.IsDone() {
		t.Error("expect queue to be done")
	}
}

// we test for leaking go routines
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
