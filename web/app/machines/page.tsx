"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { api, setCsrf, Machine, Template, stateColor } from "../../lib/api";
import CreateMachineDialog from "../../components/CreateMachineDialog";

export default function MachinesPage() {
  const router = useRouter();
  const [machines, setMachines] = useState<Machine[]>([]);
  const [me, setMe] = useState<any>(null);
  const [templates, setTemplates] = useState<Template[]>([]);
  const [showCreate, setShowCreate] = useState(false);
  const [toast, setToast] = useState<{ msg: string; ok: boolean } | null>(null);

  const notify = (msg: string, ok = true) => {
    setToast({ msg, ok });
    setTimeout(() => setToast(null), 5000);
  };

  const refresh = useCallback(async () => {
    try {
      const [m, t] = await Promise.all([api.get("/v1/machines"), api.get("/v1/templates")]);
      setMachines(m.machines || []);
      setTemplates((t.templates || []).filter((x: Template) => x.creatable && x.latest));
    } catch (err: any) {
      if (err.status === 401) router.replace("/login");
      else notify(err.message, false);
    }
  }, [router]);

  useEffect(() => {
    api.get("/v1/me").then((res) => { setMe(res); setCsrf(res.csrfToken || null); }).catch(() => router.replace("/login"));
    refresh();
    const t = setInterval(refresh, 3000);
    return () => clearInterval(t);
  }, [refresh, router]);

  async function act(id: string, action: string, body?: any) {
    try {
      await api.post(`/v1/machines/${id}/${action}`, body);
      refresh();
    } catch (err: any) {
      notify(err.message, false);
    }
  }

  async function removeMachine(m: Machine) {
    if (!window.confirm(`Delete machine "${m.name}"? This cannot be undone.`)) return;
    try {
      await api.del(`/v1/machines/${m.id}`, { name: m.name });
      notify(`Machine ${m.name} deleted`);
      refresh();
    } catch (err: any) {
      notify(err.message, false);
    }
  }

  return (
    <div style={{ maxWidth: 1100, margin: "0 auto", padding: "32px 20px" }}>
      <header style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 24 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <div style={{ fontSize: 18, fontWeight: 700, letterSpacing: "-0.02em" }}>
            Insta<span style={{ color: "var(--accent)" }}>Box</span> Sandbox
          </div>
          {me && <span className="badge dim small">{me.org?.name} · {me.org?.tier}</span>}
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button className="ghost" onClick={async () => { await api.post("/v1/auth/logout"); router.replace("/login"); }}>
            Sign out
          </button>
          <button className="primary" onClick={() => setShowCreate(true)}>+ New Machine</button>
        </div>
      </header>

      <div className="panel">
        {machines.length === 0 ? (
          <div style={{ padding: 60, textAlign: "center" }} className="dim">
            No machines yet. Create your first sandbox machine — it boots in seconds.
            <div style={{ marginTop: 16 }}>
              <button className="primary" onClick={() => setShowCreate(true)}>Create a machine</button>
            </div>
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Name</th><th>Status</th><th>Template</th><th>Spec</th><th>Disk</th><th>Created</th><th style={{ textAlign: "right" }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {machines.map((m) => (
                <tr key={m.id} style={{ cursor: "pointer" }} onClick={() => router.push(`/machines/${m.id}`)}>
                  <td>
                    <div className="mono" style={{ fontWeight: 600 }}>{m.name}</div>
                    <div className="dim small mono">{m.id}</div>
                  </td>
                  <td>
                    <span className="badge">
                      <span className="dot" style={{ background: stateColor(m.state) }} />
                      {m.state}
                    </span>
                    {m.state === "suspended" && m.runtime.processesPreserved && (
                      <div className="dim small" style={{ marginTop: 3 }}>processes preserved</div>
                    )}
                  </td>
                  <td className="dim small mono">{m.template?.id || (m.snapshot ? `snap:${m.snapshot.id.slice(0, 12)}` : "—")}</td>
                  <td className="mono small">{m.spec.vcpus} vCPU · {m.spec.memMiB / 1024} GiB</td>
                  <td className="mono small dim">{m.storage.sizeGiB} GiB {m.storage.persistent ? "persist" : ""}</td>
                  <td className="dim small">{new Date(m.createdAt).toLocaleString()}</td>
                  <td onClick={(e) => e.stopPropagation()}>
                    <div style={{ display: "flex", gap: 6, justifyContent: "flex-end" }}>
                      {m.state === "running" && (
                        <>
                          <button onClick={() => act(m.id, "suspend")}>Pause</button>
                          <button onClick={() => act(m.id, "stop")}>Stop</button>
                        </>
                      )}
                      {m.state === "suspended" && <button onClick={() => act(m.id, "resume")}>Resume</button>}
                      {m.state === "stopped" && <button onClick={() => act(m.id, "start")}>Start</button>}
                      {(m.state === "running" || m.state === "stopped") && (
                        <button className="danger" onClick={() => removeMachine(m)}>Delete</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <div className="dim small" style={{ marginTop: 14 }}>
        Free plan: max 5 machines · auto-pause after 30 min idle · max run 120 min.
      </div>

      {showCreate && (
        <CreateMachineDialog
          templates={templates}
          onClose={() => setShowCreate(false)}
          onCreated={() => { setShowCreate(false); refresh(); notify("Machine created — booting"); }}
          onError={(msg) => notify(msg, false)}
        />
      )}
      {toast && <div className={`toast ${toast.ok ? "ok" : "error"}`}>{toast.msg}</div>}
    </div>
  );
}
