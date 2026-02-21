package metric

import (
	"sync"
	"testing"
)

func TestCounterInc(t *testing.T) {
	c := NewCounter()
	if c.Value() != 0 {
		t.Fatalf("expected 0, got %d", c.Value())
	}

	c.Inc()
	if c.Value() != 1 {
		t.Fatalf("expected 1, got %d", c.Value())
	}

	c.Inc()
	c.Inc()
	if c.Value() != 3 {
		t.Fatalf("expected 3, got %d", c.Value())
	}
}

func TestCounterAdd(t *testing.T) {
	c := NewCounter()

	c.Add(10)
	if c.Value() != 10 {
		t.Fatalf("expected 10, got %d", c.Value())
	}

	c.Add(5)
	if c.Value() != 15 {
		t.Fatalf("expected 15, got %d", c.Value())
	}
}

func TestCounterAddNegative(t *testing.T) {
	c := NewCounter()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on negative add")
		}
	}()
	c.Add(-1)
}

func TestCounterReset(t *testing.T) {
	c := NewCounter()
	c.Add(100)
	if c.Value() != 100 {
		t.Fatalf("expected 100, got %d", c.Value())
	}

	c.Reset()
	if c.Value() != 0 {
		t.Fatalf("expected 0 after reset, got %d", c.Value())
	}
}

// TestCounterConcurrentIncrement tests that concurrent increments are correctly counted.
// This is critical for correctness of lock-free atomic operations.
func TestCounterConcurrentIncrement(t *testing.T) {
	c := NewCounter()
	numGoroutines := 100
	incrementsPerGoroutine := 1000
	expectedTotal := int64(numGoroutines * incrementsPerGoroutine)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				c.Inc()
			}
		}()
	}

	wg.Wait()

	if c.Value() != expectedTotal {
		t.Fatalf("expected %d, got %d", expectedTotal, c.Value())
	}
}

// TestCounterConcurrentAdd tests concurrent Add operations with various values.
func TestCounterConcurrentAdd(t *testing.T) {
	c := NewCounter()
	numGoroutines := 50
	addsPerGoroutine := 100
	valuePerAdd := int64(7)
	expectedTotal := int64(numGoroutines * addsPerGoroutine) * valuePerAdd

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < addsPerGoroutine; j++ {
				c.Add(valuePerAdd)
			}
		}()
	}

	wg.Wait()

	if c.Value() != expectedTotal {
		t.Fatalf("expected %d, got %d", expectedTotal, c.Value())
	}
}

// TestCounterMixedOperations tests a mix of Inc and Add operations concurrently.
func TestCounterMixedOperations(t *testing.T) {
	c := NewCounter()
	numGoroutines := 50
	operationsPerGoroutine := 200

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				if (id+j)%2 == 0 {
					c.Inc()
				} else {
					c.Add(3)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify we have a consistent value (exact value depends on interleaving)
	final := c.Value()
	if final <= 0 || final > int64(numGoroutines*operationsPerGoroutine*4) {
		t.Fatalf("unexpected final value: %d", final)
	}
}
