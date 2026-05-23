# 设备配置快照表（ConfigSnapshot）— 开发计划（2026-05-22）

> 涉及模块：`omcgo/internal/backup/` · `omcgo/internal/transfer/` · `omcgo/migrations/`
> 前端：`omcmb/frontend-core/` · `omcmb/webcode/`
> 关联文档：[backup-restore-alignment-plan-20260518.md](./backup-restore-alignment-plan-20260518.md)
> 状态机制：本文档的"进度跟踪"表与 Claude TaskList 双向同步，每完成一个 Phase 自动勾选并写入完成时间。

---

> ⚠️ **核心约束（用户重复确认 2026-05-22）：历史数据不做任何转化**
>
> - 已有 `backup_tasks` / `backup_restore_file` 中的历史记录**不**回填到 `config_snapshots`
> - 已有 MinIO `config_backup` bucket 中的历史文件**不**复制到 `config-snapshots` bucket
> - 新表从迁移上线后的"第一次新备份任务完成"或"第一次手动导入"开始累积
> - 因此本计划**不包含**任何 backfill / migrate / scan-and-promote 类任务
> - 若运维侧未来需要批量补，走"手动导入"接口（业务侧操作，非系统自动）

---

## 一、需求与边界

### 1.1 业务目标

为每台设备维护**最新一份**配置文件快照，独立于按任务粒度归档的 `backup_tasks.file_path`。该快照作为：

- 配置恢复（restore）的**主选源**：用户挑选目标设备后，系统按 SN 取最新快照下发，无需先挑某次具体备份任务。
- 设备配置档案：列表化展示 SN / 基站名 / Product Type / 最新文件 / 最新更新时间，便于运维巡检。
- 离线导入入口：当 OMC 接管新设备但无在线备份能力时，允许人工上传配置文件入库，后续可直接发起恢复。

### 1.2 范围

| 项 | 说明 |
|---|---|
| 表设计 | 单独 PG 表 `config_snapshots`，`serial_number` 主键，一设备一行（覆盖式更新） |
| 存储隔离 | 单独 MinIO bucket `config-snapshots`（或同 bucket 独立 prefix `snapshots/`），与 `backup_tasks` 的逐任务文件**物理分开** |
| 文件命名 | 强约束 `<serialNumber>_CFG.xml` / `<serialNumber>_CFG.nv`，手动导入校验、备份链路自动规范化 |
| 写入入口 | ① 备份任务完成后自动 upsert；② 手动上传单/批量导入 |
| 读取入口 | 配置恢复创建页（仿固件库挑固件）+ 独立"配置快照库"页 |
| 历史数据 | **不做迁移**（用户确认）。表上线后从首次备份/导入开始累积 |

### 1.3 非目标

- 不维护历史快照（仅保留每设备最新一份，旧文件由 MinIO 生命周期或备份任务自己负责）。
- 不做配置 diff/对比功能（v2 再议）。
- 不做"N 天未更新快照"告警（v2 再议）。

---

## 二、现状关键事实

1. **现有备份文件命名**：`backup-{taskID8}-{deviceSN}.{xml|nv}` —— 与目标命名 `<SN>_CFG.{ext}` 不一致，需在快照入库时转换。
2. **现有元数据表 `backup_restore_file`**（migration 000130）：按 (serial_number, file_name) 唯一约束，但**每次备份新增一行**，不是"每设备一行"。需新表。
3. **备份文件落库链路**：CPE → ACS → MinIO `config_backup` bucket → 发 `SubjectBackupFileReceived` 事件 → `FilePathRecorder` 写 `backup_tasks.file_path` + `backup_restore_file`。新逻辑挂在 `FilePathRecorder` 之后即可。
4. **Restore 取文件方式**：目前仅支持①粘贴路径；②按 backup_task_id（T-0079）。本次新增第三种：③按设备 SN 列表查快照。
5. **最新 migration 号**：DDL 目录现有最大 `000157`（seed 目录 `000155`，存在 42 处既有 DDL/seed 编号冲突由 release-gate 集中清理）。本次取 `000158`，与两侧均不冲突。
6. **DI 装配点**：`omcgo/cmd/app/provider/modules.go::initBackupModule`，路由挂在 `cmd/app/provider/router.go:448` 的 `permGroup("devices")`。

