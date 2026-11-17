> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Pricing

> Plans, what each includes, and the unit prices of usage.

You pay for a machine's vCPUs and memory only while it runs, by the second. A paused or stopped
machine costs only its disk (a macOS machine's Mac lease is billed for its 24-hour minimum either way). Paid plans include monthly credits that usage comes out of first. All
prices are in USD and match the [InstaBox pricing page](https://instabox.sh/pricing).

> [card] Free · \$0
>
> * 1 machine, 4 vCPU / 8 GiB
> * 20 GiB disk (fixed)
> * 5 snapshots
> * 100 GiB egress / month

> [card] Pro · \$20/month + usage
>
> * \$20 / month free credits
> * Max 100 concurrent machines
> * Up to 8 vCPU / 16 GiB
> * Disk up to 200 GiB
> * 200 GiB disk included
> * Unlimited snapshots
> * 1 TiB egress / month
> * macOS sandboxes
> * Email support

> [card] Builder · \$100/month + usage
>
> * \$100 / month free credits
> * Max 200 concurrent machines
> * Up to 16 vCPU / 32 GiB
> * Disk up to 500 GiB
> * 1 TiB disk included
> * Unlimited snapshots
> * 5 TiB egress / month
> * macOS sandboxes
> * Email support

> [card] Team · \$500/month + usage
>
> * \$500 / month free credits
> * 500 concurrent machines
> * Unlimited machines
> * Up to 32 vCPU / 64 GiB
> * Disk up to 1 TiB
> * 5 TiB disk included
> * Unlimited snapshots
> * 50 TiB egress / month
> * macOS sandboxes
> * Priority email support & SLAs

> [card] Enterprise · Custom
>
> * 99.9% SLA
> * SSO / SAML and SCIM
> * Audit logs
> * Dedicated pool or BYOC
> * macOS sandboxes
> * BAA available on request
> * Dedicated support

## Unit prices

Paid plans pay these for usage beyond their monthly credits. Runtime is metered by the second, only while a machine is running. Snapshots are free.

| Resource | Price | Unit |
| - | - | - |
| vCPU | \$0.004 | per vCPU-hour |
| NVIDIA L4 | \$0.64 | per GPU-hour |
| NVIDIA H100 PCIe | \$3.49 | per GPU-hour |
| Memory | \$0.00225 | per GiB-hour |
| Storage | \$0.00011 | per GiB-hour |
| Egress | \$0.02 | per GiB |

Per GPU, billed by the second. CPU and memory are billed separately.

**Storage.** Free: disk included at no charge. Paid plans include disk across all their machines (200 GiB on Pro, 1 TiB on Builder, 5 TiB on Team); beyond that, allocated /data is billed at $0.08 per GiB-month ($0.00011 per GiB-hour).

## Questions

> [accordion] What do I pay for?
>
> While a machine is running, you pay for the vCPUs and memory you gave it. Pause it, and you just pay for its disk. A macOS machine instead pays its Mac's price per hour, vCPUs and memory included, for at least 24 hours whether it runs or not. Free: disk included at no charge. Paid plans include disk across all their machines (200 GiB on Pro, 1 TiB on Builder, 5 TiB on Team); beyond that, allocated /data is billed at $0.08 per GiB-month ($0.00011 per GiB-hour). Snapshots are free.

> [accordion] How do credits work?
>
> Usage comes out of your included credits first, then you pay as you go. Credits reset every month.

> [accordion] Do you charge for network traffic?
>
> Ingress is always free. Each plan includes egress every month (100 GiB on Free, 1 TiB on Pro, 5 TiB on Builder, 50 TiB on Team), then \$0.02 per GiB.

> [accordion] What happens to the five free sandboxes early accounts got?
>
> Accounts registered before 2026-10-20 and activated before 2026-10-25 07:45 UTC got five free sandboxes until 2026-12-01 00:00 UTC. After that only one free sandbox runs at a time. The others are paused, not deleted, and the one that stays running is the most recently active. To run another, pause the running one and start it, delete the ones you do not need, or upgrade. We email affected accounts on 2026-11-26.

## GPU

A GPU machine pays for its vCPUs and memory like any machine, and for its GPUs on top. GPUs are
billed per GPU, by the second, only while the machine runs. A paused or stopped GPU machine holds no
GPU on its host, so it pays only for its disk and does not count toward your workspace's GPU limit.
Starting or resuming it takes its GPUs back, so it is refused when your running GPU machines already
use the whole limit. GPU usage comes out of your monthly credits like other usage.

| GPU | Price | Size per GPU | All in, per GPU |
| - | - | - | - |
| NVIDIA L4 (`l4`) | \$0.64 per GPU-hour | 11 vCPU / 36 GiB | \$0.765 per hour |
| NVIDIA H100 PCIe (`h100`) | \$3.49 per GPU-hour | 44 vCPU / 224 GiB | \$4.17 per hour |

For example, a machine with two NVIDIA L4 GPUs costs \$1.53 an hour while it runs. GPU machines are available on
paid plans only. See [GPU templates](https://instabox.sh/docs/concepts.md#gpu-templates).

## macOS machines

A macOS machine leases a whole Mac to your workspace, at one price per Mac per hour. The price
covers the machine's vCPUs and memory: they are not billed on top. Its disk counts as storage, like
every machine's. The lease lasts at least 24 hours, whether the machine runs or not, and is then
billed per second. macOS machines are for the Pro plan and higher. See [macOS machines](https://instabox.sh/docs/machines/macos.md).

| Model | `--model` | Per hour | 24-hour minimum |
| - | - | - | - |
| M2 | `m2` | \$0.45 | \$10.80 |
| M2 Pro | `m2pro` | \$0.85 | \$20.40 |
| M4 | `m4` | \$0.75 | \$18.00 |
| M4 Pro | `m4pro` | \$1.25 | \$30.00 |

> [card] [Billing and usage](https://instabox.sh/docs/account/billing.md)
>
> Read this month's cost and credits, and how they are computed.

> [card] [Limits](https://instabox.sh/docs/account/limits.md)
>
> What each plan allows, and the machine specs.
