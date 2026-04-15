"""Tests for the Analyzer service."""

import json

import pytest

from app import app, _analyze_metrics


@pytest.fixture
def client():
    app.config["TESTING"] = True
    with app.test_client() as client:
        yield client


class TestHealth:
    def test_health_returns_200(self, client):
        resp = client.get("/health")
        assert resp.status_code == 200

    def test_health_contains_status(self, client):
        resp = client.get("/health")
        data = resp.get_json()
        assert data["status"] == "healthy"
        assert data["service"] == "analyzer"
        assert "uptime_seconds" in data
        assert "timestamp" in data


class TestAnalyze:
    def test_analyze_valid_metrics(self, client):
        resp = client.post(
            "/analyze",
            data=json.dumps({"metrics": [10, 20, 30, 40, 50]}),
            content_type="application/json",
        )
        assert resp.status_code == 200
        data = resp.get_json()
        assert data["count"] == 5
        assert data["average"] == 30.0
        assert data["min"] == 10
        assert data["max"] == 50
        assert data["sum"] == 150
        assert data["alerts"] == []

    def test_analyze_with_alerts(self, client):
        resp = client.post(
            "/analyze",
            data=json.dumps({"metrics": [50, 95, 100]}),
            content_type="application/json",
        )
        assert resp.status_code == 200
        data = resp.get_json()
        assert len(data["alerts"]) == 2
        assert data["alerts"][0]["index"] == 1
        assert data["alerts"][1]["index"] == 2

    def test_analyze_empty_body(self, client):
        resp = client.post("/analyze", content_type="application/json")
        assert resp.status_code == 400

    def test_analyze_missing_metrics(self, client):
        resp = client.post(
            "/analyze",
            data=json.dumps({"data": [1, 2, 3]}),
            content_type="application/json",
        )
        assert resp.status_code == 400

    def test_analyze_empty_metrics_list(self, client):
        resp = client.post(
            "/analyze",
            data=json.dumps({"metrics": []}),
            content_type="application/json",
        )
        assert resp.status_code == 400

    def test_analyze_invalid_metric_value(self, client):
        resp = client.post(
            "/analyze",
            data=json.dumps({"metrics": [10, "abc", 30]}),
            content_type="application/json",
        )
        assert resp.status_code == 400


class TestAnalyzeMetrics:
    def test_basic_stats(self):
        result = _analyze_metrics([10, 20, 30])
        assert result["count"] == 3
        assert result["average"] == 20.0
        assert result["min"] == 10
        assert result["max"] == 30

    def test_single_value(self):
        result = _analyze_metrics([42])
        assert result["count"] == 1
        assert result["average"] == 42.0
        assert result["min"] == 42
        assert result["max"] == 42

    def test_alert_generation(self):
        result = _analyze_metrics([50, 91, 95, 100])
        assert len(result["alerts"]) == 3
        for alert in result["alerts"]:
            assert alert["value"] > 90.0
