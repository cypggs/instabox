"use client";

import { useState } from "react";
import { api, Template } from "../lib/api";

const SPEC_CHOICES = [
  { vcpus: 2, memMiB: 4096, label: "2 vCPU · 4 GiB" },
  { vcpus: 4, memMiB: 8192, label: "4 vCPU · 8 GiB" },
  { vcpus: 8, memMiB: 16384, label: "8 vCPU · 16 GiB" },
];
const IDLE_CHOICES = [300, 600, 900, 1200, 1500, 1800];

export default function CreateMachineDialog({ templates, onClose, onCreated, onError }: {
  templates: Template[];
  onClose: () => void;
  onCreated: (m: any) => void;
  onError: (msg: string) => void;
}) {
  const [name, setName] = useState("");
  const [templateId, setTemplateId] = useState(templates[0]?.id || "minimal");
  const [vcpus, setVcpus] = useState(4);
  const [memMiB, setMemMiB] = useState(8192);
  const [idle, setIdle] = useState(1800);
  const [envText, setEnvText] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    let env: Record<string, string> | undefined;
    if (envText.trim()) {
      env = {};
      for (const line of envText.split("\n")) {
        const t = line.trim();
        if (!t || t.startsWith("#")) continue;
        const i = t.indexOf("=");
        if (i < 1) { onError(`bad env line: ${t}`); setBusy(false); return; }
        env[t.slice(0, i).trim()] = t.slice(i + 1);
      }
    }
    try {
      const m = await api.post("/v1/machines", {
        name: name.trim().toLowerCase(),
        templateId,
        vcpus, memMiB, persistentDiskGiB: 2,
        autoSuspendAfterSeconds: idle,
        env,
      });
      onCreated(m);
    } catch (err: any) {
      onError(`${err.code}: ${err.message}${err.hint ? ` — ${err.hint}` : ""}`);
      setBusy(false);
    }
  }

  const t = templates.find((x) => x.id === templateId);

  return (
    <div className="modal-backdrop" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="modal">
        <div style={{ padding: "20px 24px", borderBottom: "1px solid var(--border)", display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div style={{ fontWeight: 700 }}>New Sandbox Machine</div>
          <button className="ghost" onClick={onClose}>✕</button>
        </div>
        <form onSubmit={submit} style={{ padding: 24 }}>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
            <div>
              <label>Name (lowercase, letters/digits/hyphen)</label>
              <input className="mono" value={name} onChange={(e) => setName(e.target.value)} pattern="[a-z0-9]([a-z0-9-]{0,62})" placeholder="my-box" required autoFocus />
            </div>
            <div>
              <label>Template</label>
              <select value={templateId} onChange={(e) => setTemplateId(e.target.value)}>
                {templates.map((x) => <option key={x.id} value={x.id}>{x.id} — {x.version}</option>)}
              </select>
            </div>
            <div>
              <label>Size</label>
              <select value={`${vcpus}/${memMiB}`} onChange={(e) => { const [v, m] = e.target.value.split("/"); setVcpus(+v); setMemMiB(+m); }}>
                {SPEC_CHOICES.map((s) => <option key={s.label} value={`${s.vcpus}/${s.memMiB}`}>{s.label}</option>)}
              </select>
            </div>
            <div>
              <label>Auto-pause when idle</label>
              <select value={idle} onChange={(e) => setIdle(+e.target.value)}>
                {IDLE_CHOICES.map((s) => <option key={s} value={s}>{s / 60} minutes</option>)}
              </select>
            </div>
          </div>
          {t && <div className="dim small" style={{ margin: "12px 0" }}>{t.description}</div>}
          <div style={{ marginTop: 8 }}>
            <label>Environment variables (one KEY=VALUE per line, optional)</label>
            <textarea className="mono" rows={3} value={envText} onChange={(e) => setEnvText(e.target.value)} placeholder={"API_KEY=sk-...\nREGION=us-east-1"} style={{ resize: "vertical" }} />
          </div>
          <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 20 }}>
            <button type="button" onClick={onClose}>Cancel</button>
            <button type="submit" className="primary" disabled={busy}>{busy ? "Creating…" : "Create machine"}</button>
          </div>
        </form>
      </div>
    </div>
  );
}
