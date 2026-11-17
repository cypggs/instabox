> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Concepts

> The few words the dashboard, the CLI and the API use.

| Word | What it is |
| - | - |
| **Machine** | A Linux VM of your own in the cloud, with its own kernel, root, and Docker. Files under `/data` (home is `/data/home`) and the software you install stay. |
| **Template** | What a new machine starts from: `agent-browser` (the default), `t3`, `herdr`, `orca`, `android`, or `omarchy` (Arch Linux with the Omarchy desktop). `instabox templates` lists them. |
| **Snapshot** | A saved copy of one of your machines, private to you. New machines can start from it. |
| **Operation** | A change to a machine (create, start, stop, pause, resume, delete). You follow it until it succeeds or fails. |
| **API key** | `ibk_...`: how programs, CI jobs and agents sign in. |
| **Org Secrets** | Encrypted credentials a machine sees only as placeholders. The outbound proxy adds them to HTTPS requests for allowed hosts. |

> [!NOTE]
>
> The free plan creates from agent-browser, android, omarchy, ci only. A snapshot is not limited.

## Machine states

| State | Files | Memory and processes | Use it to |
| - | - | - | - |
| Running | kept | running | work |
| Paused | kept | kept | step away and come back to the same session |
| Stopped | kept | lost | shut down. The next start boots fresh. |
| Deleted | gone | gone | free the slot for a new machine |

A machine pauses by itself after 30 idle minutes. It wakes by itself on the next
command, file read, SSH connection, or visit to one of its public URLs. A machine **you** paused or
stopped stays offline until you resume or start it. See [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md).

> [!NOTE]
>
> The API and CLI say `suspend`, `suspending`, and `suspended` for pause, Pausing, and Paused. They
> say `error` for Failed.

## GPU templates

A GPU machine runs on a host with NVIDIA GPUs, so you can run CUDA and PyTorch work next to your
code. Today the only GPU type is `l4`, the NVIDIA L4 with 24 GiB of GPU memory. A
machine can have 1 to 4 of them.

GPU machines start from the `gpu` template. It comes with the CUDA 13.0 toolkit (`nvcc`), PyTorch 2.8
with CUDA in `/opt/venv`, and `instabox-gpu-smoke`, which checks that the GPU works. The NVIDIA driver
and `nvidia-smi` come from the host.

> [!NOTE]
>
> GPU machines are available on every paid plan, with up to 20 GPUs per workspace. The free plan
> cannot create them.

To create a GPU machine, name the GPU type, or the type and a count. The template is `gpu` unless you
name another.

```bash CLI theme={null}
instabox create --name train --gpu l4
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "train", "gpu": "l4:2"}'
```

In the dashboard, pick the `gpu` template, then the GPU type and count. The size comes from the GPU
count: each GPU brings 11 vCPUs and 36 GiB of memory, so you cannot change the vCPUs or memory
yourself. See [Pricing](https://instabox.sh/docs/account/pricing.md#gpu) for the GPU price.

A machine created from a GPU machine's [snapshot](https://instabox.sh/docs/machines/snapshots.md) keeps its GPU type. It gets
the snapshot's GPU count unless you ask for another (`--snapshot gpu-base --gpu l4:4`).

### Memory inside a GPU machine

A GPU machine runs in a sandbox whose `/proc/meminfo` differs from a Linux kernel's in one way that
matters for large models: file pages you map or read (model weights opened with `mmap`, for example)
count as used memory, and `MemAvailable` equals `MemFree`. So once the files you map are larger than
the machine's memory, `MemFree` and `MemAvailable` read 0 and `Cached` can be larger than `MemTotal`,
even though most of that memory is file cache that can be reclaimed. Tools that size buffers from
"available" memory (psutil's `virtual_memory().available`, and the CPU offload or pinned-memory
budgets of some inference servers) then see none.

`MemTotal` is the machine's memory (36 GiB per GPU), and `AnonPages` is the memory
programs hold (shared memory and tmpfs included). To estimate what is free, treat file cache as
reclaimable:

```bash theme={null}
awk '/^MemTotal|^AnonPages/ {v[$1]=$2} END {printf "%.1f GiB\n", (v["MemTotal:"]-v["AnonPages:"])/1048576}' /proc/meminfo
```

Set such budgets from that estimate, or explicitly. A model whose weights do not fit in the machine's
memory is read from disk again and again while it runs. Give it more GPUs, and so more memory:
`--gpu l4:2` has twice the memory of `--gpu l4`.
