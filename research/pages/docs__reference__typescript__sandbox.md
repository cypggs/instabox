> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Sandbox

> Create, connect to, list, pause, resume, stop and delete machines, and their snapshots.

## Create, connect, list and kill

### Sandbox.create

Create a machine and resolve once it runs.

```ts theme={null}
Sandbox.create(opts?: SandboxOpts): Promise<Sandbox>
Sandbox.create(template: string, opts?: SandboxOpts): Promise<Sandbox>
```

`template` is a template ID (`agent-browser`), a private
snapshot name or a snapshot ID (`sn_...`). A template ID wins over a snapshot name of the same text. The default
is `agent-browser` (`DEFAULT_TEMPLATE`), or `universal` (`FALLBACK_TEMPLATE`) on a server without it. A
snapshot that is still being saved is waited for, up to `operationTimeoutMs`.

`SandboxOpts` is the [connection options](https://instabox.sh/docs/reference/sdk.md#connectionoptions) plus:

| Option | Type | Default |
| - | - | - |
| `name` | `string` | `sbx-<random>`. Lowercase letters, digits and `-` |
| `metadata` | `Record<string, string>` | none. Labels to filter `Sandbox.list` on |
| `envs` | `Record<string, string>` | none. Added to every `commands.run` of this object (client side) |
| `archiveMs` | `number` | none. Sent as the API's `archiveSeconds`. This many ms after create, the server **stops** a machine that still runs (`archiveAt`). Disks stay, and a later start restores it. It is not a delete. Leave it out for no archive deadline |
| `autoSuspendMs` | `number` | 30 minutes. Pause after this much idle time. A paid plan takes 5-30 minutes in whole minutes, and `0` for never. A part minute or a value outside that range is refused. The free plan always pauses after 30 idle minutes. A server from 2026-10-25 refuses another value on the free plan (`403 spec_not_allowed`) instead of ignoring it |
| `vcpus` | `number` | the template's |
| `memMiB` | `number` | the template's |
| `diskGiB` | `number` | the size's. The disk (`/data`), in GiB, sent as the API's `persistentDiskGiB` |
| `vault` | `boolean` | `false` turns off only personal Vault delivery. Org Secret selections are independent |

Times are sent as whole seconds.

* **Size.** A machine is 4 vCPU / 8 GiB unless `vcpus` and `memMiB` name another size of your plan
  ([Sizes](https://instabox.sh/docs/machines/sizes.md)). `GET /v1/orgs` lists each workspace's sizes as `allowedSpecs`. Any other
  size is `400 spec_not_allowed`. So is a `diskGiB` other than the disk the size comes with, so leave
  `diskGiB` out.
* **Resizing later.** A paid machine's disk grows with `PATCH /v1/machines/{id}`. Its size (`vcpus`,
  `memMiB`) changes with the same `PATCH` while it is stopped. In a REST body the disk field is
  `persistentDiskGiB`, and `diskGiB` there is `400 unknown_field`.
* **Templates.** The free plan creates only from agent-browser, android, omarchy, ci. The other templates need a paid
  plan and are `403 template_not_allowed` on free. A machine from a snapshot is not limited.

> [accordion] No kernel tracing, no kernel modules, no nested KVM
>
> The guest kernel has the eBPF syscall and JIT for root only: unprivileged BPF is off, so use `sudo`.
> Socket, cgroup and XDP programs load. The kernel lacks the tracing side: no kprobes, no ftrace or
> tracepoints, and no BTF (`/sys/kernel/btf/vmlinux` is absent). So `bpftrace`, BCC tools and
> `perf probe` do not work.
>
> There is also no `/dev/kvm` in the guest, because the kernel is a KVM guest, not a KVM host. So
> Firecracker, QEMU with KVM, Docker-in-VM runtimes that need KVM, and nested microVMs cannot run. QEMU in
> pure emulation (`-accel tcg`) works, slowly.
>
> There are no loadable kernel modules. The kernel is built with `CONFIG_MODULES` off, so `insmod`
> and `modprobe` fail with `Function not implemented` (ENOSYS). There is no `linux-headers` package
> for it, so DKMS and other out-of-tree modules cannot be built.

### Sandbox.connect

Attach to an existing machine, and resume it if it is paused or stopped.

```ts theme={null}
Sandbox.connect(sandboxId: string, opts?: SandboxConnectOpts): Promise<Sandbox>
```

`resume: false` attaches without waking the machine. `envs` works as in `create`.

### Sandbox.list

List your machines, every page of them.

```ts theme={null}
Sandbox.list(opts?: SandboxListOpts): Promise<SandboxInfo[]>
```

`metadata` keeps only the machines that have every given key and value.

### Sandbox.kill

Delete a machine and its disks.

```ts theme={null}
Sandbox.kill(sandboxId: string, opts?: ConnectionOptions): Promise<boolean>
```

Returns `false` if the machine did not exist.

## Snapshots

[`sbx.createSnapshot`](#sbx-createsnapshot) saves a snapshot. See [Snapshots](https://instabox.sh/docs/machines/snapshots.md).

### Sandbox.listSnapshots

List your private snapshots.

```ts theme={null}
Sandbox.listSnapshots(opts?: ConnectionOptions): Promise<Snapshot[]>
```

### Sandbox.getSnapshot

Get one snapshot.

```ts theme={null}
Sandbox.getSnapshot(snapshotId: string, opts?: ConnectionOptions): Promise<Snapshot>
```

### Sandbox.deleteSnapshot

Delete a snapshot.

```ts theme={null}
Sandbox.deleteSnapshot(snapshotId: string, opts?: ConnectionOptions): Promise<boolean>
```

Returns `false` if the snapshot was already gone.

## The Sandbox object

Lifecycle calls (`create`, `connect`, `pause`, `resume`, `stop`, `kill`) wait for the server's operation to
finish, up to `operationTimeoutMs`.

| Member | Returns |
| - | - |
| `sandboxId` | `string`: the machine ID, `m_...` |
| `commands` | `Commands`: [run shell commands](https://instabox.sh/docs/reference/typescript/commands.md) |
| `files` | `Files`: [read and write files](https://instabox.sh/docs/reference/typescript/files.md) |
| `desktop` | `Desktop`: [see and control the desktop](https://instabox.sh/docs/reference/typescript/desktop.md) |

### sbx.getInfo

Get the machine's current state from the server.

```ts theme={null}
getInfo(): Promise<SandboxInfo>
```

A `SandboxInfo` has `sandboxId`, `templateId`, `snapshotId?`, `name`,
`metadata`, `vault`, `startedAt`, `archiveAt?`, `autoSuspendMs?`, `state` and `machine` (the raw API
object). `state` is one of:

```
creating  starting  running  stopping  stopped
suspending  suspended  resuming  deleting  deleted  error
```

A paused machine is `suspended`. The `vault` field describes personal Vault delivery. Org Secret
selections and delivery status are at `GET /v1/machines/{id}/secrets`, independent of Vault. The
[Secrets CLI and HTTP API](https://instabox.sh/docs/security/secrets.md) manage them.

### sbx.isRunning

Check whether the machine runs.

```ts theme={null}
isRunning(): Promise<boolean>
```

Returns `true` if `state` is `running`.

### sbx.pause

Save memory and pause the machine.

```ts theme={null}
pause(): Promise<void>
```

### sbx.resume

Resume a paused machine, or start a stopped one.

```ts theme={null}
resume(): Promise<void>
```

### sbx.stop

Shut the machine down: processes end, and `/data` is kept.

```ts theme={null}
stop(): Promise<void>
```

### sbx.setAutoSuspend

Set how long the machine stays idle before it pauses.

```ts theme={null}
setAutoSuspend(idleMs: number): Promise<void>
```

A paid plan takes 5-30 minutes in whole minutes, and `0` turns the idle pause
off. A part minute or a value outside that range is refused. The free plan keeps 30
minutes and refuses another value. The setting is kept across pause and resume. See
[Auto-pause](https://instabox.sh/docs/machines/auto-pause.md).

### sbx.setMetadata

Replace the machine's labels.

```ts theme={null}
setMetadata(metadata: Record<string, string>): Promise<void>
```

`{}` clears them.

### sbx.createSnapshot

Save the machine's disks as a private snapshot.

```ts theme={null}
createSnapshot(opts: CreateSnapshotOpts): Promise<Snapshot>
```

`CreateSnapshotOpts` has `name` (required), `description`, and `rootOnly` to save only the root disk.
From a running machine, the snapshot comes back with `status: 'pending'` and turns `ready` once saved.

A machine created from the snapshot is a copy of the disks:

```ts theme={null}
const snap = await sbx.createSnapshot({ name: 'my-setup' })
const copy = await Sandbox.create(snap.id, { vault: false })
```

The copy uses the selections of its own create request, not the source machine's grants. Leaving out
`secrets` does not enroll the copy in Secrets. `vault: false` turns off only Vault delivery. To select
Secrets explicitly, use the CLI or the HTTP API ([Secrets](https://instabox.sh/docs/security/secrets.md)).

### sbx.sshCredentials

Get a short-lived SSH certificate for `publicKey`.

```ts theme={null}
sshCredentials(publicKey: string, opts?: { ttlSeconds?: number }): Promise<SshCredentials>
```

The machine must be running. `ttlSeconds` is 1 to 86 400 (default 1 hour). `SshCredentials` has `host`, `port`, `username` (the machine ID), `certificate`
(save it as `<key>-cert.pub`), `knownHostsLine`, `expiresAt` and `connectionId`. See [SSH](https://instabox.sh/docs/access/ssh.md).

### sbx.kill

Delete the machine and its disks.

```ts theme={null}
kill(): Promise<boolean>
```

Returns `false` if the machine was already gone.
