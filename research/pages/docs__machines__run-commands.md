> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Run commands

> Run a command in a machine, stream its output, and keep long jobs running in the background.

Running a command in a machine lets you build code, run tests, or start a server there without
opening a shell. The command runs as the machine's login user, in the home directory, and you get
back its output and its exit code, so a script or an agent can act on the result.

```bash theme={null}
instabox exec dev -- npm test
```

A machine that paused itself when it was idle wakes for the command, so you do not need to resume it
first. For work that takes longer than ten minutes,
[run the command in the background](#run-a-long-command-in-the-background).

## Run a command

A command's exit code becomes the result of the call, so your script can tell whether the command
worked. The CLI exits with the same code, and the TypeScript SDK throws `CommandExitError` when the
code is not zero.

```bash CLI theme={null}
instabox exec dev -- npm test
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.connect('dev')
const result = await sbx.commands.run('npm test')
console.log(result.stdout)
```

```python Python theme={null}
from instabox import Client

client = Client()
result = client.machines.exec("dev", "npm test")
print(result.stdout, result.exit_code)
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../exec \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" -d '{"cmd": "npm test"}'
```

The CLI treats one argument as a shell string, so when you want a pipeline to run inside the machine
and not on your computer, put it in quotes. When you pass several arguments, they run as that exact
program and arguments, with no shell in between.

```bash theme={null}
instabox exec dev -- 'cat package.json | grep version'
```

## Set the directory, environment, and timeout

By default a command starts in the home directory, `/data/home`, and is stopped after 60 seconds.
You can set another directory, add environment variables for this one command, and set a longer
timeout for a build or a test run, up to 600 seconds. A command that runs past its timeout is killed
and exits with code 124.

```bash CLI theme={null}
instabox exec dev --cwd /data/home/app --env NODE_ENV=test \
  --timeout 300 -- npm test
```

```ts TypeScript theme={null}
await sbx.commands.run('npm test', {
  cwd: '/data/home/app',
  envs: { NODE_ENV: 'test' },
  timeoutMs: 300_000,
})
```

```python Python theme={null}
client.machines.exec(
    "dev",
    "npm test",
    cwd="/data/home/app",
    env={"NODE_ENV": "test"},
    timeout=300,
)
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../exec \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"cmd": "npm test", "cwd": "/data/home/app",
       "envs": {"NODE_ENV": "test"}, "timeoutSeconds": 300}'
```

These variables apply to one command only. To set variables for every command in a machine, see
[Environment variables](https://instabox.sh/docs/machines/environment.md).

## Stream output as it is written

By default the output arrives all at once when the command ends. You can stream it instead, which
lets you watch a build or a test run while it happens. Streamed output also has no size cap, so use
streaming for a command that prints a lot.

```bash CLI theme={null}
instabox exec dev --stream -- npm run build
```

```ts TypeScript theme={null}
await sbx.commands.run('npm run build', {
  onStdout: (out) => process.stdout.write(out),
  onStderr: (err) => process.stderr.write(err),
})
```

## Send input to a command

When a command reads from standard input, you can pipe data into it from your computer, such as a
SQL file for a database client. The CLI takes up to 5 GiB this way, binary data included.

```bash CLI theme={null}
cat schema.sql | instabox exec dev --stdin -- psql app
```

```ts TypeScript theme={null}
await sbx.commands.run('psql app', { stdin: 'select 1;' })
```

```python Python theme={null}
client.machines.exec("dev", "psql app", stdin="select 1;")
```

## Run a long command in the background

A background command is detached from your terminal and has no time limit, so use it for a dev
server, a long build, or an agent run. The command returns a job ID at once, and you use that ID to
follow the job or end it later.

```bash CLI theme={null}
instabox exec --background dev -- npm run dev
```

```ts TypeScript theme={null}
const job = await sbx.commands.run('npm run dev', { background: true })
console.log(job.jobId)
```

The CLI and the SDK see the same jobs, so you can start a job from code and follow it from a
terminal.

## Follow, wait for, and stop a job

When you do not have a job's ID at hand, list the machine's jobs. The list shows each job's ID and
whether the job is running or has exited.

```bash theme={null}
instabox jobs ls dev
```

To watch a job, follow its output until the job ends. The command then exits with the job's exit
code, so it behaves like a command that you ran in the foreground.

```bash CLI theme={null}
instabox jobs logs dev <job-id> -f
```

```ts TypeScript theme={null}
await job.wait({ onStdout: (out) => process.stdout.write(out) })
```

In a script, you can wait for a job to end without printing its output. When the timeout passes
first, `wait` exits with code 124 and leaves the job running.

```bash theme={null}
instabox jobs wait dev <job-id> --timeout 10m
```

When you no longer need a job, such as a dev server you are done with, stop it. This sends `TERM` to
everything that the job started.

```bash CLI theme={null}
instabox jobs kill dev <job-id>
```

```ts TypeScript theme={null}
await job.kill()
```

## Run one command in a new machine

`instabox run` creates a machine, runs the command in it, and deletes the machine afterwards, also
when the command fails or you press Ctrl-C. Use it for a CI job or a one-off script, where you want
a clean machine each time and nothing left behind. You can add `--push` to upload a local folder
into the machine before the command runs.

```bash theme={null}
instabox run --push . -- npm test
```

`run` exits with the command's exit code, so a CI step fails when the command fails. For a full
pipeline, see [CI](https://instabox.sh/docs/use-cases/ci.md).

## See what ran

The machine keeps a history of the commands that were run through `exec`, by anyone, together with
their exit codes. Read it to find out what an agent or a teammate did in the machine. The history
also works while the machine is stopped.

```bash theme={null}
instabox history dev
```

The output of each command is stored too, so you can print the output of the last command again, or
of an earlier one by its ID.

```bash theme={null}
instabox output dev
```

The history shows commands that were run before. To see what is running now, list the machine's
processes.

```bash theme={null}
instabox ps dev
```

## Limits

* A command's timeout is 60 seconds by default and 600 seconds at most. A background command has no
  time limit.
* Without streaming, each of stdout and stderr is cut at 1 MiB, and the result's `truncated` field
  says so.
* A background job writes stdout and stderr to one log, so the SDK returns both in `stdout`.
* Over the API, `stdin` holds up to 1 MiB of UTF-8 text. For more, or for binary data, write a file
  first and redirect it into the command.
* The history keeps the newest 200 commands, with the first 64 KiB of each output stream.
* `exec` exits with code 65 when a command succeeded but its output was binary. Copy files with
  `instabox pull` or `instabox cp` instead.

> [card] [CLI reference](https://instabox.sh/docs/reference/cli.md#run-commands)
>
> Every flag of `exec`, `jobs`, `run`, `history`, and `output`.

> [card] [TypeScript commands](https://instabox.sh/docs/reference/typescript/commands.md)
>
> `commands.run`, streaming, and background handles.
