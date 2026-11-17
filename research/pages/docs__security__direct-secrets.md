> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Direct Secrets

> Org Secrets whose real value is put in the machine as a variable or a file, for credentials the proxy cannot carry.

A direct Secret is an Org Secret whose **real value** is put in the machine, as an environment
variable or a file. A proxy [Org Secret](https://instabox.sh/docs/security/secrets.md) leaves only a placeholder in the machine,
and the outbound proxy adds the real value to HTTPS headers. With a direct Secret, the program in the
machine reads the credential itself. Use one when the proxy cannot carry the credential, such as a
database URL, an SSH key, or a login file.

To save a Secret as direct, pass `--direct` in the CLI, or send `delivery: "env"` in the API. You need
to be an owner or admin of the org.

```bash theme={null}
instabox secrets set PROD_PG --direct
```

Without a value on stdin, the CLI prompts for it.

> [!WARNING]
>
> Any code in the machine, including an agent or a package it installs, can read and copy a direct
> Secret. Prefer the proxy for HTTPS API keys. To revoke a direct credential, rotate it at its service,
> because copies already read stay.

## Choose between proxy and direct

The proxy replaces a placeholder in an HTTPS header, so it carries API keys, tokens, and git over
HTTPS. A direct Secret is for the credentials in the other rows of this table, and you deliver it as a
file, a variable, or both.

| Use | Delivery |
| - | - |
| API keys and tokens in HTTPS headers, git over HTTPS | proxy (default) |
| Login files (`~/.local/share/opencode/auth.json`, `~/.config/gh/hosts.yml`), SSH keys | direct, as a file |
| Database, Redis, or SMTP URLs. Request signing (AWS SigV4, service-account JWTs, HMAC). Keys in bodies or query strings. Pinned clients. | direct, as a variable or a file |

## Save a login file as a direct Secret

A direct Secret can hold a whole file, such as the login file of a tool. As an owner or admin, save
the Secret as direct. This command reads the value from a login file on stdin.

```bash theme={null}
instabox secrets set OPENCODE_AUTH --direct \
  < ~/.local/share/opencode/auth.json
```

Who may reference direct Secrets, and the rules for names and paths, are under
[Reference](#reference) below.

## Reference a direct Secret from a machine

Saving a direct Secret does not give it to any machine. After you save one, reference it from a
machine as a variable, a home file, or both:

* `--secret-env VAR=SECRET`, or `--secret-env SECRET` for the same name
* `--secret-file PATH=SECRET`

> [!NOTE]
>
> Files (`--secret-file`), variables named differently from their Secret (`--secret-env VAR=SECRET`),
> and `exec --secret-env` need a feature that is not on for instabox.sh yet. Until it is, the CLI
> refuses them and says so. A variable with the Secret's own name works today: `--secret-env SECRET`.

This command creates a machine with one Secret as a file and another as the variable `DATABASE_URL`.

```bash theme={null}
instabox create --name agent \
  --secret-file .local/share/opencode/auth.json=OPENCODE_AUTH \
  --secret-env DATABASE_URL=PROD_PG
```

To reference direct Secrets over HTTP, send `directSecrets` or `envSecrets`, as under
[Select Secrets over the HTTP API](https://instabox.sh/docs/security/secrets.md#select-secrets-over-the-http-api).

## Remove a machine's direct references

To remove every direct reference from a machine, select `none`. The variables and the files are then
removed from the machine at the next delivery.

```bash theme={null}
instabox secrets select agent --secret-env none
```

## Give a direct Secret to one command

When only one command needs the credential, you can pass `--secret-env` to `instabox exec`, and the
variable is set for this command only. This flag needs the feature that is not on for instabox.sh
yet, as the note under
[Reference a direct Secret from a machine](#reference-a-direct-secret-from-a-machine) says.

```bash theme={null}
instabox exec agent --secret-env DATABASE_URL=PROD_PG \
  -- psql -c 'select 1'
```

## How it works

Variables and files carry a direct Secret into the machine, and the accordion below says where the
value lives. Which processes see the variables, and how rotation, removal, and forks behave, are under
[How it works](https://instabox.sh/docs/security/secrets.md#how-it-works) on Org Secrets.

> [accordion] Where a direct value lives
>
> Variables go to `/run/instabox/secrets/env`. A file's value is written to
> `/run/instabox/secrets/files/` (tmpfs, mode 0600, owned by the machine's user), and the home path
> becomes a symlink to it. So the disk, disk snapshots, and forks never hold the value.
>
> If the path already holds a file of the machine's, the file is left alone and `secrets show` reports
> the error. Move the file and run `secrets push`. A tool that rewrites its login file replaces the
> link with a real file on the disk, which is then the machine's own.

## Reference

> [accordion] Who can reference direct Secrets
>
> Only an org owner or admin can save a Secret as direct. Anyone who can change a machine
> (`machines:write`) in the org can reference the org's direct Secrets. A proxy Secret can never be
> referenced as direct, and a Secret of another org cannot be referenced at all.

> [accordion] Direct Secrets: names, paths and limits
>
> Variable names follow the Secret name rule. They cannot be `PATH`, `HOME`, `BASH_ENV`, `IFS`, or the
> CA variables, and they cannot start with `INSTABOX_` or `SSH_`.
>
> Paths are relative to the machine's home directory: write `.ssh/id_ed25519`, not
> `~/.ssh/id_ed25519`. The shell turns an unquoted `~/` into your own home, and the CLI takes that back
> as home-relative. A machine has at most 64 direct references.
>
> No API returns a value. `secrets show` and `GET /v1/machines/{id}/secrets` list `directSecrets` as
> names and paths. Each reference (create, select, fork, exec) is audited by name.

> [card] [Org Secrets](https://instabox.sh/docs/security/secrets.md)
>
> Keep the real value out of the machine, and let the proxy add it to HTTPS headers.

> [card] [Environment variables](https://instabox.sh/docs/machines/environment.md)
>
> Plain variables, and the names a machine refuses.
