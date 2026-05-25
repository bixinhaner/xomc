# T-0165 全系统设备唯一标识切换 OUI+SN（A-D 四 sub-task）总实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把全系统设备唯一标识从 `device_id (uuid)` 切换为 TR-069 标准的 `(oui, serial_number)` 双键；devices 表去掉 `PARTITION BY LIST (carrier)` 并加 `UNIQUE (oui, serial_number)`；30+ 张业务表（alarms / mr_records / device_parameters / ne_message_logs / config_backup_sub_tasks / device_tasks 等）的业务键改用 OUI+SN；事件总线 payload、REST API、前端契约统一切换。

**Why:**
- **TR-069 标准合规** — TR-069/CWMP Inform 自报 `(OUI, SerialNumber)` 作为 Device.DeviceInfo 的唯一身份；OMC 内部用 uuid 等于在所有南向接口再做一次"业务键 → 内部键"翻译，违背标准。
- **防止跨厂商 SN 撞键** — `serial_number` 在单厂商内唯一，跨厂商不保证；当前依赖 `(serial_number, carrier)` partial unique 兜底（migration 000089），但 carrier 仅作分区裂表用、不携带身份语义。oui 才是 TR-069 定义的厂商身份位。
- **避免上线后再大改** — 30+ 张表 + 667 处 Go 引用 + 413 处前端引用，上线后改造代价指数级上升；当前数据库都是 dev 数据可重灌。
- **简化分区策略** — 目前按 carrier 分区只产出 3 个分区且实测 100% 数据落在 `devices_cmcc`，分区无收益反而强制 PK 携带 carrier 列；去分区让 PK 回归单 uuid，业务键独立 UNIQUE(oui, serial_number) 约束。

**Architecture:**
- **业务键 vs 内部键分离**：`device_id (uuid)` **保留作 internal PK**（外键 / 内部数据完整性），新增 `UNIQUE (oui, serial_number)` 作为业务侧唯一索引和对外契约；业务表的"逻辑引用"列从 `device_id uuid` 升格为 `(oui, serial_number)` 双列。
- **devices 表去分区**：删除 `PARTITION BY LIST (carrier)` + 3 个子分区（cmcc/ctcc/cucc），重建为普通表；PRIMARY KEY 从 `(id, carrier)` 收回 `(id)`；carrier 字段保留（业务过滤用）。
- **事件总线 payload 双载**：T-0165-C 期间 SubjectXxx 事件 payload 同时携带 `device_id (legacy)` + `oui` + `serial_number`，subscriber 优先消费 oui+sn；待 D 段完成后下个 wave 删 device_id 字段。
- **公开契约一次切换**：T-0165-D 在 REST API + 前端契约同步切换，handler path 参数从 `:id` → `:oui/:serial_number`（或保留 `:id` 作 fallback 路由 + 新 `:oui/:sn` 路由两路并存一个 wave，给前端 deploy buffer）。

**Tech Stack:** Go 1.25 + pgx/v5 + Squirrel + TimescaleDB（hypertable + compression）+ PostgreSQL 分区表重建 + React 19 + TypeScript（前端契约切换）。

---

## 0. 关键文档与依赖

| 类型 | 路径 | 说明 |
|------|------|------|
| 决策依据 | TR-069 Amendment 6 §A.3.2.1 Device.DeviceInfo | OUI + SerialNumber 标准定义 |
| 前置依赖 | T-0164 全部完成（G1-G8 8 子任务全 done） | PM 流水线先内部切换 oui+sn（独立 fix commit），系统级切换在 T-0164 收尾后启动 |
| 关联设计 | `omcgo/migrations/000003_devices.sql` | 现 devices 表分区结构 + PK (id, carrier) |
| 关联设计 | `omcgo/migrations/000089_devices_partial_unique_serial_number.sql` | 现 `(serial_number, carrier)` partial unique（T-0165-A 后由 `(oui, serial_number)` 替代） |
| 现状调研 | Go: `DeviceID` 667 处 / `DeviceSN` 1064 处 | 业务层已大面积用 sn；切换主要是把"以 device_id 为业务键"的代码改回 oui+sn |
| 现状调研 | DB: 30+ 张表有 `device_id` 列 / 30+ 张表有 `device_sn` 列 | 很多表双键并存（alarms_active/alarms_history/config_backup_sub_tasks/device_tasks），本任务统一收口 |
| 现状调研 | 前端: 413 处 `device_id`/`deviceId` | T-0165-D 整批替换 |
| 真实数据 | 所有 device.carrier='cmcc' 默认值 | 分区表实际 100% 单分区 → 去分区零数据风险 |
| 真实数据 | OUI 字段多值（48BF74 / 00E0FC 等） | OUI+SN 双键能区分厂商 |

