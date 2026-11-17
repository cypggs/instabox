> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Fork

> Copy a machine's disks into new machines on its host, in seconds.

A fork is a copy of a machine that you get right away, with the same files, template, and size. You
can use it to try a risky change without touching the original, to run several agents from one
set-up machine, or to give each CI job its own copy of a prepared machine.

A fork is made at once, on the same host as the source. When you want to keep a machine's state and
start machines from it later, on any host, save a [snapshot](https://instabox.sh/docs/machines/snapshots.md) instead.

```bash theme={null}
instabox fork dev --name dev2
```

The copy is made on the source's host, without a snapshot, and takes about as long as a create.

## Fork a machine

`fork` (or `clone`) copies a running or stopped machine into one new machine, and the command waits
until the new machine runs. The new machine has the source's template and size by default. You can
give it a different size with `--vcpus` and `--mem-mib`.

```bash CLI theme={null}
instabox fork dev --name dev2
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../fork \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" -d '{"name": "dev2"}'
```

## Fork many machines at once

When you want several copies of one machine, for example to try a few approaches side by side, add
`--count`. One command can make up to 5 new machines. This command creates `try-1`, `try-2`, and
`try-3`.

```bash theme={null}
instabox fork dev --name try --count 3 --json
```

* Forks of one source wait for each other only while their disks are copied, which takes a moment
  each, and then they start in parallel. Forks of different sources do not wait for each other.
* Each new machine counts toward your plan's running machines and machine starts a minute. See
  [Limits](https://instabox.sh/docs/account/limits.md).
* When you reach the start limit, `fork` waits it out for as long as the server's `Retry-After`
  says, up to 5 times.
* `fork` does not wait for the running limit. When your workspace cannot run `--count` more machines,
  `fork` creates none, says how many fit, and exits with code 5. You can then retry with that
  `--count`, or stop or pause some machines first.

## What a fork is

A fork is a copy-on-write copy of the source's disks, which are the system disk and `/data`. Memory
and running programs are not copied, so the new machine starts from the copied disks in the same way
as a machine that was stopped and started again. Services that are enabled as systemd units start on
their own, and you start anything else again yourself.

The source machine is not changed by a fork, and it keeps running.

## What the new machine gets

The new machine takes most of what the source has, and it gets an identity of its own:

* The source's files on the system disk and in `/data`, as they were at the fork.
* The template and the size.
* The source's kernel, if the source is pinned to one. Otherwise the fork starts on the host's
  current default kernel, which can differ from the one the source runs.
* The environment variables.
* The Secret selection. Each Secret gets a new placeholder in the new machine.
* Its own hostname, machine ID, and SSH host keys.
* The label `instabox-fork-of=<source ID>`.

A fork does not get the source's published ports, so publish them again on the new machine with
`instabox port publish`.

## List a machine's forks

Every fork carries the label `instabox-fork-of` with the source's ID as its value, so you can find
the copies of a machine later. Filter the machine list on that label to see the forks of one
machine.

```bash theme={null}
instabox ls --meta instabox-fork-of=<ID>
```

## Keep or change the source's settings

By default a fork keeps the source's auto-pause, max runtime, labels, and delete time. You can
change any of them by passing the flag for it, which is useful when the copies should not live as
long as the source. This command gives the fork its own delete time and adds a label.

```bash theme={null}
instabox fork ci-base --delete-after 30m --meta job=42
```

| Setting | CLI flag | API field |
| - | - | - |
| Auto-pause | `--auto-suspend` | `autoSuspendAfterSeconds` |
| Max runtime | `--max-run-minutes` | `maxRunMinutes` |
| Labels (added to the source's) | `--meta` | `metadata` |
| Delete time | `--delete-after` | `deleteAfterSeconds` |

A delete time that the fork keeps is the same moment as the source's, so a source that is deleted in
2 hours gives forks that are deleted in 2 hours. A source that is due for deletion within a minute is
refused unless you pass `--delete-after`.

Your plan's limits apply to these values as they do on `create`, whether you set them or the fork
kept them from the source. When a kept value is one that your plan does not allow, the fork is
refused and nothing is created. The error names the flag to pass instead.

## Limits

> [accordion] Fork a machine that is writing
>
> Before the copy, the source's file systems are frozen, which writes its cached files to disk. So the
> copy has every write that a program made before the fork, synced or not, and it mounts normally.
> Data that a program still holds in its own memory is not in the copy. A file that was half written is
> half written in the copy. The source pauses for a moment while its disks are copied, and for a few
> seconds under heavy writes. A fork through a snapshot (below) copies the disks the same way.

> [accordion] Source state
>
> * **Running or stopped:** the fork works.
> * **Paused:** refused with `409 fork_unavailable`, which is not retryable. Resume the machine first.
>   A paused machine cannot be stopped.
> * **Resuming:** refused with `409 operation_conflict`, which is retryable. Try again once it runs.
> * **Just stopped:** the fork waits until the stop has saved the disks, so it can take a few seconds
>   longer. If the save is still running after that wait, the fork is refused with
>   `409 operation_conflict`, which is retryable.
>
> A fork whose copy has started finishes, even if you delete the source. If you delete the source
> before the copy starts, the fork fails with `fork_unavailable` and nothing is created. A fork of a
> fork works too.

> [accordion] A source host under maintenance or full
>
> The new machine is placed on the source's host. If that host is under maintenance or full, nothing
> is created and the answer is a retryable `409 fork_unavailable`. Its `reason` is
> `source_host_unavailable` (maintenance) or `source_host_full` (no room). `instabox fork` then uses a
> snapshot, whose copies can start on any host. To fork natively instead, stop and start the source to
> move it to another host, then fork again.

> [accordion] When the CLI uses a snapshot instead
>
> Sometimes the server cannot fork natively: an older server, a stopped machine whose files are only
> in storage (no host), a host that cannot make the copy, or a source host that is under maintenance or
> full. Then `instabox fork` says so on stderr and saves a snapshot and creates the copies from it, as
> `--keep-snapshot` always does. This path is different:
>
> * It takes longer: a snapshot save, then a create from it.
> * The copies are ordinary creates, so they can start on any host.
> * The copies do not get the source's environment variables. They do get its size, Secret selection,
>   auto-pause, max runtime (on a paid plan), labels, and delete time, as a native fork does.
> * A running source is saved with a live snapshot, which copies the disks as a native fork does. A
>   server without live snapshots refuses it. Stop the source, then fork.
> * The snapshot is deleted afterwards, also on error or Ctrl-C, unless you pass `--keep-snapshot`.
>
> A paused source is refused on both paths.

> [card] [Parallel forks](https://instabox.sh/docs/use-cases/parallel-forks.md)
>
> Try several approaches at once from one set-up machine.

> [card] [Errors](https://instabox.sh/docs/reference/errors.md)
>
> `fork_unavailable` and every other error code.
