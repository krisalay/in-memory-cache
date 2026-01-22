package types

import "time"

// CacheEntry is intentionally mutable for timestamps.
// Timestamp races are acceptable.
type CacheEntry struct {
	Key            string
	Value          any
	CreatedAt      time.Time
	LastAccessedAt time.Time
	ExpireAt       time.Time // zero => no TTL
}
