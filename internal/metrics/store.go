package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/api/router"
	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

// SharedMetricsStore manages shared metrics across multiple server instances
type SharedMetricsStore struct {
	filePath     string
	instanceID   string
	mutex        sync.RWMutex
	lastUpdate   time.Time
	updateTicker *time.Ticker
	stopChan     chan bool
}

// InstanceMetrics represents metrics for a single server instance
type InstanceMetrics struct {
	InstanceID         string                         `json:"instance_id"`
	LastUpdate         time.Time                      `json:"last_update"`
	TotalRequests      int64                          `json:"total_requests"`
	SuccessfulRequests int64                          `json:"successful_requests"`
	FailedRequests     int64                          `json:"failed_requests"`
	FallbackAttempts   int64                          `json:"fallback_attempts"`
	HealthStatus       map[string]*router.HealthStatus `json:"health_status"`
	ProviderMetrics    map[string]router.ProviderMetrics `json:"provider_metrics"`
	OverallLatency     router.OverallLatencyMetrics   `json:"overall_latency"`
}

// AggregatedMetrics represents combined metrics from all instances
type AggregatedMetrics struct {
	TotalRequests      int64                          `json:"TotalRequests"`
	SuccessfulRequests int64                          `json:"SuccessfulRequests"`
	FailedRequests     int64                          `json:"FailedRequests"`
	FallbackAttempts   int64                          `json:"FallbackAttempts"`
	ActiveInstances    int                            `json:"ActiveInstances"`
	HealthStatus       map[string]*router.HealthStatus `json:"HealthStatus"`
	ProviderMetrics    map[string]router.ProviderMetrics `json:"ProviderMetrics"`
	OverallLatency     router.OverallLatencyMetrics   `json:"OverallLatency"`
}

// StoredMetrics represents the entire metrics file structure
type StoredMetrics struct {
	Instances map[string]*InstanceMetrics `json:"instances"`
	Updated   time.Time                   `json:"updated"`
}

// NewSharedMetricsStore creates a new shared metrics store
func NewSharedMetricsStore() (*SharedMetricsStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	metricsDir := filepath.Join(homeDir, ".mcp-code-api")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metrics directory: %w", err)
	}

	filePath := filepath.Join(metricsDir, "metrics.json")
	instanceID := fmt.Sprintf("mcp-%d", os.Getpid())

	store := &SharedMetricsStore{
		filePath:   filePath,
		instanceID: instanceID,
		stopChan:   make(chan bool),
	}

	// Initialize file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := store.writeMetrics(&StoredMetrics{
			Instances: make(map[string]*InstanceMetrics),
			Updated:   time.Now(),
		}); err != nil {
			return nil, fmt.Errorf("failed to initialize metrics file: %w", err)
		}
	}

	return store, nil
}

// Start begins periodic updates of this instance's metrics
func (s *SharedMetricsStore) Start(router *router.EnhancedRouter) {
	// Update every 2 seconds
	s.updateTicker = time.NewTicker(2 * time.Second)

	go func() {
		// Initial update
		s.UpdateMetrics(router)

		for {
			select {
			case <-s.updateTicker.C:
				s.UpdateMetrics(router)
			case <-s.stopChan:
				return
			}
		}
	}()

	logger.Infof("Shared metrics store started for instance: %s", s.instanceID)
}

// Stop stops the metrics updater and cleans up this instance
func (s *SharedMetricsStore) Stop() {
	if s.updateTicker != nil {
		s.updateTicker.Stop()
	}
	close(s.stopChan)

	// Remove this instance from the metrics file
	s.mutex.Lock()
	defer s.mutex.Unlock()

	stored, err := s.readMetrics()
	if err != nil {
		logger.Warnf("Failed to read metrics on shutdown: %v", err)
		return
	}

	delete(stored.Instances, s.instanceID)
	stored.Updated = time.Now()

	if err := s.writeMetrics(stored); err != nil {
		logger.Warnf("Failed to clean up instance metrics: %v", err)
	}

	logger.Infof("Shared metrics store stopped for instance: %s", s.instanceID)
}

// UpdateMetrics updates this instance's metrics in the shared store
func (s *SharedMetricsStore) UpdateMetrics(r *router.EnhancedRouter) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Read current metrics
	stored, err := s.readMetrics()
	if err != nil {
		return fmt.Errorf("failed to read metrics: %w", err)
	}

	// Get current router metrics
	routerMetrics := r.GetMetrics()
	healthStatus := r.GetHealthStatus()
	providerMetrics := r.GetProviderMetrics()
	overallLatency := r.GetOverallLatencyMetrics()

	// Update this instance's metrics
	stored.Instances[s.instanceID] = &InstanceMetrics{
		InstanceID:         s.instanceID,
		LastUpdate:         time.Now(),
		TotalRequests:      routerMetrics.TotalRequests,
		SuccessfulRequests: routerMetrics.SuccessfulRequests,
		FailedRequests:     routerMetrics.FailedRequests,
		FallbackAttempts:   routerMetrics.FallbackAttempts,
		HealthStatus:       healthStatus,
		ProviderMetrics:    providerMetrics,
		OverallLatency:     overallLatency,
	}

	// Clean up stale instances (older than 10 seconds)
	staleThreshold := time.Now().Add(-10 * time.Second)
	for id, instance := range stored.Instances {
		if instance.LastUpdate.Before(staleThreshold) {
			logger.Debugf("Removing stale instance: %s (last update: %s)", id, instance.LastUpdate)
			delete(stored.Instances, id)
		}
	}

	stored.Updated = time.Now()

	// Write back to file
	if err := s.writeMetrics(stored); err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}

	s.lastUpdate = time.Now()
	return nil
}

