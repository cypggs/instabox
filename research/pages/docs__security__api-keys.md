> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# API keys

> Make a key, send it, pick its scopes, and know which workspace it reaches.

An API key (`ibk_...`) is the credential that code, CI jobs, and agents use to call InstaBox when
nobody is there to sign in. The HTTP API, the SDK, the CLI, and SSH all accept a key. People sign in
with GitHub or Google instead, both to use the dashboard and to make keys.

```bash theme={null}
curl -H "X-API-Key: $INSTABOX_API_KEY" \
  https://api.instabox.sh/v1/machines
```

## Send a key

Over HTTP, send the key as `X-API-Key: <key>` or `Authorization: Bearer <key>`. The CLI and the SDK
read the key from `INSTABOX_API_KEY`, so set that variable once.

```bash theme={null}
export INSTABOX_API_KEY=ibk_...
```

With the key set, the CLI needs no `instabox login`, which suits a CI job or a script. The CLI and
the SDK read the server from `INSTABOX_API_URL`, which defaults to `https://api.instabox.sh`.

## Make a key

You can make a key from the dashboard or the CLI, or you can let an agent ask for one and approve
its request. You see the key's secret only once, so copy it when it is shown.

> [tab] Dashboard
>
> Choose **API keys** in the sidebar, then **Create Key**
> ([dashboard.instabox.sh/dashboard/api-keys](https://dashboard.instabox.sh/dashboard/api-keys)).
> **Full access** suits the SDK, and **Restricted** picks scopes. The key belongs to the workspace
> selected in the sidebar.

> [tab] CLI
>
> This command makes a key of your personal workspace.
>
> ```bash theme={null}
> instabox keys create --name ci
> ```

> [tab] An agent
>
> With nobody at the keyboard, the agent runs this command, and then you approve it. See
> [Agent sign-up](https://instabox.sh/docs/security/agent-sign-up.md).
>
> ```bash theme={null}
> instabox login --agent
> ```

Only a sign-in makes and revokes keys, so a key never makes another key, whatever its scopes are.

## Make a key for a workspace

A key reaches only the workspace it belongs to, so a team's machines need a key of the team's
workspace. To make a key of a workspace other than your personal one, name the workspace with `--org`.
You need to be the owner or an admin of that workspace.

```bash theme={null}
instabox keys create --name ci --org team
```

## List your keys

List your keys to see which ones exist before you make or revoke one. The list shows your keys and
never their secrets.

```bash theme={null}
instabox keys ls
```

## Revoke a key

Revoke a key when you no longer need it, so that nobody can call InstaBox with it again. Revoking a
key also ends its SSH certificates.

```bash theme={null}
instabox keys rm <key>
```

## Pick a key's scopes

A scope limits what a key can do, so you can give a job only the access it needs, such as a key that
reads machines and changes nothing. Pass `--scope` when you make the key, or choose **Restricted** in
the dashboard.

```bash theme={null}
instabox keys create --name ro --scope machines:read
```

| Scope | Allows |
| - | - |
| `machines:read` | list and get machines, operations, metrics and usage |
| `machines:write` | create, start, stop, pause, resume, resize, set auto-pause and labels, exec, files and SSH |
| `machines:delete` | delete machines |
| `images:read`, `images:write`, `images:delete` | list, save and delete snapshots |
| `workspace:admin` | a workspace's key only: everything above plus the workspace's members, invites and Secrets, as an admin of that workspace |

Write and delete include read. Without `--scope`, `instabox keys create` gives a personal key
`machines:read`, `machines:write`, `machines:delete`, and `images:read`, and a workspace key
`workspace:admin`.

> [!NOTE]
>
> Give `images:read` to keys that create machines. The SDK and CLI use it to look snapshot names up.

> [!WARNING]
>
> A key with `machines:write` can select [Org Secrets](https://instabox.sh/docs/security/secrets.md) for a machine and run code
> there, and that code can send the selected Secrets to their allowed hosts. For a new machine that
> should get credentials from neither store, pass `"vault": false, "secrets": []` on create.

## Which workspace a key reaches

Every key belongs to exactly one workspace and reaches nothing outside it. Where you make the key
decides which workspace that is.

* A key made with `instabox keys create` (no `--org`), with `POST /v1/keys`, or on **API keys** with
  your personal workspace selected belongs to your personal workspace. One made with `--org W`, with
  `POST /v1/orgs/{id}/keys`, or with `W` selected belongs to `W`.
* Machines a key creates go in its workspace. Other workspaces' machines, snapshots, and settings are
  `404` to it, and so is a create naming another workspace's `orgId`.
* To work in another workspace, use a key of that workspace, which its owner or an admin makes. Only a
  sign-in reaches every workspace you belong to (`instabox create --org W`).

## Find a key's workspace

When you are not sure which workspace a key belongs to, list workspaces with that key. Because a key
reaches one workspace only, the list holds that one workspace.

```bash CLI theme={null}
instabox workspace ls
```

```bash API theme={null}
curl https://api.instabox.sh/v1/orgs -H "X-API-Key: $INSTABOX_API_KEY"
```

For a workspace key, `GET /v1/me` also has `orgId`.

## Key errors

A bad key gets a `401` whose `error.code` says why: `missing_key`, `malformed_key`, `invalid_key`,
`expired_key`, or `revoked_key`. A key without the scope that a route needs gets `403 missing_scope`,
and `requiredScopes` in the response names the scopes that the route accepts. The full list of error
codes is on [Errors](https://instabox.sh/docs/reference/errors.md).

## What a key cannot do

Some routes take a sign-in (`instabox login` or the dashboard) and refuse any API key with
`403 permission_denied` ("API keys cannot be used on this route"). They cover managing API keys, your
Vault, GitHub CI installations and jobs, creating and deleting workspaces, billing, and sessions.

A workspace's [Org Secrets](https://instabox.sh/docs/security/secrets.md) take a `workspace:admin` key instead. To set up CI,
install the GitHub App from the dashboard ([CI](https://instabox.sh/docs/use-cases/ci.md)).

> [accordion] Every route that refuses an API key
>
> * **API keys:** `GET /v1/keys`, `POST /v1/keys`, `DELETE /v1/keys/{id}`, `GET /v1/orgs/{id}/keys`,
>   `POST /v1/orgs/{id}/keys`, `DELETE /v1/orgs/{id}/keys/{keyId}`
> * **Vault:** `GET /v1/vault`, `PUT /v1/vault/{name}`, `PATCH /v1/vault/{name}`,
>   `DELETE /v1/vault/{name}`, `POST /v1/vault/import`, `POST /v1/vault/{name}/export`
> * **GitHub CI:** `GET /v1/orgs/{id}/github/installations`,
>   `DELETE /v1/orgs/{id}/github/installations/{installationId}`, `GET /v1/orgs/{id}/github/jobs`,
>   `GET /v1/orgs/{id}/github/jobs/{job}`, `GET /v1/orgs/{id}/github/stats`
> * **Workspaces and billing:** `POST /v1/orgs`, `DELETE /v1/orgs/{id}`, `PATCH /v1/orgs/{id}/billing`,
>   `POST /v1/orgs/{id}/billing/checkout`, `POST /v1/orgs/{id}/billing/portal`,
>   `POST /v1/orgs/{id}/billing/plan`, `POST /v1/orgs/{id}/billing/credits/subscribe`,
>   `DELETE /v1/orgs/{id}/billing/subscription`, `POST /v1/orgs/{id}/billing/subscription/resume`,
>   `POST /v1/orgs/{id}/billing/move-personal`, `POST /v1/orgs/move-personal`
> * **Sessions:** `GET /v1/sessions`, `DELETE /v1/sessions/{id}`
>
> The [API reference](https://instabox.sh/docs/api-reference/introduction.md) lists every route a key can call.

> [card] [SDK quick start](https://instabox.sh/docs/integrations/sdk-quickstart.md)
>
> Use the key from TypeScript or HTTP.

> [card] [Agent sign-up](https://instabox.sh/docs/security/agent-sign-up.md)
>
> An agent gets its own key, approved by you.
