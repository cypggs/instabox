> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Migrate from Daytona

> Move code that creates Daytona sandboxes to InstaBox machines, call by call, in Python or TypeScript.

This page is for code that uses the Daytona SDK (`daytona` in Python, `@daytonaio/sdk` in TypeScript) and
should run on InstaBox instead. A Daytona **sandbox** is an InstaBox **machine**: a full Linux VM with a
persistent disk at `/data`, created from a template or a snapshot. Most calls map one to one. The
differences are in [What has no equivalent](#what-has-no-equivalent).

## Concept map

| Daytona | InstaBox |
| - | - |
| `Daytona()` with `DAYTONA_API_KEY` | Python `Client()`, TypeScript `Sandbox`, with `INSTABOX_API_KEY` |
| `daytona.create(...)` | Python `client.machines.create(...)`, TypeScript `Sandbox.create(...)` |
| Sandbox ID | Machine ID, `m_...` |
| Default snapshot, `language="python"` | The default template, `agent-browser`: Python 3, Node.js, git, curl and a desktop |
| A snapshot built from an image | A snapshot imported from an image: `instabox snapshot import <image>` |
| `sandbox.create_snapshot(name)` | Python `client.snapshots.create(machine, name)` |
| `process.exec(cmd, cwd, env, timeout)` | Python `client.machines.exec(machine, cmd, cwd=, env=, timeout=)`, TypeScript `sbx.commands.run(cmd, opts)` |
| `process.code_run(code)` | Run the interpreter: `exec(machine, ["python3", "-c", code])` |
| Sessions, `run_async` | TypeScript `sbx.commands.run(cmd, { background: true })`, CLI `instabox exec --background` |
| `fs.upload_file`, `fs.download_file` | Python `client.files.write`, `read_bytes`. TypeScript `sbx.files.write`, `read` |
| `fs.list_files`, `fs.delete_file` | `files.list`, `files.remove` |
| `git.clone`, `git.commit`, `git.push` | Run `git` with `exec` |
| `get_preview_link(port)` | Publish the port: `POST /v1/machines/{id}/ports`, TypeScript `sbx.ports.publish(port)` (SDK 0.8.0 or later) |
| `x-daytona-preview-token` header | A private port's token, sent as `Authorization: Bearer ibp_...` |
| `auto_stop_interval` | The idle pause, `auto_suspend_after_seconds` (Python), `autoSuspendMs` (TypeScript) |
| `auto_archive_interval` | Not needed: a paused or stopped machine keeps its disk until you delete it |
| `auto_delete_interval`, `ephemeral` | `delete_after_seconds` at create (Python), `instabox create --delete-after` |
| `sandbox.stop()`, `start()` | `stop`, `start`. To keep memory and processes too, use `pause` and `resume` |
| `sandbox.delete()` | Python `client.machines.delete(machine)`, TypeScript `sbx.kill()` |
| `labels`, `set_labels`, `list(labels)` | `metadata` at create, `update(machine, metadata=)`, `list(metadata=)` |
| `daytona.get(id)` | Python `client.machines.get(machine)`, TypeScript `Sandbox.connect(id)` |
| `get_work_dir()`, `get_user_home_dir()` | HOME is `/data/home`, on the persistent disk. It starts with a few files of the template, such as `README.md` and `AGENTS.md` |

## Move your code

**1. Install the SDK and set the key**

Create an API key on **API keys** in the dashboard. See [API keys](https://instabox.sh/docs/security/api-keys.md).

```bash Python theme={null}
pip uninstall daytona
pip install instabox-sdk
export INSTABOX_API_KEY=ibk_...
```

```bash TypeScript theme={null}
npm uninstall @daytonaio/sdk
npm install @insforge/instabox-sdk@^0.8.0
export INSTABOX_API_KEY=ibk_...
```

The TypeScript steps need SDK 0.8.0 or later, the first version with `sbx.ports`.

Remove `DAYTONA_API_KEY`, `DAYTONA_API_URL` and `DAYTONA_TARGET`. InstaBox machines have no region to
pick.

**2. Create a machine**

```python Daytona theme={null}
from daytona import Daytona, CreateSandboxFromSnapshotParams

daytona = Daytona()
sandbox = daytona.create(CreateSandboxFromSnapshotParams(
    language="python",
    labels={"customer": "acme"},
    auto_stop_interval=15,
))
```

```python InstaBox theme={null}
from instabox import Client

client = Client()
box = client.machines.create(
    metadata={"customer": "acme"},
    delete_after_seconds=3600,  # optional: like ephemeral, deleted after an hour
)
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create('agent-browser', { metadata: { customer: 'acme' } })
```

`create` waits until the machine runs. The machine pauses itself after 30 idle
minutes, which replaces Daytona's auto-stop. To set another idle time on a paid plan, pass
`auto_suspend_after_seconds` (Python) or `autoSuspendMs` (TypeScript).

**3. Run commands and code**

```python Daytona theme={null}
r = sandbox.process.exec("ls -la", cwd="/home/daytona", timeout=60)
print(r.exit_code, r.result)

r = sandbox.process.code_run('print(6 * 7)')
print(r.result)
```

```python InstaBox theme={null}
r = client.machines.exec(box.id, "ls -la", cwd="/data/home", timeout=60)
print(r.exit_code, r.stdout)

r = client.machines.exec(box.id, ["python3", "-c", "print(6 * 7)"])
print(r.stdout)
```

```ts TypeScript theme={null}
import { CommandExitError } from '@insforge/instabox-sdk'

const r = await sbx.commands.run('ls -la', { cwd: '/data/home', timeoutMs: 60_000 })
console.log(r.exitCode, r.stdout)

const code = await sbx.commands.run(`python3 -c 'print(6 * 7)'`)
console.log(code.stdout)

try {
  await sbx.commands.run('exit 3')
} catch (e) {
  if (e instanceof CommandExitError) console.log(e.exitCode)  // 3
}
```

Daytona's `result` is InstaBox's `stdout`. InstaBox also returns `stderr` on its own. In Python, a
non-zero exit is a result, as in Daytona: read `exit_code`. In TypeScript, `sbx.commands.run` throws
`CommandExitError` on a non-zero exit, and the error carries `exitCode`, `stdout` and `stderr`.

**4. Move files**

```python Daytona theme={null}
sandbox.fs.upload_file(b"hello", "/home/daytona/data/in.txt")
data = sandbox.fs.download_file("/home/daytona/data/in.txt")
for f in sandbox.fs.list_files("/home/daytona/data"):
    print(f.name, f.is_dir, f.size)
sandbox.fs.delete_file("/home/daytona/data/in.txt")
```

```python InstaBox theme={null}
client.files.write(box.id, "/data/home/data/in.txt", b"hello")  # creates the directory
data = client.files.read_bytes(box.id, "/data/home/data/in.txt")
for f in client.files.list(box.id, "/data/home/data"):
    print(f.name, f.type == "dir", f.size)
client.files.remove(box.id, "/data/home/data/in.txt")
```

```ts TypeScript theme={null}
await sbx.files.write('/data/home/data/in.txt', 'hello')
const data = await sbx.files.read('/data/home/data/in.txt', { format: 'bytes' })
for (const f of await sbx.files.list('/data/home/data')) console.log(f.name, f.type)
await sbx.files.remove('/data/home/data/in.txt')
```

To upload a local file from disk, pass an open file: `client.files.write(box.id, path, open("in.txt", "rb"))`.
To copy a whole directory, use `client.files.upload(box.id, "local_dir", "/data/home/app")` and
`download`. For `create_folder`, `move_files`, `find_files`, `replace_in_files` and
`set_file_permissions`, run `mkdir -p`, `mv`, `grep`, `sed` or `chmod` with `exec`.

**5. Run git**

```python Daytona theme={null}
sandbox.git.clone("https://github.com/octocat/Hello-World.git", "/home/daytona/repo")
```

```python InstaBox theme={null}
r = client.machines.exec(box.id, "git clone --depth 1 https://github.com/octocat/Hello-World.git /data/home/repo")
assert r.exit_code == 0, r.stderr
```

Every other git call is a `git` command too: `git -C /data/home/repo add -A`, `commit`, `push`. For a
private repository, give the machine a token from your secret store as an `env` value. See
[Environment and secrets](https://instabox.sh/docs/machines/environment.md).

**6. Serve a port at a URL**

Start the server detached, so it keeps running after the command returns. Then publish the port.

```python Daytona theme={null}
sandbox.process.exec("python3 -m http.server 8000 &")
link = sandbox.get_preview_link(8000)
print(link.url, link.token)  # send the token as x-daytona-preview-token
```

```python InstaBox theme={null}
import os, httpx

client.machines.exec(box.id, "setsid nohup python3 -m http.server 8000 >/tmp/http.log 2>&1 < /dev/null &")
api = os.environ.get("INSTABOX_API_URL", "https://api.instabox.sh")
port = httpx.post(f"{api}/v1/machines/{box.id}/ports",
                  headers={"X-API-Key": os.environ["INSTABOX_API_KEY"]},
                  json={"port": 8000}).json()
print(port["url"])  # https://8000-<id>.<domain>, public
```

```ts TypeScript theme={null}
await sbx.commands.run('python3 -m http.server 8000', { background: true })
const port = await sbx.ports.publish(8000)
console.log(port.url)
```

`sbx.ports` needs `@insforge/instabox-sdk` 0.8.0 or later. Older versions have no `ports`, so call
the API as the Python tab does. The Python SDK has no ports call yet, so the Python tab calls the API with `httpx`, which
`pip install instabox-sdk` installs. The URL answers once the server listens, so retry the first
request for a few seconds. A port is public by default, like a Daytona sandbox with `public=True`. For a link only some people
can open, publish it with `"access": "private"` (TypeScript `{ access: 'private' }`). Share the
returned `tokenUrl`, or send `Authorization: Bearer <token>` from scripts. A visit to the URL wakes a
machine that paused itself. See [Public and private ports](https://instabox.sh/docs/access/ports.md).

**7. Stop, pause and delete**

```python Daytona theme={null}
sandbox.stop()
sandbox.start()
sandbox.archive()
sandbox.delete()
```

```python InstaBox theme={null}
client.machines.pause(box.id)    # keeps memory and running processes
client.machines.resume(box.id)
client.machines.stop(box.id)     # shuts down, keeps /data
client.machines.start(box.id)
client.machines.delete(box.id)
```

```ts TypeScript theme={null}
await sbx.pause()
await sbx.resume()
await sbx.stop()
await sbx.kill()
```

There is no archive step. A paused or stopped machine stays, with its disk, until you delete it, and
its disk counts as stored data. See [Storage and retention](https://instabox.sh/docs/machines/storage.md).

## Find machines by label

```python Daytona theme={null}
for s in daytona.list(ListSandboxesQuery(labels={"customer": "acme"})):
    print(s.id)
sandbox.set_labels({"stage": "review"})
```

```python InstaBox theme={null}
for m in client.machines.list(metadata={"customer": "acme"}):
    print(m.id)
client.machines.update(box.id, metadata={"stage": "review"})
```

```ts TypeScript theme={null}
for (const m of await Sandbox.list({ metadata: { customer: 'acme' } })) console.log(m.sandboxId)
await sbx.setMetadata({ stage: 'review' })
```

## Bring your image

Daytona builds snapshots from images. InstaBox imports a Docker or OCI image from a registry as a
snapshot, then creates machines from it:

```bash theme={null}
instabox snapshot import python:3.12-slim --name py312
```

```python theme={null}
snap = next(s for s in client.snapshots.list() if s.name == "py312")
box = client.machines.create(snapshot_id=snap.id)
```

In TypeScript, `Sandbox.create('py312')` takes the snapshot name. A Dockerfile must be built and pushed to
a registry first. To save a machine you set up by hand, use `client.snapshots.create(box.id, "my-env")`.
See [Custom images](https://instabox.sh/docs/machines/custom-images.md) and [Snapshots](https://instabox.sh/docs/machines/snapshots.md).

## What has no equivalent

| Daytona | On InstaBox |
| - | - |
| Volumes shared between sandboxes | Not available. Each machine has its own persistent `/data`. Copy data with `files.upload` and `download`, or [fork](https://instabox.sh/docs/machines/fork.md) a machine |
| Declarative image builder in the SDK | Not available. Build a Dockerfile, push it, then `instabox snapshot import` |
| Sessions that keep shell state between calls | Not available. Each `exec` is a new shell. Chain steps in one command, or run a background job |
| `code_run` charts and artifacts | Not available. Write files from your code and read them back |
| Git helpers with credentials | Run `git` with `exec`, and pass the token as an `env` value |
| `find_files`, `replace_in_files`, `set_file_permissions` | Run `grep`, `sed` and `chmod` with `exec` |
| Signed preview URLs that expire | A private port's token lasts until you rotate it: publish again with `rotateToken` |
| LSP server | Not available. Run a language server in the machine yourself |
| `target` region | Not available. Machines have no region to choose |
| A ports call in the Python SDK | Use the HTTP call above, or run `instabox-port publish 8000` in the machine |

## Let your coding agent do the port

Paste this prompt into your coding agent, in your repository:

```text theme={null}
Port this repository from the Daytona SDK to the InstaBox SDK. Follow https://instabox.sh/docs/migrate/from-daytona.

Mapping to apply:
- Daytona() / DAYTONA_API_KEY -> Python (pip install instabox-sdk): from instabox import Client; client = Client(); TypeScript (npm install @insforge/instabox-sdk@^0.8.0): import { Sandbox } from '@insforge/instabox-sdk'. Key: INSTABOX_API_KEY.
- daytona.create(params) -> client.machines.create(metadata=..., delete_after_seconds=...) / Sandbox.create('agent-browser', { metadata }). Sandbox IDs become machine IDs (m_...).
- sandbox.process.exec(cmd, cwd, env, timeout) -> client.machines.exec(box.id, cmd, cwd=, env=, timeout=) (.result becomes .stdout; a non-zero exit is a result) / sbx.commands.run(cmd, { cwd, envs, timeoutMs }) (a non-zero exit throws CommandExitError with exitCode).
- sandbox.process.code_run(code) -> client.machines.exec(box.id, ["python3", "-c", code]).
- fs.upload_file / download_file / list_files / delete_file -> client.files.write / read_bytes / list / remove (sbx.files.write / read / list / remove). Paths: /home/daytona becomes /data/home.
- sandbox.git.* -> run git with exec.
- get_preview_link(port) -> start the server detached (setsid nohup ... &), then POST {INSTABOX_API_URL}/v1/machines/{id}/ports with {"port": N} (TypeScript: sbx.ports.publish(N)); use the returned url.
- auto_stop_interval -> auto_suspend_after_seconds / autoSuspendMs; auto_delete_interval or ephemeral -> delete_after_seconds; auto_archive_interval -> remove it.
- stop/start -> stop/start (pause/resume to keep memory); delete -> client.machines.delete / sbx.kill.
- labels -> metadata; daytona.get -> client.machines.get / Sandbox.connect; daytona.list(labels) -> client.machines.list(metadata=) / Sandbox.list({ metadata }).

Then run the project's tests against a real InstaBox machine and fix what fails. Stop and ask me before working around anything in the "What has no equivalent" table on that page (shared volumes, the declarative image builder, sessions, code_run charts, LSP, region targets). Do not invent InstaBox calls that the page does not name.
```

## Troubleshooting

> [accordion] A background server stops when the command returns
>
> Start it with `setsid nohup <cmd> >/tmp/log 2>&1 < /dev/null &`, so it leaves the command's session.
> In TypeScript, use `sbx.commands.run(cmd, { background: true })`.

> [accordion] The port URL answers 503
>
> The machine is paused by you, stopped, or the server in it is not listening yet. Resume or start the
> machine, then check the server with `curl localhost:<port>` through `exec`.

> [accordion] Create fails with template\_not\_allowed or spec\_not\_allowed
>
> The free plan creates from agent-browser, android, omarchy, ci only, and each plan has fixed sizes. See
> [Sizes and create options](https://instabox.sh/docs/machines/sizes.md).

> [accordion] A command has no git, Node.js or Python
>
> The default template, `agent-browser`, has all three. A machine from an imported image has what the
> image has, so install the tool in the image or with `exec`.

> [card] [Python SDK](https://instabox.sh/docs/reference/python-sdk.md)
>
> Every Python call and error.

> [card] [SDK quick start](https://instabox.sh/docs/integrations/sdk-quickstart.md)
>
> The same steps in TypeScript and HTTP.
