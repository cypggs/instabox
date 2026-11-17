> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Agent sign-up

> An agent with no browser signs itself up, and the account owner approves later with a code.

Agent sign-up lets an agent that has no instabox credentials, and nobody at the keyboard, get a key
by itself. The agent registers, prints a code for the account owner, and keeps working. Once the owner
approves, which they can do from any device, the agent holds an ordinary [API key](https://instabox.sh/docs/security/api-keys.md)
of the owner's account with the scopes that the owner granted.

```bash theme={null}
instabox login --agent
```

The command prints a claim page and a code, then waits for the owner.

```text theme={null}
Ask the account owner to approve it:
  open https://dashboard.instabox.sh/dashboard/claim?code=KFXR-7Q2M
  (or https://dashboard.instabox.sh/dashboard/claim and enter the code KFXR-7Q2M)
Waiting for the account owner to approve…
```

Relay the page and the code to the person and wait for the approval. The code works for 24 hours, so the
owner does not have to approve right away.

## Register with a name, scopes, and an expiry

The default name is `agent@<hostname>`, and the default scopes are `machines:read machines:write
machines:delete images:read`. To ask for a different name, other scopes, or an expiry, add the flags
when the agent registers.

```bash theme={null}
instabox login --agent --name 'claude on ci' \
  --scope machines:read,machines:write --expires-in-days 30
```

## Approve from the dashboard

> [!WARNING]
>
> Approve only an agent you started yourself. The key appears on your **API keys** page under the
> agent's name. When the agent is done, revoke the key there or with `instabox keys rm`.

Open the page that the agent printed. If you are signed out, sign in or sign up with GitHub or Google
first, and the page comes back. The page shows the agent's name, when and from which address it
registered, and what it asks to do, so you can check that this is your agent. Untick the permissions
that you do not want to give, then choose **Approve** or **Deny**.

## Approve from your computer

You can also decide from a terminal on your own computer instead of the dashboard. Sign in there and
run `instabox claim` with the code. The command shows what the agent asks for, then asks you before it
approves.

```bash theme={null}
instabox claim KFXR-7Q2M
```

## Grant less than the agent asks for

You can grant fewer scopes or a shorter expiry than the agent asked for when its task needs less. Pass
`--scope` and `--expires-in-days` with the values that you want to grant. With `--yes`, the command
approves without asking.

```bash theme={null}
instabox claim KFXR-7Q2M --scope machines:read \
  --expires-in-days 7 --yes
```

## Deny an agent

To refuse the agent, pass `--deny`. The agent's command then ends with an error, and the agent gets
no key.

```bash theme={null}
instabox claim KFXR-7Q2M --deny
```

## Continue after approval

After the owner approves, the agent's `instabox login --agent` command finishes on its own, so the
agent needs no second step. The command stores the key in `~/.config/instabox/credentials.json`, and
every later `instabox` command uses that key.

## Limits

* Before approval, the agent has no account and no credential, so it cannot read or create anything.
  What it needs to plan with (templates, sizes, prices, and limits) is in these docs and in
  `/llms.txt`.
* An agent's key never manages keys, workspaces, or billing.
* An account holds at most 20 active keys.

> [accordion] Waiting, resuming and failures
>
> * `--no-wait` prints the page and code and exits. With `--json`, it prints them as JSON. Running
>   `instabox login --agent` again resumes the same registration instead of printing a second code.
> * A denial or an expired code ends with an error. Register again.
> * If the key cannot be stored, such as on a full disk, the command says so without printing the key
>   and exits non-zero. Run it again within 10 minutes and the server hands over the same key. A key
>   neither used nor stored within those 10 minutes is revoked.

> [accordion] --json prints the secret
>
> `--json` prints the key itself, and `--no-wait --json` prints the claim token. Keep that output out
> of transcripts and logs.

> [accordion] Over HTTP
>
> `POST /v1/auth/agent` takes `{"name", "email", "scopes", "expiresInDays"}`, all optional. It answers
> `claimToken`, `userCode`, `verificationUri`, `verificationUriComplete`, `interval`, and `expiresIn`.
>
> Poll `POST /v1/auth/agent/token` with `{"claimToken"}` every `interval` seconds. It answers `400`
> with one of these errors:
>
> | Error | Meaning |
> | - | - |
> | `authorization_pending` | the owner has not decided yet |
> | `slow_down` | poll less often: the server added 5 s to its interval. Doubling yours catches up from any gap. |
> | `expired_token` | register again |
> | `access_denied` | the owner refused |
>
> Then, once, it answers `200` with `key` (the secret, `ibk_...`), `apiKey` (its ID, name, and scopes),
> and the account's `userId` and `login`. From then on, send the key as `X-API-Key`.

> [card] [API keys](https://instabox.sh/docs/security/api-keys.md)
>
> Scopes, workspaces and revoking.

> [card] [Coding agents](https://instabox.sh/docs/use-cases/coding-agents.md)
>
> Give each agent a machine of its own.
