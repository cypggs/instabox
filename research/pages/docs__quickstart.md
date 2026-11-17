> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Quick start

> Install the CLI and start your first machine.

At the end you have a machine named `dev`, a shell on it, and the commands to pause, resume, and
delete it.

**1. Install the CLI**

```bash theme={null}
npx instabox@latest
```

The command signs you in, installs the `instabox` CLI, and adds the instabox skills to your coding
agent.

**2. Start a machine**

```bash theme={null}
instabox create --name dev
instabox ssh dev                 # open a shell
instabox exec dev -- uname -a    # or run one command
```

You can also ask your agent to "create an instabox machine and get me into it".

**3. Pause or delete the machine**

```bash theme={null}
instabox suspend dev    # pause: files, memory and processes are kept
instabox resume dev     # pick up where you left off
instabox rm dev         # delete it and its files
```

> [!TIP]
>
> `npx instabox@latest sh` does the first two steps in one go.

> [card] [Concepts](https://instabox.sh/docs/concepts.md)
>
> The machine states (running, paused, and stopped) and the other words the CLI uses.

> [card] [SDK quick start](https://instabox.sh/docs/integrations/sdk-quickstart.md)
>
> The same steps from TypeScript or HTTP.
