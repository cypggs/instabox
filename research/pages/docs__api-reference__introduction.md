> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# API overview

> Base URL, authentication, operations, idempotency and errors that every endpoint shares.

The REST API is what the CLI, the SDK, the MCP server and the dashboard call. Every endpoint shares
the rules on this page. The endpoints themselves are in the sidebar.

```bash theme={null}
curl https://api.instabox.sh/v1/machines -H "X-API-Key: $INSTABOX_API_KEY"
```

## Base URL

```
https://api.instabox.sh
```

Request and response bodies are JSON, except on these endpoints:

* **File uploads and downloads** are raw bytes (`application/octet-stream`): the request body of
  `PUT /v1/machines/{id}/files` and the response of `GET /v1/machines/{id}/files/content`.
* **Streams** are server-sent events (`text/event-stream`): `POST /v1/machines/{id}/exec` with
  `Accept: text/event-stream`, and `GET /v1/machines/{id}/desktop/watch`.
* **WebSockets** answer `101 Switching Protocols` and then carry a stream, not JSON:
  `GET /v1/machines/{id}/terminal` and `GET /v1/machines/{id}/vnc`.
* **`204 No Content`** responses, such as a revoke or a delete, have no body.

Each endpoint page shows its content type. Times are RFC 3339, in UTC.

## Authentication

