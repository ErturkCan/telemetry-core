package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/ErturkCan/telemetry-core/pkg/health"
	"github.com/ErturkCan/telemetry-core/pkg/metric"
	"github.com/ErturkCan/telemetry-core/pkg/trace"
)

// Global metrics and traces
var (
	requestCounter    *metric.Counter
	requestLatency    *metric.Histogram
	activeRequests    *metric.Gauge
	requestSizeGauge  *metric.Gauge
	spanLog           *trace.RingLog
	healthRegistry    *health.Registry
)

func init() {
	// Initialize metrics
	requestCounter = metric.NewCounter()
	metric.Register("http_requests_total", requestCounter)

	requestLatency = metric.NewHistogram(0, 5000, 50) // 0-5000ms with 50 buckets
	metric.Register("http_request_latency_ms", requestLatency)

	activeRequests = metric.NewGauge()
	metric.Register("http_active_requests", activeRequests)

	requestSizeGauge = metric.NewGauge()
	metric.Register("http_request_size_bytes", requestSizeGauge)

	// Initialize traces
	spanLog = trace.NewRingLog(1000) // Keep last 1000 spans

	// Initialize health checks
	healthRegistry = health.NewRegistry()
	healthRegistry.Register("memory", health.AlwaysHealthy())
	healthRegistry.Register("service", health.ServiceCheck("api", func(ctx context.Context) bool {
		return true // Simulate healthy service
	}))
}

func main() {
	// Simulate some work in the background
	go simulateWork()

	// Register HTTP handlers
	http.HandleFunc("/metrics", handleMetrics)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/traces", handleTraces)
	http.HandleFunc("/work", handleWork)
	http.HandleFunc("/", handleRoot)

	addr := ":8080"
	log.Printf("Telemetry demo server listening on http://localhost%s", addr)
	log.Printf("  /metrics - Prometheus metrics")
	log.Printf("  /health  - Health check status")
	log.Printf("  /traces  - Recent trace spans")
	log.Printf("  /work    - Trigger some work")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// handleRoot returns a simple info page
func handleRoot(w http.ResponseWriter, r *http.Request) {
	root := trace.NewSpan("GET /")
	defer root.End()

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<html>
<head><title>Telemetry Demo</title></head>
<body>
<h1>Telemetry Core Demo</h1>
<p>Production observability library demonstration</p>
<ul>
  <li><a href="/metrics">Prometheus Metrics</a></li>
  <li><a href="/health">Health Status</a></li>
  <li><a href="/traces">Recent Traces</a></li>
  <li><a href="/work">Trigger Work</a></li>
</ul>
</body>
</html>`)

	root.WithAttribute("status", http.StatusOK)
	spanLog.Add(root)
}

// handleMetrics exports metrics in Prometheus format
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	root := trace.NewSpan("GET /metrics")
	defer root.End()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, metric.ExportPrometheus())

	root.WithAttribute("status", http.StatusOK)
	spanLog.Add(root)
}

// handleHealth returns health check status as JSON
func handleHealth(w http.ResponseWriter, r *http.Request) {
	root := trace.NewSpan("GET /health")
	defer root.End()

	status := healthRegistry.RunAll(500 * time.Millisecond)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if status.Status == health.StatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if status.Status == health.StatusDegraded {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(status)

	root.WithAttribute("status", http.StatusOK)
	root.WithAttribute("health_status", status.Status)
	spanLog.Add(root)
}

// handleTraces returns recent trace spans as JSON
func handleTraces(w http.ResponseWriter, r *http.Request) {
	root := trace.NewSpan("GET /traces")
	defer root.End()

	recent := spanLog.Recent(50)
	type TraceSpan struct {
		ID        string                 `json:"id"`
		Parent    string                 `json:"parent_id"`
		Name      string                 `json:"name"`
		Duration  string                 `json:"duration"`
		Status    string                 `json:"status"`
		Attrs     map[string]interface{} `json:"attributes"`
	}

	var spans []TraceSpan
	for _, s := range recent {
		status := "unset"
		switch s.Status {
		case trace.StatusOK:
			status = "ok"
		case trace.StatusError:
			status = "error"
		}
		spans = append(spans, TraceSpan{
			ID:       s.SpanID.String(),
			Parent:   s.ParentID.String(),
			Name:     s.Name,
			Duration: s.Duration.String(),
			Status:   status,
			Attrs:    s.Attributes,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count": len(spans),
		"spans": spans,
	})

	root.WithAttribute("status", http.StatusOK)
	root.WithAttribute("span_count", len(spans))
	spanLog.Add(root)
}

// handleWork simulates work with random latency
func handleWork(w http.ResponseWriter, r *http.Request) {
	root := trace.NewSpan("GET /work")
	defer root.End()

	activeRequests.Inc()
	defer activeRequests.Dec()

	requestCounter.Inc()

	// Simulate a request with random latency
	latency := time.Duration(rand.Intn(2000)) * time.Millisecond
	time.Sleep(latency)

	// Record latency in histogram
	requestLatency.Record(float64(latency.Milliseconds()))

	// Random response size
	size := rand.Int63n(1024) + 512
	requestSizeGauge.Set(float64(size))

	// Create child spans for sub-operations
	dbSpan := trace.NewChildSpan(root, "db_query")
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	dbSpan.End()
	spanLog.Add(dbSpan)

	cacheSpan := trace.NewChildSpan(root, "cache_lookup")
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	cacheSpan.End()
	spanLog.Add(cacheSpan)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "work completed",
		"latency": latency.String(),
		"size":    size,
	})

	root.WithAttribute("status", http.StatusOK)
	root.WithAttribute("latency_ms", latency.Milliseconds())
	spanLog.Add(root)
}

// simulateWork continuously generates background traffic
func simulateWork() {
	time.Sleep(1 * time.Second) // Wait for server to start
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var wg sync.WaitGroup
	for range ticker.C {
		// Simulate background operations
		go func() {
			wg.Add(1)
			defer wg.Done()

			span := trace.NewSpan("background_task")
			defer span.End()

			// Simulate work with exponential and uniform distribution
			var latency time.Duration
			switch rand.Intn(3) {
			case 0: // Fast operation (90% of requests)
				latency = time.Duration(rand.Intn(50)) * time.Millisecond
			case 1: // Normal operation
				latency = time.Duration(200+rand.Intn(300)) * time.Millisecond
			default: // Slow operation
				latency = time.Duration(1000+rand.Intn(2000)) * time.Millisecond
			}

			time.Sleep(latency)
			requestLatency.Record(float64(latency.Milliseconds()))
			requestCounter.Inc()

			// Sometimes simulate an error
			if rand.Float64() < 0.05 { // 5% error rate
				span.SetStatus(trace.StatusError, "simulated error")
			} else {
				span.SetStatus(trace.StatusOK, "success")
			}

			span.WithAttribute("latency_ms", latency.Milliseconds())
			spanLog.Add(span)
		}()
	}

	wg.Wait()
}

// Demonstrate concurrent metric operations
func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			// Simulate stress test with concurrent increments
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 1000; j++ {
						requestCounter.Inc()
						requestLatency.Record(float64(rand.Intn(5000)))
					}
				}()
			}
			wg.Wait()
		}
	}()
}
