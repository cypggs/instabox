> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# Create a sandbox.

> Returns a `create` operation whose `machineId` is the new sandbox. Start from a template (`templateId`) or one of your snapshots (`snapshotId`). A sandbox gets the default size (4 vCPUs, 8 GiB memory) when the size fields are left out. For another size of your plan (Pro: 2/4, 4/8, 8/16; Builder: 2/4 up to 16/32; Team and Enterprise: 2/4 up to 32/64) or one an admin granted the org (`GET /v1/orgs` `allowedSpecs` lists them), name its `vcpus` and `memMiB`. Any other size is refused with `400 spec_not_allowed`. The free plan creates from agent-browser, android, omarchy or ci only (`403 template_not_allowed`); a create from a snapshot is not limited, except a snapshot of an internal-test template (`restricted` in `GET /v1/templates`), which only the accounts allowed to create from the template (and whoever took the snapshot) may use (`403 template_not_allowed`). An internal-test template has a limit on its machines (`403 template_limit`); a template whose guest kernel no host has is `503 kernel_unavailable`, and one with no version on any host (during a template rollout) is `503 template_unavailable`. Counts toward your plan's sandbox limit (`429 quota_exceeded`). When no host has room, the operation fails with `no_capacity` (retryable); with `failFast` the create is refused at once instead (`503 no_capacity`, nothing created, no operation). Retry `no_capacity` with backoff (see `Unavailable`) or try another size or template. A create from a template or from a system-files-only snapshot that fails before it gets a host (`no_capacity`, or any other error at that point) has its sandbox deleted with the failure (GET returns 404), so it does not count toward your limit and its name is free to retry. A create from a snapshot with storage stays, `stopped`; start it later. API keys: needs one of the scopes `machines:write`.



## OpenAPI

