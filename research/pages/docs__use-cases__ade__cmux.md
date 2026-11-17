> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# cmux

> Open a cmux workspace on an instabox machine from one link.

[cmux](https://cmux.com) is a macOS terminal for coding agents. A `cmux://` link opens a workspace on
a machine.

You need cmux (`brew install --cask cmux`), the [instabox CLI](https://instabox.sh/docs/quickstart.md) and `jq`.

**1. Create a machine**

```bash theme={null}
instabox create --name cloud-dev
instabox exec cloud-dev -- sh -c 'mkdir -p ~/.local/state && chmod go-w ~/.local ~/.local/state'
```

cmux refuses a group-writable state directory, which a fresh machine would create. Without the
second line, the first link fails with "could not prepare secure daemon state directory".

**2. Open the link**

```bash theme={null}
read -r token host port < <(instabox ssh-token cloud-dev --ttl never --json | jq -r '"\(.token) \(.host) \(.port)"')
open "cmux://ssh?host=$host&user=$token&port=$port&title=cloud-dev&host-key-policy=accept-new"
```

The user is an [SSH token](https://instabox.sh/docs/access/ssh.md). `host-key-policy=accept-new` trusts the gateway's key.

![cmux asking Open SSH Workspace in cmux?, with the token hidden](https://mintcdn.com/instacloud/3nwJeMr8TWeQQdlO/images/cmux/confirm-workspace.png?fit=max\&auto=format\&n=3nwJeMr8TWeQQdlO\&q=85\&s=967ff6f8c26e42e83d73ab180d788080)

**3. Confirm**

Tick **I trust this SSH target** and select **Open**. The workspace shows as **Connected**.

![A connected cmux workspace running whoami, hostname and uname -r on the machine](https://mintcdn.com/instacloud/3nwJeMr8TWeQQdlO/images/cmux/workspace-connected.png?fit=max\&auto=format\&n=3nwJeMr8TWeQQdlO\&q=85\&s=04478854728739593838382d64915ed3)

> [!WARNING]
>
> The link holds the token: anyone with it gets a shell. Prefer `cmux://` over
> `https://cmux.com/deeplink/ssh?…`, which sends the token to cmux.com.

> [!NOTE]
>
> Opening the link wakes an idle-paused machine. See
> [what wakes a machine](https://instabox.sh/docs/machines/auto-pause.md#what-wakes-a-machine).

> [accordion] Connect from a cmux terminal
>
> ```bash theme={null}
> cmux ssh --port 2222 --name cloud-dev 'ibssh_…@ssh.instabox.sh'
> ```
>
> Works only inside cmux.

> [accordion] Error: SSH authentication requires a terminal
>
> The link lacks `host-key-policy=accept-new`.

> [card] [SSH](https://instabox.sh/docs/access/ssh.md)
>
> Tokens, expiry, and revoking access.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> What counts as idle and what wakes a machine.
