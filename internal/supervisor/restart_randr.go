package supervisor

import (
	"math/rand"
	"sync"
)

// randFloat64Runtime returns a sample in [0, 1) using math/rand.
// The math/rand global is auto-seeded since Go 1.20 so production
// calls get a non-deterministic stream. Tests that need
// determinism override the rngFn package var directly.
//
// A mutex wraps the call so concurrent computeBackoff callers
// (one per crashing worker) don't share mutable global state
// outside of what the standard library guarantees.
func randFloat64Runtime() float64 {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	return rand.Float64()
}
