package metric

import "sync/atomic"

// Counter is a lock-free monotonic counter using atomic operations.
// Safe for concurrent access without locks on the hot path.
type Counter struct {
	value atomic.Int64
}

// NewCounter creates a new counter initialized to zero.
func NewCounter() *Counter {
	return &Counter{}
}

// Inc increments the counter by one.
// Zero-copy, lock-free operation using atomic compare-and-swap.
func (c *Counter) Inc() {
	c.value.Add(1)
}

// Add increments the counter by the given amount.
// Panics if n is negative.
func (c *Counter) Add(n int64) {
	if n < 0 {
		panic("counter: cannot add negative value")
	}
	c.value.Add(n)
}

// Value returns the current counter value.
// Provides a consistent snapshot at the moment of call.
func (c *Counter) Value() int64 {
	return c.value.Load()
}

// Reset sets the counter back to zero.
// Useful for testing; avoid in production hot paths.
func (c *Counter) Reset() {
	c.value.Store(0)
}
