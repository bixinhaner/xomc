# DD-08: 数据模型与配置管理（F02）

> 关联功能域：F02（数据模型与配置管理��
> 关联 backend-design.md 章节：第十三章（数据模型抽象）、第五章（PostgreSQL Schema）
> 实施阶段：Phase 2（核心功能）
> 依赖文档：DD-02, DD-03, DD-05

---

## 1. 概述

### 1.1 模块定位

数据模型与配置管理（`internal/config/`）是 OMC 系统的数据基础，定义了每种运营商/制式/厂商/产品型号的 TR069 参数树，支持三级回退解析和三级缓存。

### 1.2 核心职责

- 数据模型注册表（三级回退解析：product → oui → carrier_default）
- 三级缓存（内存 L1 + Redis L2 + PostgreSQL L3）
- 数据模型导入/导出工作流
- 数据模型生命周期管理（draft → active → deprecated）
- 配置模板管理
- 配置审计与备份

---

## 2. 接口设计

### 2.1 DataModelRegistry — `internal/config/datamodel/registry.go`

```go
type DataModelRegistry struct {
    repo         DataModelRepository
    cache        *DataModelCache
    localCache   sync.Map    // 内存 L1
    cacheVersion int64
    logger       *zap.Logger
}

// Resolve 三级回退解析
func (r *DataModelRegistry) Resolve(ctx context.Context, carrier model.CarrierCode,
    tech model.Technology, oui, productClass string) (*DataModel, error)

// ResolveForDevice 为设备解析并关联数据模型
func (r *DataModelRegistry) ResolveForDevice(ctx context.Context, device *model.Device) (*DataModel, error)

// InvalidateCache 缓存失效
func (r *DataModelRegistry) InvalidateCache(ctx context.Context, model *DataModel) error
```

### 2.2 DataModelRepository — `internal/config/datamodel/repository.go`

```go
type DataModelRepository interface {
    Create(ctx context.Context, model *DataModel) error
    GetByID(ctx context.Context, id uuid.UUID) (*DataModel, error)
    Update(ctx context.Context, model *DataModel) error
    Delete(ctx context.Context, id uuid.UUID) error  // 仅 draft 可删
    List(ctx context.Context, filter DataModelFilter) ([]*DataModel, error)
    FindActive(ctx context.Context, carrier model.CarrierCode, tech model.Technology,
        oui, productClass string, scope DataModelScope) (*DataModel, error)
    Activate(ctx context.Context, id uuid.UUID) error
    Deprecate(ctx context.Context, id uuid.UUID) error
    Statistics(ctx context.Context) (*DataModelStats, error)
}
```

### 2.3 DataModelCache — `internal/config/datamodel/cache.go`

```go
type DataModelCache struct {
    redis redis.UniversalClient
}

// 模型内容缓存
func (c *DataModelCache) GetModel(ctx context.Context, scope, carrier, tech, oui, productClass string) (*DataModel, error)
func (c *DataModelCache) SetModel(ctx context.Context, model *DataModel) error

// 解析结果缓存
func (c *DataModelCache) GetResolveResult(ctx context.Context, carrier, tech, oui, productClass string) (uuid.UUID, error)
func (c *DataModelCache) SetResolveResult(ctx context.Context, carrier, tech, oui, productClass string, modelID uuid.UUID) error

// 缓存版本号
func (c *DataModelCache) GetCacheVersion(ctx context.Context) (int64, error)
func (c *DataModelCache) IncrCacheVersion(ctx context.Context) (int64, error)

// 失效
func (c *DataModelCache) InvalidateModel(ctx context.Context, model *DataModel) error
```

**Redis Key 设计**：

```
datamodel:product:{carrier}:{tech}:{oui}:{product_class}  → 压缩 JSON (TTL 24h)
datamodel:oui:{carrier}:{tech}:{oui}                      → 压缩 JSON (TTL 24h)
datamodel:default:{carrier}:{tech}                         → 压缩 JSON (TTL 24h)
datamodel:resolve:{carrier}:{tech}:{oui}:{product_class}  → UUID (TTL 1h)
datamodel:cache_version                                     → Counter (无 TTL)
```

---

## 3. 数据模型

### 3.1 核心结构体

