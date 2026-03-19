# 04 — 数据模型与配置服务

> config-rpc 服务：三级回退解析、三级缓存、配置模板管理

---

## 1. 服务概述

config-rpc 是 go-zero zRPC 服务，负责 F02 功能域的核心能力：

| 能力 | 说明 |
|------|------|
| 数据模型注册 | 管理 TR069 参数树定义（JSONB） |
| 三级回退解析 | product → oui → carrier_default |
| 三级缓存 | L1 内存 → L2 Redis → L3 PostgreSQL |
| 配置模板 | 开站/变更配置模板管理 |
| OUI 注册表 | 厂商 OUI 映射 |
| 缓存协调 | 跨实例缓存失效通知 |

---

## 2. Proto 接口定义

```protobuf
// api/proto/config.proto
syntax = "proto3";

package config;
option go_package = "omcgo/service/config/rpc/pb";

service ConfigService {
    // ========== 数据模型解析 ==========

    // 三级回退解析：根据设备标识找到最匹配的数据模型
    rpc ResolveDataModel(ResolveDataModelReq) returns (ResolveDataModelResp);

    // ========== 数据模型 CRUD ==========

    rpc GetDataModel(GetDataModelReq) returns (DataModelResp);
    rpc ListDataModels(ListDataModelsReq) returns (ListDataModelsResp);
    rpc ImportDataModel(ImportDataModelReq) returns (ImportDataModelResp);
    rpc ActivateDataModel(ActivateDataModelReq) returns (ActivateDataModelResp);
    rpc DeprecateDataModel(DeprecateDataModelReq) returns (DeprecateDataModelResp);

    // ========== 配置模板 ==========

    rpc GetConfigTemplate(GetConfigTemplateReq) returns (ConfigTemplateResp);
    rpc MatchTemplate(MatchTemplateReq) returns (ConfigTemplateResp);
    rpc ListTemplates(ListTemplatesReq) returns (ListTemplatesResp);
    rpc CreateTemplate(CreateTemplateReq) returns (ConfigTemplateResp);
    rpc UpdateTemplate(UpdateTemplateReq) returns (ConfigTemplateResp);
    rpc DeleteTemplate(DeleteTemplateReq) returns (DeleteTemplateResp);

    // ========== OUI 注册表 ==========

    rpc ListOUI(ListOUIReq) returns (ListOUIResp);
    rpc GetOUIByCode(GetOUIByCodeReq) returns (OUIResp);
}

// ========== 数据模型解析 ==========

message ResolveDataModelReq {
    string carrier = 1;        // cmcc / ctcc / cucc
    string technology = 2;     // lte / nr
    string oui = 3;            // 设备 OUI
    string product_class = 4;  // 设备产品类型
}

message ResolveDataModelResp {
    int64 model_id = 1;
    string scope = 2;           // product / oui / carrier_default
    string carrier = 3;
    string technology = 4;
    string version = 5;
    bytes parameter_tree = 6;   // JSON 编码的参数树
    bool from_cache = 7;        // 是否命中缓存
}

// ========== 数据模型 CRUD ==========

message GetDataModelReq {
    int64 id = 1;
}

message DataModelResp {
    int64 id = 1;
    string carrier = 2;
    string technology = 3;
    string scope = 4;
    string oui = 5;
    string product_class = 6;
    string version = 7;
    string status = 8;           // draft / active / deprecated
    bytes parameter_tree = 9;    // JSON
    int64 created_at = 10;
    int64 updated_at = 11;
}

message ListDataModelsReq {
    string carrier = 1;
    string technology = 2;
    string scope = 3;
    string status = 4;
    int32 page = 5;
    int32 page_size = 6;
}

message ListDataModelsResp {
    repeated DataModelResp models = 1;
    int64 total = 2;
}

message ImportDataModelReq {
    string carrier = 1;
    string technology = 2;
    string scope = 3;
    string oui = 4;
    string product_class = 5;
    string version = 6;
    bytes parameter_tree = 7;    // JSON 编码
}

message ImportDataModelResp {
    int64 id = 1;
    string status = 2;          // 创建为 draft 状态
}

message ActivateDataModelReq {
    int64 id = 1;
}

message ActivateDataModelResp {
    bool success = 1;
    int64 deprecated_model_id = 2; // 被自动废弃的旧模型 ID
}

message DeprecateDataModelReq {
    int64 id = 1;
}

message DeprecateDataModelResp {
    bool success = 1;
}

// ========== 配置模板 ==========

message GetConfigTemplateReq {
    int64 id = 1;
}

message MatchTemplateReq {
    string carrier = 1;
    string technology = 2;
    string product_class = 3;
}

message ConfigTemplateResp {
    int64 id = 1;
    string name = 2;
    string carrier = 3;
    string technology = 4;
    string product_class = 5;    // 为空表示通用模板
    int32 priority = 6;
    bytes template_data = 7;     // JSON: 参数列表、下载项、重启标志
    int64 created_at = 8;
    int64 updated_at = 9;
}

message ListTemplatesReq {
    string carrier = 1;
    string technology = 2;
    int32 page = 3;
    int32 page_size = 4;
}

message ListTemplatesResp {
    repeated ConfigTemplateResp templates = 1;
    int64 total = 2;
}

message CreateTemplateReq {
    string name = 1;
    string carrier = 2;
    string technology = 3;
    string product_class = 4;
    int32 priority = 5;
    bytes template_data = 6;
}

message UpdateTemplateReq {
    int64 id = 1;
    string name = 2;
    int32 priority = 3;
    bytes template_data = 4;
}

message DeleteTemplateReq {
    int64 id = 1;
}

message DeleteTemplateResp {
    bool success = 1;
}

// ========== OUI ==========

message ListOUIReq {
    string manufacturer = 1;   // 过滤厂商
    int32 page = 2;
    int32 page_size = 3;
}

message ListOUIResp {
    repeated OUIResp ouis = 1;
    int64 total = 2;
}

message GetOUIByCodeReq {
    string oui = 1;
}

message OUIResp {
    string oui = 1;
    string manufacturer = 2;
    string description = 3;
}
```

