package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTest() {
	store = NewMetricStore()
	logger = nil
	// Use a discard logger for tests
	logger = newTestLogger()
}

func newTestLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestHealthHandler(t *testing.T) {
	setupTest()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got '%s'", resp.Status)
	}
	if resp.Service != "collector" {
		t.Errorf("expected service 'collector', got '%s'", resp.Service)
	}
}

func TestHealthHandlerMethodNotAllowed(t *testing.T) {
	setupTest()

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestIngestHandler(t *testing.T) {
	setupTest()

	body := IngestRequest{
		ServiceName: "test-service",
		Metrics: []MetricPoint{
			{Name: "cpu", Value: 65.5},
			{Name: "memory", Value: 80.2},
		},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ingestHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp IngestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Accepted != 2 {
		t.Errorf("expected 2 accepted, got %d", resp.Accepted)
	}
	if resp.ServiceID != "test-service" {
		t.Errorf("expected service_id 'test-service', got '%s'", resp.ServiceID)
	}
}

func TestIngestHandlerMissingServiceName(t *testing.T) {
	setupTest()

	body := IngestRequest{
		Metrics: []MetricPoint{{Name: "cpu", Value: 50}},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ingestHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestIngestHandlerEmptyMetrics(t *testing.T) {
	setupTest()

	body := IngestRequest{
		ServiceName: "test-service",
		Metrics:     []MetricPoint{},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ingestHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestIngestHandlerEmptyMetricName(t *testing.T) {
	setupTest()

	body := IngestRequest{
		ServiceName: "test-service",
		Metrics:     []MetricPoint{{Name: "", Value: 50}},
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ingestHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestIngestHandlerInvalidJSON(t *testing.T) {
	setupTest()

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ingestHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestMetricsHandler(t *testing.T) {
	setupTest()

	store.Add("test-service", []MetricPoint{
		{Name: "cpu", Value: 50},
		{Name: "memory", Value: 75},
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics?service=test-service", nil)
	w := httptest.NewRecorder()
	metricsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["service"] != "test-service" {
		t.Errorf("expected service 'test-service', got '%v'", resp["service"])
	}
	if int(resp["count"].(float64)) != 2 {
		t.Errorf("expected count 2, got %v", resp["count"])
	}
}

func TestMetricsHandlerMissingService(t *testing.T) {
	setupTest()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	metricsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestMetricStore(t *testing.T) {
	s := NewMetricStore()

	if s.TotalCount() != 0 {
		t.Error("expected empty store")
	}

	s.Add("svc1", []MetricPoint{{Name: "cpu", Value: 50}})
	s.Add("svc2", []MetricPoint{{Name: "mem", Value: 75}, {Name: "disk", Value: 30}})

	if s.TotalCount() != 3 {
		t.Errorf("expected 3 total, got %d", s.TotalCount())
	}

	svc1Metrics := s.Get("svc1")
	if len(svc1Metrics) != 1 {
		t.Errorf("expected 1 metric for svc1, got %d", len(svc1Metrics))
	}

	svc2Metrics := s.Get("svc2")
	if len(svc2Metrics) != 2 {
		t.Errorf("expected 2 metrics for svc2, got %d", len(svc2Metrics))
	}
}
