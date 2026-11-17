> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Snapshots

> Save a machine once, start new ones from it, fork it, or restore it in place.

A snapshot saves a machine's installed software and its files, so that every machine you start from
it is already set up. You can use one to set up a development environment once and start new
machines from it later, to give several agents the same starting point, or to put a machine back to
an earlier state when something goes wrong.

When you want a copy of a machine right now, you can [fork the machine](#fork-a-machine) instead,
which does not need a snapshot.

```bash theme={null}
instabox snapshot save dev --name dev-base
```

The snapshot holds the installed software and the files under `/data`, which are your home
directory, your workspace, and your saved sign-ins (gh, Claude, Codex, git, SSH). Memory and running
processes are not saved, so a machine from a snapshot starts without the programs that were running.
A snapshot is shared with the org of its machine. In your personal org that is only you, and in a
workspace it is the workspace's members.

You can do the same in the dashboard. Choose **Save snapshot** in a machine's menu, and then choose
**New machine from it**, either right away or later on the **Snapshots** page.

> [!TIP]
>
> Set up once, reuse everywhere. In the **Playground**, start a machine and install your tools, repos,
> and agent sign-ins in its terminal. Then choose **Save as snapshot**, and every machine you start
> from it is already set up.

## Save a snapshot

You can save a machine while it runs or after you stop it, so you do not have to interrupt your
work to take a snapshot. The CLI command waits until the snapshot is saved, which means the next
command can already use it. When you want the command to return at once, add `--no-wait`.

```bash CLI theme={null}
instabox snapshot save dev --name dev-base
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.connect('dev', { resume: false })
const snap = await sbx.createSnapshot({ name: 'dev-base' })
```

```python Python theme={null}
from instabox import Client

client = Client()
client.snapshots.create("dev", "dev-base")
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../snapshots \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" -d '{"name": "dev-base"}'
```

What happens during the save depends on the state that the machine is in:

* A **stopped** machine is saved as it is.
* A **running** machine is saved live. It keeps running during the save, but the snapshot may leave
  out data that programs have not written to disk yet.
* A **running GPU machine** cannot be saved live, so the save is refused with `409` until the
  machine is stopped. Add `--restart` and the CLI stops it, saves it, and starts it again, also when
  the save fails. It prints each step on stderr. Your files are kept, but the programs that were
  running must be started again. `--restart` does not stop a machine that can be saved live, and a
  stopped machine stays stopped.
* A **paused** machine is refused with `409`, and it cannot be stopped while it is paused. To save
  it, resume it and then save it live. Where live saves are off, resume it, then stop it, and then
  save it. Your files are kept either way.

```bash theme={null}
instabox snapshot save gpu-box --name gpu-base --restart
```

You can also save only the installed software by adding `--root-only`. Machines that you start from
that snapshot have the software, but they start with an empty home and workspace.

## Start a machine from a snapshot

When you create a machine from a snapshot, the new machine starts with the snapshot's software and
files in place, so you can skip the setup. The CLI takes the snapshot's name with `--snapshot`, and
the API takes its ID as `snapshotId`.

```bash CLI theme={null}
instabox create --name dev2 --snapshot dev-base
```

```ts TypeScript theme={null}
const sbx = await Sandbox.create('dev-base', { name: 'dev2' })
```

```python Python theme={null}
client.machines.create("dev2", snapshot_id="sn_...")
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "dev2", "snapshotId": "sn_..."}'
```

A machine from a GPU machine's snapshot gets the same GPU type and, by default, the same number of
GPUs. To fan a prepared one-GPU setup out to bigger machines, ask for another count of that type:
`instabox create --name render-1 --snapshot gpu-base --gpu l4:4` (API: `"gpu": "l4:4"` with
`snapshotId`). The size follows the count, as for a new GPU machine. Another GPU type is refused.

## List your snapshots

List your snapshots to see what you have saved, for example before you start a machine from one or
delete one that you no longer need.

```bash CLI theme={null}
instabox snapshot ls
```

```ts TypeScript theme={null}
const snapshots = await Sandbox.listSnapshots()
```

```python Python theme={null}
client.snapshots.list()
```

```bash API theme={null}
curl https://api.instabox.sh/v1/snapshots \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

## Browse a snapshot's files

You can list the files in a snapshot without starting a machine from it, which lets you check what
a snapshot holds before you use it. This command lists the snapshot's `/etc` directory.

```bash theme={null}
instabox snapshot files dev-base /etc
```

## Delete a snapshot

Delete a snapshot when you no longer want to start machines from it. The machines that you already
made from it are not affected. The CLI asks before it deletes, and you can skip the question with
`--yes`.

```bash CLI theme={null}
instabox snapshot rm dev-base
```

```ts TypeScript theme={null}
await Sandbox.deleteSnapshot('sn_...')
```

```python Python theme={null}
client.snapshots.delete("sn_...")
```

```bash API theme={null}
curl -X DELETE https://api.instabox.sh/v1/snapshots/sn_... \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "X-Confirm-Delete: sn_..."
```

The reverse also holds: when you delete a machine, its snapshots are kept.

## Fork a machine

Fork a machine when you want a copy of it now and do not need to keep a snapshot for later. `fork`
(or `clone`) copies a running or stopped machine into one new machine with the same template and
size, and the command waits until the copy runs.

```bash theme={null}
instabox fork dev --name dev2
```

You can make up to 5 copies at once with `--count`, for example to try several approaches side by
side. This command creates `try-1`, `try-2`, and `try-3`.

```bash theme={null}
instabox fork dev --name try --count 3 --json
```

The server copies the disks on the machine's host and boots the copies in a few seconds, so it does
not need a snapshot. Memory and running programs are not copied, and the source machine is not
changed. To see what a fork keeps and when it uses a snapshot, read [Fork](https://instabox.sh/docs/machines/fork.md).

> [!NOTE]
>
> A paused machine cannot be forked or stopped, so resume it first. Each new machine counts toward your
> plan's machine starts a minute, and `fork` waits that limit out. Each new machine also runs, so when
> your workspace cannot run `--count` more machines, `fork` creates none and says how many fit.

> [accordion] Fork through a snapshot
>
> In three cases `fork` saves a snapshot and creates the copies from it: with an older server that has
> no native fork, with a source host that is under maintenance or full, and with `--keep-snapshot`.
> Afterwards it deletes the snapshot, also on error or Ctrl-C, unless you passed `--keep-snapshot`.
> Over the API, a native fork is `POST /v1/machines/{id}/fork`.

## Restore a machine in place

Restore a machine when you want to undo what happened on it since a snapshot, such as a broken
install or a failed experiment. `restore` puts the machine's disks back to the snapshot's. The
machine keeps its ID, name, hostname, public port URLs, and ssh-config block, so everything that
names the machine keeps working.

> [!WARNING]
>
> Everything on the root disk and `/data` since the snapshot is gone, running programs included.
> `restore` asks first, and `--yes` skips the question. To keep the machine as it is, use
> `create --snapshot` or `fork` instead.

```bash CLI theme={null}
instabox restore dev --snapshot dev-base
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../restore \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" -d '{"snapshotId": "sn_..."}'
```

When the machine is running, `restore` stops it first. The machine then boots from the restored
disks, and the command waits until it runs. The API call needs the `machines:write` scope and
answers with a `restore` operation.

When you leave out `--snapshot`, `restore` uses the snapshot that the machine was created from. You
can also add `--stopped` when you want the machine to stay stopped after the restore.

```bash theme={null}
instabox restore dev --yes --stopped
```

The machine gets new SSH host keys in a restore, so an SSH client that pinned the old keys asks you
about them again.

## Limits

* The free plan keeps 5 snapshots.
* Older snapshots that hold only installed software start machines with an empty home and
  workspace, as `--root-only` snapshots do.
* A restore is refused when the snapshot was saved with `--root-only`, or when its `/data` is larger
  than the machine's disk. Grow the disk first with `instabox resize <machine> --disk N`.

> [accordion] Saving is refused
>
> On a server without live snapshots, a save of a running machine is refused too. Stop the running
> machine and then save it, or add `--restart` (api.instabox.sh has live snapshots on, except on
> GPU machines). If the save is refused because
> the machine has unsaved changes, stop it again and then retry.

> [card] [Custom images](https://instabox.sh/docs/machines/custom-images.md)
>
> Turn a Docker image into a snapshot.

> [card] [Parallel forks](https://instabox.sh/docs/use-cases/parallel-forks.md)
>
> Run several agents from one set-up machine.
