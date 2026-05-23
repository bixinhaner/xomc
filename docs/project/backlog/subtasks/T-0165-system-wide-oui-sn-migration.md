# T-0165 拆分子任务（全系统 OUI+SN 双键迁移 A-D）

> 从 `docs/project/backlog.md` §3 Active 中的 umbrella T-0165 拆出（2026-05-23，立项即拆 4 子任务）。
> 行 schema 与 backlog.md §3 Active 主表对齐（13 列）。
> umbrella 行仍在主 backlog §3，状态联动通过本表 sub-task 推进。
>
> **来源**：T-0164 PM/KPI 流水线改造期间发现的系统级问题（当前用 `device_id (uuid)` 作设备唯一标识，按 TR-069 标准应该用 `(oui, serial_number)` 双键）；用户 2026-05-23 拍板：① T-0164-P3 内部切换 oui+sn 走独立 fix commit；② 系统级切换作 T-0165 独立 task；③ T-0165 等 T-0164 全部完成（G1-G8 全 done）后启动
> **关联**：T-0164 PM/KPI 流水线改造（前置依赖，必须全 done 才启动 T-0165）
> **PgM 决策（2026-05-23）**：仅做 plan + 登记，不动代码；4 子任务全 State=planned 等 T-0164 完成后再 Owner 分配 / Sprint 排期
>
> **关键约束**：
> - **项目未上生产** — devices 表重建直接 DROP + CREATE + 数据导入，不做在线 zero-downtime 迁移
> - device_id (uuid) **保留作 internal PK**（外键 + 历史快照），新增 UNIQUE(oui, serial_number) 作业务键
> - **去 carrier 分区**：实测 100% 数据落 cmcc 分区，分区无收益反而强制 PK 携带 carrier 列
> - **严格顺序 A → B → C → D**：每段独立合 main 后再开下一段；A 段是关键路径
> - **C 段事件 payload 双载过渡 + D 段路由并存过渡** 仅一个 wave，下次 task 删 legacy
> - **T-0164 已在 PM 内部切换的不动**，本 task 仅做收口验证

### 4.6 T-0165 拆分子任务（4 条，2026-05-23 立项即拆）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0165-A | **devices 表重构（去 carrier 分区 + UNIQUE(oui, serial_number)）** — 删 `PARTITION BY LIST (carrier)` + 3 子分区（cmcc/ctcc/cucc）；PK 从 `(id, carrier)` 收回 `(id)`；新增 `UNIQUE(oui, serial_number)`；carrier 字段保留作业务过滤维度；步骤：盘点 FK → 设计 migration 172 → Go 适配 → 单测 → DoD | feat | F06/device+infra/migration | P1 | planned | — | L (~3-4d) | T-0164 全部完成（G1-G8 all done） | wave-3-post-T0164 | `docs/project/plan-T-0165-system-wide-oui-sn-migration.md` §5.A | 2026-05-23 | 关键路径；卡 60+ min 无进展必须解掉再下走（B/C/D 全依赖 A）；FK 盘点 SQL `SELECT conname, conrelid::regclass FROM pg_constraint WHERE confrelid='devices'::regclass`；migration down 段重建分区表，声明 dev-only |
| T-0165-B | **业务存储层切 OUI+SN（30+ 张表 repository/model）** — 加 oui+serial_number 列 + UNIQUE 索引 + 数据迁移 + repository 改造 + service / handler 适配；分 10 batch（alarm / pm / mr / device_parameters / task / backup / syslog / acs / stationlog / 其他）；device_id 保留作 internal FK | feat | F02/F03/F04/F05/F06+infra | P1 | planned | — | L (~4-5d) | T-0165-A | wave-3-post-T0164 | `docs/project/plan-T-0165-system-wide-oui-sn-migration.md` §5.B | 2026-05-23 | 每 batch 单独 PR + 单测 + e2e；本批 migration 数 5-8（不全塞一文件）；数据迁移用 `UPDATE ... FROM devices WHERE table.device_id = devices.id` 回填 oui+sn；T-0164-P3 已在 PM 内部切，本段做收口验证 |
| T-0165-C | **事件总线 payload 统一（SubjectXxx 主标识改 oui+sn）** — 所有 SubjectXxx 事件 payload 加 Oui + SerialNumber 字段；保留 DeviceID 作 legacy 字段（标 `// legacy: 保留 1-2 wave 后删除`）；subscriber 优先消费 oui+sn 主路径；涉及 device.* / alarm.* / command.* / pm.* / mr.* / task.* 等 5+ 事件主题 | feat | infra+F04/notification+events | P2 | planned | — | M (~2d) | T-0165-B | wave-3-post-T0164 | `docs/project/plan-T-0165-system-wide-oui-sn-migration.md` §5.C | 2026-05-23 | 双载过渡仅 1-2 wave；下次 task 删 legacy device_id 字段独立做；集成测试用 CPE 模拟器 Inform → ACS → notification subscriber 全链路 + NATS 抓包验证 payload |
| T-0165-D | **REST API + 前端契约切换** — handler 路由从 `/api/v1/devices/:id` 扩展为 `/api/v1/devices/:oui/:sn/*`；前端 413 处 device_id/deviceId 引用整批替换；frontend-core (services 29 文件 + hooks 24 文件 + types + store + mock) + webcode 主皮肤 + webcode-v2/v3 typecheck 不破 | feat | F06/device + frontend + frontend-core | P1 | planned | — | L (~3-4d) | T-0165-C | wave-3-post-T0164 | `docs/project/plan-T-0165-system-wide-oui-sn-migration.md` §5.D | 2026-05-23 | 公开契约改动；建议拆 sub-batch（services/hooks/store/types/mock 一批 / pages 主皮肤一批 / v2 v3 一批），每 batch typecheck 兜底；playwright MCP 端到端：登录→设备→告警→PM→MML 全链路验证 oui+sn；`/devices/:id` fallback 路由保留 1 wave 后下次 task 删 |

