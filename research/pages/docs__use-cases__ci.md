> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# CI

> Run your GitHub Actions jobs on fresh machines, or run one test command on InstaBox from any CI.

Your tests can run on InstaBox machines in two ways, and you can follow either one from start to
finish. With GitHub Actions, you change one
line of a workflow, and each job then runs on a fresh machine of its own and uses no GitHub Actions
minutes. With any other CI, or when you want no GitHub App, a step calls the CLI, which runs one
command on a machine and deletes the machine afterwards.

| | `runs-on: instabox-medium` (recommended) | The CLI from any CI |
| - | - | - |
| What runs on InstaBox | Each job, on a fresh machine of its own | One command per step |
| Setup | Install the GitHub App, change `runs-on` | An API key as a CI secret |
| Works with | GitHub Actions | Any CI |
| GitHub Actions minutes | None | The job's runner uses them |

## Run GitHub Actions jobs on InstaBox

**1. Connect GitHub**

On the dashboard's **CI** page, click **Connect GitHub** to install the instabox GitHub App.
You can read what the App asks for under [GitHub App permissions](#github-app-permissions).

![The CI page: Connect GitHub, then change runs-on to instabox-medium](https://mintcdn.com/instacloud/N9PM4qc9k2LHbGZu/images/ci-setup.webp?fit=max\&auto=format\&n=N9PM4qc9k2LHbGZu\&q=85\&s=8ca6ef275693bcfcee35d8eed13e4684)

**2. Change runs-on in your workflow**

In your workflow file, set the job's `runs-on` to an InstaBox label. `instabox-medium` is the
label that every plan can use.

```yaml theme={null}
jobs:
  test:
    runs-on: instabox-medium
```

Each job with this label gets a fresh Ubuntu 24.04 machine that has the tools of GitHub's
`ubuntu-24.04` image, and the machine is deleted when the job ends.

**3. See your jobs**

When a job has to wait or is refused, GitHub shows it only as queued. To see your jobs and the
reason why one waits or was refused, open the dashboard's **CI** page or list the jobs with
the CLI.

```bash theme={null}
instabox ci
```

The list shows your recent jobs with the newest first. Each row has the job's size, its
state, and how long the job waited for a machine and ran.

### Choose a machine size

The label in `runs-on` sets the size of the job's machine, and your plan decides which labels you
can use. The free plan runs `instabox-medium` only.

| Label | Machine | Plans |
| - | - | - |
| `instabox-small` | 2 vCPU, 4 GiB | Pro, Builder, Team |
| `instabox-medium` | 4 vCPU, 8 GiB | Free, Pro, Builder, Team |
| `instabox-large` | 8 vCPU, 16 GiB | Pro, Builder, Team |
| `instabox-xlarge` | 16 vCPU, 32 GiB | Builder, Team |
| `instabox-2xlarge` | 32 vCPU, 64 GiB | Team |

### Check what the machine has installed

The image is built from GitHub's runner-images release `ubuntu24/20261026.354`, so its tools and
versions are the ones in that release's
[Ubuntu 24.04 software list](https://github.com/actions/runner-images/blob/ubuntu24/20261026.354/images/ubuntu/Ubuntu2404-Readme.md).
A job can read the release in `$ImageVersion`. To print what the job's machine has, run this
command in a step of the job.

```bash theme={null}
/usr/local/lib/instabox/ci/software-list
```

> [!NOTE]
>
> Not installed: Haskell, Swift, Julia, CodeQL, Edge, and Homebrew. The Android SDK is a trimmed one,
> with a single NDK (27).

## Start jobs from a snapshot

A job can start from a snapshot of your own instead of a fresh machine, so your repository,
toolchain, and package caches are already there when the job begins. You set a machine up once,
save it as a snapshot, and then name the snapshot in `runs-on`.

**1. Prepare a machine**

Create a machine from the `ci` template, because a job only starts from a snapshot that was
saved from a `ci` machine.

```bash theme={null}
instabox create --name ci-warm --template ci
```

Open a shell in the machine and set it up as your jobs need it.

```bash theme={null}
instabox ssh ci-warm
```

In the shell, clone your repository to `~/work/<repo>/<repo>`, which is where
`actions/checkout` puts it. Then install your toolchain and fill your package caches.

**2. Save a snapshot**

Stop the machine first, because a snapshot of a running machine may leave out data that
programs have not written to disk yet.

```bash theme={null}
instabox stop ci-warm
```

Then save the snapshot under a name that you can use in a label.

```bash theme={null}
instabox snapshot save ci-warm --name acme-warm
```

The command waits until the snapshot is saved, so the snapshot is ready when the command
returns.

**3. Name the snapshot in runs-on**

Add `-snapshot-` and the snapshot's name to the size label.

```yaml theme={null}
jobs:
  test:
    runs-on: instabox-medium-snapshot-acme-warm
```

This label is the job's whole `runs-on`, so put no size label, `self-hosted`, or other label
next to it. Only a job with this exact `runs-on` can run on a machine from the snapshot. The
size still sets vCPUs and memory, and storage is at least the snapshot's.

You can also name the snapshot by its ID, as in `instabox-medium-snapshot-sn_...`.

**4. Check which snapshot a job used**

`instabox ci` and the dashboard's **CI** page show which snapshot each job started from.

```bash theme={null}
instabox ci
```

The `SNAPSHOT` column has the snapshot's name for a job that started from one, and `-` for a
job on a fresh machine.

```text theme={null}
ID    REPO      JOB        SIZE             SNAPSHOT   STATE               WAIT  RUN  REASON
1042  acme/app  CI / test  instabox-medium  acme-warm  running             9s    41s
1041  acme/app  CI / lint  instabox-medium  -          completed: success  12s   1m
```

> [!WARNING]
>
> Keep secrets out of the snapshot, which means no tokens, SSH keys, or `.env` files. Any workflow
> on a branch of a linked repository can name the snapshot. Jobs get secrets from GitHub Actions as
> usual. A job cannot change the snapshot, because each job gets its own copy.

### When a job that names a snapshot is refused

A job is refused, with the reason in `instabox ci`, when the snapshot is not in the workspace
that the GitHub App is linked to, when it is still saving or failed, or when it was not saved from
a `ci` machine. A refused job never falls back to a fresh machine.

Snapshots work in private repositories only, and only with **Run workflows from fork pull
requests** turned off in the repository's **Settings > Actions > General**. In a public repository,
or in one that runs workflows from forks, jobs that name a snapshot are refused, because a fork's
workflow could name the snapshot too. If GitHub does not answer InstaBox's check, the job waits and
is tried again.

> [accordion] What to put in the snapshot
>
> Put in the repository under `~/work`, your toolchains, and caches such as `~/.cargo/registry`,
> `~/.npm`, `~/.cache/pip`, or `~/go/pkg/mod`. Leave out secrets, and stop any services that you
> started before you save. A leftover runner registration or work directory does no harm,
> because each job's runner gets its own registration.

> [accordion] Keep the snapshot fresh
>
> The snapshot keeps the GitHub Actions runner that it was saved with, and GitHub stops sending
> jobs to a runner that is more than 30 days behind its latest release. Save the snapshot again
> from a new `ci` machine at least once a month.

## Run the CLI from any CI

This path works in any CI and needs no GitHub App. A step of your pipeline creates a machine,
uploads the checkout, runs one command, and deletes the machine. The example below is a GitHub
Actions workflow, and the same command works in a step of another CI.

**1. Store an API key as a CI secret**

The CLI in your pipeline signs in with an API key, so create one and store it as a secret of
your CI. This command creates a key that expires in 90 days and passes it to `gh`, which saves
it as the repository secret `INSTABOX_API_KEY`.

```bash theme={null}
instabox keys create --name github-ci --expires-in-days 90 --json \
  | jq -r .key \
  | gh secret set INSTABOX_API_KEY
```

The key goes straight into the secret and is not printed.

**2. Run the tests**

Add a job that installs the CLI and runs your tests with `instabox run`. In this workflow,
`npm ci && npm test` stands for your own test command.

```yaml theme={null}
jobs:
  test:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    env:
      INSTABOX_API_KEY: ${{ secrets.INSTABOX_API_KEY }}
    steps:
      - uses: actions/checkout@v4
      - name: Install the instabox CLI
        run: npm install -g instabox
      - name: Run the tests
        run: |
          instabox run --push . --timeout 10m -e CI=1 -- \
            'npm ci && npm test'
```

The step's log shows the push, the command's output, and the machine's deletion. The step
fails when the command fails, and the command can run for 10 minutes at most.

### Pass secrets to the command

When your tests need a secret, pass it to the command as a variable with `-e NAME="$NAME"`. Pull
requests from forks get no secrets, so the step cannot run there.

## GitHub App permissions

> [accordion] What the GitHub App asks for
>
> The [instabox GitHub App](https://github.com/apps/instabox-cloud) asks for these permissions:
>
> | Permission | Used for |
> | - | - |
> | Metadata: read | Required by GitHub for every App |
> | Actions: read | A job's branch, commit, and failed step |
> | Administration: read and write | Registering and deleting each job's runner |
> | Members: read (organizations) | Checking that whoever links an organization belongs to it |
>
> The App asks for nothing on your code, pull requests, issues, or workflows.

> [accordion] Why Administration?
>
> GitHub lets an App add a runner to a repository only through Administration. Personal accounts
> have no organization-level runners, so Administration is the only way to run their CI.

> [accordion] Don't want to grant it?
>
> You can call the CLI from any CI instead, which needs no GitHub App and only an API key, as
> [Run the CLI from any CI](#run-the-cli-from-any-ci) shows. On Enterprise, we can set up a
> narrower App for your organization: [contact us](https://instabox.sh/contact).

> [card] [API keys](https://instabox.sh/docs/security/api-keys.md)
>
> Scopes and expiry for the key your CI uses.

> [card] [CLI reference](https://instabox.sh/docs/reference/cli.md)
>
> `instabox run` and its exit codes.
