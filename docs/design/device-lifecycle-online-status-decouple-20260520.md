# Device List 页前后端对齐总改 — 设计方案

> **状态**：草案，待审批
> **作者**：Claude（与维护者协作）
> **日期**：2026-05-20
> **关联 Backlog**：建议落 **T-0162**
>
> **本次范围**（一次合并实施）：
> 1. **解耦** `devices.status` → `lifecycle_state` + `is_online` 两个正交字段
> 2. **修对齐 bug** device list 其它筛选项的前后端值不一致问题（op_state / network_type / 等）
> 3. **补齐空下拉** 设备型号 / 软件版本 / 固件版本走字典管理 + 初始化种子数据

---

## 1. 决策诉求

device list 页有 3 个根问题，本次一次解：

### 1.1 status 字段双重语义

`devices.status` 单字段当前承担 7 个枚举值，混淆了**生命周期**（业务流程进度）与**实时在线状态**（连接活动）两个本应正交的维度。这是 "在线统计 1 / 筛选 online 列表为空" bug 的根因；代码作者自己在 `device.go:79` 注释里写过 "在线却未激活 / 离线却激活之类自相矛盾"。

### 1.2 多个筛选项前后端值不对齐

审计后发现 connStatus 不是孤例，op_state、network_type 都有不同程度的映射问题（见 §2.4 审计表）。

### 1.3 3 个筛选项下拉是空的

[DeviceList/index.tsx:434-447](omcmb/webcode/src/pages/device/DeviceList/index.tsx) 里 `modelName` / `softwareVersion` / `firmwareVersion` 三个下拉 `options=[]`，仅留 TODO 注释。后端 `DeviceFilter` 也没对应字段，整条筛选链断开。

**目标**：一次合并改完，让 device list 页所有筛选项前后端语义闭环。

---

## 2. 现状盘点

### 2.1 当前 7 个 status 取值

| 取值 | 真实含义 | 写入路径 |
|------|---------|----------|
| `discovered` | 设计预留"首次发现"瞬态 | **❌ 全代码库无写入**（4 处都是只读） |
| `registered` | 已入库，未配置 | Admin POST /devices 手动建（device_service.go:1222） |
| `provisioning` | 自动开站中 | provision/engine.go 状态机内部 |
| `active` | 在线运行 | ACS Bootstrap Inform 自动写入 + 心跳恢复 |
| `maintenance` | 运维窗口期下线 | device_info_handler.go 管理 API |
| `offline` | 心跳超时 | heartbeat.go:103 自动写入 |
| `decommissioned` | 永久退役 | device_info_handler.go 管理 API |

### 2.2 多套口径并存的证据

**GeoStats handler**（device_handler.go:463）：
```
onlineActive   = active
onlineInactive = registered + provisioning
offline        = offline + maintenance + discovered + decommissioned
```

**前端 mapStatus()**（deviceApi.ts:144）：
```
online  = active + maintenance + discovered + registered + provisioning
offline = offline
```

**前端 connStatusMap**（筛选侧）：
```
connStatus='1'（在线） → 后端 status='active'   仅 active
connStatus='0'（离线） → 后端 status='offline'
```

→ **同一系统的同一字段、3 套口径互相矛盾**。

### 2.3 已经解过的另一半：激活状态

`device_info.first_online_time` 单调标记，设备首次 inform 写入永不变。`model.DeriveOpStateActivated()` 用它派生"已激活/未激活" 二值，与 status 独立。这个解法证明三维分离的方向是对的。

### 2.4 其它筛选项审计表

device list 页 9 个筛选项的前后端对齐情况：

