> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Paradev

> Add an instabox machine as a Paradev host, so its agents run there.

[Paradev](https://github.com/paradevai/paradev) is an open source ADE that runs coding agents in
parallel. At the end of this page, an instabox machine is a Paradev host with an agent coding on it.

You need the Paradev desktop app and the [instabox CLI](https://instabox.sh/docs/quickstart.md).

**1. Create a machine and copy its SSH URI**

```bash theme={null}
instabox create --name cloud-dev
```

In the dashboard, open `cloud-dev`, then **Details**. Under **SSH**, select **Create token** and
copy the **URI**.

![The SSH section of a machine with the URI row outlined in red](https://mintcdn.com/instacloud/COlCIjXJ_YLj8tCe/images/paradev/token-uri.png?fit=max\&auto=format\&n=COlCIjXJ_YLj8tCe\&q=85\&s=0bc12d6999bc2b403437bd9e22668262)

**2. Open the host setup in Paradev**

Go to **Settings**, then **Hosts**, and select **Set up a host over SSH**.

![Paradev settings with Hosts and Set up a host over SSH outlined in red](https://mintcdn.com/instacloud/COlCIjXJ_YLj8tCe/images/paradev/hosts-settings.png?fit=max\&auto=format\&n=COlCIjXJ_YLj8tCe\&q=85\&s=c1f4f557c840c5e2569059a7d31f4ab6)

**3. Paste the URI**

Enter the whole URI, in the form `ssh://ibssh_…@ssh.instabox.sh:2222`, and select **Set up**.
Answer the prompts in the terminal that opens.

![Paradev's Set up a host over SSH dialog with an ssh:// URI entered](https://mintcdn.com/instacloud/COlCIjXJ_YLj8tCe/images/paradev/set-up-host.png?fit=max\&auto=format\&n=COlCIjXJ_YLj8tCe\&q=85\&s=2be6dbe783b1c8d1a97f735bbcbbcc19)

**4. Set up GitHub**

```bash theme={null}
instabox ssh cloud-dev
gh auth login
```

Paradev uses this sign-in to list and clone your repositories on the machine.

**5. Start coding**

In Paradev, select the machine as the host and clone your repository. Press **⌘N**, describe the
task, pick an agent, and create the workspace. The agent runs on the machine.

The first time you use an agent, start it once in the `instabox ssh cloud-dev` shell and sign in. The
**Device** filter in **Workspaces** lists the machine under **Other devices**.

> [!WARNING]
>
> Anyone with the URI can open a shell in the machine. To revoke it, run
> `instabox ssh-token --revoke cloud-dev`. A host that is already set up keeps running after you revoke.

> [!NOTE]
>
> Known gap: Paradev cannot wake a paused machine, so new prompts do not reach it. The machine shows as
> offline within about two minutes. Run `instabox wake cloud-dev`, or `instabox resume cloud-dev` if you
> paused it yourself. The host reconnects in seconds, with its terminals intact.

> [accordion] Open the Hosts settings from a script
>
> ```bash theme={null}
> instabox ssh-token cloud-dev --ttl never --json | jq -r '"ssh://\(.token)@\(.host):\(.port)"' | pbcopy
> open 'superset://settings/hosts'
> ```
>
> Copies the URI and opens **Settings → Hosts**. Select **Set up a host over SSH** and paste.

> [accordion] Add the host from the Paradev CLI
>
> ```bash theme={null}
> superset hosts add 'ssh://ibssh_…@ssh.instabox.sh:2222'
> ```
>
> The CLI kept the name `superset` when the product was renamed to Paradev. To get the URI without
> the dashboard, run `instabox ssh-token cloud-dev --ttl never --json`, and build
> `ssh://<token>@<host>:<port>` from its fields.

> [accordion] Remove the host
>
> Run this before you revoke the token, because it connects over SSH:
>
> ```bash theme={null}
> superset hosts remove 'ssh://ibssh_…@ssh.instabox.sh:2222'
> ```
>
> The command stops the host on the machine and revokes the API key that setup created. Then delete
> the offline host in **Settings**, under **Hosts**.

> [card] [SSH](https://instabox.sh/docs/access/ssh.md)
>
> Tokens, expiry, and revoking access.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> What counts as idle and what wakes a machine.
