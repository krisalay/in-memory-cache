// This file implements LFU eviction.

package eviction

// lfuNode represents one key tracked by LFU.
type lfuNode struct {
	key  string // cache key
	freq int    // how many times this key was accessed
}

type lfu struct {
	// nodes lets us quickly find the node for a key
	nodes map[string]*lfuNode

	// freqMap groups keys by how many times they were accessed
	freqMap map[int]map[string]*lfuNode

	// minFreq keeps track of the smallest frequency currently present in the cache.
	// This avoids scanning the entire map on eviction.
	minFreq int
}

func newLFU() *lfu {
	return &lfu{
		nodes:   make(map[string]*lfuNode),
		freqMap: make(map[int]map[string]*lfuNode),
	}
}

// OnGet is called whenever a key is read from the cache.
func (l *lfu) OnGet(k string) {
	n, ok := l.nodes[k]
	if !ok {
		// Key not tracked; nothing to do
		return
	}

	// Remember old frequency
	old := n.freq

	// Increase frequency
	n.freq++

	// Remove key from old frequency bucket
	delete(l.freqMap[old], k)

	// If that bucket becomes empty, clean it up
	if len(l.freqMap[old]) == 0 {
		delete(l.freqMap, old)

		// If this was the minimum frequency,
		// we need to increase minFreq
		if l.minFreq == old {
			l.minFreq++
		}
	}

	// Add key to new frequency bucket
	if l.freqMap[n.freq] == nil {
		l.freqMap[n.freq] = make(map[string]*lfuNode)
	}
	l.freqMap[n.freq][k] = n
}

// OnPut is called when a new key is added to the cache.
func (l *lfu) OnPut(k string) {
	if _, ok := l.nodes[k]; ok {
		// Key already tracked
		return
	}

	// New key starts with frequency 1
	n := &lfuNode{key: k, freq: 1}
	l.nodes[k] = n

	// Add to frequency bucket 1
	if l.freqMap[1] == nil {
		l.freqMap[1] = make(map[string]*lfuNode)
	}
	l.freqMap[1][k] = n

	// Since a new key with freq=1 exists, minFreq must be 1
	l.minFreq = 1
}

// Evict is called when the cache is full. Evict ANY key that has the lowest frequency (minFreq).
// If multiple keys share the same frequency, this implementation evicts one of them arbitrarily.
func (l *lfu) Evict() string {

	// Look into the bucket with the smallest frequency
	for k := range l.freqMap[l.minFreq] {

		// Remove key from frequency bucket
		delete(l.freqMap[l.minFreq], k)

		// Remove key from global map
		delete(l.nodes, k)

		return k
	}

	// Nothing to evict
	return ""
}

// Remove is called when a key is explicitly removed (not because of eviction).
// This ensures LFU’s internal state remains correct.
func (l *lfu) Remove(k string) {
	n, ok := l.nodes[k]
	if !ok {
		// Key not tracked
		return
	}

	// Remove from frequency bucket
	delete(l.freqMap[n.freq], k)

	// Remove from nodes map
	delete(l.nodes, k)
}
