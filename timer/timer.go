// timer/timer.go
package timer

import (
	"container/heap"
	"sync"
	"time"
)

type TimerTask struct {
	Id       int64
	Execute  time.Time
	Interval time.Duration
	Callback func()
	index    int
}

type TimerQueue []*TimerTask

func (q TimerQueue) Len() int { return len(q) }

func (q TimerQueue) Less(i, j int) bool {
	return q[i].Execute.Before(q[j].Execute)
}

func (q TimerQueue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *TimerQueue) Push(x interface{}) {
	n := len(*q)
	task := x.(*TimerTask)
	task.index = n
	*q = append(*q, task)
}

func (q *TimerQueue) Pop() interface{} {
	old := *q
	n := len(old)
	task := old[n-1]
	task.index = -1
	*q = old[0 : n-1]
	return task
}

type TimerManager struct {
	queue   TimerQueue
	mutex   sync.Mutex
	nextId  int64
	trigger chan *TimerTask
}

func NewTimerManager() *TimerManager {
	manager := &TimerManager{
		queue:   make(TimerQueue, 0),
		trigger: make(chan *TimerTask, 1000),
		nextId:  1,
	}
	heap.Init(&manager.queue)
	go manager.process()
	return manager
}

func (m *TimerManager) AddTimer(delay time.Duration, interval time.Duration, callback func()) int64 {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	task := &TimerTask{
		Id:       m.nextId,
		Execute:  time.Now().Add(delay),
		Interval: interval,
		Callback: callback,
	}
	m.nextId++

	heap.Push(&m.queue, task)
	return task.Id
}

func (m *TimerManager) RemoveTimer(timerId int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	for i, task := range m.queue {
		if task.Id == timerId {
			heap.Remove(&m.queue, i)
			break
		}
	}
}

func (m *TimerManager) process() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mutex.Lock()
			now := time.Now()

			for m.queue.Len() > 0 {
				task := m.queue[0]
				if task.Execute.After(now) {
					break
				}

				heap.Pop(&m.queue)
				m.trigger <- task

				if task.Interval > 0 {
					task.Execute = now.Add(task.Interval)
					heap.Push(&m.queue, task)
				}
			}
			m.mutex.Unlock()

		case task := <-m.trigger:
			go task.Callback()
		}
	}
}
