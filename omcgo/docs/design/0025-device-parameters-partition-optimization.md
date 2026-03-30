# 0025 device_parameters 表分区优化 — 技术方案

> 对 device_parameters 表进行 Hash 分区改造，新增 fap_instance + param_group 分类列，
> 消除所有 LIKE 查询，支撑 10 万～100 万设备 × 500 参数的规模。

---

## 1. 背景与目标

### 1.1 当前状况

`device_parameters` 是 EAV（Entity-Attribute-Value）结构的 K-V 表，存储每台设备的全部 TR069 参数：

```sql
-- migrations/000002_create_device_parameters.up.sql
CREATE TABLE device_parameters (
    device_id        UUID NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN DEFAULT false,
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, parameter_path)
);
CREATE INDEX idx_device_params_device ON device_parameters (device_id);

-- migrations/000061_device_parameters_path_index.up.sql
CREATE INDEX idx_device_params_path_prefix
ON device_parameters (device_id, parameter_path varchar_pattern_ops);
```

**存在的问题**：

| 问题 | 影响 |
|------|------|
| 中缀 LIKE 无法走索引 | `LIKE '%MmePoolConfigParam.%'` 在 5000 万行上全表扫描 |
| 冗余索引 | `idx_device_params_device` 被 PK 首列覆盖；`varchar_pattern_ops` 索引为 LIKE 补丁，代价高 |
| 无分区 | 单表 5000 万～5 亿行时 VACUUM 阻塞 2+ 小时 |
| 语义模糊 | Go 代码用 `filterByPrefix(allParams, "MmePoolConfigParam.")` 做内存字符串匹配 |

### 1.2 容量规模

| 设备规模 | 参数/设备 | 总行数 | 表数据 | PK 索引 | 合计 |
|---------|----------|--------|--------|---------|------|
| 10 万 | 500 | 5,000 万 | 8.6 GB | 3.5 GB | ~12 GB |
| 100 万 | 500 | 5 亿 | 86 GB | 35 GB | ~121 GB |

写入压力（Inform 间隔 5 分钟）：

| 指标 | 10 万设备 | 100 万设备 |
|------|----------|-----------|
| BatchUpsert 频率 | 333 次/秒 | 3,333 次/秒 |
| 行 UPSERT 频率 | 166,500 行/秒 | 1,665,000 行/秒 |

### 1.3 设计目标

1. **消除 LIKE** — 所有查询变等值匹配
2. **Hash 分区** — VACUUM 并行化，单分区 <5 分钟
3. **减少索引** — 删除 2 个冗余索引，仅保留 PK
4. **代码最小改动** — 分区对应用透明，仅需调整写入和部分读取方法

---

## 2. 方案设计

### 2.1 Schema 变更

```sql
CREATE TABLE device_parameters (
    device_id        UUID         NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN      DEFAULT false,
    last_updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    fap_instance     SMALLINT     NOT NULL DEFAULT 0,    -- 新增
    param_group      VARCHAR(32)  NOT NULL DEFAULT 'other', -- 新增
    PRIMARY KEY (device_id, parameter_path)
) PARTITION BY HASH (device_id);

-- 32 个分区（不再需要任何额外索引）
```

### 2.2 新增列说明

| 列 | 类型 | 说明 | 提取规��� |
|----|------|------|---------|
| `fap_instance` | SMALLINT | FAPService 小区编号 | 路径 `FAPService.{N}` → N，其余 → 0 |
| `param_group` | VARCHAR(32) | 功能分组标签 | 按路径关键字匹配 12 个分组 |

### 2.3 分区策略

**32 个 Hash 分区 by device_id**：

| 指标 | 每分区（10万） | 每分区（100万） |
|------|-------------|---------------|
| 行数 | 156 万 | 1,562 万 |
| 表数据 | ~274 MB | ~2.7 GB |
| PK 索引 | ~110 MB | ~1.1 GB |
| VACUUM 耗时 | <1 分钟 | <5 分钟 |

