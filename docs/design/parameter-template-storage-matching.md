# 参数模版存储与匹配方案

> 版本：v1.1 | 日期：2026-03-23 | 状态：Phase 1-5 已实现，Phase 6 部分完成

---

## 1. 背景与目标

当前 `data_model_definitions` 表已支持 OUI、ProductClass、FirmwareVersion 字段和三级/四级回退解析。但存在以下不足：

1. **自动发现模版与手动导入模版**无明确的生命周期管理策略
2. **模版匹配精度**需进一步明确：先精确匹配（含版本），再宽泛匹配（不含版本）
3. **模版过期清理**机制缺失，长期运行后自动发现模版会无限累积

### 目标

| 编号 | 目标 | 说明 |
|------|------|------|
| G1 | 精确匹配 | OUI + ProductClass + FirmwareVersion 优先，OUI + ProductClass 兜底 |
| G2 | 模版分类 | 自动模版（auto）vs 手动模版（manual），各有不同存储和过期策略 |
| G3 | 过期清理 | 自动模版 15 天无访问过期，手动模版 60 天无访问过期 |
| G4 | 访问追踪 | 每次匹配命中时更新 `last_accessed_at`，作为过期判断依据 |

---

## 2. 模版类型定义

### 2.1 自动模版（auto_discovered）

- **来源**：设备首次 Inform 触发自动发现（GetParameterNames 遍历参数树）
- **存储键**：`OUI + ProductClass`（**不含 FirmwareVersion**）
- **原因**：同一 OUI + ProductClass 的设备，不同固件版本的参数树差异通常很小；自动发现的目的是获取设备参数树骨架，不需要按版本区分
- **过期策略**：15 天无访问自动删除
- **创建时机**：`DiscoveryService.HandleDiscoveryResult()` 中创建

### 2.2 手动模版（manual）

- **来源**：Web 页面导入（上传 CSV/XML 文件或手工编辑）
- **存储键**：`OUI + ProductClass`（基础），可选附加 `FirmwareVersion`
- **原因**：手动导入时用户可以选择是否限定固件版本。不限定版本时，模版适用于该 OUI + ProductClass 的所有固件版本
- **过期策略**：60 天无访问自动删除
- **创建时机**：`DataModelHandler` 的导入 API

---

## 3. 匹配优先级

### 3.1 二级精确匹配

参数模版匹配采用精确二级策略，不做 OUI-only 或 carrier_default 兜底：

```
优先级 1（精确匹配）: OUI + ProductClass + FirmwareVersion
    ↓ 未命中
优先级 2（产品级）  : OUI + ProductClass（firmware_version IS NULL 或为空）
    ↓ 未命中
    → 无匹配模版（设备需等待自动发现或手动导入）
```

### 3.2 匹配规则细化

| 优先级 | Scope | OUI | ProductClass | FirmwareVersion | 说明 |
|--------|-------|-----|-------------|-----------------|------|
| 1 | product | 匹配 | 匹配 | 匹配（非空） | 手动模版限定了版本 |
| 2 | product | 匹配 | 匹配 | NULL/空 | 自动模版 或 手动模版未限定版本 |

**关键规则**：
- 两个优先级都属于 `scope = 'product'`，通过 `firmware_version` 是否为空区分
- 自动模版始终落在优先级 2（因为 `firmware_version` 为空）
- 手动模版可以落在优先级 1（限定版本）或优先级 2（不限定版本）
- 同一 OUI + ProductClass 可以同时存在一个自动模版和一个手动模版（通过 `source_type` 区分）
- **不做 OUI-only 或 carrier_default 兜底**——模版匹配必须精确到 OUI + ProductClass

### 3.3 同级冲突处理

当优先级 2 存在**同时命中**的自动模版和手动模版时：

```
手动模版（manual）优先于自动模版（auto_discovered）
```

理由：手动模版是用户明确导入的，代表用户意图，应该覆盖自动发现的结果。

---

## 4. 数据库变更

### 4.1 新增字段：`last_accessed_at`

```sql
-- Migration: 000058_datamodel_add_last_accessed_at.up.sql
ALTER TABLE data_model_definitions
    ADD COLUMN last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 初始化：将现有记录的 last_accessed_at 设为 updated_at
UPDATE data_model_definitions SET last_accessed_at = updated_at;

-- 索引：支持过期清理查询
CREATE INDEX idx_dm_expiry_check
    ON data_model_definitions (source_type, last_accessed_at)
    WHERE is_active = true;

COMMENT ON COLUMN data_model_definitions.last_accessed_at IS
    '最后一次被匹配命中的时间，用于过期清理判断';
```

