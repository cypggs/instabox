> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Errors

> Every error code the API, CLI and SDK return, with whether to retry and what to do.

Each code below has its own anchor, which an error body's `docs_url` links to. The body itself and
the authentication errors are in the [API overview](https://instabox.sh/docs/api-reference/introduction.md#errors).

Every error has an HTTP status and a JSON body: `{"error": {"code", "message", "requestId", "retryable", "operationId", "docs_url"}}`. Branch on `code`, never on `message`. `retryable` is true when the same request may succeed later; `retryAfterSeconds` (and the `Retry-After` header), when present, is how many seconds to wait first. `docs_url` links the code here. A status below with "or" means the code is sent with either. Retry with the same `Idempotency-Key` only to repeat the exact same request. The [API overview](https://instabox.sh/docs/api-reference/introduction.md#errors) has the body in full.

Retryable below: **yes** means wait and send the same request again; **no** means change something first (the fix says what); **sometimes** means read `retryable` in the body.

## 400 Bad Request

### `bad_request`

HTTP 400 or 405 or 408 or 413 or 416. Retryable: no.

* When: The request is malformed: invalid JSON, a missing or invalid field, a value out of range. 405: a write to a read-only admin route. 408: a file transfer ran past its time limit. 413: feedback (`instabox feedback`) too large for the support service.
* What to do: Read the message, which names the field, fix the request and send it again.

### `feature_disabled`

HTTP 400. Retryable: no.

* When: The route exists but this server has not turned it on yet.
* What to do: Do without it for now, or ask support when it will be on.

### `unknown_field`

HTTP 400. Retryable: no.

* When: A machine create, fork, restore, start, stop, suspend, resume, wake or update body has a field the request does not have (such as `diskGiB`, the TypeScript SDK name, in a REST body). Nothing was changed.
* What to do: Drop or rename the field; the message suggests the closest one.

### `resize_restart_off`

HTTP 400. Retryable: no.

* When: `PATCH /v1/machines/{id}` sent `apply: "restart"` to a server that does not take it yet (`GET /v1/me` `resizeRestart` is absent).
* What to do: Stop the machine, send the `PATCH`, then start it.

### `authorization_pending`

HTTP 400. Retryable: yes.

* When: Device sign-in: the user has not approved the code yet.
* What to do: Keep polling at the `interval` the start of the flow returned.

### `slow_down`

HTTP 400. Retryable: yes.

* When: Device sign-in: polling faster than the `interval`.
* What to do: Poll less often (add 5 seconds to the interval), then keep polling.

### `expired_token`

HTTP 400. Retryable: no.

* When: Device sign-in: the device code expired before it was approved.
* What to do: Start the sign-in again (`instabox login`).

### `access_denied`

HTTP 400. Retryable: no.

* When: Device sign-in: the user denied the request.
* What to do: Start the sign-in again if that was a mistake.

### `invalid_grant`

HTTP 400. Retryable: no.

* When: A sign-in code exchange: the code is unknown, used or expired, or the verifier or redirect URI does not match.
* What to do: Start the sign-in again.

### `invite_invalid`

HTTP 400. Retryable: no.

* When: The invite code is unknown, expired, revoked or used up.
* What to do: Check the code, or ask for a new one.

### `spec_not_allowed`

HTTP 400 or 403 or 409. Retryable: no.

* When: 400: the request names a machine size or disk size the plan does not offer, or tries to resize a free machine. 403: the machine is larger than the plan now offers, or the request asks an idle time (`autoSuspendAfterSeconds`) the plan does not offer (the free plan keeps 30 minutes). 409: a restore from a snapshot that does not fit the machine (another host pool, or a disk size it cannot take).
* What to do: Leave the size fields out, or pick one of the `allowedSpecs` that `GET /v1/orgs` (`instabox workspace ls --json`) lists. For another size or idle time, change the plan ([Pricing](https://instabox.sh/docs/account/pricing.md)); on the free plan leave `autoSuspendAfterSeconds` out. For a restore, create a new machine from the snapshot instead.

### `invalid_guest_user`

HTTP 400. Retryable: no.

* When: A create's `guestUser` is not a valid Linux user name, or is reserved (`root`, `instabox`, system users and groups such as `sudo`).
* What to do: Choose another lower-case user name, or leave `guestUser` out.

### `minimum_lease_required`

HTTP 400. Retryable: no.

* When: A macOS sandbox leases a whole Mac to its workspace for at least 24 hours, billed whatever the sandbox does. A create, fork, start or restore that would start a new lease did not accept it; the message gives the price.
* What to do: Accept the lease: send `acceptMinimumLease: true` (CLI `--accept-24h-minimum`, Python `accept_minimum_lease=True`), or confirm the dialog in the dashboard.

### `data_disk_not_browsable`

HTTP 400. Retryable: no.

* When: `instabox snapshot files` (GET /v1/snapshots/\{id}/files) names a path on /data, the data disk, which it does not read. The message says whether the snapshot holds that disk.
* What to do: Create a sandbox from the snapshot and look there (`instabox create --snapshot S`), or browse a path outside /data.

## 401 Unauthorized

### `unauthenticated`

HTTP 401. Retryable: no.

* When: A bad refresh token, ticket or other credential that is not an API key (older servers sent it for every credential error).
* What to do: Sign in again, or send a valid API key.

### `missing_key`

HTTP 401. Retryable: no.

* When: No `Authorization` or `X-API-Key` header, or an empty bearer token (the variable holding the key is unset).
* What to do: Send the key as `X-API-Key: <key>` or `Authorization: Bearer <key>`; create one under **API keys** in the dashboard or with `instabox keys create`. Details: [authentication errors](https://instabox.sh/docs/api-reference/introduction.md#authentication-errors).

### `malformed_key`

HTTP 401. Retryable: no.

* When: Not an API key (`ibk_` and 52 characters) or access token, or `Authorization` without `Bearer`.
* What to do: Send the whole key, unchanged, as `X-API-Key: <key>` or `Authorization: Bearer <key>`.

### `invalid_key`

HTTP 401. Retryable: no.

* When: A well-formed API key or access token that does not exist.
* What to do: Check you copied the right key for this server, or create a new one.

### `expired_key`

HTTP 401. Retryable: no.

* When: The API key or access token expired.
* What to do: Create a new key (`instabox keys create --name N`), or sign in again.

### `revoked_key`

HTTP 401. Retryable: no.

* When: The API key was revoked, or the session signed out (or its user was disabled).
* What to do: Create a new key, or sign in again.

### `session_expired`

HTTP 401. Retryable: no.

* When: The dashboard's session cookie is no longer valid.
* What to do: Sign in again.

### `api_key_expired`

HTTP 401. Retryable: no.

* When: Older servers: the same as `expired_key`.
* What to do: Create a new key.

### `api_key_revoked`

HTTP 401. Retryable: no.

* When: Older servers: the same as `revoked_key`.
* What to do: Create a new key.

## 402 Payment Required

### `credit_exhausted`

HTTP 402. Retryable: no.

* When: The account's free machine used its hours for the month, so it cannot be created, started or resumed. Or a workspace paying a plan from credits (`--credits`) has too few credits for the rest of this month's plan, at subscribe or at a plan change (the message gives the balance and what is needed).
* What to do: Free hours: wait for the next month (UTC), or upgrade the workspace: [Billing](https://dashboard.instabox.sh/dashboard/billing/plans) in the dashboard, or `instabox workspace subscribe <workspace> --plan pro`. Too few credits: subscribe or change the plan with a card instead of credits ([Billing](https://dashboard.instabox.sh/dashboard/billing/plans), or `instabox workspace subscribe <workspace> --plan pro` without `--credits`), or choose a cheaper plan.

### `spend_limit_reached`

HTTP 402. Retryable: no.

* When: The paid workspace's usage this month reached its spend cap (or its allowance while a payment is past due); its machines cannot run.
* What to do: Raise or remove the cap on [Billing](https://dashboard.instabox.sh/dashboard/billing) in the dashboard or with `instabox workspace spend-cap <workspace> <usd>|off`, or pay the open invoice (`instabox workspace billing <workspace>` prints the billing portal).

### `subscription_required`

HTTP 402. Retryable: no.

* When: The workspace's subscription ended; it cannot be used until it subscribes again (its data is kept for 30 days).
* What to do: Subscribe again: [Billing](https://dashboard.instabox.sh/dashboard/billing/plans) in the dashboard, or `instabox workspace subscribe <workspace> --plan pro`.

### `payment_method_required`

HTTP 402. Retryable: no.

* When: A free workspace has no card on file, so its machines cannot run.
* What to do: Add a card: [Billing](https://dashboard.instabox.sh/dashboard/billing) in the dashboard, or `instabox workspace card <workspace>`.

### `early_allowance_ended`

HTTP 402. Retryable: no.

* When: The early free-sandbox allowance ended on 2026-12-01, and the org already runs as many free sandboxes as the standard limit allows.
* What to do: Pause a running sandbox (`instabox suspend <sandbox>`), delete some, or upgrade: [Billing](https://dashboard.instabox.sh/dashboard/billing/plans) in the dashboard, or `instabox workspace subscribe <workspace> --plan pro`.

### `payment_failed`

HTTP 402. Retryable: no.

* When: The charge a plan change needed did not go through, so the plan is unchanged. Or the workspace's payment failed, so its plan is paused: its sandboxes cannot run, and it cannot create sandboxes, snapshots or bigger sandboxes, until it is paid.
* What to do: Update the card or pay the open invoice on [Billing](https://dashboard.instabox.sh/dashboard/billing) in the dashboard (or the billing portal, `instabox workspace billing <workspace>`), then try again.

## 403 Forbidden

### `missing_scope`

HTTP 403. Retryable: no.

* When: The API key lacks the scope the route needs; `requiredScopes` lists the scopes it accepts.
* What to do: Use a key with one of `requiredScopes`, or create one (keys cannot gain scopes). Some routes take no API key at all (see `permission_denied`).

### `permission_denied`

HTTP 403. Retryable: no.

* When: You may not do this: not the owner or an admin of the workspace, not on the allowlist, or an API key on a route only a signed-in session may call (managing keys, for one).
* What to do: Ask a workspace owner or admin, or do it signed in (the dashboard, or the CLI after `instabox login`).

### `invite_required`

HTTP 403. Retryable: no.

* When: The account must redeem an invite code before it can use the API.
* What to do: Redeem a code (`POST /v1/invites/redeem`, or the dashboard) and send the request again.

### `invite_email_mismatch`

HTTP 403. Retryable: no.

* When: A workspace invite was emailed to another address than the account signing in.
* What to do: Sign in with the invited address, or ask for an invite to yours.

### `insufficient_scope`

HTTP 403. Retryable: no.

* When: Older servers: the same as `missing_scope`.
* What to do: Use a key with the scope the route needs.

### `template_not_allowed`

HTTP 403. Retryable: no.

* When: The workspace's plan creates machines from other templates (free: `agent-browser`, `android`, `omarchy` or `ci`), or the snapshot holds a template this account may not use. macOS machines need the Pro plan or higher.
* What to do: Create from one of the templates the message names, from a snapshot, or upgrade ([Pricing](https://instabox.sh/docs/account/pricing.md)).

### `template_limit`

HTTP 403. Retryable: no.

* When: You already have as many machines of the template as it allows per account, or an internal-test template has reached its total.
* What to do: Delete one of your machines from it, or use another template.

## 404 Not Found

### `not_found`

HTTP 404. Retryable: no.

* When: No such machine, snapshot or other resource (deleted ones included), or it belongs to another workspace.
* What to do: Check the ID or name, and that the key belongs to the workspace that owns it (`instabox workspace ls`).

### `missing_parent`

HTTP 404. Retryable: no.

* When: A file write's directory does not exist.
* What to do: Create the directory first, or send `parents=true`.

## 409 Conflict

### `operation_conflict`

HTTP 409. Retryable: sometimes: read `retryable`.

* When: Another operation is running on the machine (`operationId` names it), or the change is not allowed in the machine's state.
* What to do: Wait for that operation (`GET /v1/operations/{id}` with that `operationId`), then try again with a new `Idempotency-Key`; or bring the machine to the state the change needs (stop it first, for example).

### `idempotency_conflict`

HTTP 409. Retryable: no.

* When: The `Idempotency-Key` was already used for a different request (keys are kept 7 days).
* What to do: Use a new key for a new request; reuse a key only to retry the exact same request.

### `machine_not_running`

HTTP 409. Retryable: sometimes: read `retryable`.

* When: The machine is not running (stopped or paused) and the request needs it running. A terminal reconnect that must not wake the machine (`wake=0`) gets it, retryable, while the machine is not running or an operation is under way.
* What to do: Start or resume it (`instabox start`, `instabox resume`), then send the request again. When `retryable` is true (the terminal reconnect), retry once the machine runs again.

### `restore_unavailable`

HTTP 409. Retryable: no.

* When: `POST /v1/machines/{id}/restore`: the machine's host cannot restore in place yet; nothing changed.
* What to do: Create a new machine from the snapshot instead (`instabox create --snapshot N`).

### `delete_confirmation_required`

HTTP 409. Retryable: no.

* When: A delete's `X-Confirm-Delete` header is neither the machine's (or snapshot's) ID nor its name, or a machine delete with an API key created from the rollout on left the header out; nothing was deleted.
* What to do: Send `X-Confirm-Delete` with the ID or exact name the message names (the same `Idempotency-Key` works).

### `fork_unavailable`

HTTP 409. Retryable: sometimes: read `retryable`.

* When: The machine cannot be forked on its host; nothing was created. Not retryable when it is paused, has no host, or its host cannot copy disks. A fork is made only on the source's host, so when that host is under maintenance or full the error is retryable and `reason` is `source_host_unavailable` (maintenance) or `source_host_full` (no room).
* What to do: Save a snapshot and create from it (the CLI does this for you), or resume the machine and fork it. For a host under maintenance or full: stop and start the machine to move it to another host, then fork it, or wait `retryAfterSeconds`.

### `machine_cooldown`

HTTP 409. Retryable: yes.

* When: The machine rests until `cooldownUntil`; every way of waking it is refused until then. Either it reached its maximum runtime (`maxRunMinutes`), or an automatic abuse check paused it and locked it for 24 hours: then `reason` names the check (`abuse_egress_blocklist`, `abuse_network_flows`, `abuse_cpu` or `abuse_proxy`) and the message says when the lock ends. The machine's data is kept.
* What to do: Wait `retryAfterSeconds` (also the `Retry-After` header), then try again. If an abuse check locked it by mistake, request a review with `instabox feedback` or by email to the address the message names.

### `max_run_disabled`

HTTP 409 or 400. Retryable: no.

* When: 409: waking a machine whose `maxRunMinutes` is 0. 400: a resize with `apply: "restart"` that leaves `maxRunMinutes` at 0.
* What to do: Set `maxRunMinutes` to another value (`instabox max-run`), then wake it; for the resize, resize first and set it after.

### `desktop_not_running`

HTTP 409. Retryable: no.

* When: The machine's desktop (VNC) was asked for before it started.
* What to do: Start the desktop (`POST /v1/machines/{id}/desktop`), then ask again.

### `disk_too_small`

HTTP 409. Retryable: no.

* When: A `/data` shrink to a size its data does not fit (data plus 10%, at least 1 GiB); nothing changed.
* What to do: Pick a size at least what the message names, or delete data first.

### `grow_needs_stop`

HTTP 409. Retryable: no.

* When: A running machine's disk cannot grow online right now (unreachable, or an older host).
* What to do: Stop the machine and send the size again; it grows at the next start (`instabox resize --disk N --restart`).

### `orca_unsupported`

HTTP 409. Retryable: no.

* When: The machine's template has no Orca runtime.
* What to do: Create the machine from the `orca` template.

### `runtime_not_ready`

HTTP 409. Retryable: yes.

* When: The Orca runtime is not reachable yet.
* What to do: Wait a few seconds and try again.

### `not_a_file`

HTTP 409. Retryable: no.

* When: A file write or download names a directory or something that is not a regular file.
* What to do: Name a file; list a directory with `GET /v1/machines/{id}/files` (`instabox files`).

## 413 Content Too Large

### `too_large`

HTTP 413. Retryable: no.

* When: A file upload is over 5 GiB.
* What to do: Split the file, or fetch it from inside the machine (`curl`, `git`).

## 429 Too Many Requests

### `quota_exceeded`

HTTP 429. Retryable: sometimes: read `retryable`.

* When: A limit was reached: machines, snapshots, stored data, API keys, concurrent `exec` requests, terminals, or agent sign-ups. The message says which ([Limits](https://instabox.sh/docs/account/limits.md)).
* What to do: When `retryable` is true (`exec` concurrency, agent sign-up), wait and retry, after `retryAfterSeconds` when present. Otherwise delete a machine, snapshot or key first, or upgrade ([Pricing](https://instabox.sh/docs/account/pricing.md)).

### `rate_limited`

HTTP 429. Retryable: yes.

* When: The workspace made its plan's machine starts (create, start, resume, wake) for the last minute. A start that failed on our side (`no_capacity`, `host_unavailable`, `internal`) is not counted.
* What to do: Wait `retryAfterSeconds` (also the `Retry-After` header), then try again.

### `concurrency_limit`

HTTP 429. Retryable: sometimes: read `retryable`.

* When: The workspace's plan allows no more machines running at once; or too many `exec` requests at once (retryable).
* What to do: Stop or pause a running machine first (`instabox stop`, `instabox suspend`), or upgrade; for `exec`, wait and retry.

## 500 Internal Server Error

### `restore_failed`

HTTP 500. Retryable: no.

* When: A resume (`instabox resume`, or a request that wakes the sandbox) could not restore its suspended memory on its host. Its processes are lost; the sandbox is stopped with its files kept. The operation's error and the sandbox's `lastError` have this code.
* What to do: Start the sandbox (`instabox start`): it boots from its disks. If it keeps happening, contact support.

### `internal`

HTTP 500 or 502 or 503. Retryable: sometimes: read `retryable`.

* When: 500: an unexpected server error. 502: a service the request needed (sign-in, GitHub, email) failed. 503: a service this server needs is not configured or not ready (billing, the GitHub App, stats).
* What to do: When `retryable` is true, retry with backoff. Otherwise the message says what is missing, and retrying will not help. If a 500 keeps happening, report it with the `requestId` (`instabox feedback`).

## 503 Service Unavailable

### `private_ports_unavailable`

HTTP 503. Retryable: yes.

* When: A private port was asked for while the public-port proxy still runs a release without them.
* What to do: Try again in a few minutes, once the rollout is done.

### `no_capacity`

HTTP 503 or 409. Retryable: sometimes: read `retryable`.

* When: No host has room for the machine right now (as a 503, or a failed operation); the message says what could not run and when to retry. 409: no other host could take the machines an operator was moving off a host.
* What to do: When `retryable` is true, wait `retryAfterSeconds` (60), then retry with backoff, or try another size or template. A create whose operation failed replays it for the same `Idempotency-Key`, so retry with a new key. When it is false (an account assigned to a host pool this server does not have), retrying will not help: contact support.

### `host_unavailable`

HTTP 503 or 502. Retryable: sometimes: read `retryable`.

* When: 503: the machine's host cannot be reached right now; while it does not answer, the machine shows `runtime.hostUnreachable` and a `start` or `resume` of it while running is refused with this code. 502: the machine's guest, or an access backend, did not answer (the process list, for one).
* What to do: When `retryable` is true, retry after a few seconds. An `exec` that already started is not retryable, since the command may have run: check before running it again.

### `metrics_unavailable`

HTTP 503. Retryable: sometimes: read `retryable`.

* When: Admin metrics routes: the metrics store is off, down or failing.
* What to do: Retry when `retryable` is true; otherwise the store is turned off on this server.

### `logs_unavailable`

HTTP 503. Retryable: sometimes: read `retryable`.

* When: Admin logs routes: the logs store is off, down or failing.
* What to do: Retry when `retryable` is true; otherwise the store is turned off on this server.

### `kernel_unavailable`

HTTP 503. Retryable: yes.

* When: The template needs a guest kernel no usable host has yet.
* What to do: Try again later, or create from another template.

### `template_unavailable`

HTTP 503. Retryable: yes.

* When: No usable host holds any version of the template right now, typically during a template rollout before a host has the new image. Nothing was created.
* What to do: Try again in a few minutes; `GET /v1/templates` lists the versions you can create from.

### `vault_unavailable`

HTTP 503. Retryable: no.

* When: Secret encryption or delivery is unavailable: an item was sealed under a key this server does not have.
* What to do: Report it with the `requestId` (`instabox feedback`).

### `support_unavailable`

HTTP 503 or 504. Retryable: sometimes: read `retryable`.

* When: This server relays no feedback or tickets, or the support service did not answer (504 when it did not confirm in time and may have recorded it).
* What to do: Try again later; after a 504, check your tickets before sending it again.

## 507 Insufficient Storage

### `no_space`

HTTP 507. Retryable: no.

* When: The machine's disk filled during a file write.
* What to do: Free space in the machine, or grow `/data` (`instabox resize --disk N`), then write again.

## Reserved

### `vault_retired`

HTTP none. Retryable: no.

* When: Reserved: not sent. Secret writes and delivery remain supported.
* What to do: Nothing to do.

### `host_oom`

HTTP none. Retryable: no.

* When: Reserved: not sent as a response. A stopped sandbox's `lastError` has this code, with `at` (when), when the host ran out of memory and stopped the sandbox.
* What to do: Start the sandbox again. If it keeps happening, use a smaller size or contact support.

### `vmm_crashed`

HTTP none. Retryable: no.

* When: Reserved: not sent as a response. A stopped sandbox's `lastError` has this code, with `at` (when), when the sandbox's virtual machine process exited on its own and the guest had not shut down.
* What to do: Start the sandbox again. If it keeps happening, contact support.

### `guest_reboot_loop`

HTTP none. Retryable: no.

* When: Reserved: not sent as a response. A stopped sandbox's `lastError` has this code, with `at` (when), when the sandbox kept shutting down or restarting soon after it booted and was left stopped.
* What to do: Check the services that run at boot, then start the sandbox again.
