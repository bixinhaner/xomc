# device_parameters 存储方案

> 500 参数/设备 × 10 万～100 万设备的容量分析与表结构设计。
> 消除 LIKE 查询，Hash 分区应对大规模写入。

---

## 1. TR069 参数路径结构

### 1.1 路径层级

```
Device
├── DeviceInfo                          ← 设备基础信息（GPS、天线、SAS、硬件）
│   ├── HardwareVersion / SoftwareVersion / FAP_adminstate
│   ├── X_COM_GPS_Status / X_COM_BDS_Status / X_COM_1588_Status  ← 同步
│   ├── AntennaInfo.{Height,Azimuth,...}                          ← 天线
│   └── X_COM_LICENSE.{...}                                       ← License
├── ManagementServer
│   ├── ConnectionRequestURL
│   └── tfcsSyncState / tfcsManagerPrimsrc                        ← 同步
├── Services.FAPService
│   ├── {1,2,3}                         ← 小区实例（索引从 1 开始）
│   │   ├── FAPControl.LTE.{OpState,CellOpState,RFTxStatus,...}
│   │   ├── FAPControl.LTE.Gateway.X_COM_MmePool.{...}
│   │   ├── FAPControl.LTE.X_COM_LICENSE.Capacity.{1-32}.{...}
│   │   ├── CellConfig.LTE.RAN.RF.{PhyCellID,EARFCNDL,...}
│   │   ├── CellConfig.LTE.MmePoolConfigParam.{1-16}.{...}
│   │   └── CellConfig.LTE.EPC.PLMNList.{...}
│   └── Ipsec.{IPSEC_TUNNEL1_STATUS,...}                          ← 无实例编号
├── FAP.GPS.{LockedLatitude,LockedLongitude,...}
├── FaultMgmt.CurrentAlarm.{1-N}.{...}
├── IP.Interface.{1}.IPv4Address.{1}.IPAddress
└── SoftwareCtrl.SystemBackupVersion
```

### 1.2 多实例索引规则

TR-069 Amendment 6（BBF 标准）规定多实例对象索引从 **1** 开始：

| 模式 | 含义 | FAPService 实例 |
|------|------|----------------|
| SC (Single Carrier) | 单载波 | `FAPService.1` |
| CA (Carrier Aggregation) | 载波聚合 | `FAPService.1`, `FAPService.2` |
| TC (Triple Carrier) | 三载波 | `FAPService.1`, `FAPService.2`, `FAPService.3` |

---

## 2. 容量分析

### 2.1 行数规模

| 设备规模 | 参数/设备 | 总行数 | 级别 |
|---------|----------|--------|------|
| 10 万 | 500 | **5,000 万** | 中等 |
| 100 万 | 500 | **5 亿** | 大型 |

### 2.2 单行存储估算

```
PostgreSQL 行结构（HeapTupleHeader + 数据 + 对齐）：

字段                    字节数
──────────────────────────────
Tuple Header            23
NULL bitmap              2
device_id (UUID)        16
parameter_path (avg 80) 84    ← 4 字节 varlena header + 80 字节内容
parameter_value (avg 15) 19
parameter_type (avg 8)  12
writable (BOOL)          1
last_updated_at (TSTZ)   8
fap_instance (SMALLINT)  2
param_group (avg 10)    14
对齐填充                  ~3
──────────────────────────────
合计                    ~184 字节/行
```

### 2.3 存储总量（含索引）

| | 10 万设备 × 500 | 100 万设备 × 500 |
|-|-----------------|------------------|
| 表数据 | 5000 万 × 184B = **8.6 GB** | 5 亿 × 184B = **86 GB** |
| PK 索引 (device_id, parameter_path) | ~3.5 GB | ~35 GB |
| **总计（仅 PK）** | **~12 GB** | **~121 GB** |

**对比不加 fap_instance + param_group**：每行减少 ~16 字节 → 节省 800 MB（10万）/ 8 GB（100万），仅占总量 6.5%。新增两列的存储代价极低。

### 2.4 写入压力分析

设备每 5 分钟上报一次 Inform（BatchUpsert）：

