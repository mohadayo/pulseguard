// Package main implements the PulseGuard Collector service.
// It provides high-performance metric ingestion via a REST API.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type MetricPoint struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp,omitempty"`
}

type IngestRequest struct {
	ServiceName string        `json:"service_name"`
	Metrics     []MetricPoint `json:"metrics"`
}

type IngestResponse struct {
	Accepted  int    `json:"accepted"`
	ServiceID string `json:"service_id"`
	Message   string `json:"message"`
}

type HealthResponse struct {
	Status        string `json:"status"`
	Service       string `json:"service"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Timestamp     string `json:"timestamp"`
	StoredMetrics int    `json:"stored_metrics"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type MetricStore struct {
	mu      sync.RWMutex
	metrics map[string][]MetricPoint
}

func NewMetricStore() *MetricStore {
	return &MetricStore{
		metrics: make(map[string][]MetricPoint),
	}
}

func (s *MetricStore) Add(serviceName string, points []MetricPoint) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range points {
		if points[i].Timestamp == "" {
			points[i].Timestamp = now
		}
	}
	s.metrics[serviceName] = append(s.metrics[serviceName], points...)
	return len(points)
}

func (s *MetricStore) Get(serviceName string) []MetricPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]MetricPoint, len(s.metrics[serviceName]))
	copy(result, s.metrics[serviceName])
	return result
}

func (s *MetricStore) TotalCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, v := range s.metrics {
		count += len(v)
	}
	return count
}

var (
	store     *MetricStore
	startTime time.Time
	logger    *log.Logger
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}
	uptime := int64(time.Since(startTime).Seconds())
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:        "healthy",
		Service:       "collector",
		UptimeSeconds: uptime,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		StoredMetrics: store.TotalCount(),
	})
}

func ingestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}

	var req IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("WARN: Failed to decode request body: %v", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Invalid JSON body"})
		return
	}

	if req.ServiceName == "" {
		logger.Println("WARN: Missing service_name")
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "'service_name' is required"})
		return
	}

	if len(req.Metrics) == 0 {
		logger.Println("WARN: Empty metrics list")
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "'metrics' must be a non-empty list"})
		return
	}

	for _, m := range req.Metrics {
		if m.Name == "" {
			logger.Println("WARN: Metric with empty name")
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "Each metric must have a 'name'"})
			return
		}
	}

	count := store.Add(req.ServiceName, req.Metrics)
	logger.Printf("INFO: Ingested %d metrics for service '%s'", count, req.ServiceName)

	writeJSON(w, http.StatusCreated, IngestResponse{
		Accepted:  count,
		ServiceID: req.ServiceName,
		Message:   fmt.Sprintf("Successfully ingested %d metrics", count),
	})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "Method not allowed"})
		return
	}

	serviceName := r.URL.Query().Get("service")
	if serviceName == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "'service' query parameter is required"})
		return
	}

	points := store.Get(serviceName)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"service": serviceName,
		"count":   len(points),
		"metrics": points,
	})
}

func main() {
	port := getEnv("COLLECTOR_PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "INFO")

	logger = log.New(os.Stdout, "", log.LstdFlags)
	store = NewMetricStore()
	startTime = time.Now()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ingest", ingestHandler)
	mux.HandleFunc("/metrics", metricsHandler)

	logger.Printf("INFO: Starting Collector service on port %s (log_level=%s)", port, logLevel)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Fatalf("ERROR: Server failed: %v", err)
	}
}
