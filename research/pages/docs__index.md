> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# InstaBox

> A Linux machine of its own in the cloud for every agent.

InstaBox gives each agent a full Linux VM in the cloud, with its own kernel, root, and Docker. Files and
installed software stay between sessions. A machine pauses by itself when idle, so you pay for runtime
only while it works.

```bash theme={null}
npx instabox@latest sh
```

The command signs you in, installs the `instabox` CLI, and opens a shell on your first machine.

> [card] [Quick start](https://instabox.sh/docs/quickstart.md)
>
> Install the CLI and start your first machine in a minute.

> [card] [Concepts](https://instabox.sh/docs/concepts.md)
>
> Machines, templates, snapshots, and operations.

> [card] [SDK quick start](https://instabox.sh/docs/integrations/sdk-quickstart.md)
>
> Create, pause and delete machines from TypeScript or HTTP.

> [card] [Use cases](https://instabox.sh/docs/use-cases/index.md)
>
> Coding agents, computer use, preview URLs, CI, and Android.

## What you can do with a machine

> [card] [SSH in](https://instabox.sh/docs/access/ssh.md)
>
> With the CLI, or one `ssh` line from any terminal.

> [card] [Publish a port](https://instabox.sh/docs/access/ports.md)
>
> An HTTPS URL for a web server inside the machine.

> [card] [Watch the desktop](https://instabox.sh/docs/access/desktop.md)
>
> See and drive a browser or desktop app.

> [card] [Save a snapshot](https://instabox.sh/docs/machines/snapshots.md)
>
> Set a machine up once, start new ones from it.

> [card] [Keep secrets out](https://instabox.sh/docs/security/secrets.md)
>
> The machine sees a placeholder. The proxy adds the real key.

> [card] [Connect over MCP](https://instabox.sh/docs/integrations/mcp.md)
>
> Let Claude, Cursor, or Codex drive machines as tools.

> [!TIP]
>
> Docs for agents: every page is also Markdown (add `.md` to its URL). The whole site is at
> `/llms.txt` and `/llms-full.txt`.