| 指标 | 10 万设备 | 100 万设备 |
|------|----------|-----------|
| Inform 频率 | 100,000 / 300s = **333 次/秒** | 3,333 次/秒 |
| 行 UPSERT 频率 | 333 × 500 = **166,500 行/秒** | 1,665,000 行/秒 |
| 每秒写入数据量 | 166,500 × 184B = **29 MB/s** | 290 MB/s |

PostgreSQL `ON CONFLICT DO UPDATE` 每次产生一个新的行版本（MVCC dead tuple），需要 VACUUM 清理。

### 2.5 PostgreSQL 单表极限

| 维度 | 5000 万行 | 5 亿行 | 评估 |
|------|----------|--------|------|
| B-tree 索引深度 | 3-4 层 | 4-5 层 | 查询仍 <2ms |
| Index Scan 性能 | 优秀 | 良好 | 多 1 次 I/O |
| VACUUM 全表 | ~10 分钟 | **~2 小时** | 100 万时有风险 |
| 索引膨胀 | 可控 | 需要监控 | 定期 REINDEX |
| autovacuum 跟得上？ | 跟得上 | **可能跟不上** | 16 万行/秒 UPDATE 很激进 |
| HOT Update 可行？ | 是 | 是 | 更新列不在索引中 → HOT |

### 2.6 结论

| 规模 | 单表可行？ | 推荐方案 |
|------|-----------|---------|
| **10 万 × 500 = 5000 万行** | **可行** | Hash 分区（改善 VACUUM） |
| **100 万 × 500 = 5 亿行** | **需要分区** | Hash 分区（必须） |

**核心瓶颈不是查询性能（索引都走得通），而是 VACUUM 和写入放大。**

---

## 3. 推荐方案：K-V + Hash 分区

### 3.1 设计原则

1. **Hash 分区 by device_id** — 每个分区独立 VACUUM，互不阻塞
2. **fap_instance + param_group 列** — 消除 LIKE，等值过滤
3. **仅保留 PK 索引** — 减少写入放大（所有查询都是 device-scoped）
4. **不加额外二级索引** — 500 行内存过滤代价忽略不计

### 3.2 为什么不需要二级索引

所有 device_parameters 查询都先过滤 `device_id`：

```
PK Index Scan: WHERE device_id = $1
  → 返回该设备全部 ~500 行（<1ms）
  → 再按 param_group / fap_instance 内存过滤 ~50 行

额外二级索引的收益：从 500 行直接跳到 50 行 → 节省 ~0.1ms
额外二级索引的代价：每次 BatchUpsert 500 行 × 每个索引 1 次写入
  → 10 万设备 = 166,500 次/秒 × 索引数 = 大量写放大
```

**结论：二级索引在 device-scoped 查询中收益极低，但写入代价巨大。不加。**

### 3.3 Schema

```sql
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
```

### 3.4 分区数选择

| 分区数 | 10万（行/分区） | 100万（行/分区） | 评估 |
|--------|---------------|-----------------|------|
| 16 | 312 万 | 3,125 万 | 100 万时偏大 |
| **32** | **156 万** | **1,562 万** | **两个阶段都舒适** |
| 64 | 78 万 | 781 万 | planner 开销增加 |
| 128 | 39 万 | 390 万 | 分区过多，管理复杂 |

**推荐 32 分区**：每分区 156 万行（10万）→ 1562 万行（100万），VACUUM 单次 <5 分钟。

### 3.5 分区容量明细

| 指标 | 每分区（10万） | 每分区（100万） |
|------|-------------|---------------|
| 行数 | 156 万 | 1,562 万 |
| 表数据 | ~274 MB | ~2.7 GB |
| PK 索引 | ~110 MB | ~1.1 GB |
| VACUUM 耗时 | <1 分钟 | <5 分钟 |
| autovacuum 可跟上 | 轻松 | 轻松 |

### 3.6 为什么 Hash 分区适合

