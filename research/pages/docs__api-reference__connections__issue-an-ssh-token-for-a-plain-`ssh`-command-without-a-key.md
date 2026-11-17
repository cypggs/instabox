> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Issue an SSH token, for a plain `ssh` command without a key.

> Returns a token to use as the SSH user name:
`command` is `ssh -p <port> <token>@<host>`. The server keeps its
sha256 and, when a vault key is configured, the token sealed, which
`GET /v1/machines/{id}/ssh-tokens` returns to its maker in a sign-in
session. This request takes no `Idempotency-Key`; a retry makes another
token. The sandbox may be in
any state but `deleting`: while it is not running, a client that logs
in is told to start it. The token works until `expiresAt` (absent with
`noExpiry`: never), until
`DELETE /v1/machines/{id}/ssh-tokens`, until the sandbox is deleted or,
for a token an API key made, until the key is revoked or expires.
API keys: needs one of the scopes `machines:write`, `machines:ssh` (legacy).




## OpenAPI

````yaml /api-reference/openapi.yaml post /v1/machines/{id}/ssh-tokens
openapi: 3.1.0
info:
  title: instabox API
  version: 1.0.0
  summary: Create and manage cloud sandboxes for AI coding agents.
  description: >
    The instabox API creates and manages sandboxes: Linux computers in the

    cloud that keep their files and software, for coding agents such as

    Claude Code, Codex and OpenCode. It also issues short-lived SSH

    credentials for reaching them. The `instabox` CLI, the dashboard, herdr,

    cmux and Orca all use it. The Go types in `pkg/instaboxapi` are the

    source of truth; this document mirrors them.


    The API is older than the name "sandbox". Paths, fields and scopes call a

    sandbox a machine (`/v1/machines`, `machineId`, `machines:read`), and

    pausing is `suspend` and `resume`. The text here says "sandbox" and

    "pause"; code keeps the API's names.


    ## Authentication


    The base URL is `https://api.instabox.sh`. Programs and agents send an API

    key: `X-API-Key: ibk_...` or `Authorization: Bearer ibk_...`, for example

    `curl -H "X-API-Key: $INSTABOX_API_KEY"
    https://api.instabox.sh/v1/machines`.

    Create one in the dashboard (API keys) or with `instabox keys create`; see

    API keys below for scopes. A bad key gets a 401 whose `error.code` says

    why: `missing_key`, `malformed_key`, `invalid_key`, `expired_key`,

    `revoked_key`, or `session_expired` for a dashboard session, with a `hint`

    and `docs` (https://instabox.sh/docs/api#authentication-errors). A key

    without the scope a route needs gets `403 missing_scope`. Docs for agents:

    https://instabox.sh/llms.txt.


    ## Sandboxes


    - A new sandbox gets the default size, 4 vCPUs and 8 GiB of memory
      (`limits.machineSpec` in `GET /v1/usage`), unless `vcpus` and `memMiB`
      name another size of your plan (Pro: 2/4, 4/8, 8/16; Builder: 2/4 up to 16/32; Team and Enterprise: 2/4 up to 32/64) or one an admin granted the org; `GET /v1/me` `allowedSpecs`
      lists the personal org's, `GET /v1/orgs` each org's. Leave `persistentDiskGiB` and `rootDiskGiB` out at create:
      the disks come with the size, and a disk size other than the one the sandbox gets is `400 spec_not_allowed`.
      A paid sandbox's disk grows afterwards with `PATCH /v1/machines/{id}`
      `{"persistentDiskGiB": N}`: 1-1024 GiB; a running sandbox's disk grows live, a shrink
      applies while it is stopped, at its next boot (refused as `disk_too_small` when the data
      no longer fits), up to the plan's `maxDiskGiB` and the stored-data limit.
      Its vCPUs and memory move to another of the plan's sizes the same way
      (`{"vcpus": N, "memMiB": M}`, while it is stopped, applied at its next boot);
      a free sandbox's size is fixed.
    - A sandbox has one **disk** (`/data`, which holds HOME at `/data/home`), 20
    GiB
      by default, kept across stop, pause and start. The operating system and the
      template's software sit on a root filesystem the template sizes; only admins grow it
      (`rootDiskGiB`).
    - A free org (every personal org) can have 1 sandbox in any state
      (stopped and paused ones count), 5 snapshots and 200 GiB of stored data
      across sandboxes and snapshots; Pro 500 sandboxes (100 running),
      Builder 1000 sandboxes (200 running), Team no sandbox limit (500 running).
      `GET /v1/usage` shows the current
      figures. An admin can change an org's sandbox limit.

    ## Conventions


    - JSON bodies use camelCase. Times are RFC 3339 in UTC.

    - All endpoints except `/v1/auth/device`, `/v1/auth/device/token`,
      `/v1/auth/cli/token`, `/v1/auth/refresh` and `/v1/openapi.json` require `Authorization: Bearer <accessToken>`, or the
      dashboard's `instabox_session` cookie (see `sessionCookie`; mutations then need
      `X-Instabox-CSRF` and a same-origin `Origin`). Sandbox, snapshot and operation endpoints
      also accept an API key (see API keys).
    - `/v1/admin/*` and `/metrics` (from outside `metrics.allowFrom`) are
      admin-only; other users get `403 permission_denied`.
    - Access: GitHub and Google sign-in is public. A verified provider identity
      creates a regular user. With the server's invite gate on (`inviteRequired`
      in server.json or `INSTABOX_INVITE_REQUIRED=true`), a new account must then
      redeem an invite code (`POST /v1/invites/redeem`) before it may use anything
      but `GET /v1/me` and `POST /v1/auth/logout`: every other route answers
      `403 invite_required` until it does. `GET /v1/me` says whether the gate is
      on (`inviteRequired`) and whether the account passed it (`activated`);
      admins and accounts from before the gate always pass. Disabled users cannot
      sign in, refresh credentials, call authenticated APIs or connect over SSH.
      GitHub users keep their positive numeric IDs; Google users get distinct
      negative IDs assigned by instabox. Matching emails never link accounts.
    - Every non-2xx response has the error envelope
      `{"error": {"code", "message", "requestId", "retryable", "operationId"}}`.
    - Changes to a sandbox are asynchronous: they return `202 Accepted` with an
      `Operation`. Poll `GET /v1/operations/{id}?wait=25s` (a long poll) until
      `state` is `succeeded` or `failed`. Accepted does not mean ready.

    ## API keys


    Programs, CI and agents use an API key (`ibk_...`) instead of signing in:

    send `X-API-Key: ibk_...` (or `Authorization: Bearer ibk_...`; when

    both are sent they must carry the same key, else `400 bad_request`). You

    create keys while signed in (`POST /v1/keys`, API keys in the dashboard

    sidebar, or `instabox keys create`); the secret is shown once and only its

    sha256 is stored. Every key belongs to one org and reaches no other: `POST
    /v1/keys`

    makes one of your personal org, `POST /v1/orgs/{id}/keys` one of any org you

    manage (see Workspace API keys). A key expires after `expiresInDays` (1-365)
    or, without it, never; an org has at

    most 20 active keys. An expired or revoked key gets `401 expired_key` or

    `401 revoked_key`. Disabling a user revokes their keys, and every request

    checks that the user is not disabled. SSH certificates a key

    issued stop working when the key is revoked or expires.


    A key carries scopes: read, write and delete for each resource. A route the

    key's scopes do not cover returns `403 missing_scope`. A `:write` or

    `:delete` scope includes the `:read` scope of the same resource. Signed-in

    sessions have every scope.


    - `machines:read`: `GET /v1/machines`, `GET /v1/machines/{id}`, `GET
    /v1/machines/{id}/connection-info`, `GET /v1/machines/{id}/metrics`, `GET
    /v1/machines/{id}/usage`, `GET /v1/operations/{id}`, `GET /v1/templates`,
    `GET /v1/usage`, `GET /v1/orgs/{id}/github/jobs`, `GET
    /v1/orgs/{id}/github/jobs/{job}`, `GET /v1/orgs/{id}/github/stats` (CI jobs)

    - `machines:write`: `POST /v1/machines`, `PATCH /v1/machines/{id}` (name,
    idle pause, metadata), `POST /v1/machines/{id}/start`, `/stop`, `/suspend`,
    `/resume`, `/restore` (disks back to a snapshot's, in place), `POST
    /v1/machines/{id}/connections` (SSH certificates), `POST`, `DELETE
    /v1/machines/{id}/ssh-tokens` (SSH tokens), `POST /v1/machines/{id}/exec`
    (run a command)

    - `machines:delete`: `DELETE /v1/machines/{id}`

    - `images:read`: `GET /v1/snapshots`, `GET /v1/snapshots/{id}`, `GET
    /v1/snapshots/{id}/files`, `GET /v1/templates`, `GET /v1/usage`, `GET
    /v1/operations/{id}`

    - `images:write`: `POST /v1/machines/{id}/snapshots`

    - `images:delete`: `DELETE /v1/snapshots/{id}`


    The snapshot scopes keep the names they had when snapshots were called

    images (`images:*`), because existing keys store them.


    Keys created before these scopes may still carry `machines:ssh` (SSH

    certificates and connection info) or `machines:exec` (exec). Those keep

    working on their routes, but new keys cannot be given them.


    Any key may call `GET /v1/me`. Keys never work on

    `/v1/keys` (a leaked key cannot create more keys), `/v1/sessions`,

    `/v1/auth/logout`, `/v1/admin/*` or `/metrics`, and never carry admin

    rights: `start` with `host` gets `403 permission_denied`, another user's

    snapshot is `404` as for any other user, and resizing gets

    `400 spec_not_allowed`.


    ## Workspace API keys


    A workspace's owner and admins make keys that belong to the

    workspace, not to them (`POST /v1/orgs/{id}/keys`, signed-in sessions

    only). Without `scopes`, a workspace key carries the `workspace:admin`
    scope: it is an

    admin of that one workspace on every key-capable route, including the

    workspace's management routes (rename, members, invites, invite code,

    Secrets), which other keys cannot use (`403 missing_scope`); with

    `APIKeyScope`s it is limited to them. It

    acts as the workspace's owner (`GET /v1/me` names the owner, without

    their email, and `orgId`) yet reaches nothing outside the workspace:

    other orgs, their sandboxes and snapshots are `404`, a create that names

    no `orgId` goes in the workspace, and the sandboxes it creates have no

    Vault (use org Secrets). Billing, the Vault routes and key management

    stay with sessions; desktop links are the machine creator's, signed in

    or with their personal key (for a machine of their personal workspace),

    so a workspace key gets `403 permission_denied` there. It keeps working when
    the member who made it leaves; it stops

    when it is revoked or expires, or the workspace is deleted or purged. A
    workspace

    has at most 20 active keys.


    ## Idempotency


    Mutating POST and DELETE requests require an `Idempotency-Key` header

    (1-200 printable ASCII characters). Keys are scoped to (user, key), a

    workspace API key's to its workspace:

    - Same key with the same method, path and body hash returns the original
      response (status and body) without performing the request again.
    - Same key with a different request returns `409 idempotency_conflict`.

    - Records are kept for 7 days; after that the key may be reused.

    Generate a fresh random key for each action and save it before sending,

    so a retry after a timeout or crash reuses it.


    ## Sandbox lifecycle


    - A sandbox runs at most one operation at a time. A different change
      while one is running returns `409 operation_conflict` with `operationId`.
      Repeating the same change returns the running operation.
    - A change that is already in effect (for example `start` on a running
      sandbox) returns an operation that has already `succeeded`. If the
      sandbox's host is not answering (`runtime.hostUnreachable`), `start` and
      `resume` of a running sandbox return `503 host_unavailable` (retryable)
      instead: nothing was done, and the sandbox cannot be reached until its
      host answers again.
    - Pause (`suspend`) keeps the sandbox's running programs, and resume brings
      them back. `start` on a paused sandbox resumes it. If its programs cannot
      be brought back, the sandbox boots fresh and `runtime.processesPreserved`
      is `false`. Files are kept either way.
    - Stop shuts the sandbox down; running programs end, files are kept.
      `stop` on a paused sandbox returns `409 operation_conflict` ("resume it
      first, then stop it, or delete it").
    - Creating an SSH certificate connection resumes a sandbox paused for
      idleness; other non-running sandboxes return `409 machine_not_running`.
    - Use requests may resume a sandbox paused for idleness. Manual pause/stop
      and archive require an explicit start or resume.
    - Sandboxes have no lifetime timeout: they pause when idle (below). An
      archive deadline reached while idle-paused requires an explicit resume
      afterwards.
    - Pause when idle: a sandbox with `autoSuspendAfterSeconds` is paused
      through an ordinary `suspend` operation (`trigger: idle`) once it has
      reported no activity for that long (docs/idle-suspend.md); activity is
      checked about once a minute. Every new sandbox has it, 30 minutes, unless
      its create request gives another time or 0 (never). The setting stays: a
      later resume or start keeps it. Every template accepts it: one with the
      `activity` capability ships probes for its agents; on the others, terminal
      and CPU activity alone count. Real work (exec, files, terminal traffic and preview requests)
      and task activity leases also prevent idle suspension. Exec, files,
      terminal and SSH certificate requests resume sandboxes paused for
      idleness before proceeding (`trigger: request`). Manual pause/stop and
      archive are never automatically undone. Preview links remain
      offline until resumed. Status reads and keepalives do not record activity.
    - `revision` increases with each accepted change. An operation's results
      are written only if the sandbox's revision still equals the operation's.
    - Deleted sandboxes are not listed; `GET /v1/machines/{id}` returns
      `404 not_found` for them.

    ## Templates and snapshots


    A sandbox starts from a template (`templateId`: software that ships with

    instabox) or from a snapshot (`snapshotId`). A snapshot is a saved copy of

    a sandbox's system files and storage; a sandbox created from it gets both.

    A system-files-only snapshot (`rootOnly`, no `dataSizeBytes`) gives the new

    sandbox empty storage. Memory and running programs are not saved. Saving

    a snapshot of a stopped sandbox needs it fully saved (`storage.synced`),

    and every snapshot needs a template with the `identity-reset` capability.

    HOME is `/data/home`, so credentials kept there (gh, Claude, Codex, git,

    SSH) are included in new snapshots. A snapshot belongs to its source

    sandbox's org: every member of the org uses it. New snapshot IDs start with
    `sn_`; older ones with `im_`,

    and stay valid.


    Where the server allows live snapshots, you can also snapshot a running

    sandbox: the request answers `202` with the snapshot in

    `status: pending`. The sandbox's files are frozen for well under a

    second while they are copied, then the copy uploads in the background

    and the snapshot becomes `ready`, or `failed` with `error`.

    Poll `GET /v1/snapshots/{id}`. Only a `ready` snapshot can be used. The

    copy is like the state after a sudden power cut: data that programs had

    not written to disk yet is not included.


    ## Deprecated aliases


    Until 2026-10 snapshots were called images. These names keep working

    and are marked `deprecated` here; replies to them carry a `Deprecation`

    header (RFC 9745) and, for the routes, a `Link` to the successor

    (`rel="successor-version"`):


    - `POST /v1/machines/{id}/images`, `GET /v1/images`, `GET /v1/images/{id}`,
      `GET /v1/images/{id}/files`, `DELETE /v1/images/{id}`: the
      `/snapshots` routes. They reply in the old shape: `GET /v1/images`
      returns `{"images": [...]}`, and each snapshot also has
      `official: false`, `owned: true` and `preinstalled: []`.
    - `imageId` in `POST /v1/machines`: `snapshotId` (both: they must be equal).

    - `Machine.image`, `Usage.images` and `UsageLimits.maxImages`: sent with
      the same values as `snapshot`, `snapshots` and `maxSnapshots`.

    Official images were removed: software ships in templates.


    ## SSH


    Connect with `ssh -p <port> <machineId>@<host>`. Clients

    authenticate with a user certificate from

    `POST /v1/machines/{id}/connections` (principal = sandbox ID) and trust the

    server with a known_hosts line

    `@cert-authority [<host>]:<port> <hostTrust.publicKey>`.


    Without a key or the instabox CLI, a plain ssh client logs in with an SSH

    token from `POST /v1/machines/{id}/ssh-tokens` as its user name:

    `ssh ibssh_…@<host>`, with `-p <port>` unless SSH is on port 22

    (the reply's `command`). The token is accepted with the `none` method,

    which OpenSSH tries first, and ssh asks once to confirm the server's host

    key (`hostKeyFingerprint`). A token is a secret: it works for anyone who

    has it until it expires, is revoked (`DELETE /v1/machines/{id}/ssh-tokens`)

    or the sandbox is deleted. Supported:

    sessions (pty, shell, exec, subsystem, env, signals) and local port

    forwarding. Not supported in v1: agent forwarding, X11, remote forwarding.


    mosh works through the same login: when a session runs `mosh-server new`,

    the server opens a UDP port that relays to the sandbox's mosh-server and

    puts it in the `MOSH CONNECT` line. The relay outlives the SSH session and

    closes after a day without traffic, when the sandbox stops or is deleted,

    or when the grant is revoked.

    Connections are closed when the sandbox stops, pauses or is deleted.


    ## Exec


    `POST /v1/machines/{id}/exec` runs one command without an SSH client: the

    server logs in to the sandbox the way SSH does (user `dev`, HOME

    `/data/home`), runs `bash -lc <cmd>` and returns its output and exit code

    in the response. With `Accept: text/event-stream` the output is streamed

    as the command writes it instead. Use SSH for interactive work.
servers:
  - url: https://api.instabox.sh
security:
  - bearerAuth: []
  - sessionCookie: []
tags:
  - name: auth
  - name: machines
  - name: operations
  - name: connections
  - name: orca
  - name: snapshots
  - name: usage
  - name: orgs
  - name: sessions
  - name: keys
  - name: secrets
  - name: webhooks
  - name: exports
  - name: vault
  - name: support
  - name: docs
paths:
  /v1/machines/{id}/ssh-tokens:
    parameters:
      - $ref: '#/components/parameters/MachineID'
    post:
      tags:
        - connections
      summary: Issue an SSH token, for a plain `ssh` command without a key.
      description: >
        Returns a token to use as the SSH user name:

        `command` is `ssh -p <port> <token>@<host>`. The server keeps its

        sha256 and, when a vault key is configured, the token sealed, which

        `GET /v1/machines/{id}/ssh-tokens` returns to its maker in a sign-in

        session. This request takes no `Idempotency-Key`; a retry makes another

        token. The sandbox may be in

        any state but `deleting`: while it is not running, a client that logs

        in is told to start it. The token works until `expiresAt` (absent with

        `noExpiry`: never), until

        `DELETE /v1/machines/{id}/ssh-tokens`, until the sandbox is deleted or,

        for a token an API key made, until the key is revoked or expires.

        API keys: needs one of the scopes `machines:write`, `machines:ssh`
        (legacy).
      operationId: createSSHToken
      requestBody:
        required: false
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateSSHTokenRequest'
      responses:
        '201':
          description: Token issued.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SSHToken'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthenticated'
        '403':
          $ref: '#/components/responses/PermissionDenied'
        '404':
          $ref: '#/components/responses/NotFound'
        '409':
          $ref: '#/components/responses/Conflict'
      security:
        - bearerAuth: []
        - apiKey: []
        - sessionCookie: []
components:
  parameters:
    MachineID:
      name: id
      in: path
      required: true
      description: Sandbox ID (`m_` + 26 lowercase base32 characters).
      schema:
        type: string
        pattern: ^m_[a-z2-7]{26}$
  schemas:
    CreateSSHTokenRequest:
      type: object
      properties:
        name:
          type: string
          description: >-
            Optional display name; whitespace is trimmed, at most 100 UTF-8
            bytes. Existing callers may omit it.
        ttlSeconds:
          type: integer
          minimum: 1
          maximum: 2592000
          description: Default 604800 (7 days), max 2592000 (30 days).
        noExpiry:
          type: boolean
          description: >-
            A token that works until it is revoked: the reply has no
            `expiresAt`. Does not go with `ttlSeconds`.
    SSHToken:
      type: object
      required:
        - id
        - name
        - machineId
        - host
        - port
        - token
        - command
        - hostKeyFingerprint
      properties:
        name:
          type: string
        id:
          type: string
          description: Connection ID (`cn_…`) of the token's grant.
        machineId:
          type: string
        host:
          type: string
        port:
          type: integer
        token:
          type: string
          description: >-
            `ibssh_` and 26 lowercase base32 characters (128 bits): the SSH user
            name. Shown only here.
        command:
          type: string
          description: '`ssh -p <port> <token>@<host>` (without `-p` on port 22).'
        moshCommand:
          type: string
          description: >-
            The same login with mosh: `mosh --ssh="ssh -p <port>"
            <token>@<host>` (`mosh <token>@<host>` on port 22). Absent when mosh
            is not available.
        expiresAt:
          type: string
          format: date-time
          description: 'Absent: the token does not expire (`noExpiry`).'
        hostKeyFingerprint:
          type: string
          description: >-
            SHA256 fingerprint of the SSH server's host key, which ssh asks to
            confirm on the first connection.
    ErrorBody:
      type: object
      required:
        - error
      properties:
        error:
          $ref: '#/components/schemas/Error'
    Error:
      type: object
      required:
        - code
        - message
        - requestId
        - retryable
        - operationId
      properties:
        code:
          type: string
          description: Machine-readable code. HTTP status per code is fixed.
          enum:
            - unauthenticated
            - missing_key
            - malformed_key
            - invalid_key
            - expired_key
            - revoked_key
            - session_expired
            - missing_scope
            - permission_denied
            - not_found
            - bad_request
            - feature_disabled
            - unknown_field
            - operation_conflict
            - idempotency_conflict
            - machine_not_running
            - restore_unavailable
            - delete_confirmation_required
            - fork_unavailable
            - machine_cooldown
            - max_run_disabled
            - desktop_not_running
            - disk_too_small
            - grow_needs_stop
            - resize_restart_off
            - private_ports_unavailable
            - quota_exceeded
            - no_capacity
            - host_unavailable
            - metrics_unavailable
            - logs_unavailable
            - restore_failed
            - internal
            - authorization_pending
            - slow_down
            - expired_token
            - access_denied
            - invalid_grant
            - invite_required
            - invite_invalid
            - invite_email_mismatch
            - api_key_expired
            - api_key_revoked
            - insufficient_scope
            - orca_unsupported
            - runtime_not_ready
            - spec_not_allowed
            - template_not_allowed
            - template_limit
            - kernel_unavailable
            - template_unavailable
            - invalid_guest_user
            - vault_unavailable
            - vault_retired
            - credit_exhausted
            - spend_limit_reached
            - minimum_lease_required
            - subscription_required
            - payment_method_required
            - early_allowance_ended
            - rate_limited
            - concurrency_limit
            - payment_failed
            - missing_parent
            - not_a_file
            - no_space
            - too_large
            - data_disk_not_browsable
            - support_unavailable
            - host_oom
            - vmm_crashed
            - guest_reboot_loop
        message:
          type: string
        requestId:
          type: string
        retryable:
          type: boolean
          description: True if the same request may succeed later.
        operationId:
          type:
            - string
            - 'null'
          description: For `operation_conflict`, the active operation.
        reason:
          type: string
          description: >-
            The same as `code` for credential errors (401 `missing_key`,
            `malformed_key`, `invalid_key`, `expired_key`, `revoked_key`,
            `session_expired`; 403 `missing_scope`). For 409 `machine_cooldown`,
            the automatic abuse check whose pause locks the sandbox:
            `abuse_egress_blocklist` (it tried to connect to a blocked
            destination on the egress blocklist, not a DNS lookup),
            `abuse_network_flows` (scanning or attack traffic), `abuse_cpu`
            (mining); absent for a max-run cooldown. For a retryable 409
            `fork_unavailable`, why the source's host cannot take the fork:
            `source_host_unavailable` (it takes no new sandboxes) or
            `source_host_full` (no room now). Absent on other errors and from
            older servers.
          enum:
            - missing_key
            - malformed_key
            - invalid_key
            - expired_key
            - revoked_key
            - session_expired
            - missing_scope
            - abuse_egress_blocklist
            - abuse_network_flows
            - abuse_cpu
            - source_host_unavailable
            - source_host_full
        hint:
          type: string
          description: >-
            The next step that fixes the error (credential errors). Absent
            otherwise.
        docs:
          type: string
          format: uri
          description: A URL documenting the error (credential errors). Absent otherwise.
        docs_url:
          type: string
          format: uri
          description: >-
            The code's entry on the errors page, on every error response:
            https://instabox.sh/docs/errors#<code> with underscores as dashes
            (e.g. #credit-exhausted): its HTTP status, whether to retry and what
            to do. Snake case, unlike the other fields. Absent from a failed
            operation's `error` and from older servers.
        requiredScopes:
          type: array
          items:
            type: string
          description: >-
            For `missing_scope`, the scopes the route accepts; the key needs one
            of them.
        retryAfterSeconds:
          type: integer
          minimum: 1
          description: >-
            Seconds to wait before trying again: a retryable `no_capacity` (60,
            as a 503 or on a failed operation's `error`), `rate_limited`,
            `machine_cooldown` and agent sign-up `quota_exceeded`. An error
            response also sends it as the `Retry-After` header; a failed
            operation's `error` has only this field. Absent otherwise and from
            older servers.
        at:
          type: string
          format: date-time
          description: >-
            When it happened, for a stop the platform found (a sandbox's
            `lastError` with code `host_oom`, `vmm_crashed` or
            `guest_reboot_loop`; `hint` says what to do). The next successful
            start clears the error. Absent otherwise and from older servers.
  responses:
    BadRequest:
      description: >-
        `bad_request`: invalid input or missing Idempotency-Key.
        `unknown_field`: a body key that is no field of the request's schema
        (create, fork, restore, start, stop, suspend, resume, wake and `PATCH`
        of a sandbox); the message names it and the closest known field (`did
        you mean "templateId"?`) and lists the known ones. Nothing is changed.
        `spec_not_allowed`: sandbox sizes are fixed (4 vCPUs, 8 GiB memory by
        default; `limits.machineSpec`) and the request names another size
        (create, create from a snapshot), or someone other than an admin tries
        to grow the disk of a free sandbox or of a sandbox whose owner may not
        resize disks. On `PATCH /v1/machines/{id}` it is also a `vcpus`/`memMiB`
        size the workspace's plan does not offer (the message lists the plan's
        sizes) or any new size for a free sandbox, whose size is fixed. The
        message gives the size the sandbox gets; leaving the size fields out
        always works. `invalid_guest_user`: a create's `guestUser` is not a
        valid Linux user name or is reserved (`root`, `instabox`, the Debian
        base system's users and groups such as `sudo`, `systemd-*`).
        `vault_unavailable` (503): a Vault item was sealed under a master key
        this server does not have (docs/vault.md).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorBody'
    Unauthenticated:
      description: >-
        A credential 401 has its own code: `missing_key`, `malformed_key`,
        `invalid_key`, `expired_key`, `revoked_key` or `session_expired`, with
        `hint` and `docs` (older servers sent `unauthenticated`,
        `api_key_expired` or `api_key_revoked`). Other 401s (refresh tokens,
        tickets) are `unauthenticated`. For a missing or bad API key or access
        token the message ends with how to get a key (session and refresh-token
        401s say to sign in again instead), e.g. `create an API key at
        https://dashboard.instabox.sh/dashboard/api-keys and send it as
        "Authorization: Bearer <key>" or "X-API-Key: <key>"`.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorBody'
    PermissionDenied:
      description: >-
        `permission_denied`: the user is not allowed (not on the allowlist, or
        not the owner), or an API key was used where only a sign-in session is;
        `missing_scope`: the API key lacks the route's scope (`requiredScopes`
        names it; older servers sent `insufficient_scope`).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorBody'
    NotFound:
      description: '`not_found`: no such resource (also for deleted sandboxes).'
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorBody'
    Conflict:
      description: >
        `operation_conflict` (another operation is active; `operationId` names
        it,

        or the transition is not allowed), `idempotency_conflict` (key reused
        for

        a different request), `delete_confirmation_required` (a delete's

        `X-Confirm-Delete` header is not the sandbox's or snapshot's ID or name,

        or a key that must send it left it out),

        `machine_not_running` (the sandbox is not running),
        `desktop_not_running` (the

        VNC endpoint before the desktop started), or for Orca connections

        `orca_unsupported` and `runtime_not_ready` (retryable).
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorBody'
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      description: >-
        Opaque instabox access token (1 hour). Refresh with `/v1/auth/refresh`
        (refresh token valid 30 days, rotates on use).
    sessionCookie:
      type: apiKey
      in: cookie
      name: instabox_session
      description: >
        Dashboard browser session from `/auth/github/login` (7 days idle, 30
        days at

        most). Used only when no Authorization header is sent. Requests other
        than

        GET and HEAD must send `X-Instabox-CSRF` (from `GET /v1/me`) and an
        `Origin`

        equal to the server's `publicUrl`; otherwise `403 permission_denied`.
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
      description: >
        API key (`ibk_...`) with scopes; also accepted as `Authorization: Bearer
        ibk_...`.

        Accepted on sandbox, snapshot, operation, template and usage routes per
        scope

        (see API keys), never on key management, sessions, logout or admin
        routes.

````