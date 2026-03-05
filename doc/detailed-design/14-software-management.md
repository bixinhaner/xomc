# DD-14: 固件/软件管理（F06 子模块）

> 关联功能域：F06（OMC-R 核心功能）
> 关联 backend-design.md 章节：第二章（模块 6 — omcr/software）
> 实施阶段：Phase 4（北向与规模化）
> 依赖文档：DD-07, DD-09

---

## 1. 概述

### 1.1 模块定位

固件/软件管理（`internal/omcr/software/`）负责基站设备的固件版本管理和远程升级编排，通过 TR069 Download RPC 实现固件推送。

### 1.2 核心职责

- 固件版本管理（版本号、兼容设备列表、文件存储）
- 升级工作流（下载 → 传输 → 重启 → 验证）
- 升级任务调度（单设备、批量、滚动升级）
- 升级失败恢复

---

## 2. 接口设计

### 2.1 SoftwareService — `internal/omcr/software/service.go`

```go
type SoftwareService struct {
    firmwareRepo FirmwareRepository
    upgradeRepo  UpgradeTaskRepository
    cmdQueue     acs.CommandQueue
    connReq      *connreq.ConnReqClient
    minioClient  *minio.Client
    eventBus     event.EventBus
    logger       *zap.Logger
}

// UploadFirmware 上传固件文件
func (s *SoftwareService) UploadFirmware(ctx context.Context, fw *FirmwareVersion, file io.Reader) error

// StartUpgrade 启动单设备升级
func (s *SoftwareService) StartUpgrade(ctx context.Context, deviceID, firmwareID uuid.UUID) (*UpgradeTask, error)

// BatchUpgrade 批量升级
func (s *SoftwareService) BatchUpgrade(ctx context.Context, deviceIDs []uuid.UUID, firmwareID uuid.UUID) ([]UpgradeTask, error)

// HandleTransferComplete 处理升级完成事件
func (s *SoftwareService) HandleTransferComplete(ctx context.Context, evt event.Event) error
```

### 2.2 数据模型

```go
type FirmwareVersion struct {
    ID            uuid.UUID
    Carrier       model.CarrierCode
    ProductClass  string
    Version       string
    FileName      string
    FileSize      int64
    MinIOPath     string
    CompatibleOUI []string
    ReleaseNotes  string
    CreatedAt     time.Time
}

type UpgradeTask struct {
    ID           uuid.UUID
    DeviceID     uuid.UUID
    FirmwareID   uuid.UUID
    Status       string // pending, downloading, rebooting, verifying, completed, failed
    ErrorMessage string
    StartedAt    *time.Time
    CompletedAt  *time.Time
    CreatedAt    time.Time
}
```

### 2.3 升级工作流

```
1. 创建 UpgradeTask (status=pending)
2. 向设备命令队列推入 Download RPC (FileType="1 Firmware")
3. 发送 Connection Request 触发设备连接
4. ACS 执行 Download → CPE 开始下载固件
5. CPE 下载完成 → TransferComplete Inform
6. ACS 发送 Reboot → CPE 重启
7. CPE 重启后 Inform (BOOT事件)
8. ACS 读取 FirmwareVersion 参数验证
9. 版本匹配 → completed / 不匹配 → failed
```

---

## 3. 数据库 Schema

```sql
CREATE TABLE firmware_versions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier        VARCHAR(4) NOT NULL,
    product_class  VARCHAR(64),
    version        VARCHAR(64) NOT NULL,
    file_name      VARCHAR(256) NOT NULL,
    file_size      BIGINT,
    minio_path     VARCHAR(512) NOT NULL,
    compatible_oui JSONB,
    release_notes  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE upgrade_tasks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id     UUID NOT NULL REFERENCES devices(id),
    firmware_id   UUID NOT NULL REFERENCES firmware_versions(id),
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### MinIO 存储路径

```
firmware/{carrier}/{product_class}/{version}/firmware.bin
```

### REST API

```
GET    /api/v1/firmware                        固件版本列表
POST   /api/v1/firmware                        上传固件
POST   /api/v1/devices/{id}/firmware-upgrade   触发升级
GET    /api/v1/upgrade-tasks                   升级任务列表
```

---

## 4. 实施子阶段

### 阶段 14a：版本管理 + 文件存储（Phase 4）
### 阶段 14b：升级工作流 + 批量升级（Phase 4）

---

## 5. 文件清单

```
internal/omcr/software/service.go
internal/omcr/software/repository.go
internal/omcr/software/upgrade.go
```

---

## 6. 参考

- doc/features/06-omc-core-functions.md：F06 OMC-R 核心功能
