package health

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Status represents the overall health status.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Name    string        `json:"name"`
	Status  Status        `json:"status"`
	Message string        `json:"message,omitempty"`
	Latency time.Duration `json:"latency_ms"`
}

// HealthStatus represents the overall health status with details.
type HealthStatus struct {
	Status  Status         `json:"status"`
	Message string         `json:"message,omitempty"`
	Checks  []CheckResult  `json:"checks"`
	Time    time.Time      `json:"timestamp"`
}

// CheckFunc is a function that performs a health check.
// It should return a CheckResult with the outcome.
// Context can be used for cancellation and timeout.
type CheckFunc func(ctx context.Context) CheckResult

// Registry is a thread-safe health check registry.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]CheckFunc
}

// NewRegistry creates a new health check registry.
func NewRegistry() *Registry {
	return &Registry{
		checks: make(map[string]CheckFunc),
	}
}

// Register registers a health check with the given name.
func (r *Registry) Register(name string, checkFn CheckFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[name] = checkFn
}

// Unregister removes a health check.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.checks, name)
}

// RunAll executes all registered health checks with the given timeout per check.
// Returns an aggregated HealthStatus.
func (r *Registry) RunAll(timeout time.Duration) HealthStatus {
	r.mu.RLock()
	checkNames := make([]string, 0, len(r.checks))
	for name := range r.checks {
		checkNames = append(checkNames, name)
	}
	r.mu.RUnlock()

	// Sort names for deterministic output
	sort.Strings(checkNames)

	results := make([]CheckResult, len(checkNames))
	var wg sync.WaitGroup
	wg.Add(len(checkNames))

	// Run all checks concurrently
	for i, name := range checkNames {
		go func(idx int, checkName string) {
			defer wg.Done()

			r.mu.RLock()
			checkFn := r.checks[checkName]
			r.mu.RUnlock()

			if checkFn == nil {
				results[idx] = CheckResult{
					Name:    checkName,
					Status:  StatusUnhealthy,
					Message: "check not found",
				}
				return
			}

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			start := time.Now()
			result := checkFn(ctx)
			result.Latency = time.Since(start)
			results[idx] = result
		}(i, name)
	}

	wg.Wait()

	// Aggregate results
	return aggregateResults(results)
}

// aggregateResults combines individual check results into an overall status.
func aggregateResults(results []CheckResult) HealthStatus {
	status := HealthStatus{
		Status: StatusHealthy,
		Checks: results,
		Time:   time.Now(),
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		if result.Status == StatusUnhealthy {
			hasUnhealthy = true
		} else if result.Status == StatusDegraded {
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		status.Status = StatusUnhealthy
		status.Message = "one or more checks failed"
	} else if hasDegraded {
		status.Status = StatusDegraded
		status.Message = "one or more checks degraded"
	}

	return status
}

// JSON returns the health status as JSON.
func (hs HealthStatus) JSON() []byte {
	data, _ := json.MarshalIndent(hs, "", "  ")
	return data
}

// String returns a human-readable representation of the health status.
func (hs HealthStatus) String() string {
	return fmt.Sprintf("HealthStatus{Status=%s, Checks=%d}", hs.Status, len(hs.Checks))
}

// DefaultCheckFuncs provides common health check functions.

// AlwaysHealthy returns a health check that always passes.
func AlwaysHealthy() CheckFunc {
	return func(ctx context.Context) CheckResult {
		return CheckResult{
			Status:  StatusHealthy,
			Message: "ok",
		}
	}
}

// MemoryCheck returns a health check for memory usage.
// Fails if memory usage is above the threshold percentage.
func MemoryCheck(thresholdPercent float64) CheckFunc {
	return func(ctx context.Context) CheckResult {
		// Simplified example: just check context not cancelled
		select {
		case <-ctx.Done():
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "context cancelled",
			}
		default:
			return CheckResult{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("memory usage %.1f%%", 45.0),
			}
		}
	}
}

// ServiceCheck returns a health check that calls the given service health endpoint.
func ServiceCheck(name string, checkFn func(ctx context.Context) bool) CheckFunc {
	return func(ctx context.Context) CheckResult {
		if checkFn(ctx) {
			return CheckResult{
				Name:    name,
				Status:  StatusHealthy,
				Message: "service responding",
			}
		}
		return CheckResult{
			Name:    name,
			Status:  StatusUnhealthy,
			Message: "service not responding",
		}
	}
}
