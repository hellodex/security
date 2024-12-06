package system

import (
	"fmt"
	"sync"
)

type processSlotHandler func(slot uint64, lock *sync.WaitGroup)

type SlotQueue struct {
	sync.Mutex
	slots  []uint64
	notify chan struct{}
}

func NewSlotQueue() *SlotQueue {
	return &SlotQueue{
		slots:  make([]uint64, 0),
		notify: make(chan struct{}, 1),
	}
}

func (q *SlotQueue) Enqueue(item uint64) {
	q.Lock()
	q.slots = append(q.slots, item)
	select {
	case q.notify <- struct{}{}:
	default:
	}
	q.Unlock()
}

func (q *SlotQueue) Dequeue() (uint64, error) {
	q.Lock()
	defer q.Unlock()

	if len(q.slots) == 0 {
		return 0, fmt.Errorf("queue is empty")
	}
	item := q.slots[0]
	q.slots = q.slots[1:]
	return item, nil
}

func (q *SlotQueue) BatchDequeue(size int) ([]uint64, error) {
	q.Lock()
	defer q.Unlock()

	if size <= 0 {
		return nil, fmt.Errorf("wrong size")
	}
	var ret []uint64
	if len(q.slots) > size {
		ret = make([]uint64, size)
		copy(ret, q.slots[:size])
		q.slots = q.slots[size:]
	} else {
		ret = make([]uint64, len(q.slots))
		copy(ret, q.slots)
		q.slots = []uint64{}
	}
	return ret, nil
}

func (q *SlotQueue) Size() int {
	q.Lock()
	defer q.Unlock()
	return len(q.slots)
}

func (q *SlotQueue) First() int {
	q.Lock()
	defer q.Unlock()
	if len(q.slots) == 0 {
		return 0
	}
	return int(q.slots[0])
}

func (q *SlotQueue) Last() int {
	q.Lock()
	defer q.Unlock()
	if len(q.slots) == 0 {
		return 0
	}
	return int(q.slots[len(q.slots)-1])
}

func (q *SlotQueue) Consumer(size int, handler processSlotHandler) {
	for {
		<-q.notify
		for {
			items, err := q.BatchDequeue(size)
			if err != nil {
				fmt.Printf("Get Consume Batch: %d, %v", size, err)
				continue
			}
			var wg sync.WaitGroup
			for _, item := range items {
				wg.Add(1)
				go handler(item, &wg)
			}
			wg.Wait()
		}
	}
}
