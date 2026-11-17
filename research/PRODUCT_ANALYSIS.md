# InstaBox Sandbox 产品考古报告

> 调研日期：2026-11-17 · 调研方式：Ego Lite 浏览器实际登录操作 + 官方文档全量下载（98 篇）+ OpenAPI 规范（132 路径）+ 真实 API 响应采样
>
> 证据留存：`research/screenshots/`（页面截图）、`research/pages/`（文档原文 + 页面文本）、`research/data-model/`（API 真实响应 + openapi.json）
>
> 信息分级：【事实-官方】= 文档明确披露；【事实-观察】= 页面/接口行为观察到；【推断】= 工程经验推断，非官方确认

---

## 1. 产品定位

【事实-官方】InstaBox 是"给每个 Agent 一台云端 Linux 机器"的 Sandbox 平台（"A Linux machine of its own in the cloud for every agent"）。一台 Machine 是**一台完整的 Linux VM**：自己的内核、root 文件系统、Docker，文件在 `/data`（home 为 `/data/home`），安装的软件在暂停/停止后保留。

核心场景：编码 Agent（Claude Code / Codex / OpenCode）、AI 生成代码的执行环境、并行 Fork 实验、Computer Use（远程桌面）、预览 URL、CI、Android 模拟。

## 2. 页面结构与导航

【事实-观察】Dashboard（dashboard.instabox.sh）布局：

```
顶栏: [Logo] [Org切换器: 邮箱+Personal] [Docs] [Rules] [账户菜单]
侧边栏:
  [Search ⌘K]
  Machines / Snapshots / Secrets / API Keys / Webhooks
  CI
  Members / Usage / Billing
  [Upgrade Plan]
```

页面清单（全部实际访问验证）：

| 页面 | URL | 内容 |
|---|---|---|
| Machines | `/dashboard/machines` | 机器表格 + Create Machine 对话框 |
| Snapshots | `/dashboard/snapshots` | Official/Personal 两个 tab，模板+我的快照 |
| Secrets | `/dashboard/secrets` | 0/100 secrets used，Add Secret |
| API Keys | `/dashboard/api-keys` | 密钥列表（名称/前缀/权限/创建/最近使用/过期） |
| Webhooks | `/dashboard/webhooks` | 1 of 10 endpoints，Add endpoint |
| CI | `/dashboard/ci` | 3 步引导：Connect GitHub → runs-on: instabox-medium → run |
| Members | `/dashboard/members` | 个人 org 说明 + Create a workspace |
| Usage | `/dashboard/usage` | 账期、计划、用量统计 |
| Billing | `/dashboard/billing` | 订阅计划管理 |

机器详情不是独立路由，而是 `?open=m_<id>` 打开的**全屏 Dialog**（【事实-观察】）：

```
┌──────────────────────────────────────────────┐
│ halo-dsh                    [Save snapshot] ✕│
├──────────────────────┬───────────────────────┤
│ TERMINAL             │ [Computer|Details|    │
│ [canvas terminal]    │  Metrics|Ports|       │
│ [Restart][Hide]      │  Filesystem|          │
│                      │  Integrations|Settings]│
├──────────────────────┴───────────────────────┤
```

## 3. Machines 列表页（已截图 01）

- 标题 `Machines (4/5)` —— 当前数/计划上限
- 表格列：checkbox（批量选择）、NAME（可排序）、STATE（可排序）、SOURCE（模板名或 Snapshot）、RESOURCES（vCPU/GiB/GiB 图标组）、CREATED（相对时间）、ACTIONS（Pause/Resume + More）
- 行内 More 菜单：Details / Rename / Save snapshot / Restart / Auto-pause（开关）/ Delete
- 底部提示：`Free plan: never billed; pauses after 30 idle minutes or 8 running hours`
- 升级卡片：`Need more machines? Pro: up to 500 machines with up to 8 vCPUs each`

## 4. Create Machine 对话框（已截图 02）