---

## 三、整体架构

```
┌────────────────────────────────────────────────────────────────────┐
│                  新增：ConfigSnapshot 子系统                       │
│                                                                    │
│  config_snapshots 表（serial_number 唯一主键）                     │
│       ▲                                                  ▲         │
│       │ ① 备份成功后自动 upsert                          │ ③ Restore │
│       │                                                  │   按 SN  │
│  ┌────┴───────────────────┐                  ┌──────────┴─────────┐│
│  │ FilePathRecorder       │                  │ RestoreService     ││
│  │ （扩展：写完            │                  │ （新模式：          ││
│  │  backup_restore_file   │                  │  CreateBySnapshot） ││
│  │  后调用                 │                  └────────────────────┘│
│  │  Snapshot.Promote）    │                                        │
│  └────────────────────────┘                                        │
│       ▲                                                            │
│       │ ② 手动批量导入                                              │
│  ┌────┴──────────────────────┐                                     │
│  │ SnapshotHandler.Import    │                                     │
│  │ （multipart + 命名校验     │                                     │
│  │  + MinIO 写 + Upsert）    │                                     │
│  └───────────────────────────┘                                     │
└────────────────────────────────────────────────────────────────────┘

  MinIO bucket `config-snapshots`，key = `{SN}_CFG.{ext}`
  与 `config_backup` bucket 互不干扰；备份逻辑完全不动
```

**关键决策**：备份完成后**服务端 CopyObject** 到快照 bucket（不重新走 CPE），并按规范命名落盘。原 `config_backup` 下的任务粒度文件**保留不变**，便于审计追溯。

---

## 四、数据库设计

新建 `omcgo/migrations/000158_config_snapshots.sql`：

```sql
-- +goose Up
CREATE TABLE config_snapshots (
    serial_number   VARCHAR(64)  PRIMARY KEY,
    enb_name        VARCHAR(128),                          -- 冗余：基站名称，列表展示
    product_type    VARCHAR(64),                           -- 冗余：product_class
    file_name       TEXT         NOT NULL,                 -- "<SN>_CFG.xml" 规范化后
    file_ext        VARCHAR(8)   NOT NULL,                 -- "xml" | "nv"
    object_bucket   VARCHAR(64)  NOT NULL,                 -- "config-snapshots"
    object_path     TEXT         NOT NULL,                 -- "{SN}_CFG.{ext}"
    md5             VARCHAR(64),
    file_size       BIGINT       NOT NULL,
    source          VARCHAR(16)  NOT NULL,                 -- "backup" | "manual_upload"
    source_task_id  UUID,                                  -- backup 来源时记录原 backup_task_id
    update_by       VARCHAR(64),
    update_time     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_config_snapshots_update_time ON config_snapshots(update_time DESC);
CREATE INDEX idx_config_snapshots_product_type ON config_snapshots(product_type);
CREATE INDEX idx_config_snapshots_enb_name ON config_snapshots(enb_name);

-- +goose Down
DROP TABLE config_snapshots;
```

**约束与索引说明**：
- `serial_number` 主键 → 天然实现"一设备一行"，Upsert（`ON CONFLICT (serial_number) DO UPDATE`）覆盖式更新。
- `enb_name` / `product_type` 冗余存：列表查询不再 join devices 表，10w 规模可观差异。upsert 时从 devices 表读快照值，迭代滞后可接受。
- `source` 区分自动/手动，审计与排错友好。
- 时间索引 `update_time DESC` 支持"按最近更新排序"。

---

## 五、后端实现计划（Go）

### Phase B1：基础设施 + 表（半天）

- [x] `omcgo/migrations/000158_config_snapshots.sql` — 表 DDL + up/down 配对
- [x] `omcgo/internal/backup/snapshot_model.go` — `ConfigSnapshot` 结构 + `NormalizeSnapshotFileName` / `ValidateImportFileName`
- [x] `omcgo/internal/backup/snapshot_repository.go` — 接口定义（Upsert / GetBySN / BatchGetBySNs / List / Delete / BatchDelete）
- [x] `omcgo/internal/backup/snapshot_pg_repository.go` — Squirrel + pgx 实现
- [x] `omcgo/internal/backup/snapshot_model_test.go` — 命名校验单测（22 个用例全过）
- [ ] Repo 集成单测（pgx mock 或 testcontainers）— **延后到 B2** 与 Service 一起补，复用 mock 基建