| Filter | 前端 options 来源 | 字典 value | 前端→后端 query | 后端字段 | 后端实际数据 | 状态 |
|--------|-----------------|----------|----------------|---------|-------------|-----|
| **searchText** | 用户输入 | — | `?search=` | `Search *string` (BuildSearchOR) | 多字段 fuzzy LIKE | ✅ |
| **connStatus** | dict `conn_status` | `'1'/'0'` | `connStatusMap` 翻 → `?status=active/offline` | `Status *DeviceStatus` 单值 | 7 个 status 之一 | ❌ Q1 bug，本次拆 |
| **opState** | dict `op_state` | `'active'/'inactive'` | `?op_state=` 直传 | `OpState *string` | 实际存的是 `'1'/'0'`（DeriveOpStateActivated） | ❌ **dict value 与 DB 值不一致，filter 永远 0 结果** |
| **networkType** | dict `network_type` | `'eNB'/'gNB'` | `eNB→lte / gNB→nr` 翻译 → `?technology=` | `Technology *Technology` | DB 存 `lte/nr` | ⚠️ 通过指数翻译 work，但脆弱（dict label/value 与 db 不一致） |
| **productModel** | dict `product_type` | （未审） | `?product_class=` | `ProductClass *string` | DB 存 ProductClass | ⚠️ 字段名混乱（前端叫 productModel 后端叫 product_class）但 value 可能对齐 |
| **modelName** | **`[]` 写死** | — | （前端不发） | **无 ModelName 字段** | DB 有 `devices.model_name` | ❌ **完全断**：选项空 + 后端不接 |
| **softwareVersion** | **`[]` 写死** | — | （前端不发） | **无 SoftwareVersion 字段** | DB 有 `device_info.software_version` | ❌ 完全断 |
| **firmwareVersion** | **`[]` 写死** | — | （前端不发） | **无 FirmwareVersion 字段** | DB 有 `devices.firmware_version` | ❌ 完全断 |
| **groupId** | useDeviceGroups（实时） | UUID | `?group_id=` | `GroupID *uuid.UUID` | DB OK | ✅ |

### 2.5 现有字典 seed（[migrations/seed/000136](omcgo/migrations/seed/000136_seed_device_filter_dictionaries.sql)）

| dict.type | label/value | 与 DB 一致性 |
|-----------|-------------|------------|
| conn_status | 在线='1' 离线='0' | ❌ 一次跳两层 frontend mapping，无法直 filter |
| op_state | 激活='active' 未激活='inactive' | ❌ DB 实存 '1'/'0'，filter 永 0 命中 |
| network_type | eNB (LTE)='eNB' gNB (NR)='gNB' | ⚠️ DB 实存 lte/nr，靠前端额外翻译救场 |
| product_type | （由 seed/000004 注入） | 需补审 |

---

## 3. 目标态：三正交维度 + 字典全对齐

### 3.1 三正交维度

| 维度 | 字段 | 类型 | 取值 | 写入方 | 语义 |
|------|------|------|------|--------|------|
| **生命周期** | `devices.lifecycle_state` | VARCHAR(20) | `discovered` / `registered` / `provisioning` / `commissioned` / `maintenance` / `decommissioned` | Admin API、Provisioning Engine、ACS Bootstrap、运维操作 | 业务流程进度，由人/系统决策推进 |
| **实时在线** | `devices.is_online` | BOOLEAN | `true` / `false` | HeartbeatMonitor、ACS Inform 接收 | 当前心跳是否活跃，由系统自动维护 |
| **已激活**（已有，不动） | `device_info.first_online_time` | TIMESTAMPTZ | NULL / timestamp | ACS 首次 Inform | 单调标记，"是否曾上线过"，不可逆 |

三维度合法组合示意：

| lifecycle | is\_online | first\_online | 业务含义 |
|-----------|-----------|---------------|---------|
| commissioned | true | set | **业务最常态**：已入网、在线 |
| commissioned | false | set | 已入网但当前掉线（故障/网络/重启） |
| maintenance | true | set | 维护中但能远程连 |
| maintenance | false | set | 维护中且断电断网 |
| registered | true | NULL | Admin 建后设备首次连进 |
| registered | false | NULL | Admin 建了但设备从未连进 |
| decommissioned | * | set | 已退役 |

### 3.2 字典对齐总规则

**值域字典 type 与 DB 实际列值 1:1 对应**——`sys_dictionary_details.value` 必须就是后端 SQL `WHERE` 直接吃的值，前端不做翻译。