### 2.4 索引策略：仅 PK

**不新增任何���级索引**。理由：

所有 device_parameters 查询都是 device-scoped（先过滤 device_id）。PK 索引扫描返回该设备全部 ~500 行，再按 `param_group` / `fap_instance` 内存过滤，总延迟 <1ms。

二级索引在此场景下收益 <0.1ms，但写入代价巨大：
- 每次 BatchUpsert 500 行 × 每个二级索引 1 次写入 = 166,500 额外 index ops/s（10 万规模）

**同时删除现有冗余索引**：

| 删除的索引 | 原因 |
|-----------|------|
| `idx_device_params_device` | PK 首列已是 device_id，完全冗余 |
| `idx_device_params_path_prefix` | LIKE 查询已被 param_group 等值取代 |

### 2.5 param_group 分类规则

12 个分组，按匹配优先级排列：

| param_group | 匹配规则 | 参数数 |
|---|---|---|
| `mme_pool` | 含 `MmePoolConfigParam.` 或 `X_COM_MmePool.` | ~51 |
| `license` | 含 `X_COM_LICENSE.` | ~42 |
| `antenna` | 含 `AntennaInfo.` | ~7 |
| `alarm` | 含 `FaultMgmt.CurrentAlarm.` | ~30 |
| `sync` | 含 `X_COM_GPS`/`X_COM_BDS`/`X_COM_1588`/`X_COM_GLONASS` 或前缀 `Device.FAP.GPS.` 或含 `tfcsSync`/`tfcsManager` | ~12 |
| `radio` | 含 `CellConfig` 且含 `RAN.`/`EPC.PLMN` | ~10 |
| `fap_control` | 含 `FAPControl.`（前面未命中） | ~15 |
| `device_info` | 前缀 `Device.DeviceInfo.`（前面未命中） | ~30 |
| `management` | 前缀 `Device.ManagementServer.`（前面未命中） | ~5 |
| `ipsec` | 含 `Ipsec` 或 `IPSEC` | ~5 |
| `network` | 前缀 `Device.IP.` | ~2 |
| `software` | 前缀 `Device.SoftwareCtrl.` | ~1 |
| `other` | 以上都不匹配 | ~2 |

---

## 3. 影响范围分析

### 3.1 需要修改的文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `migrations/000064_*.sql` | **新建** | 分区表重建 + 数据迁移 |
| `internal/device/param_classify.go` | **新建** | `ExtractFAPInstance` + `ClassifyParamGroup` |
| `internal/device/param_classify_test.go` | **新建** | 分类函数测试 |
| `internal/core/model/parameter.go` | **修改** | DeviceParameter 结构体新增 2 字段 |
| `internal/device/param_repository.go` | **修改** | 接口新增 3 个方法 |
| `internal/device/pg_param_repository.go` | **修改** | 实现新方法 + 修改 BatchUpsert + 修改列名列表 |
| `internal/device/service.go` | **修改** | `GetDeviceDetailComposite` 用 GetByGroup 替代 filterByPrefix |
| `internal/device/info_sync.go` | **微调** | `SyncFromParameters` 可选优化（仅查需要的分组） |
| `internal/device/handler_test.go` | **修改** | mock 实现增加新接口方法 |
| `internal/device/service_test.go` | **修改** | mock 实现增加新接口方法 |
| `internal/device/inform_handler_test.go` | **修改** | mock 实现增加新接口方法 |

### 3.2 不需要修改的文件

| 文件 | 原因 |
|------|------|
| `param_handler.go` | 参数树展示仍使用 `GetByDevice`/`GetByPathPrefix`/`SearchByKeyword`/`GetDirectChildLeaves`���这些方法不变 |
| `detail_assembler.go` | 入参不变（`[]model.DeviceParameter`），只是调用方传入方式变了 |
| `detail_dto.go` | DTO 类型不变 |
| `info_calc.go` | 计算函数入参是 `map[string]string`，不涉及数据库 |

