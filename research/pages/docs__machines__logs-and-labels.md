> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Logs and labels

> Read a machine's console log and recent operations, and tag machines with key=value labels.

A machine's logs show what it printed while it booted and what its recent operations did. Read them
when a machine does not boot, is unreachable, or failed to start or resume. Labels are `key=value`
tags that you put on machines so that you can find them again later.

```bash theme={null}
instabox logs job
```

## Read a machine's logs

`instabox logs` prints the last 200 lines of a machine's console log and its last 20 operations. The
operations are create, start, stop, pause, and resume, and each one is shown with its errors. Reading
the log works in every state and never wakes the machine, so you can use it on a machine that is
paused or stopped.

```bash CLI theme={null}
instabox logs job
```

```bash API theme={null}
curl https://api.instabox.sh/v1/machines/m_.../logs \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

The console log is the machine's serial console. It holds the kernel and systemd boot messages,
anything written to `/dev/console`, and errors from the VM monitor.

## Follow new output

When you want to watch a machine as it boots, you can keep the log open and print new console output
as the machine writes it. Add `-f` to follow the log, and use `--tail` to set how many earlier lines
are printed first. Press Ctrl-C to stop.

```bash theme={null}
instabox logs job --tail 50 -f
```

## Get the log as JSON

For a script that needs to parse the log, add `--json`. The CLI then prints the response of
`GET /v1/machines/{id}/logs`.

```bash theme={null}
instabox logs job --json
```

> [accordion] Over the API
>
> `GET /v1/machines/{id}/logs?tail=N` (scope `machines:read`) returns `console`, `cursor`, `state` and
> `operations` (oldest first). To follow it, pass `cursor` as `after` on the next request to get only
> newer output. When the console cannot be read, for example because the machine's disks are in
> storage, the response is still `200`, with `consoleUnavailable` and the operations.

## Read a command's output

The output of a command you ran is kept with its background job, and the console log does not hold
it. To read a command's output, pass the job's ID to `instabox jobs logs`.

```bash theme={null}
instabox jobs logs job <job-id>
```

## Read the guest journal

When the console log does not tell you enough, you can read the full guest journal by running
`journalctl` over exec. This command prints the journal of the current boot.

```bash theme={null}
instabox exec job -- sudo journalctl -b --no-pager
```

## Labels

A label is a `key=value` pair on a machine, and the API calls a machine's labels its `metadata`. You
give a machine labels when you create it, and you can then filter the list of machines by them. For
example, an agent can tag every machine it creates with `--meta session=ID` and find them later with
the same flag on `instabox ls`.

> [!WARNING]
>
> Labels are stored in plain text. Use them for labels, not secrets.

## Label a machine

To label a machine, pass one `--meta` flag for each label when you create it. The SDKs and the API
take all the labels together as `metadata`.

```bash CLI theme={null}
instabox create --name job --meta task=review
```

```ts TypeScript theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create({
  name: 'job',
  metadata: { task: 'review' },
})
```

```python Python theme={null}
from instabox import Client

client = Client()
client.machines.create(name="job", metadata={"task": "review"})
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "job", "metadata": {"task": "review"}}'
```

## Find machines by label

To list only the machines that carry a label, pass the same `--meta` flag to `instabox ls`. With
several `--meta` flags, a machine must match all of them. The server does the filtering, and it lists
only the machines of your own orgs.

```bash CLI theme={null}
instabox ls --meta task=review
```

```ts TypeScript theme={null}
const machines = await Sandbox.list({ metadata: { task: 'review' } })
```

```bash API theme={null}
curl "https://api.instabox.sh/v1/machines?metadata=task:review" \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

## Limits

* The host keeps the last 8 MiB of the console log across stops, pauses, and restarts.
* A machine has up to 32 labels. The key and value limits are under `metadata` in
  [Create parameters](https://instabox.sh/docs/machines/sizes.md#create-parameters).