| 特性 | 说明 |
|------|------|
| **UUID 天然均匀** | device_id 是 UUID，hash 后分布极均匀 |
| **查询自动路由** | `WHERE device_id = $1` 自动 partition pruning，只命中 1 个分区 |
| **透明** | 应用层代码零修改，PG 自动处理 |
| **VACUUM 并行** | 32 个分区各自 autovacuum，不会锁住整张大表 |
| **HOT Update** | 每个分区独立 HOT 优化，dead tuple 比例更低 |
| **在线维护** | 可对单个分区 REINDEX / CLUSTER 不影响其他分区 |

---

## 4. 当前问题与解决

### 4.1 现有 Schema

```sql
CREATE TABLE device_parameters (
    device_id        UUID NOT NULL,
    parameter_path   VARCHAR(512) NOT NULL,
    parameter_value  TEXT,
    parameter_type   VARCHAR(20),
    writable         BOOLEAN DEFAULT false,
    last_updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id, parameter_path)
);

-- 现有索引
CREATE INDEX idx_device_params_device ON device_parameters (device_id);
CREATE INDEX idx_device_params_path_prefix ON device_parameters
    (device_id, parameter_path varchar_pattern_ops);
```

### 4.2 LIKE 查询问题

```sql
-- 中缀匹配无法走索引，必然全表扫描
WHERE device_id = $1 AND parameter_path LIKE '%MmePoolConfigParam.%'
WHERE device_id = $1 AND parameter_path LIKE '%X_COM_LICENSE.%'

-- 多路径 OR 同样无法优化
WHERE device_id = $1 AND (
    parameter_path LIKE '%X_COM_GPS%'
    OR parameter_path LIKE '%X_COM_BDS%'
    OR parameter_path LIKE '%tfcsSync%'
)
```

### 4.3 解决：fap_instance + param_group

在 `BatchUpsert` 写入时从 `parameter_path` 自动提取两个维度：

| 列 | 含义 | 提取规则 |
|----|------|---------|
| `fap_instance` | FAPService 实例编号 | `FAPService.{N}` → N，非 FAPService → 0 |
| `param_group` | 功能分组 | 按路径关键字匹配 12 个预定义分组 |

**所有 LIKE 查询变等值匹配**：

```sql
-- 旧：LIKE '%MmePoolConfigParam.%'（全扫）
-- 新：WHERE device_id = $1 AND param_group = 'mme_pool'（PK 扫 500 行 + 内存过滤）

-- 旧：5 个 OR LIKE（无法优化）
-- 新：WHERE device_id = $1 AND param_group = 'sync'（一次等值）
```

### 4.4 现有多余索引清理

分区后以下索引**可以删除**：

| 索引 | 原因 |
|------|------|
| `idx_device_params_device` | PK 首列已是 device_id，完全冗余 |
| `idx_device_params_path_prefix` | LIKE 查询已被 param_group 等值取代 |

**省下两个索引 = 减少 ~30% 写入放大。**

---

## 5. param_group 分类规则

### 5.1 分组定义

12 个分组，按匹配优先级从高到低排列：

| param_group | 匹配规则 | 典型参数数 | 典型查询场景 |
|---|---|---|---|
| `mme_pool` | 含 `MmePoolConfigParam.` 或 `X_COM_MmePool.` | ~51 | MME 池详情 |
| `license` | 含 `X_COM_LICENSE.` | ~42 | License 状态 |
| `antenna` | 含 `AntennaInfo.` | ~7 | 天线参数 |
| `alarm` | 含 `FaultMgmt.CurrentAlarm.` | ~30 | 当前告警 |
| `sync` | 含 `X_COM_GPS`/`X_COM_BDS`/`X_COM_1588`/`X_COM_GLONASS` 或 `Device.FAP.GPS.` 或 `tfcsSync`/`tfcsManager` | ~12 | 同步状态 |
| `radio` | 含 `CellConfig` 且含 `RAN.`/`EPC.PLMN` | ~10 | 射频参数 |
| `fap_control` | 含 `FAPControl.`（前面未命中） | ~15 | 小区状态 |
| `device_info` | 前缀 `Device.DeviceInfo.`（前面未命中） | ~30 | 设备信息 |
| `management` | 前缀 `Device.ManagementServer.`（前面未命中） | ~5 | 管理配置 |
| `ipsec` | 含 `Ipsec` 或 `IPSEC` | ~5 | IPSec |
| `network` | 前缀 `Device.IP.` | ~2 | IP 地址 |
| `software` | 前缀 `Device.SoftwareCtrl.` | ~1 | 备份版本 |
| `other` | 以上都不匹配 | ~2 | 兜底 |

