> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Orca

> Run each Orca workspace on its own instabox machine.

At the end of this page, a new Orca workspace runs on its own machine, with your repo cloned and your
branch checked out. The Instabox plugin creates the machine with the workspace, pauses it when the
workspace sleeps, and deletes it with the project.

You need:

* Orca 1.4.245 or later.
* `node` on your `PATH`.
* A repository with an `origin` remote. The start commit you pick must be pushed, because the machine
  fetches it from the Git host.

**1. Install the CLI**

```bash theme={null}
npx instabox@latest
```

The command signs you in and puts `instabox` on your `PATH`. The plugin runs this CLI on your
computer. Restart Orca so that it finds the CLI.

**2. Turn on Cloud VM**

In Orca, open **Settings**, then **Experimental**, and turn on **Cloud VM**. Without this setting,
**Per-Workspace Environment** does not show in **Run on**.

**3. Install the Instabox plugin**

Open the [Instabox plugin page](https://onorca.dev/plugins/insforge.instabox?catalog=community).
It opens the plugin in **Settings**, under **Plugins**. If plugins are off, turn them on first.
Select **Install**, then turn the plugin on with its switch. The plugin shows **On** when it is
turned on.

![The Instabox plugin in Orca's Plugins settings, with its switch turned on and the status On](https://mintcdn.com/instacloud/rpjO_4qZKxC6_teT/images/orca/plugin-enabled.png?fit=max\&auto=format\&n=rpjO_4qZKxC6_teT\&q=85\&s=3cc7d9e18fd11d6dc40c290e7a095ca0)

**4. Save a GitHub token for a private repository**

Skip this step for a public repository.

```bash theme={null}
gh auth token | instabox secrets set GH_TOKEN --host github.com --host api.github.com
```

Each new machine selects the `GH_TOKEN` Secret. The outbound proxy adds the real token to requests
to GitHub only.

**5. Sign Claude Code in on your machines**

```bash theme={null}
claude setup-token
instabox secrets set CLAUDE_CODE_OAUTH_TOKEN --ai anthropic
```

The first command prints a token. Paste it at the second command's hidden prompt. New machines
then run Claude Code on your Claude subscription, with no real credential in the machine.

To use Codex on your ChatGPT sign-in, run `codex login --device-auth` on the machine.

**6. Create a workspace on a machine**

In Orca, create a worktree. In **Run on**, select **Per-Workspace Environment**, then **Instabox**.

![Orca's Create worktree dialog with the Run on menu open, Per-Workspace Environment selected and Instabox in its submenu](https://mintcdn.com/instacloud/rpjO_4qZKxC6_teT/images/orca/run-on-instabox.png?fit=max\&auto=format\&n=rpjO_4qZKxC6_teT\&q=85\&s=b53f7ad69728e4b500916ae2fdbb36b6)

The plugin creates a machine named
`orca-<workspace>-<id>`, clones your repo into the machine's home, and checks out your branch.
Orca then installs its server on the machine and connects.

To see the machine, run `instabox ls`. If a step fails, the plugin deletes the machine again.

## Pause, resume, and delete

| In Orca | On the machine |
| - | - |
| The workspace sleeps | The machine pauses. |
| The workspace wakes | The machine resumes and the workspace reconnects. |
| Right-click the workspace and select **Remove Project from Orca** | The machine is deleted with its files, and its SSH token stops working. |

**Delete Worktree** stays disabled, because the clone on the machine is the project's primary checkout.

> [accordion] How Orca connects
>
> Each workspace gets an [SSH token](https://instabox.sh/docs/access/ssh.md) that does not expire. Orca connects to
> `ssh.instabox.sh:2222` with the token as the user name and no key. Orca stores the token with the
> workspace, and the token stops working when the machine is deleted.

> [accordion] Add a machine you already have
>
> ```bash theme={null}
> instabox ssh-token cloud-dev --ttl never --json | jq -r '"Host: \(.host)\nPort: \(.port)\nUser: \(.token)"'
> ```
>
> In **Settings → SSH Hosts**, select **Add host**, paste the fields and leave the identity file
> empty. Connect from the hosts menu at the bottom right, then **Add a project** → the host →
> `/data/home` → **Open as Folder**. The plugin does not manage this machine; see
> [what wakes a machine](https://instabox.sh/docs/machines/auto-pause.md#what-wakes-a-machine).

> [accordion] Which template the machines use
>
> Machines use your account's default template. An SSH remote such as
> `git@github.com:org/repo.git` is cloned over HTTPS, because the machine has no SSH key for your Git
> host.

> [card] [Org Secrets](https://instabox.sh/docs/security/secrets.md)
>
> Proxy and direct Secrets, hosts, and updating a running machine.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> What counts as idle and what wakes a machine.
