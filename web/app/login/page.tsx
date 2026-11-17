"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, setCsrf } from "../../lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("admin@ibox.local");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const res = await api.post("/v1/auth/login", { email, password });
      setCsrf(res.csrfToken || null);
      router.replace("/machines");
    } catch (err: any) {
      setError(err.message || "login failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ minHeight: "100vh", display: "flex", alignItems: "center", justifyContent: "center" }}>
      <div className="panel" style={{ width: 380, padding: 32 }}>
        <div style={{ marginBottom: 24 }}>
          <div style={{ fontSize: 20, fontWeight: 700, letterSpacing: "-0.02em" }}>
            Insta<span style={{ color: "var(--accent)" }}>Box</span> Sandbox
          </div>
          <div className="dim small" style={{ marginTop: 4 }}>Sign in to the console (clone build)</div>
        </div>
        <form onSubmit={submit}>
          <div style={{ marginBottom: 14 }}>
            <label>Email</label>
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="email" required />
          </div>
          <div style={{ marginBottom: 18 }}>
            <label>Password</label>
            <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required />
          </div>
          {error && (
            <div style={{ color: "var(--red)", fontSize: 13, marginBottom: 14 }} className="mono">{error}</div>
          )}
          <button type="submit" className="primary" style={{ width: "100%" }} disabled={busy}>
            {busy ? "Signing in…" : "Sign in"}
          </button>
        </form>
      </div>
    </div>
  );
}
