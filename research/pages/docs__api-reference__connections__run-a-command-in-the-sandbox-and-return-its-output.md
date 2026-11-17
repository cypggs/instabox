> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Run a command in the sandbox and return its output.

> Runs `cmd` with `bash -lc` as the sandbox's login user (`dev`, HOME
`/data/home`, the user SSH sessions get), after exporting `envs` and
changing to `cwd` (default HOME, even if the profile changes directory;
a `cwd` that does not exist makes the command fail with exit code 1). `stdin` is fed to the command, then EOF:
UTF-8 text of at most 1 MiB (1048576 bytes, not characters); bytes that
are not UTF-8 become U+FFFD, so base64-encode binary input or write it
with `PUT /v1/machines/{id}/files` first (see `ExecRequest.stdin`).

Instead of `cmd`, send `argv` (a program and its arguments) to run
it with no shell: `{"argv": ["printf", "%s|", "a b", "$X"]}` prints
`a b|$X|`. The login profile, `envs` and `cwd` apply the same way;
only the parsing of the command differs. Set exactly one of `cmd`
and `argv`.

The request blocks until the command exits or `timeoutSeconds` passes.
On timeout the command's process group is killed (SIGKILL) and the
result has `timedOut: true` and `exitCode: 124` (also when the sandbox
did not start the command in time). A command killed by a signal has
`signal` set and `exitCode` 128+N for standard signal N, otherwise
255. A non-zero exit code is a `200` result,
not an error. The request returns when the command exits, even if
a process it started in the background keeps its stdout or stderr
open: that process keeps running, output it writes later than about
a second after the exit is not returned, and its later writes to
that output fail with a broken pipe, so redirect a server's stdout
and stderr to a file (`nohup server >log 2>&1 &`). stdout and stderr
keep their first 1 MiB each (`truncated: true` when either was cut);
invalid UTF-8 is replaced with U+FFFD (`invalidUtf8: true`), so binary
output is not byte-exact: read files with
`GET /v1/machines/{id}/files/content`.

