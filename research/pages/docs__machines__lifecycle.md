> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Lifecycle

> Start, pause, resume, stop, restart, restore and delete a machine.

Pause, resume, stop, start, restore and delete a machine from the CLI, the SDKs or the API. The CLI and
SDK calls wait until the change is done. An API call returns an operation at once: see
[How it works](#how-it-works). API paths take the machine ID (`m_...`). The CLI and the SDKs also
take its name.

| State | Memory and processes | Files | Billed for runtime |
| - | - | - | - |
| Running | kept | kept | yes |
| Paused | kept | kept | no |
| Stopped | gone | kept | no |
| Deleted | gone | gone | no |

## Pause a machine

Pause keeps memory and running processes, so a resume picks up where you left off.

A machine also pauses by itself when it is idle. To change when, see [Auto-pause](https://instabox.sh/docs/machines/auto-pause.md).

```bash CLI theme={null}
instabox suspend dev
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.connect('dev', { resume: false })
await sbx.pause()
```

```python Python theme={null}
from instabox import Client

client = Client()
client.machines.pause("dev")
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../suspend -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)"
```

A paused machine cannot be stopped. Resume or delete it. The TypeScript SDK's `stop()` starts a paused
machine first, then stops it.

## Resume a paused machine

```bash CLI theme={null}
instabox resume dev
```

```ts TypeScript theme={null}
const sbx = await Sandbox.connect('dev', { resume: false })
await sbx.resume()
```

```python Python theme={null}
client.machines.resume("dev")
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../resume -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)"
```

If a resume cannot bring memory back, the machine starts fresh with its files.

## Stop a machine

Stop shuts the machine down. Files stay. Processes end, and the next start boots fresh.

```bash CLI theme={null}
instabox stop dev
```

```ts TypeScript theme={null}
const sbx = await Sandbox.connect('dev', { resume: false })
await sbx.stop()
```

```python Python theme={null}
client.machines.stop("dev")
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../stop -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)"
```

## Start a stopped machine

```bash CLI theme={null}
instabox start dev
```

```ts TypeScript theme={null}
const sbx = await Sandbox.connect('dev', { resume: false })
await sbx.resume()  // starts a stopped machine
```

```python Python theme={null}
client.machines.start("dev")
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../start -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)"
```

A machine you paused or stopped stays offline until you resume or start it. A machine that paused
**itself** when idle wakes on use. See [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md).

## Restart a stuck machine

`restart` (or `recover`) power-cycles a machine that stopped answering. The root disk and `/data` are
kept. Processes and memory are not.

```bash theme={null}
instabox restart dev
```

> [accordion] How a restart stops the machine
>
> Stop asks the guest to power off, then sends Ctrl+Alt+Del. Once the timeout runs out (60 seconds, or
> `--timeout` up to 10 minutes), it kills the VM, so a hung guest comes back too. Writes the guest had
> not flushed can be lost. A machine stuck in `starting` with no start in progress cannot be
> power-cycled.

## Restore a snapshot in place

Restore puts the machine's disks back to a snapshot's. The machine keeps its ID, name, hostname, public
port URLs, and ssh-config block.

> [!WARNING]
>
> Everything on the root disk and `/data` since the snapshot is gone, running programs included. The
> SSH host keys are new afterwards.

```bash CLI theme={null}
instabox restore dev --snapshot before-upgrade
```

```python Python theme={null}
client.machines.restore("dev", "sn_...")  # a snapshot ID
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../restore -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" -d '{"snapshotId": "sn_..."}'
```

Restore stops a running machine first, then boots it from the snapshot. To leave it stopped, add
`--stopped` (Python: `stopped=True`, API: `"desiredState": "stopped"`). To start a **new** machine from a snapshot instead, see
[Snapshots](https://instabox.sh/docs/machines/snapshots.md).

## Delete a machine

Delete removes the machine and its files. Snapshots are kept.

```bash CLI theme={null}
instabox rm dev
```

```ts TypeScript theme={null}
const sbx = await Sandbox.connect('dev', { resume: false })
await sbx.kill()
```

```python Python theme={null}
client.machines.delete("dev", confirm="dev")  # refuses to delete anything but dev
```

```bash API theme={null}
curl -X DELETE https://api.instabox.sh/v1/machines/m_... -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "X-Confirm-Delete: m_..."
```

## Reboot or shut down from inside the machine

| Inside the machine | What happens |
| - | - |
| `sudo reboot`, `systemctl reboot`, `shutdown -r` | Starting, then Running again on the same disks, with new processes |
| `sudo poweroff`, `halt`, `shutdown -h` | Stopped until you start it |

> [accordion] Older templates and reboot loops
>
> Machines created from a template older than this behavior stop on a reboot too. A machine that
> reboots more than 5 times in 10 minutes is left stopped.

## How it works

Every change to a machine is an **operation**, and a machine runs at most one at a time. The
dashboard shows operations in the operations tray, and these states: Creating, Starting, Running,
Stopping, Stopped, Pausing, Paused, Resuming, Deleting, Failed.

Over the API, lifecycle calls answer `202 Accepted` with an operation. Follow the operation with Get an
operation until it is `succeeded` or `failed`, as
[API overview](https://instabox.sh/docs/api-reference/introduction.md#changes-run-as-operations) describes. The CLI and the SDKs
do this for you.

> [card] [Sizes and create options](https://instabox.sh/docs/machines/sizes.md)
>
> vCPUs, memory, disk, templates, and every create field.

> [card] [Storage and retention](https://instabox.sh/docs/machines/storage.md)
>
> What is kept, for how long, and the archive and delete deadlines.