---

## 4. 实施步骤

### Phase 1: 新建分类函数（无破坏性）

**新建文件**：`internal/device/param_classify.go`

```go
package device

import (
    "strconv"
    "strings"
)

// ExtractFAPInstance 从 TR069 路径中提取 FAPService 实例编号。
// 非 FAPService 路径返回 0。
func ExtractFAPInstance(path string) int {
    const marker = "FAPService."
    idx := strings.Index(path, marker)
    if idx < 0 {
        return 0
    }
    rest := path[idx+len(marker):]
    dotIdx := strings.Index(rest, ".")
    if dotIdx < 0 {
        return 0
    }
    n, err := strconv.Atoi(rest[:dotIdx])
    if err != nil {
        return 0 // FAPService.Ipsec 等非数字路径
    }
    return n
}

// ClassifyParamGroup 根据 TR069 路径返回功能分组标签。
func ClassifyParamGroup(path string) string {
    switch {
    case strings.Contains(path, "MmePoolConfigParam.") ||
        strings.Contains(path, "X_COM_MmePool."):
        return "mme_pool"
    case strings.Contains(path, "X_COM_LICENSE."):
        return "license"
    case strings.Contains(path, "AntennaInfo."):
        return "antenna"
    case strings.Contains(path, "FaultMgmt.CurrentAlarm."):
        return "alarm"
    case strings.Contains(path, "X_COM_GPS") ||
        strings.Contains(path, "X_COM_BDS") ||
        strings.Contains(path, "X_COM_1588") ||
        strings.Contains(path, "X_COM_GLONASS") ||
        strings.HasPrefix(path, "Device.FAP.GPS.") ||
        strings.Contains(path, "tfcsSync") ||
        strings.Contains(path, "tfcsManager"):
        return "sync"
    case strings.Contains(path, "CellConfig") &&
        (strings.Contains(path, "RAN.") || strings.Contains(path, "EPC.PLMN")):
        return "radio"
    case strings.Contains(path, "FAPControl."):
        return "fap_control"
    case strings.HasPrefix(path, "Device.DeviceInfo."):
        return "device_info"
    case strings.HasPrefix(path, "Device.ManagementServer."):
        return "management"
    case strings.Contains(path, "Ipsec") || strings.Contains(path, "IPSEC"):
        return "ipsec"
    case strings.HasPrefix(path, "Device.IP."):
        return "network"
    case strings.HasPrefix(path, "Device.SoftwareCtrl."):
        return "software"
    default:
        return "other"
    }
}
```

**新建文件**：`internal/device/param_classify_test.go`

Table-driven 测试覆盖所有 12 ���分组 + fap_instance 提取，验证优先级和边界情况。

### Phase 2: 修改 Model 结构体

**修改文件**：`internal/core/model/parameter.go`

```go
type DeviceParameter struct {
    DeviceID       uuid.UUID     `json:"device_id" db:"device_id"`
    ParameterPath  string        `json:"parameter_path" db:"parameter_path"`
    ParameterValue string        `json:"parameter_value" db:"parameter_value"`
    ParameterType  ParameterType `json:"parameter_type" db:"parameter_type"`
    Writable       bool          `json:"writable" db:"writable"`
    LastUpdatedAt  time.Time     `json:"last_updated_at" db:"last_updated_at"`
    FAPInstance    int           `json:"fap_instance" db:"fap_instance"`       // 新增
    ParamGroup     string        `json:"param_group" db:"param_group"`         // 新增
}
```

### Phase 3: 扩展 Repository 接口

**修改文件**：`internal/device/param_repository.go`

在现有接口末尾新增 3 个方法：

