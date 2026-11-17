> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# CLI reference

> Every instabox command and flag, grouped by what it works on.

Every command of the `instabox` CLI, grouped by what it works on. Add `--help` to any command for its
details. `instabox help <command>` prints the same. `<id-or-name>` is a machine's ID (`m_...`) or name.

```bash theme={null}
npx instabox@latest login
instabox create --name dev
instabox exec dev -- uname -a
```

## Sign in

### login

Sign in with a device code, or sign up as an agent with no browser.

```
login [--provider github|google]
login --agent [--name N] [--scope S ...] [--expires-in-days N] [--no-wait]
```

`--agent` signs up an agent with no browser. It prints a claim page and a code for the account owner,
waits for their approval, and stores the API key they grant. See [Agent sign-up](https://instabox.sh/docs/security/agent-sign-up.md).

### claim

Approve an agent that printed `<code>`. You must be signed in.

```
claim <code> [--scope S ...] [--expires-in-days N] [--deny] [--yes]
```

`--deny` refuses the agent instead.

### logout

Revoke and forget the stored credentials.

```
logout
```

### whoami

Show the signed-in user.

```
whoami
```

## Create and inspect machines

### templates

List the templates a machine can start from.

```
templates [--json]
```

### ls

List your machines. `list` works too.

```
ls [--meta K=V ...] [--json]
```

`--meta K=V` keeps only machines with that metadata. With several `--meta` flags, a machine must match
all of them, so each key may be given once (a repeated key is a usage error). The server does the filtering (`GET /v1/machines?metadata=K:V`), and only your own orgs'
machines are listed. An agent can tag what it creates with `create --meta session=ID` and find those
machines later with `ls --meta session=ID`.

### create

Create a machine.

```
create --name N [--template T | --snapshot S] [--org WS]
       [--auto-suspend 10m|0] [--delete-after 1h] [--meta K=V ...]
       [--no-wait] [--fail-fast] [--setup-file PATH|-] [--wait-setup]
       [--env K=V ...] [--env-file PATH|-] [--devcontainer [PATH|DIR]]
       [--secrets all|none|NAMES] [--no-vault] [--ready-timeout D] [--json]
```

| Flag | What it does |
| - | - |
| `--template`, `--snapshot` | What the machine starts from. [Sizes and create options](https://instabox.sh/docs/machines/sizes.md) |
| `--org` | Put the machine in a workspace instead of your personal org |
| `--auto-suspend` | Pause once idle that long. `0` turns it off. [Auto-pause](https://instabox.sh/docs/machines/auto-pause.md) |
| `--delete-after` | Delete the machine after that long |
| `--meta K=V` | Metadata labels, to filter `ls` on |
| `--env`, `--env-file` | Environment variables for the whole machine. They are readable inside it, so they are not for secrets. [Environment](https://instabox.sh/docs/machines/environment.md) |
| `--fail-fast` | When no host has room, fail at once and create nothing |
| `--setup-file` | Run a script (at most 64 KB) once the machine runs. A failed script exits with its code and keeps the machine. `get --json` shows `metadata.setupStatus`, `setupExitCode` and `setupLog` |
| `--wait-setup` | Wait for the setup script to finish |
| `--devcontainer` | Apply a `devcontainer.json` (default `.devcontainer/devcontainer.json`): push the workspace into `workspaceFolder`, and apply `containerEnv` and `remoteEnv`, features, and the `onCreate`, `updateContent`, `postCreate` and `postStartCommand` hooks. `image` and `build` are not booted |
| `--secrets` | Which org Secrets the machine gets. [Secrets](https://instabox.sh/docs/security/secrets.md) |
| `--no-vault` | Turn off only personal Vault delivery |
| `--ready-timeout D` | If the machine is not running `D` after the request (`1s` to `1h`), try to delete it and exit 6 (`create_timeout`) within seconds. Ctrl-C while waiting does the same (exit 130). The setup script and devcontainer are not counted |

`create --json` prints the machine once it runs, as `get --json` does. `readyAt` is when it became ready
(SSH, exec and files work), so `readyAt` minus `createdAt` is how long the create took. The setup script
and devcontainer run after it and are not counted. With `--no-wait`, `--json` prints the operation, which
has no `readyAt`; `instabox get --json` shows it once the machine is ready. A machine that is not ready
has no `readyAt` key.

```json theme={null}
{"id":"m_...","name":"job","state":"running","runtime":{"ready":true},
  "createdAt":"2026-11-06T10:00:00.000Z","readyAt":"2026-11-06T10:00:04.200Z", ...}
```

`create` exits 75 when no host has room (retryable `no_capacity`), with or without `--fail-fast`.

`--ready-timeout` limits only how long `create` waits; it is not a lifetime limit. A machine that is ready
in time runs as usual (use `--delete-after` to delete it later).

The server deletes a machine only once its create has ended, and a running create cannot be cancelled.
So at the deadline `create` sends the delete, waits a few seconds for a create that is still running,
and then exits either way. With `--json`, the error carries the machine's ID and what happened:

```json theme={null}
{"error":{"code":"create_timeout","message":"job (m_...) was not ready within 2m; deleted it",
  "operationId":"op_...","machineId":"m_...","deleted":true}}
```

If the create was still running, `deleted` is `false`, `createRunning` is `true`, and `hint` names the
`instabox rm` command to run once `instabox get` no longer shows the machine creating. Pass
`--delete-after` too as a backstop: the server then deletes the machine at that time whatever happens.
If the server accepted the delete but `create` did not see it finish within 20 seconds, `deleted` is
`false`, `deletePending` is `true`, `deleteOperationId` names the delete, and `hint` says to check with
`instabox get` and run `instabox rm` if the machine is still there. `deleted` is `true` only when the
delete is confirmed, so check it. If the delete failed, `deleted` is `false` and `hint` names the
`instabox rm` command. If the create request itself got no answer, `machineId` is absent and `hint` says how to
check for the machine.

### get

Show one machine.

```
get <id-or-name> [--json]
```

`get --json` also has `vcpus` and `memMiB` at the top level, as well as under `spec`.

### rename

Rename a machine in place, in any state.

```
rename <id-or-name> <new-name>
```

The ID, port URLs and SSH certificate stay. `rename` also renames the machine's `Host` block in
`~/.ssh/config`.

### usage

Show your machine and snapshot counts, and your limits.

```
usage [--json]
```

## Lifecycle

See [Lifecycle](https://instabox.sh/docs/machines/lifecycle.md) for what each state keeps.

### start, stop, suspend, resume, rm

Start, stop, pause, resume or delete a machine.

```
start   <id-or-name> [--no-wait]
stop    <id-or-name> [--no-wait]
suspend <id-or-name> [--no-wait]
resume  <id-or-name> [--no-wait]
rm      <id-or-name> [--no-wait] [--yes]
```

* `suspend` pauses the machine. `pause` works too.
* `rm` deletes the machine. `delete` works too.
* `rm` and the other `rm` commands ask y/N first. `--yes` (`-y`, `--force`) skips the question. Off a
  terminal, `--yes` is required.

### wake

Resume a machine that paused itself, and wait until it runs (up to 45 seconds).

```
wake <id-or-name>
```

A machine that you paused or stopped stays paused or stopped. Use `resume` or `start` for it.

### restart

Power-cycle a machine: stop it, then start it. `recover` works too.

```
restart <id-or-name> [--timeout 60s] [--json]
```

Disks are kept. A guest that has not shut down by `--timeout` is killed.

### restore

Put the machine's disks back to a snapshot's, in place: the same ID, name, hostname and URLs.

```
restore <id-or-name> [--snapshot S] [--stopped] [--yes] [--json]
```

Everything since the snapshot is lost, so `restore` asks first. `--stopped` leaves the machine stopped.

### auto-suspend

Pause the machine once it is idle that long. `0` turns it off.

```
auto-suspend <id-or-name> <duration>|0
```

See [Auto-pause](https://instabox.sh/docs/machines/auto-pause.md).

### max-run

Pause the machine after that many running minutes, even while it is active.

```
max-run <id-or-name> <minutes>
```

`-1` is unlimited, and `0` pauses the machine now. On the free plan you cannot change it: each run
starts at 120 minutes and grows while the machine is in use, up to
480 minutes ([Runtime on the free plan](https://instabox.sh/docs/machines/auto-pause.md#runtime-on-the-free-plan)).
The idle limit and the runtime limit both apply: the first one reached pauses the machine.

### resize

Change the size of `/data`, or give the machine another of the plan's sizes.

```
resize <id-or-name> --disk N [--json]
resize <id-or-name> --vcpus N --mem-mib M [--restart] [--json]
```

* `--disk N` grows or shrinks `/data` to N GB, on paid plans. A running machine's disk grows live.
* `--vcpus` and `--mem-mib` give the machine another of the plan's sizes.
* A shrink or a new size applies while the machine is stopped. `--restart` stops the machine, resizes it
  and starts it: files stay, processes do not.

See [Sizes](https://instabox.sh/docs/machines/sizes.md).

## Run commands

### exec

Run one command in a machine and exit with its code.

```
exec <id-or-name> [--cwd D] [--env K=V ...] [--timeout S] [--stdin] [--stream] [--json] -- <cmd...>
```

One argument is a shell string (`'a | b'`). Several arguments run as that exact argv.

| Flag | What it does |
| - | - |
| `--cwd D` | Working directory |
| `--env K=V` | Environment variable for this command |
| `--timeout S` | Whole seconds before the command is killed (default 60, at most 600). A killed command exits 124 |
| `--stdin` | Pipe input to the command, up to 5 GiB, binary too. Input larger than 1 MiB, or not UTF-8, goes up as a file first (its SHA-256 checked), and the command reads it from there. A CLI built before 2026-10-26 refuses stdin past 1 MiB (`stdin is larger than 1048576 bytes`): run `instabox update` instead of splitting the input |
| `--stream` | Print the output as the command writes it, not when it ends (no 1 MiB cap). With `--json`, one `{type: stdout\|stderr, data}` object per line, then `{type: exit, exitCode, ...}` |
| `--json` | `{stdout, stderr, exitCode, signal, timedOut, truncated, invalidUtf8, durationMs}` |

`exec` exits 65 when the command exited 0 but its output was binary. Copy files with `pull` or `cp`
instead.

### `exec --background`

Start a command detached, with no time limit, and print its job ID.

```
exec --background <id-or-name> [--cwd D] [--env K=V ...] [--json] -- <cmd...>
```

`--json` prints `{jobId, pid}`.

### jobs

List, read, wait for and stop a machine's background jobs.

```
jobs ls   <id-or-name> [--json]
jobs logs <id-or-name> <job> [-f] [--json]
jobs wait <id-or-name> <job> [--timeout D] [--json]
jobs kill <id-or-name> <job> [--signal SIG] [--json]
```

* `ls`: the machine's background jobs, with ID, status (`running`, `exited(N)`, `lost`), start time and
  command.
* `logs`: a job's output. `-f` follows it until the job ends, then exits with the job's exit code.
* `wait`: wait for a job to end, and exit with its exit code. After `--timeout`, `wait` exits 124 and
  leaves the job running.
* `kill`: send a signal (default `TERM`) to the job's process group.

### run

Create a machine, run one command in it, and delete the machine, also on error or Ctrl-C.

```
run [--template T | --from SNAPSHOT] [--name N] [--push DIR] [--cwd D] [-e K=V ...]
    [--timeout 60s] [--keep] [--fail-fast] [--setup-file PATH|-] [--devcontainer [PATH|DIR]]
    [--json] -- <cmd...>
```

* `--push DIR` uploads the unignored files of DIR into the working directory first.
* `--keep` keeps the machine and prints its name to stderr. A machine whose create operation failed is
  not kept. A machine whose wait was cut short is kept.

`run` exits with the command's code, or 124 when the command timed out. Its own failures have their own
codes:

* 75: no host has room, with `--fail-fast`.
* 125: an API, network, create, upload or cleanup error. No room without `--fail-fast` is one of these:
  retry later, or try another size or template.
* 130: interrupted.

When the CLI is killed, the server deletes its machine at the machine's `deleteAt`: 5 minutes after
`--timeout`, or 10 minutes with `--push`. A failed cleanup delete is retried 5 times with backoff.

### ps

List the machine's processes.

```
ps <id-or-name> [--all] [--json]
```

Each row has pid, ppid, user, CPU and memory %, elapsed time and command. `--all` adds kernel threads.

### history

List the commands run in the machine through `exec`, newest first.

```
history <id-or-name> [--limit N] [--cursor C] [--json]
```

Each row has ID, start time, duration, exit code and command. The list has commands run by anyone, and
`history` works while the machine is stopped too. The server keeps the newest 200 commands, with the
first 64 KiB of each output stream. `--limit` is 1-200 (default 50). `--cursor` reads the next page.

### output

Print a past command's stored stdout and stderr, and exit with its exit code, as `exec` did.

```
output <id-or-name> [<command-id>] [--json]
```

Without `<command-id>`, `output` prints the last command. `--json` prints the record with stdout and
stderr.

### agent

Drive a coding agent (Claude Code or Codex) inside a machine.

```
agent prompt    <id-or-name> [--provider claude|codex] [--model M] [--new | --resume ID]
                [--attach FILE ...] [--cwd D] [--env K=V ...] [-f] [--json] <text...>
agent events    <id-or-name> [-c ID] [-f] [--json]
agent steer     <id-or-name> "text"
agent interrupt <id-or-name>
agent ls        <id-or-name>
```

* `prompt`: start a turn and print the conversation ID. Without `--resume`, it starts a new
  conversation (default provider `claude`). `--attach` uploads a file and adds its path to the prompt. A
  text of `-` reads the prompt from stdin (at most 256 KiB). `-f` follows the events and exits with the
  turn's exit code.
* `events`: a conversation's JSONL events (default: the conversation prompted last). `-f` follows the
  running turn until it ends, then exits with its exit code. Ctrl-C stops following and leaves the turn
  running.
* `steer`: interrupt the running turn, if there is one, and resume with the text.
* `interrupt`: stop the running turn (INT, then TERM, then KILL).
* `ls`: the conversations, with ID, provider, model and status (`running`, `idle`, `interrupted`,
  `failed(N)`, `lost`).

Each prompt is a turn: one background job that runs `claude -p --output-format stream-json` or
`codex exec --json`, with permissions and sandboxing skipped (the machine is the boundary). A
conversation lives in `~/.instabox/agents/<id>` in the machine, and later turns resume the agent's
session.

Provider keys come from the machine's environment, from your Secrets or `--env`:
`ANTHROPIC_API_KEY` or `CLAUDE_CODE_OAUTH_TOKEN` for Claude Code, and `OPENAI_API_KEY` or `CODEX_API_KEY`
for Codex. When the template lacks the agent CLI, the turn installs it with npm.

## Copy files

`push`, `pull` and `cp` copy directories by these rules:

* SRC into an existing DST lands in `DST/<name>`: `push box app /data/w` gives `/data/w/app`.
* `SRC/`, `SRC/.` or `.` copies the contents into DST itself: `push box app/ /data/w` puts the files of
  app in `/data/w`.
* With DST omitted, the copy keeps its name: `push box .` gives `HOME/<dirname>`.

For large files, use `push` or `cp`, not scp or rsync over `ssh.instabox.sh`. See
[SSH](https://instabox.sh/docs/access/ssh.md#copy-large-files-with-the-cli-not-scp) for why and the measured throughput.

### files

List a live directory or print a text file, as root.

```
files <id-or-name> [PATH] [--json]
```

### push

Copy a file or directory into a machine.

```
push <id-or-name> LOCAL [REMOTE] [-p] [--all]
```

REMOTE defaults to HOME. A file streams up to 5 GiB, and a file of 64 MiB or more goes up as 4 parallel
PUTs. In a git repo, `push` copies only tracked files. `--all` adds untracked files. `-p` creates the
parents of REMOTE.

Past 5 GiB, `split` the file, push the parts, and `cat` them together in the machine.

### pull

Copy a file or directory out of a machine.

```
pull <id-or-name> REMOTE [LOCAL]
```

LOCAL defaults to the current directory. A file streams, at any size.

### cp

Copy scp-style, in either direction.

```
cp [-r] [-p] SRC DST
```

`cp ./x.tgz box:/data/home/x.tgz` copies in, and `cp box:/data/home/x.tgz .` copies out.

## Monitor

These commands never wake a paused machine.

### disk

Show how full `/data` and `/` are.

```
disk <id-or-name> [--json]
```

The levels are `ok` below 80%, `high` from 80%, `critical` from 90%, and `full`.

### metrics

Show CPU, memory, network and disk I/O, recorded every 30 seconds.

```
metrics <id-or-name> [--since 1h] [--json]
```

`--since` goes back up to 24h.

### logs

Show the console log and the recent operations, in any state.

```
logs <id-or-name> [--tail N] [-f] [--json]
```

`-f` follows the log. See [Logs and labels](https://instabox.sh/docs/machines/logs-and-labels.md).

## Snapshots

See [Snapshots](https://instabox.sh/docs/machines/snapshots.md). `snapshots`, `image` and `images` work as `snapshot`.

### snapshot save

Save the machine's disks as a private snapshot.

```
snapshot save <id-or-name> --name N [--description D] [--root-only] [--restart]
```

`--root-only` saves installed software, not home and workspace. `--restart` saves a running machine
that cannot be saved live, such as a GPU machine: it stops the machine, saves it, and starts it again,
also when the save fails. A machine that can be saved live is not stopped.

### snapshot ls

List your snapshots.

```
snapshot ls [--json]
```

### snapshot files

List a directory of the snapshot's installed files, or print a text file.

```
snapshot files <id-or-name> [PATH] [--json]
```

### snapshot import

Build a private snapshot from a Docker or OCI image. It pulls the image in a throwaway machine, then
deletes that machine.

```
snapshot import IMAGE --name N [--description D] [--registry-username U] [--registry-password-stdin]
  [--no-entrypoint] [--no-verify] [--keep-builder] [--timeout 30m] [--json]
```

See [Custom images](https://instabox.sh/docs/machines/custom-images.md) for what the snapshot keeps and every flag.

### snapshot rm

Delete a snapshot. It asks y/N first, as `rm` does.

```
snapshot rm <id-or-name> [--yes]
```

### fork

Copy a machine into `--count` (1-5) machines. `clone` works too.

```
fork <id-or-name> [--name N] [--count N] [--keep-snapshot] [--json]
```

On the machine's host, `fork` uses no snapshot. Otherwise it saves a snapshot and creates the copies
from it. See [Parallel forks](https://instabox.sh/docs/use-cases/parallel-forks.md).

## SSH access

See [SSH](https://instabox.sh/docs/access/ssh.md).

### ssh

Open an SSH session to a machine, or run a command in it over SSH.

```
ssh <id-or-name> [--no-paste] [-- ssh args or command]
```

`--no-paste` turns off the Ctrl+V image paste. See [SSH](https://instabox.sh/docs/access/ssh.md).

### ssh-config

Print a `~/.ssh/config` Host block.

```
ssh-config <id-or-name>
```

### ssh-cert

Refresh only the SSH certificate.

```
ssh-cert <id-or-name>
```

### ssh-token

Print a plain `ssh` command, or a `mosh` command with `--mosh`, that needs no CLI or key.

```
ssh-token <id-or-name> [--ttl 7d] [--mosh] [--json]
ssh-token --revoke <id-or-name>
```

`--revoke` revokes the machine's ssh commands.

### mosh

Open a session as `ssh` does, over mosh. mosh must be installed.

```
mosh <id-or-name> [-- command]
```

### port-forward

Forward local ports over SSH until Ctrl-C. `forward` works too.

```
port-forward <id-or-name> [LOCAL:]REMOTE [...] [--remote R [--local L]] [--address A]
```

## Ports

See [Ports and URLs](https://instabox.sh/docs/access/ports.md). `ports` works as `port`.

### port publish

Give a port an HTTPS URL until you unpublish it.

```
port publish <id-or-name> <port> [--name NAME] [--private|--public] [--rotate-token] [--tcp] [--json]
```

* `--private`: only the printed link, with its `_token`, opens the URL.
* `--tcp`: a public raw TCP service at a TLS address (SNI).

### port ls

List the published ports of one machine or all of them.

```
port ls [<id-or-name>]
```

### port unpublish

Remove the URL and close its connections.

```
port unpublish <id-or-name> <port>
```

### port setup

Install the `instabox-port` command in the machine.

```
port setup <id-or-name>
```

## Orca

### orca connect

Print a one-time Orca connection, as JSON.

```
orca connect <id-or-name> [--scope runtime|mobile]
```

## Secrets

Org Secrets reach a machine only as placeholders. See [Secrets](https://instabox.sh/docs/security/secrets.md). `secret` works as
`secrets`.

* Org commands default to your personal org. With `INSTABOX_API_KEY` set, they default to the key's org.
* Setting, importing and removing values require the org owner or an admin: a signed-in session, or a
  workspace's `workspace:admin` key in `INSTABOX_API_KEY`.
* `--host` can be repeated for a Secret used with several hosts. Without `--host`, every public HTTPS
  destination is allowed. `--ai anthropic|openai|gemini|openrouter` adds that provider's API host.
* Names in selection flags can be comma-separated or a JSON array.
* `create ... [--secrets all|none|NAMES]` selects Secrets when the machine is created.
* `--no-vault` turns off only personal Vault delivery. To turn off both stores on a new machine, also
  pass `--secrets none`.

### secrets set

Set a Secret. The value is read from stdin or a hidden prompt.

```
secrets set NAME [--host HOST] [--ai PROVIDER] [--org ORG]
```

### secrets import

Import a file of Secrets atomically. A FILE of `-` reads stdin.

```
secrets import FILE [--host HOST] [--ai PROVIDER] [--org ORG]
```

### secrets ls

List Secrets: metadata only, never values.

```
secrets ls [--org ORG] [--json]
```

### secrets rm

Delete a Secret.

```
secrets rm NAME [--org ORG] [--yes]
```

### secrets show

Show a machine's Secret selection and delivery status.

```
secrets show <id-or-name>
```

### secrets select

Choose which Secrets a machine gets: `all`, `none` or names.

```
secrets select <id-or-name> --secrets all|none|NAMES
```

### secrets push

Refresh Secret delivery on a running machine.

```
secrets push <id-or-name>
```

## Vault

Vault delivers personal env values and files, with metadata selectors. It is independent of org Secrets:
`--no-vault` does not clear Secret selections. The CLI warns about a future Vault retirement, and these
commands remain supported.

### vault set

Set an env value. The value is read from stdin or a hidden prompt.

```
vault set NAME
```

### vault set-file

Set a file item. Its contents are read from stdin.

```
vault set-file PATH
```

### vault import

Import a `.env` file.

```
vault import FILE
```

### vault ls, vault rm

List the items (metadata only), or remove one.

```
vault ls [--json]
vault rm NAME [--yes]
```

### vault show

Show the items that match a machine, and its last delivery.

```
vault show <id-or-name>
```

### vault push

Refresh Vault delivery.

```
vault push <id-or-name>
```

### vault export

Write an item that an earlier explicit migration retained to a new private file. It never prints the
value.

```
vault export NAME --out recovered.json
```

> [!WARNING]
>
> Export works only on items retained by a migration, and the file holds real values. Keep it private.

## Webhooks

See [Webhooks](https://instabox.sh/docs/integrations/webhooks.md) for the events and the signature. `webhook` works as `webhooks`.
Org commands default to your personal org. With `INSTABOX_API_KEY` set, they default to the key's org.

### webhooks create

Create a webhook. It prints the signing secret once.

```
webhooks create URL [--event TYPE ...] [--description D] [--disabled] [--org ORG]
```

### webhooks ls, get, update, rm

List, show, change or delete webhooks.

```
webhooks ls
webhooks get ID
webhooks update ID [--url U] [--event T ...] [--enable|--disable]
webhooks rm ID
```

### webhooks rotate, test, deliveries, retry, events

Manage a webhook's signing secret and deliveries, and list the event types.

```
webhooks rotate ID
webhooks test ID
webhooks deliveries ID [--limit N]
webhooks retry ID DELIVERY
webhooks events
```

## Workspaces

Machines, snapshots and stored data are counted against an org: your personal org, or a workspace.
`org`, `orgs`, `ws` and `workspaces` work as `workspace`. `<ws>` is an org ID or name, and
`create --org <ws>` puts a machine in it. `ls`, `get` and `members` work with an API key. The other
workspace commands need `instabox login`.

### workspace ls, get

List your orgs, or show one.

```
workspace ls [--json]
workspace get <ws>
```

`ls` lists your personal org first, then your workspaces. `get` shows an org's plan, limits, this
month's usage and allowance.

### workspace create, rename, delete

Create, rename or delete a workspace.

```
workspace create <name> [--plan P]
workspace rename <ws> <name>
workspace delete <ws>
```

`create` prints the Stripe Checkout URL that pays the plan. The workspace exists once the checkout
completes.

### Members and invites

Manage a workspace's members and invitations.

```
workspace members <ws>
workspace invite <ws> <email> [--admin]
workspace invites <ws>
workspace resend-invite <ws> <invite-id>
workspace revoke-invite <ws> <invite-id>
workspace join <code>
workspace role <ws> <user-id> admin|member
workspace rm-member <ws> <user-id>
workspace leave <ws>
```

An invitation is valid for 7 days. The person accepts it signed in with that email.

### Billing

Manage an org's plan, card, spend cap and usage alerts.

```
workspace subscribe <ws>
workspace card <ws>
workspace plan <ws> pro|builder|team
workspace cancel <ws>
workspace resume <ws>
workspace migrate (<ws> | --new <name> [--plan <tier>]) [--dry-run] [--yes]
workspace billing <ws>
workspace spend-cap <ws> <usd>|off
workspace alerts <ws> [--usage N|off] [--overage on|off]
```

* `subscribe` puts a workspace or your personal org on a paid plan, or renews it. `upgrade` works too.
* `card` saves a card for a free workspace.
* `cancel` ends the subscription at the end of the period. `resume` undoes the cancel.
* `migrate` moves your personal sandboxes, credits and plan into a workspace you own, or with
  `--new` into a new one. Into a workspace without a plan, or a new one, your active plan moves with
  you, with its card and renewal date, and nothing is charged. Without a personal plan, a new
  workspace is paid from the credits that move. See [Move a personal plan](https://instabox.sh/docs/account/move-personal-plan.md).
* `billing` prints the Stripe portal URL. `portal` works too.
* `spend-cap` caps this month's usage beyond the allowance.
* `alerts` shows or sets the usage alert emails.

See [Billing](https://instabox.sh/docs/account/billing.md).

## API keys

See [API keys](https://instabox.sh/docs/security/api-keys.md). `key` works as `keys`.

With `INSTABOX_API_KEY=ibk_...` set, and optionally `INSTABOX_API_URL`, every command other than the
`keys` commands uses the key instead of a sign-in. The `keys` commands need a sign-in.

### keys create

Create an API key. It prints the secret once.

```
keys create --name N [--scope S ...] [--expires-in-days N]
```

### keys ls

List your API keys.

```
keys ls [--json]
```

### keys rm

Revoke an API key.

```
keys rm <key-id> [--yes]
```

## Integrations

### mcp

Run the instabox MCP server, or add it to an MCP client's config.

```
mcp start
mcp init claude|cursor|codex|opencode|antigravity [--hosted] [--inline-key] [--project]
```

* `start`: a stdio MCP server that uses this CLI's sign-in or `INSTABOX_API_KEY`, and never opens a
  browser. MCP clients run it.
* `init`: add the instabox MCP server to that client's config, and keep the rest of the file. By
  default the client runs `instabox mcp start`. `--hosted` writes the hosted URL with the `X-API-Key`
  header, read from `INSTABOX_API_KEY` when the client starts. `--inline-key` writes the key itself, with
  file mode 0600. `--project` writes the project's config file instead of the user's.

See [MCP](https://instabox.sh/docs/integrations/mcp.md).

### ci

List the org's recent GitHub Actions jobs on instabox CI machines, newest first.

```
ci [--org O] [--json]
```

Each job shows its size and state, how long it waited for a machine and ran, and why it waits or was
refused. See [CI](https://instabox.sh/docs/use-cases/ci.md).

### feedback

Report a problem with instabox itself to the instabox team.

```
feedback --type T --component C --title TEXT --detail TEXT [--json]
```

* `--type`: `bug`, `feature-request`, `friction`, `other`.
* `--component`: `cli`, `api`, `sandbox`, `dashboard`, `skills`, `docs`, `other`.

## The CLI itself

### version

Print the CLI version.

```
version
```

`version --json` adds `"outdated"`, and `"latest"` when `"outdated"` is true.

### update

Replace this CLI with the latest `instabox` npm package, the build that `npx instabox@latest` installs.
`self-update` works too.

```
update
```

`update` checks the download against the registry's integrity hash and swaps it in place, or prints
"already up to date". `INSTABOX_NPM_REGISTRY` names a registry mirror.

> [accordion] The daily outdated check
>
> Once a day, the CLI compares its version with the registry's latest. The result is cached in
> `update-check.json` in the config directory, the check has a 1.5-second limit, and an unreachable
> registry is ignored. While the versions differ, every command except `update` ends with one line on
> stderr, never stdout: `instabox <version> is outdated (latest <npm version>): run instabox update`.
> `ssh`, `mosh` and `port-forward` print the line before they connect. A `--json` failure leaves the
> line out, so its stderr has only the error line. Development builds are not checked. `INSTABOX_NO_UPDATE_CHECK=1`
> turns the check and the line off.

### completion

Print a tab-completion script for commands, flags and machine names.

```
completion bash|zsh|fish|powershell
```

Add `eval "$(instabox completion bash)"` to `~/.bashrc`, or `eval "$(instabox completion zsh)"` to
`~/.zshrc`. `instabox completion help` covers fish and PowerShell. Machine names come from your machine
list, which never wakes a machine.

## Global flags

| Flag | Environment | What it does |
| - | - | - |
| `--server URL` | `INSTABOX_SERVER` or `INSTABOX_API_URL` | API server |
| `--ca-cert FILE` | `INSTABOX_CA_CERT` | Extra CA certificate for the server |
| `--insecure-skip-verify` | | Do not verify the server certificate (development only) |

## JSON output

`--json` works on every command, and each command's `-h` lists it. `-o json`, `--output json` and
`--format json` are the same. The exceptions are `ssh`, `mosh` and `port-forward`: they are interactive
and refuse it with the JSON error code `json_unsupported`. Use `exec --json` instead.

With JSON output, a failure prints `{"error":{"code","message",...}}` on stdout and one line,
`instabox: error CODE: MESSAGE`, on stderr, and still exits non-zero (2 for usage errors, code `usage`). API errors keep the server's
`code`, `status`, `requestId` and `retryable` (present when true).

When stdout is not a terminal (a pipe or a file), the list and status commands print JSON without the
flag: `ls`, `get`, `templates`, `usage`, `snapshot ls`, `port ls`, `keys ls`, `secrets ls`, `vault ls`,
and `workspace ls`, `get`, `members` and `invites`. `INSTABOX_NO_AUTO_JSON=1` or `--output table` keeps
the table. Every other command, `exec`, `ssh`, `files` and `create` among them, prints the same output
piped or not.

## Exit codes

A failed command exits with a code for the kind of error, so a script can tell them apart. Each code is
non-zero, so `if ! instabox ...` and `$? -ne 0` checks work. `--json` errors exit the same way.

| Code | Meaning |
| - | - |
| 0 | Success |
| 1 | Any other error |
| 2 | Usage: an unknown command or flag, a bad flag value, a missing argument |
| 3 | Auth: not signed in, a bad, expired or revoked key or session, no permission (`401`, `403`) |
| 4 | Not found: no such machine, snapshot, key or other resource (`404`) |
| 5 | Quota or plan limit: `quota_exceeded`, `credit_exhausted`, `spend_limit_reached`, `spec_not_allowed`, `template_not_allowed`, `template_limit`, `rate_limited` and other `402` and `429` errors |
| 6 | `create --ready-timeout` passed (`create_timeout`): the machine was not ready in time; `--json` `deleted` says whether `create` could delete it |
| 75 | Retryable: a `retryable` error such as `no_capacity` or `host_unavailable`, or another server error (`5xx`). Try again later |

Commands that run something exit with its code instead: `exec`, `ssh`, `jobs wait`, `jobs logs -f`,
`agent prompt -f`, `output` and `run`. The failures of `run` itself have their own codes: 2, 75, 125 and
130\. Every error code is in [Errors](https://instabox.sh/docs/reference/errors.md).