---

## 3. 三级回退解析

### 3.1 解析算法

```
输入: carrier, technology, oui, product_class
输出: 最匹配的 DataModel

1. 查缓存 (L1 → L2 → L3):
   Key = resolve:{carrier}:{tech}:{oui}:{product_class}
   → 命中则直接返回

2. product 级查询:
   SELECT * FROM data_model_definitions
   WHERE carrier = $1 AND technology = $2 AND oui = $3
     AND product_class = $4 AND scope = 'product' AND status = 'active'
   → 命中则写入缓存并返回

3. oui 级回退:
   SELECT * FROM data_model_definitions
   WHERE carrier = $1 AND technology = $2 AND oui = $3
     AND scope = 'oui' AND status = 'active'
   → 命中则写入缓存并返回

4. carrier_default 级回退:
   SELECT * FROM data_model_definitions
   WHERE carrier = $1 AND technology = $2
     AND scope = 'carrier_default' AND status = 'active'
   → 命中则写入缓存并返回

5. 未找到 → 返回错误
```

### 3.2 解析优先级

| 优先级 | Scope | 匹配条件 | 适用场景 |
|:------:|-------|---------|---------|
| 1（最高） | product | carrier + tech + OUI + ProductClass | 特定设备型号 |
| 2 | oui | carrier + tech + OUI | 厂商默认 |
| 3（最低） | carrier_default | carrier + tech | 运营商默认 |

### 3.3 实现代码（Logic 层）

```go
// service/config/rpc/internal/logic/resolvedatamodellogic.go

func (l *ResolveDataModelLogic) ResolveDataModel(
    in *pb.ResolveDataModelReq,
) (*pb.ResolveDataModelResp, error) {

    cacheKey := fmt.Sprintf("resolve:%s:%s:%s:%s",
        in.Carrier, in.Technology, in.Oui, in.ProductClass)

    // 1. L1 内存缓存
    if cached, ok := l.svcCtx.L1Cache.Load(cacheKey); ok {
        resp := cached.(*pb.ResolveDataModelResp)
        resp.FromCache = true
        return resp, nil
    }

    // 2. L2 Redis 缓存
    redisKey := "datamodel:" + cacheKey
    if data, err := l.svcCtx.Redis.Get(l.ctx, redisKey).Bytes(); err == nil {
        var resp pb.ResolveDataModelResp
        if json.Unmarshal(data, &resp) == nil {
            resp.FromCache = true
            l.svcCtx.L1Cache.Store(cacheKey, &resp) // 回填 L1
            return &resp, nil
        }
    }

    // 3. L3 PostgreSQL 三级回退查询
    model, scope, err := l.svcCtx.DataModelRepo.Resolve(l.ctx,
        in.Carrier, in.Technology, in.Oui, in.ProductClass)
    if err != nil {
        return nil, fmt.Errorf("resolve data model: %w", err)
    }

    resp := &pb.ResolveDataModelResp{
        ModelId:       model.ID,
        Scope:         scope,
        Carrier:       model.Carrier,
        Technology:    model.Technology,
        Version:       model.Version,
        ParameterTree: model.ParameterTree,
        FromCache:     false,
    }

    // 写入 L2 Redis（TTL 1 小时）
    if data, err := json.Marshal(resp); err == nil {
        l.svcCtx.Redis.Set(l.ctx, redisKey, data, time.Hour)
    }
    // 写入 L1 内存
    l.svcCtx.L1Cache.Store(cacheKey, resp)

    return resp, nil
}
```

---

## 4. 三级缓存策略

### 4.1 缓存层级

```
┌─────────────────────────────────────────────────────────┐
│ L1: 进程内内存 (sync.Map)                                │
│ 延迟: < 1μs | 容量: ~1000 条 | 失效: 进程重启/主动清除    │
├─────────────────────────────────────────────────────────┤
│ L2: Redis                                               │
│ 延迟: < 1ms  | 容量: 无限    | TTL: 解析结果 1h / 模型 24h │
├─────────────────────────────────────────────────────────┤
│ L3: PostgreSQL (data_model_definitions)                  │
│ 延迟: < 10ms | 容量: 无限    | 持久化存储                  │
└─────────────────────────────────────────────────────────┘
```

