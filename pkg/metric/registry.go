package metric

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Metric is the interface that all metrics must implement for export.
type Metric interface {
	// Name returns the metric name
	Name() string
	// Type returns the metric type (counter, gauge, histogram)
	Type() string
	// PrometheusFormat returns the metric in Prometheus exposition format
	PrometheusFormat() string
}

// counterMetric wraps a Counter for the Metric interface
type counterMetric struct {
	name  string
	value int64
}

func (cm *counterMetric) Name() string                { return cm.name }
func (cm *counterMetric) Type() string                { return "counter" }
func (cm *counterMetric) PrometheusFormat() string    { return cm.name }

// gaugeMetric wraps a Gauge for the Metric interface
type gaugeMetric struct {
	name  string
	value float64
}

func (gm *gaugeMetric) Name() string                { return gm.name }
func (gm *gaugeMetric) Type() string                { return "gauge" }
func (gm *gaugeMetric) PrometheusFormat() string    { return gm.name }

// histogramMetric wraps a Histogram for the Metric interface
type histogramMetric struct {
	name string
	h    *Histogram
}

func (hm *histogramMetric) Name() string             { return hm.name }
func (hm *histogramMetric) Type() string             { return "histogram" }
func (hm *histogramMetric) PrometheusFormat() string { return hm.name }

// Registry is a thread-safe global metric registry.
// Registration is protected by a mutex; metric reads are lock-free.
type Registry struct {
	mu      sync.RWMutex
	metrics map[string]interface{} // Stores *Counter, *Gauge, or *Histogram
}

var defaultRegistry = &Registry{
	metrics: make(map[string]interface{}),
}

// Register registers a metric by name in the global registry.
// Thread-safe with mutex only for registration (not on hot path).
func Register(name string, metric interface{}) {
	defaultRegistry.Register(name, metric)
}

// Get retrieves a metric by name from the global registry.
func Get(name string) interface{} {
	return defaultRegistry.Get(name)
}

// ExportPrometheus exports all metrics in Prometheus exposition format.
func ExportPrometheus() string {
	return defaultRegistry.ExportPrometheus()
}

// NewRegistry creates a new metric registry.
func NewRegistry() *Registry {
	return &Registry{
		metrics: make(map[string]interface{}),
	}
}

// Register registers a metric in this registry.
func (r *Registry) Register(name string, metric interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	switch m := metric.(type) {
	case *Counter:
		r.metrics[name] = m
	case *Gauge:
		r.metrics[name] = m
	case *Histogram:
		r.metrics[name] = m
	default:
		panic(fmt.Sprintf("registry: unknown metric type %T", metric))
	}
}

// Get retrieves a metric by name.
func (r *Registry) Get(name string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.metrics[name]
}

// GetCounter returns a metric as a Counter, or nil if not found or wrong type.
func (r *Registry) GetCounter(name string) *Counter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.metrics[name].(*Counter); ok {
		return c
	}
	return nil
}

// GetGauge returns a metric as a Gauge, or nil if not found or wrong type.
func (r *Registry) GetGauge(name string) *Gauge {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if g, ok := r.metrics[name].(*Gauge); ok {
		return g
	}
	return nil
}

// GetHistogram returns a metric as a Histogram, or nil if not found or wrong type.
func (r *Registry) GetHistogram(name string) *Histogram {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if h, ok := r.metrics[name].(*Histogram); ok {
		return h
	}
	return nil
}

// ExportPrometheus exports all metrics in Prometheus exposition format.
// Format: metric_name metric_value [labels]
func (r *Registry) ExportPrometheus() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var lines []string

	// Collect all metrics sorted by name for deterministic output
	names := make([]string, 0, len(r.metrics))
	for name := range r.metrics {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		metric := r.metrics[name]
		switch m := metric.(type) {
		case *Counter:
			lines = append(lines, fmt.Sprintf("# HELP %s Counter metric\n# TYPE %s counter\n%s %d",
				name, name, name, m.Value()))

		case *Gauge:
			lines = append(lines, fmt.Sprintf("# HELP %s Gauge metric\n# TYPE %s gauge\n%s %g",
				name, name, name, m.Value()))

		case *Histogram:
			// Histogram exports as multiple time series: _count, _sum, _bucket
			count := m.Count()
			sum := m.sum.Load()
			boundaries, buckets := m.Buckets()

			var histLines []string
			histLines = append(histLines, fmt.Sprintf("# HELP %s Histogram metric\n# TYPE %s histogram", name, name))
			histLines = append(histLines, fmt.Sprintf("%s_count %d", name, count))
			histLines = append(histLines, fmt.Sprintf("%s_sum %g", name, float64(sum)/1e6))

			cumulative := int64(0)
			for i, bound := range boundaries {
				cumulative += buckets[i]
				histLines = append(histLines, fmt.Sprintf("%s_bucket{le=\"%g\"} %d",
					name, bound, cumulative))
			}
			histLines = append(histLines, fmt.Sprintf("%s_bucket{le=\"+Inf\"} %d", name, count))
			lines = append(lines, strings.Join(histLines, "\n"))
		}
	}

	return strings.Join(lines, "\n\n") + "\n"
}

// ListMetrics returns the names of all registered metrics.
func (r *Registry) ListMetrics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.metrics))
	for name := range r.metrics {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