| dict.type | 新 value | 对齐到 DB 列 |
|-----------|---------|-------------|
| `lifecycle_state`（**新**） | 6 个：discovered/registered/provisioning/commissioned/maintenance/decommissioned | `devices.lifecycle_state` |
| `is_online`（**新**，二值） | `true`/`false` | `devices.is_online` |
| ~~`conn_status`~~ | **删除**（被 is_online 替代） | — |
| `op_state` | **改值**：activated='1', not_activated='0' | `device_info.op_state` 实存 '1'/'0' |
| `network_type` | **改值**：lte='lte', nr='nr'（label 仍叫 eNB/gNB） | `devices.technology` 实存 lte/nr |
| `product_type` | 维持，由 seed/000004 提供 | `devices.product_class` |
| `device_model`（**新**） | 由实际设备型号列表 seed | `devices.model_name` |
| `software_version`（**新**） | 由实际版本列表 seed | `device_info.software_version` |
| `firmware_version`（**新**） | 由实际固件列表 seed | `devices.firmware_version` |

### 3.3 命名决策（D1-D5 锁定）

- D1：**老 `devices.status` 列不保留**——同 migration 内 DROP（硬切，无过渡期）
- D2：**maintenance 不参与"在线统计"**（与 commissioned 平级，作独立维度）
- D3：**decommissioned 不可逆**（state machine 终态）
- D4：**加 `dict_lifecycle_state`** 字典 + 同步修 op_state/network_type/删 conn_status + 新增 device_model/software_version/firmware_version 3 个
- D5：**`ListResponse[T]` 加 `Stats` 字段**，顺手把 Q2 分析里的"前端类型骗自己"修了

---

## 4. 数据库迁移

### 4.1 Migration `000137_device_lifecycle_online_decouple.sql`

