package trace

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

var spanIDCounter atomic.Int64

// SpanID is a globally unique span identifier.
type SpanID uint64

// NewSpanID generates a new unique span ID.
func NewSpanID() SpanID {
	// Combine counter and random bits for uniqueness
	counter := spanIDCounter.Add(1)
	random := rand.Int63()
	return SpanID((counter << 32) | (random & 0xFFFFFFFF))
}

// String returns the hex representation of the span ID.
func (id SpanID) String() string {
	return hex.EncodeToString([]byte(fmt.Sprintf("%016x", id)))
}

// Span represents a single unit of work in a trace.
// Immutable after creation for lock-free reads.
type Span struct {
	// IDs and relationships
	SpanID   SpanID
	ParentID SpanID // 0 if root span

	// Metadata
	Name       string
	StartTime  time.Time
	Duration   time.Duration
	Attributes map[string]interface{} // Labels and tags

	// Status
	Status StatusCode
	Error  string
}

// StatusCode represents the status of a span.
type StatusCode int

const (
	StatusUnset StatusCode = iota
	StatusOK
	StatusError
)

// NewSpan creates a new root span with the given name.
func NewSpan(name string) *Span {
	return &Span{
		SpanID:     NewSpanID(),
		ParentID:   0,
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Status:     StatusUnset,
	}
}

// NewChildSpan creates a new span as a child of the parent span.
func NewChildSpan(parent *Span, name string) *Span {
	return &Span{
		SpanID:     NewSpanID(),
		ParentID:   parent.SpanID,
		Name:       name,
		StartTime:  time.Now(),
		Attributes: make(map[string]interface{}),
		Status:     StatusUnset,
	}
}

// WithAttribute adds an attribute to the span and returns the span for chaining.
func (s *Span) WithAttribute(key string, value interface{}) *Span {
	s.Attributes[key] = value
	return s
}

// End marks the span as complete and returns the duration.
func (s *Span) End() time.Duration {
	s.Duration = time.Since(s.StartTime)
	return s.Duration
}

// SetStatus sets the span status.
func (s *Span) SetStatus(code StatusCode, msg string) {
	s.Status = code
	if code == StatusError {
		s.Error = msg
	}
}

// String returns a human-readable representation of the span.
func (s *Span) String() string {
	statusStr := "UNSET"
	switch s.Status {
	case StatusOK:
		statusStr = "OK"
	case StatusError:
		statusStr = "ERROR"
	}

	return fmt.Sprintf(
		"Span{ID=%s Parent=%s Name=%q Duration=%v Status=%s}",
		s.SpanID, s.ParentID, s.Name, s.Duration, statusStr,
	)
}
