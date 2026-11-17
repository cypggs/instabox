"use client";

import { useEffect, useState } from "react";
import { api } from "../lib/api";

interface Op {
  id: string;
  kind: string;
  status: string;
  error?: { code: string; message: string };
  createdAt: string;
  finishedAt?: string;
}

export default function LogsPanel({ machineId }: { machineId: string }) {
  const [ops, setOps] = useState<Op[]>([]);
  const [consoleLog, setConsoleLog] = useState("");

  useEffect(() => {
    let alive = true;
    async function load() {
      try {
        const res = await api.get(`/v1/machines/${machineId}/logs`);
        if (!alive) return;
        setOps(res.operations || []);
        setConsoleLog(res.console || "");
      } catch { }
    }
    load();
    const t = setInterval(load, 4000);
    return () => { alive = false; clearInterval(t); };
  }, [machineId]);

  return (
    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
      <div>
        <div className="small dim" style={{ marginBottom: 8 }}>OPERATIONS</div>
        <div className="panel" style={{ maxHeight: 400, overflow: "auto" }}>
          {ops.length === 0 ? <div className="dim small" style={{ padding: 16 }}>No operations yet.</div> : (
            <table>
              <tbody>
                {ops.map((o) => (
                  <tr key={o.id}>
                    <td className="mono small">{o.kind}</td>
                    <td>
                      <span className="badge" style={{
                        color: o.status === "succeeded" ? "var(--green)" : o.status === "failed" ? "var(--red)" : "var(--amber)",
                      }}>{o.status}</span>
                      {o.error && <div className="dim small mono" style={{ marginTop: 3 }}>{o.error.code}: {o.error.message}</div>}
                    </td>
                    <td className="dim small" style={{ textAlign: "right" }}>{new Date(o.createdAt).toLocaleTimeString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>
      <div>
        <div className="small dim" style={{ marginBottom: 8 }}>CONSOLE</div>
        <pre className="panel mono" style={{ padding: 14, maxHeight: 400, overflow: "auto", fontSize: 12, margin: 0, whiteSpace: "pre-wrap" }}>
          {consoleLog || "No console output captured."}
        </pre>
      </div>
    </div>
  );
}
