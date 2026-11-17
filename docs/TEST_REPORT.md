# ibox E2E 测试报告 (Phase 5)

日期：2026-11-17
环境：macOS (Apple M4) · Lima VM "ibox" (Ubuntu 24.04 aarch64, 6 vCPU / 10 GiB) ·
gVisor `runsc release-20261116.0` · 控制平面 `go run ./cmd/control` · SQLite (WAL)

测试方法：`./scripts/e2e.sh` — 对真实部署执行 15 步全生命周期验证，
每一步通过 REST API / WebSocket / SSE 断言真实结果（无 mock）。

<!-- RESULTS:FILLED-BY-RUN -->

## 最终结果（2026-11-17 23:38–23:39 本地运行）

```
E2E RESULT: 31 passed, 0 failed   (exit 0)
```

完整原始日志：`/tmp/ibox-e2e-run.log`（tee 自 `./scripts/e2e.sh`）。
本次运行前状态：0 machines、0 runsc 容器；运行后复核：0 runsc 容器残留、
机器列表为空 —— 删除路径的物理清理验证通过。

| # | 步骤 | 断言 | 结果 |
|---|------|------|------|
| 1 | 登录 | session cookie + CSRF token | PASS ×2 |
| 2 | 创建机器（异步 202） | 返回机器 id、`creating` 初始状态、重名 409 `already_exists` | PASS ×3 |
| 3 | 等待 running | 状态机 create→start→running | PASS |
| 4 | 沙箱内环境 | `E2E_MARKER` 注入、guest 用户 `user`（uid 1000）、hostname=机器名、exit 0 | PASS ×4 |
| 5 | 文件写入/读取 API | PUT 204、内容回读一致 | PASS ×2 |
| 6 | 目录列举 | `/data/e2e` 列表含 `hello.txt` | PASS |
| 7 | exec 与文件 API 互通 | exec `cat` 读到 API 写入内容 | PASS |
| 8 | Web Terminal | WS 握手、resize 帧、键击帧、真实 PTY 回显 `term-42-ok` | PASS |
| 9 | CPU/内存/metrics | `MemTotal`=8 GiB、`nproc`=4（cpuset 生效）、metrics API 返回采样点 | PASS ×3 |
| 10 | 停止 | `stopped`（gVisor 进程真实被杀） | PASS |
| 11 | 再次启动 | `running`（全新 `runsc run`） | PASS |
| 12 | /data 持久化 | 停机/启动后文件内容不变 | PASS |
| 13 | 快照/修改/恢复/回滚 | 快照创建、恢复受理、回滚后内容为快照版本 | PASS ×3 |
| 14 | 暂停（真 checkpoint）/恢复 | `suspended` + `processesPreserved=true`（`runsc checkpoint --image-path`），恢复后 running | PASS ×3 |
| 15 | 删除+清理 | 202、GET 404、列表为空、节点 0 容器残留 | PASS ×3 |

合计 **31 项断言全部通过**。

### 测试过程中修复的真实缺陷（均由 E2E 暴露）

1. `runsc destroy` 不存在（应为 `delete -force`）→ 所有 stop/delete 留下僵尸
   gVisor 进程；恢复后旧容器被"收养"导致文件系统为空。
2. `runsc exec` 无 `-t` 旗标 → Web Terminal 不可用；改为 PTY slave 作 stdio。
3. 流多路复用数据帧在顶层 JSON，Agent 误从 `params` 解析 → 键击全部丢弃。
4. OCI `cpu.cpus` 是 cpuset 列表而非数量：`"4"` 把沙箱钉在单个宿主机 CPU。
5. 快照 tar 未排除 `.gvisor.*` filestore 元数据 → 恢复后 `runsc run` 拒绝启动。
6. `runsc checkpoint --state-file` 旗标不存在 → 暂停静默降级为 freezer；
   改为 `--image-path`（真 suspend-to-disk），restore 同样修正。
7. detached `runsc restore` + `CombinedOutput` 管道死锁（与 start 同类）。
8. `/data` 属主为 root → guest 无法写；创建/恢复时 chown 1000:1000。
9. 控制面 `files.list` 解码形状错误、`machines`/`metrics` 空列表序列化为
   `null`、metrics 响应字段与 UI 契约不符、DELETE 返回 204 而非 202、
   hostname 用了机器 id 而非名字。


## 未实现能力（诚实标注）

以下原始产品能力在本复刻中**未实现**，测试不涉及：

| 能力 | 状态 | 说明 |
|---|---|---|
| Desktop / VNC | 未实现 | 原产品 Pro 功能 |
| SSH 网关 | 未实现 | terminal/exec 已覆盖远程使用场景 |
| Webhooks | 未实现 | — |
| 文件 upload-url | 存根 | 返回 `not_implemented`，UI 走 files.write 直传 |
| Stripe 计费 | 未实现 | 仅免费版配额逻辑 |
| x86_64 模板 | 不适用 | 复刻为 aarch64-only |

## 已知限制

- `network=host` 为 Lima 单机开发模式；生产设计应使用独立 netns + 出向代理
  （架构文档有详细说明）。
- 节点 RPC 超时后操作需等待 stale 阈值（5 分钟）才能被 reconcile 重试 ——
  控制面重启场景已验证可自愈。