Send an [API key](https://instabox.sh/docs/security/api-keys.md) in the `X-API-Key` header, or as `Authorization: Bearer`.
Create keys in the dashboard under **API keys** in the sidebar. A restricted key reaches only the
endpoints its scopes allow.

## Changes run as operations

Creating, starting, stopping, pausing, resuming and deleting a machine answer `202 Accepted` with an
**operation** at once. [Get an operation](https://instabox.sh/docs/api-reference/operations/get-an-operation-the-progress-of-a-sandbox-change.md)
follows it until it ends `succeeded` or `failed`. A machine runs one operation at a time.

## Idempotency

Endpoints that list `Idempotency-Key` under **Headers** require one: any unique string, such as a UUID.
Retrying a completed request with the same key returns its response instead of acting twice. The same
key on a different request is `409 idempotency_conflict`. Keys are kept for 7 days.

## Confirming deletes

Deleting a machine (`DELETE /v1/machines/{id}`) and deleting a snapshot (`DELETE /v1/snapshots/{id}`)
take an `X-Confirm-Delete` header: the ID or exact name of the machine or snapshot to delete. Any other value is
`409 delete_confirmation_required`, and nothing is deleted, so a mistyped ID cannot remove another
machine. The same `Idempotency-Key` then works for the corrected request.

> [!NOTE]
>
> API keys created from the rollout on must send the header on a machine delete. Without it, the delete
> is `409 delete_confirmation_required`, the message names the ID to send, and the machine is kept.
> Older keys and sign-in sessions (the dashboard, `instabox login`) can still leave it out. A delete by an
> older key without the header gets a `Warning` header in the response. The CLI's `rm` and `snapshot rm`,
> the SDKs, the MCP server and the dashboard send the header.

## Errors

Errors use the HTTP status and a JSON body whose `code` says what went wrong. Branch on `code`, never
on `message`.

```json theme={null}
{
  "error": {
    "code": "machine_not_running",
    "message": "machine is stopped; start it first",
    "requestId": "req_fbckiomplsq3fqin5dvsvzpnli",
    "retryable": false,
    "operationId": null
  }
}
```

* `retryable` is true when the same request can succeed later.
* `retryAfterSeconds`, when present (`no_capacity`, `rate_limited`, `machine_cooldown`, agent sign-up
  `quota_exceeded`), is how many seconds to wait before a retry. An error response also sends it as the
  `Retry-After` header. A failed operation carries only the field, in its `error`.
* `docs_url` links the code's entry in [Errors](https://instabox.sh/docs/reference/errors.md), which lists every code, the 402
  billing codes among them, with whether to retry and what to do.

| Status | Code | When |
| - | - | - |
| 400 | `bad_request` | The request is malformed: invalid JSON, a missing or invalid field, a value out of range |
| 400 | `unknown_field` | A body field the request does not have, on create, fork, restore, start, stop, suspend, resume, wake and update of a machine. The message suggests the closest field |
| 400 | `spec_not_allowed` | A size your plan does not offer, a disk size at create, or growing a free machine's disks. `GET /v1/orgs` lists each workspace's sizes as `allowedSpecs` |
| 401 | `missing_key`, `malformed_key`, `invalid_key`, `expired_key`, `revoked_key`, `session_expired` | A credential problem: [authentication errors](#authentication-errors) |
| 401 | `unauthenticated` | A bad refresh token or ticket |
| 403 | `permission_denied` | The caller is not allowed to do this, for example not the owner or an admin of the workspace |
| 403 | `missing_scope` | The key lacks the scope |
| 403 | `template_not_allowed` | The free plan creates only from agent-browser, android, omarchy, ci. A snapshot is not limited |
| 404 | `not_found` | No such machine, snapshot or other resource, or it belongs to another workspace |
| 409 | `operation_conflict` | Another operation runs, which `operationId` names, or the change is not allowed now |
| 409 | `idempotency_conflict` | The same `Idempotency-Key` on a different request ([Idempotency](#idempotency)) |
| 409 | `machine_not_running` | The request needs the machine running, and it is stopped or paused |
| 409 | `delete_confirmation_required` | `X-Confirm-Delete` names neither the ID nor the name, or a key that must send it left it out |
| 409 | `fork_unavailable` | The machine cannot be forked on its host. Not retryable when it is suspended, has no host, or its host cannot copy disks: save a snapshot instead. Retryable when its host is under maintenance (`reason` `source_host_unavailable`) or full (`source_host_full`): stop and start the machine to move it, or save a snapshot |
| 429 | `quota_exceeded` | A [limit](https://instabox.sh/docs/account/limits.md) was reached. The message says which |
| 503 | `no_capacity` | No host has room. With `retryAfterSeconds` and `Retry-After` when retryable |
| 503 | `host_unavailable` | The machine's host cannot be reached. Retry when `retryable` is true. An `exec` that already started is not retryable |

> [accordion] Retrying no\_capacity
>
> `no_capacity` (no host has room right now, as a 503 or a failed operation) is retryable. Wait
> `retryAfterSeconds` (60, which the 503 also sends as the `Retry-After` header). Then retry with
> exponential backoff (for example every few minutes), or try another size or template. A create whose
> operation failed replays that failure for the same `Idempotency-Key`, so retry with a new key. The
> CLI's `create` exits 75 on `no_capacity`. `run` exits 75 with `--fail-fast`, and 125 without it.

## Authentication errors

Each credential problem has its own `code` (`reason` repeats it), with `hint` saying how to fix it and
`docs` linking here. All are HTTP 401. Older servers sent `unauthenticated`, `api_key_expired` or
`api_key_revoked` instead. Those are credential errors too.

| `code` | When |
| - | - |
| `missing_key` | No `Authorization` or `X-API-Key` header, or an empty `Bearer` token (an unset variable) |
| `malformed_key` | Not an API key (`ibk_` and 52 characters) or access token, or `Authorization` without `Bearer` |
| `invalid_key` | Well-formed, but no such key or access token |
| `expired_key` | The key or token expired |
| `revoked_key` | The key was revoked, or the session signed out |
| `session_expired` | The dashboard's session cookie is no longer valid |

```bash theme={null}
curl https://api.instabox.sh/v1/machines -H 'Authorization: Bearer garbage'
```

```json theme={null}
{
  "error": {
    "code": "malformed_key",
    "reason": "malformed_key",
    "message": "malformed API key: expected an API key (\"ibk_...\") or an access token (\"iba_...\"); create an API key at …",
    "hint": "create an API key at https://dashboard.instabox.sh/dashboard/api-keys and send it as \"Authorization: Bearer <key>\" or \"X-API-Key: <key>\"",
    "docs": "https://instabox.sh/docs/api#authentication-errors"
  }
}
```

A key without the scope a route needs gets `403 missing_scope` (older servers: `insufficient_scope`)
with the scopes the route accepts in `requiredScopes`. The message names them too, such as
`this API key lacks the machines:delete scope this route needs`, and a `hint` says to create a key
with one.

## OpenAPI

The full machine-readable description, every public endpoint included, is at
[https://api.instabox.sh/v1/openapi.json](https://api.instabox.sh/v1/openapi.json).

> [card] [Create a machine](https://instabox.sh/docs/api-reference/machines/create-a-sandbox.md)
>
> `POST /v1/machines`: template or snapshot, size, idle pause, metadata.

> [card] [Get an operation](https://instabox.sh/docs/api-reference/operations/get-an-operation-the-progress-of-a-sandbox-change.md)
>
> `GET /v1/operations/{id}`: follow a change until it succeeds or fails.

> [card] [Run a command](https://instabox.sh/docs/api-reference/connections/run-a-command-in-the-sandbox-and-return-its-output.md)
>
> `POST /v1/machines/{id}/exec`: one command, its output and exit code.

> [card] [Errors](https://instabox.sh/docs/reference/errors.md)
>
> Every code, with whether to retry and what to do.
