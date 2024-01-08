package pool

import (
	"errors"
	"time"
)

var (
	// errQueueIsFull will be returned when the worker queue is full.
	errQueueIsFull = errors.New("the queue is full")

	// errQueueIsReleased will be returned when trying to insert item to a released worker queue.
	errQueueIsReleased = errors.New("the queue length is zero")
)

type worker interface {
	run()
	finish()
	lastUsedTime() time.Time
	inputFunc(func())
	inputParam(interface{})
}

type workerQueue interface {
	len() int
	isEmpty() bool
	insert(worker) error
	detach() worker
	refresh(duration time.Duration) []worker // clean up the stale workers and return them
	reset()
}

type queueType int

const (
	queueTypeLoopQueue queueType = 1 << iota
)

func newWorkerQueue(qType queueType, size int) workerQueue {
	switch qType {
	case queueTypeLoopQueue:
		return newWorkerLoopQueue(size)
	default:
		return newWorkerLoopQueue(size)
	}
}
