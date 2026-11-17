> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Commands

> Run shell commands in a machine with sbx.commands.

## commands.run

Run a shell command in the machine.

```ts theme={null}
run(cmd: string, opts?: CommandOptions & { background?: false }): Promise<CommandResult>
run(cmd: string, opts: CommandOptions & { background: true }): Promise<CommandHandle>
```

| Option | Type | Default |
| - | - | - |
| `cwd` | `string` | the server's: HOME, `/data/home` |
| `envs` | `Record<string, string>` | none. Merged over the `envs` given to `Sandbox.create` |
| `timeoutMs` | `number` | the server's. Sent as whole seconds, at most 600 000 |
| `stdin` | `string` or `Uint8Array` | none. Text or bytes, passed byte for byte. Up to 1 MiB of UTF-8 (`MAX_STDIN_BYTES`) goes in the request. More, or bytes that are not UTF-8, go up as a temporary file first (up to 5 GiB) |
| `onStdout` | `(data: string) => unknown` | none. Called with each chunk of stdout as the command writes it |
| `onStderr` | `(data: string) => unknown` | none. The same, for stderr |
| `background` | `boolean` | `false`. `true` starts the command detached and returns at once |
| `signal` | `AbortSignal` | none. Aborting it kills the command, and `run` throws `aborted`. With `background: true` it aborts only the start, and the job keeps running |

The result (`CommandResult`) is `{ stdout, stderr, exitCode, timedOut, truncated, error? }`. Without
callbacks, the result arrives in one response with each output stream cut at 1 MiB, and `truncated` says
so.
Commands are never retried.

A non-zero exit throws `CommandExitError`, which carries the same fields. A command that runs past
`timeoutMs` throws it with `timedOut: true` and exit code 124.

```ts theme={null}
import { CommandExitError } from '@insforge/instabox-sdk'

try {
  await sbx.commands.run('exit 3')
} catch (e) {
  if (e instanceof CommandExitError) console.log(e.exitCode, e.stderr)
}
```

## Streaming output

With `onStdout` or `onStderr` the output is streamed: each callback is called with every chunk as the
command writes it. A returned promise is awaited before the next chunk. `run` still resolves, or throws
`CommandExitError`, with the whole `stdout` and `stderr` once the command ends. Streamed output is not
capped, so `truncated` is false. A callback that throws ends the command, and `run` rejects with that
error.

```ts theme={null}
// Prints 1, 2 and 3 about a second apart, as the command writes them.
const r = await sbx.commands.run('for i in 1 2 3; do echo $i; sleep 1; done', {
  onStdout: (out) => process.stdout.write(out),
  onStderr: (err) => process.stderr.write(err),
})
console.log(r.stdout) // "1\n2\n3\n"
```

## Background commands

With `background: true`, `run` starts the command detached, as a job in its own session, and returns a
`CommandHandle` at once. It is not limited to 10 minutes.

* `pid`: the shell, which leads the process group.
* `jobId`: the job's ID. The CLI sees the same job: `instabox jobs ls`,
  `instabox jobs logs <id-or-name> <jobId> -f`, `instabox jobs wait`.
* `kill()`: sends SIGTERM to everything the command started. Returns `false` if all of it had exited.
* `wait({ onStdout?, timeoutMs?, signal? })`: follows the job's log, calling `onStdout` with each chunk
  from the start, and resolves with `{ stdout, stderr: '', exitCode, timedOut: false, truncated: false }`
  once it ends. A background command's stdout and stderr go to one log (`~/.instabox/jobs/<jobId>/out`),
  so `stdout` holds both. A non-zero exit throws `CommandExitError`. A job that ended without an exit
  code (the machine restarted) throws `job_lost`. `timeoutMs` gives up with `InstaboxTimeoutError`
  (`job_wait_timeout`), and `signal` stops waiting (`aborted`). The job keeps running in both cases.

`run` does not take `onStdout`, `onStderr` or `stdin` with `background: true`: it throws `InstaboxError`
with code `bad_request`. Pass `onStdout` to `wait()` instead. Background commands need bash, util-linux
`setsid` and GNU coreutils in the machine. Without them, `run` throws `background_unsupported`, naming
what is missing.

```ts theme={null}
const build = await sbx.commands.run('npm test', {
  background: true,
  cwd: '/data/home/app',
})
console.log(build.jobId)
// Follows the log; throws CommandExitError if the tests fail.
await build.wait({ onStdout: (out) => process.stdout.write(out) })

const server = await sbx.commands.run('node server.js', { background: true })
await server.kill()
```

## commands.exec

Run a command and return its `CommandResult` without throwing on a non-zero exit.

```ts theme={null}
exec(cmd, opts?): Promise<CommandResult>
```

Takes `cwd`, `envs`, `timeoutMs` and `stdin`. `exec` is marked internal. `run` is the supported call.