```sql
-- Migration: 000058_datamodel_add_last_accessed_at.down.sql
DROP INDEX IF EXISTS idx_dm_expiry_check;
ALTER TABLE data_model_definitions DROP COLUMN IF EXISTS last_accessed_at;
```

### 4.2 调整唯一约束

当前唯一约束 `idx_dm_active_product` 是 `(carrier, technology, oui, product_class) WHERE is_active AND scope='product'`。

这意味着同一 OUI + ProductClass 只能有一条 active 的 product scope 记录。但我们需要支持：
- 一条 `firmware_version = '21.301.001'` 的手动模版（优先级 1）
- 一条 `firmware_version IS NULL` 的自动模版（优先级 2）

**方案**：修改唯一约束，加入 `firmware_version` 和 `source_type` 维度：

```sql
-- 在同一个迁移文件 000058 中

-- 移除旧的唯一约束
DROP INDEX IF EXISTS idx_dm_active_product;

-- 新约束 1: 带 firmware_version 的手动模版唯一性
-- 同 carrier+tech+oui+product_class+firmware_version 只能有一条 active 手动模版
CREATE UNIQUE INDEX idx_dm_active_product_versioned
    ON data_model_definitions (carrier, technology, oui, product_class, firmware_version)
    WHERE is_active = true
      AND scope = 'product'
      AND firmware_version IS NOT NULL
      AND firmware_version != '';

-- 新约束 2: 不带 firmware_version 的模版唯一性（按 source_type 区分）
-- 同 carrier+tech+oui+product_class+source_type 只能有一条 active 无版本模版
CREATE UNIQUE INDEX idx_dm_active_product_unversioned
    ON data_model_definitions (carrier, technology, oui, product_class, source_type)
    WHERE is_active = true
      AND scope = 'product'
      AND (firmware_version IS NULL OR firmware_version = '');
```

这允许同一 OUI + ProductClass 下共存：
- 最多 1 条 auto_discovered 无版本模版
- 最多 1 条 manual 无版本模版
- 每个 firmware_version 最多 1 条 manual 带版本模版

---

## 5. Go 代码变更

### 5.1 Model 变更

```go
// internal/config/datamodel/model.go

type DataModel struct {
    // ... 现有字段 ...
    LastAccessedAt time.Time           `json:"last_accessed_at"`
}
```

### 5.2 Repository 变更

```go
// internal/config/datamodel/repository.go — 新增接口方法

type DataModelRepository interface {
    // ... 现有方法 ...

    // TouchLastAccessed 更新 last_accessed_at 为当前时间
    TouchLastAccessed(ctx context.Context, id uuid.UUID) error

    // DeleteExpired 删除过期模版
    // autoMaxAge: 自动模版最大未访问天数（15）
    // manualMaxAge: 手动模版最大未访问天数（60）
    // 返回删除的记录数
    DeleteExpired(ctx context.Context, autoMaxAge, manualMaxAge int) (int64, error)

    // FindActiveForMatch 按匹配优先级查找 active 模版
    // 返回最佳匹配（考虑 firmware_version 和 source_type 优先级）
    FindActiveForMatch(ctx context.Context, carrier model.CarrierCode,
        tech model.Technology, oui, productClass, firmwareVersion string,
    ) (*DataModel, error)
}
```

### 5.3 Repository 实现

