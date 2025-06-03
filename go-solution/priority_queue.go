package main

import (
	"container/heap"
	"sync"
)

// QueueItem represents an item in the priority queue.
// It stores the board hash, its priority (F-cost), and associated game data.
// The index is used by the heap.Interface implementation.
type QueueItem struct {
	BoardHash uint64      // Unique hash of the board state
	Priority  int         // F-cost for A*
	BoardData interface{} // Can store move description or other relevant data
	index     int         // Index of the item in the heap, managed by heap.Interface
}

// priorityQueueHeap implements heap.Interface for a slice of *QueueItem.
type priorityQueueHeap []*QueueItem

func (pqh priorityQueueHeap) Len() int { return len(pqh) }

func (pqh priorityQueueHeap) Less(i, j int) bool {
	// Min-heap: prioritize lower F-cost.
	return pqh[i].Priority < pqh[j].Priority
}

func (pqh priorityQueueHeap) Swap(i, j int) {
	pqh[i], pqh[j] = pqh[j], pqh[i]
	pqh[i].index = i
	pqh[j].index = j
}

func (pqh *priorityQueueHeap) Push(x interface{}) {
	item := x.(*QueueItem)
	item.index = len(*pqh)
	*pqh = append(*pqh, item)
}

func (pqh *priorityQueueHeap) Pop() interface{} {
	old := *pqh
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pqh = old[0 : n-1]
	return item
}

// PriorityQueue implements a thread-safe min-priority queue.
type PriorityQueue struct {
	mutex  sync.Mutex
	cond   *sync.Cond
	items  priorityQueueHeap
	closed bool // True if no more items will be added
}

// NewPriorityQueue creates and initializes a new priority queue.
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{}
	pq.cond = sync.NewCond(&pq.mutex)
	// heap.Init(&pq.items) // Not strictly needed if items starts empty
	return pq
}

// Add pushes a new state with a given priority onto the queue.
func (pq *PriorityQueue) Add(priority int, boardHash uint64, boardData interface{}) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	if pq.closed {
		return // Do not add to a closed queue
	}

	item := &QueueItem{
		BoardHash: boardHash,
		Priority:  priority,
		BoardData: boardData,
	}
	heap.Push(&pq.items, item)
	pq.cond.Signal() // Wake up a waiting PopMin, if any
}

// PopMin removes and returns the state with the lowest priority.
// Returns (QueueItem, bool): the item and true if successful,
// or (zero QueueItem, false) if queue is closed and empty.
func (pq *PriorityQueue) PopMin() (QueueItem, bool) {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()

	for len(pq.items) == 0 {
		if pq.closed {
			return QueueItem{}, false
		}
		pq.cond.Wait()
	}

	item := heap.Pop(&pq.items).(*QueueItem)
	return *item, true
}

// SetDoneAdding signals that no more items will be added to the queue.
func (pq *PriorityQueue) SetDoneAdding() {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	pq.closed = true
	pq.cond.Broadcast() // Wake up all waiting PopMin goroutines
}

// IsEmpty checks if the priority queue is empty.
func (pq *PriorityQueue) IsEmpty() bool {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	return len(pq.items) == 0 && !pq.closed // Consider closed state
}

// Size returns the number of items currently in the priority queue.
func (pq *PriorityQueue) Size() int {
	pq.mutex.Lock()
	defer pq.mutex.Unlock()
	return len(pq.items)
}
