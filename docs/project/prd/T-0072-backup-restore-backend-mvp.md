# PRD: 备份恢复后端 MVP（T-0072 / R-102 followup）

> **关联**: Backlog T-0072 / Sprint-07 / Domain=F06/backup / Type=feat
> **作者**: Claude（代 Owner=架构+电信）
> **创建**: 2026-04-29
> **状态**: 草案 → 实施
> **关键决策**: TR-069 Download(FileType=3) 路径 + download handler 流式解压 + 直接路径输入（FE 拆 T-0078，task↔file_path 链路拆 T-0079）

---

## 1. 业务背景

T-0016 audit 发现 RestoreData 1123 行纯 mock，后端无 `/backup/restore` endpoint。R-102 Backup 五块（Tasks/FTP/Schedule/Policy/Restore）剩 Restore 未做。

T-0074/T-0077 已让 backup 文件以 `.gz/.zst/.lz4/.bz2` 落 MinIO，但 download handler 当前**不做解压** — CPE 从 ACS download endpoint 拉到的是原始压缩字节，**几乎所有 CPE 都不会自行解压**，restore 当前不可用。

---

## 2. ULTRATHINK 决策

### 2.1 "restore" 三种语义辨析

任务描述显式问"restore 是配置同步还是备份解压恢复"，实际有三种解读：

| # | 语义 | 实现 | 评估 |
|---|------|------|------|
| (1) 配置同步 | 解析备份 XML → SetParameterValues 批量推送 | OMC 自行解析厂商配置（运营商×厂商×参数树差异巨大） | ❌ 巨大且脆弱 |
| (2) 文件解压下载 | MinIO 对象解压 → 提供给运维下载查阅 | 简单但**不等于"恢复设备"** | ❌ 不是真正 restore |
| **(3) TR-069 Download** | ACS 发 `Download(FileType=3 Vendor Configuration File, URL)` → CPE 自行 GET + 应用 + 重启 | CPE 自己解析厂商私有格式（OMC 不参与），是 TR-069 标准做法 | ✅ **采纳** |

**选 (3) 的理由**：TR-069 标准把"vendor configuration file 的解析与应用"明确划给 CPE 而非 ACS；OMC 只管"拷贝文件给设备"。这与 T-0074 Upload(FileType=3) 完全对称（CPE 上传 ↔ CPE 下载），架构闭合。配置树差异、参数校验、reboot 时机全交给 CPE，OMC 不需做厂商适配。

### 2.2 解压发生在哪？

T-0074 落盘是 `.gz/.zst/.lz4/.bz2` 流式压缩。CPE 从 download endpoint 拉文件时**期望明文**（绝大多数 CPE 不内置 gzip/zstd/lz4/bzip2 解压链路）。

唯一干净的解压点：**`acs/download/handler.go` 内**，按对象扩展名 `.gz/.zst/.lz4/.bz2` 检测 → wrap MinIO `obj` reader 为对应 decompressor → 写到 ResponseWriter。
- 流式（不缓存全量），与 T-0074 压缩同模式
- Content-Disposition filename 去扩展名（CPE 看到的是 "cfg.xml"，不是 "cfg.xml.gz"）
- Content-Length **必须移除**（解压后大小未知）

**完美对称**：T-0074 上传时 wrap reader 加压缩 → MinIO put；T-0072 下载时 wrap MinIO obj 减压缩 → response。

### 2.3 backup_task ↔ file_path 链路

`BackupTask.FilePath *string` 字段已存（T-0016 时建 schema），**但当前没有任何代码写入**：执行器只 enqueue Upload device task，CPE 上传文件落 MinIO，但**不回写 backup_task 行**。

要让 restore 通过 `backup_task_id` 找到文件，需要：
- ACS upload handler 解析"这次上传属于哪个 backup_task"（需在 Upload RPC URL 嵌 `backup_task_id` query / TargetFileName 编码 / 监听 TransferComplete + commandKey 反查）
- 然后 update backup_tasks set file_path = ...