字段（全部可选，`{}` 即合法请求 —— 【事实-官方】）：

| 字段 | 类型 | 默认值 | 约束 |
|---|---|---|---|
| Name | string | `box-` + 8位随机hex | 1-63位 `a-z0-9-`，字母或数字开头 |
| Snapshot（模板/快照选择器） | string | `standard`（实际默认模板 agent-browser） | creatable=true 的模板或自己的快照 |
| Resources | enum | medium | small(2/4/20) medium(4/8/20) large(8/16/20)；small/large 标 "Get Pro to Unlock" |
| Auto-pause | switch | 开（30分钟） | 文案 "After 30 minutes idle" |
| Environment variables | list + .env 导入 | 无 | 提示 "Set when the machine is created. Anything in it can read them, so use Secrets for API keys." |

- 底部显示 `4 of 5 machines`，Cancel / Create
- 模板下拉分组：Template（android / omarchy / standard）、My Snapshots、Get Pro to Unlock（GPU (Linux) / minimal / macOS 26 Tahoe）

## 5. 机器详情页各 Tab

### 5.1 Computer（远程桌面）
【事实-观察】全屏桌面画布 + "Click the desktop to take control; it goes back to the agent after a minute idle."。Fullscreen 按钮。桌面经 `GET /v1/machines/{id}/desktop/watch`（SSE）+ `GET /v1/machines/{id}/vnc`（WebSocket）。

### 5.2 Details
字段：State（Running + Pause 按钮）、Name、Source template、User（`user`）、Auto-pause（pauses after 30m idle）、Max runtime（2 hours, grows with use, up to 8 hours）、Run started、Cooldown until、Created、RESOURCES、SSH（Create token 按钮）。

### 5.3 Metrics
时间范围 1h / 24h。四张图：CPU（%）、Memory（GiB）、Network（In/Out B/s）、Disk I/O（Read/Write B/s）。数据源 `GET /v1/machines/{id}/metrics?since=...`。注：`Host samples every 30s. Network excludes SSH and public ports.`

### 5.4 Ports
- Publish port 按钮；Public ports 表格：PORT / URL / ACTIONS（Edit / Unpublish）
- URL 格式：`https://{port}-{machineId去掉m_前缀}.instabox.tech`
- 私有端口：名称旁标注，注文案 "A private port answers only the link printed by `instabox port publish --private` (its _token); --rotate-token prints a new link."

### 5.5 Filesystem
- 实时浏览运行中机器的**整个根文件系统**（以 root 读取）：NAME / SIZE / MODIFIED，目录与符号链接（`bin → usr/bin`）可辨
- 注：`Read live from the running machine, as root. Home and the workspace are under /data.`
- 数据源 `GET /v1/machines/{id}/files?path=/`

### 5.6 Integrations
Paradev / cmux / Orca / T3 Code 四个第三方 ADE 集成入口。

### 5.7 Settings
- NAME（Rename）、RESOURCES 展示
- 磁盘说明：`A free sandbox's disk is fixed at 20 GB... Paid sandboxes... $0.08 per GB-month ($0.00011 per GB-hour)`
- DANGER ZONE：Delete this machine

### 5.8 Terminal
画布终端（xterm.js），Restart/Hide 按钮，自动重连。数据源 `GET /v1/machines/{id}/terminal`（WebSocket）。

## 6. 其他页面交互

**Secrets**：`0/100 secrets used`。Add Secret 对话框：Name、Allowed hosts（逗号分隔，留空=所有目的地）、Value（"Values are never read back"）。
**API Keys**：Create Key 对话框：Key Name、Expires（可选日期，最长一年，无=永不过期）、Permissions（Full access / Restricted）。
**Webhooks**：`Signed HTTPS events when sandboxes and snapshots change. 1 of 10 endpoints`。
**CI**：3 步引导卡。

## 7. 生命周期与状态机

【事实-官方】状态：`Creating / Starting / Running / Stopping / Stopped / Pausing / Paused / Resuming / Deleting / Failed`（API 用 `suspended/suspending`，Failed 用 `error`）。

