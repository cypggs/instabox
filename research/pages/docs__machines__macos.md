> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# macOS machines

> Create a macOS machine on a Mac leased to your workspace, on the Pro plan and higher.

A macOS machine runs macOS 26 Tahoe with Xcode on a Mac that is leased to your workspace for at least
24 hours. You need the Pro plan or higher. When you create the machine, you accept the 24-hour minimum:

```bash CLI theme={null}
instabox create --name mac --template macos --accept-24h-minimum
```

```python Python theme={null}
m = client.machines.create("mac", template="macos", accept_minimum_lease=True)
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "mac", "templateId": "macos", "acceptMinimumLease": true}'
```

In the dashboard, open **Create Machine**, then:

1. Under **Snapshot**, select **macOS 26 Tahoe**. Your macOS snapshots are listed under **My Snapshots**.
2. Under **Resources**, select a Mac model. Each model shows the machine's vCPU, memory and disk. A model
   with no Mac free right now is greyed out.
3. Select the box that accepts the 24-hour minimum.
4. Select **Create**.

## Price and the 24-hour minimum

Apple's macOS license lets InstaBox lease a Mac to only one customer at a time, for at least 24 hours.
So your workspace leases a whole Mac, at one price per Mac per hour:

| Model | `--model` | Per hour | 24-hour minimum |
| - | - | - | - |
| M2 | `m2` | \$0.45 | \$10.80 |
| M2 Pro | `m2pro` | \$0.85 | \$20.40 |
| M4 | `m4` | \$0.75 | \$18.00 |
| M4 Pro | `m4pro` | \$1.25 | \$30.00 |

* **The price is per Mac, all in.** The machine's vCPUs and memory are not billed on top.
* **Minimum: 24 hours** per Mac. You pay it even if you stop or delete the machine sooner.
* **After 24 hours**, the lease is billed per second until the machine stops.
* **Disk** counts as storage, as every machine's does ([Pricing](https://instabox.sh/docs/account/pricing.md)).

`instabox templates --json` (`GET /v1/templates`) shows the current price. Each model's
`minimumLease` has `hours`, `hourUSD` (the price per Mac-hour) and `minimumUSD` (the 24-hour minimum).

The minimum starts when the first machine runs on the Mac. Until it ends, the Mac belongs to your
workspace, and no other workspace's machine runs on it. **Usage** and **Billing** show a **macOS** line
in Mac-hours.

## Mac models

A macOS machine takes a whole Mac: one machine per Mac, at the Mac's full size. Pick the model with
`--model` (Python `model=`, the dashboard's **Resources**). Without it, you get the template's first
model. The models and their `--model` names are in the price table above.

`instabox templates` (`GET /v1/templates`, the template's `models`) and the dashboard's **Resources**
show each model's vCPUs, memory and price.

```bash theme={null}
instabox create --name mac --template macos --model m4pro --accept-24h-minimum
```

When no Mac of the model is free, the create is refused at once with `503 no_capacity`, and no machine
is made. The dashboard greys the model out. Try again in a few minutes.

## User and password

A macOS machine's user is `admin`, and its password is `admin`. macOS asks for the password when you
unlock the screen or change a protected setting, for example in **Privacy & Security**. SSH, `exec`
and the desktop in the dashboard's **Computer** tab do not use it, and `sudo` does not ask for it. The
machine's **Details** tab in the dashboard shows the user and the password.

## Environment variables

Set variables at create, as for any machine:

```bash theme={null}
instabox create --name mac --template macos \
  --env API_URL=https://example.com --accept-24h-minimum
```

A login shell, `exec` and the desktop's Terminal all see them. zsh reserves some names, for example
`PATH`, `HOME` and `status`, and a create that sets one of them is refused. To set one of these, use
`exec`'s `env` for that command.

## Stop or delete inside the minimum

You pay for the Mac's 24-hour minimum either way. So while the Mac's lease is inside its minimum, a
stop or a delete of a running macOS machine is scheduled: the machine keeps running, and the stop or
delete happens when the minimum ends.

To stop or delete it at once, add `--now`. You still pay for the minimum:

```bash theme={null}
instabox stop mac --now
instabox rm mac --now
```

To reset a broken machine, delete it with `--now` and create a new one. If your workspace still leases
the Mac, the new machine uses the same lease.

To cancel a scheduled stop or delete, run `stop` or `rm` with `--cancel`:

```bash theme={null}
instabox stop mac --cancel
```

`instabox get mac` shows the lease and the scheduled time. In the dashboard, the machine's
**Details** tab shows **Mac lease** and **Stops at** or **Deleted at**, with **Cancel**.

After the minimum, a stop or a delete happens at once.

## Plan requirement

Only the Pro plan and higher can create a macOS machine. On the free plan:

* A create is refused with `403 template_not_allowed`.
* The dashboard lists **macOS 26 Tahoe** under **Get Pro to Unlock**, with **Upgrade Plan**.

See [Pricing](https://instabox.sh/docs/account/pricing.md) for the plans.

> [accordion] Errors
>
> * `400 minimum_lease_required`: the create did not accept the minimum. The message gives the
>   price. Send `acceptMinimumLease: true` (CLI `--accept-24h-minimum`, Python
>   `accept_minimum_lease=True`). On a terminal, the CLI asks before it sends the create again.
>   Without a terminal, for example in an agent or CI, it fails.
> * `400 bad_request`: an environment variable name that zsh reserves.
> * `403 template_not_allowed`: the workspace is on the free plan.
> * `402 spend_limit_reached`: the workspace's spend limit leaves less than the minimum this
>   month.
> * `402 subscription_required`: the workspace's plan ends before the minimum would.
> * `503 no_capacity`: no Mac is free. The create is refused at once and makes no machine. Try again
>   in a few minutes.

> [accordion] Start a stopped machine after its lease ended
>
> A stopped macOS machine keeps its disk. If its lease ended, starting it starts a new lease and a
> new 24-hour minimum. The start needs the same consent: `instabox start mac     --accept-24h-minimum`, or `client.machines.start(m, accept_minimum_lease=True)`. Forks and
> creates from a macOS snapshot need it too.

> [accordion] What a macOS machine cannot do
>
> A macOS machine cannot be paused, and it is never paused when idle. It runs until you stop it or
> its max run time ends ([Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)).

> [card] [Pricing](https://instabox.sh/docs/account/pricing.md)
>
> Plans and the unit prices of usage.

> [card] [Errors](https://instabox.sh/docs/reference/errors.md#minimum-lease-required)
>
> What each error code means and what to do.