### 5.2 fap_instance 提取规则

```
Device.Services.FAPService.1.FAPControl.LTE.OpState         → fap_instance = 1
Device.Services.FAPService.2.FAPControl.LTE.CellOpState     → fap_instance = 2
Device.Services.FAPService.Ipsec.IPSEC_TUNNEL1_STATUS       → fap_instance = 0（非数字）
Device.DeviceInfo.X_COM_GPS_Status                          → fap_instance = 0（非 FAPService）
```

### 5.3 分类示例

| parameter_path | fap | group |
|---|---|---|
| `Device.DeviceInfo.HardwareVersion` | 0 | device_info |
| `Device.DeviceInfo.X_COM_GPS_Status` | 0 | sync |
| `Device.DeviceInfo.AntennaInfo.Height` | 0 | antenna |
| `Device.ManagementServer.tfcsSyncState` | 0 | sync |
| `Device.FAP.GPS.LockedLatitude` | 0 | sync |
| `Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity` | 0 | alarm |
| `Device.Services.FAPService.1.FAPControl.LTE.OpState` | 1 | fap_control |
| `Device.Services.FAPService.1.FAPControl.LTE.X_COM_LICENSE.Code` | 1 | license |
| `Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID` | 1 | radio |
| `Device.Services.FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status` | 1 | mme_pool |
| `Device.Services.FAPService.2.FAPControl.LTE.CellOpState` | 2 | fap_control |
| `Device.Services.FAPService.Ipsec.IPSEC_TUNNEL1_STATUS` | 0 | ipsec |

---

## 6. 查询模式

### 6.1 所有查询都是 device-scoped

```sql
-- PK Index Scan → 一个分区 → 该设备全部 ~500 行 → 内存过滤

-- 按分组（取代 LIKE '%MmePoolConfigParam.%'）
WHERE device_id = $1 AND param_group = 'mme_pool'

-- 按小区
WHERE device_id = $1 AND fap_instance = 1

-- 组合
WHERE device_id = $1 AND fap_instance = 1 AND param_group = 'license'

-- 同步参数（取代 5 个 OR LIKE）
WHERE device_id = $1 AND param_group = 'sync'
```

### 6.2 查询性能（分区后）

| 查询 | 路径 | 延迟 |
|------|------|------|
| 单设备全量 (500 行) | partition prune → PK scan | <1ms |
| 单设备某分组 (~50 行) | PK scan 500 行 + 内存过滤 | <1ms |
| 单设备单参数 | PK 精确命中 | <0.5ms |
| BatchUpsert 500 行 | 一个分区内 500 次 UPSERT | ~5ms |

---

## 7. Go 代码 Demo

### 7.1 路径分类函数

