# 系统管理 — 设备分类（System / Device Classification）PRD

> 文档目的：以"网络制式 → 产品类型"两级树展示设备库存，支持按分类钻取设备列表。
> ⚠️ 当前前端使用 mock 数据，后端无专项 API，整体处于"占位"状态。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 现状记录 + 后端实现路径 |

**关联功能域**：F06 OMC-R 核心 / 设备管理（与 F02 数据模型有交叉）

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/DeviceClassification/index.tsx](../../../../omcmb/webcode/src/pages/system/DeviceClassification/index.tsx) |
| 后端（待实现）| `internal/device/` 或新模块 `internal/device_classification/` |
| 数据库 | `devices` 表（[migrations/000003_devices.sql](../../../migrations/000003_devices.sql)）|

---

## 1. 业务背景

OMC 管理多种设备类型，按两级分类组织：

```
全部设备
├── 4G 设备 (LTE)
│   ├── eNB 基站
│   ├── RRU
│   └── BBU
├── 5G 设备 (NR)
│   ├── gNB 基站
│   ├── AAU
│   ├── CU
│   └── DU
└── CPE 终端
    ├── 室内 CPE
    └── 室外 CPE
```

**用途**：
- 运维人员快速定位某一类设备
- 统计各类设备库存 / 在线率
- 与 F02 数据模型解析挂钩（同类设备共用 product 级数据模型）

**与「设备分组」（DeviceGroup）的区别**：
- **设备分类**：按设备**类型**（产品维度）划分，全局固定
- **设备分组**：按**业务用途**划分（如"华东运维分组"），可灵活配置，用于数据权限（参 [users.md §1.3](./users.md)）

---

## 2. 实体模型

### 2.1 当前可用字段（`devices` 表，[migrations/000003_devices.sql](../../../migrations/000003_devices.sql)）

设备分类的依据字段：

| 字段 | 用途 |
|------|------|
| `network_type` | 网络制式：`lte` / `nr` / `gsm` / `cpe` / `egw` |
| `product_class` | 产品类型：`eNB` / `gNB` / `RRU` / `BBU` / 等（自由文本，TR-069 上报）|
| `oui` | 厂商 OUI（与 `product_class` 联合定位精确产品型号）|
| `vendor` | 厂商显示名 |

**当前现状**：
- 没有专门的 `device_classifications` 表
- 分类树**完全由 `devices` 表的 `(network_type, product_class)` 聚合派生**

### 2.2 派生分类树查询（建议实现）

```sql
SELECT
    network_type,
    product_class,
    COUNT(*) AS device_count,
    COUNT(*) FILTER (WHERE conn_status = 'online') AS online_count
FROM devices
WHERE deleted_at IS NULL
GROUP BY network_type, product_class
ORDER BY network_type, product_class;
```

派生结果由 service 层重组为前端期望的两级树：

```json
[
  {
    "key": "lte",
    "label": "4G 设备 (LTE)",
    "deviceCount": 177,
    "children": [
      { "key": "lte:eNB", "label": "eNB 基站", "deviceCount": 35, "onlineCount": 32 },
      { "key": "lte:RRU", "label": "RRU", "deviceCount": 120, "onlineCount": 110 },
      { "key": "lte:BBU", "label": "BBU", "deviceCount": 22, "onlineCount": 20 }
    ]
  },
  ...
]
```

### 2.3 自定义分类（可选 P2）

如果运营商需要自定义分类（如"核心区设备"），考虑加表：

