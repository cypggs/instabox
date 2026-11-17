> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# SSH

> Open a shell, run a command or forward a port into a machine.

SSH is how you work inside a machine from your own terminal. You can open a shell to work in the
machine by hand, run one command there from a script, or forward a machine port so that a server in
the machine answers on your own computer.

How you connect depends on where you connect from. On a computer that has the `instabox` CLI, you
need no token. On a terminal without the CLI, you create an SSH token first and paste the line it
gives you.

| You connect from | Do this |
| - | - |
| A computer with the CLI | Use `instabox ssh`. The CLI handles the certificate and `known_hosts`. |
| A terminal without the CLI | [Create an SSH token](#connect-from-any-terminal-without-the-cli), then paste its line. |
| An editor or ADE | Follow [ADEs](https://instabox.sh/docs/use-cases/ade). |

The most common way in is `instabox ssh` with the machine's name, which opens a shell. The command
gets the certificate, writes the `known_hosts` line, and starts ssh in one step, so there is nothing
to set up before your first connection.

```bash theme={null}
instabox ssh dev
```

## Run one command

When you want the output of one command and not a shell, put the command after `--`. The command
runs in the machine and `instabox ssh` returns when it ends, which makes this the form to use in a
script.

```bash theme={null}
instabox ssh dev -- uname -a
```

## Connect from any terminal without the CLI

An SSH token is one line that you can paste into any terminal, so the computer you connect from
needs neither the CLI nor an SSH key. You can create the token in the dashboard, or with the CLI or
the API on another computer.

```bash CLI theme={null}
instabox ssh-token dev
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../ssh-tokens \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

In the dashboard, open **SSH tokens** in a machine's **Connect** section and select **Create token**.
The line stays in that section, so you can come back and copy it later.

Whichever way you create the token, you get a line like this one. The API returns the line in the
`command` field of its response.

```text theme={null}
ssh ibssh_…@<host>
```

The first time you connect, ssh asks you to confirm the gateway's host key.

A token works for 7 days by default. You can pass `--ttl` to set another time, up to 30 days, when
you want access to end sooner or last longer. For a token that does not expire, pass `--ttl never`.

> [!WARNING]
>
> Anyone who has the line can open a shell in the machine, so treat it like a password. To cut off
> access, revoke the machine's tokens.

## Revoke SSH tokens

Revoke a machine's tokens when a line has reached someone who should not have it, or when the
access is no longer needed. Running `ssh-token` with `--revoke` revokes every token of that machine
at once.

```bash theme={null}
instabox ssh-token --revoke dev
```

## Forward a port to your computer

A forward makes a port in the machine reachable on your own computer, so you can open a server that
runs in the machine with the browser and tools you already have. This example serves the machine's
port 3000 at `http://127.0.0.1:8080`.

```bash theme={null}
instabox port-forward dev 8080:3000
```

The forward runs until you press Ctrl-C, and `forward` is an alias for `port-forward`. When you
give only one port, the local port defaults to the remote port, so this command serves the
machine's port 3000 on your port 3000.

```bash theme={null}
instabox port-forward dev 3000
```

You can forward several ports at once by listing them in one command, which saves you a terminal
for each port.

```bash theme={null}
instabox port-forward dev 8080:3000 5432
```

If you prefer named flags to the `8080:3000` form, `--remote` and `--local` spell out the same
forward.

```bash theme={null}
instabox port-forward dev --remote 3000 --local 8080
```

A forward reaches any TCP service in the machine, not only a web server. Only you can use it,
because it listens on `127.0.0.1` unless `--address` or `--bind` says otherwise. When you want a URL
that other people can open, publish the port instead, as described in
[Public and private ports](https://instabox.sh/docs/access/ports.md).

## Keep a session alive with mosh

[mosh](https://mosh.org) keeps a session alive through sleep, network changes, and packet loss, so
it suits a laptop that moves between networks or a connection that drops. mosh has to be installed
on your computer before you connect. On macOS, you can install it with Homebrew.

```bash theme={null}
brew install mosh
```

Then connect with `instabox mosh`, which works like `instabox ssh` and opens the same kind of
session. On a computer without the CLI, use the mosh command that **Create token** shows.

```bash theme={null}
instabox mosh dev
```

> [accordion] Machines from a snapshot made before mosh was added
>
> A machine that comes from a snapshot made before mosh was added needs one more step. Install mosh
> in the machine once, and `instabox mosh` works from then on.
>
> ```bash theme={null}
> sudo apt install mosh
> ```

## Use your own SSH client

You can also connect with plain ssh, scp, or an editor, when you want a tool other than the CLI to
reach the machine. Print a `Host` block for the machine and add it to `~/.ssh/config`, so that those
tools know how to connect.

```bash theme={null}
instabox ssh-config dev
```

The certificate that the block points to is short-lived, and your own client does not renew it.
When it has expired, run `ssh-cert` to refresh only the certificate, without connecting.

```bash theme={null}
instabox ssh-cert dev
```

## Paste images

You can paste an image from your clipboard into a session, which is how you show a screenshot to a
coding agent that runs in the machine. In `instabox ssh` on macOS, or on Linux with `xclip` or
`wl-paste`, press Ctrl+V while a PNG image is on your clipboard. The CLI uploads the image to the
machine's `/tmp` and pastes its path, which Claude Code then attaches as an image.

> [accordion] Ctrl+V details and turning it off
>
> You can press Ctrl+C to cancel a slow upload, and a paste that fails rings the terminal bell. When
> the clipboard holds no image, or holds a JPEG, Ctrl+V is sent to the machine as usual.
>
> Some programs use Ctrl+V themselves, such as vim (visual block) and readline (quoted insert). To
> let the key reach them while an image is on the clipboard, pass `--no-paste` or set
> `INSTABOX_NO_PASTE=1`.

## Copy large files with the CLI, not scp

When you upload a large file, use `instabox push` or `instabox cp`, because they stay fast when
you are far from the machine. Both commands use the HTTP API, which sends a file of 64 MiB or more
as 4 parallel PUTs. scp or rsync over `ssh.instabox.sh` keeps about 2 MiB in flight per connection,
so it slows down as the distance grows.

> [accordion] Measured throughput
>
> At a 5 ms round trip, both paths do about 104 MB/s, so close to the machine the choice makes no
> difference. At 305 ms, the API does 11-13 MB/s per stream and SSH does 3-6 MB/s. Together, 4
> parallel API PUTs did 39 MB/s.

## How it works

You do not need this section to connect, but it shows what the CLI sets up for you. Every
connection goes to the instabox gateway, which terminates SSH. A connection is an ssh login to the
gateway, with the machine ID as the user name and a short-lived user certificate.

```text theme={null}
ssh -p <port> <machine-id>@<host>
```

Your client trusts the gateway through its host CA key, which is what the `known_hosts` line holds.

```text theme={null}
@cert-authority [<host>]:<port> <host CA key>
```

The gateway relays mosh's UDP too, so only the gateway has to be reachable from your computer. A
port forward is `ssh -N -L` with the instabox certificate.

## Limits

* **Supported:** sessions (pty, shell, exec, subsystem, env, and signals) and local port forwarding.
* **Not supported:** agent forwarding, X11, and remote forwarding.
* Connections close when the machine stops, pauses, or is deleted.

> [card] [Public and private ports](https://instabox.sh/docs/access/ports.md)
>
> An HTTPS URL for a web server in the machine.

> [card] [Desktop](https://instabox.sh/docs/access/desktop.md)
>
> Watch and drive the machine's desktop.
