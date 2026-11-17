> ## Documentation Index
> Fetch the complete documentation index at: https://instabox.sh/docs/llms.txt
> Use this file to discover all available pages before exploring further.

# MCP

> Machine tools for Claude Code, Cursor, Codex, OpenCode and Antigravity, and a public server that reads these docs.

MCP lets a client call instabox tools without a CLI in its own environment. The hosted server at
`https://api.instabox.sh/mcp` is the recommended way to connect. To let an agent use the `instabox`
CLI instead, see [Coding agents](https://instabox.sh/docs/use-cases/coding-agents.md).

## Connect a client to the hosted server

**1. Install the CLI**

```bash theme={null}
npx instabox@latest
```

The command signs you in and installs the `instabox` CLI. You use the CLI to write the client's
config. The client itself does not need it.

**2. Create an API key**

```bash theme={null}
instabox keys create --name mcp
```

The command prints the key once. Set it as `INSTABOX_API_KEY` in the shell or environment that
starts your client.

**3. Add the server to your client**

```bash Claude Code theme={null}
instabox mcp init claude --hosted
```

```bash Cursor theme={null}
instabox mcp init cursor --hosted
```

```bash Codex theme={null}
instabox mcp init codex --hosted
```

```bash OpenCode theme={null}
instabox mcp init opencode --hosted
```

```bash Antigravity theme={null}
instabox mcp init antigravity --hosted --inline-key
```

The command prints the config file it wrote and tells you to restart the client.

**4. Restart the client and try a tool**

Ask your client to "list my instabox machines". It calls `list_machines` and shows your machines.

> [!WARNING]
>
> Antigravity reads no variables from its config, so `--inline-key` writes the key itself into the
> file, readable by you only. Keep such a file out of git.

## Use the local server without an API key

```bash theme={null}
instabox mcp init claude     # or cursor, codex, opencode, antigravity
```

The client then starts `instabox mcp start`, a local stdio server. It sends each message to the
hosted server with the CLI's credentials: `INSTABOX_API_KEY` when set, otherwise your `instabox login`
sign-in, refreshed as needed. It never opens a browser, so sign in once first. The CLI must stay
installed where the client runs.

> [accordion] Where init writes the config
>
> `init` adds one server named `instabox` and keeps the rest of the file. It writes to your config,
> or with `--project` to the project's file:
>
> | Client | Your config | `--project` |
> | - | - | - |
> | Claude Code | `~/.claude.json` | `./.mcp.json` |
> | Cursor | `~/.cursor/mcp.json` | `./.cursor/mcp.json` |
> | Codex | `~/.codex/config.toml` | `./.codex/config.toml` |
> | OpenCode | `~/.config/opencode/opencode.json` | `./opencode.json` |
> | Antigravity | `~/.gemini/config/mcp_config.json` | `./.agents/mcp_config.json` |

> [accordion] How each client references the key
>
> With `--hosted`, the file holds a reference to `INSTABOX_API_KEY`, not the key:
> `${INSTABOX_API_KEY}` for Claude Code, `${env:INSTABOX_API_KEY}` for Cursor, `env_http_headers` for
> Codex, and `{env:INSTABOX_API_KEY}` for OpenCode.

## Connect to the hosted server by hand

The URL is `https://api.instabox.sh/mcp` (Streamable HTTP). Send an [API key](https://instabox.sh/docs/security/api-keys.md) as
`X-API-Key: ibk_...` or `Authorization: Bearer ibk_...`. A signed-in CLI's access token also works as a
bearer token. Create a key with `instabox keys create --name mcp` or on the **API keys** page.

To add the server to Claude Code by hand, run:

```bash theme={null}
claude mcp add --transport http instabox https://api.instabox.sh/mcp --header "X-API-Key: $INSTABOX_API_KEY"
```

A tool call runs the same route as the REST API, so the key's scopes, your plan, quotas, and rate
limits apply as they do there. The server keeps no session. Every request is one JSON-RPC message with
its own credentials, and there is no `Mcp-Session-Id` and no event stream (`GET /mcp` is `405`).

Each tool answers JSON. A refused call is a tool error with the API's `code` and `message`, for example
`missing_scope` and the scope it needs. Every tool and its scope is under [Reference](#reference) below.

## Read these docs over MCP without signing in

`https://instabox.sh/docs/mcp` is a public MCP server for these docs, so an agent can read them before
anything is installed. It needs no account, and it has no access to your machines.

```bash Claude Code theme={null}
claude mcp add --scope user --transport http instabox-docs https://instabox.sh/docs/mcp
```

```bash Codex theme={null}
codex mcp add instabox-docs --url https://instabox.sh/docs/mcp
```

```json Cursor theme={null}
// ~/.cursor/mcp.json
{"mcpServers":{"instabox-docs":{"url":"https://instabox.sh/docs/mcp"}}}
```

```json OpenCode theme={null}
// opencode.json
{"mcp":{"instabox-docs":{"type":"remote","url":"https://instabox.sh/docs/mcp"}}}
```

Its tools: `search_insta_box` searches the docs, `query_docs_filesystem_insta_box` runs read-only
queries over the pages and OpenAPI specs, and `submit_feedback` reports a problem with a page. The
server speaks Streamable HTTP. `https://instabox.sh/mcp` is the same server.

## Reference

### Tools and scopes

| Tool | Does | Scope |
| - | - | - |
| `list_machines`, `get_machine` | your machines, or one by ID or name | `machines:read` |
| `create_machine` | from a template (default `agent-browser`) or snapshot, waits until it runs | `machines:write` |
| `exec` | `argv` (no shell) or `cmd` (`bash -lc`), `timeout_seconds` up to 600. A non-zero exit is a result. | `machines:write` |
| `read_file`, `write_file` | up to 1 MiB. Text as UTF-8, other bytes as base64. | `machines:write` |
| `suspend_machine`, `resume_machine` | pause and resume, memory kept. They wait for the operation. | `machines:write` |
| `save_snapshot`, `list_snapshots` | save a machine's disks, list snapshots | `images:write`, `images:read` |
| `delete_machine` | deletes it for good. `confirm` must be its ID or name. | `machines:delete` |
| `get_usage` | counts, running time and limits (or one machine's) | `machines:read` or `images:read` |

A tool given a name instead of an ID looks the machine up first, which needs `machines:read`.

> [accordion] Waits and read sizes
>
> Lifecycle tools wait up to 120 s by default. `wait_seconds` is at most 600: a larger value waits 600,
> and 0 or less means the default. `wait: false` returns the operation at once.
>
> `read_file` reads up to `max_bytes`, 256 KiB by default and at most 1 MiB. A larger value is taken as
> 1 MiB, and 0 or less as the default. For files over 1 MiB, use `instabox cp`.

> [card] [API keys](https://instabox.sh/docs/security/api-keys.md)
>
> Scopes for the key your MCP client sends.

> [card] [Coding agents](https://instabox.sh/docs/use-cases/coding-agents.md)
>
> Give each agent a machine of its own.
