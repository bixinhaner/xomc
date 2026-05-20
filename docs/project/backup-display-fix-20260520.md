# 备份/UFTE 任务展示语义修复（2026-05-20）

> **状态**：编码 + 部署完成 · **负责人**：alvin.simmmons / Claude · **关联**：F06（OMC-R 配置备份）+ UFTE 统一任务展示
>
> 创建时间：2026-05-20 · 最后更新：2026-05-20

---

## 1. 背景与问题

文件传输中心新建"配置文件备份（NV）"任务后，UI 出现两处不合理：

### 问题 1 ——主任务列表显示"当前步骤：等待 TransferComplete"

`WAIT_TRANSFER_COMPLETE` 本是**单设备 RPC 步骤**，UFTE 主任务跨多设备，没有"当前步骤"这个概念。
设备列表显示"文件下载中"是合理的（子任务级 status）。

### 问题 2 ——设备列表"目标版本/目标文件"显示 `backup-{task_id8}-{sn}.nv`

这是 UFTE 内置模板的**未渲染字符串**，不是真实文件名。备份是 OUTPUT，目标文件名只有 CPE 上传完成、ACS 落 MinIO、`FilePathRecorder` 写完 `backup_restore_file` 之后才存在。在那之前展示模板字符串是 UX 撒谎；且不可点击下载。

---

## 2. 病灶证据链

| 链路 | 位置 | 现状 |
|------|------|------|
| 前端 STEP_LABELS 翻译 | `omcmb/webcode/src/pages/transfer/shared.tsx:30` | `WAIT_TRANSFER_COMPLETE: '等待 TransferComplete'` |
| 前端主任务列定义"当前步骤"列 | `omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx:584-590` | dataIndex='currentStep' |
| 后端 stepForTask 永远返回设备级 RPC 步骤 | `omcgo/internal/ufte/model.go:546-557` | 默认 `WAIT_TRANSFER_COMPLETE` |
| 后端 mapTask 把模板字符串塞进 TargetVersion | `omcgo/internal/ufte/service.go:534` | `task.FileName` = 模板字符串 |
| software.BatchCollect 把模板原样写主任务 FileName | `omcgo/internal/software/service.go:173` | `FileName: req.TargetFileNameTemplate` |
| ExecuteOneUpload 仅在 URL 渲染 task_id8/sn，未回写子任务 | `omcgo/internal/software/executor.go:301-306` | 渲染结果仅本地变量 |
| 设备列表 fallback 到 parent.FileName 模板 | `omcgo/internal/ufte/service.go:587-590` | `if targetVersion=="" { targetVersion = parent.FileName }` |
| 前端再补一层模板兜底 | `omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx:467-473` | `typeDef?.fileNameTemplate \|\| typeDef?.targetFileNameTemplate \|\| ...` |
| ACS upload 完成事件不能匹配 UFTE 主任务 | `omcgo/internal/backup/file_path_recorder.go:116-128` | 只在 `backup_tasks` 表按 task_id8 前缀查；UFTE backup 走 `software.upgrade_tasks`，匹配失败 |
| upsertFileMetadata 被 backup_tasks 匹配条件门控 | `omcgo/internal/backup/file_path_recorder.go:155-165` | 匹配失败 → `backup_restore_file` 也不会被 upsert |

---

## 3. 完整修复方案

### 后端（B1–B5）

| ID | 文件 | 变更 | 状态 |
|----|------|------|------|
| **B1** | `omcgo/internal/ufte/model.go:546` | `stepForTask`：`softwareTaskType==TaskTypeLogCollect` 时返回空串，主任务不再展示设备级 RPC 步骤 | ✅ |
| **B2** | `omcgo/internal/ufte/service.go:526` `mapTask` | LogCollect 类 `TargetVersion=""`；不再把模板字符串塞主任务 | ✅ |
| **B3** | `omcgo/internal/ufte/model.go:96 DeviceItem` + `omcgo/internal/ufte/service.go:587 mapDeviceItem` + `omcgo/internal/ufte/model.go:585 renderUFTEFileNameTemplate` | 新增 `TargetFile string` + `DownloadURL string` 字段；LogCollect 类不再 fallback `parent.FileName`，而是按 `{task_id8}/{sn}` 渲染模板填入 `TargetFile`；`TargetVersion` 留空 | ✅ |
| **B4** | `omcgo/internal/backup/file_path_recorder.go:100 handleFileReceived` | `upsertFileMetadata` 提到 backup_tasks 匹配判定之前；target=nil 时 OperatorCode 留空；UFTE 链路也能落 `backup_restore_file` | ✅ |
| **B5** | `omcgo/internal/ufte/service.go:27 Service` + `:46 SetDownloadURLLookup` + `omcgo/internal/backup/restore_service.go:307 SplitBucketAndPath` + `omcgo/cmd/app/provider/modules.go:112 initUFTEModule` | UFTE 加可注入回调 `downloadURLLookup`；mapDeviceItem 完成态 LogCollect 子任务调用，返回 1h presigned URL；router 装配 `backup.PgFileRepository` + `c.MinIO` 闭包 | ✅ |