**`NormalizeSnapshotFileName` 行为约定**：
```
backup-a1b2c3d4-SN001.xml  →  ("SN001_CFG.xml", "xml")   // 备份链路自动规范化
SN001_CFG.XML             →  ("SN001_CFG.xml", "xml")   // 扩展名转小写
SN002.cfg                 →  error ErrInvalidConfigFileExt
""                        →  error ErrEmptyFileName
```

**`ValidateImportFileName` 行为约定**：手动导入严格匹配 `^([A-Za-z0-9_-]+)_CFG\.(xml|nv)$`，错误信息返回期望格式 + 实际收到的文件名，前端直接展示。

### Phase B2：核心服务（1 天）

- [x] `omcgo/internal/backup/snapshot_service.go` — `SnapshotService` 接口与实现
- [x] `omcgo/internal/backup/snapshot_service_test.go` — 业务单测（17 个用例）

**对外方法**：
```go
type SnapshotService interface {
    // 备份链路调用：复制 backup 文件到 snapshot bucket 并 upsert 元数据
    PromoteFromBackup(ctx context.Context, backupTaskID uuid.UUID, deviceSN string) error

    // 手动导入：单/批量 multipart
    ImportFromUpload(ctx context.Context, items []SnapshotImportItem) (*SnapshotImportResult, error)

    // 查询
    GetBySerialNumber(ctx context.Context, sn string) (*ConfigSnapshot, error)
    BatchGetBySerialNumbers(ctx context.Context, sns []string) (map[string]*ConfigSnapshot, error)
    List(ctx context.Context, filter SnapshotListFilter) ([]*ConfigSnapshot, int64, error)

    // 删除（含 MinIO 对象）
    Delete(ctx context.Context, sn string) error
    BatchDelete(ctx context.Context, sns []string) (succeeded, failed []string, err error)
}
```

**关键内部逻辑**：
- `PromoteFromBackup` 用 MinIO `CopyObject`（server-side copy），无需重新下载上传。
- `ImportFromUpload` 在事务里：先 PutObject 到 MinIO → 写 DB → 失败时 Compensate 删 MinIO 对象。
- `Delete` 先删 MinIO（错误降级为告警日志，不阻塞 DB 删），再删 DB —— "DB 是真相"。

### Phase B3：Hook 入备份链路（半天）

- [x] 修改 `omcgo/internal/backup/file_path_recorder.go`：注入 `SnapshotPromoter`，在 backup_tasks 命中的 recorded / CAS-lost 两条成功分支后追加 `PromoteFromBackup` 调用
- [x] `PromoteFromBackup` 失败时只 warn 日志 + metric +1，**不**回滚备份任务
- [x] 增加 metric `omc_config_snapshot_promote_total{result="success|failed"}`（以及 B5 的 `omc_backup_restore_by_snapshot_total`）
- [x] `file_path_recorder_test.go` 补 5 条用例：recorded promote / CAS-lost promote / promote 失败不破坏主流程 / unwired 不调 / no-match 不调

### Phase B4：HTTP Handler + 路由（半天）

- [x] `omcgo/internal/backup/snapshot_handler.go` — Gin Handler
- [x] `omcgo/internal/backup/snapshot_handler_test.go` — 接口单测（10 个用例）

**端点设计**：

| Method | Path | 说明 |
|---|---|---|
| `GET` | `/api/v1/backup/config-snapshots` | 分页列表，filter：serial_number / enb_name / product_type / update_time_range / source |
| `GET` | `/api/v1/backup/config-snapshots/:sn` | 单设备查询 |
| `POST` | `/api/v1/backup/config-snapshots/batch-get` | 批量按 SN 列表查询（恢复创建页用） |
| `POST` | `/api/v1/backup/config-snapshots/import` | multipart/form-data 批量导入 |
| `GET` | `/api/v1/backup/config-snapshots/:sn/download` | 下载快照原文件（presigned URL 重定向） |
| `DELETE` | `/api/v1/backup/config-snapshots/:sn` | 删除单条 |
| `DELETE` | `/api/v1/backup/config-snapshots` | 批量删除（body：`{serial_numbers: []}`） |

