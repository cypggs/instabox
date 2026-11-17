"use client";

import { useState } from "react";
import { api, Machine } from "../lib/api";

export default function SettingsPanel({ machine, notify, refresh }: {
  machine: Machine;
  notify: (msg: string, ok?: boolean) => void;
  refresh: () => void;
}) {
  const [name, setName] = useState(machine.name);
  const [idle, setIdle] = useState(machine.autoSuspendAfterSeconds);
  const [busy, setBusy] = useState(false);
  const id = machine.id;

  async function saveName(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.patch(`/v1/machines/${id}`, { name: name.trim().toLowerCase() });
      notify("Renamed");
      refresh();
    } catch (err: any) { notify(`${err.code}: ${err.message}`, false); }
    finally { setBusy(false); }
  }

  async function saveIdle(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api.patch(`/v1/machines/${id}`, { autoSuspendAfterSeconds: idle });
      notify("Auto-pause updated");
      refresh();
    } catch (err: any) { notify(`${err.code}: ${err.message}`, false); }
    finally { setBusy(false); }
  }

  return (
    <div style={{ maxWidth: 460 }}>
      <form onSubmit={saveName} style={{ marginBottom: 22 }}>
        <label>Machine name</label>
        <div style={{ display: "flex", gap: 8 }}>
          <input className="mono" value={name} onChange={(e) => setName(e.target.value)} pattern="[a-z0-9]([a-z0-9-]{0,62})" />
          <button type="submit" disabled={busy || name === machine.name}>Save</button>
        </div>
      </form>
      <form onSubmit={saveIdle} style={{ marginBottom: 22 }}>
        <label>Auto-pause when idle (seconds)</label>
        <div style={{ display: "flex", gap: 8 }}>
          <input type="number" step={300} min={300} max={1800} value={idle} onChange={(e) => setIdle(+e.target.value)} />
          <button type="submit" disabled={busy || idle === machine.autoSuspendAfterSeconds}>Save</button>
        </div>
      </form>
      <div className="panel" style={{ padding: 16 }}>
        <div className="small dim" style={{ lineHeight: 1.7 }}>
          <div><span className="mono">id</span>: {machine.id}</div>
          <div><span className="mono">template</span>: {machine.template ? `${machine.template.id}@${machine.template.version}` : "—"}</div>
          <div><span className="mono">image digest</span>: {machine.template?.digest || "—"}</div>
          <div><span className="mono">revision</span>: {machine.revision}</div>
          <div><span className="mono">created</span>: {new Date(machine.createdAt).toLocaleString()}</div>
          <div><span className="mono">env vars</span>: {(machine.envNames || []).join(", ") || "—"}</div>
          <div><span className="mono">max run</span>: {machine.maxRunMinutes} min (auto-extends)</div>
        </div>
      </div>
      <div className="dim small" style={{ marginTop: 16, lineHeight: 1.6 }}>
        Not built in this clone (honest gaps): desktop/VNC, SSH gateway, webhooks,
        paid plans/billing, x86_64 templates. See README for details.
      </div>
    </div>
  );
}
