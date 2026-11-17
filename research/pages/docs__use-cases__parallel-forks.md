> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Parallel forks

> Set up one machine, copy it into several, try a different approach in each copy, and keep the one that works best.

When a task has more than one plausible solution, you can try them side by side and keep the one
that works best. You set up one machine with the repo, tools, and sign-ins, and then you copy it into
a few machines. An agent or a script works in each copy, and you compare the results. Every copy
starts with the same disks, so nobody repeats the setup.

**1. Set up one machine**

Create the machine that the copies are made from.

```bash theme={null}
instabox create --name base
```

The command waits until the machine runs, and then it prints the machine's row as `instabox ls`
shows it.

Go to your repo's folder on your computer. On this page, `my-repo`, `npm ci`, and `npm test`
stand for your own repo and its commands.

```bash theme={null}
cd my-repo
```

Push the folder to the machine. Because you give no destination, the folder lands in
`/data/home/my-repo`.

```bash theme={null}
instabox push base .
```

Then install the dependencies in the machine, so that every copy already has them.

```bash theme={null}
instabox exec base --cwd /data/home/my-repo -- npm ci
```

`exec` prints the command's output when the command exits.

**2. Fork it**

Copy the machine into three new machines. One command can make up to 5 copies.

```bash theme={null}
instabox fork base --name try --count 3 --json
```

This command makes `try-1`, `try-2`, and `try-3` with the same template and size, and it waits
until they run. With `--json`, it prints the new machines under `machines`. Without `--json`, it
prints their rows and the time that the fork took.

The copies have the source's disks but not its memory or running programs, and the source machine
is not changed. See [What a fork is](https://instabox.sh/docs/machines/fork.md#what-a-fork-is).

**3. Run one approach in each**

Start a different script in each copy as a background job, so that the three run at the same
time. `approach-a.sh` to `approach-c.sh` stand for your own scripts.

```bash theme={null}
instabox exec --background try-1 \
  --cwd /data/home/my-repo -- ./approach-a.sh
```

Start the other two approaches in the same way, each in its own copy.

```bash theme={null}
instabox exec --background try-2 \
  --cwd /data/home/my-repo -- ./approach-b.sh
```

```bash theme={null}
instabox exec --background try-3 \
  --cwd /data/home/my-repo -- ./approach-c.sh
```

Each `exec --background` prints the ID of the job that it started. To check on a copy, list its
jobs, which shows whether each job is running or has exited.

```bash theme={null}
instabox jobs ls try-1
```

When a coding agent should do the work, you can send a prompt to each copy instead of a script.
See [Coding agents](https://instabox.sh/docs/use-cases/coding-agents.md#run-the-agent-without-a-terminal).

```bash theme={null}
instabox agent prompt try-1 -f "..."
```

**4. Compare the copies and keep one**

When the jobs have exited, look at what each copy changed and whether its tests pass. This
command does both in `try-2`, and it prints the diff summary and then the test output.

```bash theme={null}
instabox exec try-2 --cwd /data/home/my-repo \
  -- 'git diff --stat && npm test'
```

You can also copy a result to your computer to compare it there. This command copies the `dist`
folder of `try-2` to `./dist-b`.

```bash theme={null}
instabox pull try-2 /data/home/my-repo/dist ./dist-b
```

Then delete the copies that you do not keep. `--yes` skips the question that `rm` asks first.

```bash theme={null}
instabox rm try-1 --yes
```

```bash theme={null}
instabox rm try-3 --yes
```

Each copy is a machine, so it counts toward your plan's machines and machine starts a minute. Before
you raise `--count`, see [Fork many machines at once](https://instabox.sh/docs/machines/fork.md#fork-many-machines-at-once).

A paused machine cannot be forked, so resume it first. For the other cases in which a fork is refused
or goes through a snapshot, see [Limits](https://instabox.sh/docs/machines/fork.md#limits) on the Fork page. For what the copies
keep from the source, see [What the new machine gets](https://instabox.sh/docs/machines/fork.md#what-the-new-machine-gets).

## Start the copies from a snapshot instead

A fork copies a machine now. When you want to keep the setup and start machines from it later, save
a snapshot instead. A snapshot starts any number of machines, at any time, on any host.

```bash theme={null}
instabox snapshot save base --name base-setup
```

Then create each machine from the snapshot.

```bash theme={null}
instabox create --name try-4 --snapshot base-setup
```

You can do the same from the TypeScript SDK. This code saves a snapshot of a set-up machine and
starts three machines from it at once.

```ts theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const base = await Sandbox.create('agent-browser')
// ... set it up ...
const snap = await base.createSnapshot({ name: 'base-setup' })
const tries = await Promise.all(
  [1, 2, 3].map(() => Sandbox.create(snap.id)),
)
```

> [accordion] Size, Secrets, and variables of a machine from a snapshot
>
> A fork keeps the source's size, environment variables, and Secret selection. A machine that you
> create from a snapshot gets the plan's default size unless you pass one. It also gets only the
> variables and Secrets in its own create request.

> [accordion] Fork over the API
>
> `POST /v1/machines/{id}/fork` answers the create operation. Its `machineId` is the new machine,
> and the new machine's `instabox-fork-of` label names the source.

> [card] [Snapshots](https://instabox.sh/docs/machines/snapshots.md)
>
> Save, fork, restore in place, and what a snapshot holds.

> [card] [Errors](https://instabox.sh/docs/reference/errors.md)
>
> `fork_unavailable` and every other error code.