## 1. 实施顺序（依赖图）

```
T-0165-A (devices 表重构 — 去分区 + UNIQUE(oui,sn), L ~3-4d)
         │
         ▼  其他表的 FK / 业务键改造依赖 devices 新 UNIQUE 约束
T-0165-B (业务存储层切 OUI+SN — 30+ 张表 repository/model, L ~4-5d)
         │
         ▼  事件 payload 改造依赖业务层已切完
T-0165-C (事件总线 payload 统一, M ~2d)
         │
         ▼  公开契约最后切（给前端集中改 413 处的窗口）
T-0165-D (REST API + 前端契约切换, L ~3-4d)
```

**总工期估算**：~12-15 个工作日（按 1 人节奏）；A → B → C → D 严格顺序，每段独立合入 main 后再开下一段，避免长 PR 难以 review。

**关键约束**：
- 项目未上生产 → devices 表重建可"导出 → DROP → CREATE → 导入"硬切，**不做在线 zero-downtime 迁移**（用户偏好做减法）
- 每段独立 commit + 走 `/commit` skill；T-0165-A 是整个 task 的关键路径，卡 60+ min 无进展时跳到 B 不行（B 依赖 A），必须卡 A 解掉再下走

## 2. 文件结构总览

### 后端 (omcgo/)

**新建**：
- `omcgo/migrations/000172_devices_drop_partition_add_oui_sn_unique.sql` — T-0165-A devices 表重建（去分区 + UNIQUE(oui,serial_number)）
- `omcgo/migrations/000173_business_tables_add_oui_sn_cols.sql` — T-0165-B 批量给 30+ 张业务表加 `oui VARCHAR(6)` + `serial_number VARCHAR(64)` 列 + UNIQUE 索引（device_id 列保留作 internal FK）
- `omcgo/migrations/000174_business_tables_drop_legacy_device_id.sql` — T-0165-B 后期清理：业务表 device_id 列降级（仅用于内部 FK 关联；如果某些表完全不需要内部 FK，DROP 该列）
- `omcgo/migrations/000175_events_payload_oui_sn_doc.sql`（可能不需要 DDL，仅注释占位记录契约变更）— T-0165-C 文档性 marker

> 注：迁移版本号 172 是基于"T-0164 占用 164-171"假设；实际启动时按"现有最大版本号 +1"自查重排。

**修改**（关键模块）：

T-0165-A（devices 表）：
- `omcgo/internal/device/model.go` — Device 结构体保留 ID/OUI/SerialNumber，去掉对 `(id, carrier)` 复合主键的依赖
- `omcgo/internal/device/pg_repository.go` — 所有 `WHERE id=$1 AND carrier=$2` 改为 `WHERE id=$1`；新增 `GetByOuiAndSN(oui, sn)` 查询方法
- 所有引用 devices 复合 PK 的模块（grep `id, carrier` / `device_id, carrier`）配套改造

