// This file implements LRU eviction.

package eviction

// lruNode represents ONE key inside the LRU structure. We use a doubly-linked list to track usage order.
type lruNode struct {
	// key is the cache key this node represents
	key string

	// prev points to the node that was used just after this one
	prev *lruNode

	// next points to the node that was used just before this one
	next *lruNode
}

// lru is the concrete implementation of the LRU eviction policy.
type lru struct {
	// nodes maps cache keys to their corresponding list nodes.
	// This allows us to find and move nodes in O(1) time.
	nodes map[string]*lruNode

	// head points to the MOST recently used key
	head *lruNode

	// tail points to the LEAST recently used key
	tail *lruNode
}

func newLRU() *lru {
	return &lru{nodes: make(map[string]*lruNode)}
}

// OnGet is called whenever a key is read from the cache. If a key is accessed, it becomes "recently used".
// So we: Find its node and move it to the front of the list
func (l *lru) OnGet(k string) {
	if n, ok := l.nodes[k]; ok {
		l.moveToFront(n)
	}
}

// OnPut is called whenever a new key is added to the cache.
// - If the key already exists, do nothing (it will be handled by OnGet instead)
// - If the key is new: Create a node and add it to the front (most recently used)
func (l *lru) OnPut(k string) {
	if _, ok := l.nodes[k]; ok {
		return
	}
	n := &lruNode{key: k}
	l.nodes[k] = n
	l.addFront(n)
}

// Evict is called when the cache is full. Removes the LEAST recently used key.
// That key is always at the tail of the list.
func (l *lru) Evict() string {
	if l.tail == nil {
		// Nothing to evict
		return ""
	}

	// Least recently used key
	k := l.tail.key

	// Remove from linked list
	l.remove(l.tail)

	// Remove from map
	delete(l.nodes, k)
	return k
}

// Remove is called when a key is explicitly removed (not evicted due to capacity).
// This keeps LRU’s internal state consistent.
func (l *lru) Remove(k string) {
	if n, ok := l.nodes[k]; ok {
		l.remove(n)
		delete(l.nodes, k)
	}
}

// addFront adds a node to the front of the linked list. This marks the node as "most recently used".
func (l *lru) addFront(n *lruNode) {
	n.next = l.head
	if l.head != nil {
		l.head.prev = n
	}
	l.head = n

	// If the list was empty, head and tail are the same
	if l.tail == nil {
		l.tail = n
	}
}

// remove removes a node from the linked list.
// It correctly updates:
// - Previous node’s next pointer
// - Next node’s prev pointer
// - Head and tail if needed
func (l *lru) remove(n *lruNode) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		l.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		l.tail = n.prev
	}
}

// moveToFront is used when a key is accessed.
// 1. Remove node from its current position
// 2. Add it to the front
// This marks it as most recently used.
func (l *lru) moveToFront(n *lruNode) {
	l.remove(n)
	l.addFront(n)
}