| 状态 | 文件 | 内存与进程 | 计费 |
|---|---|---|---|
| Running | 保留 | 保留 | 是 |
| Paused | 保留 | **保留**（暂停到磁盘） | 否 |
| Stopped | 保留 | 丢失（下次启动全新引导） | 否 |
| Deleted | 删除 | 删除 | 否 |

- **Pause = microVM 挂起到磁盘**。Resume 恢复现场；若内存无法恢复，"starts fresh with its files"（【事实-官方】）
- **自动暂停**：空闲 30 分钟（免费计划固定；付费 5-30 分钟可调或 0=关闭）。空闲=无 Agent 活动+无终端/SSH 流量+CPU 无负载。"信号缺失或不清晰时计为 busy"
- **唤醒**：exec/文件读写/SSH/公网端口请求/桌面/终端都会唤醒自动暂停的机器，等待最多 45s；手动暂停的不唤醒
- **最大运行时长**：免费计划每 run 120 分钟起，使用中按规则增长到 480 上限（最后30分钟内使用≥5分钟→+120；1-4分钟→+30；0→到时暂停）。冷却：第1次 5-20min、连读第2次 60-75min、第3次+ 120-135min（基础+0-15min 随机）
- **Activity lease**：`PUT /v1/machines/{id}/activity-leases/{taskID}` ttlSeconds 30-3600，保活
- **停止/暂停后存储同步**：磁盘状态后台上传对象存储，`storage.synced=false` 期间只能回原主机启动
- **archiveSeconds / deleteAfterSeconds**：60-86400，到期自动停止/删除（CI/Agent 防泄漏）

## 8. 数据模型（来自真实 API 响应）

### Machine（`research/data-model/api-responses.json`）
```json
{
  "id": "m_agqyxark46f72uak3gv2b5rmee",       // m_ + ULID
  "name": "halo-dsh",
  "desiredState": "running", "state": "running",
  "revision": 65,                              // 乐观并发控制
  "operationId": null,
  "template": {"id": "agent-browser", "version": "2026-10-29.18", "digest": "sha256:..."},
  "spec": {"vcpus": 4, "memMiB": 8192, "persistentDiskGiB": 20, "rootDiskGiB": 16},
  "storage": {"sizeGiB": 20, "mountPath": "/data", "persistent": true, "synced": true, "snapshotSeq": 32},
  "runtime": {"ready": true, "processesPreserved": true},
  "autoSuspendAfterSeconds": 1800, "maxRunMinutes": 120,
  "runStartedAt": "...", "readyAt": "...", "cooldownUntil": "...",
  "metadata": {}, "vault": true, "guestUser": "user",
  "orgId": "org_...", "free": true, "oomKills": 0,
  "createdAt": "...", "updatedAt": "..."
}
```
状态字段推断：`desiredState`（意图）与 `state`（实际）分离 = 典型的 **reconciliation 控制回路**（【推断】类似 Kubernetes controller 模式）。

### Snapshot
```json
{"id": "sn_...", "name": "1109-01", "sourceMachineId": "m_...", "sourceSnapshotSeq": 1,
 "template": {...}, "rootSizeBytes": 17179869184, "dataSizeBytes": 21474836480,
 "defaultSpec": {...}, "status": "ready", "guestUser": "user"}
```
**Live snapshots**：运行中机器可直接快照（`liveSnapshots: true`），GPU 机器除外。

### Template（内容寻址镜像）
```json
{"id": "universal", "version": "2026-10-12.1", "digest": "sha256:...",
 "description": "gh, Node.js, Bun, pnpm..., Docker, Python (uv), Go, Rust, JDK 21, C toolchain",
 "arch": "x86_64", "defaultSpec": {"vcpus": 4, "memMiB": 8192, "persistentDiskGiB": 20, "rootDiskGiB": 16},
 "capabilities": ["identity-reset", "root-grow", "activity", "guest-user"], "creatable": true, "latest": true}
```
- 根文件系统为 **OCI 式 sha256 digest**；omarchy 模板额外带独立 `kernel` digest → 独立内核镜像
- 能力标签：`identity-reset`（机器重建时重置身份）、`root-grow`（根盘在线扩容）、`activity`（活动检测）、`guest-user`、`orca`

