> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# ADEs

> Connect a desktop app for coding agents to a machine, so its agents work there and not on your laptop.

Some desktop apps are made for running coding agents, and cmux, Orca, Paradev, and T3 Code are
four of them. Such an app is often called an agentic development environment (ADE). When you
connect one to an InstaBox machine, its agents run in the machine and not on your laptop.

If you use one of these four apps, follow its page. For any other app that can connect over SSH,
follow the steps under [Connect any ADE that supports SSH](#connect-any-ade-that-supports-ssh).

> [card] [cmux](https://instabox.sh/docs/use-cases/ade/cmux.md)
>
> Open a cmux workspace on a machine from one link.

> [card] [Orca](https://instabox.sh/docs/use-cases/ade/orca.md)
>
> Give each Orca workspace its own machine. A plugin creates, pauses, and deletes the machines.

> [card] [Paradev](https://instabox.sh/docs/use-cases/ade/paradev.md)
>
> Add a machine as a Paradev host with its SSH URI.

> [card] [T3 Code](https://instabox.sh/docs/use-cases/ade/t3-code.md)
>
> Pair T3 Code with a machine from the `t3` template.

## Connect any ADE that supports SSH

When an ADE has no InstaBox plugin, you can add the machine to your SSH config. The ADE then lists
the machine as a host.

**1. Create the machine**

Create the machine that the agents will work in. This page calls it `dev`.

```bash theme={null}
instabox create --name dev
```

The command waits until the machine runs, and then it prints the machine's row as
`instabox ls` shows it.

**2. Add the machine to your SSH config**

Add a `Host` block for the machine to `~/.ssh/config`, so that your ADE and plain `ssh` know
how to connect. The command also creates the certificate that the block uses.

```bash theme={null}
instabox ssh-config dev >> ~/.ssh/config
```

To check the block, run `ssh` with the `Host` name from the block, which is `instabox-dev` for
a machine named `dev`.

```bash theme={null}
ssh instabox-dev
```

You get a shell in the machine.

**3. Connect from the ADE**

In the ADE's remote or SSH settings, pick the `Host` name from the block, which
is `instabox-dev` here.

## Reconnect when the certificate has expired

The certificate is short-lived. When the ADE can no longer connect, refresh the certificate, and
then connect again. [Use your own SSH client](https://instabox.sh/docs/access/ssh.md#use-your-own-ssh-client) says more about
the certificate.

```bash theme={null}
instabox ssh-cert dev
```

> [card] [SSH](https://instabox.sh/docs/access/ssh.md)
>
> Tokens, port forwarding, and mosh.

> [card] [Coding agents](https://instabox.sh/docs/use-cases/coding-agents.md)
>
> Run an agent in a machine from your terminal.
