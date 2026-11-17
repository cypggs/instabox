> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Coding agents

> Give Claude Code, Codex, or OpenCode a Linux machine of its own to work on your repo, with the provider key kept out of the machine.

A coding agent works best when it can install packages, run Docker, and break things freely, which
is not what you want on your laptop. This guide gives Claude Code a Linux machine of its own to work
on your repo. Its work stays on the machine's disk between sessions, and an Org Secret keeps the
model provider's key out of the machine.

The steps use the `agent-browser` template, which is the default and comes with Claude Code, Codex,
and OpenCode installed.

**1. Save your provider key as a Secret**

Save the key as an Org Secret, so that the machine only ever sees a placeholder for it. With
`--ai anthropic`, the outbound proxy puts the real key into requests to `api.anthropic.com` and
to no other host.

```bash theme={null}
instabox secrets set ANTHROPIC_API_KEY --ai anthropic
```

The command reads the value from a hidden prompt and confirms that the Secret is saved in your
workspace. When your key is from another provider, you can pass `openai`, `gemini`, or
`openrouter` to `--ai` instead. See
[Scope a model provider key to its API](https://instabox.sh/docs/security/secrets.md#scope-a-model-provider-key-to-its-api).

**2. Create the machine**

Create a machine and select the Secret for it, so that the agent in the machine can use the key.

```bash theme={null}
instabox create --name agent --secrets ANTHROPIC_API_KEY
```

The command waits until the machine runs, and then it prints the machine's row as `instabox ls`
shows it.

**3. Push your repo**

Go to your repo's folder on your computer. `my-repo` stands for that folder on this page.

```bash theme={null}
cd my-repo
```

Then push the folder to the machine. In a git repository, `push` copies only the files that git
tracks, and you can add `--all` when the agent also needs your untracked files.

```bash theme={null}
instabox push agent .
```

Because you gave no destination, the folder lands in the machine's home as `/data/home/my-repo`.
The command reports how many files it pushed.

**4. Run the agent**

To work with the agent yourself, connect to the machine over SSH, which opens a shell.

```bash theme={null}
instabox ssh agent
```

In that shell, go to your repo and start Claude Code.

```bash theme={null}
cd my-repo && claude
```

Claude Code starts in your repo. To show it a screenshot, press Ctrl+V while a PNG is on your
clipboard, which uploads the image and pastes its path. See
[Paste images](https://instabox.sh/docs/access/ssh.md#paste-images).

**5. Pause the machine when you are done**

Pause the machine when you stop working, so that it keeps its files, memory, and processes.

```bash theme={null}
instabox suspend agent
```

When you come back, resume the machine, and the agent carries on where it left off.

```bash theme={null}
instabox resume agent
```

You can also leave the machine alone, because a machine pauses by itself when it is idle and
wakes on the next command or SSH connection. See
[Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md).

## Run the agent without a terminal

When a script should drive the agent, or you do not want to keep an SSH session open, you can
send prompts from your own computer. `agent prompt` runs Claude Code as a background job in the
machine, and you can run Codex instead with `--provider codex`. This command starts the agent in your
repo and follows its work.

```bash theme={null}
instabox agent prompt agent --cwd /data/home/my-repo \
  -f "fix the failing test"
```

The command prints the turn and the job that it started. Because of `-f`, it then follows the agent's
events in your terminal until the turn ends.

You can correct the agent while it works. `agent steer` interrupts the running turn, if there is
one, and resumes the agent with your text.

```bash theme={null}
instabox agent steer agent "use the existing helper instead"
```

To watch a turn that you started without `-f`, follow its events. You can press Ctrl-C to stop
following, which leaves the turn running.

```bash theme={null}
instabox agent events agent -f
```

For any other long command, such as a build, see
[Run a long command in the background](https://instabox.sh/docs/machines/run-commands.md#run-a-long-command-in-the-background).

## Start every agent machine from a snapshot

When you have set a machine up with your repos, tools, and sign-ins, you can save it as a snapshot,
so that you do the setup only once.

```bash theme={null}
instabox snapshot save agent --name agent-base
```

Then start each new agent machine from that snapshot, and it is already set up. See
[Snapshots](https://instabox.sh/docs/machines/snapshots.md).

```bash theme={null}
instabox create --name agent2 --snapshot agent-base
```

## Other credentials

To clone private repos from inside the machine, save a GitHub token as a Secret for `github.com` and
`api.github.com`, and then run `gh auth setup-git` in the machine. See
[Use a Secret for git over HTTPS](https://instabox.sh/docs/security/secrets.md#use-a-secret-for-git-over-https).

Some tools need a login file and not an HTTPS header, such as an OpenCode `auth.json` or an SSH key.
For such a tool, see
[Save a login file as a direct Secret](https://instabox.sh/docs/security/direct-secrets.md#save-a-login-file-as-a-direct-secret).

> [accordion] Let the agent manage machines itself
>
> `npx instabox@latest` adds the instabox skills to your coding agent, so you can ask the agent to
> "create an instabox machine and run the tests there". When your client has no CLI, it can use the
> MCP server instead. See [MCP](https://instabox.sh/docs/integrations/mcp.md).

> [card] [Org Secrets](https://instabox.sh/docs/security/secrets.md)
>
> Proxy and direct Secrets, hosts, and updating a running machine.

> [card] [SSH](https://instabox.sh/docs/access/ssh.md)
>
> Sessions, port forwarding, mosh, and large file copies.

> [card] [Auto-pause and max-run](https://instabox.sh/docs/machines/auto-pause.md)
>
> What counts as idle and what wakes a machine.

> [card] [Parallel forks](https://instabox.sh/docs/use-cases/parallel-forks.md)
>
> Run several agents on copies of the same machine.
