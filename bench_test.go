package main

import (
	"context"
	"testing"
	"time"

	"github.com/ErturkCan/telemetry-core/pkg/health"
	"github.com/ErturkCan/telemetry-core/pkg/metric"
	"github.com/ErturkCan/telemetry-core/pkg/trace"
)

// BenchmarkCounterInc benchmarks lock-free counter increment.
// Tests the hot path performance under no contention.
func BenchmarkCounterInc(b *testing.B) {
	c := metric.NewCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Inc()
	}
}

// BenchmarkCounterIncParallel benchmarks counter increment under high contention.
// All goroutines increment the same counter simultaneously.
func BenchmarkCounterIncParallel(b *testing.B) {
	c := metric.NewCounter()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Inc()
		}
	})
}

// BenchmarkCounterAdd benchmarks adding a value to the counter.
func BenchmarkCounterAdd(b *testing.B) {
	c := metric.NewCounter()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Add(1)
	}
}

// BenchmarkCounterAddParallel benchmarks Add under high contention.
func BenchmarkCounterAddParallel(b *testing.B) {
	c := metric.NewCounter()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Add(1)
		}
	})
}

// BenchmarkCounterValue benchmarks reading the counter value.
func BenchmarkCounterValue(b *testing.B) {
	c := metric.NewCounter()
	c.Add(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = c.Value()
	}
}

// BenchmarkGaugeSet benchmarks setting a gauge value.
func BenchmarkGaugeSet(b *testing.B) {
	g := metric.NewGauge()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Set(42.5)
	}
}

// BenchmarkGaugeSetParallel benchmarks Set under high contention.
func BenchmarkGaugeSetParallel(b *testing.B) {
	g := metric.NewGauge()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Set(42.5)
		}
	})
}

// BenchmarkGaugeInc benchmarks incrementing a gauge.
func BenchmarkGaugeInc(b *testing.B) {
	g := metric.NewGauge()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Inc()
	}
}

// BenchmarkGaugeIncParallel benchmarks Inc under high contention.
func BenchmarkGaugeIncParallel(b *testing.B) {
	g := metric.NewGauge()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			g.Inc()
		}
	})
}

// BenchmarkHistogramRecord benchmarks recording a value in a histogram.
// This is the hot path for latency measurement.
func BenchmarkHistogramRecord(b *testing.B) {
	h := metric.NewHistogram(0, 5000, 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Record(float64(i % 2500))
	}
}

// BenchmarkHistogramRecordParallel benchmarks histogram record under high contention.
func BenchmarkHistogramRecordParallel(b *testing.B) {
	h := metric.NewHistogram(0, 5000, 50)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			h.Record(float64(i % 2500))
			i++
		}
	})
}

// BenchmarkHistogramPercentile benchmarks calculating a percentile.
func BenchmarkHistogramPercentile(b *testing.B) {
	h := metric.NewHistogram(0, 5000, 50)
	for i := 0; i < 10000; i++ {
		h.Record(float64(i % 5000))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = h.Percentile(99)
	}
}

// BenchmarkSpanCreate benchmarks creating a new span.
func BenchmarkSpanCreate(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trace.NewSpan("operation")
	}
}

// BenchmarkSpanEnd benchmarks ending a span.
func BenchmarkSpanEnd(b *testing.B) {
	spans := make([]*trace.Span, b.N)
	for i := 0; i < b.N; i++ {
		spans[i] = trace.NewSpan("operation")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		spans[i].End()
	}
}

// BenchmarkSpanWithAttribute benchmarks adding an attribute to a span.
func BenchmarkSpanWithAttribute(b *testing.B) {
	span := trace.NewSpan("operation")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		span.WithAttribute("key", "value")
	}
}

// BenchmarkRingLogAdd benchmarks adding a span to the ring buffer.
func BenchmarkRingLogAdd(b *testing.B) {
	rl := trace.NewRingLog(1000)
	spans := make([]*trace.Span, b.N)
	for i := 0; i < b.N; i++ {
		s := trace.NewSpan("operation")
		s.End()
		spans[i] = s
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Add(spans[i])
	}
}

// BenchmarkRingLogAddParallel benchmarks adding spans to the ring buffer under contention.
func BenchmarkRingLogAddParallel(b *testing.B) {
	rl := trace.NewRingLog(10000)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s := trace.NewSpan("operation")
			s.End()
			rl.Add(s)
		}
	})
}

