# InstaBox Clone — 技术架构

> 版本：v1 · 2026-11-17
> 目标：在单机（macOS Apple Silicon）上复刻 InstaBox 的核心能力：真正的 Linux 沙箱生命周期、终端、文件、端口、快照。

## 1. 总体架构

```
┌─────────────────────────── macOS 宿主机 ───────────────────────────┐
│                                                                    │
│  ┌─────────────┐   HTTP/WS    ┌─────────────────────────────────┐ │
│  │  Web Dashboard│ ◄────────► │        Control Plane (Go)       │ │
│  │  Next.js :3000│             │  :8080  api + :8081 ports proxy │ │
│  └─────────────┘              │                                 │ │
│                               │  · REST API /v1/*               │ │
│  ┌─────────────┐              │  · Session + API Key 认证        │ │
│  │ CLI / SDK    │ ◄────────►  │  · 状态机 + Operation 队列       │ │
│  └─────────────┘              │  · 幂等 / 删除确认 / 配额        │ │
│                               │  · SQLite 持久化                 │ │
│                               │  · 指标采样调度 / 自动暂停        │ │
│                               └──────────┬──────────────────────┘ │
│                                          │ mTLS WebSocket tunnel │
│  ┌───────────────────────────────────────┴──────────────────────┐│
│  │              Node Agent (Go, 运行 Lima Linux VM 内)           ││
│  │  · containerd 沙箱管理（gVisor runsc / runc）                  ││
│  │  · exec / PTY terminal / 文件读写                              ││
│  │  · runsc checkpoint/restore 实现 Pause/Resume                  ││
│  │  · 根盘 + /data 卷 / 快照（本地 CAS 存储）                     ││
│  │  · cgroup 指标采集                                            ││
│  └───────────────────────────────────────────────────────────────┘│
└────────────────────────────────────────────────────────────────────┘
```

**控制平面与执行平面解耦**：Control Plane 不直接碰容器运行时；所有沙箱操作通过 Node Agent 的 RPC 执行。Control Plane 可连接多个 Node Agent（水平扩展预留）。

## 2. 为什么这样选型

| 决策 | 选择 | 原因 |
|---|---|---|
| 宿主机无 Docker | Lima (vz, Apple Silicon) | macOS 上运行 Linux VM 的最轻量开源方案；brew 可装；支持 Rosetta |
| 沙箱隔离 | gVisor (runsc) + runc 双运行时 | 任务要求"不把 Docker 容器当强安全边界"。runsc 拦截系统调用，是用户态内核沙箱，支持 **checkpoint/restore**（Pause 保留内存进程的关键）；模板/自定义镜像走 runc 兜底 |
| 暂停/恢复 | `runsc checkpoint --state-file` | 真实保留进程内存现场，恢复失败回退 fresh boot（与原产品语义一致） |
| 控制平面语言 | Go | 单二进制、并发模型适合长连接代理；与原产品 API 语义对齐 |
| 数据库 | SQLite (modernc.org/sqlite, 纯 Go) | 单机交付零依赖；WAL 模式支持并发读；schema 与 Postgres 兼容，文档中说明迁移路径 |
| 前端 | Next.js 14 + TypeScript + Tailwind + shadcn/ui 风格 | 任务指定；还原原产品暗色控制台风格；xterm.js 终端 |
| 终端 | WebSocket + containerd exec + PTY | 无 Docker socket 暴露；经 Agent 的 PTY 流转发 |
| 端口发布 | Control Plane 内嵌反向代理 | `{port}-{machine}.localhost` 泛域名/Host 路由到 Agent 端口转发 |
| 快照/磁盘 | 本地 CAS 目录（内容寻址 tar.zst） | 复刻原产品 sha256 digest 模板与逻辑/存储字节去重语义；ext4 镜像文件 + 写时复制用 overlay 不可取（rootfs 需完整 VM 语义），采用**独立磁盘镜像文件 + 挂到沙箱** |

## 3. 模块职责

### Control Plane (`control/`)
- **API Server**：REST `/v1/*`（与调研的 InstaBox API 语义对齐的子集）
- **Auth**：session cookie（dashboard 登录，预置本地用户）+ API Key（`ibk_` 前缀 + scope）
- **Machine Controller**：`desiredState`/`state` reconciliation；Operation 表（create/start/stop/suspend/resume/delete/fork/restore/snapshot）串行化执行（每机一次一个操作）
- **Quota**：免费计划 5 机 / 5 快照 / 200GiB / 16 端口
- **Auto-suspend**：空闲检测（exec/文件/端口流量/终端/CPU 活动心跳），30 分钟默认；max-run 免费计划 120→480 分钟增长 + cooldown
- **Metrics collector**：每 30s 向 Agent 拉 cgroup 采样 → `metrics` 表
- **Port proxy**：按 Host `{port}-{machineId}.ibx.localhost` 反代至 Agent
- **Event/webhook 分发**（P1）

