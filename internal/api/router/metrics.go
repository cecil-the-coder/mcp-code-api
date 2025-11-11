package router

import (
	"sort"
	"sync"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/types"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

// ProviderMetrics holds detailed metrics for a single provider or model
type ProviderMetrics struct {
	Name               string        `json:"Name"`
	Model              string        `json:"Model,omitempty"`      // For multi-model providers
	IsModel            bool          `json:"IsModel,omitempty"`    // True if this is a model, not a provider
	TotalRequests      int64         `json:"TotalRequests"`
	SuccessfulRequests int64         `json:"SuccessfulRequests"`
	FailedRequests     int64         `json:"FailedRequests"`
	MinLatency         time.Duration `json:"MinLatency"`
	MaxLatency         time.Duration `json:"MaxLatency"`
	P50Latency         time.Duration `json:"P50Latency"`
	P95Latency         time.Duration `json:"P95Latency"`
	P99Latency         time.Duration `json:"P99Latency"`
	AvgLatency         time.Duration `json:"AvgLatency"`
	TotalLatency       time.Duration `json:"-"` // For calculating average
	LastUsed           time.Time     `json:"LastUsed"`
	TotalTokens        int64         `json:"TotalTokens"`
	AvgTokensPerSec    float64       `json:"AvgTokensPerSec"`
}

// LatencyTracker maintains latency history for percentile calculations
type LatencyTracker struct {
	mutex     sync.RWMutex
	latencies []time.Duration
	maxSize   int
}

// NewLatencyTracker creates a new latency tracker with a max size
func NewLatencyTracker(maxSize int) *LatencyTracker {
	return &LatencyTracker{
		latencies: make([]time.Duration, 0, maxSize),
		maxSize:   maxSize,
	}
}

// Add adds a latency measurement
func (lt *LatencyTracker) Add(latency time.Duration) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	if len(lt.latencies) >= lt.maxSize {
		// Remove oldest entry (FIFO)
		lt.latencies = lt.latencies[1:]
	}
	lt.latencies = append(lt.latencies, latency)
}

// GetPercentiles calculates percentiles from stored latencies
func (lt *LatencyTracker) GetPercentiles() (min, p50, p95, p99, max time.Duration) {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()

	if len(lt.latencies) == 0 {
		return 0, 0, 0, 0, 0
	}

	// Copy and sort
	sorted := make([]time.Duration, len(lt.latencies))
	copy(sorted, lt.latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	min = sorted[0]
	max = sorted[len(sorted)-1]
	p50 = sorted[len(sorted)*50/100]

	if len(sorted) > 1 {
		p95 = sorted[len(sorted)*95/100]
		p99 = sorted[len(sorted)*99/100]
	} else {
		p95 = sorted[0]
		p99 = sorted[0]
	}

	return min, p50, p95, p99, max
}

// GetAverage calculates the average latency
func (lt *LatencyTracker) GetAverage() time.Duration {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()

	if len(lt.latencies) == 0 {
		return 0
	}

	var sum time.Duration
	for _, l := range lt.latencies {
		sum += l
	}
	return sum / time.Duration(len(lt.latencies))
}

// ProviderMetricsTracker tracks metrics and latencies for a provider
type ProviderMetricsTracker struct {
	metrics         *ProviderMetrics
	latencyTracker  *LatencyTracker
	mutex           sync.RWMutex
}

// NewProviderMetricsTracker creates a new provider metrics tracker
func NewProviderMetricsTracker(providerName string) *ProviderMetricsTracker {
	return &ProviderMetricsTracker{
		metrics: &ProviderMetrics{
			Name:    providerName,
			IsModel: false,
		},
		latencyTracker: NewLatencyTracker(1000), // Keep last 1000 requests
	}
}

// NewModelMetricsTracker creates a new model metrics tracker
func NewModelMetricsTracker(providerName, modelName string) *ProviderMetricsTracker {
	return &ProviderMetricsTracker{
		metrics: &ProviderMetrics{
			Name:    providerName,
			Model:   modelName,
			IsModel: true,
		},
		latencyTracker: NewLatencyTracker(1000), // Keep last 1000 requests
	}
}

// RecordRequest records a request attempt
func (pmt *ProviderMetricsTracker) RecordRequest(success bool, latency time.Duration, tokenUsage *types.Usage) {
	pmt.mutex.Lock()
	defer pmt.mutex.Unlock()

	pmt.metrics.TotalRequests++
	pmt.metrics.LastUsed = time.Now()

	if success {
		pmt.metrics.SuccessfulRequests++

		// Track latency for successful requests
		pmt.latencyTracker.Add(latency)

		// Update min/max
		if pmt.metrics.MinLatency == 0 || latency < pmt.metrics.MinLatency {
			pmt.metrics.MinLatency = latency
		}
		if latency > pmt.metrics.MaxLatency {
			pmt.metrics.MaxLatency = latency
		}

		// Update total for average calculation
		pmt.metrics.TotalLatency += latency

		// Track token usage
		if tokenUsage != nil {
			oldTotal := pmt.metrics.TotalTokens
			pmt.metrics.TotalTokens += int64(tokenUsage.TotalTokens)
			logger.Debugf("Metrics [%s]: Accumulating tokens - Previous: %d, Adding: %d, New total: %d",
				pmt.metrics.Name, oldTotal, tokenUsage.TotalTokens, pmt.metrics.TotalTokens)
		} else {
			logger.Warnf("Metrics [%s]: Received nil tokenUsage, not accumulating tokens", pmt.metrics.Name)
		}
	} else {
		pmt.metrics.FailedRequests++
	}
}

// GetMetrics returns a snapshot of current metrics with calculated percentiles
func (pmt *ProviderMetricsTracker) GetMetrics() ProviderMetrics {
	pmt.mutex.RLock()
	defer pmt.mutex.RUnlock()

	metrics := *pmt.metrics

	// Calculate percentiles
	min, p50, p95, p99, max := pmt.latencyTracker.GetPercentiles()
	metrics.MinLatency = min
	metrics.P50Latency = p50
	metrics.P95Latency = p95
	metrics.P99Latency = p99
	metrics.MaxLatency = max

	// Calculate average from latency tracker
	metrics.AvgLatency = pmt.latencyTracker.GetAverage()

	// Calculate tokens per second
	if metrics.SuccessfulRequests > 0 && metrics.TotalTokens > 0 && metrics.AvgLatency > 0 {
		// tokens/sec = total_tokens / (avg_latency_seconds * successful_requests)
		avgLatencySeconds := metrics.AvgLatency.Seconds()
		metrics.AvgTokensPerSec = float64(metrics.TotalTokens) / (avgLatencySeconds * float64(metrics.SuccessfulRequests))
	}

	return metrics
}