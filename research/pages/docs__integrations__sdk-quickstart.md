> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# SDK quick start

> Create, run, pause and delete machines from TypeScript or plain HTTP.

This page walks through the calls an app needs: create a machine, run a command in it, pause and resume
it, and delete it, then open a shell and the desktop and read usage. Everything runs on your server with
an [API key](https://instabox.sh/docs/security/api-keys.md) (`ibk_...`).

> [!WARNING]
>
> Never send the API key to a browser.

## Install and authenticate

```bash theme={null}
npm install @insforge/instabox-sdk
export INSTABOX_API_KEY=ibk_...
export INSTABOX_API_URL=https://api.instabox.sh   # the default
```

The SDK reads `INSTABOX_API_KEY` and `INSTABOX_API_URL`, or takes `{ apiKey, apiUrl }` as options for
a key from your own secret store. Without a key, every call throws `InstaboxAuthError`
(`missing_api_key`) before sending a request. Over HTTP, send the key as `X-API-Key` or
`Authorization: Bearer`.

> [accordion] Running the examples
>
> The examples use `import` and top-level `await`. Save one as a `.mjs` file and run it with Node.js 18
> or later, or set `"type": "module"` in your `package.json` for a `.js` file.
>
> So that a failed step does not leave a machine running, wrap the steps after `create` in `try` and
> `finally { await sbx.kill() }`. Your app keeps the machine and deletes it when your user is done.

## Create a machine

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create('agent-browser', {  // a template, a snapshot name, or a snapshot ID
  metadata: { customer: 'acme' },                    // your own tags, to find it later
  vault: false,                                      // no personal Vault delivery
})
console.log(sbx.sandboxId)                           // m_...: store it with your user
```

```bash curl theme={null}
H="X-API-Key: $INSTABOX_API_KEY"
curl -s -H "$H" -H "Idempotency-Key: $(uuidgen)" -H 'Content-Type: application/json' \
  -d '{"name": "box", "templateId": "agent-browser"}' "$INSTABOX_API_URL/v1/machines"
curl -s -H "$H" "$INSTABOX_API_URL/v1/operations/op_...?wait=25s"  # long poll, until "state": "succeeded"
```

`Sandbox.create` waits until the machine runs, so the script prints the new machine's ID, `m_...`.
`agent-browser` is the default template: Linux, a desktop, and a browser. Later, from another request:

```ts theme={null}
const sbx = await Sandbox.connect(sandboxId)                 // attach by ID; resumes it
await Sandbox.list({ metadata: { customer: 'acme' } })     // find machines by tag
```

> [!NOTE]
>
> Over HTTP, lifecycle calls (create, pause, resume, delete) return an **operation**. Poll it until
> `succeeded` or `failed` before the next change to that machine. Lifecycle calls need an
> `Idempotency-Key` header and answer `400` without one. Reuse a key only to retry the same request.

> [accordion] Create over HTTP: defaults and refusals
>
> `POST /v1/machines` takes `{}`, which creates an agent-browser machine named `box-` and 8 random hex
> digits. An unknown `templateId` is `400`, listing the templates your plan can create from. A body
> field the API lacks is `400 unknown_field`. For `"template"`, the error asks: did you mean
> `"templateId"`?

## Run a command and move files

```ts TypeScript theme={null}
const r = await sbx.commands.run('python3 -c "print(6 * 7)"', {
  cwd: '/data/home',
  timeoutMs: 60_000,
})
console.log(r.stdout, r.exitCode)   // a non-zero exit throws CommandExitError (stdout, stderr, exitCode)

await sbx.files.write('/data/home/app.py', 'print("hi")')
console.log(await sbx.files.read('/data/home/app.py'))
```

```bash curl theme={null}
curl -s -H "$H" -H 'Content-Type: application/json' \
  -d '{"cmd": "uname -a", "timeoutSeconds": 60}' "$INSTABOX_API_URL/v1/machines/m_.../exec"
curl -s -H "$H" -H 'Content-Type: application/json' \
  -d '{"argv": ["printf", "%s|", "a b", "$X"]}' "$INSTABOX_API_URL/v1/machines/m_.../exec"  # no shell: prints a b|$X|
```

The TypeScript prints the command's output, `42`, and its exit code, `0`. Then it prints the file it
wrote, `print("hi")`. To run a command without throwing on a non-zero exit, use `sbx.commands.exec`.

> [accordion] exec over HTTP
>
> * Send `cmd` (a shell string, run with `bash -lc`) or `argv` (a program and its arguments, run with no
>   shell), not both.
> * It returns `stdout`, `stderr`, and `exitCode` when the command exits. After `timeoutSeconds` (60 by
>   default, at most 600), the command is killed.
> * With `Accept: text/event-stream` (`instabox exec --stream`), it streams output as the command runs.
>   To run a longer job detached, use `instabox exec --background` and `instabox jobs`.
> * It takes no `Idempotency-Key`, so a retry runs the command again.
> * When a disk of the machine is at least 90% full, or the command failed with
>   `No space left on device`, the result carries a `diskWarning`. Its fields are `mount`, `usedPct`,
>   `level`, `noSpace`, and `message`, and it names the full disk and the fix:
>   `instabox resize <box> --disk <GiB>` grows `/data`, live on a running machine. `instabox exec` prints
>   it on stderr.

## Pause and resume the machine

```ts TypeScript theme={null}
await sbx.pause()                     // memory, files and running processes are kept
await sbx.resume()                    // carries on where it left off, usually in about a second

const peek = await Sandbox.connect(sbx.sandboxId, { resume: false })  // look without waking it
console.log((await peek.getInfo()).state)
const again = await Sandbox.connect(sbx.sandboxId)                   // connect also resumes it
console.log(await again.isRunning())
```

```bash curl theme={null}
curl -s -X POST -H "$H" -H "Idempotency-Key: $(uuidgen)" "$INSTABOX_API_URL/v1/machines/m_.../suspend"
curl -s -X POST -H "$H" -H "Idempotency-Key: $(uuidgen)" "$INSTABOX_API_URL/v1/machines/m_.../resume"
```

The first log shows the machine's state without waking it. The second prints `true`, because `connect`
resumes the machine.

Idle machines pause by themselves after 30 minutes. To change that, call
`sbx.setAutoSuspend(idleMs)`. `sbx.stop()` shuts the machine down and keeps `/data`. See
[Lifecycle](https://instabox.sh/docs/machines/lifecycle.md).

## Delete the machine

```ts TypeScript theme={null}
const id = sbx.sandboxId
console.log(await sbx.kill())           // deletes the machine and its disks; false if already gone
console.log(await Sandbox.kill(id))     // the same, by ID from any later request
```

```bash curl theme={null}
curl -s -X DELETE -H "$H" -H "Idempotency-Key: $(uuidgen)" -H "X-Confirm-Delete: m_..." \
  "$INSTABOX_API_URL/v1/machines/m_..."
```

The first log prints `true`. The second prints `false`, because the machine is already gone.

## Open an SSH session

`sbx.sshCredentials(publicKey)` takes your public key and returns a short-lived certificate. Over HTTP,
send your public key to `POST /v1/machines/{id}/connections`, with an `Idempotency-Key`. It returns the
certificate and the host CA. See [SSH](https://instabox.sh/docs/access/ssh.md).

> [accordion] Full example
>
> The example prints the `ssh` command to run.
>
> ```ts theme={null}
> import { execFileSync } from 'node:child_process'
> import { existsSync, mkdtempSync, readFileSync } from 'node:fs'
> import { homedir, tmpdir } from 'node:os'
> import { join } from 'node:path'
> import { Sandbox } from '@insforge/instabox-sdk'
>
> // your key pair; a throwaway one if you have none
> let key = join(homedir(), '.ssh', 'id_ed25519')
> if (!existsSync(`${key}.pub`)) {
>   key = join(mkdtempSync(join(tmpdir(), 'instabox-ssh-')), 'id_ed25519')
>   execFileSync('ssh-keygen', ['-q', '-t', 'ed25519', '-N', '', '-f', key])
> }
> const publicKey = readFileSync(`${key}.pub`, 'utf8')
> const sbx = await Sandbox.create('agent-browser', { vault: false })
> try {
>   const ssh = await sbx.sshCredentials(publicKey, { ttlSeconds: 3600 })
>   // save ssh.certificate as <key>-cert.pub next to the private key,
>   // add ssh.knownHostsLine to known_hosts, then:
>   console.log(`ssh -i ${key} -p ${ssh.port} ${ssh.username}@${ssh.host}`)
> } finally {
>   // your app keeps the machine while the session lasts
>   await sbx.kill()
> }
> ```

## Drive the desktop

```ts theme={null}
await sbx.desktop.start()                     // safe to call again
console.log(await sbx.desktop.status())       // { installed, running, width, height }
await sbx.desktop.open('https://example.com')
const png = await sbx.desktop.screenshot()
await sbx.desktop.click(640, 400)
await sbx.desktop.type('hello')
await sbx.desktop.press('Enter')
```

Any key with `machines:write` that can reach the machine, including a workspace key, opens its desktop
over `GET /v1/machines/{id}/vnc`. That route is a WebSocket carrying the VNC (RFB) stream. A share link
(`sbx.desktop.streamUrl()`) is for the machine's creator only. See [Desktop](https://instabox.sh/docs/access/desktop.md) for links,
embedding, and an agent loop.

> [accordion] Open the VNC WebSocket from Node.js
>
> ```ts theme={null}
> import { randomBytes } from 'node:crypto'
> import http from 'node:http'
> import https from 'node:https'
>
> const api = new URL(process.env.INSTABOX_API_URL ?? 'https://api.instabox.sh')
> const status = await new Promise((ok, fail) => {
>   const req = (api.protocol === 'https:' ? https : http).request(
>     new URL(`/v1/machines/${sbx.sandboxId}/vnc`, api),
>     {
>       headers: {
>         'X-API-Key': process.env.INSTABOX_API_KEY,
>         Connection: 'Upgrade',
>         Upgrade: 'websocket',
>         'Sec-WebSocket-Version': '13',
>         'Sec-WebSocket-Key': randomBytes(16).toString('base64'),
>       },
>     },
>   )
>   // 101: hand the socket to your VNC client; this example closes it
>   req.on('upgrade', (res, socket) => {
>     socket.destroy()
>     ok(res.statusCode)
>   })
>   req.on('response', (res) => fail(new Error(`vnc: HTTP ${res.statusCode}`)))
>   req.on('error', fail)
>   req.end()
> })
> console.log(status)
> ```

## Read usage and metrics per customer

Tag each machine with your customer when you create it, then read usage by that tag:

```ts theme={null}
const machines = await Sandbox.list({ metadata: { customer: 'acme' } })
let seconds = 0
for (const m of machines) {
  const u = await Sandbox.getUsage(m.sandboxId)        // running seconds this UTC month
  seconds += u.runningSeconds
  const samples = await Sandbox.getMetrics(m.sandboxId) // CPU and memory, one sample per 30 s, oldest first
  console.log(m.sandboxId, samples.at(-1)?.cpuUsedPct)
}
console.log(machines.length, seconds)
```

The script prints each machine's latest CPU use, then the number of machines and their total running
seconds.

`getUsage` covers the current UTC month by default and counts running time only, not paused or stopped
time. A `Sandbox` object has the same calls: `sbx.getUsage()` and `sbx.getMetrics()`.
`Sandbox.usage()` returns account counts, storage, and limits.

## Handle errors

Calls throw `InstaboxError` with `status` and `code`, such as `not_found` or `quota_exceeded`. A
missing or bad key throws `InstaboxAuthError`. See [Errors](https://instabox.sh/docs/reference/errors.md).

## Use the CLI with a key

With `INSTABOX_API_KEY` set, the CLI needs no login:

```bash theme={null}
npm install -g instabox
instabox create --name box --template agent-browser
instabox exec box -- uname -a
instabox exec --background box -- 'npm run build'   # detached job: prints its ID
instabox jobs logs box <job> -f                     # follow it; exits with its code
instabox agent prompt box --provider codex -f 'fix the failing test'   # a coding agent turn, its JSONL events
instabox agent steer box 'use the existing helper instead'             # interrupt it and follow up
instabox ssh box
instabox rm --yes box
instabox run -- uname -a        # one-shot: create, run, delete
instabox run --push . -- ls     # upload this directory (git-ignored files skipped) first
```

## Reference

> [accordion] SDK calls and their HTTP endpoints
>
> | SDK | HTTP |
> | - | - |
> | `Sandbox.create` | `POST /v1/machines` |
> | `pause`, `resume` | `POST /v1/machines/{id}/suspend`, `.../resume` |
> | `kill` | `DELETE /v1/machines/{id}` |
> | `commands.run` | `POST /v1/machines/{id}/exec` |
> | `sshCredentials` | `POST /v1/machines/{id}/connections` |
> | `desktop.start` | `POST /v1/machines/{id}/desktop` |
> | `desktop.streamUrl`, `revokeStreamUrls` | `POST` and `DELETE /v1/machines/{id}/desktop/stream` (creator only) |
> | your VNC viewer | `GET /v1/machines/{id}/vnc` (WebSocket, any key with `machines:write`, a workspace key too) |
> | `getMetrics` | `GET /v1/machines/{id}/metrics` |
> | `getUsage` | `GET /v1/machines/{id}/usage` |
> | `Sandbox.usage` | `GET /v1/usage` |

> [card] [SDK reference](https://instabox.sh/docs/reference/sdk.md)
>
> Every call, by task: snapshots, labels, files and more.

> [card] [API reference](https://instabox.sh/docs/api-reference/introduction.md)
>
> Every endpoint an API key calls.

> [card] [Python SDK](https://instabox.sh/docs/reference/python-sdk.md)
>
> The same calls from Python: `pip install instabox-sdk`.
