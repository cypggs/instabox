> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# T3 Code

> Pair T3 Code with an instabox machine, so its agents run there.

[T3 Code](https://github.com/pingdotgg/t3code) runs coding agents. Pair the desktop app with a
machine, and its agents run there.

The quickest way is the machine's **Integrations → T3 Code** in the dashboard. It works on any
machine: **Get pairing link** sets up the T3 server if needed and opens T3 Code. The steps below do
the same from the CLI on a `t3` machine, which comes with the server running.

You need the [T3 Code app](https://github.com/pingdotgg/t3code/releases/latest), the
[instabox CLI](https://instabox.sh/docs/quickstart.md) and `jq`.

**1. Create a t3 machine**

```bash theme={null}
instabox create --name t3-dev --template t3
```

**2. Open the pairing link**

```bash theme={null}
code=$(instabox exec t3-dev -- sh -c 't3 pair --base-dir "$HOME/.t3-instabox/$(cat /etc/machine-id)" --label "Instabox client"' | sed -n 's/^Token: //p')
url=$(instabox port publish t3-dev 3774 --json | jq -r .url)
open "t3code://pair?$(jq -rn --arg u "$url/pair#token=$code" '"pairingUrl=" + ($u|@uri)')"
```

This makes a pairing code and [publishes](https://instabox.sh/docs/access/ports.md) port `3774`. On Linux, use `xdg-open`.

![T3 Code asking to add the backend at https://3774-….instabox.tech](https://mintcdn.com/instacloud/3nwJeMr8TWeQQdlO/images/t3-code/confirm-backend.png?fit=max\&auto=format\&n=3nwJeMr8TWeQQdlO\&q=85\&s=3fc21617eebefb04e4b6720c91081a87)

**3. Confirm**

Select **Confirm**. **Settings → Environments** shows the machine as **Connected**.

![T3 Code Environments listing the machine as Connected](https://mintcdn.com/instacloud/3nwJeMr8TWeQQdlO/images/t3-code/environment-connected.png?fit=max\&auto=format\&n=3nwJeMr8TWeQQdlO\&q=85\&s=80ccf873c4d66bf92937fd88b073adc8)

Agents run in `/data/workspace`. To sign one in, run its login once in `instabox ssh t3-dev`.

> [!WARNING]
>
> A pairing code works once, for 5 minutes, for whoever opens the link first. Don't share it. To
> unpair, remove the environment in **Settings → Environments**.

> [!NOTE]
>
> The machine stays awake while T3 Code is connected, and wakes when it reconnects. See
> [what wakes a machine](https://instabox.sh/docs/machines/auto-pause.md#what-wakes-a-machine).

> [!NOTE]
>
> On Linux, T3 Code needs a system keyring (GNOME Keyring or KWallet). Without one, **Confirm** fails
> with "Desktop secure storage is unavailable".

> [accordion] Pair by hand
>
> In **Settings → Environments**, select **Add environment** and paste the pairing URL,
> `https://3774-….instabox.tech/pair#token=<code>`, in **Host**.

> [accordion] List paired clients
>
> ```bash theme={null}
> instabox exec t3-dev -- sh -c 't3 auth session list --base-dir "$HOME/.t3-instabox/$(cat /etc/machine-id)"'
> ```

> [card] [Ports](https://instabox.sh/docs/access/ports.md)
>
> Publish, list, and unpublish ports.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> What counts as idle and what wakes a machine.