// BenchmarkRingLogRecent benchmarks retrieving recent spans.
func BenchmarkRingLogRecent(b *testing.B) {
	rl := trace.NewRingLog(1000)
	for i := 0; i < 1000; i++ {
		s := trace.NewSpan("operation")
		s.End()
		rl.Add(s)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rl.Recent(50)
	}
}

// BenchmarkRegistryRegister benchmarks registering a metric.
func BenchmarkRegistryRegister(b *testing.B) {
	reg := metric.NewRegistry()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := metric.NewCounter()
		reg.Register("metric", c)
	}
}

// BenchmarkRegistryGet benchmarks getting a metric from the registry.
func BenchmarkRegistryGet(b *testing.B) {
	reg := metric.NewRegistry()
	c := metric.NewCounter()
	reg.Register("metric", c)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.Get("metric")
	}
}

// BenchmarkPrometheusExport benchmarks exporting metrics in Prometheus format.
func BenchmarkPrometheusExport(b *testing.B) {
	reg := metric.NewRegistry()
	for i := 0; i < 50; i++ {
		reg.Register("counter_"+string(rune(i)), metric.NewCounter())
		reg.Register("gauge_"+string(rune(i)), metric.NewGauge())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = reg.ExportPrometheus()
	}
}

// BenchmarkHealthCheckRun benchmarks running a single health check.
func BenchmarkHealthCheckRun(b *testing.B) {
	registry := health.NewRegistry()
	registry.Register("check1", health.AlwaysHealthy())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.RunAll(100 * time.Millisecond)
	}
}

// BenchmarkHealthCheckRunMultiple benchmarks running multiple health checks.
func BenchmarkHealthCheckRunMultiple(b *testing.B) {
	registry := health.NewRegistry()
	for i := 0; i < 10; i++ {
		registry.Register("check"+string(rune(i)), health.AlwaysHealthy())
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.RunAll(100 * time.Millisecond)
	}
}

// BenchmarkCombinedMetricsHotPath simulates a realistic hot path:
// recording a request latency across counter, gauge, and histogram simultaneously.
func BenchmarkCombinedMetricsHotPath(b *testing.B) {
	counter := metric.NewCounter()
	gauge := metric.NewGauge()
	histogram := metric.NewHistogram(0, 5000, 50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		counter.Inc()
		gauge.Set(float64(i))
		histogram.Record(float64(i % 5000))
	}
}

// BenchmarkCombinedMetricsHotPathParallel simulates realistic contention
// with multiple goroutines recording metrics simultaneously.
func BenchmarkCombinedMetricsHotPathParallel(b *testing.B) {
	counter := metric.NewCounter()
	gauge := metric.NewGauge()
	histogram := metric.NewHistogram(0, 5000, 50)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			counter.Inc()
			gauge.Set(float64(i))
			histogram.Record(float64(i % 5000))
			i++
		}
	})
}

// BenchmarkEndToEndRequest simulates recording a complete HTTP request.
func BenchmarkEndToEndRequest(b *testing.B) {
	counter := metric.NewCounter()
	histogram := metric.NewHistogram(0, 5000, 50)
	rl := trace.NewRingLog(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Start span
		span := trace.NewSpan("HTTP_GET")
		counter.Inc()

		// Record latency
		histogram.Record(123.45)

		// End span and log
		span.End()
		rl.Add(span)
	}
}

// BenchmarkEndToEndRequestParallel simulates concurrent HTTP request handling.
func BenchmarkEndToEndRequestParallel(b *testing.B) {
	counter := metric.NewCounter()
	histogram := metric.NewHistogram(0, 5000, 50)
	rl := trace.NewRingLog(10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			span := trace.NewSpan("HTTP_GET")
			counter.Inc()
			histogram.Record(123.45)
			span.End()
			rl.Add(span)
		}
	})
}

// BenchmarkContextScenario tests the behavior with context cancellation
// which is important for graceful shutdown.
func BenchmarkContextScenario(b *testing.B) {
	registry := health.NewRegistry()
	registry.Register("service", health.ServiceCheck("api", func(ctx context.Context) bool {
		select {
		case <-ctx.Done():
			return false
		default:
			return true
		}
	}))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.RunAll(10 * time.Millisecond)
	}
}
