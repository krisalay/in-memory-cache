// This file defines the idea of a "refresh hook".
// This hook allows the cache to do something extra WHEN data is read from the cache.
// The goal of refresh is: "Keep data fresh without slowing down reads"

package refresh

import "github.com/krisalay/in-memory-cache/types"

/*
Hook is the interface for refresh behavior.
If a refresh hook is configured, it will be called every time a cache entry is successfully read.

This gives us a chance to:
- Check if the entry is about to expire
- Trigger a background refresh
- Log access patterns
- Preload related data

The cache itself does NOT care what the hook does.
It just calls OnRead and moves on.
*/
type Hook interface {

	/*
		OnRead is called after a successful cache read.
		This method MUST be fast and non blocking because this method runs on the hot read path.
		Blocking here would slow down every cache read.
	*/
	OnRead(key string, ent *types.CacheEntry)
}
