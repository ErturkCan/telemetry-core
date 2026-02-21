package metric

import (
	"math"
	"testing"
)

func TestHistogramCreation(t *testing.T) {
	h := NewHistogram(0, 100, 10)
	if h.Count() != 0 {
		t.Fatalf("expected count 0, got %d", h.Count())
	}
	if h.Mean() != 0 {
		t.Fatalf("expected mean 0, got %g", h.Mean())
	}
}

func TestHistogramRecord(t *testing.T) {
	h := NewHistogram(0, 100, 10)

	h.Record(50)
	if h.Count() != 1 {
		t.Fatalf("expected count 1, got %d", h.Count())
	}

	h.Record(25)
	h.Record(75)
	if h.Count() != 3 {
		t.Fatalf("expected count 3, got %d", h.Count())
	}
}

func TestHistogramMean(t *testing.T) {
	h := NewHistogram(0, 1000, 100)

	h.Record(10.0)
	h.Record(20.0)
	h.Record(30.0)

	mean := h.Mean()
	expected := 20.0
	if math.Abs(mean-expected) > 0.01 {
		t.Fatalf("expected mean ~%g, got %g", expected, mean)
	}
}

func TestHistogramPercentile(t *testing.T) {
	h := NewHistogram(0, 100, 50)

	// Record values 1-100
	for i := 1; i <= 100; i++ {
		h.Record(float64(i))
	}

	tests := []struct {
		percentile float64
		expected   float64
		tolerance  float64
	}{
		{0, 1, 2},      // Min
		{50, 50, 5},    // Median
		{90, 90, 5},    // 90th percentile
		{99, 99, 2},    // 99th percentile
		{100, 100, 2},  // Max
	}

	for _, tt := range tests {
		actual := h.Percentile(tt.percentile)
		if math.Abs(actual-tt.expected) > tt.tolerance {
			t.Errorf("percentile(%g): expected ~%g, got %g", tt.percentile, tt.expected, actual)
		}
	}
}

func TestHistogramIgnoresInvalidValues(t *testing.T) {
	h := NewHistogram(0, 100, 10)

	h.Record(50)
	initialCount := h.Count()

	// These should be ignored
	h.Record(math.NaN())
	h.Record(math.Inf(1))
	h.Record(math.Inf(-1))

	if h.Count() != initialCount {
		t.Fatalf("invalid values should not increase count")
	}
}

func TestHistogramBuckets(t *testing.T) {
	h := NewHistogram(0, 10, 5)

	h.Record(1)
	h.Record(2)
	h.Record(3)
	h.Record(7)
	h.Record(9)

	boundaries, counts := h.Buckets()
	if len(boundaries) != 5 {
		t.Fatalf("expected 5 boundaries, got %d", len(boundaries))
	}
	if len(counts) != 5 {
		t.Fatalf("expected 5 counts, got %d", len(counts))
	}

	totalCount := int64(0)
	for _, c := range counts {
		totalCount += c
	}
	if totalCount != h.Count() {
		t.Fatalf("bucket counts don't sum to total count: %d vs %d", totalCount, h.Count())
	}
}

// TestHistogramConcurrentRecord tests that concurrent Record operations are safe.
func TestHistogramConcurrentRecord(t *testing.T) {
	h := NewHistogram(0, 1000, 100)

	numGoroutines := 100
	recordsPerGoroutine := 100

	done := make(chan bool, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < recordsPerGoroutine; j++ {
				value := float64((id*recordsPerGoroutine + j) % 1000)
				h.Record(value)
			}
			done <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	expectedCount := int64(numGoroutines * recordsPerGoroutine)
	if h.Count() != expectedCount {
		t.Fatalf("expected count %d, got %d", expectedCount, h.Count())
	}

	// Verify percentiles are monotonic
	p50 := h.Percentile(50)
	p90 := h.Percentile(90)
	if p50 > p90 {
		t.Fatalf("percentiles not monotonic: p50=%g > p90=%g", p50, p90)
	}
}

// TestHistogramPercentileAccuracy tests that percentiles are reasonably accurate.
func TestHistogramPercentileAccuracy(t *testing.T) {
	h := NewHistogram(0, 1000, 100)

	// Record a uniform distribution
	for i := 0; i < 10000; i++ {
		h.Record(float64(i % 1000))
	}

	// For uniform distribution, percentiles should roughly match their value
	for p := 10.0; p < 100.0; p += 10.0 {
		percentile := h.Percentile(p)
		expected := p * 10.0 // 0-1000 range, so p% is p*10
		tolerance := expected * 0.15 // 15% tolerance for estimation
		if math.Abs(percentile-expected) > tolerance {
			t.Errorf("percentile(%g): expected ~%g±%g, got %g",
				p, expected, tolerance, percentile)
		}
	}
}