这是独立的"数据回灌"工作，单独立 **T-0079**（拟）。本 MVP **不做** task→path 链路；restore endpoint 直接接受 `{bucket, object_path, target_devices}` 三元组，**path 由调用方提供**。

短期 path 来源：
- 运维通过 filemanager 浏览 MinIO 找路径
- 后续 FE T-0078 RestoreData 实现"backup 文件浏览器"
- T-0079 完成后可加 `restore_by_task_id` 模式（API 增强，向下兼容）

### 2.4 Restore 任务追踪

复用 `backup_tasks` 表 vs 建独立 `restore_tasks` 表？

| 方案 | 优点 | 缺点 |
|------|------|------|
| 复用 backup_tasks（加 task_type=restore） | 单表 + 现有 list/cancel/delete 路由复用 | TaskType 枚举从 full/incremental/config_only 扩到含 restore 语义膨胀 |
| **独立 restore_tasks 表** | 语义清晰；schema 可针对 restore 语义（多设备 N 行 vs backup 1 行的不对称问题）；前端表格需求不同 | 1 张新表 + handler/repo + DI |

**采纳独立表 `restore_tasks`**：每个 restore 请求 = 1 行；TargetIDs 数组 + 每设备的 status / progress 通过链接到 device_tasks 表（已有）。MVP schema：
- id（PK）, source_object_path, source_bucket, target_device_sns（jsonb），status, progress, error_message, started_at, completed_at, created_at, updated_at, created_by

注意：**device_tasks 是 partition table**（CLAUDE.md §5.5.3），无法 FK；用应用层 join。

### 2.5 Scope 边界（FE 拆 T-0078 / data 链路拆 T-0079）

| 子项 | 决策 |
|------|------|
| 后端 download handler 解压 | **本任务** |
| 后端 restore service + endpoint | **本任务** |
| 后端 restore_tasks 表 | **本任务**（migration 000048） |
| 后端 e2e claim | **本任务** |
| 前端 RestoreData 重写（1123 行 mock）+ 文件浏览器 + useBackupRestore hook | **T-0078**（新登记，est=L FE-only） |
| backup_task → file_path 回填 | **T-0079**（新登记，est=M，含 upload handler 钩子 / TransferComplete 监听） |
| restore_by_task_id API 模式 | T-0079 之后增强；本任务 API 设计向前兼容 |

---

## 3. 用户故事

| 角色 | 故事 |
|------|------|
| 运维 | 我有一份昨天备份的 cfg.xml.gz，今天发现设备配置坏了，想把昨天的备份恢复到设备 |
| 网管开发 | 给我一个 `POST /backup/restore` endpoint，传入 `{bucket, object_path, target_devices: [sn1...]}`，后端把每台设备 enqueue Download 任务 |
| QA | e2e 能验证 endpoint 存在 + 401/400/200 路径覆盖 |
| 后端开发者 | T-0078 FE 接入时，我希望 endpoint 契约稳定，不必改 |

---

## 4. 验收标准（GWT）

### V1 — 解压：gzip 对象的 download
- **Given** MinIO 存有 `config_backup/backup/2026/04/29/cfg.xml.gz`（gzip 压缩）
- **When** GET `/smallcell/FileDownloadService/config_backup/backup/2026/04/29/cfg.xml.gz` (with auth)
- **Then** 200 OK；response body 为解压后的明文 XML；`Content-Disposition: attachment; filename="cfg.xml"`（去 .gz）；**无 Content-Length** 头（流式未知大小）

### V2 — 解压：zstd 对象同理
- 同 V1 但 `.zst` 扩展名

### V3 — 透传：无扩展名 / 非压缩对象
- **Given** MinIO 有 `firmware/img/v2.0.bin`
- **When** GET 该路径
- **Then** 行为不变（与 T-0072 之前一致），含 Content-Length

### V4 — POST /backup/restore 创建 restore 任务
- **Given** policy 通；MinIO 有 `config_backup/backup/2026/04/29/cfg.xml.gz`；3 个目标设备 SN
- **When** `POST /api/v1/backup/restore` body=`{"bucket":"config_backup","object_path":"backup/2026/04/29/cfg.xml.gz","target_device_sns":["SN001","SN002","SN003"]}`
- **Then** 200 OK；response 含 `restore_task_id`；DB `restore_tasks` 1 行 status=pending；3 个 device_tasks 行 method=Download params 含 url="config_backup/backup/2026/04/29/cfg.xml.gz" file_type="3"