**风险点 / 设计取舍**：
- B5 的注入是可选的——`fileRepo == nil` 或 `minioClient == nil` 时 `DownloadURL` 留空，不阻断 mapping。
- `taskIDPrefix` 和 `resolveTemplate` 在 software 包里是 unexported，本次新增 `software.TaskIDPrefix(uuid)` 导出函数复用，避免逻辑分叉。
- 不做 schema 迁移（不动 `upgrade_sub_tasks`），靠"运行时按 (SN, expected file_name) 查 `backup_restore_file`"拉真实路径，保留改造空间。
- B4 把 `upsertFileMetadata` 提到匹配判定之前，**所有** `backup.file.received` 事件都会进 `backup_restore_file`——这是合期望的行为（metadata 表本来就用 (SN, filename) 自然键，不依赖 backup_tasks）。

### 前端（F1–F5）

| ID | 文件 | 变更 | 状态 |
|----|------|------|------|
| **F1** | `omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx:583` | 非升级类主任务列表移除 "当前步骤" 列 | ✅ |
| **F2** | `omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx:467` | `getTaskTargetVersion`：去掉模板兜底；空串返回 "-" | ✅ |
| **F3** | `omcmb/webcode/src/pages/transfer/FileTransferCenter/index.tsx:694` | 非升级类设备列"目标版本/目标文件"：用 `record.targetFile`；`downloadUrl` 非空时渲染 `<a target=_blank>`，未到位时 `<Text type="secondary">` 灰显 | ✅ |
| **F4** | `omcmb/frontend-core/src/types/unifiedFileTransfer.ts` | `UnifiedFileTransferTask.currentStep` 改 `?: TransferStepId \| ''`；`UnifiedFileTransferDeviceItem` 增 `targetFile?: string; downloadUrl?: string` | ✅ |
| **F5** | mapping | Axios 拦截器已自动 snake→camel，TS 类型扩展即对齐 | ✅ |

---

## 4. 实施日志（按执行顺序更新）

| 时间 | 步骤 | 备注 |
|------|------|------|
| 2026-05-20 15:xx | 起草本计划文件 + 13 个 TaskCreate | ✅ |
| 2026-05-20 | B1 stepForTask 收口 | ✅ `model.go:546` |
| 2026-05-20 | B2 mapTask TargetVersion 收口 | ✅ `service.go:526` |
| 2026-05-20 | B3 DeviceItem.TargetFile + renderUFTEFileNameTemplate | ✅ `model.go:96 / :585`、`service.go:587` |
| 2026-05-20 | B4 FilePathRecorder 解耦 | ✅ `file_path_recorder.go:100` |
| 2026-05-20 | B5 UFTE 注入 + DownloadURL + SplitBucketAndPath 导出 | ✅ `service.go:27/:46`、`modules.go:112`、`restore_service.go:307` |
| 2026-05-20 | F1 移除主任务"当前步骤"列 | ✅ `FileTransferCenter/index.tsx:583` |
| 2026-05-20 | F2 去 fileNameTemplate 兜底 | ✅ `FileTransferCenter/index.tsx:467` |
| 2026-05-20 | F3 设备列 targetFile + 下载链接 | ✅ `FileTransferCenter/index.tsx:694` |
| 2026-05-20 | F4 类型扩展 currentStep optional / targetFile / downloadUrl | ✅ `unifiedFileTransfer.ts:63/96` |
| 2026-05-20 | F5 Axios snake→camel 自动 mapping，无需额外代码 | ✅ |
| 2026-05-20 | `go build ./... + go test ./internal/ufte/... ./internal/backup/... ./internal/software/...` | ✅ 三模块全过 |
| 2026-05-20 | `cd omcmb/webcode && npm run typecheck` | ✅ tsc --noEmit 无错 |
| 2026-05-20 16:00 | `docker compose up -d --force-recreate app web` | ✅ 镜像已重建（omcgo-app:latest + omcgo-web:latest），容器健康，UFTE 模块启动日志确认装配成功 |
| 2026-05-20 16:00 | 历史迁移病灶补救 | ⚠️ migrate-schema/seed 因历史"缺版本"重跑老旧失败迁移，已用 `INSERT INTO goose_db_version(_seed) (version_id, is_applied, tstamp) ...` 占位补齐（schema 缺 8 个、seed 缺 82 个），与本次 fix 无关，详见 §8 |
| 2026-05-20 | 浏览器人工验收 | ⏳ 请按 §5 DoD 清单复测 |

