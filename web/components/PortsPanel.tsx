"use client";

import { useEffect, useState } from "react";
import { api, Port } from "../lib/api";

export default function PortsPanel({ machineId, running, notify }: { machineId: string; running: boolean; notify: (msg: string, ok?: boolean) => void }) {
  const [ports, setPorts] = useState<Port[]>([]);
  const [port, setPort] = useState("");
  const [access, setAccess] = useState<"public" | "private">("public");
  const [busy, setBusy] = useState(false);

  const load = async () => {
    try {
      const res = await api.get(`/v1/machines/${machineId}/ports`);
      setPorts(res.ports || []);
    } catch { /* machine may not be running */ }
  };
  useEffect(() => { load(); const t = setInterval(load, 5000); return () => clearInterval(t); }, [machineId]);

  async function publish(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      const res = await api.post(`/v1/machines/${machineId}/ports`, { port: +port, access });
      notify(`Port ${port} published`);
      if (res.tokenUrl) window.prompt("Private URL (shown once — copy it):", res.tokenUrl);
      setPort("");
      load();
    } catch (err: any) {
      notify(`${err.code}: ${err.message}`, false);
    } finally { setBusy(false); }
  }

  async function unpublish(p: Port) {
    try {
      await api.del(`/v1/machines/${machineId}/ports/${p.port}`);
      notify(`Port ${p.port} unpublished`);
      load();
    } catch (err: any) { notify(err.message, false); }
  }

  return (
    <div>
      <div className="dim small" style={{ marginBottom: 14 }}>
        Expose a TCP port running inside the sandbox on a stable URL.
        {running ? "" : " Machine must be running to publish ports."}
      </div>
      <form onSubmit={publish} style={{ display: "flex", gap: 8, marginBottom: 18, maxWidth: 480 }}>
        <input type="number" min={1024} max={65535} placeholder="8080" value={port} onChange={(e) => setPort(e.target.value)} disabled={!running} required style={{ width: 120 }} />
        <select value={access} onChange={(e) => setAccess(e.target.value as any)} disabled={!running} style={{ width: 120 }}>
          <option value="public">public</option>
          <option value="private">private</option>
        </select>
        <button type="submit" className="primary" disabled={!running || busy || !port}>Publish</button>
      </form>
      {ports.length === 0 ? (
        <div className="dim" style={{ padding: "20px 0" }}>No published ports.</div>
      ) : (
        <table className="panel">
          <thead><tr><th>Port</th><th>Access</th><th>URL</th><th /></tr></thead>
          <tbody>
            {ports.map((p) => (
              <tr key={p.port}>
                <td className="mono">{p.port}</td>
                <td><span className="badge">{p.access}</span></td>
                <td className="mono small">
                  {p.url ? <a href={p.url} target="_blank" rel="noreferrer">{p.url}</a> : <span className="dim">—</span>}
                </td>
                <td style={{ textAlign: "right" }}><button className="ghost" style={{ color: "var(--red)" }} onClick={() => unpublish(p)}>Unpublish</button></td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