T-0165-B（业务存储层）：
- 30+ 张表的 repository / model 改造（按模块分批，每模块单独 PR）：
  - `omcgo/internal/alarm/` — alarms_active / alarms_history 业务键改 oui+sn
  - `omcgo/internal/pm/` — pm_metrics（T-0164-P3 已合并表）业务键改 oui+sn（注：T-0164 已在 PM 内部切，本段做收口验证）
  - `omcgo/internal/mr/` — mr_records 业务键改 oui+sn
  - `omcgo/internal/config/parammodel/` — device_parameters 业务键改 oui+sn（含 32 个 hash 分区）
  - `omcgo/internal/task/` — device_tasks 业务键改 oui+sn
  - `omcgo/internal/backup/` — config_backup_sub_tasks 业务键改 oui+sn
  - `omcgo/internal/syslog/` — ne_message_logs 业务键改 oui+sn
  - `omcgo/internal/acs/` — sessions / 心跳 / 协议日志业务键改 oui+sn
  - `omcgo/internal/stationlog/` — station_fault_logs 业务键改 oui+sn
  - 其他模块按 grep 结果列出

T-0165-C（事件总线 payload）：
- `omcgo/internal/events/handlers/*.go` — 所有 SubjectXxx 事件 payload 加 `Oui` + `SerialNumber` 字段
- `omcgo/internal/notification/` / `omcgo/internal/task/` 等 subscriber 改为优先消费 oui+sn

T-0165-D（REST API）：
- `omcgo/internal/device/handler.go` — handler 路由从 `/api/v1/devices/:id` 扩展为 `/api/v1/devices/:oui/:sn`，handler 函数从 `c.Param("id")` 改为 `c.Param("oui"), c.Param("sn")`
- 其他模块 handler 同步：`alarm/handler.go` / `pm/handler.go` / `mr/handler.go` / 等

### 前端 (omcmb/)

T-0165-D（前端契约切换）：
- `omcmb/frontend-core/src/services/api/*.ts` — 29 个 API 服务文件中所有 `device_id` / `deviceId` 字段替换为 `(oui, serial_number) / (oui, serialNumber)`
- `omcmb/frontend-core/src/types/device.ts` 等 — Device / 各业务实体类型字段重构
- `omcmb/frontend-core/src/hooks/api/*.ts` — useDeviceXxx 等 24 个 Hook 入参 / 查询键改造
- `omcmb/frontend-core/src/store/*.ts` — userStore / appStore / tabStore / taskStore / alarmStore 涉及设备引用的字段改造
- `omcmb/frontend-core/src/mock/*.ts` — Mock 数据形态对齐
- `omcmb/webcode/src/pages/**/*.tsx` — 主皮肤页面 / 组件中所有设备引用改造（413 处的主体）
- `omcmb/webcode-v2/` + `omcmb/webcode-v3/` — typecheck 验证不破

## 3. 全局验收

### 后端验收（每段 sub-task 自含验收，此处列跨段联动）

- [ ] `go build ./...` 通过（每段完成后）
- [ ] `go test ./...` 全过（每段完成后）
- [ ] `golangci-lint run` 0 警告
- [ ] migration 版本号严格连续递增（`bash omcgo/scripts/check-migrations.sh` 通过）
- [ ] migrate up + down + up 双向幂等（注意：A 段重建表 down 会丢数据，commit message 显式声明"项目未上生产，down 仅用于本地回滚"）
- [ ] E2E 验证：`bash omcgo/scripts/e2e_verify.sh http://localhost:8081` 全过；新加 `(oui, sn)` 路由的端点要补 e2e claim
- [ ] CPE 模拟器验证：`python3 omcgo/scripts/cpe_simulator.py` 注册一个 (oui=48BF74, sn=TEST001) 设备 → devices 表落库 → 各业务表（alarms / pm_metrics / device_tasks）写入时业务键 = oui+sn

### 前端验收

- [ ] `cd omcmb/webcode && npm run typecheck` 通过（T-0165-D 完成后）
- [ ] `cd omcmb/webcode-v2 && npm run typecheck` + `cd omcmb/webcode-v3 && npm run typecheck` 通过（多皮肤 baseline 不破）
- [ ] `npm run lint` 0 警告
- [ ] `npm run test` Vitest 单测过
- [ ] playwright MCP 自测：登录 → 设备列表 → 设备详情 → 告警查看 → PM 查询 → MML 命令下发，全链路 URL / API payload 用 oui+sn