```go
type DeviceParameterRepository interface {
    // === 现有 8 个方法保持不变 ===
    BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
    GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
    GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)
    DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error
    GetByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error)
    CountByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) (int, error)
    SearchByKeyword(ctx context.Context, deviceID uuid.UUID, keyword string, limit int) ([]model.DeviceParameter, error)
    GetDirectChildLeaves(ctx context.Context, deviceID uuid.UUID, prefix string, limit, offset int) ([]model.DeviceParameter, int, error)

    // === 新增：按分组查询（取代 LIKE） ===
    GetByGroup(ctx context.Context, deviceID uuid.UUID, group string) ([]model.DeviceParameter, error)
    GetByFAPInstance(ctx context.Context, deviceID uuid.UUID, instance int) ([]model.DeviceParameter, error)
    GetByFAPInstanceAndGroup(ctx context.Context, deviceID uuid.UUID, instance int, group string) ([]model.DeviceParameter, error)
}
```

### Phase 4: 修改 PG 实现

**修改文件**：`internal/device/pg_param_repository.go`

**4.1 所有 SELECT 列名加入新字段**

定义 helper 常量避免重复：

```go
var paramColumns = []string{
    "device_id", "parameter_path", "parameter_value",
    "parameter_type", "writable", "last_updated_at",
    "fap_instance", "param_group",
}
```

**4.2 所有 Scan 调用加入新字段**

```go
rows.Scan(&p.DeviceID, &p.ParameterPath, &p.ParameterValue,
    &p.ParameterType, &p.Writable, &p.LastUpdatedAt,
    &p.FAPInstance, &p.ParamGroup)
```

受影响方法（7 处 Scan）：
- `GetByDevice` (L75)
- `GetByPath` (L97)
- `GetByPathPrefix` (L136)
- `SearchByKeyword` (L188)
- `GetDirectChildLeaves` (L248)
- 新增：`GetByGroup`
- 新增：`GetByFAPInstance`
- 新增：`GetByFAPInstanceAndGroup`

**4.3 修改 BatchUpsert — 写入时自动分类**

```go
func (r *PgDeviceParameterRepository) BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error {
    if len(params) == 0 {
        return nil
    }
    batch := &pgx.Batch{}
    now := time.Now()

    for _, p := range params {
        query, args, err := psql.Insert("device_parameters").
            Columns("device_id", "parameter_path", "parameter_value",
                "parameter_type", "writable", "last_updated_at",
                "fap_instance", "param_group").                        // ← 新增 2 列
            Values(deviceID, p.ParameterPath, p.ParameterValue,
                p.ParameterType, p.Writable, now,
                ExtractFAPInstance(p.ParameterPath),                    // ← 自动提取
                ClassifyParamGroup(p.ParameterPath)).                  // ← 自动分类
            Suffix("ON CONFLICT (device_id, parameter_path) DO UPDATE SET " +
                "parameter_value = EXCLUDED.parameter_value, " +
                "parameter_type = EXCLUDED.parameter_type, " +
                "writable = EXCLUDED.writable, " +
                "last_updated_at = EXCLUDED.last_updated_at").         // ← 不更新分类列
            ToSql()
        if err != nil {
            return fmt.Errorf("build upsert query: %w", err)
        }
        batch.Queue(query, args...)
    }
    // ... 后续不变
}
```

**4.4 新增 3 个查询方法**