```go
// internal/device/param_classify.go

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
        return 0 // FAPService.Ipsec 等非数字实例
    }
    return n
}

// ClassifyParamGroup 根据 TR069 路径返回功能分组标签。
// 匹配规则按优先级从高到低排列。
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

### 7.2 BatchUpsert（自动分类）

```go
func (r *PgDeviceParameterRepository) BatchUpsert(
    ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter,
) error {
    batch := &pgx.Batch{}
    now := time.Now()

    for _, p := range params {
        query, args, err := psql.Insert("device_parameters").
            Columns("device_id", "parameter_path", "parameter_value",
                    "parameter_type", "writable", "last_updated_at",
                    "fap_instance", "param_group").
            Values(deviceID, p.ParameterPath, p.ParameterValue,
                   p.ParameterType, p.Writable, now,
                   ExtractFAPInstance(p.ParameterPath),
                   ClassifyParamGroup(p.ParameterPath)).
            Suffix(`ON CONFLICT (device_id, parameter_path) DO UPDATE SET
                    parameter_value = EXCLUDED.parameter_value,
                    parameter_type = EXCLUDED.parameter_type,
                    writable = EXCLUDED.writable,
                    last_updated_at = EXCLUDED.last_updated_at`).
            ToSql()
        if err != nil {
            return fmt.Errorf("build upsert query: %w", err)
        }
        batch.Queue(query, args...)
    }

    br := r.pool.SendBatch(ctx, batch)
    defer br.Close()
    for i := 0; i < len(params); i++ {
        if _, err := br.Exec(); err != nil {
            return fmt.Errorf("exec batch upsert item %d: %w", i, err)
        }
    }
    return nil
}
```

`ON CONFLICT` 不更新 `fap_instance` 和 `param_group`——它们由 `parameter_path` 唯一确定，不会变化。

### 7.3 Repository 接口

```go
type DeviceParameterRepository interface {
    // 写入
    BatchUpsert(ctx context.Context, deviceID uuid.UUID, params []model.DeviceParameter) error
    DeleteByDevice(ctx context.Context, deviceID uuid.UUID) error

    // 读取 — 全量
    GetByDevice(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error)
    GetByPath(ctx context.Context, deviceID uuid.UUID, path string) (*model.DeviceParameter, error)

    // 读取 — 按分组（取代 LIKE）
    GetByGroup(ctx context.Context, deviceID uuid.UUID, group string) ([]model.DeviceParameter, error)
    GetByFAPInstance(ctx context.Context, deviceID uuid.UUID, instance int) ([]model.DeviceParameter, error)
    GetByFAPInstanceAndGroup(ctx context.Context, deviceID uuid.UUID, instance int, group string) ([]model.DeviceParameter, error)

    // 参数树（保留，给前端树形展示用）
    GetByPathPrefix(ctx context.Context, deviceID uuid.UUID, prefix string) ([]model.DeviceParameter, error)
    SearchByKeyword(ctx context.Context, deviceID uuid.UUID, keyword string, limit int) ([]model.DeviceParameter, error)
}
```

### 7.4 读取示例

```go
// 详情页 DTO 组装 — 按需查库，不再加载全量后内存过滤

mmeParams, _ := paramRepo.GetByGroup(ctx, deviceID, "mme_pool")
mmePool := AssembleMMEPool(mmeParams)

licenseParams, _ := paramRepo.GetByGroup(ctx, deviceID, "license")
license := AssembleLicenseDetail(licenseParams)

syncParams, _ := paramRepo.GetByGroup(ctx, deviceID, "sync")
syncStatus := CalcSyncStatus(toMap(syncParams))

cell1Params, _ := paramRepo.GetByFAPInstanceAndGroup(ctx, deviceID, 1, "fap_control")
cellStatus := CalcCellStatus(toMap(cell1Params))
```

---

## 8. 迁移 SQL

### 8.1 Up — 分区重建 + 数据迁移

```sql
-- migrations/000064_partition_device_parameters.up.sql

-- ============================================================
-- Step 1: 重命名旧表
-- ============================================================
ALTER TABLE device_parameters RENAME TO device_parameters_old;

-- ============================================================
-- Step 2: 创建分区表
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
    -- fap_instance: 从路径提取 FAPService.{N}
    COALESCE(
        (regexp_match(parameter_path, 'FAPService\.(\d+)\.'))[1]::SMALLINT,
        0
    ),
    -- param_group: 按关键字分类（优先级从高到低）
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
-- Step 5: 清理旧表和冗余索引
-- ============================================================
DROP TABLE device_parameters_old;

-- ============================================================
-- Step 6: 更新统计信息
-- ============================================================
ANALYZE device_parameters;
```

### 8.2 Down — 回退到单表

```sql
-- migrations/000064_partition_device_parameters.down.sql

-- 重命名分区表
ALTER TABLE device_parameters RENAME TO device_parameters_partitioned;

-- 创建原始单表
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

-- 回填数据（丢弃 fap_instance 和 param_group）
INSERT INTO device_parameters (
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at
)
SELECT
    device_id, parameter_path, parameter_value,
    parameter_type, writable, last_updated_at
FROM device_parameters_partitioned;