### 跨段联动验收

- [ ] T-0165-A 完成后：psql 查 devices 表无分区（`\d devices` 无 Partitioned table 标记）+ UNIQUE 约束已建（`\d+ devices` 看到 `oui_sn_unique`）
- [ ] T-0165-B 完成后：30+ 张业务表的 repository 单测覆盖 `(oui, sn)` 查询路径
- [ ] T-0165-C 完成后：NATS 抓 SubjectAlarmRaised / SubjectDeviceRegistered 等 5+ 事件 payload，确认含 oui + serial_number 字段
- [ ] T-0165-D 完成后：浏览器开发者工具抓所有 /api/v1 请求，无 `device_id=uuid` 形式参数

## 4. 风险登记

| 风险 | 级别 | 缓解 |
|------|------|------|
| **devices 表重建期间停机** | 中 | 项目未上生产 → 直接 DROP + CREATE + 数据导入。生产部署时改为先建影子表 → 双写 → 切流量 → 弃旧表的标准模式（届时另立 task） |
| **30+ 张业务表 FK 级联影响面大** | 高 | A 段做之前先 `SELECT conname, conrelid::regclass FROM pg_constraint WHERE confrelid = 'devices'::regclass` 列全 FK；B 段每模块独立 PR + 单测 + e2e 覆盖；如某 FK 引用 devices(id, carrier) 复合键，A 段重建时一并修 |
| **A 段 down 不可逆** | 中 | 项目未上生产 + 重建表，down 段只能给出"重建分区表 + 数据回填提示"占位语义；commit message 显式声明 |
| **B 段 30+ 张表 migration 一次过大** | 中 | 按"alarms / pm / mr / device_parameters / task / backup / syslog / acs / stationlog" 分批，每批一个 migration + 一个 PR；总 migration 数 5-8 个（不全塞 173 一文件） |
| **C 段事件 payload 改造影响在跑的 NATS consumer** | 中 | 双载策略：payload 同时带 device_id + oui + sn，subscriber 容错读两路；待 D 段稳定后下个 wave 删 device_id 字段 |
| **D 段前端 413 处改动量大易遗漏** | 高 | 用 `git grep -n "device_id\|deviceId"` 列清单，按文件 batch 替换；T-0165-D 内拆 sub-batch（services/hooks/store/types/mock 一批 / pages 主皮肤一批 / webcode-v2/v3 一批）；每 batch typecheck 兜底 |
| **运行时 ParamRegistry / ProductRegistry 等缓存 key 涉及 device_id** | 低 | grep Redis key 命名 `device:*` / `acs:*:device_*` → 切到 `device:{oui}:{sn}` 形态；启动期清缓存即可 |
| **审计日志 / 历史告警等"快照"类表的 device_id 含语义历史** | 中 | 历史表（alarms_history / station_fault_logs）双键并存即可（device_id 历史快照 + oui+sn 当前业务键）；不强制清 device_id 字段 |

## 5. 4 个 Sub-task 详情

### T-0165-A: devices 表重构（去 carrier 分区 + UNIQUE(oui, serial_number)）

**Est:** L (~3-4d) — 关键路径

**Goal:** devices 表去掉 `PARTITION BY LIST (carrier)`，PRIMARY KEY 从 `(id, carrier)` 收回 `(id)`，新增 `UNIQUE (oui, serial_number)` 约束作为业务侧唯一键；保留 carrier 字段（业务过滤用）。

**难点:**
- 分区表去分区 = 必须重建表 + 数据迁移（PostgreSQL 不支持 `ALTER TABLE ... DROP PARTITIONING`）
- 30+ 张业务表通过 `device_id` 列逻辑引用 devices（部分模块在 application 层校验外键），切换前需先盘点所有 FK 引用
- migration 000089 的 `idx_devices_serial_number` partial unique 索引需重建为 `UNIQUE(oui, serial_number)`（如果该索引仍有逻辑必要性）

**步骤（5 步）:**

