> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Environment variables

> Plain variables for every shell in a machine, and where secrets go instead.

Environment variables are plain `NAME=value` settings that every shell in a machine gets. Use them
for configuration that is not secret, such as the mode an app runs in, the name of a region, or a
feature flag.

You set the variables for the whole machine when you create it. Every login shell (`exec`, SSH, the
dashboard terminal) exports them, and so does every process those shells start. The variables are
there from the first command, and they are there again after each start and resume.

```bash theme={null}
instabox create --name api \
  --env NODE_ENV=production --env REGION=us-west-2
```

> [!WARNING]
>
> **Not for secrets.** Every process and user in the machine can read the values (`env`,
> `/run/instabox/env/env`, on macOS `/var/run/instabox/env/env`), and so can anyone who can run a command in it. For an API key you want kept
> out of the machine, use [Org Secrets](https://instabox.sh/docs/security/secrets.md).

## Set variables at create

A machine's variables are set in the request that creates it. In the CLI you pass one `--env` flag
for each variable. Over the API you send all of them together as the `env` object in
`POST /v1/machines`.

```bash CLI theme={null}
instabox create --name api \
  --env NODE_ENV=production --env REGION=us-west-2
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"name": "api", "env": {"NODE_ENV": "production"}}'
```

## Load variables from a file

When your settings are already in a file of `KEY=VALUE` lines, you can pass the file with
`--env-file` and not repeat `--env` for each line. A value of `-` reads the lines from stdin. When
`--env` and the file set the same name, `--env` wins, so you can override one value from the file.

```bash theme={null}
instabox create --name api --env-file ./app.env
```

## Set a variable for one command

You can also set a variable for a single command, for example to run the tests in another mode. Pass
`--env` to `instabox exec`, and the variable applies to that command only.

```bash theme={null}
instabox exec api --env NODE_ENV=test -- npm test
```

## List a machine's variables

To check which variables a machine has, read the machine with `instabox get --json`
(`GET /v1/machines/{id}`). The answer lists the names as `envNames` but never the values, which are
stored encrypted at rest.

```bash theme={null}
instabox get api --json
```

## How it works

* **Fixed at create.** A machine's variables are set at create and stay as they are. A fork gets its
  source's variables, but a machine created from a snapshot gets only the ones in its own create
  request.
* **What wins.** When an exec sets a name the machine also sets, the exec's own `envs` (`--env`) win
  for that command. A Vault item or env Secret with the same name also wins over a machine variable.
* **Not for boot services.** Services that systemd starts at boot do not run a login shell, so they do
  not get the variables. Such a service can read `/run/instabox/env/env` itself.
* **macOS.** Login shells of bash and zsh export the variables: `exec`, SSH and Terminal. Apps you open
  from the Dock or Finder do not get them.

## Keep credentials in Org Secrets

Use [Org Secrets](https://instabox.sh/docs/security/secrets.md) for an API key or another named credential that is shared
within an org. A machine sees only placeholders, and the outbound proxy adds the credentials to the
HTTPS headers that are sent to each Secret's allowed hosts. You can manage the values, and which
machines get them, on the dashboard's **Secrets** page, in the CLI, or over the API.

## Vault

The Vault is each user's own store, and it still works alongside Org Secrets. It supports environment
variables, home files, and metadata selectors. You manage its items with `instabox vault set`,
`set-file`, `import`, `ls`, `rm`, `show`, and `push`.

> [!NOTE]
>
> Upgrading does not move Vault items into Org Secrets or enroll existing machines in Secrets.

## Limits

* Names match `^[A-Za-z_][A-Za-z0-9_]{0,127}$`.
* A machine takes up to 128 variables and 32 KiB in all.
* Windows machines take no env (`400 bad_request`).

> [accordion] Refused names
>
> These names are refused with `400 bad_request`:
>
> * Set by the login shell: `PATH`, `HOME`, `USER`, `LOGNAME`, `SHELL`, `PWD`, `OLDPWD`, `SHLVL`,
>   `HOSTNAME`, `IFS`, `ENV`, `BASH_ENV`. Set them for one command with exec's `envs` (`--env`) instead.
> * Bash's readonly variables: `UID`, `EUID`, `PPID`, `SHELLOPTS`, `BASHOPTS`, `BASH_VERSINFO`.
> * Set by each session: `TERM`, `XDG_RUNTIME_DIR`, `DBUS_SESSION_BUS_ADDRESS`, and names starting
>   `SSH_`.
> * Names starting `INSTABOX_`.
> * The trust roots Org Secrets manage: `SSL_CERT_FILE`, `SSL_CERT_DIR`, `REQUESTS_CA_BUNDLE`,
>   `CURL_CA_BUNDLE`, `NODE_EXTRA_CA_CERTS`.
> * On macOS only, zsh's special parameters, which a zsh login cannot take from the variables
>   (`man zshparam`; for example `status`, `path`, `options`, `HISTSIZE`, `PS1`, `MANPATH` and
>   `USERNAME`), and `ZDOTDIR`. The 400 names the one refused.

> [card] [Org Secrets](https://instabox.sh/docs/security/secrets.md)
>
> Credentials a machine sees only as placeholders.

> [card] [Sizes and create options](https://instabox.sh/docs/machines/sizes.md)
>
> Every field create takes, `env` included.
