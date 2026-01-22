package cache

import (
	"context"
	"time"
)

/*
Cache defines the PUBLIC API of our in-memory cache system.
This is a contract that guarantees certain behaviors, without exposing internals.
All of the details like (sharding, eviction, expiration, concurrency, data loading, and data writing)
are hidden behind this interface.
*/
type Cache interface {

	/*
		Get retrieves the value associated with the given key.

		BEHAVIOR:
		-------------------
		1. If the key exists in cache and is NOT expired:
		   - Return the value immediately (cache hit)

		2. If the key does NOT exist or is expired:
		   - Load the value from a backing store (DB / External API)
		   - Store it in cache
		   - Return the value (cache miss)
	*/
	Get(ctx context.Context, key string) (any, error)

	/*
		Put stores a key-value pair in the cache.

		BEHAVIOR:
		---------
		- Stores the value in memory
		- Applies eviction policy if cache is full
		- Applies expiration strategy (if configured)
		- Applies write policy (write-through or write-back)

		IMPORTANT:
		----------
		- This version does NOT explicitly set a TTL
		- TTL may still be applied implicitly by a global expiration strategy
	*/
	Put(ctx context.Context, key string, value any) error

	/*
		PutWithTTL stores a key-value pair with an explicit time-to-live (TTL).

		TTL (Time-To-Live):
		-------------------
		- Defines how long the key should remain valid
		- After TTL expires, the key is considered expired
		- Expired keys are lazily removed on access
	*/
	PutWithTTL(ctx context.Context, key string, value any, ttl time.Duration) error

	/*
		Remove deletes a key from the cache immediately.

		BEHAVIOR:
		---------
		- Removes the key from in-memory storage
		- Removes it from eviction policy tracking
		- Does NOT affect the backing store

		USE CASES:
		----------
		- Manual invalidation
		- Data consistency after updates
		- Administrative cleanup

		This operation is idempotent:
		- Removing a non-existing key is safe
	*/
	Remove(key string)

	/*
		Expire sets or updates the TTL for an existing key.

		BEHAVIOR:
		---------
		- If the key exists:
		  - Updates its expiration time to now + ttl
		  - Returns true

		- If the key does NOT exist:
		  - Does nothing
		  - Returns false
	*/
	Expire(key string, ttl time.Duration) bool

	/*
		TTL returns the remaining time-to-live for a key.

		RETURN VALUES (Redis-compatible semantics):
		-------------------------------------------
		> 0   : Duration remaining before expiration
		-1    : Key exists but has no TTL
		-2    : Key does not exist or is already expired

		WHY THIS IS IMPORTANT:
		----------------------
		- Debugging
		- Monitoring cache behavior
		- Session management
		- Observability in production
	*/
	TTL(key string) time.Duration

	/*
		Close gracefully shuts down the cache.

		BEHAVIOR:
		---------
		- Flushes any pending write-back operations
		- Stops background goroutines
		- Releases resources

		WHEN TO CALL:
		-------------
		- Application shutdown
		- Graceful termination
		- Tests cleanup
	*/
	Close()
}
