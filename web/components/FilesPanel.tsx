"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api, FileEntry } from "../lib/api";

export default function FilesPanel({ machineId, running, notify }: {  machineId: string;
  running: boolean;
  notify: (msg: string, ok?: boolean) => void;
}) {
  const [path, setPath] = useState("/data");
  const [entries, setEntries] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [selected, setSelected] = useState<FileEntry | null>(null);
  const [content, setContent] = useState<string | null>(null);
  const [editing, setEditing] = useState("");
  const fileInput = useRef<HTMLInputElement>(null);

  const load = useCallback(async (p: string) => {
    setLoading(true);
    try {
      const res = await api.get(`/v1/machines/${machineId}/files?path=${encodeURIComponent(p)}`);
      setEntries(res.entries || []);
      setPath(p);
      setSelected(null);
      setContent(null);
    } catch (err: any) {
      notify(`${err.code}: ${err.message}`, false);
    } finally {
      setLoading(false);
    }
  }, [machineId, notify]);

  useEffect(() => { if (running) load("/data"); }, [running, load]);

  async function openFile(f: FileEntry) {
    setSelected(f);
    setContent(null);
    try {
      const res = await fetch(
        `${(await import("../lib/api")).API_BASE}/v1/machines/${machineId}/files/content?path=${encodeURIComponent(f.path)}`,
        { credentials: "include" }
      );
      if (!res.ok) throw new Error((await res.json())?.error?.message || res.statusText);
      const text = await res.text();
      if (text.length > 256 * 1024) { notify("file too large to preview", false); return; }
      setContent(text);
      setEditing(text);
    } catch (err: any) {
      notify(err.message, false);
    }
  }

  async function writeRaw(full: string, data: Uint8Array) {
    const res = await fetch(
      `${(await import("../lib/api")).API_BASE}/v1/machines/${machineId}/files?path=${encodeURIComponent(full)}&parents=true`,
      { method: "PUT", credentials: "include", headers: { "content-type": "application/octet-stream" }, body: data as any }
    );
    if (!res.ok) {
      const body = await res.json().catch(() => null);
      throw new Error(`${body?.error?.code || res.status}: ${body?.error?.message || res.statusText}`);
    }
  }

  async function runInSandbox(cmd: string) {
    const res = await api.post(`/v1/machines/${machineId}/exec`, { cmd, timeoutSeconds: 30 });
    if (res.exitCode !== 0) throw new Error(res.stderr || `exit ${res.exitCode}`);
    return res;
  }

  async function saveFile() {
    if (!selected) return;
    try {
      await writeRaw(selected.path, new TextEncoder().encode(editing));
      setContent(editing);
      notify(`Saved ${selected.path}`);
    } catch (err: any) {
      notify(err.message, false);
    }
  }

  async function newFile() {
    const name = window.prompt("New file path under " + path + ":");
    if (!name) return;
    const full = path === "/" ? "/" + name : path + "/" + name;
    try {
      await writeRaw(full, new Uint8Array(0));
      notify(`Created ${full}`);
      load(path);
    } catch (err: any) { notify(err.message, false); }
  }

  async function newDir() {
    const name = window.prompt("New directory under " + path + ":");
    if (!name) return;
    const full = path === "/" ? "/" + name : path + "/" + name;
    try {
      await runInSandbox(`mkdir -p ${shellQuote(full)}`);
      notify("Directory created");
      load(path);
    } catch (err: any) { notify(err.message, false); }
  }

  async function removeEntry(f: FileEntry) {
    if (!window.confirm(`Delete ${f.path}?`)) return;
    try {
      await runInSandbox(`rm -rf ${shellQuote(f.path)}`);
      notify(`Deleted ${f.path}`);
      load(path);
    } catch (err: any) { notify(err.message, false); }
  }

  async function upload(file: File) {
    const full = path === "/" ? "/" + file.name : path + "/" + file.name;
    try {
      await writeRaw(full, new Uint8Array(await file.arrayBuffer()));
      notify(`Uploaded ${file.name} (${file.size} bytes)`);
      load(path);
    } catch (err: any) { notify(err.message, false); }
  }

  async function download(f: FileEntry) {
    const res = await fetch(
      `${(await import("../lib/api")).API_BASE}/v1/machines/${machineId}/files/content?path=${encodeURIComponent(f.path)}`,
      { credentials: "include" }
    );
    if (!res.ok) { notify("download failed", false); return; }
    const blob = await res.blob();
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = f.name;
    a.click();
    URL.revokeObjectURL(a.href);
  }

  if (!running) return <div className="dim" style={{ padding: 30, textAlign: "center" }}>Machine is not running.</div>;

  const parent = path === "/" ? null : path.split("/").slice(0, -1).join("/") || "/";

  return (
    <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 16 }}>
      <div>
        <div style={{ display: "flex", gap: 6, marginBottom: 10, alignItems: "center", flexWrap: "wrap" }}>
          <button className="ghost" disabled={!parent} onClick={() => parent && load(parent)}>↑ Up</button>
          <span className="mono small dim" style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis" }}>{path}</span>
          <button onClick={newFile}>+ File</button>
          <button onClick={newDir}>+ Dir</button>
          <button onClick={() => fileInput.current?.click()}>Upload</button>
          <input ref={fileInput} type="file" hidden onChange={(e) => { const f = e.target.files?.[0]; if (f) upload(f); e.target.value = ""; }} />
        </div>
        <div className="panel" style={{ maxHeight: 420, overflow: "auto" }}>
          {loading ? <div className="dim" style={{ padding: 20 }}>Loading…</div> : (
            <table>
              <tbody>
                {entries.map((f) => (
                  <tr key={f.path} style={{ cursor: "pointer", background: selected?.path === f.path ? "rgba(124,108,245,.12)" : undefined }}
                    onClick={() => f.type === "dir" ? load(f.path) : openFile(f)}>
                    <td className="mono small" style={{ width: 24 }}>{f.type === "dir" ? "📁" : f.type === "symlink" ? "🔗" : "📄"}</td>
                    <td className="mono small">
                      {f.name}
                      {f.type === "symlink" && <span className="dim"> → {f.linkTarget}</span>}
                    </td>
                    <td className="dim small mono" style={{ textAlign: "right" }}>{f.type === "file" ? (f.size ?? 0) + " B" : ""}</td>
                    <td onClick={(e) => e.stopPropagation()} style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                      {f.type === "file" && <button className="ghost small" onClick={() => download(f)}>↓</button>}
                      <button className="ghost small" style={{ color: "var(--red)" }} onClick={() => removeEntry(f)}>✕</button>
                    </td>
                  </tr>
                ))}
                {entries.length === 0 && <tr><td className="dim" style={{ padding: 20 }}>Empty directory</td></tr>}
              </tbody>
            </table>
          )}
        </div>
      </div>
      <div>
        {selected && content !== null ? (
          <>
            <div className="small dim mono" style={{ marginBottom: 8 }}>{selected.path} — {content.length} bytes</div>
            <textarea
              className="mono"
              value={editing}
              onChange={(e) => setEditing(e.target.value)}
              rows={16}
              style={{ fontSize: 12, resize: "vertical", background: "#0a0a0f" }}
            />
            <div style={{ marginTop: 8 }}>
              <button className="primary" onClick={saveFile} disabled={editing === content}>Save</button>
              <button className="ghost" style={{ marginLeft: 8 }} onClick={() => setEditing(content)}>Revert</button>
            </div>
          </>
        ) : (
          <div className="dim" style={{ padding: 30, textAlign: "center" }}>Select a file to view / edit. Files live under <span className="mono">/data</span> and persist across restarts.</div>
        )}
      </div>
    </div>
  );
}

function shellQuote(s: string): string {
  return `'${s.replace(/'/g, `'\\''`)}'`;
}