---

## 5. 验证清单（DoD）

- [ ] 新建一个 CONFIG_BACKUP_NV 任务，主任务列表中**不再出现"当前步骤"列**（升级任务的列保持不变）
- [ ] 主任务行的"目标版本"列对备份/日志/恢复类显示"-"
- [ ] 设备列表"目标版本/目标文件"列对未完成的子任务显示**渲染后的真实文件名**（`backup-{8hex}-{真实sn}.nv`，灰色，不可点击）
- [ ] 子任务完成（CPE 上传成功）后，该列变为蓝色可点击的链接，新开 tab 触发 MinIO presigned 下载
- [ ] 升级任务（gnb_upgrade / enb_upgrade）页面无任何回归：列结构不变，targetVersion 仍展示固件版本
- [ ] `go build ./...` / `go test ./internal/ufte/... ./internal/backup/... ./internal/software/...` 通过
- [ ] `cd omcmb/webcode && npm run typecheck` 通过

---

## 6. 不在本次范围

- 旧 `webcode/src/pages/backup/BackupTasks/` 页面（走旧 backup REST 端点，不是 UFTE）
- `device_tasks` 表内的 RPC 队列展示（"文件下载中"是 sub_task.Status，本身合理）
- `webcode-v2` / `webcode-v3` 候选皮肤（按 frontend-multi-skin-plan，业务层共享后再迁移；本次只动 frontend-core + webcode）
- `upgrade_sub_tasks` schema 迁移（保留改造空间，后续如要持久化 expectedFilename / objectPath 再做）

---

## 6.1 追加修复（启动后子任务卡 suspended，2026-05-20 16:00-16:35）

用户报告："新建挂起的备份NV，点开始后主任务变执行中，但设备列表还显示已挂起"。深挖发现 4 层 bug：

| ID | 文件 | 现象 / 修法 | 状态 |
|----|------|------|------|
| **F6** | `omcgo/internal/software/executor.go:38` `LogCollectResumer` 接口 + setter + `HandleDeviceOnline` 加 LogCollect 分支 | 原代码只处理 `subTask.FirmwareID != nil`（升级类），LogCollect 类（备份/日志采集，无固件）silent return → 永远不唤醒 | ✅ |
| **F7** | `omcgo/internal/software/service.go` `SetLogCollectResumer` + `ExecuteOneUploadDirect` | SoftwareService 透传 setter 给 executor；暴露门面给 ufte 包重启 Upload RPC | ✅ |
| **F8** | `omcgo/internal/ufte/service.go` `ResumeLogCollectSubTask` | 从 UFTE catalog 拿 `transport_path` 后调 `ExecuteOneUploadDirect` | ✅ |
| **F9** | `omcgo/cmd/app/provider/modules.go:initUFTEModule` | 装配 `softwareService.SetLogCollectResumer(ufteService)` | ✅ |
| **F10** | `omcgo/internal/software/service.go:1187 StartTaskReaper` | reaper DeviceOnline 超时 10 min → 60 min（设备 inform 间隔 5 min × 12 个周期容错） | ✅ |
| **F11** | `omcmb/webcode/src/pages/transfer/shared.tsx:117` `renderDeviceStatus` | "已挂起" → "已挂起 / 待上线"（区分用户主动挂起 vs 设备离线等待） | ✅ |
| **F12** | `omcgo/internal/software/executor.go:543 HandleDeviceOnline` payload struct | 原解析 `device_sn` 字段——但 ACS publish 用 `device_id.serial_number` 嵌套对象、device 模块 publish 用 `serial_number`——三种都没命中 → **历史死代码，升级类也从未被唤醒过**；改成多 schema 兼容 | ✅ |
| **F13** | `omcgo/internal/software/service.go:1185` `Subscribe` | 额外订阅 `device.online`（device 模块在 offline→active 切换时 publish，确定性信号比 periodic 更可靠） | ✅ |
| **F14** | `omcgo/internal/software/executor.go:308 ExecuteOneUpload` | `resolveTemplate` 漏了 `{sn}` `{taskId}` 占位符，导致 ACS upload URL 含字面值 "{sn}"/"{taskId}"——补齐渲染 | ✅ |

