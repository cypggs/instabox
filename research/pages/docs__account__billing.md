> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Billing and usage

> This month's cost and credit balance, where to read them, and how they are computed.

To see what this month has cost so far and what credit is left, run `instabox usage`.

## Read this month's cost and credits

```bash theme={null}
instabox usage           # counts and limits, then Cost, Allowance and Credits
instabox usage --json    # the same, with the figures under cost
```

To read the same figures over the API, use any key that has `machines:read` or `images:read`:

```bash theme={null}
curl -s -H "X-API-Key: $INSTABOX_API_KEY" https://api.instabox.sh/v1/usage | jq '.cost | {costUSD, creditsUSD, ratedUntil}'
```

`GET /v1/usage` answers for your personal workspace, or with a workspace's API key for that
workspace. The MCP tool `get_usage` returns the same figures.

Invoices, payment methods, and receipts are on the billing portal. Open it with
`instabox workspace billing`, or from **Account** in the dashboard.

## How the cost is computed

Once an hour the server rates the hour that has ended for every workspace and writes one ledger row
per resource. The cost is the sum of this month's rows. Nothing is estimated on request.

* **Runtime**: each paid machine's vCPUs and GiB of memory times the seconds it was `running` (not
  starting, paused, or stopped), at the unit prices on [Pricing](https://instabox.sh/docs/account/pricing.md).
* **Disk**: paid plans include disk across all their machines. Beyond that, allocated `/data` is
  billed by the second while the machine exists (running, paused, or stopped). The system disk,
  snapshots, and a paused machine's saved memory are free. Free plan disk is included.
* **Egress**: what the workspace's machines sent to the internet beyond the plan's monthly egress,
  per GiB. Ingress is free.

A free machine is never billed, so a free month is \$0.00. An hour is priced at the prices in force
then. The month is the calendar month in UTC (`periodStart` to `periodEnd`), and the figures cover
it up to `ratedUntil`, so they lag by up to an hour.

## Usage fields

| Field | Meaning |
| - | - |
| `costUSD` | This month's usage so far: `runtimeUSD` + `storageUSD` (disk) + `egressUSD` |
| `vcpuUSD`, `memUSD` | Compute's two parts: `runtimeUSD` is their sum |
| `vcpuHours`, `memGiBHours`, `storageGiBHours`, `egressGiB` | The quantities they were priced from (on the free plan with nothing billed, the free machine's running time in the same units) |
| `allowanceUSD` | What the month includes: on a paid plan, the subscription fee paid for the month |
| `creditRemainingUSD`, `overageUSD` | The allowance left, and usage beyond it (billed at the month's end, or when it reaches the threshold) |
| `remainingUSD` | What may still be used this month before machines are stopped, when a spend cap applies |
| `creditsUSD`, `creditsExpireAt` | Your credit balance and when the earliest credits expire. Credits pay a plan's fee and usage beyond the allowance, earliest to expire first |
| `creditsUsedUSD` | What credits paid this month |
| `usageCreditsUSD` | The part of `creditsUsedUSD` that paid usage beyond the allowance |
| `chargedUSD` | Usage beyond the allowance already charged this month |

> [accordion] Another workspace's figures
>
> `GET /v1/orgs/{id}` (`instabox workspace get <name>`) returns any workspace you belong to, with the
> same figures as `usage`, plus its `limits` and, while billing is on, its subscription as
> `billing`. Any member, any API key scope.

> [accordion] Per machine, for billing your own customers
>
> `Sandbox.getUsage(id)` (`GET /v1/machines/{id}/usage`) gives a machine's running seconds this
> UTC month. Tag machines with metadata and add them up per customer: see the
> [SDK quick start](https://instabox.sh/docs/integrations/sdk-quickstart.md).

> [card] [Pricing](https://instabox.sh/docs/account/pricing.md)
>
> Plans, included credits, disk and egress, and unit prices.

> [card] [Limits](https://instabox.sh/docs/account/limits.md)
>
> Machines, sizes, storage and rate limits per plan.

To move a plan you paid for personally into a workspace, see [Move a personal plan into a workspace](https://instabox.sh/docs/account/move-personal-plan.md).