### V5 — 路径校验：拒绝越界
- **When** body `bucket="any" object_path="../../etc/passwd"` 或路径包含 `..` 或绝对路径
- **Then** 400 Bad Request；response 含错误信息

### V6 — 设备不存在：跳过 + 部分成功
- **Given** target_device_sns 含 1 个不存在的 SN
- **When** POST
- **Then** 200 OK；restore_tasks 行存在；存在的设备 enqueue device_task；不存在设备记入 error_message JSON 列；progress 计算包含跳过

### V7 — restore_tasks 列表 GET 端点（最小观测）
- **When** GET `/api/v1/backup/restore-tasks`
- **Then** 200 OK；返回最近 N 条 restore 任务（带 status/progress/started_at）

### V8 — 4 算法兼容（regression × T-0074/T-0077）
- gzip/zstd/lz4/bzip2 文件 download 全部解压成功；魔术字节正确移除（响应 body 不含原压缩 magic）

---

## 5. 运营商差异矩阵

无差异。restore 走 TR-069 标准 Download(FileType=3)，三家运营商均支持；CPE 自行处理厂商私有 config 解析。

---

## 6. 非目标

| # | 非目标 | 后续承接 |
|---|--------|---------|
| N1 | 前端 RestoreData 页面重写 + 文件浏览器 + Hook 接入 | T-0078（新） |
| N2 | backup_task_id → file_path 链路回填 | T-0079（新） |
| N3 | restore_by_task_id API 模式 | T-0079 后增强 |
| N4 | restore 进度实时回写（device_task 完成后更新 restore_tasks.progress） | 可后续做；MVP progress=0 直到所有 device_tasks 完成 |
| N5 | 跨厂商 / 跨型号 restore 兼容性校验（不同 OUI 之间是否能互相 restore） | CPE 自行处理 |
| N6 | 加密备份 restore（先解密后下发） | T-0075（加密任务）一并设计 |
| N7 | restore 回滚 / undo（再 restore 一次更早的备份即可，不做特殊 undo 语义） | — |

---

## 7. 依赖

| 依赖 | 状态 |
|------|------|
| T-0074 ✅ + T-0077 ✅ | 4 算法压缩落盘 |
| `acs/download/handler.go` | 已存在；本任务扩展 |
| `acs/rpc/dispatcher.go` Download RPC | 已存在；URL 翻译已支持 |
| `device_tasks` 表 + Method=Download | 已支持 |
| `pkg/tr069.FileTypeConfig = "3"` | ✅ |

无新外部依赖（解压 lib 全部已在 go.mod，T-0074/T-0077 已添加）。

---

## 8. 度量

| metric | 说明 |
|--------|------|
| `omc_backup_restore_requests_total{result}` | counter；result ∈ {accepted, rejected_invalid_path, rejected_no_target} |
| `omc_backup_restore_devices_enqueued_total` | counter；累计 enqueue 的 device_tasks 数 |
| `omc_backup_download_decompress_total{format}` | counter；download 解压触发次数 |
| `omc_backup_download_decompress_errors_total{format,reason}` | counter；解压失败 |

复用 `backup.PolicyMetrics` 命名空间（前缀 `omc_backup_*`），新增 4 个 collector 在 `restore_metrics.go`（独立文件保持模块边界）。

---

## 9. 设计备忘（S2）

### 9.1 Download handler 解压改造

新增 `acs/download/decompress.go`：

```go
// detectCompression returns the Decompressor + clean filename if the path has
// a recognized compressed extension; ok=false means pass-through.
func detectCompression(objectPath string) (decomp Decompressor, cleanName string, ok bool)

type Decompressor interface {
    Wrap(src io.Reader) (io.ReadCloser, error)
    Format() string  // for metrics label
}

// gzipDecompressor / zstdDecompressor / lz4Decompressor / bzip2Decompressor
// — mirror the Compressor interface from internal/backup/compression.go
```

