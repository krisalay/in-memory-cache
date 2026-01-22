// This file implements FIFO eviction.

package eviction

type fifo struct {
	// queue keeps keys in the order they were inserted.
	// The front of the queue (index 0) is the oldest key.
	queue []string

	// set keeps track of which keys are currently in the queue.
	set map[string]struct{}
}

func newFIFO() *fifo {
	return &fifo{
		queue: make([]string, 0),
		set:   make(map[string]struct{}),
	}
}

// OnGet is called when a key is read from the cache. Different eviction strategies
// care about different events. FIFO ignores reads completely.
func (f *fifo) OnGet(string) {}

// OnPut is called when a key is added to the cache.
// If the key is already being tracked: Do nothing. FIFO only cares about the first insertion
// If the key is new: Add it to the end of the queue, and record it in the set
func (f *fifo) OnPut(k string) {
	if _, ok := f.set[k]; ok {
		return
	}
	f.queue = append(f.queue, k)
	f.set[k] = struct{}{}
}

// Evict is called when the cache is full and needs space.
// It returns the key to be evicted
func (f *fifo) Evict() string {
	if len(f.queue) == 0 {
		return ""
	}
	// Oldest key
	k := f.queue[0]
	// Remove it from the queue
	f.queue = f.queue[1:]
	// Remove it from the set
	delete(f.set, k)
	return k
}

/*
Remove is called when a key is explicitly removed from the cache (not because of eviction).
This method ensures that FIFO’s internal data structures stay consistent.

Steps:
------
1. Check if the key is tracked
2. Remove it from the set
3. Remove it from the queue
*/
func (f *fifo) Remove(k string) {
	if _, ok := f.set[k]; !ok {
		// Key not tracked; do nothing
		return
	}

	// Remove from set
	delete(f.set, k)

	// Remove from queue while preserving order
	for i, v := range f.queue {
		if v == k {
			f.queue = append(f.queue[:i], f.queue[i+1:]...)
			break
		}
	}
}
