package metric

import (
	"math"
	"sync/atomic"
)

// Gauge is a lock-free numeric gauge that can go up or down.
// Values are stored as fixed-point int64 (scaled by 1e9) to avoid floating-point
// atomic issues. This provides ~9 decimal places of precision.
type Gauge struct {
	// value stores the gauge as a fixed-point int64 scaled by 1e9
	value atomic.Int64
}

const fixedPointScale = 1e9

// NewGauge creates a new gauge initialized to zero.
func NewGauge() *Gauge {
	return &Gauge{}
}

// Set sets the gauge to an absolute value.
// Float64 is converted to fixed-point int64 for atomic storage.
func (g *Gauge) Set(v float64) {
	scaled := int64(math.Round(v * fixedPointScale))
	g.value.Store(scaled)
}

// Inc increments the gauge by one.
// Uses compare-and-swap in a loop for atomic increment.
func (g *Gauge) Inc() {
	for {
		old := g.value.Load()
		new := old + int64(fixedPointScale)
		if g.value.CompareAndSwap(old, new) {
			break
		}
	}
}

// Dec decrements the gauge by one.
// Uses compare-and-swap in a loop for atomic decrement.
func (g *Gauge) Dec() {
	for {
		old := g.value.Load()
		new := old - int64(fixedPointScale)
		if g.value.CompareAndSwap(old, new) {
			break
		}
	}
}

// Value returns the current gauge value as a float64.
// Converts from fixed-point back to float64.
func (g *Gauge) Value() float64 {
	return float64(g.value.Load()) / fixedPointScale
}
