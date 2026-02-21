package metric

import (
	"fmt"
	"math"
	"sort"
	"sync"
)

// Histogram is a fixed-bucket histogram implementation with linear bucket spacing.
// Buckets are pre-allocated and thread-safe via a RWMutex on reads only.
// Hot path (Record) uses only atomic operations on bucket counts.
type Histogram struct {
	buckets    []*atomic.Int64
	boundaries []float64
	mu         sync.RWMutex // Only for percentile/stats calculations
	count      atomic.Int64
	sum        atomic.Int64 // Stored as fixed-point int64 * 1e6
}

// NewHistogram creates a histogram with linearly-spaced buckets.
// min, max define the range; bucketCount is the number of buckets.
// Values below min go to bucket 0, values >= max go to the last bucket.
func NewHistogram(min, max float64, bucketCount int) *Histogram {
	if bucketCount <= 0 {
		panic("histogram: bucketCount must be positive")
	}
	if min >= max {
		panic("histogram: min must be less than max")
	}

	h := &Histogram{
		buckets:    make([]*atomic.Int64, bucketCount),
		boundaries: make([]float64, bucketCount),
	}

	// Pre-allocate all bucket atomics
	for i := 0; i < bucketCount; i++ {
		h.buckets[i] = &atomic.Int64{}
		h.boundaries[i] = min + (max-min)*float64(i+1)/float64(bucketCount)
	}

	return h
}

// Record records a value in the histogram.
// Lock-free on the hot path using only atomic operations.
func (h *Histogram) Record(value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return // Ignore invalid values
	}

	// Find the appropriate bucket using binary search on boundaries
	idx := sort.SearchFloat64s(h.boundaries, value)
	if idx >= len(h.buckets) {
		idx = len(h.buckets) - 1
	}

	// Atomic increment of bucket count
	h.buckets[idx].Add(1)

	// Atomic addition to sum (as fixed-point int64)
	scaledValue := int64(math.Round(value * 1e6))
	h.sum.Add(scaledValue)

	// Atomic increment of total count
	h.count.Add(1)
}

// Count returns the total number of recorded values.
func (h *Histogram) Count() int64 {
	return h.count.Load()
}

// Mean returns the mean of recorded values.
func (h *Histogram) Mean() float64 {
	count := h.count.Load()
	if count == 0 {
		return 0
	}
	sum := h.sum.Load()
	return float64(sum) / 1e6 / float64(count)
}

// Percentile returns the approximate value at the given percentile (0-100).
// Uses linear interpolation within buckets for better accuracy.
func (h *Histogram) Percentile(p float64) float64 {
	if p < 0 || p > 100 {
		panic("histogram: percentile must be between 0 and 100")
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	count := h.count.Load()
	if count == 0 {
		return 0
	}

	targetRank := float64(count) * p / 100
	cumulative := int64(0)

	for i, bucket := range h.buckets {
		bucketCount := bucket.Load()
		if cumulative+bucketCount >= int64(targetRank) {
			// Linear interpolation within this bucket
			if bucketCount == 0 {
				return h.boundaries[i]
			}
			// Estimate position within bucket
			posInBucket := (targetRank - float64(cumulative)) / float64(bucketCount)
			lower := h.boundaries[i] * 0.5
			if i > 0 {
				lower = h.boundaries[i-1]
			}
			return lower + posInBucket*(h.boundaries[i]-lower)
		}
		cumulative += bucketCount
	}

	// Beyond all buckets
	return h.boundaries[len(h.boundaries)-1]
}

// Buckets returns the current bucket counts and boundaries.
// Useful for custom analysis or exporting.
func (h *Histogram) Buckets() (boundaries []float64, counts []int64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	boundaries = make([]float64, len(h.boundaries))
	copy(boundaries, h.boundaries)
	counts = make([]int64, len(h.buckets))
	for i, b := range h.buckets {
		counts[i] = b.Load()
	}
	return
}

// String returns a human-readable representation of the histogram.
func (h *Histogram) String() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	boundaries, counts := h.Buckets()
	s := fmt.Sprintf("Histogram(count=%d, mean=%.4f)\n", h.count.Load(), h.Mean())
	for i, bound := range boundaries {
		s += fmt.Sprintf("  [%.2f: %d]\n", bound, counts[i])
	}
	return s
}
