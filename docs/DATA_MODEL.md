# 数据模型（InstaBox Clone）

> SQLite（控制平面）。所有时间 RFC3339 UTC。ID 规范：machine `m_<ulid>`、snapshot `sn_<ulid>`、operation `op_<ulid>`、port `pv-<ulid>`、key `ibk_<rand52>`。

## ER 关系

```
users ─┬─< orgs (owner) ─┬─< machines ─┬─< operations
       │                 │             ├─< ports
       │                 │             ├─< metrics
       │                 │             ├─< exec_history
       │                 │             └─< snapshots (derived) ──< snapshot_files?(按需)
       │                 ├─< api_keys
       │                 ├─< secrets
       │                 └─< webhooks (P1)
       └─< sessions
```

## 表结构

### users
| 列 | 类型 | 说明 |
|---|---|---|
| id | INTEGER PK | |
| email | TEXT UNIQUE | 登录账号 |
| name | TEXT | 显示名 |
| password_hash | TEXT | bcrypt（本地部署预置 admin） |
| avatar_url | TEXT | |
| created_at | TEXT | |

### sessions
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `ses_<rand>` |
| user_id | INT FK | |
| token_hash | TEXT | HttpOnly cookie `ibox_session` 只存哈希 |
| csrf_token | TEXT | 写操作校验 `x-ibox-csrf` |
| expires_at | TEXT | |

### orgs
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `org_<32hex>` |
| name | TEXT | |
| kind | TEXT | personal/workspace |
| tier | TEXT | free/pro（本地默认 free，配额可配） |
| created_at | TEXT | |

### api_keys
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `key_<ulid>` |
| org_id | TEXT FK | |
| name | TEXT | |
| prefix | TEXT | `ibk_<8>` 展示用 |
| key_hash | TEXT | SHA256 |
| scopes | TEXT JSON | `["machines:read","machines:write"]` 等 |
| expires_at | TEXT NULL | |
| last_used_at | TEXT NULL | |
| created_at | TEXT | |
| revoked_at | TEXT NULL | |

### machines
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `m_<ulid>` |
| org_id | TEXT FK | |
| name | TEXT | 1-63 `a-z0-9-` |
| desired_state | TEXT | running/stopped/suspended/deleted |
| state | TEXT | creating/starting/running/stopping/stopped/suspending/suspended/resuming/deleting/error |
| revision | INT | 乐观锁，每次变更 +1 |
| operation_id | TEXT NULL | 当前进行中操作 |
| node_id | TEXT NULL | 所在 Agent |
| template_id / template_version / template_digest | TEXT | 来源模板 |
| snapshot_id | TEXT NULL FK | 来源快照（模板机为 NULL） |
| vcpus / mem_mib / root_disk_gib / persistent_disk_gib | INT | 规格 |
| guest_user | TEXT | 默认 `user` |
| env_names | TEXT JSON | 变量名列表（值加密存 env_blob） |
| env_blob | BLOB | AES-GCM 加密的 env JSON |
| auto_suspend_after_seconds | INT | 默认 1800 |
| max_run_minutes | INT | 免费计划 120→480 |
| run_started_at | TEXT NULL | |
| ready_at | TEXT NULL | |
| cooldown_until | TEXT NULL | |
| auto_suspended_at | TEXT NULL | 非空=自动暂停（可唤醒） |
| storage_synced | INT | 0/1 |
| snapshot_seq | INT | 快照代数 |
| metadata | TEXT JSON | labels |
| vault | INT | 0/1 |
| free | INT | 0/1 |
| oom_kills | INT | |
| archive_at / delete_at | TEXT NULL | 截止时间 |
| created_at / updated_at | TEXT | |

索引：`org_id+name UNIQUE`、`org_id+state`

### operations
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `op_<ulid>` |
| machine_id | TEXT FK | |
| kind | TEXT | create/start/stop/suspend/resume/delete/fork/restore/snapshot |
| status | TEXT | pending/running/succeeded/failed |
| error | TEXT JSON NULL | `{code,message,retryable,retryAfterSeconds}` |
| idempotency_key | TEXT UNIQUE | |
| request | TEXT JSON | 原始请求（幂等重放） |
| result | TEXT JSON NULL | |
| started_at / finished_at | TEXT NULL | |
| created_at | TEXT | |

### snapshots
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `sn_<ulid>` |
| org_id | TEXT FK | |
| name | TEXT | |
| source_machine_id | TEXT | |
| source_snapshot_seq | INT | |
| template_id / template_version / template_digest | TEXT | |
| root_digest / data_digest | TEXT | CAS 内容寻址 |
| root_size_bytes / data_size_bytes | INT | |
| root_only | INT | |
| status | TEXT | saving/ready/error |
| guest_user | TEXT | |
| created_at | TEXT | |

### ports
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `pv-<ulid>` |
| machine_id | TEXT FK | |
| port | INT | 1024-65535 |
| name | TEXT | |
| protocol | TEXT | http/tcp |
| access | TEXT | public/private |
| token_hash | TEXT NULL | 私有端口 |
| created_at / expires_at | TEXT NULL | |

### secrets（Org Secrets，P1）
| 列 | 类型 | 说明 |
|---|---|---|
| org_id+name | PK | |
| allowed_hosts | TEXT JSON | 空=全部 |
| value_ciphertext | BLOB | AES-GCM |
| created_at / updated_at | TEXT | |

### metrics
| 列 | 类型 | 说明 |
|---|---|---|
| machine_id+sampled_at | PK | |
| cpu_pct | REAL | |
| mem_used_bytes | INT | |
| net_rx_bytes / net_tx_bytes | INT | 间隔增量 |
| disk_read_bytes / disk_written_bytes | INT | 间隔增量 |

### exec_history
| 列 | 类型 | 说明 |
|---|---|---|
| id | TEXT PK | `ex_<ulid>` |
| machine_id | TEXT FK | |
| cmd | TEXT | |
| cwd | TEXT NULL | |
| exit_code | INT NULL | |
| stdout_head / stderr_head | BLOB | 各 64KiB |
| started_at / finished_at | TEXT | |

### activity_events（保活/计费事实源）
| 列 | 类型 | 说明 |
|---|---|---|
| machine_id | TEXT | |
| kind | TEXT | exec/file/port/terminal/cpu |
| at | TEXT | |

## 状态机不变量

1. 每机同时最多一个未结束 operation（DB 唯一部分索引 `WHERE status IN ('pending','running')`）
2. `desired_state` 只能被 API 写入；`state` 只能被 Controller 根据 Agent 上报写入
3. suspended 机器拒绝 stop/delete 之外的变更（与原产品一致：先 resume）
4. delete 需要 `X-Confirm-Delete` 匹配 id 或 name
