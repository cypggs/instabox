"use client";

import { useEffect, useState, useCallback } from "react";
import { useParams, useRouter } from "next/navigation";
import { api, setCsrf, Machine, stateColor } from "../../../lib/api";
import Terminal from "../../../components/Terminal";
import ExecRunner from "../../../components/ExecRunner";
import FilesPanel from "../../../components/FilesPanel";
import MetricsPanel from "../../../components/MetricsPanel";
import PortsPanel from "../../../components/PortsPanel";
import SnapshotsPanel from "../../../components/SnapshotsPanel";
import LogsPanel from "../../../components/LogsPanel";
import SettingsPanel from "../../../components/SettingsPanel";

const TABS = ["Terminal", "Exec", "Files", "Metrics", "Ports", "Snapshots", "Logs", "Settings"] as const;
type Tab = (typeof TABS)[number];

export default function MachineDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const [machine, setMachine] = useState<Machine | null>(null);
  const [tab, setTab] = useState<Tab>("Terminal");
  const [toast, setToast] = useState<{ msg: string; ok: boolean } | null>(null);
  const notify = (msg: string, ok = true) => { setToast({ msg, ok }); setTimeout(() => setToast(null), 5000); };

  const refresh = useCallback(async () => {
    try {
      const m = await api.get(`/v1/machines/${id}`);
      setMachine(m);
    } catch (err: any) {
      if (err.status === 401) router.replace("/login");
      else if (err.status === 404) router.replace("/machines");
    }
  }, [id, router]);

  useEffect(() => {
    api.get("/v1/me").then((res) => setCsrf(res.csrfToken || null)).catch(() => router.replace("/login"));
    refresh();
    const t = setInterval(refresh, 3000);
    return () => clearInterval(t);
  }, [refresh, router]);

  async function act(action: string, body?: any, okMsg?: string) {
    try {
      await api.post(`/v1/machines/${id}/${action}`, body);
      if (okMsg) notify(okMsg);
      refresh();
    } catch (err: any) {
      notify(`${err.code}: ${err.message}${err.retryAfterSeconds ? ` (retry in ${err.retryAfterSeconds}s)` : ""}`, false);
    }
  }

  if (!machine) return <div style={{ padding: 60 }} className="dim">Loading…</div>;

  const running = machine.state === "running";
  const needsWake = machine.state === "suspended" || machine.state === "stopped";

  return (
    <div style={{ maxWidth: 1100, margin: "0 auto", padding: "24px 20px" }}>
      <button className="ghost" onClick={() => router.push("/machines")}>← Machines</button>
      <div className="panel" style={{ marginTop: 12 }}>
        <div style={{ padding: "18px 24px", display: "flex", alignItems: "center", justifyContent: "space-between", flexWrap: "wrap", gap: 12 }}>
          <div>
            <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
              <span style={{ fontSize: 17, fontWeight: 700 }} className="mono">{machine.name}</span>
              <span className="badge"><span className="dot" style={{ background: stateColor(machine.state) }} />{machine.state}</span>
              {machine.runtime.processesPreserved && <span className="badge dim">processes preserved</span>}
            </div>
            <div className="dim small mono" style={{ marginTop: 4 }}>
              {machine.id} · {machine.spec.vcpus} vCPU · {machine.spec.memMiB / 1024} GiB · {machine.storage.sizeGiB} GiB {machine.storage.mountPath}
              {machine.template ? ` · ${machine.template.id}:${machine.template.version}` : ""}
            </div>
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            {machine.state === "running" && <>
              <button onClick={() => act("suspend", undefined, "Pausing (suspend to disk)…")}>Pause</button>
              <button onClick={() => act("restart")}>Restart</button>
              <button onClick={() => act("stop")}>Stop</button>
            </>}
            {machine.state === "suspended" && <button className="primary" onClick={() => act("resume")}>Resume</button>}
            {machine.state === "stopped" && <button className="primary" onClick={() => act("start")}>Start</button>}
            {(running || machine.state === "stopped" || machine.state === "error") && (
              <button className="danger" onClick={async () => {
                if (!window.confirm(`Delete "${machine.name}"? This cannot be undone.`)) return;
                try { await api.del(`/v1/machines/${machine.id}`, { name: machine.name }); router.push("/machines"); }
                catch (err: any) { notify(err.message, false); }
              }}>Delete</button>
            )}
          </div>
        </div>

        {machine.state === "error" && machine.error && (
          <div style={{ margin: "0 24px 16px", padding: "10px 14px", border: "1px solid var(--red)", borderRadius: 8, color: "var(--red)", fontSize: 13 }} className="mono">
            {machine.error.code}: {machine.error.message}
          </div>
        )}

        <div className="tabs">
          {TABS.map((t) => (
            <button key={t} className={tab === t ? "active" : ""} onClick={() => setTab(t)}>{t}</button>
          ))}
        </div>

        <div style={{ padding: 20, minHeight: 420 }}>
          {needsWake && tab !== "Logs" && tab !== "Settings" ? (
            <div style={{ padding: 60, textAlign: "center" }} className="dim">
              Machine is {machine.state}. {machine.state === "suspended" && machine.runtime.processesPreserved
                ? "Processes are preserved in the snapshot — resume to continue where you left off."
                : "Start it to use this feature."}
              <div style={{ marginTop: 14 }}>
                <button className="primary" onClick={() => act(machine.state === "suspended" ? "resume" : "start")}>
                  {machine.state === "suspended" ? "Resume" : "Start"}
                </button>
              </div>
            </div>
          ) : (
            <>
              {tab === "Terminal" && running && <Terminal machineId={machine.id} />}
              {tab === "Exec" && <ExecRunner machineId={machine.id} running={running} onActivity={() => {}} />}
              {tab === "Files" && <FilesPanel machineId={machine.id} running={running} notify={notify} />}
              {tab === "Metrics" && <MetricsPanel machineId={machine.id} />}
              {tab === "Ports" && <PortsPanel machineId={machine.id} running={running} notify={notify} />}
              {tab === "Snapshots" && <SnapshotsPanel machine={machine} notify={notify} refresh={refresh} />}
              {tab === "Logs" && <LogsPanel machineId={machine.id} />}
              {tab === "Settings" && <SettingsPanel machine={machine} notify={notify} refresh={refresh} />}
            </>
          )}
        </div>
      </div>
      {toast && <div className={`toast ${toast.ok ? "ok" : "error"}`}>{toast.msg}</div>}
    </div>
  );
}
