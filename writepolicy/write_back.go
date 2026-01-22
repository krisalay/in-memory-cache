package writepolicy

import (
	"context"
	"sync"

	"github.com/krisalay/in-memory-cache/types"
)

// This file implements the "write-back" policy.

// writeReq represents one pending write operationthat needs to be sent to the backing store.
type writeReq struct {
	ctx   context.Context
	key   string
	value any
}

/*
WriteBackPolicy manages asynchronous writes to the backing store.
*/
type WriteBackPolicy struct {

	// store is the backing store (DB, API, etc.)
	store types.Loader

	// ch is a buffered channel that holds pending write requests.
	//
	// Buffering is important:
	// - Allows bursts of writes without blocking
	// - Improves throughput
	ch chan writeReq

	// wg is used to wait for the worker to finish
	// during shutdown.
	wg sync.WaitGroup
}

// NewWriteBackPolicy creates a new write-back policy.
func NewWriteBackPolicy(store types.Loader, buffer int) *WriteBackPolicy {
	w := &WriteBackPolicy{
		store: store,
		ch:    make(chan writeReq, buffer),
	}

	// Start one background worker
	w.wg.Add(1)
	go w.worker()

	return w
}

// OnWrite is called whenever the cache writes a key.
// We do NOT write to the backing store immediately. Instead, we push the write into a queue.
// If the queue is full, we DROP the write. Because blocking would slow down the cache and defeat the purpose of write-back.
func (w *WriteBackPolicy) OnWrite(ctx context.Context, key string, value any) {
	select {
	case w.ch <- writeReq{ctx, key, value}:
		// queued successfully
	default:
		// intentional drop under pressure. This means:
		// - Cache stays fast
		// - Backing store may miss some updates
	}
}

/*
worker runs in the background and processes queued writes.
It continuously: reads from the channel and writes data to the backing store

This is where eventual consistency happens.
*/
func (w *WriteBackPolicy) worker() {
	defer w.wg.Done()

	for req := range w.ch {
		// Ignore errors intentionally.
		// In real systems, this might be logged or retried.
		_ = w.store.Put(req.ctx, req.key, req.value)
	}
}

/*
Close shuts down the write-back policy gracefully.
------------------
1. Close the channel (no more writes accepted)
2. Wait for the worker to finish processing queued writes

Without this, pending writes could be lost when the application shuts down.
*/
func (w *WriteBackPolicy) Close() {
	close(w.ch)
	w.wg.Wait()
}
