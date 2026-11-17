> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Sizes and create options

> Pick a machine's vCPUs and memory, a template version, and every other field create takes.

A machine's size is the number of vCPUs and the amount of memory it has. A machine is
4 vCPU / 8 GiB unless you ask for another size of your plan when you create it. You can pick a
larger size to give a build or a test run more CPUs and memory.

This page also covers how to pin a template version, how to resize a machine later, and every other
field that create takes.

```bash theme={null}
instabox create --name big --vcpus 8 --mem-mib 16384
```

## Choose a size

When the default size is not the one you want, you can create the machine at another size of your
plan. Name both the vCPUs and the memory in MiB. Over the API the fields are `vcpus` and `memMiB` on
`POST /v1/machines`.

```bash CLI theme={null}
instabox create --name big --vcpus 8 --mem-mib 16384
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create({
  name: 'big',
  vcpus: 8,
  memMiB: 16384,
})
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "big", "vcpus": 8, "memMiB": 16384}'
```

| Plan | Sizes (vCPUs / memory) | Default |
| - | - | - |
| Free | 4 / 8 GiB | 4 vCPU / 8 GiB |
| Pro | 2 / 4 GiB, 4 / 8 GiB, 8 / 16 GiB | 4 vCPU / 8 GiB |
| Builder | 2 / 4 GiB, 4 / 8 GiB, 8 / 16 GiB, 16 / 32 GiB | 4 vCPU / 8 GiB |
| Team | 2 / 4 GiB, 4 / 8 GiB, 8 / 16 GiB, 16 / 32 GiB, 32 / 64 GiB | 4 vCPU / 8 GiB |

## List your workspace's sizes

To see which sizes a workspace can use, run `instabox workspace ls --json` (`GET /v1/orgs`), which
lists each workspace's sizes as `allowedSpecs`. Check the list before you ask for a size, because the
server refuses any other size with `400 spec_not_allowed`.

```bash theme={null}
instabox workspace ls --json
```

## Pin a template version

A new machine uses the version of its template that is marked `latest`. When you need a machine from
one exact version, you can pin it by adding the version to the template name after an `@`. The
`--template-version` flag names the version too, and over the API the field is `templateVersion` on
`POST /v1/machines`.

```bash CLI theme={null}
instabox create --name job \
  --template agent-browser@2026-10-15.4
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "job", "templateId": "agent-browser",
       "templateVersion": "2026-10-15.4"}'
```

## Resize a machine

When a machine on a paid plan needs more or less than it has, you can move it to another size of its
plan. A free machine's size is fixed. Run the resize on a stopped machine, and the new size applies
at the next start.

```bash CLI theme={null}
instabox resize big --vcpus 4 --mem-mib 8192
```

```bash API theme={null}
curl -X PATCH https://api.instabox.sh/v1/machines/m_... \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"vcpus": 4, "memMiB": 8192}'
```

## Resize and restart in one step

If you do not want to stop the machine yourself, you can let the resize do it for you. Add
`--restart`, and the command stops the machine, resizes it, and starts it again.

```bash theme={null}
instabox resize big --vcpus 4 --mem-mib 8192 --restart
```

> [accordion] When the start at the new size fails
>
> The start fails with `no_capacity` if the host has no room for the new size, or `host_unavailable` if
> its node agent cannot resize yet. The machine stays stopped at the new size.

## Reference

### Create parameters

This table lists every field that `POST /v1/machines` takes, with its name in the SDKs and the CLI.
All of the fields are optional, so `{}` is a valid body. A key that is not in the table is refused
with `400 unknown_field`, and so is `region`, because the platform decides where a machine runs.