### 4.2 Redis Key 命名

```
datamodel:product:{carrier}:{tech}:{oui}:{product_class}  — 产品级模型缓存
datamodel:oui:{carrier}:{tech}:{oui}                       — OUI 级模型缓存
datamodel:default:{carrier}:{tech}                          — 默认级模型缓存
datamodel:resolve:{carrier}:{tech}:{oui}:{product_class}   — 解析结果缓存 (TTL 1h)
datamodel:cache_version                                     — 缓存版本号
```

### 4.3 缓存失效机制

```
数据模型激活 (ActivateDataModel)
    │
    ├── 1. PostgreSQL: UPDATE status = 'active', 旧模型 status = 'deprecated'
    │
    ├── 2. Redis: INCR datamodel:cache_version
    │
    ├── 3. Redis: DEL 相关 datamodel:resolve:* keys
    │
    ├── 4. NATS: Publish "datamodel.cache.invalidated" 事件
    │
    └── 5. 其他 config-rpc 实例和 ACS 实例收到事件后:
           - 清除 L1 内存缓存
           - 检查 cache_version 是否匹配
```

---

## 5. 数据库 Schema

### 5.1 data_model_definitions

```sql
CREATE TABLE data_model_definitions (
    id              BIGSERIAL PRIMARY KEY,
    carrier         VARCHAR(10)  NOT NULL,    -- cmcc / ctcc / cucc
    technology      VARCHAR(10)  NOT NULL,    -- lte / nr
    scope           VARCHAR(20)  NOT NULL,    -- product / oui / carrier_default
    oui             VARCHAR(10),              -- 厂商 OUI（scope=carrier_default 时为 NULL）
    product_class   VARCHAR(100),             -- 产品类型（scope=product 时才有值）
    version         VARCHAR(20)  NOT NULL,    -- 版本号
    status          VARCHAR(20)  NOT NULL DEFAULT 'draft',  -- draft / active / deprecated
    parameter_tree  JSONB        NOT NULL,    -- TR069 参数树
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 同分类仅一个 active 模型
CREATE UNIQUE INDEX idx_dm_active_unique
ON data_model_definitions (carrier, technology, scope, oui, product_class)
WHERE status = 'active';

-- 三级回退查询索引
CREATE INDEX idx_dm_resolve
ON data_model_definitions (carrier, technology, scope, status);
```

### 5.2 config_templates

```sql
CREATE TABLE config_templates (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(200) NOT NULL,
    carrier         VARCHAR(10)  NOT NULL,
    technology      VARCHAR(10)  NOT NULL,
    product_class   VARCHAR(100),             -- NULL = 通用模板
    priority        INT          NOT NULL DEFAULT 0,
    template_data   JSONB        NOT NULL,    -- 参数列表、下载项、重启标志
    enabled         BOOLEAN      NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ct_match
ON config_templates (carrier, technology, product_class, priority DESC)
WHERE enabled = true;
```

### 5.3 oui_registry

```sql
CREATE TABLE oui_registry (
    oui             VARCHAR(10)  PRIMARY KEY,
    manufacturer    VARCHAR(200) NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- goctl model 可生成此表的 CRUD（简单表）
```

---

## 6. .api 文件定义（REST 层）

数据模型 REST API 通过 device-api 网关暴露，device-api 调用 config-rpc。

```go
// service/device/api/internal/logic/datamodel/listdatamodelslogic.go

func (l *ListDataModelsLogic) ListDataModels(req *types.DataModelListReq) (
    *types.DataModelListResp, error,
) {
    resp, err := l.svcCtx.ConfigRpc.ListDataModels(l.ctx, &config.ListDataModelsReq{
        Carrier:    req.Carrier,
        Technology: req.Technology,
        Scope:      req.Scope,
        Status:     req.Status,
        Page:       int32(req.Page),
        PageSize:   int32(req.PageSize),
    })
    if err != nil {
        return nil, err
    }

    // 转换 RPC 响应为 API 响应
    return convertToAPIResp(resp), nil
}
```

---

## 7. 数据模型生命周期

```
  创建/导入
     │
     ▼
  ┌───────┐
  │ draft │ ← 初始状态，可编辑
  └───┬───┘
      │ ActivateDataModel
      ▼
  ┌────────┐
  │ active │ ← 生效中，同分类仅一个 active
  └───┬────┘
      │ 新模型激活时自动触发 / 手动 DeprecateDataModel
      ▼
  ┌──────────────┐
  │ deprecated   │ ← 已废弃，保留历史记录
  └──────────────┘
```

激活新模型时的事务：
1. 将同分类（carrier + tech + scope + oui + product_class）的旧 active 模型设为 deprecated
2. 将新模型设为 active
3. 递增缓存版本号
4. 发布缓存失效事件
