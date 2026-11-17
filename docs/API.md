# ibox API

Base URL: `http://localhost:8080` (control plane). All paths below are implemented by
the clone and were exercised by `scripts/e2e.sh` (see `TEST_REPORT.md`).

## Auth

Two mechanisms, like the original:

1. **Session** (dashboard): `POST /v1/auth/login` → HttpOnly cookie `ibox_session` +
   `csrfToken` in the body. Mutating requests must send `x-ibox-csrf: <token>`.
2. **API key**: `Authorization: Bearer ibk_...` — keys created via `/v1/keys`, scoped
   (`machines:read`, `machines:write`, `snapshots`, `ports`, …). Stored as SHA-256 hash.

Errors use a uniform shape:

```json
{
  "error": {
    "code": "machine_not_running",
    "message": "machine is stopped or paused; start or resume it first",
    "requestId": "req_…",
    "retryable": false,
    "hint": "…",
    "retryAfterSeconds": 0,
    "operationId": "op_…",
    "docsUrl": "…"
  }
}
```

## Endpoints

### Auth / identity
| Method | Path | Notes |
|---|---|---|
| POST | `/v1/auth/login` | `{email, password}` → session cookie + `csrfToken` |
| POST | `/v1/auth/logout` | invalidates session |
| GET | `/v1/me` | user, org, `allowedSpecs`, idle choices |
| POST | `/v1/me/terms` | accept terms (bootstrap) |
| GET/POST/DELETE | `/v1/keys`, `/v1/keys/{id}` | API key management (`ibk_` shown once) |

### Orgs / usage / templates
| Method | Path | Notes |
|---|---|---|
| GET | `/v1/orgs` | orgs + `allowedSpecs` + quotas |
| GET | `/v1/usage` | period usage (free-tier math) |
| GET | `/v1/templates` | template catalog (minimal / standard / ci; Pro ones hidden) |

### Machines
| Method | Path | Notes |
|---|---|---|
| GET | `/v1/machines` | list (org-scoped) |
| POST | `/v1/machines` | create, `202` async. `Idempotency-Key` supported. Body: `name`, `templateId`, `vcpus`, `memMiB`, `persistentDiskGiB`, `autoSuspendAfterSeconds`, `env{…}`, `snapshotId` (restore-from). Unknown fields rejected (`unknown_field`). |
| GET | `/v1/machines/{id}` | machine incl. `state`, `desiredState`, `runtime{ready, processesPreserved}`, `storage`, cooldown |
| PATCH | `/v1/machines/{id}` | rename / resize (free tier: stop first) / auto-pause / metadata |
| DELETE | `/v1/machines/{id}` | `202`; API keys must send `X-Confirm-Delete: <id>` |
| POST | `/v1/machines/{id}/start` `…/stop` `…/suspend` `…/resume` `…/restart` `…/wake` | lifecycle; suspend keeps processes (checkpoint); cooldown → `409 machine_cooldown` + `retryAfterSeconds` |
| POST | `/v1/machines/{id}/exec` | `{cmd | cmdv, cwd, envs, stdin, timeoutSeconds≤600, background, stream}` → `{stdout, stderr, exitCode}`; `stream:true` (or `Accept: text/event-stream`) → SSE |
| GET | `/v1/machines/{id}/terminal` | WebSocket PTY (binary frames = keystrokes, `{"cols","rows"}` JSON text = resize) |
| GET | `/v1/machines/{id}/files?path=` | list dir |
| PUT | `/v1/machines/{id}/files?path=&parents=true` | write file (raw body, 5 GiB cap) |
| GET | `/v1/machines/{id}/files/content?path=` | read file (`application/octet-stream`) |
| POST | `/v1/machines/{id}/files/upload-url` | stub — not implemented (honest gap) |
| GET/POST | `/v1/machines/{id}/ports` | list / publish `{port, access: public\|private}` → private returns one-time `tokenUrl` |
| DELETE | `/v1/machines/{id}/ports/{port}` | unpublish |
| GET | `/v1/machines/{id}/metrics?since=&minutes=` | `{"points":[{at,cpuPercent,memUsedMiB,memMaxMiB}]}`, 30 s sampling interval |
| POST | `/v1/machines/{id}/snapshots` | live snapshot `{name}` → `202`, quota 5 |
| POST | `/v1/machines/{id}/restore` | `{snapshotId, desiredState}` |
| POST | `/v1/machines/{id}/fork` | `{name, count}` new machines from latest state |
| GET | `/v1/machines/{id}/logs` | console + operations |
| GET | `/v1/machines/{id}/history` | exec history |
| GET | `/v1/machines/{id}/processes` | in-sandbox `ps` |
| POST | `/v1/machines/{id}/activity-leases` | `{ttlSeconds}` — anti-auto-pause lease |
| GET | `/v1/machines/{id}/usage` | machine-level usage |
| GET | `/v1/machines/{id}/connection-info` | connection metadata |

### Snapshots / secrets / meta
| Method | Path | Notes |
|---|---|---|
| GET | `/v1/snapshots?machineId=` | list |
| DELETE | `/v1/snapshots/{id}` | delete (CAS garbage) |
| GET/PUT/DELETE | `/v1/secrets`, `/v1/secrets/{name}` | org secrets (encrypted at rest) |
| GET | `/v1/operations/{id}` | operation status (202 Accepted flows) |
| GET | `/v1/openapi.json` | OpenAPI document of the implemented surface |
| GET | `/healthz` | liveness |

## Node agent (internal)

Agents connect outbound: `GET /internal/nodes/connect?node_id=…&hostname=…`
(WebSocket, `Authorization: Bearer <node token>`). JSON-RPC with multiplexed streams:

- requests: `{id, method, params, machine, stream}`
- methods: `sandbox.create|start|stop|suspend|resume|restart|delete|fork`,
  `sandbox.exec`, `sandbox.ps`, `sandbox.logs`, `sandbox.metrics`,
  `files.list|read|write`, `ports.publish|unpublish|lookup`,
  `snapshot.save|restore|list`, `templates.ensure`
- stream frames: `{"stream": id, "data": base64}` / `{"stream": id, "close": true}`
  (terminal PTY, exec I/O)
- agent events: `machine.state`, `metrics.batch`

## Published-port URLs

The proxy on `:8081` serves `http://{port}-{machineId}.ibox.localhost` (and
`:8080` under `/portproxy/`). Private ports require `?_token=ibp_…` (or the
`ibox_port_auth` cookie set on first use).