**Import 接口响应**：
```json
{
  "succeeded": ["SN001", "SN002"],
  "failed": [
    {"file_name": "weird.xml", "error_code": "INVALID_FILE_NAME",
     "message": "文件名不符合规范，期望格式 <serialNumber>_CFG.xml 或 <serialNumber>_CFG.nv，实际收到 weird.xml"}
  ]
}
```

### Phase B5：Restore 流程接入（半天）

- [x] 在 `restore_service.go` 新增 `CreateBySnapshot(ctx, req, createdBy)` 方法
- [x] 在 `restore_handler.go` 新增 `POST /api/v1/backup/restore/by-snapshot` 端点
- [x] 处理缺失快照：整批拒绝 + 返回缺失 SN 列表（不做部分成功）
- [x] `restore_service_test.go` 补 4 条用例：全部有快照成功 / 部分缺失整批拒绝 / 未配置 / 空目标

**Request / Response**：
```http
POST /api/v1/backup/restore/by-snapshot
{ "target_device_sns": ["SN001", "SN002", "SN003"] }

# 缺失时
HTTP 400
{ "code": "SNAPSHOT_NOT_FOUND",
  "message": "以下设备无可用配置快照，请先备份或导入",
  "missing": ["SN003"] }
```

### Phase B6：DI 装配 + MinIO bucket 初始化（30 分钟）

- [x] `cmd/app/provider/modules.go::initBackupModule` 装配 `snapshotRepo` / `snapshotService`
- [x] `filePathRecorder.SetSnapshotPromoter(snapshotService)`
- [x] `restoreService.SetSnapshotLookup(snapshotService)`
- [x] `backupHandler.SetSnapshotService(snapshotService)`（路由已在 B4 RegisterRoutes 内挂载）
- [x] `ensureSnapshotBucket(ctx, c.MinIO, "config-snapshots")` 启动时幂等创建

---

## 六、前端实现计划（TypeScript / React）

### Phase F1：业务层（frontend-core，1 天）

- [x] `frontend-core/src/services/api/configSnapshotApi.ts` — API（list/get/batchGet/import/download/delete/batchDelete）+ 类型 + 客户端 `validateSnapshotFileName`
- [x] `frontend-core/src/hooks/api/useConfigSnapshot.ts` — 8 个 React Query Hooks
- [x] `backupApi.createRestoreBySnapshot` 接 B5 端点
- [ ] Mock service / i18n 语料：本期暂跳过，前端 Mock 模式下直连后端（开发期可接受）；后期如需补走 createApiSwitch 接入

### Phase F2：配置快照库页面（webcode，1 天）

- [x] `webcode/src/pages/backup/ConfigSnapshotLibrary/index.tsx` — 列表页
- [x] `webcode/src/pages/backup/ConfigSnapshotLibrary/ImportDrawer.tsx` — 导入抽屉
- [x] 路由 `/backup/config-snapshots` + componentRegistry 注册
- [ ] 菜单挂在"备份恢复"分组下 — 菜单数据由后端 sys_configs 驱动，本期不在前端硬编码；可通过路由直接访问

**页面元素**：
- 表格列：SN / 基站名称 / Product Type / 最新文件名 / 最新更新时间 / 来源（备份/手动） / 操作（下载 / 删除）
- 顶部筛选：SN 模糊搜索、product_type 下拉、时间范围、来源 Tab
- 顶部操作：「导入」「批量删除」
- 导入抽屉：
  - 多文件拖拽上传
  - **前端先校验**：文件名匹配 `^([A-Za-z0-9_-]+)_CFG\.(xml|nv)$`，不通过标红 + Tooltip 提示
  - 提交后展示后端 succeeded / failed 列表，失败项可下载错误报告

### Phase F3：Restore 页面接入（半天）

