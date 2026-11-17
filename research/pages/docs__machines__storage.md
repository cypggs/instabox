> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Storage and retention

> How long a machine and its snapshots are kept, archive and delete deadlines, and storage sync.

A machine's files stay when you pause or stop it, and InstaBox keeps the machine and every snapshot
you save until **you** delete them.

For a machine that should not live forever, such as one a CI job or an agent creates, you can set a
deadline when you create it. The machine is then stopped or deleted for you, even if the program that
created it has crashed.

```bash theme={null}
instabox create --name job --delete-after 1h
```

## How long data is kept

By default a machine and its snapshots stay until you delete them, whether the machine is running,
paused, or stopped. The one exception is when the workspace that owns them is disabled or deleted.

> [!WARNING]
>
> A workspace that is **disabled** (its subscription cancelled, or unpaid past the grace period) keeps
> its machines and snapshots for 30 days, then they are deleted. A workspace you **delete** has them
> deleted right away, with no recovery window.

Data you keep still counts toward your plan's limits and your bill, even while the machine is
stopped. The free plan caps stored data at 200 GiB. Paid plans bill every allocated
GiB of each machine's disk, and the monthly credit pays for it first. See [Pricing](https://instabox.sh/docs/account/pricing.md).

## Delete a machine at a deadline

Use a delete deadline for a machine you only need for a while, so that it cannot be left behind.
When the deadline is reached, the machine is **deleted** together with its files, whatever state it is
in.

You set the deadline when you create the machine. The CLI takes a duration with `--delete-after`, and
the API takes a number of seconds as `deleteAfterSeconds`. The machine then shows the deadline as
`deleteAt`.

```bash CLI theme={null}
instabox create --name job --delete-after 1h
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" -d '{"name": "job", "deleteAfterSeconds": 3600}'
```

## Stop a machine at a deadline

Use an archive deadline when you want to keep the files but stop paying for runtime. When the
deadline is reached, a running machine is **stopped**. Its disks stay in object storage, so you can
start it again later and find your files where you left them.

The API takes a number of seconds as `archiveSeconds`, and the TypeScript SDK takes milliseconds as
`archiveMs`. The machine then shows the deadline as `archiveAt`.

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create({ name: 'job', archiveMs: 60 * 60 * 1000 })
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" -d '{"name": "job", "archiveSeconds": 3600}'
```

> [accordion] The archive deadline is one-shot
>
> Resume does not clear `archiveAt`. Once the deadline passes, running or not, `archiveAt` is cleared. A
> machine that was already paused or stopped at that moment is not archived on a later start.

## Set both deadlines for a one-shot command

`instabox run` creates a machine for one command and deletes it afterwards. It sets both deadlines
from its `--timeout`, so the machine is cleaned up even if your terminal closes, and you do not set
the deadlines yourself.

```bash theme={null}
instabox run --timeout 10m -- npm test
```

## Wait for storage sync

When you stop or pause a machine, its newest disk state is uploaded to object storage in the
background. Most of the time you can ignore this. It matters when you want to be sure the machine can
start on any host, because until the upload finishes a start can only
place the machine on the host it last ran on.

While the upload runs, the machine's `storage.synced` field is `false`. To wait for the upload, read
the machine until `synced` is `true`. A snapshot save waits for the upload by itself, so you do not
need to wait before you save one.

```bash theme={null}
curl https://api.instabox.sh/v1/machines/m_... -H "X-API-Key: $INSTABOX_API_KEY"
```

You can also pause or stop the machine a second time. When the machine is not synced yet, the
second call starts the upload at once and returns when the machine is synced.

```bash theme={null}
instabox suspend dev
```

> [accordion] How long the upload takes
>
> The upload runs in the background. It takes about ten seconds, plus any wait behind other machines'
> uploads, so after many pauses at once it can take minutes. There is no fixed bound.

## Limits

* Both deadlines are optional. For no deadline, leave the field out or send `0`.
* `archiveSeconds` and `deleteAfterSeconds` each take a value from 60 to 86400.
* Stopped machines have no forced lifetime. They are not purged after 7 or 14 days.
* You cannot choose a region. A machine has no `region` field, and a `region` in a create request is
  refused with `400 unknown_field`. The platform decides where a machine runs.

> [card] [Snapshots](https://instabox.sh/docs/machines/snapshots.md)
>
> Save a machine once and start new ones from it.

> [card] [Limits](https://instabox.sh/docs/account/limits.md)
>
> Machines, disk and snapshots per plan.
