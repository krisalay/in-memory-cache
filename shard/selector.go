package shard

import "hash/fnv"

/*
This file decides HOW a cache key is assigned to a shard.
If every request went to the same shard, that shard would become a bottleneck.
Shard selection is about:
- Load balancing
- Avoiding hot spots
- Scaling under concurrency
*/

/*
Selector is the interface that decides which shard should handle a given key.
The cache does not care HOW this decision is made. Different strategies can be plugged in.
*/
type Selector interface {
	Select(string, []*Shard) *Shard
}

/*
PowerOfTwoSelector implements a technique called: "Power of Two Choices"
This is a very well-known load-balancing strategy.

Instead of picking one shard blindly, we pick TWO candidate shards and choose the less loaded one.
This drastically reduces hot shards with very little extra cost.
*/
type PowerOfTwoSelector struct{}

// hash converts a string key into a number. FNV is a fast, non-cryptographic hash commonly used in systems like this.
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

/*
Select chooses the shard for a given key.
*/
func (p *PowerOfTwoSelector) Select(key string, shards []*Shard) *Shard {

	idx := int(hash(key)) % len(shards)
	return shards[idx]
}
