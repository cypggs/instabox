"use client";

import { useEffect, useState } from "react";
import { api, Machine, Snapshot } from "../lib/api";

export default function SnapshotsPanel({ machine, notify, refresh }: {
  machine: Machine;
  notify: (msg: string, ok?: boolean) => void;
  refresh: () => void;
}) {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const id = machine.id;
  const running = machine.state === "running";

  const load = async () => {
    try {
      const res = await api.get(`/v1/snapshots?machineId=${id}`);
      setSnapshots(res.snapshots || []);
    } catch { }
  };
  useEffect(() => { load(); const t = setInterval(load, 5000); return () => clearInterval(t); }, [id]);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.post(`/v1/machines/${id}/snapshots`, { name: name || undefined });
      notify("Snapshot in progress — it captures a live (running) machine without stopping it");
      setName("");
      setTimeout(load, 4000);
    } catch (err: any) {
      notify(`${err.code}: ${err.message}`, false);
    } finally { setBusy(false); }
  }

  async function restore(s: Snapshot) {
    if (!window.confirm(`Restore machine from snapshot "${s.name}"? Current disk contents will be replaced.`)) return;
    try {
      await api.post(`/v1/machines/${id}/restore`, { snapshotId: s.id, desiredState: "stopped" });
      notify("Restore started — machine will come back stopped with the snapshot contents");
      refresh();
      setTimeout(load, 4000);
    } catch (err: any) { notify(`${err.code}: ${err.message}`, false); }
  }

  async function fork(s: Snapshot) {
    const count = +(window.prompt("How many copies?", "1") || "0");
    if (!count) return;
    try {
      await api.post(`/v1/machines/${id}/fork`, { name: s.name + "-copy", count });
      notify(`${count} fork(s) requested`);
      refresh();
    } catch (err: any) { notify(`${err.code}: ${err.message}`, false); }
  }

  async function remove(s: Snapshot) {
    if (!window.confirm(`Delete snapshot "${s.name}"?`)) return;
    try {
      await api.del(`/v1/snapshots/${s.id}`);
      notify("Snapshot deleted");
      load();
    } catch (err: any) { notify(err.message, false); }
  }

  return (
    <div>
      <div className="dim small" style={{ marginBottom: 14 }}>
        Snapshots capture the full disk (<span className="mono">/</span> and <span className="mono">/data</span>) of a live machine without stopping it.
        Restore replaces the machine disk; fork creates new machines from a snapshot.
      </div>
      <form onSubmit={create} style={{ display: "flex", gap: 8, marginBottom: 18, maxWidth: 480 }}>
        <input placeholder="snapshot name (optional)" value={name} onChange={(e) => setName(e.target.value)} disabled={!running || busy} />
        <button type="submit" className="primary" disabled={!running || busy}>{busy ? "Saving…" : "Take snapshot"}</button>
      </form>
      {!running && <div className="dim small" style={{ marginBottom: 12 }}>Machine must be running to snapshot (live snapshot).</div>}
      {snapshots.length === 0 ? <div className="dim">No snapshots yet.</div> : (
        <table className="panel">
          <thead><tr><th>Name</th><th>Created</th><th>Seq</th><th>Size</th><th /></tr></thead>
          <tbody>
            {snapshots.map((s) => (
              <tr key={s.id}>
                <td className="mono small">{s.name}</td>
                <td className="dim small">{new Date(s.createdAt).toLocaleString()}</td>
                <td className="mono small dim">#{s.sourceSnapshotSeq}</td>
                <td className="mono small dim">{s.sizeBytes ? (s.sizeBytes / 1048576).toFixed(1) + " MiB" : "—"}</td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  <button className="ghost" onClick={() => restore(s)}>Restore</button>
                  <button className="ghost" onClick={() => fork(s)}>Fork</button>
                  <button className="ghost" style={{ color: "var(--red)" }} onClick={() => remove(s)}>✕</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
