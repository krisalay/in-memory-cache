package writepolicy

import "context"

/*
This file defines what a "write policy" is.

Different systems have different needs:
- Some want strong consistency (write-through)
- Some want high performance (write-back)
- Some want custom behavior

Instead of hard-coding one behavior, we define an interface so we can plug in different strategies.
*/

/*
WritePolicy is the contract that all write policies must follow.
The cache engine does not care which policy is used. It simply calls these methods.
*/
type WritePolicy interface {

	/*
		OnWrite is called whenever the cache writes a key.
	*/
	OnWrite(ctx context.Context, key string, value any)

	/*
		Close is called when the cache is shutting down.
	*/
	Close()
}