```go
// internal/config/datamodel/pg_repository.go

func (r *pgRepository) TouchLastAccessed(ctx context.Context, id uuid.UUID) error {
    query := `UPDATE data_model_definitions SET last_accessed_at = NOW() WHERE id = $1`
    _, err := r.pool.Exec(ctx, query, id)
    if err != nil {
        return fmt.Errorf("touch last_accessed_at for %s: %w", id, err)
    }
    return nil
}

func (r *pgRepository) DeleteExpired(ctx context.Context, autoMaxAge, manualMaxAge int) (int64, error) {
    query := `
        DELETE FROM data_model_definitions
        WHERE (
            (source_type = 'auto_discovered' AND last_accessed_at < NOW() - make_interval(days => $1))
            OR
            (source_type = 'manual' AND last_accessed_at < NOW() - make_interval(days => $2))
        )
    `
    tag, err := r.pool.Exec(ctx, query, autoMaxAge, manualMaxAge)
    if err != nil {
        return 0, fmt.Errorf("delete expired data models: %w", err)
    }
    return tag.RowsAffected(), nil
}

func (r *pgRepository) FindActiveForMatch(ctx context.Context,
    carrier model.CarrierCode, tech model.Technology,
    oui, productClass, firmwareVersion string,
) (*DataModel, error) {
    // 单条 SQL 实现二级精确匹配 + source_type 排序
    query := `
        SELECT ... FROM data_model_definitions
        WHERE is_active = true
          AND scope = 'product'
          AND carrier = $1
          AND technology = $2
          AND oui = $3
          AND product_class = $4
          AND (
              -- 优先级 1: firmware_version 精确匹配
              (firmware_version = $5 AND $5 != '')
              OR
              -- 优先级 2: 无版本限定（manual 优先于 auto）
              (firmware_version IS NULL OR firmware_version = '')
          )
        ORDER BY
          CASE
            WHEN firmware_version = $5 AND $5 != '' THEN 1
            ELSE 2
          END,
          CASE source_type WHEN 'manual' THEN 0 ELSE 1 END
        LIMIT 1
    `
    // ... scan row into DataModel ...
}
```

### 5.4 Registry 匹配逻辑变更

```go
// internal/config/datamodel/registry.go

func (r *DataModelRegistry) ResolveWithFirmware(ctx context.Context,
    carrier model.CarrierCode, tech model.Technology,
    oui, productClass, firmwareVersion string,
) (*DataModel, error) {
    key := localCacheKeyWithFirmware(carrier, tech, oui, productClass, firmwareVersion)

    // L1 缓存检查
    if val, ok := r.localCache.Load(key); ok {
        dm := val.(*DataModel)
        // 异步更新 last_accessed_at（不阻塞匹配流程）
        go r.touchLastAccessedAsync(dm.ID)
        return dm, nil
    }

    // L2 + L3 解析（现有逻辑）
    dm, err := r.resolveFromDBWithFirmware(ctx, carrier, tech, oui, productClass, firmwareVersion)
    if err != nil {
        return nil, err
    }
    if dm == nil {
        return nil, nil
    }

    // 写回缓存 + 更新访问时间
    r.localCache.Store(key, dm)
    go r.touchLastAccessedAsync(dm.ID)

    return dm, nil
}

// touchLastAccessedAsync 异步更新 last_accessed_at
// 使用限流避免高频写入：同一模版 10 分钟内最多更新一次
func (r *DataModelRegistry) touchLastAccessedAsync(id uuid.UUID) {
    // 用 sync.Map 做本地去重，key=model_id, value=上次更新时间
    key := id.String()
    if v, ok := r.touchThrottle.Load(key); ok {
        if time.Since(v.(time.Time)) < 60*time.Minute {
            return // 60 分钟内已更新过，跳过
        }
    }

    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    if err := r.repo.TouchLastAccessed(ctx, id); err != nil {
        r.logger.Warn("failed to touch last_accessed_at",
            zap.String("model_id", id.String()),
            zap.Error(err),
        )
        return
    }
    r.touchThrottle.Store(key, time.Now())
}
```

### 5.5 resolveFromDBWithFirmware 简化为二级

```go
// 使用 FindActiveForMatch 单条 SQL 实现二级精确匹配
// 移除原有的 OUI-only 和 carrier_default 回退逻辑
func (r *DataModelRegistry) resolveFromDBWithFirmware(ctx context.Context,
    carrier model.CarrierCode, tech model.Technology,
    oui, productClass, firmwareVersion string,
) (*DataModel, error) {
    return r.repo.FindActiveForMatch(ctx, carrier, tech, oui, productClass, firmwareVersion)
}
```

### 5.6 Discovery 创建自动模版时的约束

```go
// internal/provision/discovery.go — HandleDiscoveryResult

func (s *DiscoveryService) HandleDiscoveryResult(ctx context.Context, ...) error {
    // 创建自动模版时，firmware_version 留空
    dm := &datamodel.DataModel{
        Carrier:         device.Carrier,
        Technology:      device.Technology,
        OUI:             device.OUI,
        ProductClass:    device.ProductClass,
        FirmwareVersion: "",                          // 自动模版不限定版本
        Scope:           model.ScopeProduct,
        SourceType:      datamodel.SourceAutoDiscovered,
        // ...
    }

    // 检查是否已存在同 OUI+ProductClass 的 active 自动模版
    // 如果已存在则复用（更新 parameter_tree），不重复创建
    existing, err := s.dmRepo.FindExistingAuto(ctx, device.Carrier, device.Technology,
        device.OUI, device.ProductClass)
    if err == nil && existing != nil {
        // 更新现有模版的 parameter_tree 和 last_accessed_at
        return s.dmRepo.UpdateParameterTree(ctx, existing.ID, parameterTree)
    }

    // 不存在则创建新模版
    return s.dmRepo.Create(ctx, dm)
}
```

