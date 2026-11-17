> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Metrics and usage

> CPU and memory samples, and runtime and storage usage.

All of these need `machines:read`. `Sandbox.usage` also works with `images:read`.

```ts theme={null}
// the last 15 minutes of CPU and memory
const samples = await sbx.getMetrics({
  since: new Date(Date.now() - 15 * 60_000),
})
// what it used since the start of October (UTC)
const usage = await Sandbox.getUsage(sbx.sandboxId, {
  from: new Date('2026-10-01T00:00:00Z'),
})
console.log(usage.runningSeconds, usage.vcpuSeconds)
```

## getMetrics

Get a machine's resource samples: one per 30 s, oldest first, measured on the host.

```ts theme={null}
sbx.getMetrics(opts?: SandboxMetricsOpts): Promise<SandboxMetrics[]>
Sandbox.getMetrics(sandboxId: string, opts?: SandboxMetricsOpts & ConnectionOptions): Promise<SandboxMetrics[]>
```

`since` (a `Date`) is the start of the window:
by default 5 minutes ago, at most 24 h back.

* There are samples only for the times in the window when the machine ran. A machine that is paused or
  stopped now still returns the samples from when it ran. A machine that did not run in the window gets
  an empty list.
* The last sample of a running machine is its host's latest report (at most about 30 s old, CPU and
  memory only), so a new machine shows its first sample within about 30 s.
* The CLI prints the same: `instabox metrics <id-or-name> [--json]`.

| Field | Type | What |
| - | - | - |
| `timestamp` | `Date` | When the sample was taken |
| `cpuUsedPct` | `number` | CPU time over the machine's vCPUs, 0 to 100 |
| `cpuCount` | `number` | vCPUs |
| `memUsed` | `number` | Guest memory touched, in bytes |
| `memTotal` | `number` | Memory size, in bytes |
| `diskTotal` | `number` | Disk (`/data`) size, in bytes |
| `diskUsed` | `number` | Only on the latest sample of a running machine: bytes used on `/data`, read inside the machine (refreshed in the background at most once a minute, at most 10 minutes old) |

For `/data` and `/` with a fullness level, use `GET /v1/machines/{id}/disk` or `instabox disk <id-or-name>`
([CLI reference](https://instabox.sh/docs/reference/cli.md#disk)).

## getUsage

Get what a machine used between `from` and `to` (both `Date`s), for billing your own customers.

```ts theme={null}
sbx.getUsage(opts?: SandboxUsageOpts): Promise<SandboxUsage>
Sandbox.getUsage(sandboxId: string, opts?: SandboxUsageOpts & ConnectionOptions): Promise<SandboxUsage>
```

The default range is the current UTC month: `from` is the start of the month, and `to` is now. `to` is at
most 366 days after `from`. A deleted machine's usage stays readable by its account, by ID. Running time is
recorded from the deploy of 2026-10-14. Earlier time is not counted.

| Field | Type | What |
| - | - | - |
| `from`, `to` | `Date` | The window |
| `runningSeconds` | `number` | Seconds running, not paused or stopped |
| `vcpuSeconds` | `number` | vCPUs times running seconds |
| `memGiBSeconds` | `number` | Memory GiB times running seconds |
| `diskGiBSeconds` | `number` | Disk (`/data`) size times the seconds it existed in the window, paused or stopped included |
| `egressBytes` | `number` | Bytes sent to the public internet, by whole hour (every hour from the one `from` falls in up to `to`), billed or not. Absent from older servers |

## Sandbox.usage

Get your counts, stored bytes and limits (`GET /v1/usage`).

```ts theme={null}
Sandbox.usage(opts?: ConnectionOptions): Promise<Usage>
```

Bytes are measured about hourly.

```ts theme={null}
interface Usage {
  computedAt: string | null // when bytes were measured
  logicalBytes: number
  storedBytes: number
  exclusiveBytes: number // deprecated: always 0
  machines: number
  snapshots: number
  limits: UsageLimits // maxMachines, maxSnapshots, maxStoredGiB,
                      // machineSpec?, liveSnapshots?, vault?
  storage?: UsageStorage // storedBytes by machine
}
```

`storage` has `machines` (each with `id`, `name`, `state`, `sizeBytes`, `diskBytes`, `snapshots` and
`snapshotBytes`), `deletedMachineSnapshots` and `deletedMachineSnapshotBytes`. `computedAt` is null and
`storage` absent before your first measurement.
