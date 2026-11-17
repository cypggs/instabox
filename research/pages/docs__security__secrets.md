> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Org Secrets

> Credentials the outbound proxy adds to HTTPS headers, so the real value never lands in the machine.

An Org Secret is an encrypted `key=value` pair that an org owns, and it gives the apps in a machine a
credential without putting the real value in the machine. Use Org Secrets for the API keys and tokens
that your code sends in HTTPS headers, such as a model provider key or a GitHub token.

A machine sees each Secret as an environment variable whose value is an opaque placeholder
(`ibsec_...`). Your app sends that placeholder in its usual header, to its usual HTTPS URL, and the
outbound proxy swaps in the real value for the hosts that the Secret allows. The real key never lands
on the machine's disk or in its snapshots.

```mermaid theme={null}
flowchart LR
  A["Your app in the machine"] -->|"header with the placeholder"| B["Outbound proxy"]
  B -->|"header with the real value"| C["An allowed host"]
```

```bash theme={null}
instabox secrets set SERVICE_API_KEY --org team \
  --host api.example.com
```

To save a Secret, you need to be an owner or admin of the org. The CLI reads the value from stdin, or
from a prompt without echo.

## Give a Secret to a new machine

Select Secrets when you create the machine. You can pass `--secrets all` to give the machine every
current and future proxy Secret of its org.

```bash theme={null}
instabox create --name app --org team --secrets all
```

New machines made in the dashboard receive all current and future proxy Secrets of their workspace,
once Secrets is enabled for your account and on the server. Existing machines keep their selections,
and you can change those selections with the CLI or the HTTP API, as the sections below show. A machine
uses Secrets from its own org only.

## Use a Secret in your app

Your app reads `SERVICE_API_KEY` from its environment and sends it as it always would, in a header
such as `Authorization` or `X-Api-Key`. There is no separate retrieval API to call.

```bash theme={null}
curl https://api.example.com/ \
  -H "Authorization: Bearer $SERVICE_API_KEY"
```

### Use a Secret for git over HTTPS

Git sends its token as Basic auth, which the proxy handles, so `git clone` and `git push` to private
repos work with a placeholder. When you save the token, add `github.com` (for git) to the Secret's
hosts next to `api.github.com` (for `gh`).

```bash theme={null}
printf %s "$TOKEN" | instabox secrets set GH_TOKEN \
  --host api.github.com --host github.com
```

Then give git a credential helper that returns the placeholder. For `GH_TOKEN`, `gh` sets one up with
this command.

```bash theme={null}
gh auth setup-git
```

### Turn off QUIC in browsers

The proxy sees HTTPS over TCP port 443 only. HTTP/3 (QUIC, on UDP 443) bypasses the proxy, so the API
receives the placeholder and usually refuses it, for example with `401`. curl and most HTTP libraries
use TCP, but Chromium may still pick HTTP/3. When a Chromium-based browser sends Secrets, turn QUIC
off in that browser, and pass the same flag when you start it from Playwright or Puppeteer.

```bash Chromium theme={null}
chromium --disable-quic
```

```js Playwright theme={null}
chromium.launch({ args: ['--disable-quic'] })
```

```js Puppeteer theme={null}
puppeteer.launch({ args: ['--disable-quic'] })
```

The browser must also trust the sandbox CA with normal certificate checks. Do not turn verification
off.

## Choose the hosts a Secret may go to

Each Secret lists the exact hosts it may be sent to, and you name them with `--host` when you save the
Secret. Without `--host`, the Secret may go to every public HTTPS destination (`*`). Partial wildcards
such as `*.example.com` are not supported.

> [!WARNING]
>
> With `*`, any process in the machine can send the Secret to any public HTTPS host, including one it
> controls. When the credential must stay with one service, use exact hosts.

### Scope a model provider key to its API

For a model provider key, you can pass `--ai` instead of `--host`, which scopes the key to that
provider's API. The proxy puts the key into `x-api-key` or `Authorization` only for that host.

```bash theme={null}
instabox secrets set ANTHROPIC_API_KEY --ai anthropic
```