### 实施顺序

严格顺序依赖：

```
T-0165-A (devices 表重构，L ~3-4d, 关键路径)
   │
   ▼
T-0165-B (业务存储层 30+ 表 repository/model 切换, L ~4-5d)
   │
   ▼
T-0165-C (事件总线 payload 双载过渡, M ~2d)
   │
   ▼
T-0165-D (REST API + 前端契约切换 413 处, L ~3-4d)
```

**每个 sub-task 独立合入 main 后再开下一段**；A 段关键路径不可跳；卡 60+ min 必须解掉。

总工期估算 **~12-15 工作日**（按 1 人节奏）。

### 启动条件

T-0165 启动前置：**T-0164 PM/KPI 流水线改造 G1-G8 全 8 sub-task 全部 done**。

> 当前（2026-05-23）T-0164 状态：P2/P3/P4/P8 已 dev_done_pending_review；P1/P5/P6/P7 仍 planned 待开发。
> 预估 T-0164 收官后 1-2 周启动 T-0165-A。

### 自测策略

| 子任务 | 自测命令 | 工具 |
|--------|---------|------|
| T-0165-A | `make migrate-up && make migrate-down && make migrate-up` 双向通过 + `go test ./internal/device/...` + CPE 模拟器跑 (oui=48BF74, sn=NEW001) | goose + go test + python cpe_simulator |
| T-0165-B | 每 batch：`make migrate-up && go test ./internal/<module>/...` + `SELECT COUNT(*) FROM <table> WHERE oui IS NULL` = 0 + e2e_verify.sh 该模块段 | goose + go test + psql + bash |
| T-0165-C | `go test ./internal/events/... ./internal/notification/...` + 集成测试 CPE 模拟器 Inform → ACS 发 device.registered → notification subscriber 收事件 + NATS 抓包 | go test + cpe_simulator + nats sub |
| T-0165-D | `npm run typecheck`（三皮肤全过）+ `npm run lint` + `npm run test`（Vitest）+ playwright MCP 端到端登录→设备→告警→PM→MML 链路 + 浏览器开发者工具抓 API 请求 URL | npm + playwright |

---

## 设计决策回顾（合入 §5/§6/§9）

- **device_id (uuid) 保留作 internal PK**，新增 UNIQUE(oui, serial_number) 作业务键 — 用户 2026-05-23 拍板
- **devices 表去 carrier 分区** — 实测 100% 数据落 cmcc 分区，分区无收益反而强制 PK 携带 carrier 列
- **A → B → C → D 严格顺序** — 每段独立合入 main 后再开下一段，避免长 PR 难以 review
- **不做在线 zero-downtime 迁移** — 项目未上生产，devices 重建直接 DROP + CREATE + 数据导入；生产部署时另立 task 做双写切流量模式
- **C 段事件 payload 双载 + D 段 REST API 路由并存** — 仅 1-2 wave 过渡，下次 task 删 legacy 独立做
- **T-0164 已在 PM 内部切换的不动** — T-0164-P3 在 PM 内部走独立 fix commit；本 task 仅做收口验证