| Field | Type | Default | Range | SDK and CLI names |
| - | - | - | - | - |
| `name` | string | `box-` and 8 random hex digits | 1-63 of `a-z`, `0-9`, `-`, starting with a letter or digit | TypeScript: `name` · Python: `name` · CLI: `--name` |
| `templateId` | string | `agent-browser` | a template with `creatable` true, optionally `id@version` | TypeScript: first argument (`Sandbox.create('agent-browser')`) · Python: `template` · CLI: `--template` |
| `templateVersion` | string | the version `GET /v1/templates` marks `latest` | - | CLI: `--template-version` |
| `snapshotId` | string | none: a template | - | TypeScript: first argument (a snapshot ID or name) · Python: `snapshot_id` · CLI: `--snapshot` |
| `imageId` (deprecated) | string | none; the old name of `snapshotId` | - | - |
| `vcpus` | integer | the default size | 1 or more; with `memMiB`, one of the workspace's `allowedSpecs` | TypeScript: `vcpus` · Python: `vcpus` · CLI: `--vcpus` |
| `memMiB` | integer | the default size | 1 or more; with `vcpus`, one of the workspace's `allowedSpecs` | TypeScript: `memMiB` · Python: `mem_mib` · CLI: `--mem-mib` |
| `persistentDiskGiB` | integer | the size's disk (or a snapshot's, when larger) | 1 or more; that size only; grow it later with `PATCH` | TypeScript: `diskGiB` · Python: `persistent_disk_gib` |
| `rootDiskGiB` | integer | the template's | 1 or more; that size only (admins grow it) | TypeScript: `rootDiskGiB` |
| `desiredState` | string | `running` | `running` or `stopped` | Python: `desired_state` |
| `metadata` | object | none | 32 keys of 1-64 `[A-Za-z0-9_.-]`, values up to 512 characters, 8 KiB as JSON | TypeScript: `metadata` · Python: `metadata` · CLI: `--meta key=value` |
| `archiveSeconds` | integer | none (0: none) | 60 to 86400 | TypeScript: `archiveMs` (milliseconds) · Python: `archive_seconds` |
| `deleteAfterSeconds` | integer | never (0: never) | 60 to 86400 | Python: `delete_after_seconds` · CLI: `--delete-after` (a duration) |
| `maxRunMinutes` | integer | the plan's ([Idle pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)) | -1 to 2147483647 | CLI: `--max-run-minutes` |
| `autoSuspendAfterSeconds` | integer | 1800 (30 minutes) | the plan's choices, or 0 (off) where the plan allows ([Auto-pause](https://instabox.sh/docs/machines/auto-pause.md)) | TypeScript: `autoSuspendMs` (milliseconds) · Python: `auto_suspend_after_seconds` · CLI: `--auto-suspend` (a duration) |
| `vault` | boolean | `true` | - | TypeScript: `vault` · CLI: `--no-vault` (false) |
| `secrets` | "all" or string\[] | none (all when only `envSecrets` is sent) | - | CLI: `--secrets` |
| `envSecrets` | string\[] | `[]` | each `^[A-Za-z_][A-Za-z0-9_]{0,127}$` | - |
| `directSecrets` | object\[] | none | 64 references to direct Secrets; real values readable in the sandbox | CLI: `--secret-env VAR=SECRET`, `--secret-file PATH=SECRET` |
| `env` | object | none | 128 names of `^[A-Za-z_][A-Za-z0-9_]{0,127}$`, not `PATH`, `HOME`, the other shell names, `INSTABOX_*` or `SSH_*`; 32 KiB in all; not on Windows; readable in the sandbox, not for secrets | CLI: `--env KEY=VALUE`, `--env-file` |
| `pool` | string | your pool assignment (admins only) | - | - |
| `guestUser` | string | `user` (from a snapshot: its user) | `^[a-z_][a-z0-9_-]{0,31}$`; at most 32 characters | CLI: `--user` |
| `failFast` | boolean | `false` | - | Python: `fail_fast` · CLI: `--fail-fast` |
| `acceptMinimumLease` | boolean | - | required for a macOS template: a whole Mac for at least 24 hours; Pro and up | Python: `accept_minimum_lease` · CLI: `--accept-24h-minimum` |
| `model` | string | the template's own model | a `name` of the macOS template's `models` | Python: `model` · CLI: `--model` |
| `gpu` | string | none: no GPU | `TYPE` or `TYPE:COUNT` of the `gpu` template's `gpuTypes` (e.g. `l4`, `l4:2`); paid plans only, 20 GPUs per org | CLI: `--gpu` |
| `orgId` | string | your personal org (from a snapshot: its org) | - | Python: `org_id` · CLI: `--org` |

> [accordion] Names that differ between REST and the SDKs
>
> The TypeScript SDK's `diskGiB` and the Python SDK's `persistent_disk_gib` are the REST field
> `persistentDiskGiB`. `diskGiB` in a REST body is `400 unknown_field`. The TypeScript SDK takes times
> in milliseconds (`archiveMs`, `autoSuspendMs`). The last column lists only the clients that set the
> field, and `-` means none does.

> [accordion] Retired lifetime timeout
>
> There is no lifetime timeout (retired 2026-10-20). `timeoutSeconds`, `onTimeout`, or `timeout` is
> `400 unknown_field`. The message points to `deleteAfterSeconds`, `maxRunMinutes`, and
> `autoSuspendAfterSeconds`. See [Storage and retention](https://instabox.sh/docs/machines/storage.md) and
> [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md).

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> When a machine pauses by itself, and what wakes it.

> [card] [Environment variables](https://instabox.sh/docs/machines/environment.md)
>
> Variables every shell in the machine gets.
