> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Python SDK

> Create machines, run commands, move files and delete machines from Python with instabox-sdk.

The Python SDK is [`instabox-sdk` on PyPI](https://pypi.org/project/instabox-sdk/). It runs on Python 3.9
to 3.13, has one dependency (`httpx`) and imports as `instabox`.

## Install and authenticate

```bash theme={null}
pip install instabox-sdk
export INSTABOX_API_KEY=ibk_...
```

Create the key on **API keys** in the dashboard: **Full access**, or a restricted key with
`machines:write` and `machines:delete`, which is enough for the quickstart. The snapshot calls also
need `images:write` (create), `images:read` (list and get) and `images:delete` (delete). See
[API keys](https://instabox.sh/docs/security/api-keys.md). `Client()` reads
`INSTABOX_API_KEY` and `INSTABOX_API_URL` (default `https://api.instabox.sh`), or takes
`Client(api_key=..., base_url=...)`.

## Quickstart

```python theme={null}
from instabox import Client

client = Client()  # reads INSTABOX_API_KEY (and INSTABOX_API_URL, default https://api.instabox.sh)

box = client.machines.create(name="my-box")  # template agent-browser; waits until ready
try:
    r = client.machines.exec(box.id, "uname -a")
    print(r.stdout, end="")

    r = client.machines.exec(box.id, ["sh", "-c", "echo oops >&2; exit 3"])
    print(r.exit_code, r.stderr, end="")  # 3 oops
finally:
    client.machines.delete(box.id)
```

`create` waits until the machine is ready. `exec` takes a shell string, run with `bash -lc`, or an argv
list, run with no shell. A non-zero exit code is a result, not an error: read it from `r.exit_code`.

## Pick up a machine later

A machine pauses after 30 idle minutes by default. `ensure_running` starts or resumes it, whatever its
state, and waits until it is ready. This block runs on its own: it makes `my-box` first, as an earlier
run would have, and deletes it at the end:

```python theme={null}
from instabox import Client

client = Client()
client.machines.create(name="my-box")  # stands in for a machine made earlier
try:
    box = client.machines.ensure_running("my-box")  # starts or resumes it, waits until ready
    print(client.machines.exec(box.id, "ls /data/home").stdout, end="")
    client.machines.update(box.id, metadata={"stage": "review"})  # labels: any plan
    # Paid plans only; on the free plan another idle time is 403 spec_not_allowed:
    # client.machines.update(box.id, auto_suspend_after_seconds=600)
finally:
    client.machines.delete("my-box", confirm="my-box")  # refuses to delete anything but my-box
```

## Create a macOS machine

From 0.5.0. A macOS machine runs on a Mac leased to your workspace for at least 24 hours, on the Pro
plan and higher ([macOS machines](https://instabox.sh/docs/machines/macos.md)). Read the minimum from the template, then accept
it:

> [!WARNING]
>
> Running this leases a Mac to your workspace for 24 hours, which you pay for even if you delete the
> machine sooner.

```python theme={null}
from instabox import Client, PermissionDeniedError

client = Client()
mac = next((t for t in client.templates.list() if t.id == "macos" and t.latest), None)
if mac is None:
    raise SystemExit("macOS is not offered to this account")
lease = mac.minimum_lease
print(f"at least {lease.hours} h, at most ${lease.minimum_usd:.2f} for them before disk")
try:
    box = client.machines.create("mac", template="macos", accept_minimum_lease=True)
    print(box.lease.charge_until)  # billed at least until then
except PermissionDeniedError as e:  # template_not_allowed: the free plan
    print(e.message)
```

Pick the Mac model with `model=` (`Template.models` lists them); each model has one size, the whole Mac.

Without `accept_minimum_lease=True` the create is refused with `BadRequestError`, code
`minimum_lease_required`. While the Mac's lease is inside its minimum, `stop` and `delete` of a
running machine are scheduled for the end of the minimum: the operation succeeds, and
`op.raw["scheduledAt"]` says when. `now=True` does them at once, and
`update(m, cancel_scheduled=True)` cancels them.

## Calls

Every call that takes `machine` accepts a machine ID (`m_...`) or its name.

| Call | Does |
| - | - |
| `client.machines.create(name=None, *, template=None, snapshot_id=None, metadata=None, ...)` | Create a machine; waits until it is ready |
| `client.machines.list()`, `get(machine)` | List your machines, get one |
| `client.machines.exec(machine, command, *, cwd=None, env=None, stdin=None, timeout=60)` | Run a command: `stdout`, `stderr`, `exit_code`, `timed_out`; from 0.4.0 `stdin` may be bytes |
| `client.machines.pause`, `resume`, `start`, `stop`, `ensure_running` | Lifecycle; each waits for its operation |
| `client.machines.update(machine, *, name=None, metadata=None, auto_suspend_after_seconds=None, ...)` | Rename, set labels or the idle pause |
| `client.machines.delete(machine, *, confirm=None)` | Delete |
| `client.files.write`, `read`, `read_bytes`, `list`, `exists`, `remove` | Files on the machine, at absolute paths (or, from 0.4.0, relative to HOME) |
| `client.files.read_stream`, `upload(machine, local_dir, remote_dir)`, `download(machine, remote_dir, local_dir)` | From 0.4.0: stream a file of any size, copy a whole directory each way |
| `client.snapshots.create(machine, name)`, `list()`, `get(id)`, `delete(id)` | Snapshots; `status` goes `pending` to `ready` |
| `client.templates.list()` | Templates; from 0.5.0 templates have `sizes` and `latest`, and a macOS template `minimum_lease` |

Every argument and model field is in the [SDK README](https://pypi.org/project/instabox-sdk/).

## Errors

Every error is an `instabox.InstaboxError`. API errors are `instabox.APIError` subclasses with the
API's `code`, `message`, `request_id`, `retryable` and `status`:

| Status | Class | Example `code` |
| - | - | - |
| 400 | `BadRequestError` | `bad_request`, `spec_not_allowed` (a size or disk the plan does not offer) |
| 401 | `AuthenticationError` | `missing_key`, `malformed_key`, `invalid_key`, `expired_key`, `revoked_key`; `missing_api_key` when no key is set |
| 403 | `PermissionDeniedError` | `missing_scope`, `template_not_allowed`, `spec_not_allowed` (another idle time on the free plan) |
| 404 | `NotFoundError` | `not_found` |
| 409 | `ConflictError` | `machine_not_running`, `operation_conflict` |
| 429 | `RateLimitError` | `quota_exceeded`, `rate_limited` |
| 5xx | `ServerError` | `no_capacity`, `host_unavailable` |

`spec_not_allowed` is a plan or size limit and takes the status of what was refused: `400` for a
size or disk size the plan does not offer, `403` for another idle time
(`auto_suspend_after_seconds`) on the free plan or starting a paid machine larger than its plan
now allows, `409` for restoring a snapshot that does not fit the machine's disk.

A failed operation raises `OperationFailedError`, a wait that runs out `WaitTimeoutError`. Every code:
[Errors](https://instabox.sh/docs/reference/errors.md).