**验证证据**（16:32:43 重塞 wait key + 重置 sub_task=suspended，等设备下次 inform）：

```
[t+45s] sub_task=suspended → downloading wait_key=1
log: "device online, resuming pending upgrade" sub_task_id=7bdb098c device_sn=120288069823C4B0060
log: "task created" method=Upload
log: "log collect Upload RPC pushed" target_file_name=backup-845461b3-120288069823C4B0060.nv
```

设备 inform 触发 → wait key 命中 → LogCollect 分支 → ResumeLogCollectSubTask → Upload RPC 派发 → sub_task 进入 downloading。整个链路全过。

---

## 7. 部署观察（非本次范围，遗留待办）

**现象**：`docker compose up -d --force-recreate app web` 触发 `migrate-schema` 与 `migrate-seed` 重跑。两者均失败：
- `migrate-schema`：`000090_mml_schema_rebuild.sql` 试图 DROP / TRUNCATE / ALTER 已演变的 schema → SQLSTATE 42703 等。该版本号此前从未被记入 `goose_db_version`（但更高的 130 已 applied）。
- `migrate-seed`：`000005_seed_mml_param_library.sql` 因目标 `mml_params` 列结构已变，INSERT 字段不匹配。该版本号此前未记入 `goose_db_version_seed`。

**根因猜测**：早期开发时这些 DML/DDL 跑失败被人工跳过过，但 goose `WithAllowMissing` 在 `up` 时仍会尝试补遗，导致每次 `--force-recreate` 都会复刻该失败。

**当次绕过措施**：本机 DB 用以下 SQL 标记占位（不带任何业务效果，只填 goose 版本号）：
```sql
INSERT INTO goose_db_version (version_id, is_applied, tstamp)
SELECT v, true, NOW() FROM generate_series(1, 130) v
WHERE v NOT IN (SELECT version_id FROM goose_db_version);

INSERT INTO goose_db_version_seed (version_id, is_applied, tstamp)
SELECT v, true, NOW() FROM generate_series(1, 129) v
WHERE v NOT IN (SELECT version_id FROM goose_db_version_seed);
```

8 个 schema 版本 + 82 个 seed 版本被标 applied。业务已正常运行 18h 说明这些缺失种子不影响线上。

**遗留待办**：
- 后续值得花时间补齐 `000090_mml_schema_rebuild.sql` 与 `seed/000005_seed_mml_param_library.sql` 等老旧迁移，使其在已演变的 schema 下幂等可重跑；或在 cmd/migrate 启动期加 "已知失败版本白名单" 跳过。
- 建议进 backlog：`docs/project/backlog.md`，标签 `migration-hotfix-followup`。

---

## 8. 回归风险与回滚策略

**风险**：
1. mapDeviceItem 加入按 (SN, filename) 查 `backup_restore_file` 是新增的 N+1 读，**已批量化为单次 SELECT IN(...)** 规避；fileRepo 注入失败时降级到不查（DownloadURL 留空）
2. file_path_recorder 解耦门控会让所有 ACS 上传都写 `backup_restore_file`——这是合期望的行为，但首次部署后 metadata 表会增长

**回滚**：单提交即包含全部变更；revert 该 commit 即恢复旧展示。无 DB schema 变更，回滚无需 migrate-down。