1. **盘点 FK 引用** — `pg_dump --schema-only` + SQL `SELECT conname, conrelid::regclass FROM pg_constraint WHERE confrelid = 'devices'::regclass` 列出所有真 FK；grep `migrations/*.sql` 找逻辑引用（device_id 列 + comment）；产出 FK 清单文档
2. **设计 migration 000172** — `BEGIN` → 创建 `devices_new`（无分区 + PK(id) + UNIQUE(oui, serial_number) + 复制所有原列 / 索引 / trigger）→ `INSERT INTO devices_new SELECT FROM devices` → DROP 旧表（CASCADE 清理子分区 + 旧索引）→ `ALTER TABLE devices_new RENAME TO devices` → 重建 partial unique（如仍需）→ `COMMIT`；Down 段反向重建分区表 + 数据回填（声明 dev-only）
3. **Go 代码适配** — `internal/device/pg_repository.go` 所有 `WHERE id=$1 AND carrier=$2` → `WHERE id=$1`；新增 `GetByOuiAndSN(ctx, oui, sn)` 方法；扫其他模块对 devices 复合 PK 的引用（grep `(id, carrier)`），统一收口
4. **单测 + 集成测试** — `device_test.go` 覆盖 GetByID / GetByOuiAndSN / Create（含 (oui, sn) 唯一性冲突路径）；启动一个 devices 含 2 个 oui 的 fixture，验证查询正确性
5. **DoD 验证** — `make migrate-up && make migrate-down && make migrate-up` 双向通过；CPE 模拟器注册 (oui=48BF74, sn=NEW001) 设备 → devices 表落库验证；commit + 走 `/commit` skill

**验收:**
- [ ] devices 表去分区（`\d devices` 无 Partitioned table 标记）
- [ ] `UNIQUE(oui, serial_number)` 约束建成
- [ ] 写入 (oui=A, sn=X) + (oui=B, sn=X) 成功（跨厂商 SN 不撞键）
- [ ] 写入两条 (oui=A, sn=X) 第二条失败（唯一性生效）
- [ ] 30+ 业务表的 device_id 引用仍可用（A 段不动业务表，B 段做）
- [ ] `go test ./internal/device/...` 全过

### T-0165-B: 业务存储层切 OUI+SN（30+ 张表 repository/model）

**Est:** L (~4-5d) — 分批推进

**Goal:** 30+ 张业务表的 repository / model 改造，业务键从"以 device_id 为主"切到"以 (oui, serial_number) 为主"；device_id 保留作 internal FK 关联（不删列）。

**涉及表（按模块分批）:**

| Batch | 模块 | 表 |
|-------|------|----|
| B1 | alarm | alarms_active, alarms_history |
| B2 | pm（T-0164 已部分切，本段做收口验证） | pm_metrics（含聚合表 hourly/daily/weekly/monthly） |
| B3 | mr | mr_records, mrs_records, mre_records |
| B4 | config/parammodel | device_parameters（32 hash 分区） |
| B5 | task | device_tasks |
| B6 | backup | config_backup_sub_tasks |
| B7 | syslog | ne_message_logs |
| B8 | acs | acs_sessions（如有 PG 持久化）、protocol_logs |
| B9 | stationlog | station_fault_logs（含 abnormal reboot） |
| B10 | 其他 | grep 兜底（device_info / device_groups / role_device_groups / 等） |

**步骤（每 Batch 内 4 步）:**

1. **加列 migration** — 给 batch 内所有表加 `oui VARCHAR(6)` + `serial_number VARCHAR(64)` 列 + UNIQUE 索引（业务键复合唯一性）；migration 单独一文件（如 173_alarm_oui_sn / 174_pm_oui_sn / ...）；数据迁移用 `UPDATE ... FROM devices WHERE table.device_id = devices.id`
2. **repository 改造** — model.go 加 Oui + SerialNumber 字段；pg_repository.go 查询从 `WHERE device_id=$1` → `WHERE oui=$1 AND serial_number=$2`（主路径）；保留 `GetByDeviceID(ctx, id)` 作 internal 路径
3. **service / handler 适配** — service 层入参用 oui+sn，handler 在 D 段切公开契约前临时保留 device_id 兼容路径
4. **单测覆盖** — 每模块 `_test.go` 覆盖 oui+sn 查询路径 + 跨厂商 SN 不撞键场景

