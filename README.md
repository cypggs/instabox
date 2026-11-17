# InstaBox Sandbox — 完整复刻 (clone)

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-15-black?logo=next.js)](web/)

> **给每个 Agent 一台云端 Linux 机器的 Sandbox 平台** —— 本项目是对
> [InstaBox Sandbox](https://instabox.sh) 的**深度逆向 + 完整可运行复刻**。
> 逆向分析文档见 [`research/PRODUCT_ANALYSIS.md`](research/PRODUCT_ANALYSIS.md)。
>
> A working re-implementation of InstaBox Sandbox — Linux sandbox "machines" with a
> real lifecycle, built from reverse-engineering the original product. Independent
> project for research & education; **not affiliated with or endorsed by InstaBox**.

Everything here is **real and runnable**: machines are OCI sandboxes executed by
[gVisor](https://gvisor.dev) (`runsc`) inside a Linux VM (Lima on macOS), driven by a
Go control plane through a WebSocket node-agent. No mock data, no fake buttons.

## Features

| Capability | Implementation | Verified |
|---|---|---|
| Machine CRUD + lifecycle (create/start/stop/restart/delete) | desired-state reconciler → `runsc run`/`delete -force` | E2E |
| Specs: vCPU / memory / disk | cgroup v2 cpuset + quota, memory limit | `nproc`, `/proc/meminfo` |
| Web Terminal | xterm.js ↔ WS ↔ PTY (`runsc exec` + `/dev/ptmx`) | E2E roundtrip |
| Command exec (sync + SSE stream) | `runsc exec`, JSON `{stdout,stderr,exitCode}` | E2E |
| File upload/download/manage | host-side direct I/O via node agent | E2E |
| Env vars | AES-GCM encrypted at rest, injected into OCI config | E2E |
| Live snapshots + restore + rollback | CAS `tar+zstd+sha256`, content-addressed | E2E rollback |
| Pause (suspend-to-disk) / resume | `runsc checkpoint/restore --image-path`, processes preserved | E2E |
| `/data` persistent volume | survives stop/start, chown to guest uid | E2E |
| Metrics | cgroup v2 sampling, 30 s interval, `points` API | E2E |
| Published ports | reverse proxy `{port}-{machineId}.ibox.localhost`, `ibp_` tokens (hash-only) | manual |
| API keys (`ibk_`) with scopes | hashed at rest | unit |
| Idempotency keys, duplicate-name 409, quotas, cooldowns, auto-suspend, max-run | control plane | E2E / unit |

Captured runs (31/31 assertions): [`docs/TEST_REPORT.md`](docs/TEST_REPORT.md).
UI screenshots: [`research/screenshots/`](research/screenshots/) (login, machines,
web terminal with real `4.19.0-gvisor` output, files, metrics).

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Browser (Next.js dashboard, xterm.js)   :3000              │
└──────────────┬──────────────────────────────────────────────┘
               │ REST /v1 + WebSocket (terminal)  + SSE (exec)
┌──────────────▼──────────────────────────────────────────────┐
│  Control plane (Go)                     :8080 / :8081       │
│  · REST API, session cookie + CSRF, API keys (ibk_)         │
│  · SQLite (WAL) state machine + operations (idempotent)     │
│  · reconciler: desired state → transitions → cooldowns,     │
│    auto-suspend, max-run, reaper                            │
│  · published-ports reverse proxy (:8081)                    │
└──────────────┬──────────────────────────────────────────────┘
               │ WebSocket JSON-RPC + stream mux (node agent)
┌──────────────▼──────────────────────────────────────────────┐
│  Execution node — Lima VM "ibox" (Ubuntu 24.04, aarch64)    │
│  · ibox-agent: sandbox manager                              │
│  · runsc (gVisor) OCI runtime: kernel-level isolation       │
│  · per-machine rootfs copy + persistent /data volume        │
│  · CAS snapshots (tar+zstd+sha256), runsc checkpoint pause  │
└─────────────────────────────────────────────────────────────┘
```

Design docs: `docs/ARCHITECTURE.md`, `docs/DATA_MODEL.md`, `docs/API.md`.
Test evidence: `docs/TEST_REPORT.md`.

## Quick start (macOS + Lima)

Prereqs: Go 1.23+, Node 20+, [Lima](https://lima-vm.io).

```bash
# 1. execution plane VM (first boot also installs runsc + tools)
limactl start --name=ibox deploy/ibox-vm.yaml

# 2. build + install the node agent into the VM
GOOS=linux GOARCH=arm64 go build -o /tmp/ibox-agent ./cmd/agent
limactl copy /tmp/ibox-agent ibox:/tmp/ibox-agent
limactl shell ibox -- sudo install -m 0755 /tmp/ibox-agent /usr/local/bin/ibox-agent

# 3. control plane (separate terminal)
IBOX_NODE_TOKEN=dev-node-token go run ./cmd/control -db /tmp/ibox.db

# 4. agent (inside the VM)
limactl shell ibox -- sudo /usr/local/bin/ibox-agent \
  -control ws://host.lima.internal:8080/internal/nodes/connect \
  -node-id node-1 -token dev-node-token

# 5. dashboard (separate terminal)
cd web && npm install && npm run dev   # http://localhost:3000
```

Login: `admin@ibox.local` / `admin1234` (override with `IBOX_ADMIN_EMAIL` / `IBOX_ADMIN_PASSWORD`).

The first machine creation bootstraps the template rootfs with `debootstrap`
(1–3 min); later creations take seconds (per-machine reflink copy of the template).

`make vm`, `make control`, `make agent`, `make web` wrap the steps above.

## Port publishing URLs

Published ports get URLs like `http://8080-m0xxxx.ibox.localhost` served by the
control plane's port proxy on `:8081`. Map the domain locally:

```bash
echo '127.0.0.1 ibox.localhost' | sudo tee -a /etc/hosts
# access http://<port>-<machineid>.ibox.localhost:8081
```

Private ports append a one-time `?_token=ibp_…` (hash-only storage, like the original).

## E2E test

With control plane + agent running:

```bash
./scripts/e2e.sh
```

15 steps: create → running → env vars → exec → files write/read/list → terminal
WebSocket roundtrip → CPU/mem → stop → start → /data persistence → live snapshot →
restore rollback → suspend/resume with preserved processes → delete + cleanup.
See `docs/TEST_REPORT.md` for a captured run.

## Security model

- Sandboxes run under **gVisor** (`runsc`) — a userspace kernel boundary, not plain
  containers. The clone's security posture matches the product intent; `runc` is
  available as a dev fallback (`-runtime runc`) but is *not* the security boundary.
- Non-root sandbox processes (guest user `user`, `noNewPrivileges`), capability
  bounding set of 3, cgroup v2 CPU/memory/pids limits per machine.
- Filesystem isolation: per-machine rootfs copy; only `/data` persists.
- The **host Docker socket is never mounted** into sandboxes. (Docker-in-sandbox on
  the `ci` template talks to a socket inside the sandbox only.)
- Env vars are AES-GCM encrypted at rest; API-key and port tokens are stored as
  SHA-256 hashes (shown once).
- Session HMAC cookies (HttpOnly) + CSRF header on mutations; bcrypt passwords.

Known intentional deviations / honest gaps:

| Area | Status |
|---|---|
| Desktop / VNC | not built (original: Pro) |
| SSH gateway | not built (terminal + exec cover remote use) |
| Webhooks | not built |
| Billing / paid plans / quotas UI | org tier + spec gating implemented; Stripe not built |
| x86_64 templates | clone is aarch64-only (Apple Silicon Lima VM) |
| `network=host` mode | documented tradeoff for Lima single-node dev; production design is per-sandbox netns with egress proxy |

## Repository layout

```
cmd/control        control plane (REST, SQLite, reconciler, port proxy)
cmd/agent          node agent (runs inside the VM)
internal/api       HTTP handlers, auth, SSE exec, WS terminal bridge
internal/controller  desired-state reconciler, reaper, metrics
internal/store     SQLite schema + queries
internal/agent/    sandbox manager (runsc lifecycle, files, ports, CAS, metrics)
internal/nodes     node registry + JSON-RPC stream multiplexing
web/               Next.js dashboard (machines, terminal, files, …)
deploy/ibox-vm.yaml  Lima VM definition
scripts/e2e.sh     Phase-5 end-to-end test
research/          reverse-engineering notes + captured API data
docs/              ARCHITECTURE.md, DATA_MODEL.md, API.md, TEST_REPORT.md
```