-- 重建 varchar_pattern_ops 索引（原 000061 迁移）
CREATE INDEX idx_device_params_path_prefix
ON device_parameters (device_id, parameter_path varchar_pattern_ops);

-- 清理
DROP TABLE device_parameters_partitioned;

ANALYZE device_parameters;
```

### 8.3 迁移注意事项

| 事项 | 说明 |
|------|------|
| **停机窗口** | 数据迁移需要排他锁，建议在维护窗口执行 |
| **大表迁移耗时** | 5000 万行 INSERT 约 5-10 分钟，5 亿行约 1-2 小时 |
| **磁盘空间** | 迁移期间新旧表共存，需要 2× 空间 |
| **应用兼容** | 分区表对应用完全透明，Go 代码无需修改（仅新增 fap_instance/param_group 写入） |

---

## 9. 存储示例

双载波基站的 device_parameters 数据分布在某个分区中：

```
device_id  | parameter_path                                          | value    | fap | group
-----------|---------------------------------------------------------|----------|-----|------------
d1234567.. | Device.DeviceInfo.HardwareVersion                       | A01      |  0  | device_info
d1234567.. | Device.DeviceInfo.X_COM_GPS_Status                      | 1        |  0  | sync
d1234567.. | Device.DeviceInfo.AntennaInfo.Height                    | 30       |  0  | antenna
d1234567.. | Device.ManagementServer.tfcsSyncState                   | DISP     |  0  | sync
d1234567.. | Device.FAP.GPS.LockedLatitude                           | 39.9042  |  0  | sync
d1234567.. | Device.FaultMgmt.CurrentAlarm.1.PerceivedSeverity       | Critical |  0  | alarm
d1234567.. | Device.Services.FAPService.1.FAPControl.LTE.OpState     | true     |  1  | fap_control
d1234567.. | Device.Services.FAPService.1.FAPControl.LTE.CellOpState | 1        |  1  | fap_control
d1234567.. | ...FAPService.1.FAPControl.LTE.X_COM_LICENSE.Capacity.1.State | 1   |  1  | license
d1234567.. | ...FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID         | 100      |  1  | radio
d1234567.. | ...FAPService.1.CellConfig.LTE.MmePoolConfigParam.1.MME1Status | 1  |  1  | mme_pool
d1234567.. | Device.Services.FAPService.2.FAPControl.LTE.CellOpState | 0        |  2  | fap_control
d1234567.. | Device.Services.FAPService.Ipsec.IPSEC_TUNNEL1_STATUS   | false    |  0  | ipsec
```

---

## 10. 数据流总结

```
CPE Inform 报文
     │
     ▼
BatchUpsert（自动调用 ExtractFAPInstance + ClassifyParamGroup）
     │
     ▼
device_parameters（Hash 分区表，32 分区，仅 PK 索引）
     │
     ├──→ GetByGroup("sync")     ──→ CalcSyncStatus / CalcGPSStatus
     ├──→ GetByGroup("mme_pool") ──→ CalcMMEStatus / AssembleMMEPool
     ├──→ GetByGroup("license")  ──→ CalcLicenseStatus / AssembleLicenseDetail
     ├──→ GetByGroup("antenna")  ──→ AssembleAntennaInfo
     ├──→ GetByFAPInstance(1)    ──→ CalcCellStatus (主小区)
     ├──→ GetByFAPInstance(2)    ──→ CalcCellStatus (小区 2, CA 模式)
     │         │
     │         ▼
     │   device_info（29 个快查列 — 列表页用）
     │
     └──→ GetByFAPInstanceAndGroup(1, "radio") ──→ 射频参数展示
```

### 三层存储架构

| 层级 | 存储 | 查询方式 | 规模 |
|------|------|---------|------|
| Tier 1 | device_info（29 列） | LEFT JOIN + WHERE 等值 | 10 万行 |
| Tier 2 | device_parameters（500 行/设备 K-V） | PK scan + 内存过滤 | 5000 万～5 亿行（32 分区） |
| Tier 3 | 独立业务表（alarm, pm 等） | 各自模块查询 | 按业务独立增长 |