```go
func (r *PgDeviceParameterRepository) GetByGroup(ctx context.Context, deviceID uuid.UUID, group string) ([]model.DeviceParameter, error) {
    query, args, err := psql.Select(paramColumns...).
        From("device_parameters").
        Where(sq.Eq{"device_id": deviceID, "param_group": group}).
        OrderBy("parameter_path ASC").
        ToSql()
    // ... 标准 rows 扫描逻辑
}

func (r *PgDeviceParameterRepository) GetByFAPInstance(ctx context.Context, deviceID uuid.UUID, instance int) ([]model.DeviceParameter, error) {
    query, args, err := psql.Select(paramColumns...).
        From("device_parameters").
        Where(sq.Eq{"device_id": deviceID, "fap_instance": instance}).
        OrderBy("parameter_path ASC").
        ToSql()
    // ... 标准 rows 扫描逻辑
}

func (r *PgDeviceParameterRepository) GetByFAPInstanceAndGroup(ctx context.Context, deviceID uuid.UUID, instance int, group string) ([]model.DeviceParameter, error) {
    query, args, err := psql.Select(paramColumns...).
        From("device_parameters").
        Where(sq.Eq{"device_id": deviceID, "fap_instance": instance, "param_group": group}).
        OrderBy("parameter_path ASC").
        ToSql()
    // ... 标准 rows 扫描逻辑
}
```

### Phase 5: 修改 Service 层

**修改文件**：`internal/device/service.go`

`GetDeviceDetailComposite` 方法（L656-711）改为按分组查询：

```go
func (s *DeviceService) GetDeviceDetailComposite(ctx context.Context, deviceID uuid.UUID) (*DeviceDetailComposite, error) {
    // ... 获取 device 和 device_info（不变）

    // MME pool — 旧：filterByPrefix(allParams, "MmePoolConfigParam.")
    mmeParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "mme_pool")
    if err != nil {
        s.logger.Warn("get mme_pool params", zap.Error(err))
    }
    result.MMEPool = AssembleMMEPool(mmeParams)

    // License — 旧：filterByPrefix(allParams, "X_COM_LICENSE.")
    licenseParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "license")
    if err != nil {
        s.logger.Warn("get license params", zap.Error(err))
    }
    result.License = AssembleLicenseDetail(licenseParams)

    // Antenna — 旧：filterByPrefix(allParams, "AntennaInfo.")
    antennaParams, err := s.paramRepo.GetByGroup(ctx, deviceID, "antenna")
    if err != nil {
        s.logger.Warn("get antenna params", zap.Error(err))
    }
    result.Antenna = AssembleAntennaInfo(antennaParams)

    // Cells — 需要 fap_control 组按小区分组
    numOfCells := 1
    if result.Info != nil && result.Info.NumOfCells > 0 {
        numOfCells = result.Info.NumOfCells
    }
    // 仍使用全量查询 + 内存过滤（Cells 需要跨多个 group）
    allParams, err := s.paramRepo.GetByDevice(ctx, deviceID)
    if err != nil {
        s.logger.Warn("get all params for cells", zap.Error(err))
        return result, nil
    }
    result.Cells = AssembleCells(allParams, numOfCells)

    return result, nil
}
```

**删除 `filterByPrefix` 函数**（L713-722）— 不再使用。

### Phase 6: 更新 Mock 实现

以下 3 个测试文件中的 mock 结构体需要新增 3 个方法的空实现：

| 文件 | mock 结构体名 |
|------|-------------|
| `handler_test.go` | `fakeParamRepo` |
| `service_test.go` | `mockParamRepo` |
| `inform_handler_test.go` | `infMockParamRepo` |

每个 mock 新增：

```go
func (m *xxxParamRepo) GetByGroup(_ context.Context, _ uuid.UUID, _ string) ([]model.DeviceParameter, error) {
    return []model.DeviceParameter{}, nil
}

func (m *xxxParamRepo) GetByFAPInstance(_ context.Context, _ uuid.UUID, _ int) ([]model.DeviceParameter, error) {
    return []model.DeviceParameter{}, nil
}

func (m *xxxParamRepo) GetByFAPInstanceAndGroup(_ context.Context, _ uuid.UUID, _ int, _ string) ([]model.DeviceParameter, error) {
    return []model.DeviceParameter{}, nil
}
```

### Phase 7: 数据库迁移

**新建文件**：`migrations/000064_partition_device_parameters.up.sql`