```sql
-- +goose Up
-- ============================================================
-- T-0162 设备列表前后端对齐总改 — Schema 部分
--
-- 把 devices.status 单字段承载的 7 状态拆解为两个正交字段：
--   lifecycle_state — 业务流程进度（不含 offline）
--   is_online      — 实时心跳活跃度
-- D1：硬切，同迁移内 DROP status 列（不留过渡期）。
--
-- 详细设计见
--   docs/design/device-lifecycle-online-status-decouple-20260520.md
-- ============================================================

-- ─── 1. 新增 lifecycle_state ─────────────────────────────────────
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS lifecycle_state VARCHAR(20) NOT NULL DEFAULT 'registered';

-- 从老 status 回填
UPDATE devices d
SET lifecycle_state = CASE
    WHEN d.status = 'active'  THEN 'commissioned'
    WHEN d.status = 'offline' THEN
        CASE WHEN di.first_online_time IS NOT NULL
             THEN 'commissioned'      -- 曾激活过的 offline → commissioned + is_online=false
             ELSE 'registered'        -- 从未激活的 offline → 视为还没入网
        END
    ELSE d.status                     -- discovered/registered/provisioning/maintenance/decommissioned 原样保留
END
FROM device_info di
WHERE di.device_id = d.id;

-- device_info 不存在的设备兜底（罕见）
UPDATE devices SET lifecycle_state = CASE
    WHEN status = 'active'  THEN 'commissioned'
    WHEN status = 'offline' THEN 'registered'
    ELSE status
END WHERE NOT EXISTS (SELECT 1 FROM device_info di WHERE di.device_id = devices.id);

ALTER TABLE devices
    ADD CONSTRAINT chk_devices_lifecycle_state
    CHECK (lifecycle_state IN (
        'discovered', 'registered', 'provisioning',
        'commissioned', 'maintenance', 'decommissioned'
    ));

CREATE INDEX IF NOT EXISTS idx_devices_lifecycle_state
    ON devices (lifecycle_state);

-- ─── 2. 新增 is_online ────────────────────────────────────────────
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS is_online BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE devices SET is_online = CASE
    WHEN status = 'active'  THEN TRUE
    WHEN status = 'offline' THEN FALSE
    -- 过渡态：last_inform_at 在心跳窗口内则视作在线
    WHEN last_inform_at IS NOT NULL
         AND last_inform_at > NOW() - INTERVAL '90 seconds' THEN TRUE
    ELSE FALSE
END;

-- 部分索引：大多数 dashboard / alert 查"在线"设备，部分索引省空间
CREATE INDEX IF NOT EXISTS idx_devices_is_online
    ON devices (is_online) WHERE is_online = TRUE;

-- ─── 3. D1：硬切，DROP 老 status 列 ────────────────────────────────
DROP INDEX IF EXISTS idx_devices_carrier_status;
DROP INDEX IF EXISTS idx_devices_status;
ALTER TABLE devices DROP COLUMN status;

-- 重建按新维度的复合索引（替代被删的 idx_devices_carrier_status）
CREATE INDEX IF NOT EXISTS idx_devices_carrier_lifecycle
    ON devices (carrier, lifecycle_state);

COMMENT ON COLUMN devices.lifecycle_state IS
    'T-0162: 设备业务生命周期，与在线状态解耦。取值见 chk_devices_lifecycle_state。';
COMMENT ON COLUMN devices.is_online IS
    'T-0162: 实时在线状态，由 HeartbeatMonitor 维护。true=最近一次 inform 在心跳窗口内。';

-- +goose Down
-- 反向：恢复 status 列 + 删两个新列
ALTER TABLE devices
    ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active';

UPDATE devices SET status = CASE
    WHEN lifecycle_state = 'commissioned' AND is_online = TRUE  THEN 'active'
    WHEN lifecycle_state = 'commissioned' AND is_online = FALSE THEN 'offline'
    WHEN lifecycle_state = 'maintenance'                        THEN 'maintenance'
    WHEN lifecycle_state = 'decommissioned'                     THEN 'decommissioned'
    ELSE lifecycle_state
END;

CREATE INDEX idx_devices_carrier_status ON devices (carrier, status);
CREATE INDEX idx_devices_status ON devices (status);

DROP INDEX IF EXISTS idx_devices_carrier_lifecycle;
DROP INDEX IF EXISTS idx_devices_is_online;
DROP INDEX IF EXISTS idx_devices_lifecycle_state;
ALTER TABLE devices DROP CONSTRAINT IF EXISTS chk_devices_lifecycle_state;
ALTER TABLE devices
    DROP COLUMN IF EXISTS lifecycle_state,
    DROP COLUMN IF EXISTS is_online;
```

### 4.2 Migration `000138_seed_device_filter_dict_v2.sql`

把 D4 的字典调整一次性做到位（**老 conn_status 字典删除 + op_state/network_type 改值 + 新增 4 个字典**）：

