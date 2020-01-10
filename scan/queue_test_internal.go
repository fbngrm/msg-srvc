package scan

import "testing"

var queueTests = []struct {
	d    string // test case description
	push string // element to push
	pop  string // element to pop
	s    bool   // set done
}{
	{
		d:    "expect foo to get pushed to the queue",
		push: "foo",
		s:    false,
	},
	{
		d:    "expect bar to get pushed to the queue",
		push: "bar",
		s:    true, // set done
	},
	{
		d:   "expect foo to get popped from the queue",
		pop: "foo",
		s:   false,
	},
	{
		d:   "expect bar to get popped from the queue",
		pop: "bar",
		s:   false,
	},
}

func TestQueue(t *testing.T) {
	q := NewQueue()
	for _, tc := range queueTests {
		tt := tc
		t.Run(tt.d, func(t *testing.T) {
			if tt.push != "" {
				q.Push(tc.push)
			}
			if tt.pop != "" {
				s := q.Pop()
				if want, got := tt.pop, s; want != got {
					t.Errorf("want pop %s got %s", want, got)
				}
			}
			if tt.s {
				q.setReady()
			}
		})
	}
	if !q.IsExhausted() {
		t.Error("expect queue to be done")
	}
}
