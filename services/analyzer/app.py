"""PulseGuard Analyzer Service - Metrics analysis and alerting engine."""

import logging
import os
import time
from datetime import datetime, timezone

from flask import Flask, jsonify, request

app = Flask(__name__)

LOG_LEVEL = os.environ.get("LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("analyzer")

ALERT_THRESHOLD = float(os.environ.get("ALERT_THRESHOLD", "90.0"))
SERVICE_PORT = int(os.environ.get("ANALYZER_PORT", "5000"))

start_time = time.time()


@app.route("/health")
def health():
    uptime = time.time() - start_time
    return jsonify({
        "status": "healthy",
        "service": "analyzer",
        "uptime_seconds": round(uptime, 2),
        "timestamp": datetime.now(timezone.utc).isoformat(),
    })


@app.route("/analyze", methods=["POST"])
def analyze():
    data = request.get_json()
    if not data:
        logger.warning("Received empty request body")
        return jsonify({"error": "Request body must be JSON"}), 400

    metrics = data.get("metrics")
    if not isinstance(metrics, list) or len(metrics) == 0:
        logger.warning("Invalid metrics format: expected non-empty list")
        return jsonify({"error": "'metrics' must be a non-empty list of numbers"}), 400

    for m in metrics:
        if not isinstance(m, (int, float)):
            logger.warning("Invalid metric value: %s", m)
            return jsonify({"error": f"Invalid metric value: {m}. Must be a number."}), 400

    analysis = _analyze_metrics(metrics)
    logger.info(
        "Analyzed %d metrics: avg=%.2f, max=%.2f, alerts=%d",
        len(metrics), analysis["average"], analysis["max"], len(analysis["alerts"]),
    )
    return jsonify(analysis)


def _analyze_metrics(metrics: list[float]) -> dict:
    total = sum(metrics)
    count = len(metrics)
    average = total / count
    minimum = min(metrics)
    maximum = max(metrics)

    alerts = []
    for i, value in enumerate(metrics):
        if value > ALERT_THRESHOLD:
            alerts.append({
                "index": i,
                "value": value,
                "threshold": ALERT_THRESHOLD,
                "message": f"Metric at index {i} ({value}) exceeds threshold ({ALERT_THRESHOLD})",
            })

    return {
        "count": count,
        "average": round(average, 2),
        "min": minimum,
        "max": maximum,
        "sum": total,
        "alerts": alerts,
        "threshold": ALERT_THRESHOLD,
        "analyzed_at": datetime.now(timezone.utc).isoformat(),
    }


if __name__ == "__main__":
    logger.info("Starting Analyzer service on port %d (threshold=%.1f)", SERVICE_PORT, ALERT_THRESHOLD)
    app.run(host="0.0.0.0", port=SERVICE_PORT)
