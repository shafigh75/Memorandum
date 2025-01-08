package db

import "container/heap"

// A MinHeap is a min-heap of ValueWithTTL items.
type MinHeap []heapEntry

type heapEntry struct {
	key          string
	valueWithTTL ValueWithTTL
	index        int // Index of the item in the heap.
}

func (h MinHeap) Len() int { return len(h) }
func (h MinHeap) Less(i, j int) bool {
	return h[i].valueWithTTL.Expiration < h[j].valueWithTTL.Expiration
}
func (h MinHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *MinHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(heapEntry)
	item.index = n
	*h = append(*h, item)
}

func (h *MinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = heapEntry{} // Avoid memory leak
	item.index = -1        // For safety
	*h = old[0 : n-1]
	return item
}

func (h *MinHeap) RemoveByKey(key string) {
	for i, entry := range *h {
		if entry.key == key {
			// Remove the entry from the heap
			heap.Remove(h, i)
			break
		}
	}
}