ServeHTTP 改造：
```go
obj, err := h.minioClient.GetObject(...)
// ... existing code ...
filename := filepath.Base(objectPath)

decomp, cleanName, doDecompress := detectCompression(objectPath)
if doDecompress {
    filename = cleanName  // strip .gz/.zst/.lz4/.bz2
    body, err := decomp.Wrap(obj)
    if err != nil {
        // metric: decompress_errors_total{format, reason="open"}
        // fallback: serve compressed (best-effort)
    } else {
        defer body.Close()
        // skip Content-Length (decompressed size unknown)
        w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
        w.Header().Set("Content-Type", "application/octet-stream")
        io.Copy(w, body)
        // metric: decompress_total{format}
        return
    }
}
// pass-through path (existing behavior)
```

**关键决策**：
- 解压 lib 同 T-0074/T-0077（gzip stdlib / zstd klauspost / lz4 pierrec / bzip2 dsnet read = stdlib `compress/bzip2`）
- 解压失败 fallback to 透传（可疑文件直接给 CPE，不阻断），metric 记 error
- 流式 io.Copy（不缓存全量）

### 9.2 Restore service & endpoint

新文件 `internal/backup/restore_service.go`：

```go
type RestoreService struct {
    repo         RestoreTaskRepository
    deviceRepo   device.DeviceRepository
    taskSvc      devtask.Enqueuer
    minioClient  *minio.Client
    bucket       string  // restrict to "config_backup" by default
    metrics      *RestoreMetrics
    logger       *zap.Logger
}

type CreateRestoreRequest struct {
    Bucket          string   `json:"bucket" binding:"required"`
    ObjectPath      string   `json:"object_path" binding:"required"`
    TargetDeviceSNs []string `json:"target_device_sns" binding:"required,min=1"`
}

func (s *RestoreService) Create(ctx context.Context, req *CreateRestoreRequest) (*RestoreTask, error) {
    // 1. validate path scope (clean + reject ..)
    // 2. HEAD MinIO object — reject 404
    // 3. create restore_tasks row (pending)
    // 4. for each device SN: enqueue device task Method=Download params={file_type:3, url:"{bucket}/{object_path}", target_filename:"restore.cfg"}
    // 5. return restore task
}
```

新 handler `restore_handler.go`：
- `POST /api/v1/backup/restore` → Create
- `GET /api/v1/backup/restore-tasks` → List
- `GET /api/v1/backup/restore-tasks/:id` → GetByID

### 9.3 Migration 000048 — restore_tasks

```sql
-- +goose Up
CREATE TABLE IF NOT EXISTS restore_tasks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_bucket       VARCHAR(64) NOT NULL,
    source_object_path  TEXT NOT NULL,
    target_device_sns   JSONB NOT NULL DEFAULT '[]',
    status              VARCHAR(16) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','running','completed','failed','cancelled')),
    progress            INT NOT NULL DEFAULT 0 CHECK (progress >= 0 AND progress <= 100),
    error_message       TEXT,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          VARCHAR(64)  -- audit user; nullable for system-triggered
);

CREATE INDEX IF NOT EXISTS idx_restore_tasks_status ON restore_tasks(status);
CREATE INDEX IF NOT EXISTS idx_restore_tasks_created_at ON restore_tasks(created_at DESC);

-- updated_at trigger uses shared function from 000001
CREATE TRIGGER trg_restore_tasks_updated
BEFORE UPDATE ON restore_tasks
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
DROP TRIGGER IF EXISTS trg_restore_tasks_updated ON restore_tasks;
DROP INDEX IF EXISTS idx_restore_tasks_created_at;
DROP INDEX IF EXISTS idx_restore_tasks_status;
DROP TABLE IF EXISTS restore_tasks;
```

