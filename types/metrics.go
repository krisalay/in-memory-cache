package types

// This file defines how the cache reports what it is doing.

/*
Metrics is an interface that defines what the cache wants to measure.
Each method represents an event in the cache lifecycle. The cache will call these methods whenever something happens.
*/
type Metrics interface {

	// Hit is called when the cache successfully returns a value.
	Hit()

	// Miss is called when the cache does NOT find a key and has to load it from the backing store.
	Miss()

	// Eviction is called when a key is removed because the cache is full and needs space.
	Eviction()

	// Expire is called when a key is removed because it has passed its TTL (time-based expiration).
	Expire()

	// Refresh is called when a refresh hook is triggered.
	Refresh()
}

/*
NoopMetrics is a "do nothing" implementation of Metrics.

Why do we need this?
--------------------
We don't want to force every user of the cache
to implement metrics.

If someone does not care about metrics,
we still want the cache to work without:
- nil pointer checks everywhere
- if metrics != nil conditions

So we provide a default implementation
that simply ignores all metric events.
*/
type NoopMetrics struct{}

// All methods below intentionally do nothing.
// This satisfies the Metrics interface without side effects.

func (NoopMetrics) Hit()      {}
func (NoopMetrics) Miss()     {}
func (NoopMetrics) Eviction() {}
func (NoopMetrics) Expire()   {}
func (NoopMetrics) Refresh()  {}