```sql
-- ============================================================
-- Step 1: 重命名旧表
-- ============================================================
ALTER TABLE device_parameters RENAME TO device_parameters_old;

-- ============================================================
-- Step 2: 创建分区表（新 Schema）
-- ============================================================
CREATE TABLE device_parameters (
    device_id        UUID         NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN      DEFAULT false,
    last_updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    fap_instance     SMALLINT     NOT NULL DEFAULT 0,
    param_group      VARCHAR(32)  NOT NULL DEFAULT 'other',
    PRIMARY KEY (device_id, parameter_path)
) PARTITION BY HASH (device_id);

-- ============================================================
-- Step 3: 创建 32 个分区
-- ============================================================
DO $$
BEGIN
    FOR i IN 0..31 LOOP
        EXECUTE format(
            'CREATE TABLE device_parameters_p%s PARTITION OF device_parameters
             FOR VALUES WITH (MODULUS 32, REMAINDER %s)',
            lpad(i::text, 2, '0'), i
        );
    END LOOP;
END $$;

-- ============================================================
-- Step 4: 数据迁移 + 自动分类
-- ============================================================
INSERT INTO device_parameters (
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at,
    fap_instance, param_group
)
SELECT
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at,
    COALESCE(
        (regexp_match(parameter_path, 'FAPService\.(\d+)\.'))[1]::SMALLINT,
        0
    ),
    CASE
        WHEN parameter_path LIKE '%MmePoolConfigParam.%'
          OR parameter_path LIKE '%X_COM_MmePool.%'          THEN 'mme_pool'
        WHEN parameter_path LIKE '%X_COM_LICENSE.%'          THEN 'license'
        WHEN parameter_path LIKE '%AntennaInfo.%'            THEN 'antenna'
        WHEN parameter_path LIKE '%FaultMgmt.CurrentAlarm.%' THEN 'alarm'
        WHEN parameter_path LIKE '%X_COM_GPS%'
          OR parameter_path LIKE '%X_COM_BDS%'
          OR parameter_path LIKE '%X_COM_1588%'
          OR parameter_path LIKE '%X_COM_GLONASS%'
          OR parameter_path LIKE 'Device.FAP.GPS.%'
          OR parameter_path LIKE '%tfcsSync%'
          OR parameter_path LIKE '%tfcsManager%'             THEN 'sync'
        WHEN parameter_path LIKE '%CellConfig%'
         AND (parameter_path LIKE '%RAN.%'
           OR parameter_path LIKE '%EPC.PLMN%')              THEN 'radio'
        WHEN parameter_path LIKE '%FAPControl.%'             THEN 'fap_control'
        WHEN parameter_path LIKE 'Device.DeviceInfo.%'       THEN 'device_info'
        WHEN parameter_path LIKE 'Device.ManagementServer.%' THEN 'management'
        WHEN parameter_path LIKE '%Ipsec%'
          OR parameter_path LIKE '%IPSEC%'                   THEN 'ipsec'
        WHEN parameter_path LIKE 'Device.IP.%'               THEN 'network'
        WHEN parameter_path LIKE 'Device.SoftwareCtrl.%'     THEN 'software'
        ELSE 'other'
    END
FROM device_parameters_old;

-- ============================================================
-- Step 5: 清理旧表
-- ============================================================
DROP TABLE device_parameters_old;

-- ============================================================
-- Step 6: 更新统计信息
-- ============================================================
ANALYZE device_parameters;
```

**新建文件**：`migrations/000064_partition_device_parameters.down.sql`

```sql
ALTER TABLE device_parameters RENAME TO device_parameters_partitioned;

CREATE TABLE device_parameters (
    device_id        UUID NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN DEFAULT false,
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, parameter_path)
);

CREATE INDEX idx_device_params_device ON device_parameters (device_id);
CREATE INDEX idx_device_params_path_prefix
ON device_parameters (device_id, parameter_path varchar_pattern_ops);

INSERT INTO device_parameters (
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at
)
SELECT device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at
FROM device_parameters_partitioned;

DROP TABLE device_parameters_partitioned;
ANALYZE device_parameters;
```

