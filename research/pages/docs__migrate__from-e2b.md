> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Migrate from E2B

> Move code that creates E2B sandboxes to InstaBox machines, call by call, in Python or TypeScript.

This page is for code that creates E2B sandboxes and runs commands, code and files in them. Most calls
keep their shape: the TypeScript SDK uses the same `Sandbox` object, and the Python SDK uses a `Client`.
Read the [gaps](#what-has-no-direct-equivalent) first if your code uses `run_code` results, file
watching, or sandboxes that delete themselves after a timeout.

## What maps to what

| E2B | InstaBox Python (`instabox-sdk`) | InstaBox TypeScript (`@insforge/instabox-sdk`) |
| - | - | - |
| `E2B_API_KEY` | `INSTABOX_API_KEY` | `INSTABOX_API_KEY` |
| `Sandbox.create(template)` | `client.machines.create(template=...)` | `Sandbox.create(template, opts)` |
| `metadata={...}` on create | `metadata={...}` | `{ metadata }` |
| `envs={...}` on create | `env=` on each `exec` | `{ envs }` on create (added to each `commands.run`) |
| `sbx.commands.run(cmd)` | `client.machines.exec(box.id, cmd)` | `sbx.commands.run(cmd)` |
| `commands.run(cmd, background=True)` | `exec(box.id, "nohup cmd >log 2>&1 &")` | `commands.run(cmd, { background: true })` |
| `sbx.run_code(code)` (code interpreter) | `exec(box.id, "python3 -", stdin=code)` | `commands.run('python3 -', { stdin: code })` |
| `sbx.files.write`, `read`, `list` | `client.files.write`, `read`, `list` | `sbx.files.write`, `read`, `list` |
| `sbx.files.remove`, `exists` | `client.files.remove`, `exists` | `sbx.files.remove`, `exists` |
| `sbx.files.watch_dir` | none, see [gaps](#what-has-no-direct-equivalent) | none |
| `sbx.get_host(port)` | `POST /v1/machines/{id}/ports` (no SDK call) | `(await sbx.ports.publish(port)).url` |
| `sbx.pause()` | `client.machines.pause(box.id)` | `sbx.pause()` |
| `Sandbox.connect(id)` (resumes) | `client.machines.ensure_running(id)` | `Sandbox.connect(id)` |
| `Sandbox.list()` | `client.machines.list(metadata=...)` | `Sandbox.list({ metadata })` |
| `sbx.kill()` | `client.machines.delete(box.id)` | `sbx.kill()` |
| `timeout` on create, `set_timeout` | `delete_after_seconds` or `archive_seconds` on create | `archiveMs` on create |
| `e2b.Dockerfile`, `e2b template build` | push the image, `instabox snapshot import`, then `create(snapshot_id=...)` | the same import, then `Sandbox.create('<snapshot name>')` |
| home directory `/home/user` | `/data/home` (HOME) | `/data/home` (HOME) |

## Move your code

**1. Install the SDK and set the key**

Create a key on **API keys** in the dashboard, with `machines:write` and `machines:delete`. To
bring your E2B template ([below](#use-your-e2b-template)), also give it `images:read` and
`images:write`. **Full access** covers all of them. See [API keys](https://instabox.sh/docs/security/api-keys.md).

```bash theme={null}
pip install instabox-sdk          # Python, imports as instabox
npm install @insforge/instabox-sdk@^0.8.0   # TypeScript; ports.publish needs 0.8.0
export INSTABOX_API_KEY=ibk_...
```

**2. Create the machine**

E2B creates a sandbox that lives for its timeout. An InstaBox machine lives until you delete it, and
pauses after idle minutes on its own. `create` waits until the machine runs.

```python Python theme={null}
from instabox import Client

client = Client()  # reads INSTABOX_API_KEY
box = client.machines.create(metadata={"user": "u_123"}, delete_after_seconds=3600)
print(box.id)  # m_...
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create('agent-browser', { metadata: { user: 'u_123' } })
console.log(sbx.sandboxId) // m_...
```

The default template, `agent-browser`, has Linux, Python 3, Node.js, a desktop and a browser. To
start from your own image, see [Use your E2B template](#use-your-e2b-template).

**3. Run commands and code**

```python Python theme={null}
r = client.machines.exec(box.id, "echo $GREETING", env={"GREETING": "hi"})
print(r.stdout, r.exit_code)   # hi 0; a non-zero exit is a result, not an error

code = "import platform\nprint(platform.python_version())"
r = client.machines.exec(box.id, "python3 -", stdin=code)  # in place of run_code
print(r.stdout)
```

```ts TypeScript theme={null}
const r = await sbx.commands.run('echo $GREETING', { envs: { GREETING: 'hi' } })
console.log(r.stdout, r.exitCode) // a non-zero exit throws CommandExitError

const code = 'import platform\nprint(platform.python_version())'
const out = await sbx.commands.run('python3 -', { stdin: code }) // in place of runCode
console.log(out.stdout)
```

In Python, `exec` returns a non-zero exit code in `r.exit_code`. In TypeScript, `commands.run`
throws on it, as E2B does. A command stops after `timeout` seconds (60 by default, 600 at most).

**4. Move files**

```python Python theme={null}
client.files.write(box.id, "/data/home/data.csv", "a,b\n1,2\n")
print(client.files.read(box.id, "/data/home/data.csv"))
print([e.name for e in client.files.list(box.id, "/data/home")])
```

```ts TypeScript theme={null}
await sbx.files.write('/data/home/data.csv', 'a,b\n1,2\n')
console.log(await sbx.files.read('/data/home/data.csv'))
console.log((await sbx.files.list('/data/home')).map((e) => e.name))
```

Paths are absolute. Use `/data/home` where your E2B code used `/home/user`. Files under `/data`
survive a stop.

**5. Give a port a URL**

E2B's `get_host(port)` returns a host name. On InstaBox, publish the port once. The result has the
URL, `https://<port>-<machine id without m_>.<domain>`. A port is public by default, as on E2B.

```python Python theme={null}
import os, httpx

client.machines.exec(box.id, "nohup python3 -m http.server 8000 >/tmp/http.log 2>&1 &")
api = os.environ.get("INSTABOX_API_URL", "https://api.instabox.sh")
r = httpx.post(  # the Python SDK has no ports call yet
    f"{api}/v1/machines/{box.id}/ports",
    headers={"X-API-Key": os.environ["INSTABOX_API_KEY"]},
    json={"port": 8000},
)
r.raise_for_status()
url = r.json()["url"]
```

```ts TypeScript theme={null}
await sbx.commands.run('python3 -m http.server 8000', { background: true })
const { url } = await sbx.ports.publish(8000)
```

Ports 1024 to 65535 can be published, up to 16 per machine. For a URL only people with a link can
open, publish it private: `sbx.ports.publish(8000, { access: 'private' })` in TypeScript, or
`"access": "private"` in the Python request. A private port's plain `url` answers `401`. Share
the response's `tokenUrl` instead, the URL with the token as `?_token=`. Only the publish that
creates or rotates the token returns `tokenUrl`, so keep it, or publish again with
`rotateToken: true` for a new one. See [Public and private ports](https://instabox.sh/docs/access/ports.md).

**6. Pause, resume and delete**

```python Python theme={null}
client.machines.pause(box.id)            # memory and processes are kept
box = client.machines.ensure_running(box.id)
client.machines.delete(box.id)
```

```ts TypeScript theme={null}
await sbx.pause()
const again = await Sandbox.connect(sbx.sandboxId) // resumes it, as on E2B
await again.kill()
```

## Use your E2B template

An E2B template is built from a Dockerfile. Build the same Dockerfile with Docker, push it to a
registry, and import it as a snapshot. The import needs a key with `images:write`, and creating from
the snapshot by name needs `images:read`, as well as the machine scopes from the first step.

```bash theme={null}
docker build -t ghcr.io/you/my-template:1 -f e2b.Dockerfile .
docker push ghcr.io/you/my-template:1
npx instabox snapshot import ghcr.io/you/my-template:1 --name my-template   # waits until it is saved
```

Then create machines from the snapshot. It is a snapshot, not a template: in Python, look it up by
name and pass its ID. In TypeScript, `Sandbox.create` takes the snapshot name and looks it up.

```python Python theme={null}
snap = next(s for s in client.snapshots.list() if s.name == "my-template")
box = client.machines.create(snapshot_id=snap.id)
```

```ts TypeScript theme={null}
const fromImage = await Sandbox.create('my-template') // a snapshot name, or its ID (sn_...)
```

Only `linux/amd64` images work. The image's `ENV` reaches every command, and its `CMD` runs as a
service. See [Custom images](https://instabox.sh/docs/machines/custom-images.md).

## What has no direct equivalent

| E2B | On InstaBox |
| - | - |
| `run_code` with a stateful Jupyter kernel and rich `results` (charts, tables) | Not available. Each `python3 -` run is a new process. To keep state, write it to a file, or run your own kernel or server in the machine and call it. Read text from `stdout`. |
| `files.watch_dir` | Not available. Poll `files.list`, or run `inotifywait -m` as a background command and read its log. |
| A sandbox that is deleted when its timeout ends | Python: `delete_after_seconds` on create deletes it. TypeScript: `archiveMs` stops it and keeps its disk; call `kill()` to delete. |
| `set_timeout` after create | Not available as a deadline. Idle machines pause on their own. To change the idle time, call `client.machines.update(box.id, auto_suspend_after_seconds=...)` (paid plans). |
| `get_host` in the Python SDK | No Python call. Use `POST /v1/machines/{id}/ports`, the CLI `instabox port publish`, or `instabox-port publish` inside the machine. TypeScript has `sbx.ports.publish(port)`. |
| Port URLs for ports below 1024 | Not available. Serve on 1024 or higher. |

## Migrate with your coding agent

Paste this prompt into your coding agent in the repository that uses E2B:

```text Migration prompt theme={null}
Migrate this repository from E2B sandboxes to InstaBox machines.

1. Find every use of the e2b, e2b-code-interpreter, @e2b/code-interpreter or e2b npm/pip packages,
   E2B_API_KEY, and e2b.Dockerfile. List the files before changing anything.
2. Replace the SDK:
   - Python: pip install instabox-sdk; `from instabox import Client`; `client = Client()`.
   - TypeScript: npm install @insforge/instabox-sdk@^0.8.0; `import { Sandbox } from '@insforge/instabox-sdk'`.
   - The key is INSTABOX_API_KEY (an ibk_ key), never E2B_API_KEY. It needs machines:write and
     machines:delete, plus images:read and images:write if the repo has an e2b.Dockerfile.
3. Map the calls:
   - Sandbox.create() with E2B's default template -> Python client.machines.create(metadata=...);
     TypeScript Sandbox.create('agent-browser', { metadata, envs }).
   - Sandbox.create(<your custom template>) -> an imported snapshot (see the e2b.Dockerfile step):
     Python `snap = next(s for s in client.snapshots.list() if s.name == "<name>")`, then
     client.machines.create(snapshot_id=snap.id); TypeScript Sandbox.create('<name>', { ... }).
     Never pass a snapshot name as Python's template=: the server answers 400 unknown template.
   - sbx.commands.run(cmd) -> Python client.machines.exec(box.id, cmd) (exit code in r.exit_code,
     no exception); TypeScript sbx.commands.run(cmd) (throws CommandExitError on a non-zero exit).
   - run_code(code) -> run "python3 -" with the code as stdin. There is no Jupyter kernel: each run is
     a new process, and only stdout and stderr come back.
   - sbx.files.write/read/list/remove/exists -> client.files.<same>(box.id, path, ...) in Python,
     sbx.files.<same>(path, ...) in TypeScript.
   - Paths under /home/user -> /data/home.
   - sbx.get_host(port) -> TypeScript: `const { url } = await sbx.ports.publish(port)` for a public
     port. For a link-only port, `const { tokenUrl } = await sbx.ports.publish(port, { access: 'private' })`
     and share tokenUrl: the plain url answers 401. Python has no ports call: POST
     $INSTABOX_API_URL/v1/machines/{id}/ports (default https://api.instabox.sh) with {"port": N}
     (add "access": "private" for a link-only port) and the X-API-Key header, raise on an error
     status, and use the "url" field, or "tokenUrl" for a private port. tokenUrl comes back only
     from the publish that creates or rotates the token ("rotateToken": true). Ports must be 1024
     to 65535.
   - sbx.pause() -> client.machines.pause(box.id) / sbx.pause(); Sandbox.connect(id) ->
     client.machines.ensure_running(id) / Sandbox.connect(id).
   - Sandbox.list() -> client.machines.list(metadata=...) / Sandbox.list({ metadata }).
   - sbx.kill() -> client.machines.delete(box.id) / sbx.kill().
   - timeout on create -> Python delete_after_seconds=...; TypeScript archiveMs (stops, does not delete).
   - e2b.Dockerfile -> docker build and push the image (linux/amd64), then
     `npx instabox snapshot import <image> --name <name>` with a key that has images:write. It waits
     until the snapshot is saved. Create from it as in the custom-template mapping above.
4. Stop and ask me before working around any of these, which have no direct equivalent:
   a stateful run_code kernel or its rich results (charts, tables, display data), files.watch_dir,
   set_timeout after create, and port URLs below port 1024.
5. Run the project's tests and its example scripts against InstaBox with INSTABOX_API_KEY set.
   Delete every machine the tests create. Report what you changed, what you could not map, and the
   test results.
```

## Troubleshooting

> [accordion] 401 missing\_api\_key
>
> `INSTABOX_API_KEY` is not set in the process that runs the code. E2B's `E2B_API_KEY` is not read.

> [accordion] 429 rate\_limited: sandbox start limit reached
>
> Each workspace may start only a set number of machines a minute; the free plan allows the fewest.
> A create counts, and so does a resume, including `Sandbox.connect` and `ensure_running` on a paused
> machine. Code that created many short-lived E2B sandboxes in a loop hits this first. Retry only when
> the error's code is `rate_limited`: wait a minute and retry, or reuse one machine across jobs
> instead of creating one per job. Another `429`, such as `quota_exceeded`, does not clear by
> waiting, so do not retry it. See [Limits](https://instabox.sh/docs/account/limits.md).

> [accordion] 403 missing\_scope
>
> The key lacks a scope the call needs. `snapshot import` needs `images:write`, and creating from a
> snapshot by name (or `client.snapshots.list()`) needs `images:read`. Create a key with those scopes
> added, or with **Full access**. See [API keys](https://instabox.sh/docs/security/api-keys.md).

> [accordion] 400 bad\_request: unknown template, when creating from your imported image
>
> An imported image is a snapshot, not a template. In Python pass `snapshot_id=snap.id`, not
> `template=`. See [Use your E2B template](#use-your-e2b-template).

> [accordion] 403 template\_not\_allowed
>
> The free plan creates only from some templates. A machine from your own imported snapshot is not
> limited. See [Pricing](https://instabox.sh/docs/account/pricing.md).

> [accordion] The port URL answers 502 or 503
>
> `502`: nothing listens on the port yet, or the server listens on `127.0.0.1` only. Listen on
> `0.0.0.0`. `503`: the machine is paused or stopped. A visit wakes a machine that paused itself.

> [accordion] A command fails with No such file or directory under /home/user
>
> The home directory is `/data/home`. Change the path, or use paths relative to HOME in commands.

> [card] [SDK quick start](https://instabox.sh/docs/integrations/sdk-quickstart.md)
>
> Every TypeScript call, with curl.

> [card] [Python SDK](https://instabox.sh/docs/reference/python-sdk.md)
>
> Every Python call and error.