```sql
-- +goose Up
-- ============================================================
-- T-0162 设备列表筛选字典 v2：与 DB 列值 1:1 对齐
-- ============================================================

-- 1. 删旧 conn_status 字典（被 is_online 替代，前端读 dict_is_online）
DELETE FROM sys_dictionary_details WHERE sys_dictionary_id IN
    (SELECT id FROM sys_dictionaries WHERE type = 'conn_status');
DELETE FROM sys_dictionaries WHERE type = 'conn_status';

-- 2. 修 op_state value 为 '1'/'0'（与 device_info.op_state 实存对齐）
UPDATE sys_dictionary_details
SET value = '1' WHERE value = 'active'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');
UPDATE sys_dictionary_details
SET value = '0' WHERE value = 'inactive'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');

-- 3. 修 network_type value 为 lte/nr（与 devices.technology 实存对齐）
UPDATE sys_dictionary_details
SET value = 'lte' WHERE value = 'eNB'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');
UPDATE sys_dictionary_details
SET value = 'nr' WHERE value = 'gNB'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');

-- 4. 新增 4 个字典：lifecycle_state / is_online / device_model / software_version / firmware_version
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
    ('设备生命周期', 'lifecycle_state',  TRUE, 'T-0162: 业务流程进度，6 状态'),
    ('设备在线状态', 'is_online',        TRUE, 'T-0162: 实时心跳活跃，true=在线 false=离线'),
    ('设备型号',     'device_model',     TRUE, 'T-0162: 设备硬件型号，对齐 devices.model_name'),
    ('软件版本',     'software_version', TRUE, 'T-0162: 软件版本号，对齐 device_info.software_version'),
    ('固件版本',     'firmware_version', TRUE, 'T-0162: 固件版本号，对齐 devices.firmware_version')
ON CONFLICT DO NOTHING;

-- 5. lifecycle_state 6 项（label 中文 / value 英文，与 DB 列一致）
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT v.label, v.value, v.sort, d.id
FROM sys_dictionaries d, (VALUES
    ('已发现',     'discovered',     1),
    ('已注册',     'registered',     2),
    ('配置中',     'provisioning',   3),
    ('已入网',     'commissioned',   4),
    ('维护中',     'maintenance',    5),
    ('已退役',     'decommissioned', 6)
) AS v(label, value, sort)
WHERE d.type = 'lifecycle_state'
ON CONFLICT DO NOTHING;

-- 6. is_online 2 项
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT v.label, v.value, v.sort, d.id
FROM sys_dictionaries d, (VALUES
    ('在线', 'true',  1),
    ('离线', 'false', 2)
) AS v(label, value, sort)
WHERE d.type = 'is_online'
ON CONFLICT DO NOTHING;

-- 7. device_model / software_version / firmware_version 初始化：
--    把 devices / device_info 表里实际存在的 distinct 值灌进字典，
--    保证"用户在前端选了一个 model，后端 filter 一定能命中至少一条"
--    新设备 inform 后管理员手动补字典词条（或后续做自动同步任务）。
WITH d AS (
    SELECT DISTINCT NULLIF(TRIM(model_name), '') AS v
    FROM devices
    WHERE model_name IS NOT NULL AND TRIM(model_name) <> ''
)
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT d.v, d.v,
       ROW_NUMBER() OVER (ORDER BY d.v),
       sd.id
FROM d, sys_dictionaries sd
WHERE sd.type = 'device_model'
ON CONFLICT DO NOTHING;

WITH s AS (
    SELECT DISTINCT NULLIF(TRIM(software_version), '') AS v
    FROM device_info
    WHERE software_version IS NOT NULL AND TRIM(software_version) <> ''
)
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT s.v, s.v, ROW_NUMBER() OVER (ORDER BY s.v), sd.id
FROM s, sys_dictionaries sd WHERE sd.type = 'software_version'
ON CONFLICT DO NOTHING;

WITH f AS (
    SELECT DISTINCT NULLIF(TRIM(firmware_version), '') AS v
    FROM devices
    WHERE firmware_version IS NOT NULL AND TRIM(firmware_version) <> ''
)
INSERT INTO sys_dictionary_details (label, value, sort, sys_dictionary_id)
SELECT f.v, f.v, ROW_NUMBER() OVER (ORDER BY f.v), sd.id
FROM f, sys_dictionaries sd WHERE sd.type = 'firmware_version'
ON CONFLICT DO NOTHING;

-- +goose Down
-- 反向：删 5 个新字典 + 还原 op_state/network_type 旧值 + 重建 conn_status
DELETE FROM sys_dictionary_details WHERE sys_dictionary_id IN
    (SELECT id FROM sys_dictionaries WHERE type IN
        ('lifecycle_state','is_online','device_model','software_version','firmware_version'));
DELETE FROM sys_dictionaries WHERE type IN
    ('lifecycle_state','is_online','device_model','software_version','firmware_version');

UPDATE sys_dictionary_details
SET value = 'active' WHERE value = '1'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');
UPDATE sys_dictionary_details
SET value = 'inactive' WHERE value = '0'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'op_state');

UPDATE sys_dictionary_details
SET value = 'eNB' WHERE value = 'lte'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');
UPDATE sys_dictionary_details
SET value = 'gNB' WHERE value = 'nr'
  AND sys_dictionary_id IN (SELECT id FROM sys_dictionaries WHERE type = 'network_type');

-- 重建 conn_status 字典（兼容 P1 之前的代码）
INSERT INTO sys_dictionaries (name, type, status, description) VALUES
    ('设备在线状态(旧)', 'conn_status', TRUE, '已废弃，用 is_online 替代')
ON CONFLICT DO NOTHING;
```

