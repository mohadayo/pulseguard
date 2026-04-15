import request from "supertest";
import { app } from "./index";

describe("Gateway Service", () => {
  describe("GET /health", () => {
    it("should return 200 with health status", async () => {
      const res = await request(app).get("/health");
      expect(res.status).toBe(200);
      expect(res.body.status).toBe("healthy");
      expect(res.body.service).toBe("gateway");
      expect(res.body.uptime_seconds).toBeDefined();
      expect(res.body.timestamp).toBeDefined();
      expect(res.body.downstream).toBeDefined();
      expect(res.body.downstream.collector).toBeDefined();
      expect(res.body.downstream.analyzer).toBeDefined();
    });
  });

  describe("POST /api/ingest", () => {
    it("should return 502 when collector is unavailable", async () => {
      const res = await request(app)
        .post("/api/ingest")
        .send({
          service_name: "test",
          metrics: [{ name: "cpu", value: 50 }],
        });
      expect(res.status).toBe(502);
      expect(res.body.error).toContain("Collector service unavailable");
    });
  });

  describe("POST /api/analyze", () => {
    it("should return 502 when analyzer is unavailable", async () => {
      const res = await request(app)
        .post("/api/analyze")
        .send({ metrics: [10, 20, 30] });
      expect(res.status).toBe(502);
      expect(res.body.error).toContain("Analyzer service unavailable");
    });
  });

  describe("GET /api/metrics", () => {
    it("should return 400 when service param missing", async () => {
      const res = await request(app).get("/api/metrics");
      expect(res.status).toBe(400);
      expect(res.body.error).toContain("service");
    });

    it("should return 502 when collector is unavailable", async () => {
      const res = await request(app).get("/api/metrics?service=test");
      expect(res.status).toBe(502);
      expect(res.body.error).toContain("Collector service unavailable");
    });
  });

  describe("GET /api/status", () => {
    it("should return gateway status even when downstream is unreachable", async () => {
      const res = await request(app).get("/api/status");
      expect(res.status).toBe(200);
      expect(res.body.gateway).toBe("healthy");
      expect(res.body.services.collector).toBe("unreachable");
      expect(res.body.services.analyzer).toBe("unreachable");
    });
  });
});