// GetAggregatedMetrics returns combined metrics from all active instances
func (s *SharedMetricsStore) GetAggregatedMetrics() (*AggregatedMetrics, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stored, err := s.readMetrics()
	if err != nil {
		return nil, fmt.Errorf("failed to read metrics: %w", err)
	}

	// Aggregate metrics from all instances
	aggregated := &AggregatedMetrics{
		HealthStatus:    make(map[string]*router.HealthStatus),
		ProviderMetrics: make(map[string]router.ProviderMetrics),
	}

	for _, instance := range stored.Instances {
		aggregated.TotalRequests += instance.TotalRequests
		aggregated.SuccessfulRequests += instance.SuccessfulRequests
		aggregated.FailedRequests += instance.FailedRequests
		aggregated.FallbackAttempts += instance.FallbackAttempts
		aggregated.ActiveInstances++

		// Merge health status (use most recent)
		for provider, health := range instance.HealthStatus {
			if existing, ok := aggregated.HealthStatus[provider]; !ok || health.LastChecked.After(existing.LastChecked) {
				aggregated.HealthStatus[provider] = health
			}
		}

		// Merge provider metrics
		for providerName, metrics := range instance.ProviderMetrics {
			if existing, ok := aggregated.ProviderMetrics[providerName]; ok {
				// Sum request counters
				existing.TotalRequests += metrics.TotalRequests
				existing.SuccessfulRequests += metrics.SuccessfulRequests
				existing.FailedRequests += metrics.FailedRequests

				// Update min latency (take minimum, excluding zeros)
				if metrics.MinLatency > 0 && (existing.MinLatency == 0 || metrics.MinLatency < existing.MinLatency) {
					existing.MinLatency = metrics.MinLatency
				}

				// Update max latency (take maximum)
				if metrics.MaxLatency > existing.MaxLatency {
					existing.MaxLatency = metrics.MaxLatency
				}

				// Average the percentiles since we can't recalculate accurately
				existing.P50Latency = (existing.P50Latency + metrics.P50Latency) / 2
				existing.P95Latency = (existing.P95Latency + metrics.P95Latency) / 2
				existing.P99Latency = (existing.P99Latency + metrics.P99Latency) / 2

				// Update total latency for average calculation
				existing.TotalLatency += metrics.TotalLatency

				// Take most recent last used timestamp
				if metrics.LastUsed.After(existing.LastUsed) {
					existing.LastUsed = metrics.LastUsed
				}

				aggregated.ProviderMetrics[providerName] = existing
			} else {
				// First time seeing this provider, make a copy
				aggregated.ProviderMetrics[providerName] = metrics
			}
		}
	}

	// Recalculate average latencies for all providers
	for providerName, metrics := range aggregated.ProviderMetrics {
		if metrics.SuccessfulRequests > 0 {
			metrics.AvgLatency = metrics.TotalLatency / time.Duration(metrics.SuccessfulRequests)
			aggregated.ProviderMetrics[providerName] = metrics
		}
	}

	// Aggregate overall latency metrics across instances
	var overallMinLatency, overallP50Latency, overallP95Latency, overallP99Latency, overallMaxLatency time.Duration
	var instanceCount int
	for _, instance := range stored.Instances {
		// Update min latency (take minimum, excluding zeros)
		if instance.OverallLatency.MinLatency > 0 && (overallMinLatency == 0 || instance.OverallLatency.MinLatency < overallMinLatency) {
			overallMinLatency = instance.OverallLatency.MinLatency
		}
		// Update max latency (take maximum)
		if instance.OverallLatency.MaxLatency > overallMaxLatency {
			overallMaxLatency = instance.OverallLatency.MaxLatency
		}
		// Sum percentiles for averaging
		overallP50Latency += instance.OverallLatency.P50Latency
		overallP95Latency += instance.OverallLatency.P95Latency
		overallP99Latency += instance.OverallLatency.P99Latency
		instanceCount++
	}
	// Average the percentiles
	if instanceCount > 0 {
		overallP50Latency = overallP50Latency / time.Duration(instanceCount)
		overallP95Latency = overallP95Latency / time.Duration(instanceCount)
		overallP99Latency = overallP99Latency / time.Duration(instanceCount)
	}
	aggregated.OverallLatency = router.OverallLatencyMetrics{
		MinLatency: overallMinLatency,
		P50Latency: overallP50Latency,
		P95Latency: overallP95Latency,
		P99Latency: overallP99Latency,
		MaxLatency: overallMaxLatency,
	}

	return aggregated, nil
}

// readMetrics reads metrics from the file (caller must hold lock)
func (s *SharedMetricsStore) readMetrics() (*StoredMetrics, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &StoredMetrics{
				Instances: make(map[string]*InstanceMetrics),
				Updated:   time.Now(),
			}, nil
		}
		return nil, err
	}

	var stored StoredMetrics
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}

	if stored.Instances == nil {
		stored.Instances = make(map[string]*InstanceMetrics)
	}

	return &stored, nil
}

// writeMetrics writes metrics to the file (caller must hold lock)
func (s *SharedMetricsStore) writeMetrics(stored *StoredMetrics) error {
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	// Write to temporary file first
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpFile, s.filePath)
}