---

## 6. 过期清理机制

### 6.1 定时清理任务

使用 `robfig/cron` 注册一个每日执行的清理任务（建议凌晨 3:00）：

```go
// internal/config/datamodel/cleaner.go

const (
    AutoTemplateMaxIdleDays   = 15  // 自动模版最大空闲天数
    ManualTemplateMaxIdleDays = 60  // 手动模版最大空闲天数
)

type DataModelCleaner struct {
    repo   DataModelRepository
    cache  *DataModelCache
    logger *zap.Logger
}

func (c *DataModelCleaner) CleanExpired(ctx context.Context) error {
    deleted, err := c.repo.DeleteExpired(ctx, AutoTemplateMaxIdleDays, ManualTemplateMaxIdleDays)
    if err != nil {
        return fmt.Errorf("clean expired data models: %w", err)
    }

    if deleted > 0 {
        c.logger.Info("expired data models cleaned",
            zap.Int64("deleted_count", deleted),
            zap.Int("auto_max_idle_days", AutoTemplateMaxIdleDays),
            zap.Int("manual_max_idle_days", ManualTemplateMaxIdleDays),
        )
        // 清理后需要失效缓存
        if c.cache != nil {
            if _, err := c.cache.IncrCacheVersion(ctx); err != nil {
                c.logger.Warn("failed to increment cache version after cleanup", zap.Error(err))
            }
        }
    }

    return nil
}
```

### 6.2 Cron 注册

```go
// cmd/app/main.go 或 router/deps.go 中注册

scheduler := cron.New()
cleaner := datamodel.NewDataModelCleaner(dmRepo, dmCache, logger)

// 每天凌晨 3:00 执行清理
scheduler.AddFunc("0 3 * * *", func() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := cleaner.CleanExpired(ctx); err != nil {
        logger.Error("data model cleanup failed", zap.Error(err))
    }
})
scheduler.Start()
```

### 6.3 配置化

```yaml
# configs/app.yaml
datamodel:
  expiry:
    auto_max_idle_days: 15    # 自动模版最大空闲天数
    manual_max_idle_days: 60  # 手动模版最大空闲天数
    cleanup_cron: "0 3 * * *" # 清理 cron 表达式
```

---

## 7. Redis 缓存策略调整

### 7.1 缓存键调整

现有键格式不变，增加访问追踪相关的键：

```
# 现有（不变）
datamodel:product:{carrier}:{tech}:{oui}:{product_class}
datamodel:oui:{carrier}:{tech}:{oui}
datamodel:default:{carrier}:{tech}
datamodel:resolve:{carrier}:{tech}:{oui}:{product_class}

# 新增：带版本的产品级缓存
datamodel:product:{carrier}:{tech}:{oui}:{product_class}:{firmware_version}
```

### 7.2 访问计数去重

`touchLastAccessedAsync` 使用内存 `sync.Map` 做本地去重（60 分钟窗口），避免对数据库产生写压力。

在 10 万设备规模下，假设 5000 种 OUI+ProductClass 组合：
- 最坏情况：每 60 分钟 5000 次 UPDATE（平均 1.4 次/秒）— 完全可以接受

---

## 8. 匹配流程图

```
设备 Inform 携带 OUI=00E0FC, ProductClass=LTE-eNodeB, FW=21.301.001
                    │
                    ▼
            ┌───────────────┐
            │ L1 内存缓存    │ key: cmcc:lte:00E0FC:LTE-eNodeB:21.301.001
            └───────┬───────┘
                    │ Miss
                    ▼
            ┌───────────────┐
            │ L2 Redis 缓存  │
            └───────┬───────┘
                    │ Miss
                    ▼
            ┌───────────────────────────┐
            │ L3 DB: FindActiveForMatch │
            │                           │
            │ 1. product + FW=21.301.001│ → 有手动模版? 返回
            │    ↓ 未命中               │
            │ 2. product + FW=NULL      │ → manual优先 > auto
            │    ↓ 未命中               │
            │    → nil（无匹配模版）     │
            └───────────┬───────────────┘
                        │ 命中
                        ▼
              ┌──────────────────┐
              │ 写回 L1 + L2 缓存 │
              │ 异步更新          │
              │ last_accessed_at  │
              └──────────────────┘
```