````yaml /api-reference/openapi.yaml post /v1/machines
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
  /v1/machines:
    post:
      tags:
        - machines
      summary: Create a sandbox.
      description: >-
        Returns a `create` operation whose `machineId` is the new sandbox. Start
        from a template (`templateId`) or one of your snapshots (`snapshotId`).
        A sandbox gets the default size (4 vCPUs, 8 GiB memory) when the size
        fields are left out. For another size of your plan (Pro: 2/4, 4/8, 8/16;
        Builder: 2/4 up to 16/32; Team and Enterprise: 2/4 up to 32/64) or one
        an admin granted the org (`GET /v1/orgs` `allowedSpecs` lists them),
        name its `vcpus` and `memMiB`. Any other size is refused with `400
        spec_not_allowed`. The free plan creates from agent-browser, android,
        omarchy or ci only (`403 template_not_allowed`); a create from a
        snapshot is not limited, except a snapshot of an internal-test template
        (`restricted` in `GET /v1/templates`), which only the accounts allowed
        to create from the template (and whoever took the snapshot) may use
        (`403 template_not_allowed`). An internal-test template has a limit on
        its machines (`403 template_limit`); a template whose guest kernel no
        host has is `503 kernel_unavailable`, and one with no version on any
        host (during a template rollout) is `503 template_unavailable`. Counts
        toward your plan's sandbox limit (`429 quota_exceeded`). When no host
        has room, the operation fails with `no_capacity` (retryable); with
        `failFast` the create is refused at once instead (`503 no_capacity`,
        nothing created, no operation). Retry `no_capacity` with backoff (see
        `Unavailable`) or try another size or template. A create from a template
        or from a system-files-only snapshot that fails before it gets a host
        (`no_capacity`, or any other error at that point) has its sandbox
        deleted with the failure (GET returns 404), so it does not count toward
        your limit and its name is free to retry. A create from a snapshot with
        storage stays, `stopped`; start it later. API keys: needs one of the
        scopes `machines:write`.
      operationId: createMachine
      parameters:
        - $ref: '#/components/parameters/IdempotencyKey'
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateMachineRequest'
      responses:
        '202':
          $ref: '#/components/responses/OperationAccepted'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthenticated'
        '402':
          $ref: '#/components/responses/PaymentRequired'
        '403':
          $ref: '#/components/responses/PermissionDenied'
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
    IdempotencyKey:
      name: Idempotency-Key
      in: header
      required: true
      description: |
        Client-generated key, 1-200 printable ASCII characters, scoped to the
        user. Same key and same method+path+body returns the original response;
        same key with a different request returns `409 idempotency_conflict`.
        Records are kept for 7 days.
      schema:
        type: string
        minLength: 1
        maxLength: 200
        pattern: ^[\x21-\x7e]+$
  schemas:
    CreateMachineRequest:
      type: object
      additionalProperties: false
      description: >-
        Every field is optional; a body key that is none of these is refused
        with `400 unknown_field` (keys match case-insensitively). The lifetime
        timeout retired on 2026-10-20 is refused too; `timeoutSeconds`,
        `onTimeout` or `timeout` with any value but null is `400 unknown_field`,
        and the message names `deleteAfterSeconds`, `maxRunMinutes` and
        `autoSuspendAfterSeconds` instead (to a `User-Agent` of
        `instabox-cli/<version>`, whose builds before that date sent them, it
        says the CLI is out of date, how to install the current one, and names
        the flags `--delete-after`, `--max-run-minutes` and `--auto-suspend`).
        At most one of `templateId` and `snapshotId`; with neither, the sandbox
        starts from agent-browser, the CLI's and dashboard's default. Leave the
        size fields out for the default size, or name the `vcpus` and `memMiB`
        of another of your sizes (the org's plan's or one an admin granted the
        org; `GET /v1/orgs` `allowedSpecs`).
      properties:
        name:
          type: string
          description: >-
            1-63 lowercase letters, digits or `-`, starting with a letter or
            digit; unique among the org's sandboxes. Absent or empty: `box-` and
            8 random hex digits (the operation's `machineId` and the sandbox
            carry it).
        templateId:
          type: string
          examples:
            - agent-browser
          description: >-
            The template to create from (`GET /v1/templates`, one with
            `creatable` true), optionally with a version as `id@version` (e.g.
            `agent-browser@2026-10-15.4`, the same as `templateVersion`). Absent
            (and no `snapshotId` or `templateVersion`): agent-browser. An
            unknown or retired one is refused with `400 bad_request` whose
            message lists the templates your plan can create from. The free plan
            creates from agent-browser, android, omarchy or ci only; another
            template is refused with `403 template_not_allowed`. A create from a
            snapshot is not limited.
        templateVersion:
          type: string
          description: >-
            Default: the version `GET /v1/templates` marks `latest`.
            `templateId` may name it instead, as `id@version` (e.g.
            `agent-browser@2026-10-15.4`); both at once must agree. A version
            the template does not have is `400 bad_request` whose message lists
            its versions, newest first, the `latest` (or `default`) one tagged.
        snapshotId:
          type: string
          description: >-
            Create from this snapshot of yours: its system files and storage
            (empty storage for a system-files-only snapshot).
        imageId:
          type: string
          deprecated: true
          description: >-
            Deprecated: `snapshotId` under its old name (both: they must be
            equal). The reply carries a `Deprecation` header.
        vcpus:
          type: integer
          minimum: 1
          description: >-
            Absent: the default size (4 vCPUs, 8 GiB memory;
            `limits.machineSpec`). Else one of the org's sizes (`GET /v1/orgs`
            `allowedSpecs`: the default, its plan's others and any size an admin
            granted the org; on api.instabox.sh free 4/8192 only, Pro 2/4096,
            4/8192, 8/16384, Builder 2/4096 up to 16/32768, Team and Enterprise
            2/4096 up to 32/65536), or, from a template with a `pool` (macOS,
            Windows), its `defaultSpec` or one of its `sizes`, whatever the
            plan; name memMiB too when two of them have this many vCPUs.
            Anything else is refused with `400 spec_not_allowed`; 0 or a
            negative (no size) with `400 bad_request`.
        memMiB:
          type: integer
          minimum: 1
          description: >-
            Like vcpus: the memory (MiB) of one of your sizes, such as 8192 with
            vcpus 4; 0 or a negative is `400 bad_request`.
        persistentDiskGiB:
          type: integer
          minimum: 1
          description: >-
            Storage (`/data`) size in GiB. The disks come with the size: storage
            gets the size's (20 GiB on api.instabox.sh), or a snapshot's storage
            size when that is larger. Absent or exactly that size: accepted; any
            other is `400 spec_not_allowed` whose message names the size's
            storage next to what was asked; 0 or a negative is `400
            bad_request`. To get more storage, grow it later with `PATCH
            /v1/machines/{id}`: a running sandbox's storage grows live, and a
            shrink applies while it is stopped, at its next boot. The SDKs'
            `diskGiB` (TypeScript) and `persistent_disk_gib` (Python) are this
            field; `diskGiB` in a REST body is `400 unknown_field`.
        rootDiskGiB:
          type: integer
          minimum: 1
          description: >-
            System files size in GiB. Like persistentDiskGiB: system files get
            the template's size (or the snapshot's); naming any other is `400
            spec_not_allowed` (the message names the size's root disk), 0 or a
            negative `400 bad_request`; grow it later with `PATCH
            /v1/machines/{id}`.
        desiredState:
          type: string
          enum:
            - running
            - stopped
          description: 'Default: running.'
        metadata:
          $ref: '#/components/schemas/Metadata'
        archiveSeconds:
          type: integer
          minimum: 60
          maximum: 86400
          description: >-
            Sets `archiveAt` to now plus this many seconds: the sandbox is
            stopped then if it is running. 0 or absent: none.
        deleteAfterSeconds:
          type: integer
          minimum: 60
          maximum: 86400
          description: >-
            Sets `deleteAt` to now plus this many seconds (60-86400, else 400):
            the sandbox is then deleted, whatever its state, so it cannot
            outlive a client that died (`instabox run` sets it). 0 or absent:
            never. A server without this field ignores it.
        maxRunMinutes:
          $ref: '#/components/schemas/MaxRunMinutes'
        autoSuspendAfterSeconds:
          $ref: '#/components/schemas/AutoSuspendAfterSeconds'
        vault:
          type: boolean
          description: >-
            False disables legacy Vault delivery. Org Secret selections are
            independent.
        secrets:
          $ref: '#/components/schemas/SecretSelection'
          description: >-
            Proxy Secrets from this org. All includes current and future proxy
            Secrets; [] selects none. Omit both secrets and envSecrets to retain
            only existing Vault delivery. When envSecrets alone is supplied,
            proxy selection defaults to all; directSecrets alone selects no
            proxy Secrets. Null is rejected.
        envSecrets:
          type: array
          items:
            $ref: '#/components/schemas/SecretName'
          default: []
          description: >-
            Explicit env Secret names from this org; omitted or [] selects none.
            No all mode; null is rejected. Real values enter the sandbox.
            Shorthand for directSecrets entries with env = the Secret's name;
            not sent together with directSecrets.
        directSecrets:
          type: array
          maxItems: 64
          items:
            $ref: '#/components/schemas/DirectSecret'
          description: >-
            Direct (delivery env) Secrets from this org, each as a variable
            and/or a home file. Real values enter the sandbox; prefer proxy
            Secrets for HTTPS API keys. A fork that sends none of secrets,
            envSecrets and directSecrets copies its source's selection.
        env:
          type: object
          maxProperties: 128
          description: >-
            Environment variables for the whole sandbox: every login shell
            exports them (exec, SSH, the terminal and what they start), from the
            first command on and again after each start and resume. Names match
            `^[A-Za-z_][A-Za-z0-9_]{0,127}$`; `BASH_ENV`, `ENV`, `HOME`,
            `HOSTNAME`, `IFS`, `LOGNAME`, `OLDPWD`, `PATH`, `PWD`, `SHELL`,
            `SHLVL`, `USER`, bash's readonly `BASHOPTS`, `BASH_VERSINFO`,
            `EUID`, `PPID`, `SHELLOPTS` and `UID`, the session's `TERM`,
            `XDG_RUNTIME_DIR` and `DBUS_SESSION_BUS_ADDRESS`, the trust roots
            `SSL_CERT_FILE`, `SSL_CERT_DIR`, `REQUESTS_CA_BUNDLE`,
            `CURL_CA_BUNDLE` and `NODE_EXTRA_CA_CERTS`, and names starting
            `INSTABOX_` or `SSH_` are reserved (set them per exec); values have
            no NUL bytes; names and values at most 32 KiB together. Not on
            Windows templates (400). On macOS, bash and zsh login shells export
            them, and zsh's special parameters (`status`, `path`, `options`,
            `HISTSIZE`, `USERNAME`, ...) and `ZDOTDIR` are reserved too. Fixed
            at create: a fork takes its source's, a create from a snapshot only
            its own. An exec's `envs`, a Vault item and an env Secret of the
            same name win. Not Secrets: anything in the sandbox can read the
            values, and the sandbox shows only the names (`envNames`). The
            values are stored encrypted at rest; a site without the Secret
            master key refuses env (400 bad_request).
          additionalProperties:
            type: string
        pool:
          type: string
          description: >-
            Admins only: the host pool to place the sandbox in and no other
            (docs/hybrid-pools.md), "default" included. With snapshotId it must
            be the snapshot's pool. Absent: the user's pool assignment, else the
            server's placement.createPools.
        guestUser:
          type: string
          pattern: ^[a-z_][a-z0-9_-]{0,31}$
          maxLength: 32
          examples:
            - alice
          description: >-
            The sandbox's Linux user (`Machine.guestUser`), so `ssh <sandbox>`
            over your own network (Tailscale) logs in as the same name as on
            your computer. Lowercase letters, digits, `_` and `-`, starting with
            a letter or `_`, at most 32 characters; reserved names (`root`,
            `instabox`, the Debian base system's users and groups such as
            `sudo`, `systemd-*`) are refused with `400 invalid_guest_user`.
            Clients send the local username normalized (`DOMAIN\alice` and
            `alice@corp.example` become `alice`, lowercase, spaces become `_`)
            and nothing when that is not a valid name. Absent: `user`; with
            `snapshotId`, the snapshot's user. Ignored (the sandbox gets `dev`)
            for templates without the `guest-user` capability. Fixed at create.
        failFast:
          type: boolean
          default: false
          description: >-
            When no host has room for the sandbox right now, refuse the create
            at once with `503 no_capacity` (`retryable`; nothing is created,
            there is no operation and the `Idempotency-Key` is not kept) instead
            of answering 202 with an operation that then fails with
            `no_capacity`. Every other check and the rest of the create are
            unchanged; a host that had room can still fill up before the
            operation places the sandbox. The check does not ask for a host with
            the secret proxy, which a create with `secrets` or `envSecrets`
            needs: if only hosts without it have room, the create is accepted
            and its operation fails with `no_capacity`. A create from a snapshot
            that no host can start now is refused too, instead of being created
            stopped. A server without this field ignores it, or, if it already
            refuses unknown body keys, answers `400 unknown_field`.
        acceptMinimumLease:
          type: boolean
          description: >-
            Accepts the Mac lease of a macOS sandbox (the template's
            `minimumLease`, `GET /v1/templates`): a whole Mac for the org for at
            least 24 hours, billed whatever the sandbox does. Required to create
            one: without it the create is `400 minimum_lease_required` and
            nothing is placed.
        model:
          type: string
          examples:
            - m4pro
          description: >-
            The Mac model of a macOS template with `models`, by its `name`: the
            sandbox runs on that model's Macs, at one of that model's sizes.
            Absent: the template's `pool`. Not with `snapshotId`: a snapshot's
            sandbox runs on the snapshot's model.
        gpu:
          type: string
          examples:
            - l4
            - l4:2
          description: >-
            GPUs for the sandbox: `TYPE` (one GPU) or `TYPE:COUNT`, such as `l4`
            or `l4:2`, a type of the GPU template's `gpuTypes` (`GET
            /v1/templates`). GPU sandboxes are for paid plans: a free org's
            create is `403 template_not_allowed`. The count is at most the
            type's `maxPerMachine`, and an org's running sandboxes hold at most
            20 GPUs in all (`403 template_limit` past it, also on a start,
            resume or wake; a stopped or paused sandbox holds none). The
            template defaults to `gpu`, and the size comes from the type: one of
            its `sizes`, per GPU (`l4:2` gets twice the vCPUs and memory, the
            same disk; `vcpus` and `memMiB` pick another of its sizes,
            multiplied). An unknown type, a template without GPU support or a
            count over the limit is `400 bad_request`. With `snapshotId` of a
            GPU sandbox's snapshot, the snapshot's GPU type with any count
            (`l4:4` from a one-GPU snapshot), sized from that count; another
            type is `400 bad_request`, and absent `gpu` gets the snapshot's own
            count. GPUs are billed per GPU-hour while the sandbox runs, on top
            of its vCPUs and memory. Absent: no GPU (from a snapshot: the
            snapshot's GPU).
        orgId:
          type: string
          description: >-
            The org the sandbox counts against and is shared with (`GET
            /v1/orgs`); you must be a member. Absent: your personal org, or with
            `snapshotId` the snapshot's org (a sandbox from a snapshot lives in
            the snapshot's org; another is `400 bad_request`). Secret selections
            always resolve within this org. The org's sandbox limit and, on the
            free tier, its monthly included usage apply (`429 quota_exceeded`,
            `402 credit_exhausted`).
    Metadata:
      type: object
      description: >-
        String labels. At most 32 keys of 1-64 characters `[A-Za-z0-9_.-]`,
        values of at most 512 characters, 8 KiB as JSON.
      additionalProperties:
        type: string
    MaxRunMinutes:
      type: integer
      minimum: -1
      maximum: 2147483647
      description: >-
        Maximum continuous runtime in minutes. -1 is unlimited; 0 sleeps
        immediately. Independent of idle auto-pause; whichever condition is
        reached first pauses the sandbox. On a free machine the service sets it
        and only an admin session can change it: each run starts at 120 minutes
        and grows with use up to 480 (commands, file access, port requests,
        terminal or SSH traffic and CPU load count as use; activity leases and
        wake calls do not):
        https://instabox.sh/docs/machines/auto-pause#runtime-on-the-free-plan. A
        free machine that is not running reports 480, the most a run can last.
        The response reports the current limit from runStartedAt; it can change
        during the run. Paid machines default to -1 and their users can change
        this setting. Max-run suspension starts a cooldown, returning 409
        machine_cooldown with Retry-After on every wake path. By default it is 5
        to 20 minutes (5 plus a random 0 to 15) on a paid plan and for the first
        such pause in a row on the free plan. Back-to-back pauses on the free
        plan rest longer (60 to 75, then 120 to 135 minutes): "Longer cooldowns
        on the free plan" at
        https://instabox.sh/docs/machines/auto-pause#longer-cooldowns-on-the-free-plan.
        Once cooldown ends, 0 still refuses wake with 409 max_run_disabled until
        changed.
    AutoSuspendAfterSeconds:
      type: integer
      description: >-
        Absent: 1800 (30 minutes). The plan decides, and a value is honoured or
        refused, never silently replaced: a paid sandbox takes 300-1800 (5 to 30
        minutes) in whole minutes, as sent, and 0 (off) where the plan allows
        always on (else `403 spec_not_allowed`); below 300 (1-299), above 1800,
        negative or not a multiple of 60 is `400 bad_request`. The free sandbox
        is fixed at 1800: any other value is `403 spec_not_allowed`, a plan
        limit (older servers: `403 permission_denied`). A refused create makes
        no sandbox. The reply's `autoSuspendAfterSeconds` shows the effective
        value. Separately, an account an admin gave manual lifecycle (`GET
        /v1/me` `manualLifecycle`) sends 120-86400 idle seconds, or 0 never
        (docs/idle-suspend.md).
    SecretSelection:
      description: >-
        Proxy selection only. All includes current and future proxy Secrets in
        the sandbox's org. Explicit names resolve to stable IDs. An empty array
        selects none. Deleting and recreating a name does not restore an
        explicit grant. Null is rejected.
      oneOf:
        - type: string
          const: all
        - type: array
          items:
            $ref: '#/components/schemas/SecretName'
    SecretName:
      type: string
      pattern: ^[A-Za-z_][A-Za-z0-9_]{0,127}$
      description: >-
        A unique environment variable name within an org. The names a sandbox's
        env reserves are reserved (shell variables such as PATH, HOME, BASH_ENV,
        ENV and IFS, session and trust-store variables, and INSTABOX_ and SSH_
        prefixes). A Secret saved under a name before it was reserved is not
        delivered: the machine's guest status names it.
    DirectSecret:
      type: object
      required:
        - secret
      description: >-
        A reference to a direct (delivery env) Secret: its real value as the
        variable `env` (exported by login shells), as a file at `file`
        (home-relative; the value stays on tmpfs at /run/instabox/secrets/files
        and the home path is a symlink to it), or both. Names and paths only;
        values are never returned.
      properties:
        secret:
          $ref: '#/components/schemas/SecretName'
        env:
          type: string
          pattern: ^[A-Za-z_][A-Za-z0-9_]{0,127}$
          description: >-
            Not reserved (PATH, HOME, BASH_ENV, IFS, CA variables, ...) and no
            INSTABOX_ or SSH_ prefix.
        file:
          type: string
          maxLength: 255
          description: Clean path relative to the home directory, without .. components.
    Operation:
      type: object
      description: >-
        The progress of a change to a sandbox (`suspend` is pause; `restore`
        puts its disks back to a snapshot's; `move`, admin only, copies it to
        another host pool; `migrate`, admin only, moves a running one to another
        host with its processes). It does not return the request it was queued
        with (a start's `vcpus`/`memMiB` show in the sandbox's `spec`).
      required:
        - id
        - machineId
        - type
        - state
        - error
        - createdAt
        - updatedAt
      properties:
        id:
          type: string
        machineId:
          type: string
        type:
          type: string
          enum:
            - create
            - start
            - stop
            - suspend
            - resume
            - delete
            - restore
            - resize
            - move
            - migrate
        state:
          type: string
          enum:
            - queued
            - running
            - succeeded
            - failed
          description: '`succeeded` and `failed` are terminal.'
        phase:
          type: string
          description: >-
            Free text progress, e.g. scheduling, booting, waiting_ready,
            snapshotting.
        forkInherited:
          type: boolean
          description: >-
            On the answer to `POST /v1/machines/{id}/fork`: the control host
            that took it keeps the source's delete deadline, idle auto-suspend,
            max run and metadata, and takes `autoSuspendAfterSeconds` and
            `maxRunMinutes`. Absent: a host from before that (during a rolling
            control deploy), which gave the fork its defaults with only the
            request's `metadata` and `deleteAfterSeconds`; set the idle
            auto-suspend and max run with `PATCH /v1/machines/{id}`.
        trigger:
          type: string
          enum:
            - idle
            - max_run
            - memory_pressure
            - timeout
            - archive
            - delete_at
            - drain
            - request
            - wake
            - user_delete
            - user_disable
            - admin
            - admin_delete
            - credit
            - spend_cap
            - downgrade
            - subscription_ended
            - payment_method
            - members
            - early_allowance_ended
            - abuse_pool
            - abuse_cpu
            - abuse_scan
            - abuse_proxy
          description: >-
            Set when instabox started the operation itself: `idle` (paused when
            idle), `max_run` (maximum continuous runtime), `memory_pressure`
            (paused, its processes kept, because the host it ran on was running
            out of memory; the next use wakes it, as after `idle`), `request`
            (wake on use), `wake` (POST /v1/machines/{id}/wake), `timeout`,
            `archive`, `delete_at` (the sandbox's `deleteAt` passed), `drain`
            (maintenance by an admin), `user_delete` (an admin deleted the
            owner), `user_disable` (an admin disabled the owner), `admin` (an
            admin stopped or paused the sandbox), `admin_delete` (an admin
            deleted the sandbox), `credit` (its free-tier org used up the
            month's included usage), `spend_cap`, `downgrade`,
            `subscription_ended`, `payment_method` (its free workspace has no
            card on file), `members` (its free workspace has more members than
            the free plan allows), `early_allowance_ended` (paused because the
            account's early free-sandbox allowance ended and it has more free
            sandboxes than the free plan runs), `abuse_pool` (paused because the
            sandbox tried to connect to a blocked destination on the egress
            blocklist, such as a crypto-mining pool; a DNS lookup of a listed
            domain never suspends) or `abuse_cpu` (paused by the CPU abuse
            detector: a restricted account's sandbox ran flat out for an hour
            with no memory use, no public port, nobody using it and a connection
            to a mining port) or `abuse_scan` (paused by the flow detector: a
            restricted account's sandbox swept one port across many hosts,
            scanned many ports of one host or brute-forced a login port).
            Absent: a client request.
        error:
          oneOf:
            - $ref: '#/components/schemas/Error'
            - type: 'null'
        twin:
          allOf:
            - $ref: '#/components/schemas/TwinCreate'
          description: >-
            On the answer to a half-size macOS create that opened a Mac: the
            second sandbox the create made on it (a Mac runs two sandboxes of
            half its size).
        scheduledAt:
          type: string
          format: date-time
          description: >-
            The stop or delete of a macOS sandbox inside its Mac lease's minimum
            is not done now but at this time (the sandbox's `archiveAt` or
            `deleteAt`); the operation is `succeeded`.
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
        machine:
          allOf:
            - $ref: '#/components/schemas/Machine'
          description: >-
            The sandbox as it is when `GET /v1/operations/{id}` is read.
            Included when the operation `succeeded`, the sandbox still exists,
            and the caller may read sandboxes (`machines:read` for API keys).
            Later operations can change it; it is not a record of the state at
            completion. Absent otherwise, and from older servers.
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
    TwinCreate:
      type: object
      description: >-
        The twin sandbox of a half-size macOS create and its create operation
        (`Operation.twin`).
      required:
        - machineId
        - name
        - operationId
      properties:
        machineId:
          type: string
        name:
          type: string
        operationId:
          type: string
    Machine:
      type: object
      description: A sandbox.
      required:
        - id
        - name
        - desiredState
        - state
        - revision
        - operationId
        - template
        - spec
        - storage
        - runtime
        - autoSuspendAfterSeconds
        - maxRunMinutes
        - runStartedAt
        - cooldownUntil
        - metadata
        - vault
        - guestUser
        - orgId
        - createdAt
        - updatedAt
      properties:
        id:
          type: string
        name:
          type: string
        desiredState:
          $ref: '#/components/schemas/MachineState'
        state:
          $ref: '#/components/schemas/MachineState'
        revision:
          type: integer
          format: int64
          description: Increases with each accepted change.
        operationId:
          type:
            - string
            - 'null'
          description: The active operation, if any.
        template:
          $ref: '#/components/schemas/TemplateRef'
        snapshot:
          $ref: '#/components/schemas/SnapshotRef'
        image:
          allOf:
            - $ref: '#/components/schemas/SnapshotRef'
          deprecated: true
          description: 'Deprecated: `snapshot`, under its old name (same value).'
        spec:
          $ref: '#/components/schemas/MachineSpec'
        storage:
          $ref: '#/components/schemas/Storage'
        runtime:
          $ref: '#/components/schemas/Runtime'
        lastError:
          $ref: '#/components/schemas/Error'
        archiveAt:
          type:
            - string
            - 'null'
          format: date-time
          description: >-
            When the sandbox is stopped if it is running (set at creation with
            `archiveSeconds`): its files are kept and a start brings it back.
            Resuming does not clear it; a sandbox that is not running when it
            passes only has it cleared. Absent: none.
        lease:
          allOf:
            - $ref: '#/components/schemas/MachineLease'
          description: >-
            The Mac lease a macOS sandbox runs under. Absent on every other
            sandbox, and while a macOS sandbox runs under none (stopped after
            its lease ended).
        deleteAt:
          type:
            - string
            - 'null'
          format: date-time
          description: >-
            When the sandbox is deleted, whatever its state (set at creation
            with `deleteAfterSeconds`). The delete is an ordinary `delete`
            operation (`trigger: delete_at`), queued at `deleteAt` (within about
            a second) once no other operation is in progress. Absent: never.
        autoSuspendAfterSeconds:
          type:
            - integer
            - 'null'
          description: >-
            The running sandbox is paused once it has been idle this many
            seconds (no agent working, no terminal input or output, little CPU);
            null: off. Pause, resume, stop and start keep it.
        maxRunMinutes:
          $ref: '#/components/schemas/MaxRunMinutes'
        runStartedAt:
          type:
            - string
            - 'null'
          format: date-time
          description: >-
            Start of the current continuous run, conservatively initialized on a
            first error observation too; resets only after confirmed sleep.
        readyAt:
          type: string
          format: date-time
          readOnly: true
          description: >-
            When the running sandbox last became ready (`runtime.ready` turned
            true: SSH, exec and files work). For a new sandbox, `readyAt` minus
            `createdAt` is its create time; after a start or resume it is that
            start's. Absent while the sandbox is not ready, and on a sandbox
            that was already ready before the field existed (until its next
            start or resume).
        cooldownUntil:
          type:
            - string
            - 'null'
          format: date-time
          description: >-
            All wake paths are refused until this deadline, including admin
            actions: after a max-run suspension, or the 24-hour lock of an
            automatic abuse check's pause (cooldownReason).
        cooldownReason:
          type: string
          enum:
            - abuse_egress_blocklist
            - abuse_network_flows
            - abuse_cpu
          description: >-
            While cooldownUntil has not passed, the automatic abuse check whose
            pause locks the sandbox, the same as a wake's 409 `machine_cooldown`
            Error.reason: `abuse_egress_blocklist` (it tried to connect to a
            blocked destination on the egress blocklist, not a DNS lookup),
            `abuse_network_flows` (scanning or attack traffic), `abuse_cpu`
            (mining). Absent for a max-run cooldown, and once the lock ends.
        abuseLock:
          $ref: '#/components/schemas/MachineAbuseLock'
          description: >-
            With cooldownReason: what the owner is told about the pause behind
            the lock. Absent when cooldownReason is.
        suspendable:
          type: boolean
          description: >-
            false: the sandbox cannot be paused, since its hosts keep no memory
            of a running sandbox (macOS). Stop the sandbox instead; its files
            are kept. Absent: it can.
        autoSuspendedAt:
          type: string
          format: date-time
          description: >-
            When the sandbox was queued to pause automatically; a successful
            automatic resume or its next explicit operation clears it. Absent:
            none.
        metadata:
          $ref: '#/components/schemas/Metadata'
        envNames:
          type: array
          items:
            type: string
          description: >-
            The names of the sandbox's create-time `env`, sorted. Absent when it
            has none. The values are never shown: read them in the sandbox.
        vault:
          type: boolean
          description: >-
            Whether legacy Vault delivery is enabled. Independent of org Secret
            selections and delivery status at the sandbox's /secrets endpoint.
        guestUser:
          type: string
          readOnly: true
          examples:
            - alice
          description: >-
            The Linux user in the sandbox that SSH sessions, exec and the
            terminal log in as, fixed at create: the `guestUser` the create
            named, else `user` (a create from a snapshot: the snapshot's user).
            `dev` on sandboxes from templates without the `guest-user`
            capability, and on every sandbox created before the field.
        orgId:
          type: string
          readOnly: true
          description: The org the sandbox counts against (docs/orgs.md).
        free:
          type: boolean
          readOnly: true
          description: >-
            A free-tier org's sandbox (docs/pricing.md): never billed, pauses
            after 30 idle minutes (another `autoSuspendAfterSeconds` is refused
            with 403). On a paid plan every sandbox is billed.
        earlyAllowanceEnded:
          type: boolean
          readOnly: true
          description: >-
            Set on a stopped or paused free sandbox that cannot be started
            because its org's early free-sandbox allowance ended and as many
            free sandboxes as the standard limit allows are running: a start is
            refused (`402 early_allowance_ended`) until one is paused or
            deleted. Absent otherwise.
        oomKills:
          type: integer
          format: int64
          readOnly: true
          description: >-
            Processes the sandbox's kernel OOM killer killed since the sandbox
            was created (out of memory, or a memory cgroup's limit). Read from
            the sandbox about once a minute while it runs, and after an exec
            that exits non-zero; 0 on sandboxes whose host or guest agent
            predates the reading (absent from older servers).
        lastOom:
          $ref: '#/components/schemas/OOMKill'
          description: >-
            The newest OOM kill (absent until one is seen, or when the kernel
            log no longer held it).
        createdAt:
          type: string
          format: date-time
        updatedAt:
          type: string
          format: date-time
    MachineState:
      type: string
      enum:
        - creating
        - starting
        - running
        - stopping
        - stopped
        - suspending
        - suspended
        - resuming
        - deleting
        - deleted
        - error
    TemplateRef:
      type: object
      required:
        - id
        - version
        - digest
      properties:
        id:
          type: string
        version:
          type: string
        digest:
          type: string
        kernel:
          type: string
          description: >-
            The guest kernel the sandbox is pinned to, `sha256:<hex>`; absent:
            the host's default.
    SnapshotRef:
      type: object
      required:
        - id
      properties:
        id:
          type: string
    MachineSpec:
      type: object
      description: A sandbox's size.
      required:
        - vcpus
        - memMiB
        - persistentDiskGiB
      properties:
        vcpus:
          type: integer
        memMiB:
          type: integer
          description: Memory in MiB.
        persistentDiskGiB:
          type: integer
          description: Disk (`/data`) size in GiB.
        rootDiskGiB:
          type: integer
          description: >-
            System disk size in GiB, set by the template (only admins change
            it); absent: the template's.
        gpus:
          type: integer
          description: >-
            GPUs of `gpuType` the sandbox has (`gpu` on create); absent on a
            sandbox without one.
        gpuType:
          type: string
          examples:
            - l4
          description: >-
            The GPU type of its GPUs (`GET /v1/templates` `gpuTypes`); absent on
            a sandbox without one.
    Storage:
      type: object
      description: >-
        The sandbox's storage (`/data`, where HOME lives; `/` on macOS, whose
        disk is the root volume).
      required:
        - sizeGiB
        - mountPath
        - persistent
        - synced
      properties:
        sizeGiB:
          type: integer
        mountPath:
          type: string
          examples:
            - /data
            - /
        persistent:
          type: boolean
        synced:
          type: boolean
          description: >-
            False while the latest state of a stopped or paused sandbox is still
            being saved: for a few seconds after a stop or pause, or longer if
            saving fails (it is retried). A snapshot waits until it is true.
        snapshotSeq:
          type: integer
          format: int64
          description: >-
            Sequence number of the sandbox's latest saved state; absent before
            the first stop or pause.
    Runtime:
      type: object
      required:
        - ready
      properties:
        ready:
          type: boolean
          description: >-
            True only when the sandbox is running, reports all checks passing
            and, as checked when it boots or resumes, accepts exec, file and SSH
            connections.
        processesPreserved:
          type: boolean
          description: >-
            Set after a start or resume. False means the paused programs could
            not be brought back and the sandbox booted fresh (files are kept).
        hostUnreachable:
          type: boolean
          description: >-
            True while the host the sandbox runs on is not answering. The
            sandbox cannot be reached (exec, files and SSH fail with `503
            host_unavailable`) and `ready` is false, whatever `state` says;
            `start` and `resume` of a running sandbox return `503
            host_unavailable` instead of succeeding. Absent when the host
            answers. Retry later: it clears once the host answers again.
        hostLastSeenAt:
          type: string
          format: date-time
          description: When the sandbox's host last answered; set with `hostUnreachable`.
    MachineLease:
      type: object
      description: The Mac lease a macOS sandbox runs under (`Machine.lease`).
      required:
        - id
        - startedAt
        - chargeUntil
        - hourUSD
      properties:
        id:
          type: integer
          format: int64
        startedAt:
          type: string
          format: date-time
          description: Since when the Mac is leased to the org.
        chargeUntil:
          type: string
          format: date-time
          description: 'Until when the lease is billed at least: the end of its minimum.'
        hourUSD:
          type: number
          description: The lease's price per Mac-hour, all in.
    MachineAbuseLock:
      type: object
      description: >-
        The owner's account of the abuse check's pause behind a sandbox's
        cooldownReason: when, what the sandbox did in a sentence (explanation)
        and in detail (what, with the exact address of a blocked destination,
        the address and port of a brute force, the swept port or scanned host
        and the count of a scan, the CPU share and, when the check confirmed it,
        the mining program and its pool), and how to appeal.
      required:
        - at
        - explanation
        - what
        - appeal
      properties:
        at:
          type: string
          format: date-time
          description: When the sandbox was paused.
        explanation:
          type: string
          description: >-
            One plain sentence on what the sandbox did ("was reaching known
            abuse addresses ...").
        what:
          type: string
          description: >-
            The finding with its exact address and counts ("connected to blocked
            47.91.89.3", "30 connections to 198.51.100.7:22 (brute force)",
            "swept port 22 across 500 destinations").
        dst:
          type: string
          description: The blocked destination, the brute-forced or scanned address.
        port:
          type: integer
          description: >-
            A brute force's destination port, or the swept port. A blocked
            destination's port is not recorded (the node keeps the address
            alone).
        kind:
          type: string
          description: 'A network-flows finding''s kind: sweep, scan, brute or fanout.'
        count:
          type: integer
          description: A network-flows finding's destinations, ports or connections.
        coreShare:
          type: number
          description: The CPU check's mean share of the sandbox's vCPUs.
        process:
          type: string
          description: >-
            The mining program the CPU check found running, when it confirmed
            one.
        pool:
          type: string
          description: That program's pool.
        appeal:
          type: string
          description: How to contest the pause.
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
  responses:
    OperationAccepted:
      description: >-
        The mutation was accepted (or replayed for the same idempotency key).
        Poll the operation.
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Operation'
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
    PaymentRequired:
      description: >-
        `credit_exhausted`: the account's free sandbox used its hours for the
        month (when the server enforces a cap); it can run again next month
        (UTC). `spend_limit_reached`: the sandbox's paid org reached its spend
        cap for the month. `subscription_required`: the org's subscription
        ended. `payment_failed`: the charge a plan change needed did not
        succeed; the plan is unchanged.
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