### Phase 8: 编译验证 + 测试

```bash
cd omcgo
go build ./...           # 编译检查
go test ./internal/device/...  # 单元测试
```

---

## 5. 迁移注意事项

| 事项 | 说明 |
|------|------|
| **停机窗口** | 数据迁移需要排他锁（RENAME + INSERT ... SELECT + DROP），建议维护窗口执行 |
| **磁盘空间** | 迁移期间新旧表共存，需要 2× 空间 |
| **耗时估算** | 5000 万行 INSERT 约 5-10 分钟，5 亿行约 1-2 小时 |
| **应用兼容** | 分区表对 SQL 完全透明，无需修改查询语句 |
| **回滚方式** | `migrate down` 执行 000064.down.sql，恢复原始单表 + 索引 |

---

## 6. 验证清单

### 6.1 编译

- [ ] `go build ./...` 通过
- [ ] `go vet ./...` 通过

### 6.2 单元测试

- [ ] `param_classify_test.go` — 12 个分组 + fap_instance 提取覆盖全部边界
- [ ] `service_test.go` — `GetDeviceDetailComposite` 使用 GetByGroup mock
- [ ] 所有现有测试继续通过（mock 接口兼容）

### 6.3 迁移测试

- [ ] `migrate up` → `migrate down` → `migrate up` 幂等执行
- [ ] 迁移后数据行数一致（`SELECT COUNT(*) FROM device_parameters` = 迁移前）
- [ ] 分类结果正确（随机抽样 100 行验证 fap_instance + param_group）

### 6.4 功能测试

- [ ] 设备详情页 DTO 组装正确（MME Pool / License / Antenna / Cells）
- [ ] 参数树浏览功能正常（GetByDevice / GetByPathPrefix / GetDirectChildLeaves / SearchByKeyword）
- [ ] Inform 写入后分类列正确填充
- [ ] InfoSync 从参数同步到 device_info 正常

### 6.5 性能验证

- [ ] EXPLAIN ANALYZE 确认查询走 PK 索引（partition pruning）
- [ ] BatchUpsert 500 行 < 10ms（单设备）
- [ ] GetByGroup 延迟 < 2ms

---

## 7. 文件变更清单

| 操作 | 文件 | 改动量 |
|------|------|--------|
| 新建 | `migrations/000064_partition_device_parameters.up.sql` | ~60 行 |
| 新建 | `migrations/000064_partition_device_parameters.down.sql` | ~25 行 |
| 新建 | `internal/device/param_classify.go` | ~70 行 |
| 新建 | `internal/device/param_classify_test.go` | ~120 行 |
| 修改 | `internal/core/model/parameter.go` | +2 行（结构体新增字段） |
| 修改 | `internal/device/param_repository.go` | +3 行（接口新增方法） |
| 修改 | `internal/device/pg_param_repository.go` | ~+100 行（新方法 + 修改列名 + 修改 Scan） |
| 修改 | `internal/device/service.go` | ~20 行（替换 filterByPrefix） |
| 修改 | `internal/device/handler_test.go` | +12 行（mock 新方法） |
| 修改 | `internal/device/service_test.go` | +12 行（mock 新方法） |
| 修改 | `internal/device/inform_handler_test.go` | +12 行（mock 新方法） |

**总计**：4 个新文件 + 7 个修改文件，~430 行变更。

---

## 8. 关联文档

| 文档 | 说明 |
|------|------|
| `docs/demo/device-parameters-storage-demo.md` | 容量分析与方案推导过程 |
| `docs/design/0023-device-list-management-gap-analysis.md` | device_info 双表架构设计 |
| `files/Back-end/TR069报文全解析.md` | 252 条 TR069 参数路径原始数据 |