Each `--ai` value stands for one host. You can repeat `--ai` and combine it with `--host` when one key
must reach more than one host.

| `--ai` | Host |
| - | - |
| `anthropic` | `api.anthropic.com` |
| `openai` | `api.openai.com` |
| `gemini` | `generativelanguage.googleapis.com` |
| `openrouter` | `openrouter.ai` |

Then select the Secret for a machine, so the apps in that machine can use the key.

```bash theme={null}
instabox secrets select app --secrets ANTHROPIC_API_KEY
```

## Manage a machine's Secrets

After you create a machine, you can check which Secrets it has, change its selection, and refresh
their delivery. Each task has a CLI command and a call on `/v1/machines/{id}/secrets`.

### Check a machine's Secrets

To see which Secrets a machine has, show its selection. The output lists the selection, the direct
references, and the proxy status.

```bash CLI theme={null}
instabox secrets show app
```

```bash API theme={null}
curl https://api.instabox.sh/v1/machines/m_.../secrets \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

### Change a running machine's Secrets

Select the Secrets that a machine should have. The new selection replaces the previous one, so name
every Secret that the machine should keep.

```bash CLI theme={null}
instabox secrets select app --secrets SERVICE_API_KEY
```

```bash API theme={null}
curl -X PATCH https://api.instabox.sh/v1/machines/m_.../secrets \
  -H "X-API-Key: $INSTABOX_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"secrets": ["SERVICE_API_KEY"]}'
```

You can pass `--secrets all` to include Secrets created later, or `--secrets none` to block the
machine from all Secrets. Rotating a value keeps its placeholder, so apps need no restart after a
rotation. If you remove a Secret and recreate it with the same name, an explicit grant to the old one
is not restored.

### Refresh a machine's Secrets now

To refresh the delivery of a running machine's selection now, push it. A push is also the last step
after you fix a delivery error that `instabox secrets show` reports.

```bash CLI theme={null}
instabox secrets push app
```

```bash API theme={null}
curl -X POST https://api.instabox.sh/v1/machines/m_.../secrets \
  -H "X-API-Key: $INSTABOX_API_KEY"
