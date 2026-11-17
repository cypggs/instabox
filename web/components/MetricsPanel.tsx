"use client";

import { useEffect, useState } from "react";
import { api, MetricPoint } from "../lib/api";

export default function MetricsPanel({ machineId }: { machineId: string }) {
  const [points, setPoints] = useState<MetricPoint[]>([]);
  const [err, setErr] = useState("");

  useEffect(() => {
    let alive = true;
    async function load() {
      try {
        const res = await api.get(`/v1/machines/${machineId}/metrics?minutes=30&limit=500`);
        if (alive) { setPoints(res.points || []); setErr(""); }
      } catch (e: any) {
        if (alive) setErr(`${e.code}: ${e.message}`);
      }
    }
    load();
    const t = setInterval(load, 10000);
    return () => { alive = false; clearInterval(t); };
  }, [machineId]);

  if (err) return <div className="dim" style={{ padding: 30, textAlign: "center" }}>{err}</div>;
  if (points.length < 2) return <div className="dim" style={{ padding: 30, textAlign: "center" }}>Collecting metrics — samples appear within a minute of the machine running.</div>;

  const W = 900, H = 180, PAD = 8;
  const latest = points[points.length - 1];
  const cpu = points.map((p, i) => [i, Math.min(100, p.cpuPercent)]);
  const mem = points.map((p, i) => [i, p.memMaxMiB ? (p.memUsedMiB / p.memMaxMiB) * 100 : 0]);

  function path(data: number[][]) {
    const max = Math.max(10, ...data.map((d) => d[1]));
    return data.map(([i, v], n) =>
      `${n === 0 ? "M" : "L"}${PAD + (i / Math.max(1, data.length - 1)) * (W - PAD * 2)},${H - PAD - (v / max) * (H - PAD * 2)}`
    ).join(" ");
  }

  return (
    <div>
      <div style={{ display: "flex", gap: 24, marginBottom: 14 }}>
        <div>
          <div className="dim small">CPU</div>
          <div className="mono" style={{ fontSize: 22, color: "var(--accent)" }}>{latest.cpuPercent.toFixed(1)}%</div>
        </div>
        <div>
          <div className="dim small">Memory</div>
          <div className="mono" style={{ fontSize: 22 }}>{latest.memUsedMiB} / {latest.memMaxMiB} MiB</div>
        </div>
        <div className="dim small" style={{ alignSelf: "flex-end", marginLeft: "auto" }}>
          last {points.length} samples · 30s interval
        </div>
      </div>
      <svg width="100%" viewBox={`0 0 ${W} ${H}`} style={{ background: "#0a0a0f", borderRadius: 8 }}>
        <path d={path(cpu)} fill="none" stroke="var(--accent)" strokeWidth="1.6" />
        <path d={path(mem)} fill="none" stroke="var(--green)" strokeWidth="1.2" opacity={0.75} />
        <text x={PAD + 4} y={PAD + 12} fill="#8b8ba3" fontSize="10" fontFamily="monospace">cpu %</text>
        <text x={PAD + 48} y={PAD + 12} fill="#22c55e" fontSize="10" fontFamily="monospace">mem %</text>
      </svg>
    </div>
  );
}