**Schema 自查**（CLAUDE.md §5.5.10）：
- [x] 版本号 = 000048 = 47+1，无跳跃
- [x] up/down 段全
- [x] 无 PL/pgSQL DO 块（无 StatementBegin/End 需求）
- [x] 无 INSERT
- [x] 无 hypertable
- [x] 无分区表 FK
- [x] 无带 WHERE 的 UNIQUE
- [x] UUID 用 gen_random_uuid()
- [x] Down 删除所有 Up 创建的对象（trigger + index + table）
- [x] 复用 000001 的 `update_updated_at_column()` 共享函数（不重复创建）

### 9.4 路径校验

```go
func validateRestorePath(bucket, objectPath string) error {
    if bucket == "" || objectPath == "" {
        return fmt.Errorf("bucket and object_path required: %w", ErrInvalidInput)
    }
    cleanedBucket := pathpkg.Clean(bucket)
    cleanedPath := pathpkg.Clean(objectPath)
    if strings.Contains(cleanedBucket, "..") || strings.Contains(cleanedPath, "..") ||
       strings.HasPrefix(cleanedBucket, "/") || strings.HasPrefix(cleanedPath, "/") {
        return fmt.Errorf("path traversal: %w", ErrInvalidInput)
    }
    // MVP: restrict to config_backup bucket only (operators cannot restore from
    // arbitrary buckets like firmware/pm to prevent abuse).
    if cleanedBucket != "config_backup" {
        return fmt.Errorf("only config_backup bucket allowed for restore: %w", ErrInvalidInput)
    }
    return nil
}
```

### 9.5 文件清单

新增：
- `omcgo/internal/acs/download/decompress.go`（~180 行）
- `omcgo/internal/acs/download/handler_test.go` 或 decompress_test.go（已无 test，新建）
- `omcgo/internal/backup/restore_model.go` —RestoreTask + 状态枚举
- `omcgo/internal/backup/restore_pg_repository.go` — squirrel-based repo
- `omcgo/internal/backup/restore_repository.go` — interface
- `omcgo/internal/backup/restore_service.go` — Create / List / GetByID
- `omcgo/internal/backup/restore_service_test.go`
- `omcgo/internal/backup/restore_handler.go` — gin handlers
- `omcgo/internal/backup/restore_metrics.go` — 4 collectors
- `omcgo/migrations/000048_restore_tasks.sql`

修改：
- `omcgo/internal/acs/download/handler.go` — 调用 detectCompression + decomp.Wrap
- `omcgo/internal/backup/handler.go` — 注册 restore endpoints + RestoreService DI hook
- `omcgo/internal/backup/repository.go` — 加 RestoreTaskRepository interface
- `omcgo/cmd/app/provider/modules.go` — DI: NewPgRestoreRepo + NewRestoreService → handler.SetRestoreService
- `omcgo/cmd/acs/main.go` — pass MetricsReg into download handler for decompress metrics（or skip metrics on ACS — restore metrics live in app process）
- `omcgo/scripts/e2e_verify.sh` — +2 claims：bk-9 POST /backup/restore + bk-10 GET /backup/restore-tasks

### 9.6 Carrier 差异

无（PRD §5）。本任务不触 `internal/carrier/`。

### 9.7 待定点

| 待定 | 决策 |
|------|------|
| restore 进度回写设备级别完成 | MVP 不做（progress=0 → 100 二分跳变；细化推 N4 followup） |
| restore_tasks 是否记 created_by | 加，从 gin context 取（middleware 注入），nullable for system-triggered |
| download 解压 metrics 是放 ACS 进程还是 app 进程 | 放 ACS 进程（解压发生在 ACS）— ACS 进程已有 `inf.MetricsReg`；新加 `download.NewDecompressMetrics` |

---

## 10. 后续 followup（S7 登记）

| ID | 内容 | Est | 依赖 |
|----|------|-----|------|
| **T-0078** | 前端 RestoreData wholesale rewrite + useBackupRestore hook + 文件浏览器（接 filemanager listing）+ i18n | L | T-0072 ✅ |
| **T-0079** | backup_task → file_path 链路回填（upload handler hook / TransferComplete 监听 + restore_by_task_id 模式增强） | M | T-0072 ✅, T-0007 ✅ |
