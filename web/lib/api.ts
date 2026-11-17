// Client-side API helpers for the ibox control plane.
export const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

let csrfToken: string | null = null;

export function setCsrf(t: string | null) {
  csrfToken = t;
}

export class ApiError extends Error {
  code: string;
  status: number;
  hint?: string;
  retryAfterSeconds?: number;
  constructor(status: number, body: any) {
    super(body?.message || `HTTP ${status}`);
    this.status = status;
    this.code = body?.code || "http_error";
    this.hint = body?.hint;
    this.retryAfterSeconds = body?.retryAfterSeconds;
  }
}

async function request(path: string, opts: RequestInit = {}): Promise<any> {
  const headers: Record<string, string> = { ...(opts.headers as any) };
  if (opts.body) headers["content-type"] = "application/json";
  if (opts.method && opts.method !== "GET") headers["x-ibox-csrf"] = csrfToken || "";
  const res = await fetch(`${API_BASE}${path}`, {
    credentials: "include",
    ...opts,
    headers,
  });
  if (res.status === 204) return null;
  const text = await res.text();
  const body = text ? JSON.parse(text) : null;
  if (!res.ok) throw new ApiError(res.status, body?.error || body);
  return body;
}

export const api = {
  get: (p: string) => request(p),
  post: (p: string, body?: any) => request(p, { method: "POST", body: body === undefined ? undefined : JSON.stringify(body) }),
  put: (p: string, body?: any) => request(p, { method: "PUT", body: body === undefined ? undefined : JSON.stringify(body) }),
  patch: (p: string, body?: any) => request(p, { method: "PATCH", body: body === undefined ? undefined : JSON.stringify(body) }),
  del: (p: string, body?: any) => request(p, { method: "DELETE", body: body === undefined ? undefined : JSON.stringify(body) }),
};

export interface Machine {
  id: string;
  name: string;
  state: string;
  desiredState: string;
  revision: number;
  template?: { id: string; version: string; digest: string };
  snapshot?: { id: string };
  spec: { vcpus: number; memMiB: number; persistentDiskGiB: number; rootDiskGiB: number };
  storage: { sizeGiB: number; mountPath: string; persistent: boolean; synced: boolean; snapshotSeq: number };
  runtime: { ready: boolean; processesPreserved: boolean };
  autoSuspendAfterSeconds: number;
  maxRunMinutes: number;
  cooldownUntil?: string;
  runStartedAt?: string;
  readyAt?: string;
  guestUser: string;
  envNames?: string[];
  oomKills?: number;
  error?: { code: string; message: string };
  createdAt: string;
  updatedAt: string;
}

export interface Template {
  id: string;
  version: string;
  digest: string;
  description: string;
  arch: string;
  defaultSpec: { vcpus: number; memMiB: number; persistentDiskGiB: number; rootDiskGiB: number };
  capabilities: string[];
  hidden?: boolean;
  creatable: boolean;
  latest: boolean;
}

export interface Snapshot {
  id: string;
  name: string;
  createdAt: string;
  sourceMachineId: string;
  sourceSnapshotSeq: number;
  template?: { id: string; version: string };
  rootDigest?: string;
  dataDigest?: string;
  sizeBytes?: number;
}

export interface Port {
  port: number;
  protocol: string;
  access: string;
  url?: string;
  tokenUrl?: string;
  description?: string;
}

export interface MetricPoint {
  sampledAt: string;
  cpuPercent: number;
  memUsedMiB: number;
  memMaxMiB: number;
  rbytesPerSec?: number;
  wbytesPerSec?: number;
}

export interface FileEntry {
  name: string;
  path: string;
  type: "file" | "dir" | "symlink";
  size?: number;
  mode?: string;
  modTime?: string;
  linkTarget?: string;
}

export function stateColor(state: string): string {
  switch (state) {
    case "running": return "#22c55e";
    case "starting": case "resuming": case "creating": return "#f59e0b";
    case "stopping": case "suspending": case "deleting": return "#f59e0b";
    case "suspended": return "#6366f1";
    case "stopped": return "#94a3b8";
    case "error": return "#ef4444";
    default: return "#94a3b8";
  }
}