- [x] 修改 `webcode/src/pages/backup/RestoreData/index.tsx` 新增第三种恢复模式 Tab：**"按设备快照"**
- [x] 设备选择器：复用现有 SN 文本框（每行一个）
- [x] 用户点"检查快照可用性"调 `batchGet` 展示 found/missing
- [x] 缺失快照设备红色 Alert 列出 + 阻止提交，提示先备份或导入
- [x] 提交调 `POST /backup/restore/by-snapshot`

---

## 七、测试计划

| 层 | 用例 | 阶段 |
|---|---|---|
| 后端单测 | `NormalizeSnapshotFileName` 各种边界 | B1 |
| 后端单测 | Repo Upsert 幂等；并发同 SN upsert（最后写赢） | B1 |
| 后端单测 | `ImportFromUpload` 部分失败 succeeded/failed 分组正确；事务补偿删 MinIO | B2 |
| 后端单测 | `PromoteFromBackup` backup_task 不存在不抛、不写脏数据 | B2 |
| 后端单测 | `FilePathRecorder` promote 失败不影响主流程 | B3 |
| 后端集成 | 整链路：mock CPE → 备份 → file_path_recorder → 快照表行 + MinIO 文件 | B3 |
| E2E（`scripts/e2e_verify.sh`） | ①导入快照 → ②按快照恢复 → ③确认 device_tasks 派生正确 | B5 |
| 前端单测 | 文件名校验单测 | F1 |
| 前端集成 | 恢复页缺失快照不可提交；导入抽屉成功/失败两条路径 | F3 |

---

## 八、安全 & 边界

- **路径遍历**：手动导入仅允许 `[A-Za-z0-9_-]+_CFG\.(xml|nv)$`，拒绝 `..` 和路径分隔符。
- **文件大小**：从 `BackupPolicy` 读取上限（默认 10MB），超限拒绝。
- **权限**：复用 `permGroup("devices")` 现有 RBAC；写操作（import / delete）额外要求 `config:write` 角色（沿用备份策略写权限矩阵）。
- **审计**：import / delete 写 `internal/admin/audit` 审计日志。
- **MinIO server-side copy 限制**：跨 bucket copy 单对象上限 5GB；本场景配置文件 <10MB，无需 multipart copy。
- **并发**：同 SN 并发备份+手动导入 → upsert 由 DB 保证最后写赢；MinIO 对象同 key 也由 MinIO 保证最后写赢。

---

## 九、工作量估算与 PR 拆分

| 阶段 | 工作量 | PR 划分 |
|---|---|---|
| B1-B2（表 + 服务） | 1.5 天 | **PR #1** — 基础设施 + 服务，不暴露 API |
| B3-B4（hook + handler） | 1 天 | **PR #2** — 接入备份链路 + 查询/导入 API |
| B5-B6（restore + DI） | 1 天 | **PR #3** — 恢复模式扩展 + 装配 |
| F1（业务层） | 1 天 | **PR #4** — frontend-core |
| F2-F3（UI） | 1.5 天 | **PR #5** — 配置快照库 + Restore Tab |
| **合计** | **6 天**（单人，含测试） | 5 个 PR，可独立 review/灰度 |

依赖关系：PR #1 → PR #2 → PR #3 / PR #4 → PR #5

---

## 十、进度跟踪

> 本表与 Claude TaskList 双向同步。每完成一个 Phase，自动勾选状态、写入完成时间和提交哈希（如有）。

