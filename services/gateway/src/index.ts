import express, { Request, Response, NextFunction } from "express";

const app = express();
app.use(express.json());

const PORT = parseInt(process.env.GATEWAY_PORT || "3000", 10);
const COLLECTOR_URL = process.env.COLLECTOR_URL || "http://localhost:8080";
const ANALYZER_URL = process.env.ANALYZER_URL || "http://localhost:5000";
const LOG_LEVEL = process.env.LOG_LEVEL || "INFO";

const startTime = Date.now();

function log(level: string, message: string): void {
  if (level === "DEBUG" && LOG_LEVEL !== "DEBUG") return;
  const ts = new Date().toISOString();
  console.log(`${ts} [${level}] gateway: ${message}`);
}

// Health check
app.get("/health", (_req: Request, res: Response) => {
  const uptimeSeconds = Math.round((Date.now() - startTime) / 1000);
  res.json({
    status: "healthy",
    service: "gateway",
    uptime_seconds: uptimeSeconds,
    timestamp: new Date().toISOString(),
    downstream: {
      collector: COLLECTOR_URL,
      analyzer: ANALYZER_URL,
    },
  });
});

// Proxy ingest to collector
app.post("/api/ingest", async (req: Request, res: Response) => {
  try {
    log("INFO", `Proxying ingest request to collector`);
    const response = await fetch(`${COLLECTOR_URL}/ingest`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req.body),
    });
    const data = await response.json();
    res.status(response.status).json(data);
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown error";
    log("ERROR", `Failed to proxy to collector: ${message}`);
    res.status(502).json({ error: "Collector service unavailable", detail: message });
  }
});

// Proxy analyze to analyzer
app.post("/api/analyze", async (req: Request, res: Response) => {
  try {
    log("INFO", `Proxying analyze request to analyzer`);
    const response = await fetch(`${ANALYZER_URL}/analyze`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(req.body),
    });
    const data = await response.json();
    res.status(response.status).json(data);
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown error";
    log("ERROR", `Failed to proxy to analyzer: ${message}`);
    res.status(502).json({ error: "Analyzer service unavailable", detail: message });
  }
});

// Proxy metrics to collector
app.get("/api/metrics", async (req: Request, res: Response) => {
  try {
    const service = req.query.service as string;
    if (!service) {
      res.status(400).json({ error: "'service' query parameter is required" });
      return;
    }
    log("INFO", `Proxying metrics request for service '${service}'`);
    const response = await fetch(`${COLLECTOR_URL}/metrics?service=${encodeURIComponent(service)}`);
    const data = await response.json();
    res.status(response.status).json(data);
  } catch (err) {
    const message = err instanceof Error ? err.message : "Unknown error";
    log("ERROR", `Failed to proxy to collector: ${message}`);
    res.status(502).json({ error: "Collector service unavailable", detail: message });
  }
});

// Service status endpoint
app.get("/api/status", async (_req: Request, res: Response) => {
  const results: Record<string, string> = {};

  try {
    const collectorResp = await fetch(`${COLLECTOR_URL}/health`);
    results.collector = collectorResp.ok ? "healthy" : "unhealthy";
  } catch {
    results.collector = "unreachable";
  }

  try {
    const analyzerResp = await fetch(`${ANALYZER_URL}/health`);
    results.analyzer = analyzerResp.ok ? "healthy" : "unhealthy";
  } catch {
    results.analyzer = "unreachable";
  }

  res.json({
    gateway: "healthy",
    services: results,
    timestamp: new Date().toISOString(),
  });
});

// Error handling middleware
app.use((err: Error, _req: Request, res: Response, _next: NextFunction) => {
  log("ERROR", `Unhandled error: ${err.message}`);
  res.status(500).json({ error: "Internal server error" });
});

export { app };

if (require.main === module) {
  app.listen(PORT, () => {
    log("INFO", `Gateway service started on port ${PORT}`);
    log("INFO", `Collector URL: ${COLLECTOR_URL}`);
    log("INFO", `Analyzer URL: ${ANALYZER_URL}`);
  });
}
