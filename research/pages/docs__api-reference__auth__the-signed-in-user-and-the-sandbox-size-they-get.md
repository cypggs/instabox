> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# The signed-in user and the sandbox size they get.

> "API keys: any key." Not behind the invite gate: `inviteRequired` and `activated` tell a client whether the account must redeem an invite code (`POST /v1/invites/redeem`) before the other routes answer.



## OpenAPI

````yaml /api-reference/openapi.yaml get /v1/me
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
  /v1/me:
    get:
      tags:
        - auth
      summary: The signed-in user and the sandbox size they get.
      description: >-
        "API keys: any key." Not behind the invite gate: `inviteRequired` and
        `activated` tell a client whether the account must redeem an invite code
        (`POST /v1/invites/redeem`) before the other routes answer.
      operationId: getMe
      responses:
        '200':
          description: The user.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Me'
        '401':
          $ref: '#/components/responses/Unauthenticated'
        '403':
          $ref: '#/components/responses/PermissionDenied'
      security:
        - bearerAuth: []
        - apiKey: []
        - sessionCookie: []
components:
  schemas:
    Me:
      type: object
      required:
        - userId
        - login
      properties:
        userId:
          type: integer
          format: int64
          description: Instabox user ID.
        login:
          type: string
          description: GitHub login at last sign-in (display only).
        name:
          type: string
        email:
          type: string
          description: >-
            The account's email, which an org invite must be sent to for it to
            join.
        avatarUrl:
          type: string
          format: uri
        admin:
          type: boolean
          description: >-
            The user is an admin and may use the admin-only endpoints
            (`/v1/admin`).
        machineMetrics:
          type: boolean
          description: >-
            Clients show this account per-sandbox metrics (the dashboard's
            Metrics tab, `instabox metrics`): admins always, every account once
            the server sets `machineMetricsForAll`. Absent: hidden. `GET
            /v1/machines/{id}/metrics` answers either way.
        movePersonal:
          type: boolean
          description: >-
            This account may move its personal plan into a workspace it owns
            (`POST /v1/orgs/{id}/billing/move-personal`): admins always, every
            account once the server sets `movePersonalForAll`.
        orgId:
          type: string
          description: >-
            Set for a workspace API key: its workspace. The user is then the
            workspace's owner, whom the key acts as.
        adminUrl:
          type: string
          format: uri
          description: >-
            Admins only: the origin of the admin console when it has its own
            (the server's `adminUrl`). Absent, the console is `/admin/` on the
            dashboard's origin.
        billingEnabled:
          type: boolean
          description: >-
            Paid plans can be subscribed: the server has Stripe configured.
            False: the dashboard offers no plan actions and the billing routes
            answer 503.
        webhooksEnabled:
          type: boolean
          description: >-
            This server offers org webhooks (docs/webhooks.md). False: the
            dashboard hides them, the CLI says they are off and the webhook
            routes answer 404.
        emailInvites:
          type: boolean
          description: >-
            This server emails workspace invites (server.json `email`). Absent:
            creating or resending an invite answers 503.
        paidActivation:
          type: boolean
          description: >-
            An account the invite gate holds back may buy a workspace (`POST
            /v1/orgs`) instead, and is activated once its plan takes effect
            (docs/invites.md). Absent: only an invite code lets it in.
        plans:
          type: array
          items:
            $ref: '#/components/schemas/PlanOffer'
          description: >-
            The paid plans an org can go on (Pro, Builder, Team) as this server
            sells them, for plan pickers.
        csrfToken:
          type: string
          description: >-
            Browser sessions only. Send it as `X-Instabox-CSRF` on every request
            that is not GET or HEAD.
        allowedSpecs:
          type: array
          items:
            $ref: '#/components/schemas/MachineSpec'
          description: >-
            The sizes your new sandboxes may have in your personal org (no
            `rootDiskGiB`: the template's). The first is the server's
            `limits.machineSpec` (4 vCPUs and 8 GiB memory by default), what a
            create naming no size gets; the others are your plan's (Pro: 2/4,
            4/8, 8/16; Builder: 2/4 up to 16/32; Team and Enterprise: 2/4 up to
            32/64) and sizes an admin granted you, picked by naming their
            `vcpus` and `memMiB` on create. Any other size is refused with
            `spec_not_allowed`, admins included.
        org:
          allOf:
            - $ref: '#/components/schemas/OrgRef'
          description: >-
            The org `allowedSpecs`, `idleChoicesSeconds` and `alwaysOnAllowed`
            describe: your personal org, or a workspace key's workspace. Absent
            from older servers.
        workspaces:
          type: array
          items:
            $ref: '#/components/schemas/OrgRef'
          description: >-
            Your other orgs, each with its own plan and limits (`GET
            /v1/usage?org=ID`, `GET /v1/orgs/{id}`). Omitted when there are
            none, and for a workspace API key, which reaches only its own org.
        inviteRequired:
          type: boolean
          description: >-
            This server gates new accounts behind an invite code
            (docs/invites.md).
        activated:
          type: boolean
          description: >-
            This account passed the invite gate (admins always have). When
            `inviteRequired` and not `activated`, every route but this one,
            `POST /v1/invites/redeem` and `POST /v1/auth/logout` answers `403
            invite_required`.
        manualLifecycle:
          type: boolean
          description: >-
            An admin let this account manage its sandboxes' lifecycle
            (docs/idle-suspend.md): the `autoSuspendAfterSeconds` it sends
            applies (0: off). Otherwise every sandbox pauses after
            `fixedAutoSuspendSeconds` idle, whatever a request says.
        fixedAutoSuspendSeconds:
          type: integer
          description: >-
            The idle auto-suspend every sandbox of this account gets on a server
            that offers no choice; absent when the account chooses
            (`manualLifecycle`, or `idleChoicesSeconds`).
        idleChoicesSeconds:
          type: array
          items:
            type: integer
          description: >-
            The idle times offered (300 to 1800, 5 to 30 minutes;
            docs/pricing.md): a request may send any whole minute from the first
            to the last. 0 (off) is never listed: `alwaysOnAllowed` says whether
            it is a choice too. Absent with `manualLifecycle`.
        alwaysOnAllowed:
          type: boolean
          description: >-
            Whether 0 (off) is a choice too, on `org`'s plan (the personal org's
            for a sign-in session). A sandbox in another org follows that org's
            plan: see its `workspaces` entry's `alwaysOnAllowed`.
        diskResize:
          type: string
          enum:
            - allow
            - deny
          description: >-
            An admin's override of disk growth (`PATCH /v1/machines/{id}`
            `persistentDiskGiB`, `rootDiskGiB`): `allow` grows free sandboxes'
            disks too, `deny` none. Absent: paid sandboxes grow, free ones are
            fixed.
        resizeRestart:
          type: boolean
          description: >-
            This server takes `PATCH /v1/machines/{id}` `apply: "restart"` (one
            stop, resize, start operation). Absent: it answers `400
            resize_restart_off`; resize with your own stop, `PATCH`, start.
        directSecretRefs:
          type: boolean
          description: >-
            This site takes direct Secret file references and renamed variables
            (`directSecrets`) and exec `secretEnvs`. Absent: they are refused
            with 400 (same-name references work); clients should not send them,
            since a control host from before them would ignore or misread them.
        termsVersion:
          type: string
          description: >-
            The version of the rules of the road (the Terms of Service's
            acceptable use) this account last agreed to (`PUT /v1/me/terms`).
            Absent: none.
        waitlist:
          type: object
          required:
            - referralUrl
          description: >-
            The account's place in the line invite codes go out from and its
            referral link (docs/invites.md, "Referrals"). Absent: not in line
            (activated, the gate off, or a live code claimed for its email).
          properties:
            position:
              type: integer
              minimum: 1
              description: Absent when this server could not rank the line just now.
            referralUrl:
              type: string
              format: uri
    PlanOffer:
      type: object
      description: A paid plan as this server sells it (server.json `plans`).
      required:
        - tier
        - monthlyFeeUSD
        - maxMachines
        - maxRunning
        - maxVCPUs
        - maxDiskGiB
      properties:
        tier:
          type: string
          enum:
            - pro
            - builder
            - team
        monthlyFeeUSD:
          type: number
          description: The monthly fee, also the usage it includes each month.
        maxMachines:
          type: integer
          description: >-
            Sandboxes the org may have on this plan, every one billed; 0: no
            limit.
        maxRunning:
          type: integer
          description: 'Paid sandboxes running at once; 0: no limit.'
        maxVCPUs:
          type: integer
          description: The largest sandbox size's vCPUs.
        maxDiskGiB:
          type: integer
          description: >-
            How far a sandbox's disk (`/data`) may grow on this plan, in GiB
            (free 20, fixed: it neither grows nor shrinks; pro 200; builder 500;
            team and enterprise 1024); a disk already past it is kept. 0: no
            limit.
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
    OrgRef:
      type: object
      required:
        - id
        - name
        - tier
        - kind
      properties:
        id:
          type: string
          description: Org ID (org_...).
        name:
          type: string
        tier:
          type: string
          enum:
            - free
            - pro
            - builder
            - team
            - enterprise
          description: The org's plan.
        kind:
          type: string
          enum:
            - personal
            - workspace
        alwaysOnAllowed:
          type: boolean
          description: >-
            Set on `GET /v1/me` (`org`, `workspaces`): this org's plan lets its
            sandboxes take `autoSuspendAfterSeconds` 0 (off). Absent on a plan
            that does not, elsewhere, and from older servers.
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