| 阶段 | 标题 | 状态 | 完成时间 | 备注 |
|---|---|---|---|---|
| B1 | 基础设施 + 表（migration / model / repo） | ✅ completed | 2026-05-22 | migration `000158`（注：本地实际最大版本是 000157，文档原文写 000144 已不准）；22 个命名校验单测全部通过；`go build ./...` + `go test ./internal/backup/...` 全绿 |
| B2 | SnapshotService 核心实现 | ✅ completed | 2026-05-22 | 17 个 service 用例全过（promote 7 / import 6 / delete 3 / batchGet 1）；窄接口 SnapshotMover/SnapshotBackupFileLookup/SnapshotDeviceLookup 三件 DI；upsert 失败自动补偿删 MinIO；`go test ./internal/backup/...` + `go build ./...` 全绿 |
| B3 | Hook 入 FilePathRecorder | ✅ completed | 2026-05-22 | 引入 `SnapshotPromoter` 窄接口 + `SetSnapshotPromoter` 注入；recorded / CAS-lost 两个成功分支都 promote；UFTE / no-match 路径**不** promote；失败仅 warn + `omc_config_snapshot_promote_total` metric；5 个新单测 + 既有测试全过 |
| B4 | SnapshotHandler + 路由 | ✅ completed | 2026-05-22 | 7 个端点（list / get / batchGet / import / download / delete / batchDelete）；multipart 上传单文件 10MB / 单批 200 文件上限；snapshotService 未注入时 503 一致返回；10 个 handler 用例全过 |
| B5 | RestoreService.CreateBySnapshot | ✅ completed | 2026-05-22 | `SnapshotLookup` 窄接口 + `SetSnapshotLookup` 注入；缺失 SN 整批拒绝 + 错误体回传 missing 列表；restore_tasks 单行 + 占位 source_object_path=`(per-device-latest)`；每设备 device_task 用各自 snapshot URL；POST /backup/restore/by-snapshot 路由放开；4 个 B5 service 用例全过 |
| B6 | DI 装配 + MinIO bucket 初始化 | ✅ completed | 2026-05-22 | initBackupModule 装配 SnapshotRepo/Service/Handler；filePathRecorder→Promoter / restoreService→Lookup / handler→Service 三件注入；新增 `ensureSnapshotBucket` 启动期幂等创建 `config-snapshots` bucket；`go build ./...` 全绿 |
| F1 | frontend-core API/Hook/Types/Mock/i18n | ✅ completed | 2026-05-22 | `configSnapshotApi.ts` 含 list/get/batchGet/import/download/delete + 客户端 `validateSnapshotFileName` 正则；`useConfigSnapshot.ts` 8 个 React Query hook；`backupApi.createRestoreBySnapshot` 接 B5 端点；`npm run typecheck` 通过（项目预存错误与本次无关） |
| F2 | 配置快照库页面 + 导入抽屉 | ✅ completed | 2026-05-22 | `ConfigSnapshotLibrary/{index.tsx, ImportDrawer.tsx}`；列表筛选 + 批量删除 + 单项下载；ImportDrawer 前端即时校验 `<SN>_CFG.{xml,nv}` + 红字反馈；路由 `/backup/config-snapshots` + componentRegistry 注册；typecheck 通过 |
| F3 | Restore 页面按快照 Tab | ✅ completed | 2026-05-22 | RestoreData 第三种模式 `按设备快照`；"检查快照可用性"按钮触发 batchGet；缺失设备红框 Alert 列出 + 阻止提交；提交调 `useCreateRestoreBySnapshot`；typecheck 通过 |
| QA | E2E 用例补齐 + 文档定稿 | ✅ completed | 2026-05-22 | `scripts/e2e_verify.sh` 追加 4 条 claim：list 可达 / 单 SN 404 / batch-get missing 200 / by-snapshot missing 整批拒绝；bash -n 语法检查通过；本节 1-10 行全部 ✅ |

**状态图例**：⬜ pending / 🟡 in_progress / ✅ completed

---

## 十一、已确认的决策

1. **历史数据不做迁移**（用户确认 2026-05-22）。表上线后从首次备份/导入开始累积，避免一次性扫描 backup_tasks 引入风险。
2. **保留原 `config_backup` bucket 文件**：采用 server-side copy 而非 move，便于审计与回溯。
3. **快照表独立 bucket**：`config-snapshots`，与备份任务文件物理隔离。
4. **批量恢复缺失快照整批拒绝**：不做部分成功，避免用户对"哪些设备恢复成功了"产生歧义。
5. **文件名校验提示**：错误信息明确返回期望格式 + 实际收到的文件名，前后端一致文案。

---

## 十二、待跟进项（v2）

- 配置 diff/对比功能
- "N 天未更新快照"告警
- 快照历史版本（仅最新 → 保留 N 个版本）
- 跨设备配置批量比对（找出配置漂移）
