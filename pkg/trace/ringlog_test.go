package trace

import (
	"sync"
	"testing"
	"time"
)

func TestRingLogAdd(t *testing.T) {
	rl := NewRingLog(5)

	span1 := NewSpan("op1")
	span1.End()
	rl.Add(span1)

	if rl.Count() != 1 {
		t.Fatalf("expected count 1, got %d", rl.Count())
	}

	span2 := NewSpan("op2")
	span2.End()
	rl.Add(span2)

	if rl.Count() != 2 {
		t.Fatalf("expected count 2, got %d", rl.Count())
	}
}

func TestRingLogRecent(t *testing.T) {
	rl := NewRingLog(5)

	// Add 3 spans
	for i := 0; i < 3; i++ {
		s := NewSpan("op")
		s.End()
		rl.Add(s)
	}

	recent := rl.Recent(10) // Ask for more than we have
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent spans, got %d", len(recent))
	}

	recent = rl.Recent(2)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent spans, got %d", len(recent))
	}
}

func TestRingLogWraparound(t *testing.T) {
	rl := NewRingLog(3) // Small buffer to force wraparound

	names := []string{"first", "second", "third", "fourth", "fifth"}
	for _, name := range names {
		s := NewSpan(name)
		s.End()
		rl.Add(s)
	}

	if rl.Count() != 5 {
		t.Fatalf("expected count 5, got %d", rl.Count())
	}

	// After wraparound with capacity 3, we should have the last 3 spans
	recent := rl.Recent(3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent spans, got %d", len(recent))
	}

	// Check that we have the most recent spans (third, fourth, fifth)
	if recent[0].Name != "third" {
		t.Fatalf("expected 'third', got '%s'", recent[0].Name)
	}
	if recent[1].Name != "fourth" {
		t.Fatalf("expected 'fourth', got '%s'", recent[1].Name)
	}
	if recent[2].Name != "fifth" {
		t.Fatalf("expected 'fifth', got '%s'", recent[2].Name)
	}
}

func TestRingLogAll(t *testing.T) {
	rl := NewRingLog(3)

	for i := 0; i < 5; i++ {
		s := NewSpan("op")
		s.End()
		rl.Add(s)
	}

	all := rl.All()
	if len(all) != 3 {
		t.Fatalf("expected all() to return 3 spans (buffer size), got %d", len(all))
	}
}

func TestRingLogCap(t *testing.T) {
	rl := NewRingLog(100)
	if rl.Cap() != 100 {
		t.Fatalf("expected cap 100, got %d", rl.Cap())
	}
}

func TestRingLogClear(t *testing.T) {
	rl := NewRingLog(5)

	for i := 0; i < 5; i++ {
		s := NewSpan("op")
		s.End()
		rl.Add(s)
	}

	if rl.Count() != 5 {
		t.Fatalf("expected count 5 before clear", rl.Count())
	}

	rl.Clear()

	if rl.Count() != 0 {
		t.Fatalf("expected count 0 after clear, got %d", rl.Count())
	}

	recent := rl.Recent(5)
	if len(recent) != 0 {
		t.Fatalf("expected 0 recent spans after clear, got %d", len(recent))
	}
}

// TestRingLogConcurrentAdd tests that concurrent Add operations are safe.
func TestRingLogConcurrentAdd(t *testing.T) {
	rl := NewRingLog(1000)

	numGoroutines := 100
	addsPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < addsPerGoroutine; j++ {
				s := NewSpan("concurrent_op")
				s.WithAttribute("goroutine_id", id)
				s.WithAttribute("op_index", j)
				s.End()
				rl.Add(s)
			}
		}(i)
	}

	wg.Wait()

	expectedCount := int64(numGoroutines * addsPerGoroutine)
	if rl.Count() != expectedCount {
		t.Fatalf("expected count %d, got %d", expectedCount, rl.Count())
	}

	// All spans should be retrievable (buffer is large enough)
	all := rl.All()
	if len(all) != numGoroutines*addsPerGoroutine {
		t.Fatalf("expected all %d spans, got %d", numGoroutines*addsPerGoroutine, len(all))
	}
}

// TestRingLogConcurrentAddAndRead tests concurrent adds and reads.
func TestRingLogConcurrentAddAndRead(t *testing.T) {
	rl := NewRingLog(100)

	done := make(chan struct{})
	go func() {
		// Producer: add spans continuously
		for i := 0; i < 200; i++ {
			s := NewSpan("producer")
			s.WithAttribute("index", i)
			s.End()
			time.Sleep(1 * time.Millisecond)
			rl.Add(s)
		}
		close(done)
	}()

	// Consumer: read recent spans while they're being added
	for i := 0; i < 20; i++ {
		recent := rl.Recent(50)
		_ = recent // Just verify we can read without panicking
		time.Sleep(5 * time.Millisecond)
	}

	<-done

	// Final verification
	if rl.Count() != 200 {
		t.Fatalf("expected final count 200, got %d", rl.Count())
	}
}

func TestRingLogNilSpan(t *testing.T) {
	rl := NewRingLog(5)

	s := NewSpan("op1")
	s.End()
	rl.Add(s)

	// Adding nil should be safe
	rl.Add(nil)

	if rl.Count() != 1 {
		t.Fatalf("expected count 1 (nil not added), got %d", rl.Count())
	}
}
