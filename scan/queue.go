package scan

import (
	"container/list"
	"sync"
)

type Queue struct {
	sync.RWMutex
	list *list.List
	done bool
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
	q.Lock()
	e := q.list.Front()
	var v string
	if e != nil {
		v = e.Value.(string)
		q.list.Remove(e)
	}
	q.Unlock()
	return v
}

func (q *Queue) IsDone() bool {
	q.Lock()
	l := q.list.Len()
	q.Unlock()
	return q.done && l == 0
}

func (q *Queue) setDone() {
	q.Lock()
	q.done = true
	q.Unlock()
}
