package writepolicy

import (
	"context"

	"github.com/krisalay/in-memory-cache/types"
)

/*
This file implements the "write-through" policy.

Whenever the cache writes data, it immediately writes the same data to the backing store.

So the flow is: Cache write → DB write (synchronous)
*/

/*
It directly forwards every cache write to the backing store.
*/
type WriteThroughPolicy struct {

	// store is the backing store (DB, API, etc.) where data must be persisted immediately.
	store types.Loader
}

/*
NewWriteThroughPolicy creates a new write-through policy.
*/
func NewWriteThroughPolicy(store types.Loader) *WriteThroughPolicy {
	return &WriteThroughPolicy{store: store}
}

/*
OnWrite is called whenever the cache writes a key. We immediately write the data to the backing store.
  - This call is synchronous
  - The cache write is not considered complete
    until the backing store write finishes
  - If the backing store is slow, cache writes become slow
*/
func (w *WriteThroughPolicy) OnWrite(ctx context.Context, key string, value any) {
	// Ignore errors for simplicity.
	// In real systems, this might be handled or logged.
	_ = w.store.Put(ctx, key, value)
}

/*
Close is required by the WritePolicy interface.  Write-through does not use background workers,
so there is nothing to clean up. We intentionally leave this empty.
*/
func (w *WriteThroughPolicy) Close() {}