```go
type DataModelScope string

const (
    ScopeProduct        DataModelScope = "product"
    ScopeOUI            DataModelScope = "oui"
    ScopeCarrierDefault DataModelScope = "carrier_default"
)

type DataModelStatus string

const (
    StatusDraft      DataModelStatus = "draft"
    StatusActive     DataModelStatus = "active"
    StatusDeprecated DataModelStatus = "deprecated"
)

type DataModel struct {
    ID             uuid.UUID
    Carrier        model.CarrierCode
    Technology     model.Technology
    Version        string
    OUI            string
    ProductClass   string
    Scope          DataModelScope
    Status         DataModelStatus
    IsActive       bool
    RootObject     string
    ParameterTree  json.RawMessage // JSONB 存储
    Parameters     map[string]*Parameter
    Objects        map[string]*Object
    Source         string
    ImportedBy     string
    SpecDocumentRef string
    Description    string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type Parameter struct {
    Path        string
    UnifiedName string
    Type        model.ParameterType
    Writable    bool
    Description string
    Constraints *Constraints
    Category    string
}

type Constraints struct {
    MinValue   *int64
    MaxValue   *int64
    EnumValues []string
    Pattern    string
    MaxLength  int
}

type Object struct {
    Path          string
    MultiInstance bool
    Writable      bool
    Parameters    []string
    SubObjects    []string
}
```

### 3.2 数据库 Schema

```sql
CREATE TABLE data_model_definitions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    carrier          VARCHAR(4) NOT NULL,
    technology       VARCHAR(3) NOT NULL,
    version          VARCHAR(16) NOT NULL,
    oui              VARCHAR(6),
    product_class    VARCHAR(64),
    scope            VARCHAR(16) NOT NULL
                     CHECK (scope IN ('product', 'oui', 'carrier_default')),
    status           VARCHAR(12) NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft', 'active', 'deprecated')),
    is_active        BOOLEAN NOT NULL DEFAULT false,
    root_object      VARCHAR(64) NOT NULL DEFAULT 'Device.',
    parameter_tree   JSONB NOT NULL,
    source           VARCHAR(32),
    imported_by      VARCHAR(128),
    spec_document_ref VARCHAR(256),
    description      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_scope_fields CHECK (
        (scope = 'carrier_default' AND oui IS NULL AND product_class IS NULL) OR
        (scope = 'oui' AND oui IS NOT NULL AND product_class IS NULL) OR
        (scope = 'product' AND oui IS NOT NULL AND product_class IS NOT NULL)
    )
);

-- 每分类仅一个活跃模型
CREATE UNIQUE INDEX idx_dm_active_product
    ON data_model_definitions (carrier, technology, oui, product_class)
    WHERE is_active = true AND scope = 'product';

CREATE UNIQUE INDEX idx_dm_active_oui
    ON data_model_definitions (carrier, technology, oui)
    WHERE is_active = true AND scope = 'oui' AND product_class IS NULL;

CREATE UNIQUE INDEX idx_dm_active_carrier_default
    ON data_model_definitions (carrier, technology)
    WHERE is_active = true AND scope = 'carrier_default' AND oui IS NULL;

-- OUI 厂商注册表
CREATE TABLE oui_registry (
    oui              VARCHAR(6) PRIMARY KEY,
    manufacturer     VARCHAR(128) NOT NULL,
    short_name       VARCHAR(32) NOT NULL,
    country          VARCHAR(64),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 导入审计日志
CREATE TABLE data_model_import_log (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_model_id    UUID NOT NULL REFERENCES data_model_definitions(id),
    action           VARCHAR(16) NOT NULL,
    performed_by     VARCHAR(128) NOT NULL,
    changes_summary  JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

---

## 4. 详细设计

### 4.1 三级回退解析算法

```
Resolve(carrier, tech, oui, productClass):
  │
  ├── 1. 查内存 L1 缓存 → 命中则返回
  │
  ├── 2. 查 Redis L2 缓存（resolve 结果）→ 命中则获取模型内容并返回
  │
  ├── 3. 查 DB（三级回退）：
  │     ├── Level 1 (product): WHERE carrier=? AND tech=? AND oui=? AND product_class=? AND scope='product' AND is_active=true
  │     ├── Level 2 (oui):     WHERE carrier=? AND tech=? AND oui=? AND scope='oui' AND is_active=true
  │     └── Level 3 (default): WHERE carrier=? AND tech=? AND scope='carrier_default' AND is_active=true
  │
  ├── 4. 回写缓存（L1 + L2）
  │
  └── 5. 返回匹配的 DataModel