---

## 9. Web 导入 API 设计

### 9.1 导入接口

```
POST /api/v1/config/datamodels/import
Content-Type: multipart/form-data

参数:
  file         — CSV/XML 文件
  carrier      — 运营商代码
  technology   — 制式
  oui          — OUI（必填）
  product_class — 产品类型（必填）
  firmware_version — 固件版本（可选，不填则适用所有版本）
```

### 9.2 导入逻辑

```go
func (h *DataModelHandler) Import(c *gin.Context) {
    // 1. 解析上传文件和参数
    // 2. source_type = "manual"
    // 3. 如果 firmware_version 为空 → 创建无版本手动模版
    //    如果 firmware_version 不为空 → 创建带版本手动模版
    // 4. 检查冲突：如果已存在同 source_type+oui+product_class+firmware_version 的 active 模版
    //    → 提示用户是替换还是保留
    // 5. 创建/替换模版，设置 last_accessed_at = NOW()
}
```

---

## 10. 实施计划

### Phase 1: 数据库迁移 ✅
- [x] 编写 `000058_datamodel_add_last_accessed_at.up/down.sql`
- [x] 包含 `last_accessed_at` 字段、索引、唯一约束调整

### Phase 2: Model + Repository ✅
- [x] `DataModel` 结构体增加 `LastAccessedAt` 字段
- [x] `DataModelRepository` 增加 `TouchLastAccessed`、`DeleteExpired`、`FindActiveForMatch`
- [x] 实现 `pg_repository.go` 中的新方法

### Phase 3: Registry 匹配逻辑 ✅
- [x] `ResolveWithFirmware` 增加 `touchLastAccessedAsync` 调用
- [x] 增加 `touchThrottle sync.Map` 做访问去重
- [x] 优化 `resolveFromDBWithFirmware` 使用 `FindActiveForMatch`

### Phase 4: 过期清理 ✅
- [x] 实现 `DataModelCleaner`
- [ ] 在 `cmd/app/main.go` 注册 cron 任务（待集成）
- [x] `appconfig` 增加过期配置字段

### Phase 5: Discovery 适配 ✅
- [x] `HandleDiscoveryResult` 中确保 auto 模版 `firmware_version = ""`
- [x] 增加 "已存在则更新" 逻辑，避免重复创建

### Phase 6: 测试 ✅
- [x] 匹配优先级测试（二级精确匹配 + source_type 排序）
- [ ] 过期清理测试（待补充专项测试）
- [ ] 并发访问更新测试（待补充专项测试）
- [x] 所有现有测试通过（52 个测试模块，0 失败）

---

## 11. 风险与对策

| 风险 | 概率 | 影响 | 对策 |
|------|------|------|------|
| `last_accessed_at` 高频写入导致 DB 压力 | 低 | 中 | 60 分钟去重窗口，最坏 1.4 writes/s |
| 唯一约束变更导致迁移失败 | 低 | 高 | down 迁移恢复旧约束；先删旧索引再建新索引 |
| 清理任务误删正在使用的模版 | 极低 | 高 | 每次匹配命中都更新 `last_accessed_at`；15 天阈值足够保守 |
| L1 缓存未及时感知模版被清理 | 低 | 低 | 清理后 `IncrCacheVersion` 触发全局缓存刷新 |

---

## 12. 数据示例

### 场景：华为 LTE 小基站，移动网络

```
设备: OUI=00E0FC, ProductClass=LTE-eNodeB, FW=21.301.001, Carrier=cmcc, Tech=lte
```

数据库中可能存在的模版：

| id | oui | product_class | firmware_version | source_type | scope | 匹配优先级 |
|----|-----|--------------|-----------------|-------------|-------|-----------|
| A  | 00E0FC | LTE-eNodeB | 21.301.001 | manual | product | 1 (最高) |
| B  | 00E0FC | LTE-eNodeB | NULL | auto_discovered | product | 2 |
| C  | 00E0FC | LTE-eNodeB | NULL | manual | product | 2 (但 manual > auto) |

**匹配结果**：
- 如果 A 存在 → 返回 A（精确版本匹配）
- 如果 A 不存在，B 和 C 都存在 → 返回 C（manual 优先于 auto）
- 如果 A/B/C 都不存在 → 返回 nil（无匹配模版，设备等待自动发现或手动导入）
