> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Migrate from Boat

> Move dev machines, scripts, and runs-on boat CI jobs from Boat to InstaBox.

This page is for teams that run Boat sandboxes from the CLI, scripts, or GitHub Actions and want
the same work on InstaBox. Most commands map one to one. The CI change is one line per job.

```bash theme={null}
npx instabox@latest                       # sign in and install the instabox CLI
instabox create --name try --env NODE_ENV=development
instabox exec try -- 'uname -a'
instabox rm try --yes
```

## What maps to what

| Boat | InstaBox | Notes |
| - | - | - |
| Sandbox (`bx_...`) | Machine (`m_...`) | Commands take the machine's name or ID. |
| `BOAT_API_KEY`, `boat login --key-stdin` | `INSTABOX_API_KEY` | With it set, every command uses the key. There is no login step. Create one with `instabox keys create`. |
| `boat new --name dev` | `instabox create --name dev` | Sizes: `--vcpus` and `--mem-mib`. |
| `boat new --from <saved snapshot>` | `instabox create --name dev --snapshot <name>` | |
| `boat new -e KEY=VALUE` | `instabox create --env KEY=VALUE` or `--env-file FILE` | Fixed at create. |
| `boat new --setup-file FILE` | `instabox create --setup-file FILE` | |
| `boat list`, `boat info <id>` | `instabox ls`, `instabox get <name>` | |
| `boat exec <id> <cmd>` | `instabox exec <name> -- <cmd>` | |
| `boat exec --detach` | `instabox exec --background`, then `instabox jobs logs` | |
| `boat run --from S --push . -- cmd` | `instabox run --from S --push . -- cmd` | Deletes the machine after. |
| `boat ssh <id>` | `instabox ssh <name>` | |
| `boat push <id> ./app --to app` | `instabox push <name> ./app app` | The path is relative to the home directory. |
| `boat scp ./f bx_<id>:/path` | `instabox cp ./f <name>:/path` | Also `instabox pull`. |
| `boat forward <id>` | `instabox port-forward <name> 8080:3000` | |
| `boat host <id> 3000` | `instabox port publish <name> 3000` | `--private` works the same way. |
| `boat stop <id>` | `instabox stop <name>` | Shuts down and keeps the disk. |
| `boat resume <id>` after a stop | `instabox start <name>` | |
| | `instabox suspend <name>`, `instabox resume <name>` | Pause instead: keeps memory and processes. |
| `boat fork <id>` | `instabox fork <name>` | |
| `boat snapshot <id> <name>` | `instabox snapshot save <name> --name <snapshot>` | |
| `boat delete <id> --yes` | `instabox rm <name> --yes` | |
| Auto-stop, `boat extend` | Auto-pause, `instabox max-run` | See [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md). |
| Environment variables | `--env` at create | See [Environment variables](https://instabox.sh/docs/machines/environment.md). |
| Secret variables (`boat env set-var`) | A direct Secret, `--secret-env VAR=SECRET` | `instabox secrets set NAME --direct`. The machine gets the real value. |
| Secret files (`boat env set-file`) | A direct Secret, `--secret-file PATH=SECRET` | A file on tmpfs, linked from `PATH` in the home directory. |
| AI credentials proxy | Org Secrets with `--host` or `--ai` | The machine sees a placeholder. See [Org Secrets](https://instabox.sh/docs/security/secrets.md). |
| `boat prompt` | `instabox agent prompt` | Claude Code or Codex. |
| `runs-on: boat` | `runs-on: instabox-medium` | See [CI](https://instabox.sh/docs/use-cases/ci.md). |

## Move a dev machine

**1. Sign in and create a machine**

```bash theme={null}
npx instabox@latest
instabox create --name dev --env-file ./dev.env
```

`./dev.env` holds the plain `KEY=VALUE` lines that your Boat environment set. Put API keys,
secret variables, and secret files in Secrets instead (step 4).

**2. Copy your files in and run a command**

```bash theme={null}
instabox push dev ./app app
instabox exec dev -- 'cd app && ./test.sh'
instabox cp dev:app/report.txt ./report.txt
```

`push` copies into the machine's home directory. In a git repository it copies only tracked
files; add `--all` for the rest.

**3. Open a port**

```bash theme={null}
instabox exec --background dev -- 'cd app && python3 -m http.server 3000'
instabox port publish dev 3000
```

The CLI prints `https://3000-<machine id without m_>.<domain>/`. Add `--private` to get a link
that only its holder can open. See [Public and private ports](https://instabox.sh/docs/access/ports.md).

**4. Move your secrets**

For an API key sent in HTTPS headers, save a proxy Secret. The machine sees a placeholder, and the
outbound proxy adds the real value to requests to `api.example.com`:

```bash theme={null}
instabox secrets set SERVICE_API_KEY --host api.example.com
instabox secrets select dev --secrets SERVICE_API_KEY
instabox secrets push dev
```

For a model provider key, use `--ai anthropic` (or `openai`, `gemini`, `openrouter`) instead of
`--host`.

For a value the proxy cannot carry, such as a secret file, an SSH key, or a database URL, save a
direct Secret and give it to `dev`. The machine gets the real value as a variable, a file, or both:

```bash theme={null}
instabox secrets set DEPLOY_KEY --direct < ./deploy_key
instabox secrets set PROD_PG --direct
instabox secrets select dev --secret-file .ssh/id_ed25519=DEPLOY_KEY --secret-env DATABASE_URL=PROD_PG
instabox secrets push dev
```

These flags replace the machine's direct references and keep the proxy Secrets selected above. A
new machine takes the same flags at create, such as
`instabox create --name worker --secret-env DATABASE_URL=PROD_PG`.

The file is on tmpfs, linked from the path in the home directory, so it is not on the machine's
disk. To save a Secret, you need to be an org owner or admin. If the CLI says the site does not
take direct Secret files or renamed variables yet, use a variable with the Secret's own name:
`--secret-env PROD_PG`. See
[Put the real value in the machine with a direct Secret](https://instabox.sh/docs/security/secrets.md#put-the-real-value-in-the-machine-with-a-direct-secret).

**5. Pause, stop, and delete**

```bash theme={null}
instabox suspend dev    # pause: keeps memory and processes
instabox resume dev
instabox stop dev       # shut down, as Boat's stop does: keeps the disk
instabox start dev
```

A machine also pauses itself after 30 idle minutes, and a visit to a published port
wakes it. When you no longer need the machine, delete it:

```bash theme={null}
instabox rm dev --yes
```

## Move CI jobs

**1. Connect GitHub**

On the dashboard's **CI** page, click **Connect GitHub**. This replaces `boat ci connect`.

**2. Change runs-on**

Before:

```yaml theme={null}
jobs:
  test:
    runs-on: boat
    steps:
      - uses: actions/checkout@v4
      - run: npm ci && npm test
```

After:

```yaml theme={null}
jobs:
  test:
    runs-on: instabox-medium
    steps:
      - uses: actions/checkout@v4
      - run: npm ci && npm test
```

Each job gets a fresh Ubuntu 24.04 machine with the tools of GitHub's `ubuntu-24.04` image.

**3. Map the other labels**

| Boat | InstaBox | Size |
| - | - | - |
| `boat-small` | `instabox-small` | 2 vCPU, 4 GiB |
| `boat` | `instabox-medium` | 4 vCPU, 8 GiB |
| `boat-large` | `instabox-large` | 8 vCPU, 16 GiB |
| `boat-xlarge` | `instabox-xlarge` | 16 vCPU, 32 GiB |
| `boat/from=<snapshot>` | `instabox-medium-snapshot-<snapshot>` | The size before `-snapshot-` |

A snapshot label needs a snapshot saved from a `ci` machine, in a private repository. See
[Start jobs from a snapshot](https://instabox.sh/docs/use-cases/ci.md#start-jobs-from-a-snapshot). The plans each label
needs are in [CI](https://instabox.sh/docs/use-cases/ci.md).

**4. Watch the jobs**

`instabox ci` and the dashboard's **CI** page list your jobs. They replace `boat ci`,
`boat ci job`, and `boat ci logs`.

If your workflow calls the Boat CLI from a step instead, replace the Boat steps with one
`instabox run`:

```yaml theme={null}
jobs:
  test:
    runs-on: ubuntu-latest
    env:
      INSTABOX_API_KEY: ${{ secrets.INSTABOX_API_KEY }}
    steps:
      - uses: actions/checkout@v4
      - run: npm install -g instabox
      - run: instabox run --push . --timeout 10m -e CI=1 -- 'npm ci && npm test'
```

`instabox run` creates the machine, runs the command, and deletes the machine, also on failure. Store
the key with `instabox keys create --name github-ci --expires-in-days 90 --json | jq -r .key | gh secret set INSTABOX_API_KEY`.

## What works differently

> [accordion] Named environments
>
> InstaBox has no named, reusable environments. Keep the variables in an env file and pass
> `--env-file` at create, run setup with `--setup-file`, and save a prepared machine as a snapshot
> to start new machines from it.

> [accordion] Variables are fixed at create
>
> You cannot change a machine's variables after create. Set one for a single command with
> `instabox exec dev --env KEY=VALUE -- <cmd>`, or create the machine again.

> [accordion] Secret values are org Secrets, not environment settings
>
> Boat keeps secret variables and files in an environment. InstaBox keeps them as Secrets of the
> org, and each machine references the ones it needs. Proxy Secrets keep the real value out of the
> machine. Direct Secrets (`--direct`) put it in, as a variable or a tmpfs file, and any code in the
> machine can read and copy them, as with Boat's environment secrets.

> [accordion] Stop and pause
>
> Boat's stop saves the disk and frees the machine, so it maps to `instabox stop`, and Boat's resume
> after a stop maps to `instabox start`. InstaBox also has a pause: `suspend` keeps memory and running
> processes, and `resume` continues where the machine left off.

> [accordion] Not available
>
> * The CI `debug` option, which keeps a failed job's sandbox. To look at a failure, run the
>   command with `instabox run --keep`, which keeps the machine and prints its name.
> * Nested virtualization (`/dev/kvm`).
> * Raw UDP ports. Raw TCP ports are in preview. See [Public and private ports](https://instabox.sh/docs/access/ports.md).
> * Coding agents other than Claude Code and Codex in `instabox agent prompt`. Install another
>   agent in the machine and run it with `instabox exec`.

## Let your coding agent do the migration

Paste this prompt into your coding agent in the repository you want to move.

```text theme={null}
Migrate this repository from Boat to InstaBox. Use the InstaBox CLI (npx instabox@latest to sign in).

Map these:
- BOAT_API_KEY -> INSTABOX_API_KEY (with it set, the CLI needs no login step; drop boat login)
- boat new [--from S] [-e K=V] [--setup-file F] -> instabox create --name N [--snapshot S] [--env K=V] [--setup-file F]
- boat exec ID CMD -> instabox exec N -- CMD; boat exec --detach -> instabox exec --background
- boat run --from S --push . -- CMD -> instabox run --from S --push . -- CMD
- boat scp SRC bx_ID:PATH -> instabox cp SRC N:PATH; boat push ID DIR --to P -> instabox push N DIR P
- boat host ID PORT -> instabox port publish N PORT; boat forward -> instabox port-forward
- boat stop / resume (after a stop) / fork / delete --yes -> instabox stop / start / fork / rm --yes
  (instabox suspend / resume pauses instead and keeps memory and processes)
- boat snapshot ID NAME -> instabox snapshot save N --name NAME
- API keys sent in HTTPS headers and AI keys -> instabox secrets set NAME --host HOST (or --ai PROVIDER), then instabox secrets select N --secrets NAME
- secret variables and files (boat env set-var, boat env set-file) -> instabox secrets set NAME --direct,
  then create (or secrets select) with --secret-env VAR=NAME and --secret-file PATH=NAME
- GitHub Actions: runs-on boat-small / boat / boat-large / boat-xlarge -> instabox-small / instabox-medium / instabox-large / instabox-xlarge;
  boat/from=S -> instabox-medium-snapshot-S

Steps:
1. Find every use of the boat CLI, the Boat SDKs, BOAT_ environment variables, and runs-on: boat labels.
2. Change them with the mapping above. Do not invent commands: check each one with instabox <command> --help.
3. Run the project's tests, and run one changed script end to end against InstaBox.
4. Stop and tell me before changing anything that uses: named Boat environments,
   the CI debug option, nested virtualization (/dev/kvm), raw UDP ports, or boat prompt with an agent
   other than Claude Code or Codex. InstaBox works differently there.
5. List every file you changed and what you ran to check it.
```

## Troubleshooting

> [accordion] A job waits as queued
>
> `instabox ci` says why a job waits or was refused. Check that the GitHub App is connected to the
> repository and that your plan includes the label's size.

> [accordion] Create fails at the machine limit
>
> Every plan has a limit on machines, paused ones included. `instabox usage` shows the count and
> the limit. Delete machines you no longer need with `instabox rm`.

> [accordion] The free plan refuses a template
>
> The free plan creates machines from agent-browser, android, omarchy, ci only. A snapshot is not limited.

> [accordion] A port URL answers 502
>
> Nothing listens on the port, or the server listens on `127.0.0.1` only. Start the server on
> `0.0.0.0` or check it with `instabox exec dev -- 'curl -sI localhost:3000'`.

> [card] [CLI reference](https://instabox.sh/docs/reference/cli.md)
>
> Every command and flag.

> [card] [CI](https://instabox.sh/docs/use-cases/ci.md)
>
> Runner labels, snapshots, and the GitHub App.