---

## 5. 后端模型与状态机

### 5.1 新类型

```go
// global/consts.go

// DeviceLifecycle represents the business lifecycle stage of a device.
type DeviceLifecycle string

const (
    LifecycleDiscovered     DeviceLifecycle = "discovered"
    LifecycleRegistered     DeviceLifecycle = "registered"
    LifecycleProvisioning   DeviceLifecycle = "provisioning"
    LifecycleCommissioned   DeviceLifecycle = "commissioned"
    LifecycleMaintenance    DeviceLifecycle = "maintenance"
    LifecycleDecommissioned DeviceLifecycle = "decommissioned"
)

// 注意：DeviceStatus 类型整体删除（D1 硬切），所有 model.DeviceXxx 引用改名为
// model.LifecycleXxx。is_online 直接用 bool 字段，不引入新类型。
```

### 5.2 状态机

```go
// device/state_machine.go

var validLifecycleTransitions = map[model.DeviceLifecycle][]model.DeviceLifecycle{
    model.LifecycleDiscovered:    {model.LifecycleRegistered, model.LifecycleCommissioned},
    model.LifecycleRegistered:    {model.LifecycleProvisioning, model.LifecycleCommissioned, model.LifecycleDecommissioned},
    model.LifecycleProvisioning:  {model.LifecycleCommissioned, model.LifecycleRegistered, model.LifecycleDecommissioned},
    model.LifecycleCommissioned:  {model.LifecycleMaintenance, model.LifecycleDecommissioned},
    model.LifecycleMaintenance:   {model.LifecycleCommissioned, model.LifecycleDecommissioned},
    // decommissioned 终态，无转出（D3）
}

func ValidateLifecycleTransition(current, target model.DeviceLifecycle) error { ... }
```

### 5.3 Device 结构

```go
type Device struct {
    // ...其他字段不变...

    LifecycleState model.DeviceLifecycle `json:"lifecycle_state" db:"lifecycle_state"`
    IsOnline       bool                  `json:"is_online" db:"is_online"`

    // Status 字段彻底删除（D1 硬切，DB 列已 DROP）
}
```

---

## 6. 后端业务逻辑改造

### 6.1 ACS Inform 处理（顺手补 discovered 写入）

**改前**（device_service.go:449）：
```go
Status: model.DeviceActive,
```

**改后**：
```go
// 首次 Inform：新设备
if isNewDevice {
    LifecycleState: model.LifecycleDiscovered,  // 设上瞬态
    IsOnline:       true,
    LastInformAt:   now,
}
// 后续 Inform：维护连接活跃
existing.IsOnline = true
existing.LastInformAt = now
// lifecycle_state 由 provisioning engine / 人工运维推进，inform 不动

// 异步：publish DeviceDiscovered event → provisioning engine 决定下一步
```

### 6.2 HeartbeatMonitor

```go
// 改前：
deviceRepo.UpdateStatus(ctx, device.ID, model.DeviceOffline)

// 改后：
deviceRepo.UpdateOnlineStatus(ctx, device.ID, false)  // 只写 is_online
// lifecycle_state 完全不动
```

### 6.3 维护操作

```go
// 改前：
service.TransitionStatus(ctx, id, model.DeviceMaintenance)

// 改后：
service.TransitionLifecycle(ctx, id, model.LifecycleMaintenance)
// is_online 完全不动
```

### 6.4 List filter & Stats（同时修对齐 bug + 新增 3 字段）