```

## Import a .env file

You can import a `.env` file to save each of its variables as a Secret. The whole file is validated
and written at once, and every entry shares the host scope that you pass.

```bash theme={null}
instabox secrets import .env --org team --host api.example.com
```

Importing does not give any machine the Secrets. Select them with `instabox secrets select`, as under
[Change a running machine's Secrets](#change-a-running-machines-secrets).

## Put the real value in the machine with a direct Secret

A direct Secret (`--direct` in the CLI, `delivery: "env"` in the API) puts the **real value** in the
machine, as a variable or a file. Use one when the proxy cannot carry the credential, such as a
database URL, an SSH key, or a login file. Any code in the machine can read and copy a direct Secret,
so prefer the proxy for HTTPS API keys.

[Direct Secrets](https://instabox.sh/docs/security/direct-secrets.md) shows how to save a direct Secret and reference it from a
machine.

## Select Secrets over the HTTP API

You can select Secrets in the body of `POST /v1/machines` when you create a machine. This body selects
one proxy Secret and references two direct Secrets, one as a variable and one as a file:

```json theme={null}
{
  "name": "app",
  "templateId": "agent-browser",
  "orgId": "org_...",
  "secrets": ["SERVICE_API_KEY"],
  "directSecrets": [
    { "secret": "PROD_PG", "env": "DATABASE_URL" },
    { "secret": "OPENCODE_AUTH", "file": ".local/share/opencode/auth.json" }
  ]
}
```

After the machine exists, use `/v1/machines/{id}/secrets`:

* `PATCH` sets a new selection. Explicit `null` selections are rejected.
* `GET` returns metadata and status.
* `POST` refreshes delivery now.

`directSecrets` replaces the machine's direct references. `"envSecrets": ["DATABASE_URL"]` is the
same-name shorthand. Send one or the other, not both. `"secrets": "all"` never includes env (direct)
Secrets, which reach a machine only through `envSecrets` or `directSecrets`.

For one command, `POST /v1/machines/{id}/exec` takes `"secretEnvs": {"DATABASE_URL": "PROD_PG"}`.

To save a Secret over HTTP, see "Save a Secret" under [Reference](#reference). Full schemas are in the
[API reference](https://instabox.sh/docs/api-reference/introduction.md).

## How it works

The proxy carries proxy Secrets, and files and variables carry direct ones. The details below say what
the proxy changes in a request, when a change reaches a machine, and which processes see the values.
Where a direct value lives in the machine is under
[How it works](https://instabox.sh/docs/security/direct-secrets.md#how-it-works) on Direct Secrets.

> [accordion] What the proxy replaces, and what it does not
>
> The proxy checks the machine's authorization, the host scope, and the upstream TLS certificate. Then
> it replaces the placeholder inside a header value. It supports HTTPS on port 443 only, and it does
> not substitute URL parameters or request bodies.
>
> * For HTTP Basic auth in `Authorization`, the proxy decodes the credential, replaces the placeholder
>   inside `user:password`, and re-encodes it. Any other encoding or signing of the placeholder
>   prevents replacement.
> * The proxy intercepts all supported HTTPS hosts, even without a placeholder. Clients must trust the
>   sandbox CA and cannot pin certificates.
> * The proxy closes intercepted connections to private addresses.
> * WebSockets work. The proxy replaces placeholders in the handshake headers, not in messages.
> * The proxy drops `Alt-Svc` from the responses it relays. Chromium may still choose HTTP/3 from DNS,
>   or from an earlier visit saved in its profile.

> [accordion] Delivery status and timing
>
> `instabox secrets show` reports proxy authorization and placeholder setup separately. A proxy update
> is pending until acknowledged, and an acknowledged lease expires within at most 60 seconds without
> renewal. Requests already admitted, streams included, may finish.
>
> Servers with per-sandbox usage show each Secret's proxied and denied request counts in
> `secrets show` and `GET /v1/machines/{id}/secrets`.

> [accordion] Which processes see the variables
>
> Secrets are written to `/run/instabox/secrets/env` (tmpfs, mode 0600, owned by the machine's user)
> and loaded by `/etc/profile.d/zz-instabox-secrets.sh`. Login shells get them: SSH, `instabox exec`,
> and the exec API.
>
> Processes that skip the profile do not get them, such as systemd units, cron, and `bash -c` without
> `-l`. Running processes keep their old environment until restarted.

> [accordion] Rotation, removal and forks
>
> A new value reaches running machines within about 10 seconds. Files change in place and new shells
> get the new variable, but running processes keep the old one until restarted. Stopped and paused
> machines get the current value when they start or resume. Deleting the Secret or the reference
> removes the variable and the file at the next delivery.
>
> Forks copy the source's Secret selection, both proxy and direct, and resolve the values again. A
> machine created from a snapshot takes only its own selection.

## Reference

The rules that apply only to direct Secrets are under
[Reference](https://instabox.sh/docs/security/direct-secrets.md#reference) on Direct Secrets.

> [accordion] Save a Secret: PUT /v1/orgs/\{id}/secrets/\{name}
>
> | Field | Required | Meaning |
> | - | - | - |
> | `value` | yes | The credential, up to 64 KiB of UTF-8. Proxy values must fit in an HTTP header. |
> | `delivery` | yes | `"proxy"` or `"env"`, on every write, including a rotation. Without it: `400 delivery is required`. |
> | `hosts` | proxy only | Up to 32 exact hostnames, with no scheme, port, path, or wildcard. `"*"` allows every public HTTPS destination. Omitted or `[]` means `["*"]`. An env Secret with non-empty `hosts` is rejected. |
>
> A new Secret answers `201 Created`, and a replacement `200 OK`. Both carry metadata and never the
> value: `id`, `orgId`, `name`, `delivery`, `hosts`, `version`, `size`, `createdAt`, and `updatedAt`.
>
> `POST /v1/orgs/{id}/secrets/import` takes `{"values":{...},"delivery":...,"hosts":[...]}`, with the
> same rules for every entry. A missing `delivery` there answers `400 delivery must be proxy or env`.
> The CLI saves `proxy` unless you pass `--direct`.
>
> This example saves a proxy Secret in your personal org. It keeps the token header in a mode 0600 file
> and sends the body on stdin, so neither appears in the process list:
>
> ```bash theme={null}
> instabox whoami >/dev/null
> AUTH=$(mktemp)
> jq -r '"Authorization: Bearer " + .accessToken' \
>   "${XDG_CONFIG_HOME:-$HOME/.config}/instabox/credentials.json" >"$AUTH"
> ORG=$(curl -sS "https://api.instabox.sh/v1/orgs" -H @"$AUTH" \
>   | jq -r '.orgs[] | select(.personal) | .id')
> curl -sS -X PUT \
>   "https://api.instabox.sh/v1/orgs/$ORG/secrets/SERVICE_API_KEY" \
>   -H @"$AUTH" -H 'Content-Type: application/json' \
>   --data-binary @- <<'EOF'
> {"value":"example-api-key","delivery":"proxy","hosts":["api.example.com"]}
> EOF
> rm -f "$AUTH"
> ```
>
> A heredoc you type still lands in shell history. For a real credential, build the body from a file
> with `jq`, and pipe it into the same `curl`:
>
> ```bash theme={null}
> jq -Rs \
>   '{value: rtrimstr("\n"), delivery: "proxy", hosts: ["api.example.com"]}' \
>   <key.txt
> ```
>
> For an env Secret, send `{"value":"postgresql://...","delivery":"env"}` to
> `.../secrets/DATABASE_URL`.

> [accordion] Who can do what
>
> * **Manage values** (create, import, change, delete): an org owner or admin, signed in, or the
>   workspace's own `workspace:admin` API key sent as `X-API-Key`. After `instabox login`, the CLI's
>   access token works, and `instabox whoami` refreshes it. A personal org's key and narrower keys get
>   `403 missing_scope`. Management reads never return values.
> * **Read** Secret metadata and machine selections: keys with `machines:read`.
> * **Change selections and refresh delivery**: keys with `machines:write`.
>
> To work headless, put the `workspace:admin` key in `INSTABOX_API_KEY`. Org commands then default to
> the key's org, and `secrets rm` needs `--yes` without a terminal.

> [accordion] Reserved names
>
> The name is the variable the machine exports, so it follows the rule for
> [Environment variables](https://instabox.sh/docs/machines/environment.md). A name that env refuses is refused here too
> (`400 bad_request`), including `BASH_ENV`, `ENV`, and `IFS`.
>
> A Secret saved under a name before the name was reserved is not delivered. The rest of the selection
> is delivered, `secrets show` reports `not delivered: <names>`, and `secrets push` answers `400`. To
> fix it, save the value under another name, select it, delete the old Secret, and run `secrets push`.

> [accordion] Vault, and opting out
>
> Vault (per-user) and Org Secrets are independent stores, with no automatic migration. Existing Vault
> env values, file delivery, and metadata selectors keep working, and a machine can use both. For a
> same-named variable, an explicitly selected Secret wins.
>
> * Omitting `secrets` on create keeps the Vault behavior without enrolling the machine in Secrets.
> * `--no-vault --secrets none` disables delivery from both stores on a new machine.
> * `--no-vault`, and the SDK option `vault: false`, affect only Vault. Clearing Secret selections
>   affects only Secrets. Where an SDK does not expose Secret selection fields, use the CLI or HTTP API.
> * `instabox vault export NAME --out recovered.json` remains for items kept by an earlier explicit
>   migration. Its output contains the real value.

> [card] [Direct Secrets](https://instabox.sh/docs/security/direct-secrets.md)
>
> Put the real value in the machine when the proxy cannot carry the credential.

> [card] [API keys](https://instabox.sh/docs/security/api-keys.md)
>
> Scopes, including `workspace:admin`.