### Node Agent (`agent/`)
- 注册到 Control Plane（节点 ID、容量）
- Sandbox 运行时：containerd + runsc；每沙箱 = 独立 container + 独立 rootfs 磁盘镜像 + /data 卷
- 操作执行器：create/start/stop/**suspend(checkpoint)**/**resume(restore)**/delete/fork（本地 CoW 复制）/snapshot（CAS 导出）
- Exec（非交互 + PTY 交互）、文件读写（经 exec `cat`/`tar` 协议或 9P——实现采用 exec 管道 + 结构化协议，避免挂载）
- 端口转发：沙箱端口 → Agent 监听端口（供 Control Plane 反代）
- 指标：cgroup v2 统计

### Web Dashboard (`web/`)
- 页面：Machines（列表/创建对话框/详情 Dialog 七 Tab）、Snapshots、Secrets、API Keys、Usage、Settings
- 终端：xterm.js ↔ WebSocket
- 状态轮询 + 乐观更新；操作进度 toast（"Operations in progress" tray）

### Guest 内体验
- 模板 rootfs 内置 `ibox` guest agent 元数据（`/run/ibox/env/env` 等），env 变量按原产品语义注入

## 4. 核心状态机

```
                 create
                  │ op: create
                  ▼
              CREATING ───────► FAILED
                  │ succeeded
                  ▼
   start/resume   RUNNING ◄────────────────┐
   ┌──────────────┤  ▲ ▶ op: stop          │ wake (auto-suspended)
   │              │  │                     │ op: resume
   ▼              │  │ suspend(=pause)     │
  STARTING        │  ▼ op: suspend        │
   │              │ SUSPENDED ────────────┘
   └──────────────┤  │  (memory+processes on disk; checkpoint)
                  │  ▼ resume
                  │ RESUMING ──► (checkpoint 损坏) ──► STARTING(fresh)
                  │
                  ▼
              STOPPED ──start──► STARTING
                  │
              delete (X-Confirm-Delete)
                  ▼
               DELETED
```

- `desiredState` ∈ {running, stopped, suspended, deleted}；`state` 为实际观测状态
- 每机串行操作队列；操作 = {id, machineId, kind, status: pending|running|succeeded|failed, error, startedAt, finishedAt}
- 免费计划 max-run：120min 起，使用中按 [5min→+120, 1-4min→+30] 增长至 480min；到时 pause + cooldown（5-20min 随机；连读加倍）

## 5. 存储设计

### 控制平面（SQLite `control/ibox.db`）
表：`users`, `orgs`, `api_keys`, `machines`, `operations`, `snapshots`, `ports`, `secrets`, `metrics`, `activity_events`, `exec_history`, `webhooks`(P1)
（完整 schema 见 docs/DATA_MODEL.md）

### 执行平面（VM 内 `/var/lib/ibox-agent/`）
```
/var/lib/ibox-agent/
├── rootfs/                    # 模板与快照 CAS（内容寻址）
│   └── sha256/<aa>/<bb>/<digest>.tar.zst
├── machines/<machineId>/
│   ├── root.ext4              # 根盘（模板解包或快照克隆）
│   ├── root.ext4.meta
│   ├── data.ext4              # /data 持久盘（20GiB 稀疏）
│   ├── checkpoint/            # runsc state files（暂停时）
│   └── oom / logs
└── tmp/                       # fork 暂存
```
- 快照 = 停止写盘后把 root+data 打包 zstd → CAS；秒级 fork = 同一文件系统 reflink 复制（APFS/XFS CoW）或硬链接快照目录
- 与原产品一致：`storage.synced` 模拟（快照/暂停完成后置 true）

## 6. 安全设计

- runsc 系统调用沙箱（默认）；显式 `runtime: runc` 仅用于可信模板
- 沙箱内进程以非 root `user` 运行（guestUser）；root 操作走 Agent 侧
- CPU/内存/pids/磁盘 cgroup v2 限额；exec 默认 60s 超时 / 600s 上限；网络默认**仅经 Control Plane 出站**（VM slirp 无端口暴露，入站只能走端口发布）
- Control Plane ↔ Agent：共享密钥 + HMAC 消息签名（本机部署简化为 bearer + origin 校验）
- API key 只存哈希；SSH token/端口 token 只存哈希（与原产品一致）
- 不暴露宿主机 Docker socket；Agent 只监听 Control Plane 可达地址

## 7. 部署

见根目录 README：单命令 `make up`（启动 VM + control + agent + web）。分步 `make vm / make agent / make control / make web`。