**验收（每 Batch）:**
- [ ] migration up / down 双向通过
- [ ] 该 batch 的 repository 单测全过
- [ ] e2e_verify.sh 该模块段全过（不破现有用例）
- [ ] 真实数据迁移成功（`SELECT COUNT(*) FROM table WHERE oui IS NULL` = 0）

### T-0165-C: 事件总线 payload 统一

**Est:** M (~2d)

**Goal:** 所有 NATS SubjectXxx 事件 payload 主标识改为 `(oui, serial_number)`；保留 `device_id (legacy)` 字段作过渡，subscriber 优先消费 oui+sn；待 D 段稳定 1-2 wave 后下个 task 删 device_id 字段。

**涉及事件主题（盘点）:**
- `device.inform.bootstrap` / `device.inform.periodic` / `device.inform.value_change`
- `device.registered` / `device.online` / `device.offline` / `device.reboot.abnormal`
- `alarm.raised` / `alarm.cleared` / `alarm.acknowledged`
- `command.set_parameters` / `command.get_parameters`
- `pm.file.received` / `pm.file.parsed`
- `mr.file.received` / `mr.file.parsed`
- `task.completed` / `task.failed`
- 其他 grep `Subject\w+` 兜底

**步骤（4 步）:**

1. **盘点事件 + Payload 结构** — grep `internal/events/handlers/*.go` 列所有 Subject + Payload 结构；产出"事件 → payload 字段"清单
2. **Payload 字段扩展** — 每个 Payload 结构体加 `Oui string` + `SerialNumber string` 字段（保留 DeviceID 字段，标 `// legacy: 保留 1-2 wave 后删除`）；发布方在事件构造期填齐 oui+sn
3. **Subscriber 改造** — 所有 subscriber 优先读 oui+sn 路径（如 `if e.Oui != "" && e.SerialNumber != "" { ... } else { /* fallback device_id 查 devices 获取 oui+sn */ }`）
4. **集成测试** — 启动一次完整链路（CPE 模拟器 Inform → ACS 发 device.registered → notification subscriber 收事件）；NATS 抓包验证 payload 含 oui+sn

**验收:**
- [ ] 5+ 关键事件 payload 含 oui+sn
- [ ] subscriber 全部走 oui+sn 主路径
- [ ] 集成测试通过
- [ ] `go test ./internal/events/... ./internal/notification/...` 全过

### T-0165-D: REST API + 前端契约切换

**Est:** L (~3-4d)

**Goal:** 公开 REST API 和前端契约统一切换到 oui+sn；handler path 参数从 `/devices/:id` 扩展为 `/devices/:oui/:sn`；前端 413 处 `device_id`/`deviceId` 引用整批替换为 `(oui, serial_number) / (oui, serialNumber)`。

**步骤（5 步）:**

1. **REST API 路由扩展** — `internal/device/handler.go` / `alarm/handler.go` / `pm/handler.go` 等加新路由 `/api/v1/devices/:oui/:sn/*`；handler 实现 oui+sn 主路径；保留 `/api/v1/devices/:id` 作为 fallback 兼容（一个 wave 后下次 task 删）
2. **前端 frontend-core 业务层** — services/api/*.ts (29 个) + types/*.ts + hooks/api/*.ts (24 个) + store/*.ts + mock/*.ts 整批 grep + 替换；典型改造：URL 拼接 `/devices/${deviceId}` → `/devices/${oui}/${serialNumber}`；TypeScript 类型 `{ deviceId: string }` → `{ oui: string, serialNumber: string }`
3. **前端 webcode 主皮肤** — pages/**/*.tsx + components/**/*.tsx 中所有设备引用改造；表格列定义 / 表单字段 / URL 参数 / 跳转链接逐个修
4. **多皮肤验证** — webcode-v2 + webcode-v3 typecheck 通过（pre-existing baseline 不破）
5. **playwright MCP 端到端** — 浏览器实测：登录 → 设备列表 → 设备详情 → 告警 → PM → MML，全链路验证；网络面板抓所有 /api 请求确认 URL 含 oui+sn

