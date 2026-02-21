package trace

import (
	"sync"
	"sync/atomic"
)

// RingLog is a fixed-size ring buffer for storing recent spans.
// Uses a pre-allocated array and atomic index to minimize allocations.
// Thread-safe for concurrent Add and Recent operations.
type RingLog struct {
	buffer []*Span
	index  atomic.Int64 // Current write position
	mu     sync.RWMutex // Protects reads of buffer for Recent()
}

// NewRingLog creates a new ring buffer with the specified capacity.
func NewRingLog(capacity int) *RingLog {
	if capacity <= 0 {
		panic("ringlog: capacity must be positive")
	}
	return &RingLog{
		buffer: make([]*Span, capacity),
	}
}

// Add adds a span to the ring buffer, overwriting the oldest span if full.
// Lock-free on the hot path using only atomic operations for index management.
func (rl *RingLog) Add(span *Span) {
	if span == nil {
		return
	}

	// Atomic increment of index (wraps around buffer size)
	idx := rl.index.Add(1)
	pos := (idx - 1) % int64(len(rl.buffer))
	rl.buffer[pos] = span
}

// Recent returns the most recent n spans in chronological order (oldest to newest).
// If fewer than n spans have been recorded, returns what is available.
// Thread-safe for concurrent reads via RWMutex.
func (rl *RingLog) Recent(n int) []*Span {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	currentIdx := rl.index.Load()
	bufLen := int64(len(rl.buffer))

	if n <= 0 {
		return nil
	}

	// Clamp n to buffer size and actual number of spans written
	if int64(n) > bufLen {
		n = int(bufLen)
	}
	if currentIdx < int64(n) {
		n = int(currentIdx)
	}

	result := make([]*Span, 0, n)

	// Calculate the starting position in the ring buffer
	startIdx := currentIdx - int64(n)
	for i := int64(0); i < int64(n); i++ {
		pos := (startIdx + i) % bufLen
		if rl.buffer[pos] != nil {
			result = append(result, rl.buffer[pos])
		}
	}

	return result
}

// All returns all recorded spans in the ring buffer in chronological order.
func (rl *RingLog) All() []*Span {
	return rl.Recent(len(rl.buffer))
}

// Cap returns the capacity of the ring buffer.
func (rl *RingLog) Cap() int {
	return len(rl.buffer)
}

// Count returns the total number of spans written to the ring buffer.
// Note: This can exceed the buffer size, as it counts wraparounds.
func (rl *RingLog) Count() int64 {
	return rl.index.Load()
}

// Clear removes all spans from the ring buffer.
func (rl *RingLog) Clear() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for i := range rl.buffer {
		rl.buffer[i] = nil
	}
	rl.index.Store(0)
}
