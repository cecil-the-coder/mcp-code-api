package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cecil-the-coder/mcp-code-api/internal/logger"
)

type MetricsServer struct {
	store  *SharedMetricsStore
	host   string
	port   int
	server *http.Server
}

func NewMetricsServer(store *SharedMetricsStore, host string, port int) *MetricsServer {
	return &MetricsServer{
		store:  store,
		host:   host,
		port:   port,
	}
}

func (s *MetricsServer) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/metrics", s.handleMetrics)
	http.HandleFunc("/api/health", s.handleHealth)
	
	s.server = &http.Server{
		Addr: fmt.Sprintf("%s:%d", s.host, s.port),
	}
	
	logger.Infof("Starting metrics server on %s:%d", s.host, s.port)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Metrics server error: %v", err)
		}
	}()
	return nil
}

func (s *MetricsServer) Stop() error {
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	logger.Infof("Stopping metrics server...")
	return s.server.Shutdown(ctx)
}

func (s *MetricsServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	aggregated, err := s.store.GetAggregatedMetrics()
	if err != nil {
		logger.Errorf("Failed to get aggregated metrics: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(aggregated); err != nil {
		logger.Errorf("Failed to encode metrics: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *MetricsServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	aggregated, err := s.store.GetAggregatedMetrics()
	if err != nil {
		logger.Errorf("Failed to get aggregated metrics: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(aggregated.HealthStatus); err != nil {
		logger.Errorf("Failed to encode health status: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func (s *MetricsServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>Metrics Dashboard</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #1a1a1a; color: #e0e0e0; min-height: 100vh; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        header { text-align: center; margin-bottom: 30px; padding: 20px; background: #2d2d2d; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.3); }
        h1 { color: #4fc3f7; font-size: 2.5em; margin-bottom: 10px; }
        .last-update { color: #9e9e9e; font-size: 0.9em; }
        .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .metric-card { background: #2d2d2d; padding: 20px; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.3); border-left: 4px solid #4fc3f7; }
        .metric-card h3 { color: #81c784; margin-bottom: 10px; font-size: 1.1em; }
        .metric-value { font-size: 2em; font-weight: bold; color: #ffffff; }
        .metric-label { color: #9e9e9e; font-size: 0.9em; margin-top: 5px; }
        .loading { text-align: center; color: #9e9e9e; font-style: italic; }
        .error { color: #f44336; text-align: center; padding: 20px; background: #2d2d2d; border-radius: 10px; margin: 20px 0; }
        .metrics-section { background: #2d2d2d; padding: 20px; border-radius: 10px; box-shadow: 0 4px 6px rgba(0,0,0,0.3); margin-top: 20px; }
        .metrics-section h2 { color: #4fc3f7; margin-bottom: 20px; }
        .provider-metrics-table table { width: 100%; border-collapse: collapse; }
        .provider-metrics-table th { background: #1a1a1a; padding: 12px; text-align: left; color: #4fc3f7; border-bottom: 2px solid #4fc3f7; }
        .provider-metrics-table td { padding: 10px; border-bottom: 1px solid #3a3a3a; color: #e0e0e0; }
        .provider-metrics-table tr:hover { background: #3a3a3a; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>MCP Code API Dashboard</h1>
            <div class="last-update" id="lastUpdate">Loading...</div>
        </header>
        
        <div class="metrics-grid">
            <div class="metric-card">
                <h3>Total Requests</h3>
                <div class="metric-value" id="totalRequests">-</div>
                <div class="metric-label">All incoming requests</div>
            </div>
            <div class="metric-card">
                <h3>Successful Requests</h3>
                <div class="metric-value" id="successfulRequests">-</div>
                <div class="metric-label">Completed successfully</div>
            </div>
            <div class="metric-card">
                <h3>Failed Requests</h3>
                <div class="metric-value" id="failedRequests">-</div>
                <div class="metric-label">Errors occurred</div>
            </div>
            <div class="metric-card">
                <h3>Fallback Attempts</h3>
                <div class="metric-value" id="fallbackAttempts">-</div>
                <div class="metric-label">Provider fallbacks</div>
            </div>
            <div class="metric-card">
                <h3>Success Rate</h3>
                <div class="metric-value" id="successRate">-</div>
                <div class="metric-label">Percentage successful</div>
            </div>
            <div class="metric-card">
                <h3>Active Instances</h3>
                <div class="metric-value" id="activeInstances">-</div>
                <div class="metric-label">Running MCP servers</div>
            </div>
        </div>

        <div class="metrics-section">
            <h2>Total Processing Time (All Requests)</h2>
            <div class="metrics-grid">
                <div class="metric-card">
                    <h3>Min</h3>
                    <div class="metric-value" id="overallMin">-</div>
                    <div class="metric-label">milliseconds</div>
                </div>
                <div class="metric-card">
                    <h3>P50 (Median)</h3>
                    <div class="metric-value" id="overallP50">-</div>
                    <div class="metric-label">milliseconds</div>
                </div>
                <div class="metric-card">
                    <h3>P95</h3>
                    <div class="metric-value" id="overallP95">-</div>
                    <div class="metric-label">milliseconds</div>
                </div>
                <div class="metric-card">
                    <h3>P99</h3>
                    <div class="metric-value" id="overallP99">-</div>
                    <div class="metric-label">milliseconds</div>
                </div>
                <div class="metric-card">
                    <h3>Max</h3>
                    <div class="metric-value" id="overallMax">-</div>
                    <div class="metric-label">milliseconds</div>
                </div>
            </div>
        </div>

        <div class="metrics-section">
            <h2>Provider Performance Metrics</h2>
            <div class="provider-metrics-table" id="providerMetricsTable">
                <div class="loading">Loading provider metrics...</div>
            </div>
        </div>
    </div>
    
    <script>
        function formatDuration(nanos) {
            return (nanos / 1000000).toFixed(2);
        }
        
        function updateMetrics() {
            fetch('/api/metrics')
                .then(function(response) {
                    if (!response.ok) {
                        throw new Error('Network response was not ok');
                    }
                    return response.json();
                })
                .then(function(data) {
                    document.getElementById('totalRequests').innerHTML = data.TotalRequests || 0;
                    document.getElementById('successfulRequests').innerHTML = data.SuccessfulRequests || 0;
                    document.getElementById('failedRequests').innerHTML = data.FailedRequests || 0;
                    document.getElementById('fallbackAttempts').innerHTML = data.FallbackAttempts || 0;
                    document.getElementById('activeInstances').innerHTML = data.ActiveInstances || 0;

                    var successRate = 0;
                    if (data.TotalRequests > 0) {
                        successRate = ((data.SuccessfulRequests / data.TotalRequests) * 100).toFixed(1);
                    }
                    document.getElementById('successRate').innerHTML = successRate + '%';

                    // Update overall latency metrics
                    if (data.OverallLatency) {
                        document.getElementById('overallMin').innerHTML = formatDuration(data.OverallLatency.MinLatency || 0);
                        document.getElementById('overallP50').innerHTML = formatDuration(data.OverallLatency.P50Latency || 0);
                        document.getElementById('overallP95').innerHTML = formatDuration(data.OverallLatency.P95Latency || 0);
                        document.getElementById('overallP99').innerHTML = formatDuration(data.OverallLatency.P99Latency || 0);
                        document.getElementById('overallMax').innerHTML = formatDuration(data.OverallLatency.MaxLatency || 0);
                    } else {
                        document.getElementById('overallMin').innerHTML = '-';
                        document.getElementById('overallP50').innerHTML = '-';
                        document.getElementById('overallP95').innerHTML = '-';
                        document.getElementById('overallP99').innerHTML = '-';
                        document.getElementById('overallMax').innerHTML = '-';
                    }
                    
                    // Fetch health status to combine with metrics
                    fetch('/api/health')
                        .then(function(healthResponse) {
                            return healthResponse.json();
                        })
                        .then(function(healthData) {
                            // Update provider metrics table with health status
                            var metricsTable = document.getElementById('providerMetricsTable');
                            if (data.ProviderMetrics && Object.keys(data.ProviderMetrics).length > 0) {
                                var tableHtml = '<table><thead><tr><th>Health</th><th>Provider Name</th><th>Total Requests</th><th>Success Rate</th><th>Tokens/sec</th><th>Min (ms)</th><th>P50 (ms)</th><th>P95 (ms)</th><th>P99 (ms)</th><th>Max (ms)</th><th>Avg (ms)</th></tr></thead><tbody>';

                                // Separate providers and models
                                var providers = [];
                                var models = {};

                                for (var key in data.ProviderMetrics) {
                                    var metric = data.ProviderMetrics[key];
                                    if (metric.IsModel) {
                                        // This is a model - group under its provider
                                        if (!models[metric.Name]) {
                                            models[metric.Name] = [];
                                        }
                                        models[metric.Name].push(metric);
                                    } else {
                                        // This is a provider
                                        providers.push(metric);
                                    }
                                }

                                // Sort providers alphabetically
                                providers.sort(function(a, b) {
                                    return a.Name.localeCompare(b.Name);
                                });

                                // Render each provider and its models
                                for (var i = 0; i < providers.length; i++) {
                                    var provider = providers[i];
                                    var health = healthData[provider.Name];
                                    var providerSuccessRate = 0;
                                    if (provider.TotalRequests > 0) {
                                        providerSuccessRate = ((provider.SuccessfulRequests / provider.TotalRequests) * 100).toFixed(1);
                                    }

                                    // Determine health icon
                                    var healthIcon;
                                    if (provider.TotalRequests === 0 || !health || !health.LastChecked) {
                                        // Provider not used yet - show ?
                                        healthIcon = '<span style="color: #9e9e9e; font-size: 1.2em;">?</span>';
                                    } else if (health.IsHealthy) {
                                        healthIcon = '<span style="color: #4caf50; font-size: 1.2em;">✓</span>';
                                    } else {
                                        healthIcon = '<span style="color: #f44336; font-size: 1.2em;">✗</span>';
                                    }

                                    tableHtml += '<tr>' +
                                        '<td style="text-align: center;">' + healthIcon + '</td>' +
                                        '<td><strong>' + provider.Name + '</strong></td>' +
                                        '<td>' + (provider.TotalRequests || 0) + '</td>' +
                                        '<td>' + providerSuccessRate + '%</td>' +
                                        '<td>' + (provider.AvgTokensPerSec ? provider.AvgTokensPerSec.toFixed(0) : '-') + '</td>' +
                                        '<td>' + formatDuration(provider.MinLatency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.P50Latency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.P95Latency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.P99Latency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.MaxLatency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.AvgLatency || 0) + '</td>' +
                                        '</tr>';

                                    // Render models for this provider (sorted by AvgLatency - fastest first)
                                    if (models[provider.Name]) {
                                        // Sort models by average latency (fastest first)
                                        models[provider.Name].sort(function(a, b) {
                                            // Put models with 0 latency at the end
                                            if (a.AvgLatency === 0 && b.AvgLatency === 0) return 0;
                                            if (a.AvgLatency === 0) return 1;
                                            if (b.AvgLatency === 0) return -1;
                                            return a.AvgLatency - b.AvgLatency;
                                        });

                                        for (var j = 0; j < models[provider.Name].length; j++) {
                                            var model = models[provider.Name][j];
                                            var modelSuccessRate = 0;
                                            if (model.TotalRequests > 0) {
                                                modelSuccessRate = ((model.SuccessfulRequests / model.TotalRequests) * 100).toFixed(1);
                                            }

                                            tableHtml += '<tr>' +
                                                '<td></td>' + // No health icon for models
                                                '<td style="padding-left: 30px; color: #9e9e9e;">↳ ' + model.Model + '</td>' +
                                                '<td>' + (model.TotalRequests || 0) + '</td>' +
                                                '<td>' + modelSuccessRate + '%</td>' +
                                                '<td>' + (model.AvgTokensPerSec ? model.AvgTokensPerSec.toFixed(0) : '-') + '</td>' +
                                                '<td>' + formatDuration(model.MinLatency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.P50Latency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.P95Latency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.P99Latency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.MaxLatency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.AvgLatency || 0) + '</td>' +
                                                '</tr>';
                                        }
                                    }
                                }

                                tableHtml += '</tbody></table>';
                                metricsTable.innerHTML = tableHtml;
                            } else {
                                metricsTable.innerHTML = '<div class="loading">No provider metrics available</div>';
                            }
                        })
                        .catch(function(error) {
                            console.error('Error fetching health status:', error);
                            // If health fetch fails, just show metrics with "?" for all health
                            var metricsTable = document.getElementById('providerMetricsTable');
                            if (data.ProviderMetrics && Object.keys(data.ProviderMetrics).length > 0) {
                                var tableHtml = '<table><thead><tr><th>Health</th><th>Provider Name</th><th>Total Requests</th><th>Success Rate</th><th>Tokens/sec</th><th>Min (ms)</th><th>P50 (ms)</th><th>P95 (ms)</th><th>P99 (ms)</th><th>Max (ms)</th><th>Avg (ms)</th></tr></thead><tbody>';

                                // Separate providers and models
                                var providers = [];
                                var models = {};

                                for (var key in data.ProviderMetrics) {
                                    var metric = data.ProviderMetrics[key];
                                    if (metric.IsModel) {
                                        if (!models[metric.Name]) {
                                            models[metric.Name] = [];
                                        }
                                        models[metric.Name].push(metric);
                                    } else {
                                        providers.push(metric);
                                    }
                                }

                                providers.sort(function(a, b) {
                                    return a.Name.localeCompare(b.Name);
                                });

                                for (var i = 0; i < providers.length; i++) {
                                    var provider = providers[i];
                                    var providerSuccessRate = 0;
                                    if (provider.TotalRequests > 0) {
                                        providerSuccessRate = ((provider.SuccessfulRequests / provider.TotalRequests) * 100).toFixed(1);
                                    }

                                    var healthIcon = '<span style="color: #9e9e9e; font-size: 1.2em;">?</span>';

                                    tableHtml += '<tr>' +
                                        '<td style="text-align: center;">' + healthIcon + '</td>' +
                                        '<td><strong>' + provider.Name + '</strong></td>' +
                                        '<td>' + (provider.TotalRequests || 0) + '</td>' +
                                        '<td>' + providerSuccessRate + '%</td>' +
                                        '<td>' + (provider.AvgTokensPerSec ? provider.AvgTokensPerSec.toFixed(0) : '-') + '</td>' +
                                        '<td>' + formatDuration(provider.MinLatency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.P50Latency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.P95Latency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.P99Latency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.MaxLatency || 0) + '</td>' +
                                        '<td>' + formatDuration(provider.AvgLatency || 0) + '</td>' +
                                        '</tr>';

                                    if (models[provider.Name]) {
                                        // Sort models by average latency (fastest first)
                                        models[provider.Name].sort(function(a, b) {
                                            if (a.AvgLatency === 0 && b.AvgLatency === 0) return 0;
                                            if (a.AvgLatency === 0) return 1;
                                            if (b.AvgLatency === 0) return -1;
                                            return a.AvgLatency - b.AvgLatency;
                                        });

                                        for (var j = 0; j < models[provider.Name].length; j++) {
                                            var model = models[provider.Name][j];
                                            var modelSuccessRate = 0;
                                            if (model.TotalRequests > 0) {
                                                modelSuccessRate = ((model.SuccessfulRequests / model.TotalRequests) * 100).toFixed(1);
                                            }

                                            tableHtml += '<tr>' +
                                                '<td></td>' +
                                                '<td style="padding-left: 30px; color: #9e9e9e;">↳ ' + model.Model + '</td>' +
                                                '<td>' + (model.TotalRequests || 0) + '</td>' +
                                                '<td>' + modelSuccessRate + '%</td>' +
                                                '<td>' + (model.AvgTokensPerSec ? model.AvgTokensPerSec.toFixed(0) : '-') + '</td>' +
                                                '<td>' + formatDuration(model.MinLatency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.P50Latency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.P95Latency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.P99Latency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.MaxLatency || 0) + '</td>' +
                                                '<td>' + formatDuration(model.AvgLatency || 0) + '</td>' +
                                                '</tr>';
                                        }
                                    }
                                }

                                tableHtml += '</tbody></table>';
                                metricsTable.innerHTML = tableHtml;
                            }
                        })
                })
                .catch(function(error) {
                    console.error('Error fetching metrics:', error);
                    document.getElementById('totalRequests').innerHTML = 'Error';
                    document.getElementById('successfulRequests').innerHTML = 'Error';
                    document.getElementById('failedRequests').innerHTML = 'Error';
                    document.getElementById('fallbackAttempts').innerHTML = 'Error';
                    document.getElementById('successRate').innerHTML = 'Error';
                    document.getElementById('providerMetricsTable').innerHTML = '<div class="error">Failed to load provider metrics</div>';
                });
        }

        function updateTimestamp() {
            var now = new Date();
            var timestamp = now.toLocaleTimeString() + '.' + now.getMilliseconds().toString().padStart(3, '0');
            document.getElementById('lastUpdate').innerHTML = 'Last updated: ' + timestamp;
        }
        
        function updateAll() {
            updateMetrics();
            updateTimestamp();
        }
        
        updateAll();
        setInterval(updateAll, 2000);
    </script>
</body>
</html>`))
}