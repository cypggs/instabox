> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# TypeScript SDK

> Install, authenticate, errors and types of the TypeScript SDK, @insforge/instabox-sdk.

Install, authentication, errors and types of `@insforge/instabox-sdk`, version 0.7.0. A `Sandbox` is one
machine: its methods are on the [Sandbox](https://instabox.sh/docs/reference/typescript/sandbox.md), [Commands](https://instabox.sh/docs/reference/typescript/commands.md),
[Files](https://instabox.sh/docs/reference/typescript/files.md), [Desktop](https://instabox.sh/docs/reference/typescript/desktop.md) and
[Metrics and usage](https://instabox.sh/docs/reference/typescript/metrics.md) pages. The
common calls, step by step, are in the [SDK quickstart](https://instabox.sh/docs/integrations/sdk-quickstart.md).

```ts theme={null}
import { Sandbox } from '@insforge/instabox-sdk'

const sbx = await Sandbox.create('agent-browser')
const r = await sbx.commands.run('uname -a')
console.log(r.stdout)
await sbx.kill()
```

## Install and authenticate

```bash theme={null}
npm install @insforge/instabox-sdk
export INSTABOX_API_KEY=ibk_...
# optional, the default is https://api.instabox.sh
export INSTABOX_API_URL=https://api.instabox.sh
```

Every static call (`create`, `connect`, `list`, `kill`, the snapshot calls and the metrics and usage
calls) also takes the key and URL as options, which win over the environment. A `Sandbox` object keeps
its connection, so its own methods take no key.

```ts theme={null}
const sbx = await Sandbox.create('agent-browser', {
  apiKey: process.env.INSTABOX_API_KEY,
  apiUrl: 'https://api.instabox.sh',
})
```

With no key from either, the call throws `InstaboxAuthError` with code `missing_api_key` (status 0)
before any request. The key is sent as the `X-API-Key` header.

### ConnectionOptions

These apply to every static call.

| Option | Type | Default |
| - | - | - |
| `apiKey` | `string` | env `INSTABOX_API_KEY` |
| `apiUrl` | `string` | env `INSTABOX_API_URL`, else `https://api.instabox.sh` |
| `requestTimeoutMs` | `number` | 60 000 per HTTP request. A command gets its own timeout plus 30 000 |
| `operationTimeoutMs` | `number` | 600 000: the wait for create, pause, resume and kill |
| `fetch` | `typeof fetch` | the global `fetch` (Node 18+, Bun, Deno) |
| `pollIntervalMs` | `number` | 250: first delay of operation polling, grows 1.5x to 1 s |
| `retryDelayMs` | `number` | 250: first delay between retries |

### Scopes

The [API key](https://instabox.sh/docs/security/api-keys.md) scopes each call needs.

| Scope | Allows |
| - | - |
| `machines:read` | `Sandbox.list`, `getInfo`, `isRunning`, `getMetrics`, `getUsage`, `Sandbox.usage` |
| `machines:write` | `create`, `connect` (resume), `pause`, `resume`, `stop`, `setAutoSuspend`, `setMetadata`, `commands`, `files`, `sshCredentials` |
| `machines:delete` | `kill` |
| `images:read` | Resolve private snapshot names, `listSnapshots`, `getSnapshot` |
| `images:write` | `createSnapshot` |
| `images:delete` | `deleteSnapshot` |

`:write` and `:delete` include `:read`. Creating from a snapshot name also needs `images:read`. A call
outside the key's scopes throws `InstaboxAuthError` (`403 missing_scope`).

## Errors

| Class | When |
| - | - |
| `InstaboxError` | Base class of every error. Fields: `status`, `code`, `message`, `requestId`, `retryable`, `operationId` |
| `InstaboxAuthError` | 401 or 403: a bad or revoked key, or a missing scope. No key at all: `missing_api_key`, status 0 |
| `InstaboxQuotaError` | 429, such as `quota_exceeded` |
| `InstaboxTimeoutError` | A request or an operation did not finish in time |
| `CommandExitError` | A command exited non-zero. Also has `stdout`, `stderr`, `exitCode`, `timedOut`, `truncated`, `error` |

`status` is 0 for errors that did not come from an HTTP response. A failed operation (for example
`no_capacity` while creating) throws `InstaboxError` with the operation's `code` and `operationId`.

| Code | Meaning |
| - | - |
| `missing_api_key` | No key in the options or the environment |
| `missing_scope` | The key lacks a scope the call needs (403). Older servers send `insufficient_scope` |
| `missing_key`, `malformed_key`, `invalid_key`, `expired_key`, `revoked_key` | The key was not sent, is not a key, does not exist, expired or was revoked (401) |
| `not_found` | No such machine or snapshot |
| `operation_conflict` | Another operation runs on the machine (409). `operationId` names it |
| `quota_exceeded` | An account limit was reached (429) |
| `spec_not_allowed` | A size your plan does not offer, a disk size at create, or growing a free machine's disks (400). Another idle time on the free plan (403) |
| `template_not_allowed` | A template your plan does not offer: the free plan creates from agent-browser, android, omarchy, ci only (403) |
| `is_a_directory` | `files.write` to a directory |
| `source_error` | The stream given to `files.write` threw. The original error is the `cause`, and the file keeps its old content |
| `local_not_found` | `files.upload`: the local directory does not exist |
| `local_io_error` | `files.upload` or `files.download`: reading or writing the local disk failed |
| `not_a_directory` | `files.upload` or `files.download`: a file where a directory was expected |
| `transfer_failed` | `files.upload` or `files.download`: `tar` failed in the machine (`permission_denied`, 403, when it was refused access) |
| `unsupported` | `files.upload` or `files.download` in a browser or a Worker, which have no local file system |
| `request_timeout`, `operation_timeout` | A request or an operation took too long |
| `network_error` | The request did not reach the API |

Reads, and changes (which carry an `Idempotency-Key`), are retried up to 3 times with backoff on 502, 503,
504 and network errors. Every server code: [Errors](https://instabox.sh/docs/reference/errors.md).

## Types

| Type | What |
| - | - |
| `ConnectionOptions` | `apiKey`, `apiUrl` and the timeouts every static call takes |
| `SandboxOpts`, `SandboxConnectOpts`, `SandboxListOpts` | Options of `create`, `connect`, `list` |
| `SandboxInfo` | What `getInfo` and `list` return |
| `Machine`, `MachineSpec`, `MachineState` | The raw machine from the API, its size, its state |
| `CreateSnapshotOpts`, `Snapshot` | Snapshot options, a saved snapshot |
| `SandboxMetrics`, `SandboxMetricsOpts` | `getMetrics` samples and options |
| `SandboxUsage`, `SandboxUsageOpts` | `getUsage` result and options |
| `Usage`, `UsageLimits`, `UsageStorage`, `UsageMachineStorage` | What `Sandbox.usage` returns |
| `SshCredentials` | What `sshCredentials` returns |
| `CommandOptions`, `CommandResult`, `CommandHandle`, `CommandWaitOptions` | `commands.run` options and results, `CommandHandle.wait` options |
| `EntryInfo`, `WriteInfo` | `files.list` entries, `files.write` result |
| `WriteData`, `WriteOptions` | What `files.write` takes: a string, bytes or a stream. `signal` |
| `FileStream`, `ReadStreamOptions` | What `files.readStream` returns, and its `signal` |
| `UploadOptions`, `UploadInfo` | `files.upload` options (`git`, `all`, `signal`) and result |
| `DownloadOptions`, `DownloadInfo` | `files.download` options (`signal`) and result |
| `DesktopStatus`, `DesktopStream` | `desktop.status` and `desktop.streamUrl` results |
| `MouseButton` | `'left'`, `'right'`, `'middle'` |
| `ScrollDirection` | `'up'`, `'down'`, `'left'`, `'right'` |
| `Template`, `TemplateRef`, `Operation`, `ApiError` | API objects |

## Constants

| Constant | Value |
| - | - |
| `DEFAULT_API_URL` | `https://api.instabox.sh` |
| `DEFAULT_TEMPLATE` | `agent-browser` |
| `FALLBACK_TEMPLATE` | `universal` |
| `MAX_FILE_BYTES` | 10 MiB, the default cap of `files.read` |
| `MAX_UPLOAD_BYTES` | 5 GiB, the largest file `files.write` sends |
| `MAX_STDIN_BYTES` | 1 MiB, the most stdin sent in the request itself |
| `DESKTOP_DISPLAY` | `:1` |
