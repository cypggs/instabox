> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Custom images

> Turn a Docker or OCI image into a snapshot, then start machines from it.

An import turns a Docker or OCI image from a registry into a snapshot, so that you can start
machines that have the image's programs and files. Use an import when your environment already
exists as an image, such as the one your CI runs in or one your team publishes, and you do not want
to set it up again by hand.

```bash theme={null}
instabox snapshot import node:16-buster --name node16
```

The import takes a few minutes. The result is a snapshot like any other, so everything in
[Snapshots](https://instabox.sh/docs/machines/snapshots.md) applies to it. Import works on every plan.

## Start a machine from the image

When the import is done, create a machine from the snapshot that it saved. The snapshot stays, so
you can start more machines from it later without importing the image again.

```bash theme={null}
instabox create --name dev --snapshot node16
```

The machine has the image's programs, which you can check by running one of them. For this image,
`node -v` prints the image's version, `v16.20.2`.

```bash theme={null}
instabox exec dev -- node -v
```

## Import a private image

When the image is in a private registry, the import has to sign in to pull it. Pass the registry
user with `--registry-username`, and give the password either in the `INSTABOX_REGISTRY_PASSWORD`
variable or on standard input with `--registry-password-stdin`. The password is used for the pull
and is not kept in the snapshot.

```bash theme={null}
instabox snapshot import ghcr.io/acme/tool:1.4 --name tool \
  --registry-username bot --registry-password-stdin < token.txt
```

## Import an image you built locally

An import pulls the image from a registry, so it cannot read an image that exists only in Docker on
your computer. Push the image that you built to a registry first, and then import it by its registry
name.

```bash theme={null}
docker push ghcr.io/you/img:tag
```

## Read the output of the image's command

The image's `ENTRYPOINT` and `CMD` run as the `instabox-image.service` systemd unit, so their output
goes to the machine's journal and not to your terminal. When you want to see whether the image's
command started or why it failed, read the unit's output with `journalctl` inside the machine.

```bash theme={null}
journalctl -u instabox-image.service
```

When you want the image's files but not its command, import the image with `--no-entrypoint`, which
leaves the unit out.

## How an import works

`snapshot import` starts a throwaway machine and pulls the image there with Docker. It lays the
image's files over the machine's own, and then restarts the machine to check that it still boots and
answers. Last, it saves the machine as a snapshot and deletes the throwaway machine. The throwaway
machine is an `agent-browser` one, and it counts while it runs.

## What a machine from an image has

A machine from an image is still a machine, with the image laid on top. Where the two overlap, the
image's files win, and the parts that make the machine work stay.

* **The image's files win**: its programs, libraries, `/usr/local`, `/opt`, `/root`, `/home`, `/app`,
  and its users.
* **Its `ENV`** reaches every session (exec, SSH, login shells), and the directories its `PATH` adds
  come first.
* **Its `ENTRYPOINT` and `CMD`** run as the `instabox-image.service` systemd unit, in its `WORKDIR`, as
  its `USER` (root when unset), restarted on failure.
* **What makes it a machine stays**: systemd, sshd, sudo, the machine's user (UID 1000, so the image's
  user with that ID, such as `node`, is that user), `/data`, and the shared libraries the machine's own
  programs load.
* `/etc/instabox/image.json` records the image's reference, digest, and config.

## Limits

* Only `linux/amd64` images can be imported.
* Ports that the image declares with `EXPOSE` are not published. Publish them with
  `instabox port publish`, as for any machine. See [Public and private ports](https://instabox.sh/docs/access/ports.md).
* Snapshot limits apply (the free plan keeps 5).

> [accordion] Init, apt, and the C library
>
> A machine runs systemd as its init. The image's own init (tini, s6) is present but not run, only its
> `ENTRYPOINT` and `CMD`. `apt` manages the machine's own system. The image's packages are there as
> files, but `apt` cannot see or remove them. Debian and Ubuntu images, old ones too (Debian 10 buster),
> run on the machine's newer C library. Alpine (musl) images work best-effort.

## Reference

### `snapshot import` flags

Most imports need only the image and a name. These flags cover private registries and help when an
import fails or takes too long.

| Flag | What it does |
| - | - |
| `--registry-username U` | A private image, with `INSTABOX_REGISTRY_PASSWORD` set or `--registry-password-stdin`. The password is not kept in the snapshot. |
| `--keep-builder` | Keep the throwaway machine, also after a failure, to look around with `instabox ssh`. |
| `--no-verify` | Skip the restart check. |
| `--timeout` | Bound the pull and the overlay (default `30m`). |
| `--json` | Print the result as one object. |
