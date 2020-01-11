package scan

import (
	"container/list"
	"sync"
)

// Queue is a FIFO list, safe for concurrent access.
type Queue struct {
	sync.RWMutex
	list  *list.List
	ready bool
}

func NewQueue() *Queue {
	return &Queue{
		list: list.New(),
	}
}

func (q *Queue) Push(s string) {
	q.Lock()
	q.list.PushBack(s)
	q.Unlock()
}

func (q *Queue) Pop() string {
	var v string
	q.Lock()
	e := q.list.Front()
	if e != nil {
		v = e.Value.(string)
		q.list.Remove(e)
	}
	q.Unlock()
	return v
}

// IsExhausted determines if all elements of the queue
// have been consumed and no future pushes are intended.
func (q *Queue) IsExhausted() bool {
	q.Lock()
	l := q.list.Len()
	r := q.ready
	q.Unlock()
	return r && l == 0
}

// setReady indicates that no future writes are intended.
func (q *Queue) setReady() {
	q.Lock()
	q.ready = true
	q.Unlock()
}