```sql
CREATE TABLE device_classifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(128) NOT NULL,
    parent_id   UUID NULL REFERENCES device_classifications(id) ON DELETE CASCADE,
    rule        JSONB NOT NULL,        -- 匹配规则，如 {"network_type":"lte","oui":"00ABCD"}
    sort_order  INT DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

不在 P0/P1 范围。

---

## 3. 页面布局（左树 + 右表）

### 3.1 左侧分类树

| 节点 | 展示 |
|------|------|
| 顶级 | "全部设备" + 总数 |
| L1 | 网络制式（LTE/NR/CPE）+ 该制式下设备数 |
| L2 | 产品类型 + 该类型下设备数 |

支持搜索框过滤树节点。

### 3.2 右侧设备列表

| key | 列标题 | dataIndex | UI |
|-----|-------|-----------|------|
| `sn` | 设备序号 | `sn` | monospace |
| `name` | 设备名称 | `name` | 文本 |
| `vendor` | 厂商 | `vendor` | 文本 |
| `productType` | 产品类型 | `productType` | `<Tag>` |
| `networkType` | 网络制式 | `networkType` | `<Tag>` |
| `region` | 地区 | `region` | 文本 |
| `connStatus` | 连接状态 | `connStatus` | `<Tag>` 在线绿/离线红 |
| `alarmLevel` | 告警级别 | `alarmLevel` | `<Tag>` 严重红/警告黄/正常绿 |
| `softwareVersion` | 软件版本 | `softwareVersion` | monospace |

### 3.3 顶部右侧按钮

| 按钮 | 状态 | 行为 |
|------|------|------|
| 添加 | disabled | 设备由 TR-069 自动注册，不需要手工添加；保留按钮但禁用 |
| 删除 | disabled (批量) | 同上，删除走设备管理 |

> 这页主要是"导航 + 浏览"用途，不做 CRUD。

---

## 4. 操作清单

| 操作 | 触发 | 接口 |
|------|------|------|
| 选中分类节点 | 点击树 | 加载该分类下设备：`GET /api/v1/devices?network_type=&product_class=&page=&pageSize=` |
| 搜索设备 | 顶部搜索框 | 同上 + `&search=<keyword>`（按 sn/name 模糊匹配）|
| 跳转设备详情 | 点击行 | 跳到设备管理页 `/devices/{id}` |

---

## 5. 表单字段定义

无（无 CRUD）。

---

## 6. 接口契约

### 6.1 待实现

| Method | 路径 | 说明 | 优先级 |
|--------|------|------|--------|
| GET | `/api/v1/device-classifications/tree` | 派生分类树（按 §2.2 SQL 聚合）| P0 |
| GET | `/api/v1/devices?network_type=&product_class=` | 按分类查设备列表（**复用现有设备列表接口**，加 query 参数）| P0（已有？需确认）|

### 6.2 现有可复用

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/api/v1/devices` | 设备列表（F06 设备管理已实现），按 `network_type` / `product_class` query 过滤 |

---

## 7. 后端补齐 Backlog

### P0
1. **分类树聚合接口** `GET /device-classifications/tree`（§6.1）— 派生查询，无需新表
2. **`devices` 列表查询参数确认**：`network_type` / `product_class` 是否已支持 query 过滤

### P1
3. **缓存**：分类树查询频率高但变动慢（设备增删才变），加 5 分钟 Redis 缓存
4. **多语言制式名称**：当前前端硬编码"4G 设备 (LTE)"，从 dictionary（[data-dictionary.md](./data-dictionary.md)）取 `network_type` 字典展示

### P2
5. **自定义分类**（§2.3）：用户定义匹配规则 + 命名

---

## 8. 验收清单（DoD）

后端：
- [ ] `/device-classifications/tree` 返回 JSON 与前端期望结构对齐
- [ ] 聚合查询响应时间 < 300ms（10 万设备）

前端：
- [ ] 删除 `mockDevices` 硬编码数据，改用 `useDeviceClassificationTree()` + `useDevicesByClassification()`
- [ ] 选中树节点后右侧表实时刷新
- [ ] `npm run typecheck` & `lint` 通过

---

## 9. 非目标

- 设备 CRUD（属设备管理 PRD）
- 制式定义新增（如新增 `6G`）— 属字典管理
- 设备数据权限（属角色管理 + 用户管理 PRD）
