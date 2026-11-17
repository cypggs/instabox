> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Limits

> What each plan allows, the machine specs, and the limits you can run into inside a machine.

This page lists what each plan allows, the machine specs, and the limits you can run into inside a
machine. Your own limits and counts are in `instabox usage`, on **Account** in the dashboard, and in
`GET /v1/usage`.

| | Free | Pro | Builder | Team | Enterprise |
| - | - | - | - | - | - |
| Machines | 1 | 500 | 1,000 | unlimited | unlimited |
| Running at once | 1 | 100 | 200 | 500 | unlimited |
| Largest size | 4 vCPU / 8 GiB | 8 vCPU / 16 GiB | 16 vCPU / 32 GiB | 32 vCPU / 64 GiB | 32 vCPU / 64 GiB |
| Disk per machine | 20 GiB (fixed) | up to 200 GiB | up to 500 GiB | up to 1 TiB | up to 1 TiB |
| Disk included | - | 200 GiB | 1 TiB | 5 TiB | 5 TiB |
| Stored data cap | 200 GiB | none | none | none | none |
| Snapshots | 5 | unlimited | unlimited | unlimited | unlimited |
| Egress / month | 100 GiB | 1 TiB | 5 TiB | 50 TiB | 50 TiB |
| Starts / minute | 3 | 30 | 50 | 100 | 100 |
| Members | 3 | unlimited | unlimited | unlimited | unlimited |