The sandbox must be `running` (`409 machine_not_running`). At most 32
requests run at once per user on the free plan, per workspace for a
workspace's API keys (Pro 64; Builder, Team
and Enterprise 256), across all your sandboxes. Commands,
file reads, desktop, port-forward and Secret refresh calls share this
limit. A background command counts only while it starts. More get a
retryable `429 quota_exceeded` (server config `limits.maxExecsPerUser`,
`plans.<tier>.maxExecs` and `limits.maxExecsPerOrg`,
not reported by `GET /v1/usage`). If the sandbox cannot be reached,
`503 host_unavailable`; when the connection is lost after the command
started it is not `retryable`, since the command may have run (it is
killed, not left running). An API server that restarts mid-command
kills it and answers a `retryable` `503 host_unavailable` ("the API
server restarted before the command finished").
Not idempotent: no Idempotency-Key, and a retry runs the command again.
API keys: needs one of the scopes `machines:write`, `machines:exec` (legacy).

Streaming: send `Accept: text/event-stream` to get the output as the
command writes it, as server-sent events: a `stdout` or `stderr`
event (data: `ExecOutput`) for each chunk, a `: ping` comment every
15 s while the command is quiet, then one `exit` event (data: the
`ExecResult`, with empty `stdout` and `stderr` and never
`truncated`: streamed output is not capped) or `error` event (data:
an `ErrorBody`, e.g. `host_unavailable` when the connection to the
sandbox was lost). Errors found before the command starts (`400`,
`404`, `409`, `429`, a `503` dial failure) are plain JSON responses.
Closing the stream kills the command. Timeouts are the same as
without streaming; for a command longer than 10 minutes start it
in the background (`instabox exec --background`) and follow its log.

Keepalive: without streaming, a quiet request can be closed by a
proxy or NAT on the way (seen at about 270 s). Send
`X-Instabox-Exec-Keepalive: 1` and, once the command has run 15 s,
the server sends `200` and then a newline every 15 s; the body is
those newlines then the `ExecResult` (JSON decoders skip the
leading whitespace). An error after the `200` went out is the body
`{"error": Error, "status": N}`, N the status it would have had.
Errors before that keep their status. The CLI sends the header.




## OpenAPI

````yaml /api-reference/openapi.yaml post /v1/machines/{id}/exec
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
  /v1/machines/{id}/exec:
    parameters:
      - $ref: '#/components/parameters/MachineRef'
    post:
      tags:
        - connections
      summary: Run a command in the sandbox and return its output.
      description: >
        Runs `cmd` with `bash -lc` as the sandbox's login user (`dev`, HOME

        `/data/home`, the user SSH sessions get), after exporting `envs` and

        changing to `cwd` (default HOME, even if the profile changes directory;

        a `cwd` that does not exist makes the command fail with exit code 1).
        `stdin` is fed to the command, then EOF:

        UTF-8 text of at most 1 MiB (1048576 bytes, not characters); bytes that

        are not UTF-8 become U+FFFD, so base64-encode binary input or write it

        with `PUT /v1/machines/{id}/files` first (see `ExecRequest.stdin`).


        Instead of `cmd`, send `argv` (a program and its arguments) to run

        it with no shell: `{"argv": ["printf", "%s|", "a b", "$X"]}` prints

        `a b|$X|`. The login profile, `envs` and `cwd` apply the same way;

        only the parsing of the command differs. Set exactly one of `cmd`

        and `argv`.


        The request blocks until the command exits or `timeoutSeconds` passes.

        On timeout the command's process group is killed (SIGKILL) and the

        result has `timedOut: true` and `exitCode: 124` (also when the sandbox

        did not start the command in time). A command killed by a signal has

        `signal` set and `exitCode` 128+N for standard signal N, otherwise

        255. A non-zero exit code is a `200` result,

        not an error. The request returns when the command exits, even if

        a process it started in the background keeps its stdout or stderr

        open: that process keeps running, output it writes later than about

        a second after the exit is not returned, and its later writes to

        that output fail with a broken pipe, so redirect a server's stdout

        and stderr to a file (`nohup server >log 2>&1 &`). stdout and stderr

        keep their first 1 MiB each (`truncated: true` when either was cut);

        invalid UTF-8 is replaced with U+FFFD (`invalidUtf8: true`), so binary

        output is not byte-exact: read files with

        `GET /v1/machines/{id}/files/content`.


        The sandbox must be `running` (`409 machine_not_running`). At most 32

        requests run at once per user on the free plan, per workspace for a

        workspace's API keys (Pro 64; Builder, Team

        and Enterprise 256), across all your sandboxes. Commands,

        file reads, desktop, port-forward and Secret refresh calls share this

        limit. A background command counts only while it starts. More get a

        retryable `429 quota_exceeded` (server config `limits.maxExecsPerUser`,

        `plans.<tier>.maxExecs` and `limits.maxExecsPerOrg`,

        not reported by `GET /v1/usage`). If the sandbox cannot be reached,

        `503 host_unavailable`; when the connection is lost after the command

        started it is not `retryable`, since the command may have run (it is

        killed, not left running). An API server that restarts mid-command

        kills it and answers a `retryable` `503 host_unavailable` ("the API

        server restarted before the command finished").

        Not idempotent: no Idempotency-Key, and a retry runs the command again.

        API keys: needs one of the scopes `machines:write`, `machines:exec`
        (legacy).


        Streaming: send `Accept: text/event-stream` to get the output as the

        command writes it, as server-sent events: a `stdout` or `stderr`

        event (data: `ExecOutput`) for each chunk, a `: ping` comment every

        15 s while the command is quiet, then one `exit` event (data: the

        `ExecResult`, with empty `stdout` and `stderr` and never

        `truncated`: streamed output is not capped) or `error` event (data:

        an `ErrorBody`, e.g. `host_unavailable` when the connection to the

        sandbox was lost). Errors found before the command starts (`400`,

        `404`, `409`, `429`, a `503` dial failure) are plain JSON responses.

        Closing the stream kills the command. Timeouts are the same as

        without streaming; for a command longer than 10 minutes start it

        in the background (`instabox exec --background`) and follow its log.


        Keepalive: without streaming, a quiet request can be closed by a

        proxy or NAT on the way (seen at about 270 s). Send

        `X-Instabox-Exec-Keepalive: 1` and, once the command has run 15 s,

        the server sends `200` and then a newline every 15 s; the body is

        those newlines then the `ExecResult` (JSON decoders skip the

        leading whitespace). An error after the `200` went out is the body

        `{"error": Error, "status": N}`, N the status it would have had.

        Errors before that keep their status. The CLI sends the header.
      operationId: execCommand
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/ExecRequest'
      responses:
        '200':
          description: The command ran (whatever its exit code).
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ExecResult'
            text/event-stream:
              schema:
                type: string
                description: >-
                  Server-sent events: `stdout`/`stderr` (`ExecOutput`), then
                  `exit` (`ExecResult`) or `error` (`ErrorBody`).
                example: >+
                  event: stdout

                  data: {"data":"line 1\n"}


                  event: exit

                  data:
                  {"stdout":"","stderr":"","exitCode":0,"timedOut":false,"truncated":false}

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
        '429':
          $ref: '#/components/responses/QuotaExceeded'
        '500':
          $ref: '#/components/responses/Internal'
        '503':
          $ref: '#/components/responses/Unavailable'
      security:
        - bearerAuth: []
        - apiKey: []
        - sessionCookie: []
components:
  parameters:
    MachineRef:
      name: id
      in: path
      required: true
      description: >-
        Sandbox ID (`m_` + 26 lowercase base32 characters) or the sandbox's name
        (unique among your sandboxes; names never contain `_`, so the two never
        collide). Older servers take only the ID and answer `404` for a name.
      schema:
        type: string
        pattern: ^(m_[a-z2-7]{26}|[a-z0-9][a-z0-9-]{0,62})$
  schemas:
    ExecRequest:
      type: object
      description: >-
        Set exactly one of `cmd` (a shell string) and `argv` (a program and its
        arguments, run without a shell); both, neither or an empty `argv` is
        `400 bad_request`. An empty `cmd` counts as no `cmd`, so `{"cmd": "",
        "argv": [...]}` runs the `argv`.
      oneOf:
        - required:
            - cmd
          properties:
            cmd:
              minLength: 1
          not:
            required:
              - argv
        - required:
            - argv
          properties:
            cmd:
              maxLength: 0
      properties:
        cmd:
          type: string
          maxLength: 65536
          description: >-
            Shell command, run with `bash -lc`. At most 64 KiB, no NUL bytes.
            Set this or `argv`; an empty `cmd` is no `cmd`.
        argv:
          type: array
          minItems: 1
          items:
            type: string
          description: >-
            The program and its arguments, run with no shell: each element
            reaches the program exactly as sent (no expansion of `$`, `*` or
            `~`, no word splitting, quotes or redirections; `""` is an empty
            argument). `argv[0]` must not be empty and is looked up on the login
            profile's `PATH`; the login profile, `envs` and `cwd` apply as for
            `cmd`. A program that is not found exits 127, one that cannot run
            126. At most 64 KiB counting each element's length plus one, no NUL
            bytes. The command sent to the sandbox (`envs`, `cwd` and `argv`,
            quoted for the shell: about 9 bytes per element plus its text) must
            also stay under 128 KiB, so many short elements can be `400
            bad_request` below the 64 KiB cap; send bulk data on `stdin`. Set
            this or `cmd`.
        cwd:
          type: string
          maxLength: 4096
          description: Working directory (default HOME, `/data/home`).
        envs:
          type: object
          maxProperties: 256
          description: >-
            Environment variables exported before the command (after the login
            profile). Names match `^[A-Za-z_][A-Za-z0-9_]*$`.
          additionalProperties:
            type: string
        secretEnvs:
          type: object
          maxProperties: 64
          description: >-
            Direct (delivery env) Secrets of the sandbox's org exported for this
            command only: variable name -> Secret name. Resolved at exec time,
            never stored or written into the sandbox; each use is audited by
            name. A name may not also be in `envs`; reserved variables are
            refused. A proxy or missing Secret is 400. Only where `GET /v1/me`
            has `directSecretRefs`; elsewhere 400 (a control host from before
            secretEnvs would run the command without them).
          additionalProperties:
            $ref: '#/components/schemas/SecretName'
        timeoutSeconds:
          type: integer
          minimum: 1
          maximum: 600
          description: Default 60, max 600.
        stdin:
          type: string
          description: >-
            Text fed to the command, then EOF. It is a JSON string, so the
            command reads its UTF-8 bytes: NUL, CR and LF arrive unchanged
            (escaped as `\u0000`, `\r`, `\n`). At most 1 MiB (1048576 bytes) of
            UTF-8, counted in bytes, not characters (`é` is 2): more is `400
            bad_request` `stdin is longer than 1048576 bytes`. The schema sets
            no `maxLength` here, because `maxLength` counts characters. JSON
            cannot carry bytes that are not UTF-8; any in the request body reach
            the command as U+FFFD (`ef bf bd`), with no error. For binary input,
            send it base64-encoded and decode it in the command (`base64 -d |
            ...`; up to 768 KiB of binary fits), or write the file first with
            `PUT /v1/machines/{id}/files` and redirect it in (`cmd < file`). The
            CLI's `exec --stdin` does that upload itself for input past 1 MiB or
            not UTF-8, up to 5 GiB.
        noHistory:
          type: boolean
          description: >-
            Leave the command out of the sandbox's history (`GET
            /v1/machines/{id}/history`): for a client's own housekeeping
            commands, such as the CLI's job polls. Does not go with
            `historyCommand`.
        historyCommand:
          type: string
          description: >-
            How the command appears in the sandbox's history instead of `cmd` or
            `argv`, for a wrapper that runs something else (the CLI's `exec
            --background`). Cut at 4096 bytes like them; no NUL bytes.
    ExecResult:
      type: object
      required:
        - stdout
        - stderr
        - exitCode
        - timedOut
        - truncated
      properties:
        stdout:
          type: string
          description: First 1 MiB of standard output.
        stderr:
          type: string
          description: First 1 MiB of standard error.
        exitCode:
          type: integer
          description: >-
            Exit status; 128+N if killed by standard signal N, otherwise 255
            with `signal` set; 124 if timed out.
        signal:
          type: string
          description: >-
            The signal that killed the command, as the sandbox named it (absent
            otherwise).
        timedOut:
          type: boolean
          description: The command ran past `timeoutSeconds` and was killed.
        truncated:
          type: boolean
          description: stdout or stderr was longer than 1 MiB and was cut.
        diskWarning:
          $ref: '#/components/schemas/DiskWarning'
          description: >-
            Set when the command failed with No space left on device (read again
            right after it), or a filesystem was at least 90% full when last
            read (read in the background after an exec or metrics call, at most
            once a minute).
        invalidUtf8:
          type: boolean
          description: >-
            stdout or stderr had bytes that are not UTF-8 (binary output),
            replaced with U+FFFD, so the output is not what the command wrote.
            Absent when false; also set on a streamed exec's exit event.
        oomKilled:
          type: boolean
          description: >-
            The command exited non-zero and the sandbox's kernel OOM killer
            killed one of its own processes (the command or a child: its SSH
            session's cgroup; never another command's process, and each kill is
            reported once). `oomKill` says which, when the sandbox named it.
            Checked only for exit code 137, a signal, or stderr saying a process
            was killed. False when none was seen, the exec's session scope could
            not be read, or the sandbox's host or guest agent cannot report
            kills; a kill whose kernel log record was lost is still true when
            the session scope's oom_kill count shows it (`oomKill` absent).
            Absent from older servers. Also set on a streamed exec's exit event.
        oomKill:
          $ref: '#/components/schemas/OOMKill'
          description: The OOM kill behind `oomKilled` (the newest, when several).
    SecretName:
      type: string
      pattern: ^[A-Za-z_][A-Za-z0-9_]{0,127}$
      description: >-
        A unique environment variable name within an org. The names a sandbox's
        env reserves are reserved (shell variables such as PATH, HOME, BASH_ENV,
        ENV and IFS, session and trust-store variables, and INSTABOX_ and SSH_
        prefixes). A Secret saved under a name before it was reserved is not
        delivered: the machine's guest status names it.
    DiskWarning:
      type: object
      required:
        - mount
        - usedPct
        - level
        - message
      properties:
        mount:
          type: string
          description: >-
            The fullest filesystem (empty when it could not be read after a
            command ran out of space).
        usedPct:
          type: number
        level:
          type: string
          enum:
            - ok
            - high
            - critical
            - full
        noSpace:
          type: boolean
          description: >-
            The command failed with No space left on device (or Disk quota
            exceeded).
        message:
          type: string
          description: >-
            What is full and how to free or add space (delete files, `instabox
            resize <box> --disk N` for /data: a running sandbox's disk grows
            live).
    OOMKill:
      type: object
      description: A process the sandbox's kernel OOM killer killed.
      required:
        - at
        - pid
        - process
        - rssBytes
        - anonRssBytes
        - cgroup
      properties:
        at:
          type: string
          format: date-time
          description: When it was killed (to about a second).
        pid:
          type: integer
        process:
          type: string
          description: >-
            The killed process's name: its comm (at most 15 bytes from the
            kernel), capped at 64 bytes.
        uid:
          type: integer
          description: The killed process's user ID, when the kernel logged it.
        rssBytes:
          type: integer
          format: int64
          description: Its resident memory when it was killed (anonymous, file and shared).
        anonRssBytes:
          type: integer
          format: int64
          description: The anonymous part of rssBytes.
        cgroup:
          type: boolean
          description: >-
            A memory cgroup's limit killed it (a container, a unit's MemoryMax),
            not the sandbox running out of memory.
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
    QuotaExceeded:
      description: >-
        `quota_exceeded`: one of your limits was reached: the number of
        sandboxes (on the free plan 1, in any state, stopped and paused
        included; Pro 500; Builder 1000), stored data (on the free plan 200
        GiB), snapshots (on the free plan 5), API keys, execs or terminals. The
        message says which, e.g. "sandbox limit reached (1 of 1) in org acme
        (org_..., workspace) on the free plan: delete a sandbox first". Delete a
        sandbox or snapshot, upgrade the plan, or ask an admin to raise your
        limit.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorBody'
    Internal:
      description: '`internal`: unexpected server error.'
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/ErrorBody'
    Unavailable:
      description: >-
        `no_capacity` (there is no room for the sandbox right now; the message
        says what could not run and when to retry) or `host_unavailable` (the
        sandbox cannot be reached right now). Usually `retryable`; try again
        shortly. `no_capacity` is `retryable`: retry with exponential backoff
        (for example after 15 s, 30 s, 60 s, then every few minutes), or try
        another size or template. A create whose operation failed with it
        replays that failed operation for the same `Idempotency-Key`, so retry
        it with a new key. With `--fail-fast` the CLI's `create` and `run` exit
        75 on it (otherwise 1). A retryable `no_capacity` has
        `retryAfterSeconds` and the same `Retry-After` header: wait at least
        that long before the first retry.
      headers:
        Retry-After:
          description: Seconds to wait before retrying; the body's `retryAfterSeconds`.
          schema:
            type: integer
            minimum: 1
            examples:
              - 60
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