### Port
```json
{"id": "pv-...", "machineId": "m_...", "port": 3000, "name": "MyGo Cloud Music Web",
 "protocol": "http", "url": "https://3000-<id>.instabox.tech",
 "access": "public|private", "expiresAt": null}
```
私有端口 token `ibp_...` 只显示一次（服务端只存哈希），经 `?_token=` 或 `Authorization: Bearer` 或 Cookie `__Host-instabox_port_auth` 鉴权。每机最多 16 个端口，1024-65535（5901 VNC、6768 Orca 保留）。TCP raw 端口为预览功能。

### Usage / 配额
`logicalBytes` vs `storedBytes`（**去重/压缩**，200GiB 逻辑→30GiB 存储）→【推断】使用内容寻址块存储（如 restic/zstd chunking）。成本核算：vcpuHours / memGiBHours / storageGiBHours / egressGiB × 单价。限额：免费版 5 机器 / 5 快照 / 200GiB / 5 images / vault 100 项 256KB。

### 认证与错误
- API key `ibk_` + 52 字符；access token `iba_`；Header `X-API-Key` 或 `Authorization: Bearer`；受限 key 带 scopes
- Dashboard 会话：HttpOnly Cookie + 写操作带 `x-instabox-csrf` 头（【事实-观察】）；登录走 WorkOS
- 统一错误：`{error: {code, message, requestId, retryable, operationId, hint?, docs_url?, retryAfterSeconds?}}`，分支按 `code` 不按 `message`
- 幂等：`Idempotency-Key` 必填（变更类），保留 7 天；同 key 不同请求=409
- 删除确认：`X-Confirm-Delete: <id或name>`，否则 409
- 计划：Pro $20/月(500机/8vCPU/200GiB)、Builder $100(16vCPU)、Team $500(32vCPU)；磁盘 $0.08/GB-月

## 9. API 清单（132 路径，完整见 research/data-model/openapi.json）

核心分组：
- **auth**：WorkOS/GitHub/Google OAuth、CLI device flow、agent sign-up（`/v1/auth/agent*`）
- **machines**：CRUD + `start/stop/suspend/resume/restart(隐含)/wake/fork/restore/exec/files/ports/terminal/vnc/desktop/metrics/logs/history/processes/usage/disk/snapshots/ssh-tokens/connections/secrets/vault/activity-leases/oom`
- **operations**：`GET /v1/operations/{id}` 异步操作跟踪（202 Accepted + 轮询）
- **snapshots / images**：CRUD + 文件浏览（快照可离线浏览文件！）
- **ports**：`GET /v1/ports` 全局列表
- **orgs**：members/invites/keys/secrets/webhooks/billing/github
- **usage / stats / exports / support**
- **templates**：版本化模板列表

关键端点行为：
- `POST /v1/machines/{id}/exec`：同步执行（默认 60s 超时，最长 600s，超时 kill 退出码 124）；`Accept: text/event-stream` 时流式输出无 1MiB 截断；后台任务（job）无时限
- `GET/PUT /v1/machines/{id}/files`：目录列表 / 原始字节读写；`parents=true` 创建父目录；签名上传 URL
- `GET /v1/machines/{id}/terminal`、`/vnc`：WebSocket
- `GET /v1/machines/{id}/metrics?since=`：30s 采样的 CPU/内存/网络/磁盘序列

## 10. SDK / CLI