| Also limited | Limit |
| - | - |
| Machine sizes | `GET /v1/orgs` lists each workspace's as `allowedSpecs`. The default is 4 vCPU / 8 GiB |
| Idle pause | Free: 30 minutes, fixed. Paid plans: 5 to 30 minutes, or off |
| Max run | Free: each run starts at 120 minutes and grows while the machine is in use, up to 480 minutes ([Runtime on the free plan](https://instabox.sh/docs/machines/auto-pause.md#runtime-on-the-free-plan)). Paid plans: unlimited by default, or your choice |
| Starts | Create, start, resume, and wake count. More is a retryable `429 rate_limited` |
| API keys | 20 active |
| `exec` | 32 requests at once across your machines, shared by commands, file reads, desktop, port-forward, Secret refresh, and OOM claim calls. More get a retryable `429 quota_exceeded`. 10 minutes per command, 60 seconds by default |
| Labels | 32 per machine |
| Public ports | 16 per machine |

* Over the machine or storage limit, new machines and snapshots are refused until you delete some.
  **Account** warns from 80% of your storage.
* A machine started from a snapshot has at least the storage of the machine it was saved from.

## Machine specs

Every template but `ci` (Ubuntu 24.04) and `omarchy` (Arch Linux) is Debian 13 (trixie), amd64, with systemd, on Linux 6.18.60
(`uname -r`: `6.18.60-instabox`; `6.18.60-instabox-android` on `android`, `6.18.60-instabox-drm` on `omarchy`) in a Firecracker microVM. A machine is 4 vCPU / 8 GiB by default (`free -m` shows about 7.8 GiB on
that size) with a 20 GiB persistent ext4 disk at `/data` (home is `/data/home`).
Every template but `omarchy` has git, curl, jq, Python with venv and pip, and passwordless `sudo` with apt;
`omarchy` has git, curl, jq, Arch's Python, and passwordless `sudo` with pacman.

| Template | Root disk | Also preinstalled |
| - | - | - |
| `agent-browser` (the default) | 16 GiB | gh, Node.js 22 with npm, pnpm and Yarn, Bun, uv, Go, Rust, OpenJDK 21, zip, gcc and make, Docker with Compose, Claude Code, Codex, OpenCode, and the desktop (VNC, Xfce, Chromium) |
| `universal` | 16 GiB | `agent-browser`'s tools. The desktop is installed on request (about half a minute) |
| `t3` | 16 GiB | `universal`'s, plus the T3 Code server (port 3774); the desktop is installed on request |
| `herdr` | 8 GiB | gh, Node.js 22 and npm, Claude Code, Codex, OpenCode, the desktop, and herdr (no Docker, uv, Go, Rust or C toolchain) |
| `orca` | 8 GiB | gh, Node.js 22 and npm, Claude Code, Codex, OpenCode, the desktop, and the Orca runtime (no Docker, uv, Go, Rust or C toolchain) |
| `minimal` | 8 GiB | A small box with the main dev tools; install what you want. Pro plan. wget, gcc and make (`build-essential`), Node.js 22 and npm, vim and nano (no desktop, browser, Docker or coding agents) |
| `android` | 24 GiB | `agent-browser`'s tools and desktop, plus Android 14 with adb ([Android apps](https://instabox.sh/docs/use-cases/android.md)) |
| `omarchy` | 24 GiB | Arch Linux with [Omarchy](https://omarchy.org): its Hyprland desktop (Wayland, on the Computer tab), themes, and tools, logged in from boot, on its own kernel. Every plan, free included; no computer-use agents |
| `ci` | 80 GiB | For `runs-on` jobs ([CI](https://instabox.sh/docs/use-cases/ci.md)): GitHub's Actions runner and the tools of GitHub's `ubuntu-24.04` image. No desktop or coding agents |

The free plan creates from agent-browser, android, omarchy, ci only. To check a machine, run `uname -r`,
`cat /etc/os-release`, `nproc`, `free -m`, and `df -h / /data`.

The kernel of these templates has no loadable kernel modules. It is built with `CONFIG_MODULES`
off, so everything it supports is built in. `insmod` and `modprobe` fail with `Function not implemented`
(ENOSYS), and there is no `linux-headers` package for them, so DKMS and other out-of-tree modules
cannot be built or loaded. If you need a kernel feature that is missing, tell us with
`instabox feedback`.

### Preinstalled tool versions

| Tool | Version | Templates |
| - | - | - |
| Claude Code | 2.1.288 | every template except `ci` and `minimal` |
| Codex | 0.160.0 | every template except `ci` and `minimal` |
| OpenCode | 1.19.5 | every template except `ci` and `minimal` |
| Node.js | 22.23.3 | every template except `ci` |
| gh | 2.102.0 | every template except `ci` and `minimal` |
| Go | 1.27.1 | `agent-browser`, `universal`, `t3`, `android` |
| Rust | 1.99.0 | `agent-browser`, `universal`, `t3`, `android` |
| uv | 0.12.23 | `agent-browser`, `universal`, `t3`, `android` |
| Bun | 1.4.4 | `agent-browser`, `universal`, `t3`, `android` |
| pnpm | 12.15.0 | `agent-browser`, `universal`, `t3`, `android` |
| Yarn | 1.22.22 | `agent-browser`, `universal`, `t3`, `android` |
| Docker | 29.8.2 | `agent-browser`, `universal`, `t3`, `android` |
| Docker Compose | 5.6.0 | `agent-browser`, `universal`, `t3`, `android` |
| T3 Code | 0.1.2 | `t3` |
| Gradle | 8.7 | `android` |

The `ci` template's versions are in [CI](https://instabox.sh/docs/use-cases/ci.md). To check a tool in a machine, run its own
version command, for example `claude --version`. `dpkg -l` lists the apt packages.

## Inside a machine

> [accordion] Missing tools: apt-get update first
>
> GNU `time`, `rg` (ripgrep) and `htpasswd` are not preinstalled on every template. The image ships
> without apt lists, so `apt-get install` alone fails until `apt-get update` has run:
>
> ```bash theme={null}
> sudo apt-get update
> sudo apt-get install -y time ripgrep apache2-utils
> ```
>
> To run another image, such as `node:16-buster`, use Docker. On `agent-browser`, `universal`, and
> `t3`, the machine's user is in group `docker`, so `docker run --rm node:16-buster node -v` works
> without `sudo`.

> [accordion] npm, Rust and the JDK without sudo
>
> On every template but `ci` and `omarchy`, npm's global prefix is `/opt/npm`, owned by the machine's user:
> `npm install -g <package>` (and `claude update`) needs no `sudo`. Do not use `sudo npm install -g`:
> it leaves root-owned files that make the next install fail.
>
> On `agent-browser`, `universal`, and `t3`, `/data` belongs to the machine's user (build in
> `/data/workspace` or a new `/data/<dir>`), and so does the Rust toolchain in `/opt/rust`.
> `rustup toolchain install`, `rustup default`, and `rustup target add` need no `sudo`. A rustup of
> your own in `~/.cargo` takes over `cargo` and `rustc`. The JDK is OpenJDK 21 (Debian 13 has no
> 17\), with `JAVA_HOME=/usr/lib/jvm/java-21-openjdk-amd64` set for `exec` and non-interactive SSH.
>
> A machine on an older template version (`stat -c %U /opt/rust` prints `root`) needs
> `sudo chown -R "$USER": /opt/rust && sudo chown "$USER": /data` and
> `sudo apt-get update && sudo apt-get install -y zip openjdk-21-jdk-headless` for the same setup.

> [accordion] exec output, timeouts and background jobs
>
> Without `--stream`, the output arrives when the command exits, each stream cut at 1 MiB.
> `exec --stream` prints the output as it is written, with no cap. A command runs at most 10
> minutes (`--timeout`, 60 seconds by default). A longer job runs with `exec --background`, which
> has no exec timeout, though idle pause and max-run still apply. `jobs ls|logs|wait|kill` follow
> the job.
>
> `exec` returns when the command exits, even if it started a server in the background. Redirect
> both streams (`nohup ./server >server.log 2>&1 &`): a background process still writing to exec's
> output gets a broken pipe and may die.

> [accordion] /tmp is in memory
>
> `/tmp` is a RAM-backed tmpfs capped at a quarter of the machine's memory. Past the cap a write
> fails with `No space left on device` even when `df -h /data` shows free space. Login shells (SSH,
> the dashboard terminal, `instabox exec`) set `TMPDIR=/data/tmp`, on the persistent disk, so most
> tools put temporary files there. Files in `/data/tmp` unused for 10 days are deleted. `/dev/shm`
> is tmpfs too.
>
> On an older machine (`echo $TMPDIR` prints nothing), `/tmp` is about half the memory and
> temporary files go to memory. Run `export TMPDIR="$HOME/tmp" && mkdir -p "$TMPDIR"`, and add the
> line to `~/.bashrc`.

> [accordion] File watches and open files (ENOSPC, EMFILE)
>
> | Limit | Typical default | Symptom when hit |
> | - | - | - |
> | `fs.inotify.max_user_watches` | about 32k–64k | watchers (chokidar, Vite, webpack) log `ENOSPC` and miss edits |
> | soft `ulimit -n` | 1024 (hard often 524288) | `EMFILE` from Jest or workers opening many files |
>
> ```bash theme={null}
> sudo sysctl -w fs.inotify.max_user_watches=524288   # machine-wide, until the next restart
> sudo sysctl -w fs.inotify.max_user_instances=512
> ulimit -n 65536 && npm test                         # same shell as the workload
> ```
>
> Each `instabox exec` is a new shell that starts at 1024 again and does not read `~/.bashrc`. For
> every session, put the `ulimit -n` line in `/etc/profile.d/nofile.sh` (or use
> `/etc/security/limits.d/`), and the sysctls in `/etc/sysctl.d/`.

> [accordion] Out of memory and swap
>
> When processes use more memory than the machine and its swap have, the kernel OOM killer kills
> one. The machine stays `running`. `instabox get` shows `OOM kills`, and an exec whose command was
> killed returns `oomKilled: true`. Its exit code is usually `137`, but check `oomKilled`, not
> `signal`. An absent or 0 `OOM kills` does not prove there was none. Confirm inside the machine
> with `dmesg -T | grep -i oom`.
>
> Every machine has a swapfile, `/data/swapfile`, the size of its memory up to 4 GiB (and at most a
> quarter of `/data`), made at first boot when `/data` has room. `free -h` and `swapon --show` show
> it. To change its size:
>
> ```bash theme={null}
> sudo swapoff /data/swapfile && sudo rm /data/swapfile
> sudo fallocate -l 8G /data/swapfile && sudo chmod 600 /data/swapfile
> sudo mkswap /data/swapfile && sudo swapon /data/swapfile
> ```
>
> `sudo touch /etc/instabox/no-swap` keeps swap off. A snapshot or fork carries the swapfile, which
> can hold paged-out secrets. Before sharing a snapshot, zero the swapfile
> (`sudo swapoff /data/swapfile && sudo shred -n0 -z /data/swapfile && sudo mkswap /data/swapfile && sudo swapon /data/swapfile`)
> or save `--root-only`. For a machine that keeps running out, raise its size or its swap.

> [accordion] Outbound bandwidth
>
> What a machine sends is capped at 50 Mbit/s (about 6 MB/s), after a burst. The first 512 MiB
> goes out at full speed, and up to about 1 GiB when the machine has been quiet for about
> 90 seconds or more. After the burst, the machine is held to 50 Mbit/s. The burst refills at
> 50 Mbit/s, so it is full again after about 90 quiet seconds.
>
> The cap covers uploads, what the machine sends over SSH and port forwards, and responses from its
> public URLs. Past the burst, each further 100 MB takes about 16 seconds, so keep large artifacts in
> the machine (on `/data`, or in a snapshot). What a machine receives is not limited. Report a slow
> download with `instabox feedback`. The cap is per machine, separate from your plan's monthly
> egress. A `ci` machine that runs a GitHub Actions job for a paid workspace is capped at
> 1000 Mbit/s instead.

## Disks, snapshots, and templates

> [accordion] Grow or shrink /data
>
> A machine has one disk you size, `/data`. At create, the disk comes with the machine's size, and
> any other disk size is `400 spec_not_allowed`. To change it later:
>
> ```bash theme={null}
> instabox resize dev --disk 50             # --restart when it must stop first
> ```
>
> You can also send `PATCH /v1/machines/{id}` with `{"persistentDiskGiB": 50}`, or use
> **Resize disk** in the machine's **Settings**.
>
> * Paid plans only. A free machine's disk is fixed at 20 GiB. The disk grows to
>   200 GiB on Pro, 500 GiB on Builder, 1024 GiB on Team and Enterprise, within your stored-data limit (`429 quota_exceeded`). The API takes 1–1024 GiB.
> * **Grow**: a running machine's disk grows at once, with no restart. If it cannot grow online, the
>   answer is `409 grow_needs_stop`. Stop the machine and send the size again.
> * **Shrink**: only a stopped machine. Your data plus 10% (at least 1 GiB) must fit, or
>   `409 disk_too_small` names the smallest size that does.
> * The REST field is `persistentDiskGiB`. `diskGiB` (the TypeScript SDK's name) is
>   `400 unknown_field` in a REST body.
> * The system disk is sized by the template and is not counted. Disk beyond the plan's included
>   disk is billed: [Pricing](https://instabox.sh/docs/account/pricing.md).

> [accordion] A machine from a snapshot gets the plan's default size
>
> `instabox create --snapshot` (or `Sandbox.create(snapshotId)`) without `--vcpus` or `--mem-mib`
> uses your plan's default size, not the source machine's. `instabox fork` keeps the source size. In
> snapshot JSON, `rootSizeBytes` and `dataSizeBytes` are the logical disk sizes the next create
> provisions, not the bytes stored.

> [accordion] Template versions, creatable and no\_capacity
>
> `instabox templates` (`GET /v1/templates`) lists what you can create from. A version no host holds
> any more is `creatable: false`, and a create pinned to it is `400 bad_request` (not retryable). An
> older pin such as `agent-browser@2026-10-08.2` is held by fewer hosts, so it can fail with
> `no_capacity` while `latest` creates at once. Prefer an unpinned template unless you need the pin.

> [card] [Pricing](https://instabox.sh/docs/account/pricing.md)
>
> Plans and unit prices.

> [card] [Errors](https://instabox.sh/docs/reference/errors.md)
>
> `quota_exceeded`, `rate_limited`, `spec_not_allowed` and the rest.
