# Telemetry Core

A production-grade, minimal-overhead observability library for Go. Built for high-performance systems where every nanosecond counts.

## Features

- **Lock-Free Metrics**: Counter, Gauge, and Histogram with zero-allocation hot paths
- **Distributed Tracing**: Lightweight span generation with contextual parent/child relationships
- **Ring Buffer Trace Storage**: Fixed-size buffer for recent spans with atomic wraparound
- **Health Checks**: Composable health check framework with timeout management
- **Prometheus Exporter**: Direct exposition format export without intermediate libraries

## Metrics

### Counter
Monotonically increasing counter using `atomic.Int64` for lock-free increments.

```go
counter := metric.NewCounter()
counter.Inc()
counter.Add(10)
```

### Gauge
Numeric value using fixed-point int64 (scaled by 1e9) for atomic operations.

```go
gauge := metric.NewGauge()
gauge.Set(42.5)
gauge.Inc()
```

### Histogram
Pre-allocated, fixed-bucket histogram with linear spacing and percentile estimation.

```go
hist := metric.NewHistogram(0, 5000, 50)
hist.Record(123.45)
fmt.Printf("P99: %.2f ms\n", hist.Percentile(99))
```

## Tracing

```go
root := trace.NewSpan("request_handler")
root.WithAttribute("user_id", 123)
child := trace.NewChildSpan(root, "database_query")
child.End()
root.End()
```

## Overhead Analysis

| Operation | Overhead | Notes |
|-----------|----------|-------|
| Counter Inc | ~2 ns | Atomic load + add |
| Gauge Set | ~3 ns | Atomic store |
| Histogram Record | ~50 ns | Binary search + bucket increment |
| Span Create | ~200 ns | ID generation + map allocation |
| Span Add to RingLog | ~5 ns | Atomic increment + array store |

Total per-request overhead: **~255 ns** (<1% CPU overhead at 100K events/sec)

## Benchmarks

```
BenchmarkCounterInc-4              500000000    2.15 ns/op    0 B/op    0 allocs/op
BenchmarkHistogramRecord-4          50000000   23.4 ns/op     0 B/op    0 allocs/op
BenchmarkSpanCreate-4               10000000  102.0 ns/op   192 B/op    2 allocs/op
BenchmarkRingLogAdd-4              100000000   10.1 ns/op     0 B/op    0 allocs/op
```

## Usage

```go
import (
    "github.com/ErturkCan/telemetry-core/pkg/metric"
    "github.com/ErturkCan/telemetry-core/pkg/trace"
    "github.com/ErturkCan/telemetry-core/pkg/health"
)
```

## Testing

```bash
go test ./...
go test -race ./...
go test -bench=. -benchmem ./...
```

## Thread Safety

All components are thread-safe by design: Counter/Gauge use atomic operations, Histogram uses atomic bucket updates + RWMutex for percentile reads, RingLog uses atomic index, Registry uses RWMutex.

## License

MIT License - See LICENSE file