- **TS SDK** `@insforge/instabox-sdk`：`Sandbox.create({...})` / `Sandbox.connect(name)` / `sbx.commands.run` / `sbx.files.*` / `sbx.pause()` 等
- **Python SDK** `instabox`：`Client().machines.create/exec/pause/...`
- **CLI** `instabox`：create/run/exec/ssh/ssh-token/port/push/pull/cp/snapshot/fork/restore/resize/logs/jobs/history/ps/templates/workspace/ci
- **MCP server**：Claude Code/Cursor/Codex 的 machine tools
- **in-guest 工具**：`instabox-port`（端口发布）、guest agent（env 注入 `/run/instabox/env/env`）

## 11. 技术架构推断

【事实-官方】
- Machine 是有自己内核的 Linux VM（"its own kernel"），带 Docker
- 模板 = sha256 内容寻址的根文件系统镜像 + 可选独立内核镜像
- Pause 保留内存与进程（挂起到磁盘）；Resume 失败时 fresh start
- gVisor 语义特征：GPU 机器的 `/proc/meminfo` 行为（`MemAvailable == MemFree`、映射的文件页计入 used）是 **gVisor 的典型行为**（【事实-官方】文档描述 + 【推断】gVisor）
- 磁盘后台上传对象存储；storage.synced 标志；逻辑字节 vs 存储字节去重
- 无 region 字段，平台决定调度位置

【推断】（基于工程经验，非官方确认）
- **控制平面**：Go 后端（API 错误格式、operation 模式、ULID ID 风格）；单体重 API `api.instabox.sh` + dashboard 前端（React SPA，shadcn/ui 组件风格）
- **执行平面**：物理机 + microVM 技术栈。支持挂起/恢复内存、独立内核、x86_64 + macOS（Tahoe）模板 → 主机池混合 Linux 服务器（KVM/Firecracker 类）与 Mac 主机（Apple Virtualization.framework/Tart 类）。Linux 沙箱可能走 gVisor 提供"类 VM"语义 + 快速启动；需要完整内核的模板（omarchy）走 KVM microVM
- **存储**：写时复制根盘（rootDiskGiB 16G + persistentDiskGiB 20G 分离）+ /data 持久卷；快照 = 增量序列（snapshotSeq）；对象存储做冷备与跨主机迁移
- **网络**：网关型 —— SSH 证书（短-lived cert + `ibssh_` token）、VNC/Terminal WebSocket、HTTP 端口反代（`{port}-{id}.instabox.tech`）、出站 HTTPS 代理（Org Secrets 在代理层注入）
- **调度**：单一 org 内无 region 概念；fork 优先本机磁盘复制（"copies the disks on the machine's host"），不可用时回退快照；no_capacity 错误暗示共享主机池
- **隔离**：gVisor 沙箱（系统调用拦截）+ cgroup 资源限制 + 非 root guest user（`guestUser: user`）；env 变量明示"任何进程可读，不要放密钥"→ Secrets 走出站代理注入

## 12. 产品设计理念

1. **机器即环境**：不是"容器即沙箱"，而是"一台完整的 Linux 机器"，有内核、有 systemd、有 Docker、有桌面 —— Agent 可以把任何 Linux 工作负载扔进去
2. **以 Agent 为中心的设计**：CLI/SDK/MCP 一等公民；机器可自动暂停省钱、可 lease 保活、可 delete-after 防泄漏；exec 历史记录（审计 Agent 行为）
3. **暂停而非停止**：默认 30 分钟空闲自动挂起，内存现场保留，访问时自动唤醒 —— "服务器less 体验 + 有状态机器"的混合
4. **安全分层**：env（明文）→ Org Secrets（出站代理注入，机器内只见占位符）→ Direct Secrets；SSH 短证书；私有端口 token；删除需确认
5. **免费层获客**：5 台免费机器（4vCPU/8GiB），Pro $20/500 台 —— 定价以"机器数量"而非资源用量为锚

## 13. 复刻范围决策

完全复刻 132 个端点不现实；按任务 P0/P1/P2 分级实现（见 docs/ARCHITECTURE.md）。macOS 模板/GPU/WorkOS 登录/Stripe 计费属平台级能力，本地复刻以"明确标注未实现"处理。