```go
// device_handler.go ListDevices 新增解析
//   ?lifecycle_state=commissioned,maintenance  (CSV 多选)
//   ?is_online=true                            (bool)
//   ?op_state=1|0                              (与 DB 实存对齐)
//   ?technology=lte|nr                         (前端不再翻译，dict value 直传)
//   ?model_name=BS-FAP400                      (新)
//   ?software_version=v1.2.3                   (新)
//   ?firmware_version=fw-12345                 (新)
//   ?status=...                                (D1 后删除，404 拒绝)

// device_repository.go DeviceFilter 增字段
type DeviceFilter struct {
    // ...
    LifecycleState []model.DeviceLifecycle  // 多选
    IsOnline       *bool

    // 新增 3 字段
    ModelName        *string
    SoftwareVersion  *string  // 走 device_info JOIN
    FirmwareVersion  *string

    // Status 字段删除
}

// SQL WHERE 加：
// .Where(sq.Eq{"d.lifecycle_state": filter.LifecycleState})
// .Where(sq.Eq{"d.is_online": *filter.IsOnline})
// .Where(sq.Eq{"d.model_name": *filter.ModelName})
// .Where(sq.Eq{"di.software_version": *filter.SoftwareVersion})
// .Where(sq.Eq{"d.firmware_version": *filter.FirmwareVersion})
```

### 6.5 Stats 真正返回 + 双维度

```go
type DeviceListStats struct {
    Total       int                                  `json:"total"`
    ByLifecycle map[model.DeviceLifecycle]int        `json:"by_lifecycle"` // {commissioned:50, maintenance:3, ...}
    OnlineCount int                                  `json:"online_count"`
    Alarmed     int                                  `json:"alarmed"`
}

// ListResponse[T] 增 Stats 字段（D5）
type ListResponse[T any] struct {
    Items, Total, Page, PageSize, TotalPages ...
    Stats *DeviceListStats `json:"stats,omitempty"`
}

// repository 主 SQL 之后跑一个 group-by SQL 填 Stats
```

---

## 7. 前端类型与 UI 改造

### 7.1 Device 类型拆

```typescript
// frontend-core/src/types/device.ts
type DeviceLifecycle =
  | 'discovered' | 'registered' | 'provisioning'
  | 'commissioned' | 'maintenance' | 'decommissioned';

interface Device {
  lifecycleState: DeviceLifecycle;
  isOnline: boolean;
  opState?: '1' | '0';  // 派生自 first_online_time，不变

  // connStatus 字段彻底删除（D1 硬切）
  // mapStatus 函数彻底删除（不再做"乐观归类"骗术）
}
```

### 7.2 List 页 UI

| 旧筛选 | 新筛选 | 字典 | 后端 query |
|--------|--------|------|-----------|
| ~~connStatus~~ | **新：lifecycleState（多选）** | `lifecycle_state` | `?lifecycle_state=` |
| ~~connStatus~~ | **新：isOnline（单选）** | `is_online` | `?is_online=` |
| opState | opState | `op_state`（已对齐 '1'/'0'） | `?op_state=` |
| networkType | networkType | `network_type`（已对齐 lte/nr） | `?technology=` |
| productModel | productModel | `product_type` | `?product_class=` |
| **modelName** | **modelName**（启用） | `device_model`（**新**） | `?model_name=` |
| **softwareVersion** | **softwareVersion**（启用） | `software_version`（**新**） | `?software_version=` |
| **firmwareVersion** | **firmwareVersion**（启用） | `firmware_version`（**新**） | `?firmware_version=` |
| groupId | groupId | useDeviceGroups | `?group_id=` |

筛选区从 9 项变为 10 项（connStatus 1 项拆为 lifecycleState + isOnline 2 项）。前端不再做任何 frontend→backend 值映射，dict.value 直传后端。

### 7.3 Stats 区直读

```typescript
// 旧：fallback 用 items.filter() 算 → 全是骗局
const stats = data?.stats ?? { ... fallback };

// 新：后端 stats 必填，前端直读
const stats = data!.stats;  // 后端保证返回
// statsItems: 显示 Total / Online (online_count) / Offline (total - online_count) /
//             Commissioned / Alarmed
```

### 7.4 字典页面（system/data-dictionary）

