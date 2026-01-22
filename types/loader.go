package types

import "context"

// Loader is the contract between the cache and the backing store.
type Loader interface {

	/*
		Load is called when the cache misses.The key was not found in memory,so the cache asks the Loader to fetch it.
		1. Cache checks memory → key not found
		2. Cache calls Load(key)
		3. Loader fetches from DB/API
		4. Cache stores the result in memory
		5. Cache returns the value
	*/
	Load(ctx context.Context, key string) (any, error)

	/*
		Put is called when the cache needs to write databack to the backing store.

		This is used by write policies:
		-------------------------------
		- Write-through: write immediately
		- Write-back: write asynchronously later

		This does NOT store data in the cache.It stores data in the backing store (DB/API/etc).
	*/
	Put(ctx context.Context, key string, value any) error
}
