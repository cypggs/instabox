"use client";

import { useRef, useState } from "react";
import { api } from "../lib/api";

export default function ExecRunner({ machineId, running, onActivity }: {
  machineId: string;
  running: boolean;
  onActivity: () => void;
}) {
  const [cmd, setCmd] = useState("");
  const [output, setOutput] = useState("");
  const [exitCode, setExitCode] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  async function run(e?: React.FormEvent) {
    e?.preventDefault();
    if (!cmd.trim()) return;
    setBusy(true);
    setOutput("");
    setExitCode(null);
    const ac = new AbortController();
    abortRef.current = ac;
    const t0 = performance.now();
    try {
      const res = await fetch(
        `${(await import("../lib/api")).API_BASE}/v1/machines/${machineId}/exec`,
        {
          method: "POST",
          credentials: "include",
          signal: ac.signal,
          headers: { "content-type": "application/json" },
          body: JSON.stringify({ cmd: cmd, timeoutSeconds: 120, stream: true }),
        }
      );
      if (!res.ok || !res.body) {
        const body = await res.json().catch(() => null);
        setOutput(`error: ${body?.error?.code || res.status} ${body?.error?.message || ""}`);
        setBusy(false);
        return;
      }
      const reader = res.body.getReader();
      const dec = new TextDecoder();
      let code: number | null = null;
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        const chunk = dec.decode(value, { stream: true });
        // SSE lines: "data: {...}"
        for (const line of chunk.split("\n")) {
          if (!line.startsWith("data: ")) continue;
          try {
            const msg = JSON.parse(line.slice(6));
            if (msg.stdout) setOutput((o) => o + msg.stdout);
            if (msg.stderr) setOutput((o) => o + msg.stderr);
            if (typeof msg.exitCode === "number") code = msg.exitCode;
            if (msg.error) setOutput((o) => o + `\n[error] ${msg.error.code}: ${msg.error.message}`);
          } catch { setOutput((o) => o + line.slice(6)); }
        }
      }
      setExitCode(code);
      onActivity();
    } catch (err: any) {
      if (err.name !== "AbortError") setOutput(`request failed: ${err.message}`);
    } finally {
      setBusy(false);
      void t0;
    }
  }

  return (
    <div>
      <form onSubmit={run} style={{ display: "flex", gap: 8 }}>
        <span className="mono dim" style={{ alignSelf: "center" }}>$</span>
        <input
          className="mono"
          value={cmd}
          onChange={(e) => setCmd(e.target.value)}
          placeholder={running ? "uname -a" : "machine is not running"}
          disabled={!running || busy}
        />
        <button type="submit" className="primary" disabled={!running || busy || !cmd.trim()}>
          {busy ? "Running…" : "Run"}
        </button>
        {busy && <button type="button" onClick={() => abortRef.current?.abort()}>Cancel</button>}
      </form>
      <pre className="mono" style={{
        marginTop: 14, padding: 14, background: "#0a0a0f", borderRadius: 8,
        minHeight: 280, maxHeight: 420, overflow: "auto", fontSize: 12.5, lineHeight: 1.55,
        whiteSpace: "pre-wrap", wordBreak: "break-all",
      }}>
        {output || <span className="dim">Output will stream here…</span>}
      </pre>
      {exitCode !== null && (
        <div className="small mono" style={{ color: exitCode === 0 ? "var(--green)" : "var(--red)" }}>
          exit code {exitCode}
        </div>
      )}
    </div>
  );
}
