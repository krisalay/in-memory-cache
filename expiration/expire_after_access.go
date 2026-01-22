package expiration

import (
	"time"

	"github.com/krisalay/in-memory-cache/types"
)

/*
ExpireAfterAccess implements a very common cache behavior called "expire after access" or "sliding TTL".
Every time someone reads the data, the expiration timer is pushed forward. As long as the data keeps
getting used, it stays alive. If nobody touches it for a while, it expires.
*/
type ExpireAfterAccess struct {

	// TTL (Time-To-Live) defines how long the entry should remain valid AFTER it is accessed.
	TTL time.Duration
}

// IsExpired checks whether the entry is expired at this moment.
func (e *ExpireAfterAccess) IsExpired(ent *types.CacheEntry, now time.Time) bool {
	return !ent.ExpireAt.IsZero() && now.After(ent.ExpireAt)
}

/*
OnAccess is called every time the cache successfully returns a value. This is the key part of "expire after access".
1. Update LastAccessedAt to now
2. Push ExpireAt forward by TTL
*/
func (e *ExpireAfterAccess) OnAccess(ent *types.CacheEntry, now time.Time) {
	ent.LastAccessedAt = now
	ent.ExpireAt = now.Add(e.TTL)
}

/*
OnWrite is called when the entry is first written or replaced in the cache.
- We record when the entry was created
- We record the last access time
- We set ExpireAt if it is not already set

We only set ExpireAt if it is currently zero. Because the caller might have explicitly set a TTL
(using PutWithTTL or EXPIRE). We do NOT want to overwrite an explicit TTL.
*/
func (e *ExpireAfterAccess) OnWrite(ent *types.CacheEntry, now time.Time) {
	ent.CreatedAt = now
	ent.LastAccessedAt = now

	// Only set expiration if it wasn't explicitly set before
	if ent.ExpireAt.IsZero() {
		ent.ExpireAt = now.Add(e.TTL)
	}
}
