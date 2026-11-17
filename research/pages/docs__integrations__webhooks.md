> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Webhooks

> Signed POSTs when a machine is ready, pauses, stops, fails or is deleted.

A webhook is a signed JSON request that InstaBox POSTs to your server when a machine or a snapshot
changes state, so you do not need to poll `GET /v1/machines/{id}` to find out. You can use webhooks
to start work as soon as a machine is ready, to react when a machine pauses or stops, or to learn
that a machine failed or was deleted.

To receive events, you register an HTTPS endpoint for an org. InstaBox then sends an event to that
endpoint each time one of the org's machines or snapshots changes state.

```bash CLI theme={null}
instabox webhooks create https://example.com/hook \
  --event sandbox.started --event sandbox.error
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/orgs/$ORG/webhooks \
  -H "Authorization: Bearer $INSTABOX_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://example.com/hook",
    "events": ["sandbox.started", "sandbox.error"]
  }'
```

The CLI command subscribes the endpoint to the event types you name with `--event`, so your server
only receives the events it handles. When you leave `--event` out, the endpoint subscribes to every
type. You can find the types under [Reference](#reference).

The command prints the signing secret (`whsec_…`) once, so save it when you create the endpoint. You
need the secret later to [verify the signature](#verify-the-signature) of each request.

You can also do all of this on the dashboard's **Webhooks** page, which shows a delivery log too.

## List your endpoints

The other commands on this page take an endpoint's ID, which looks like `wh_...`. When you do not
have the ID at hand, run `ls` to list the endpoints of the org and find it.

```bash theme={null}
instabox webhooks ls
```

## Send a test event

You can check your handler before any machine changes state by sending a test event. A test send
delivers a `webhook.test` event to the endpoint. Only a test send produces that event.

```bash theme={null}
instabox webhooks test wh_...
```

## Read the delivery log

When an event did not arrive, or you want to confirm that one did, read the endpoint's delivery
log. `deliveries` lists what InstaBox sent to the endpoint.

```bash theme={null}
instabox webhooks deliveries wh_...
```

## Verify the signature

Every request is signed with the endpoint's secret, so your handler can check that a request came
from InstaBox before it acts on it. Requests follow
[Standard Webhooks](https://www.standardwebhooks.com/), which means you can verify them with its
libraries and do not have to write the check yourself.

The signature and the values it covers arrive in these headers.

| Header | Value |
| - | - |
| `webhook-id` | the event ID, the same on every retry. Dedupe on it. |
| `webhook-timestamp` | Unix seconds. Reject timestamps more than 5 minutes off. |
| `webhook-signature` | `v1,` and the base64 HMAC-SHA256 of `<webhook-id>.<webhook-timestamp>.<raw body>`, keyed with the base64-decoded part of the secret after `whsec_` |

If you do write the check yourself, this is how it looks in Node.js.

```js theme={null}
import crypto from 'node:crypto';
const key = Buffer.from(secret.slice('whsec_'.length), 'base64');
const want = crypto
  .createHmac('sha256', key)
  .update(`${id}.${timestamp}.${rawBody}`)
  .digest();
const ok = header.split(' ').some((s) => {
  const got = Buffer.from(s.startsWith('v1,') ? s.slice(3) : '', 'base64');
  return got.length === want.length && crypto.timingSafeEqual(got, want);
});
```

The code checks every signature in the header and accepts the request when one of them matches,
because the header can list more than one. It compares with `crypto.timingSafeEqual`, which takes
constant time.

## Rotate the signing secret

Rotate the secret when it may have leaked, or when you replace secrets on a schedule. `rotate`
gives the endpoint a new signing secret and prints it once, so save it as you did the first one.

```bash theme={null}
instabox webhooks rotate wh_...
```

> [!NOTE]
>
> After a rotation, the header lists two signatures for 24 hours, which gives you time to move your
> handler to the new secret. Accept either signature, and compare in constant time.

## Answer each delivery in time

InstaBox counts a delivery as done when your endpoint answers `2xx` within 10 seconds of
connecting. Reply first and do the work afterwards, so that slow work does not turn into a timeout.

The events of one machine arrive in order, and they stay in order across retries, because a later
event waits while an earlier one is retried. Events of different machines interleave, so when you
need an order between machines, sort those events by `createdAt`.

Any other answer is retried, and so is a timeout. The retries come after 5 s, 5 min, 30 min, 2 h,
5 h, 10 h, and 10 h, and then the delivery fails. A delivery that is still pending 3 days after its
event fails as expired.

## Retry a delivery

When a delivery has failed and your handler works again, you can send that delivery once more
without waiting for a new event. `retry` sends a finished delivery again, and it takes the endpoint
ID and the delivery ID.

```bash theme={null}
instabox webhooks retry ID DELIVERY
```

## Turn a disabled endpoint back on

InstaBox disables an endpoint that keeps failing, and you turn it back on yourself once your server
is fixed.

> [!WARNING]
>
> An endpoint that fails every attempt for 5 days is disabled, and the workspace's owner and admins
> get an email.

To turn the endpoint back on, update it with `--enable`.

```bash theme={null}
instabox webhooks update ID --enable
```

## Limits

* The URL must be public `https`. Redirects are not followed.
* Deliveries are kept 30 days.

> [accordion] Who can manage endpoints, and more CLI
>
> The org's owner or admin can manage endpoints, signed in or with an API key of that org. The key's
> scope decides what it can do: `machines:read` lists and reads deliveries, `machines:write`
> creates, updates, rotates, tests, and retries, and `machines:delete` deletes.
>
> With `INSTABOX_API_KEY` set, `--org` defaults to the key's org. `GET /v1/orgs` lists your orgs, and
> an API key sees only its own.
>
> Every `webhooks` subcommand takes `--json` or `-o json`, including `events`, `rm`, `test`, and
> `retry`. The routes are under `/v1/orgs/{id}/webhooks` in the
> [API reference](https://instabox.sh/docs/api-reference/introduction.md).

## Reference

### Events

These are the event types, and the table says when InstaBox sends each one.

| Type | When |
| - | - |
| `sandbox.created` | the machine is accepted (state `creating`) |
| `sandbox.started` | it is `running` and ready for exec and SSH (after create, start, resume, or wake) |
| `sandbox.stopped` | it is `stopped` |
| `sandbox.suspended` | it is paused (`suspended`): by you, when idle, or by a plan limit |
| `sandbox.error` | it is in `error` |
| `sandbox.deleted` | it is `deleted` |
| `snapshot.created`, `snapshot.failed`, `snapshot.deleted` | a snapshot is ready, failed to upload, or is deleted |
| `webhook.test` | only from a test send |

### Event body

This example is the event for a machine that was paused because it was idle. The `type` names the
event, and `data` describes the machine and the change.

```json theme={null}
{
  "id": "evt_…",
  "type": "sandbox.suspended",
  "createdAt": "…",
  "orgId": "org_…",
  "data": {
    "sandbox": { "id": "m_…", "name": "worker-1", "metadata": {} },
    "previousState": "suspending",
    "state": "suspended",
    "reason": "idle",
    "operationId": "op_…"
  }
}
```

`reason` says what moved the machine, so you can tell a pause you asked for from one that happened
when the machine was idle. Its values include `user`, `idle`, `archive`, `wake`, and `system`, which
means the server observed the change, such as a crash.

> [card] [Lifecycle](https://instabox.sh/docs/machines/lifecycle.md)
>
> The states these events report.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> Why a machine pauses when idle or at a plan limit.