无需 UI 改动——新增 3 个 dict 自动在该页可见，管理员可补/改字典值。

后续 enhancement（不在本期）：新设备 inform 时自动 upsert 字典词条，避免管理员手动维护。

---

## 8. 分阶段实施计划

| 阶段 | Scope | 估时 | 文件数 |
|------|-------|------|-------|
| **P1 — DB migration** | 000137 schema + 000138 dict seed | 0.5d | 2 sql |
| **P2 — Backend model + SM** | global/consts.go 新类型；state_machine 重写；Device 结构改字段；errors 增 LifecycleTransitionInvalid | 0.5d | ~5 .go |
| **P3 — Backend service/handler/repo** | ACS Inform / Heartbeat / Maintenance / List filter（含新 3 字段）/ Stats 双维返回；删 mapStatus 旧 status query 兼容代码（D1） | 1.5d | ~12 .go |
| **P4 — Frontend** | Device 类型拆；mapStatus 删；列表 UI 双下拉双列；apply dict.value 直传无翻译；stats 直读 | 1d | ~8 .ts/.tsx |
| **P5 — e2e** | scripts/e2e_verify.sh 新筛选用例；浏览器手测 Q1 bug 消失 | 0.5d | 1 .sh + 文档 |

**总计**：4 天，5 个独立 commit。

---

## 9. 风险与回滚

| 风险 | 影响 | 缓解 |
|------|------|------|
| Migration backfill 错误归类老 offline 设备 | 历史 commissioned 误标 registered | backfill 用 first_online_time 做兜底；migration down 段可还原 |
| ACS 流量在 P3 部署中断 | inform 写入失败 | P3 部署用滚动重启，否则部分实例先于 DB 完成迁移；建议 P3 部署前先验 P1 migration 通过 |
| 老 device list 客户端代码访问 `status` 字段 | undefined 错误 | 全栈一起部署，前端 P4 同期上 |
| 字典 device_model / software / firmware 数据空 | 下拉空选项 | seed 用 distinct from devices 灌入，至少初始化时与真实数据匹配 |
| 用户后续手动改字典误删词条 | 设备 model 显示空 / filter 不命中 | 字典页面已有审计；不在本期改造范围 |

每阶段 commit 独立可 revert。最危险的 P3（service 大改）做完后 5 分钟内冒烟，5 分钟没出大问题就视为稳定。

---

## 10. DoD（验收）

| 阶段 | DoD |
|------|-----|
| P1 | `goose up` + `goose down` 都通过；DB 检查：所有 `status='active'` → `lifecycle_state='commissioned'` + `is_online=true`；老 status 列已 DROP（`\d devices` 无 status） |
| P2 | `go build ./... && go test -race ./...` PASS；新状态机单测覆盖所有合法/非法转移（含 Decommissioned 终态拒绝） |
| P3 | e2e_verify.sh PASS；list filter 新 query 参数全部生效（`?lifecycle_state=...&is_online=...&model_name=...&software_version=...&firmware_version=...`）；ListResponse 真带 stats |
| P4 | 浏览器手测 device list 页：(a) Q1 bug 消失——无筛选 online 数与筛选后 list 长度一致；(b) 3 个空下拉填上字典选项；(c) 选中后筛选有命中；(d) stats 区显示后端真实数据；(e) 旧 `connStatus` / `mapStatus` 代码无残留 |
| P5 | scripts/e2e_verify.sh 加 5 个新筛选用例；通过 |

---

## 11. 待审批清单

请审完后回复确认或修改意见，我据此开 P1：

- [ ] §3 三正交维度方案 + §3.2 字典对齐总规则
- [ ] §4.1 schema migration（含 backfill + 硬切 DROP status）
- [ ] §4.2 dict seed migration（删 conn_status + 改 op_state/network_type + 新增 5 个字典）
- [ ] §6 后端业务逻辑改造（含新 3 筛选字段）
- [ ] §7 前端 UI 双下拉 + 字典 value 直传后端
- [ ] §8 5 阶段独立 commit 节奏