```

### 4.2 数据模型导入工作流

```
1. 管理员上传 JSON 文件 → POST /api/v1/datamodels/import
2. 校验 JSON 格式（或 dry-run: POST /api/v1/datamodels/import/validate）
3. 自动确定 scope（根据 oui/product_class 字段）
4. 写入 DB（status = "draft"）
5. 记录导入日志
6. 管理员审核后激活 → POST /api/v1/datamodels/{id}/activate
7. 激活操作：
   - 同分类旧活跃模型 → deprecated
   - 新模型 → is_active=true, status="active"
   - 清除相关 Redis 缓存
   - 自增 datamodel:cache_version
```

### 4.3 配置模板管理

```go
type ConfigTemplateService struct {
    repo ConfigTemplateRepository
}

type ConfigTemplate struct {
    ID           uuid.UUID
    Name         string
    Carrier      model.CarrierCode
    Technology   model.Technology
    TemplateType string // provisioning, batch_config, firmware_upgrade
    Parameters   map[string]interface{} // JSONB
    Version      int
    Active       bool
    CreatedAt    time.Time
}

// Match 根据设备信息匹配最佳模板
func (s *ConfigTemplateService) Match(ctx context.Context, device *model.Device) (*ConfigTemplate, error)
```

### 4.4 REST API

```
GET    /api/v1/datamodels                     列表
GET    /api/v1/datamodels/{id}                详情
POST   /api/v1/datamodels                     创建
PUT    /api/v1/datamodels/{id}                更新
DELETE /api/v1/datamodels/{id}                删除（仅 draft）
POST   /api/v1/datamodels/{id}/activate       激活
POST   /api/v1/datamodels/{id}/deprecate      废弃
POST   /api/v1/datamodels/import              导入
POST   /api/v1/datamodels/import/validate     导入校验（dry-run）
GET    /api/v1/datamodels/{id}/export         导出
GET    /api/v1/datamodels/resolve             解析测试
GET    /api/v1/datamodels/{id}/diff/{other}   差异对比
GET    /api/v1/datamodels/statistics          统计
POST   /api/v1/datamodels/cache/refresh       刷新缓存
GET    /api/v1/oui                            OUI 列表
POST   /api/v1/oui                            注册 OUI
```

---

## 5. 运营商差异

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| LTE 数据模型版本 | V2.1, V2.3 | V1.1 (贵州) | — |
| NR 数据模型版本 | V1.7, V1.9.4 | V2.1.7 | V1.1 |
| 根对象 | Device. | Device. | Device. |
| 开站模板 | 是 | 是（4G+5G）| 否 |

---

## 6. 实施子阶段

### 阶段 8a：DataModel 类型 + Repository + DB Schema（Phase 2）
### 阶段 8b：三级回退解析 + 三级缓存（Phase 2）
### 阶段 8c：导入工作流 + 种子数据（Phase 2）
### 阶段 8d：配置模板（Phase 2）
### 阶段 8e：审计 + 备份 + 批量操作（Phase 3/4）

---

## 7. 文件清单

```
internal/config/datamodel/registry.go
internal/config/datamodel/repository.go
internal/config/datamodel/cache.go
internal/config/datamodel/import.go
internal/config/datamodel/validator.go
internal/config/datamodel/parameter.go
internal/config/template/template.go
internal/config/template/matcher.go
internal/config/audit/audit.go
internal/config/backup/backup.go
internal/config/batch/batch.go
internal/config/service.go
datamodels/seed/carrier_defaults/
datamodels/seed/product_models/
datamodels/templates/model_schema.json
```

---

## 8. 测试策略

- 三级回退解析：table-driven 测试（有/无 product/oui/default 的各种组合）
- 缓存：L1/L2 命中/失效测试
- 导入工作流：JSON 解析 + scope 自动确定 + 激活流程
- 生命周期：draft → active → deprecated 状态转移
- 并发唯一约束：同分类仅一个 active

---

## 9. 参考

- backend-design.md 第十三章：数据模型抽象（13.1-13.7）
- doc/features/02-data-model.md：F02 全部子功能
- CLAUDE.md 第 5.3 节：数据模型专项规范
