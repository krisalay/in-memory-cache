// This file defines how cache entries expire over time.

package expiration

import (
	"time"

	"github.com/krisalay/in-memory-cache/types"
)

/*
Strategy is the interface that all expiration rules must follow. Instead of hard-coding
expiration logic into the cache, we define a strategy so expiration behavior can be swapped easily.
*/
type Strategy interface {

	// IsExpired checks if the entry is expired
	IsExpired(*types.CacheEntry, time.Time) bool

	// OnAccess is called whenever a cache entry is read successfully.
	OnAccess(*types.CacheEntry, time.Time)

	// OnWrite is called whenever a cache entry is written or updated.
	OnWrite(*types.CacheEntry, time.Time)
}