**验收:**
- [ ] 新路由 `/api/v1/devices/:oui/:sn/*` 端点测试全过（curl）
- [ ] 前端 typecheck 三皮肤全过
- [ ] playwright MCP 端到端通过
- [ ] 浏览器网络面板抓不到 `device_id=uuid` 形式参数（仅历史 fallback 路由保留）

## 6. Out of scope

- **不删除 device_id (uuid)** — 保留作 internal PK 与历史快照表（alarms_history / station_fault_logs）的快照字段；仅业务键侧改用 oui+sn
- **不重构 carrier 字段** — devices.carrier 保留作业务过滤维度，不再作分区键 / 不再进 PK
- **不动 T-0164 PM 内部已切换部分** — T-0164-P3 在 PM 模块内部已用 oui+sn（独立 fix commit），本 task 仅做收口验证不重做
- **不做在线 zero-downtime 迁移** — 项目未上生产，devices 重建直接 DROP + CREATE + 数据导入；生产部署时另立 task 做"双写 → 切流量"模式
- **不在本任务删除 events payload 的 device_id legacy 字段** — C 段双载过渡，删字段留下个 wave 独立 task
- **不在本任务删除 REST API 的 `/devices/:id` fallback 路由** — D 段并存过渡，删 fallback 留下个 wave 独立 task

## 7. 风险登记（与 risk-register.md 同步）

| 风险编号 | 描述 | 缓解 |
|---------|------|------|
| R-T0165-01 | devices 重建表期间需停机或 maintenance window（生产部署时） | 当前 dev 不影响；生产部署前另立 task 做双写切流量模式 |
| R-T0165-02 | FK 级联影响面大（30+ 张表逻辑引用 devices(id, carrier)） | A 段先盘点 FK 清单；B 段每模块独立 PR + 单测 + e2e 覆盖 |
| R-T0165-03 | 前端 413 处改动量大易遗漏 | D 段拆 sub-batch + typecheck 兜底每 batch + playwright MCP 端到端验证 |
| R-T0165-04 | 跨段（A → B → C → D）任何一段卡 60+ min 阻塞后续 | 每段独立合 main 后再开下一段；遇阻立刻 raise（不死磕） |

## 8. 提交节奏

每个 Sub-task 至少一个 commit（结构性变更可拆多 commit）：
- migration 单独 commit（便于回滚）
- 服务 / handler / repository 一起 commit（保持可工作状态）
- test 跟实现一起 commit（不另立 test commit）
- B 段 batch 数 ≥ 4 时按 batch 分多个 commit

commit message footer 五元组（参考 backlog/done/2026Q2.md 范例）：
```
PRD: docs/project/plan-T-0165-system-wide-oui-sn-migration.md
Sprint: wave-3-post-T0164
Risk: R-T0165-0N
Backlog: T-0165-{A|B|C|D}
Review: docs/review-report/...
```

## 9. 设计决策不重新评审

按用户拍板（2026-05-23），以下决策已锁定：
- **device_id (uuid) 保留作 internal PK**，业务键独立 UNIQUE(oui, serial_number)；不一刀切删 uuid
- **去 carrier 分区**：实测 100% 数据落 cmcc 分区，分区无收益反而强制 PK 携带 carrier 列
- **不引入 fallback / 灰度 / 兼容路径**（项目未上生产）；A → B → C → D 顺序切换，每段独立合入
- **C 段事件 payload 双载过渡**仅一个 wave，下次 task 删 legacy device_id 字段
- **D 段 REST API 路由并存过渡**仅一个 wave，下次 task 删 `/devices/:id` fallback 路由
- **T-0164 已在 PM 内部切的不动**（独立 fix commit），本 task 仅做收口验证
