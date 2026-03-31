# 0023 TR069 设备管理系统 — 综合设计方案

> 基于 `files/Back-end/` 目录下全部设备相关需求文档、前端设计文档及当前代码库的综合分析，
> 梳理符合 TR069 标准的设备管理功能体系，覆盖设备注册、设备组管理、设备列表、设备操作、
> 数据权限管控等完整能力。

---

## 1. 功能全景

### 1.1 功能清单

| 编号 | 功能 | 子功能 | 状态 | 关联模块 | 实现说明 |
|------|------|--------|------|---------|---------|
| **DM-01** | **设备注册与发现** | | | | |
| DM-01.1 | TR069 Inform 自动注册 | Bootstrap/Boot 自动入库 | ✅ 已实现 | device/ | `service.go:RegisterFromInform` |
| DM-01.2 | 手动注册（SN 录入） | 单个/批量 SN 录入预注册 | ❌ 未实现 | device/ | 缺 registration 子模块 |
| DM-01.3 | Excel 批量导入注册 | 模板下载 + 批量导入 + 校验反馈 | ❌ 未实现 | device/ | 缺 registration 子模块 |
| DM-01.4 | 注册时自动归组 | 新设备归入默认二级设备组 | ❌ 未实现 | device/ + topology/ | 缺默认组种子+归组逻辑 |
| **DM-02** | **设备组管理** | | | | |
| DM-02.1 | 两级树形结构 | 一级组/二级组，设备只归二级组 | ❌ 未实现 | topology/ | model 缺 level，DB 缺约束 |
| DM-02.2 | 默认设备组 | 系统内置不可删改的默认一级/二级组 | ❌ 未实现 | topology/ | 缺 000065 迁移种子数据 |
| DM-02.3 | 设备组 CRUD | 创建/修改/删除，含子组批量创建 | 🔶 部分实现 | topology/ | 基础 CRUD 有，缺子组批量+默认组保护 |
| DM-02.4 | 删除联动 | 删除组时设备自动归入默认组 | ❌ 未实现 | topology/ | 缺 MoveDevicesToDefaultGroup |
| DM-02.5 | 设备归属管理 | 批量添加/移除/移动设备 | 🔶 仅单个 | topology/ | 仅 AddDevice/RemoveDevice 单个 |
| DM-02.6 | 设备计数 | 树节点显示设备数量 | ❌ 未实现 | topology/ | 缺 LEFT JOIN COUNT |
| DM-02.7 | 分组统计 | 总分组/已分组/未分组统计 | ❌ 未实现 | topology/ | 缺 GetStats 方法 |
| **DM-03** | **设备列表与查询** | | | | |
| DM-03.1 | 分页列表查询 | 多条件过滤 + 分页 + 排序 | ✅ 已实现 | device/ | `ListDevicesWithInfo` |
| DM-03.2 | 模糊搜索 | SN/名称/IP/版本 联合模糊匹配 | ✅ 已实现 | device/ | DeviceFilter.Search ILIKE |
| DM-03.3 | 精准过滤 | 运营商/制式/状态/射频/小区等 | ✅ 已实现 | device/ | DeviceFilter 多字段等值 |
| DM-03.4 | 按组过滤 | 点击设备组查看组内设备 | ❌ 未实现 | device/ | DeviceFilter 缺 GroupID |
| DM-03.5 | 列自定义 | 用户自定义显示列 | ❌ 未实现 | device/ | 缺 user_column_configs 表 |
| DM-03.6 | 数据导出 | CSV/Excel 导出（同步/异步） | ❌ 未实现 | device/ | 缺 export 子模块 |
| DM-03.7 | 枚举值查询 | 状态/厂商等枚举列表 | ✅ 已实现 | device/ | `handler.go:ListEnums` |
| **DM-04** | **设备详情与操作** | | | | |
| DM-04.1 | 设备详情聚合 | 基础信息 + 参数 + 告警 + KPI | ✅ 已实现 | device/ | `GetDeviceDetail` |
| DM-04.2 | 设备重启 | Reboot RPC 命令 | ✅ 已实现 | device/ | cmdQueue Reboot |
| DM-04.3 | 参数同步 | 从设备读取最新参数 | ❌ 未实现 | paramsync/ | 缺 paramsync 模块 |
| DM-04.4 | 参数配置下发 | SetParameterValues | ✅ 已实现 | device/ | cmdQueue SetParameterValues |
| DM-04.5 | 射频开关 | RF Enable/Disable | ❌ 未实现 | device/ | 缺 RF 控制端点 |
| DM-04.6 | 激活/去激活 | 设备状态变更 | ✅ 已实现 | device/ | `Activate/DeactivateDevice` |
| **DM-05** | **固件升级** | | | | |
| DM-05.1 | 固件文件管理 | 上传/查询/删除固件包 | 🔶 部分实现 | software/ | 固件 CRUD 有，缺版本对比 |
| DM-05.2 | 升级任务管理 | 创建/暂停/恢复/查询升级任务 | ❌ 未实现 | software/ | 有 model，缺 handler 端点 |
| DM-05.3 | 批量升级 | 按设备组/条件批量升级 | ❌ 未实现 | software/ | 有 BatchUpgradeRequest，缺实现 |
| DM-05.4 | 版本回退 | 回退到上一版本 | ❌ 未实现 | software/ | 缺回退流程 |
| **DM-06** | **日志采集** | | | | |
| DM-06.1 | 立即日志收集 | Upload RPC 触发即时收集 | ❌ 未实现 | syslog/ | 仅日志列表，缺收集触发 |
| DM-06.2 | 周期日志收集 | SetParameterValues 配置周期收集 | ❌ 未实现 | syslog/ | 缺 cron 调度 |
| DM-06.3 | 异常重启日志 | Inform BOOT 事件触发自动收集 | ❌ 未实现 | syslog/ | 缺 BOOT 事件钩子 |
| **DM-07** | **数据权限** | | | | |
| DM-07.1 | 角色-设备组关联 | 角色绑定可见的设备组 | ❌ 未实现 | admin/ + topology/ | 缺 role_device_groups 表 |
| DM-07.2 | 设备查询权限过滤 | 用户只看到权限范围内的设备 | ❌ 未实现 | device/ | 缺 VisibleGroups 过滤 |
| DM-07.3 | 设备组树权限过滤 | 用户只看到权限范围内的设备组 | ❌ 未实现 | topology/ | 缺权限过滤树逻辑 |
| DM-07.4 | 权限缓存 | Redis 缓存用户可见设备组 | ❌ 未实现 | admin/ | 缺 PermissionService |

### 1.2 优先级规划

| 阶段 | 功能编号 | 说明 |
|------|---------|------|
| **P1：设备组基础** | DM-02.1~02.7 | 两级树、默认组、设备归属、批量操作 |
| **P2：数据权限** | DM-07.1~07.4 | 角色关联设备组、查询过滤、缓存 |
| **P3：设备注册增强** | DM-01.2~01.4 | 手动注册、Excel 导入、自动归组 |
| **P4：设备列表增强** | DM-03.4~03.6 | 按组过滤、列自定义、数据导出 |
| **P5：设备操作扩展** | DM-04.3, 04.5 | 参数同步、射频开关 |
| **P6：固件升级** | DM-05.1~05.4 | 完整升级回退流程 |
| **P7：日志采集** | DM-06.1~06.3 | 立即/周期/异常重启日志 |

---

## 2. 数据库详细设计

### 2.1 数据库表全景

```
┌─────────────────────────────────────────────────────────────────┐
│                          用户与权限                               │
│  users ──N:M── roles ──1:N── permissions                        │
│                  │                                               │
│                  └──N:M── role_device_groups (NEW)               │
│                              │                                   │
│                              ▼                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                      设备组管理                             │  │
│  │  device_groups (ENHANCED) ◄──1:N── device_group_members   │  │
│  │     │ parent_id (self-ref)              │                  │  │
│  │     │ level: 1=一级, 2=二级              │ device_id ───┐  │  │
│  │     │ is_default: 默认组保护              │              │  │  │
│  └─────┼────────────────────────────────────┼──────────────┘  │
│         │                                    │                  │
│         ▼                                    ▼                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                      设备核心                               │  │
│  │  devices (PARTITIONED by carrier)                          │  │
│  │     │                                                      │  │
│  │     ├──1:1── device_info (运维扩展)                        │  │
│  │     ├──1:N── device_parameters (HASH PARTITIONED × 32)    │  │
│  │     ├──1:N── device_tasks (命令队列)                       │  │
│  │     └──1:N── alarms / pm_counters / ...                   │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 已有表（无需改动）

| 表名 | 迁移编号 | 说明 |
|------|---------|------|
| `devices` | 000001 | 设备主表，按 carrier 分区（cmcc/ctcc/cucc） |
| `device_parameters` | 000002 + 000064 | 设备参数 EAV 表，已完成 Hash 32 分区 |
| `device_info` | 000062 + 000063 | 设备运维扩展信息（双表架构） |
| `device_group_members` | 000009 | 设备-组关联表（需新增约束，见 2.3） |
| `users` | 000014 | 用户表 |
| `roles` | 000014 | 角色表 |
| `user_roles` | 000014 | 用户-角色关联 |
| `permissions` | 000014 | 角色-资源权限 |

### 2.3 迁移 000065：设备组增强

**文件**：`migrations/000065_enhance_device_groups.up.sql`

对现有 `device_groups` 表追加字段和约束，建立两级树形结构：

```sql
-- ============================================================
-- 000065_enhance_device_groups.up.sql
-- 设备组管理增强：两级树约束 + 默认组 + 审计字段
-- ============================================================

-- 1. 追加字段
ALTER TABLE device_groups ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active';
ALTER TABLE device_groups ADD COLUMN remark TEXT;
ALTER TABLE device_groups ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE device_groups ADD COLUMN level SMALLINT NOT NULL DEFAULT 1;
ALTER TABLE device_groups ADD COLUMN created_by VARCHAR(64);
ALTER TABLE device_groups ADD COLUMN updated_by VARCHAR(64);

-- 2. 层级约束：只允许 1 和 2
ALTER TABLE device_groups
    ADD CONSTRAINT chk_dg_level CHECK (level IN (1, 2));

-- 3. 层级与父子关系一致性：一级无父，二级必有父
ALTER TABLE device_groups
    ADD CONSTRAINT chk_dg_level_parent
    CHECK ((level = 1 AND parent_id IS NULL) OR (level = 2 AND parent_id IS NOT NULL));

-- 4. 同一父节点下名称唯一（一级组用 COALESCE 处理 NULL parent_id）
CREATE UNIQUE INDEX idx_dg_name_parent
    ON device_groups (name, COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid));

-- 5. 设备只能归属一个组（新增唯一约束）
ALTER TABLE device_group_members
    ADD CONSTRAINT uq_dgm_device UNIQUE (device_id);

-- 6. 种子数据：默认设备组（固定 UUID，代码常量引用）
INSERT INTO device_groups (id, name, parent_id, level, is_default, status, remark, created_by)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Default Level Group',
    NULL,
    1,
    TRUE,
    'active',
    '系统默认一级设备组，不可修改删除',
    'system'
);

INSERT INTO device_groups (id, name, parent_id, level, is_default, status, remark, created_by)
VALUES (
    '00000000-0000-0000-0000-000000000002',
    'Default Level Group',
    '00000000-0000-0000-0000-000000000001',
    2,
    TRUE,
    'active',
    '系统默认二级设备组，删除组后设备自动归入此组',
    'system'
);

-- 7. 将现有未归组的设备自动加入默认二级组
INSERT INTO device_group_members (group_id, device_id, added_at)
SELECT '00000000-0000-0000-0000-000000000002'::uuid, d.id, NOW()
FROM devices d
WHERE NOT EXISTS (
    SELECT 1 FROM device_group_members dgm WHERE dgm.device_id = d.id
)
ON CONFLICT (device_id) DO NOTHING;
```

**回滚**：`migrations/000065_enhance_device_groups.down.sql`

```sql
-- 移除种子数据（先清理成员关系）
DELETE FROM device_group_members
WHERE group_id IN (
    '00000000-0000-0000-0000-000000000001'::uuid,
    '00000000-0000-0000-0000-000000000002'::uuid
);
DELETE FROM device_groups WHERE is_default = TRUE;

-- 移除约束和字段
ALTER TABLE device_group_members DROP CONSTRAINT IF EXISTS uq_dgm_device;
DROP INDEX IF EXISTS idx_dg_name_parent;
ALTER TABLE device_groups DROP CONSTRAINT IF EXISTS chk_dg_level_parent;
ALTER TABLE device_groups DROP CONSTRAINT IF EXISTS chk_dg_level;
ALTER TABLE device_groups DROP COLUMN IF EXISTS updated_by;
ALTER TABLE device_groups DROP COLUMN IF EXISTS created_by;
ALTER TABLE device_groups DROP COLUMN IF EXISTS level;
ALTER TABLE device_groups DROP COLUMN IF EXISTS is_default;
ALTER TABLE device_groups DROP COLUMN IF EXISTS remark;
ALTER TABLE device_groups DROP COLUMN IF EXISTS status;
```

**增强后 device_groups 完整字段：**

| 字段 | 类型 | 默认值 | 说明 | 来源 |
|------|------|--------|------|------|
| id | UUID PK | gen_random_uuid() | 主键 | 000008 |
| name | VARCHAR(128) | — | 设备组名称 | 000008 |
| parent_id | UUID FK→self | NULL | 父组ID，NULL=一级组 | 000008 |
| carrier | VARCHAR(4) | NULL | 运营商筛选（可选） | 000008 |
| description | TEXT | NULL | 描述 | 000008 |
| sort_order | INTEGER | 0 | 排序 | 000008 |
| **status** | VARCHAR(16) | 'active' | active / disabled | **000065** |
| **remark** | TEXT | NULL | 备注 | **000065** |
| **is_default** | BOOLEAN | FALSE | 默认组标记 | **000065** |
| **level** | SMALLINT | 1 | 1=一级, 2=二级 | **000065** |
| **created_by** | VARCHAR(64) | NULL | 创建人用户名 | **000065** |
| **updated_by** | VARCHAR(64) | NULL | 更新人用户名 | **000065** |
| created_at | TIMESTAMPTZ | NOW() | 创建时间 | 000008 |
| updated_at | TIMESTAMPTZ | NOW() | 更新时间（trigger） | 000008 |

**增强后 device_group_members 约束：**

| 约束 | 说明 | 来源 |
|------|------|------|
| PRIMARY KEY (group_id, device_id) | 组-设备联合主键 | 000009 |
| FK group_id → device_groups(id) CASCADE | 删组自动清理 | 000009 |
| idx_dgm_device (device_id) | 反向查找索引 | 000009 |
| idx_dgm_group (group_id) | 正向查找索引 | 000009 |
| **uq_dgm_device UNIQUE (device_id)** | **设备唯一归属** | **000065** |

### 2.4 迁移 000066：角色数据权限表

**文件**：`migrations/000066_create_role_device_groups.up.sql`

```sql
-- ============================================================
-- 000066_create_role_device_groups.up.sql
-- 角色-设备组数据权限关联表
-- ============================================================

CREATE TABLE role_device_groups (
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    group_id   UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, group_id)
);

CREATE INDEX idx_rdg_role ON role_device_groups (role_id);
CREATE INDEX idx_rdg_group ON role_device_groups (group_id);

COMMENT ON TABLE role_device_groups IS
    '角色数据权限：角色可见的设备组。关联一级组=可见其下所有二级组的设备';
```

**回滚**：`migrations/000066_create_role_device_groups.down.sql`

```sql
DROP TABLE IF EXISTS role_device_groups;
```

**角色数据权限表字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| role_id | UUID FK→roles(id) | 角色ID |
| group_id | UUID FK→device_groups(id) | 设备组ID（一级或二级） |
| PRIMARY KEY (role_id, group_id) | 联合主键 | 防重复 |

### 2.5 迁移 000067：设备注册表（预注册）

**文件**：`migrations/000067_create_device_registrations.up.sql`

支持手动注册和 Excel 批量导入的设备预注册记录：

```sql
-- ============================================================
-- 000067_create_device_registrations.up.sql
-- 设备预注册表：手动/批量导入注册的设备 SN 及附属信息
-- ============================================================

CREATE TABLE device_registrations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    serial_number   VARCHAR(64) NOT NULL,
    group_id        UUID NOT NULL REFERENCES device_groups(id),
    device_id       UUID,                    -- 设备实际上线后关联
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
                    -- pending: 已注册待上线
                    -- online: 设备已上线关联
                    -- expired: 超时未上线

    -- 导入时可携带的规划信息
    site_name       VARCHAR(128),
    device_name     VARCHAR(128),
    longitude       DOUBLE PRECISION,
    latitude        DOUBLE PRECISION,
    height          DECIMAL(10,2),
    azimuth         SMALLINT,                -- 水平方位角 [0, 359]
    tilt_angle      SMALLINT,                -- 机械下倾角 [0, 9]
    beam_width      SMALLINT,                -- 垂直3dB波束宽度 [1, 9]
    remark          TEXT,

    -- 审计
    created_by      VARCHAR(64),
    import_batch_id UUID,                    -- Excel 批量导入的批次ID
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_dr_sn ON device_registrations (serial_number)
    WHERE status = 'pending';  -- 只对待上线的 SN 做唯一约束
CREATE INDEX idx_dr_group ON device_registrations (group_id);
CREATE INDEX idx_dr_status ON device_registrations (status);
CREATE INDEX idx_dr_batch ON device_registrations (import_batch_id)
    WHERE import_batch_id IS NOT NULL;

CREATE TRIGGER trigger_dr_updated_at
    BEFORE UPDATE ON device_registrations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.6 迁移 000068：用户列配置表

**文件**：`migrations/000068_create_user_column_configs.up.sql`

```sql
-- ============================================================
-- 000068_create_user_column_configs.up.sql
-- 用户自定义列配置
-- ============================================================

CREATE TABLE user_column_configs (
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    page_key     VARCHAR(64) NOT NULL,     -- 页面标识：device_list, alarm_list, ...
    columns      JSONB NOT NULL,           -- 列配置 JSON 数组
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, page_key)
);

CREATE TRIGGER trigger_ucc_updated_at
    BEFORE UPDATE ON user_column_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 2.7 数据库 ER 关系总结

```
users ──N:M── roles ──N:M── role_device_groups ──→ device_groups
  │                                                      │
  └── user_column_configs                      parent_id (self-ref)
                                                         │
                                               device_group_members
                                                   │          │
                                           group_id ↑    device_id ↓
                                                         devices
                                                           │
                                           ┌───────────────┼───────────────┐
                                      device_info   device_parameters   device_registrations
```

---

## 3. 设备组管理详细设计（DM-02）

### 3.1 业务规则

| 规则 | 描述 |
|------|------|
| **R1 两级树** | 设备组严格两级：一级组（parent_id=NULL, level=1）和二级组（parent_id≠NULL, level=2） |
| **R2 默认组** | 系统内置默认一级+二级组，is_default=TRUE，不可修改/删除 |
| **R3 设备归属** | 设备**只能归属二级组**，每个设备**同一时刻只属于一个二级组** |
| **R4 删除联动** | 删除组时，组下设备自动移至默认二级组 |
| **R5 同级唯一** | 同一父节点下名称不重复（DB 唯一索引保证） |
| **R6 新设备归组** | TR069 Inform 注册的新设备自动归入默认二级组 |
| **R7 预注册归组** | 预注册指定目标二级组，设备上线后自动关联 |

### 3.2 模型设计

```go
// topology/model.go

type DeviceGroup struct {
    ID          uuid.UUID         `json:"id"`
    Name        string            `json:"name"`
    ParentID    *uuid.UUID        `json:"parent_id"`
    Level       int               `json:"level"`          // 1=一级, 2=二级
    Status      string            `json:"status"`         // active / disabled
    IsDefault   bool              `json:"is_default"`
    Carrier     model.CarrierCode `json:"carrier,omitempty"`
    Description string            `json:"description,omitempty"`
    Remark      string            `json:"remark,omitempty"`
    SortOrder   int               `json:"sort_order"`
    CreatedBy   string            `json:"created_by,omitempty"`
    UpdatedBy   string            `json:"updated_by,omitempty"`
    DeviceCount int               `json:"device_count"`   // 查询时计算，不持久化
    Children    []DeviceGroup     `json:"children,omitempty"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

// 默认设备组固定 UUID（全局常量）
const (
    DefaultLevel1GroupID = "00000000-0000-0000-0000-000000000001"
    DefaultLevel2GroupID = "00000000-0000-0000-0000-000000000002"
)
```

### 3.3 API 设计

#### 3.3.1 设备组 CRUD

| Method | Path | 说明 | 权限 |
|--------|------|------|------|
| GET | /api/v1/device-groups/tree | 设备组树（含设备计数 + 统计） | device-groups:read |
| POST | /api/v1/device-groups | 新建设备组 | device-groups:write |
| GET | /api/v1/device-groups/:id | 设备组详情 | device-groups:read |
| PUT | /api/v1/device-groups/:id | 修改设备组 | device-groups:write |
| DELETE | /api/v1/device-groups/:id | 删除设备组 | device-groups:write |
| GET | /api/v1/device-groups/:id/check-delete | 删除前检查 | device-groups:read |
| GET | /api/v1/device-groups/stats | 分组统计 | device-groups:read |
| PUT | /api/v1/device-groups/sort | 批量排序 | device-groups:write |

#### 3.3.2 设备组成员管理

| Method | Path | 说明 | 权限 |
|--------|------|------|------|
| GET | /api/v1/device-groups/:id/devices | 组内设备列表（分页） | device-groups:read |
| POST | /api/v1/device-groups/:id/devices | 批量添加设备到组 | device-groups:write |
| DELETE | /api/v1/device-groups/:id/devices | 批量移除设备（归默认组） | device-groups:write |
| POST | /api/v1/device-groups/move-devices | 批量移动设备到目标组 | device-groups:write |

#### 3.3.3 请求/响应结构

**创建设备组请求：**

```json
{
  "name": "华东区域",
  "parent_id": null,
  "remark": "华东区域基站分组",
  "sub_groups": [
    {
      "name": "上海站点组",
      "remark": "上海区域",
      "device_ids": ["uuid1", "uuid2"]
    },
    {
      "name": "杭州站点组"
    }
  ]
}
```

- `parent_id` 为空 → 创建一级组（可同时带 `sub_groups` 创建二级子组）
- `parent_id` 非空 → 创建二级组（可同时带 `device_ids` 分配设备）

**删除前检查响应：**

```json
{
  "can_delete": true,
  "has_devices": true,
  "device_count": 15,
  "message": "删除后 15 个设备将移动到默认设备组"
}
```

**设备组树响应（含计数和统计）：**

```json
{
  "items": [
    {
      "id": "00000000-0000-0000-0000-000000000001",
      "name": "Default Level Group",
      "level": 1,
      "is_default": true,
      "device_count": 0,
      "children": [
        {
          "id": "00000000-0000-0000-0000-000000000002",
          "name": "Default Level Group",
          "level": 2,
          "is_default": true,
          "device_count": 128
        }
      ]
    }
  ],
  "stats": {
    "total_groups": 12,
    "grouped_devices": 115,
    "ungrouped_devices": 0
  }
}
```

> **注意**：在两级结构 + 设备唯一归属约束下，`ungrouped_devices` 正常情况下应为 0（所有设备都在默认二级组或其他二级组中）。但迁移过渡期可能存在未归组设备。

### 3.4 核心业务流程

#### 创建设备组

```
CreateDeviceGroup(ctx, req, operator):
  1. 判断层级：parent_id 为空 → level=1，非空 → level=2
  2. 如果 level=2：
     - 校验 parent 存在且 parent.level = 1
     - 校验 parent.status = 'active'
  3. 校验同级同父下名称不重复（ExistsByParentAndName）
  4. 事务开始 —
     a. 创建设备组，created_by = operator
     b. 如果一级组 + 有 sub_groups：
        - 逐个创建二级子组（parent_id = 新一级组 ID）
        - 每个子组的 device_ids 执行 MoveDevices（从原组移出 + 加入新组）
     c. 如果二级组 + 有 device_ids：
        - MoveDevices 到新创建的二级组
  5. 事务提交
```

#### 删除设备组

```
DeleteDeviceGroup(ctx, id, operator):
  1. 查询目标组
  2. is_default = true → 拒绝
  3. 事务开始 —
     a. 收集该组及所有子组的 ID 列表
     b. 查询这些组关联的所有设备 ID
     c. 将设备全部移至默认二级组：
        INSERT INTO device_group_members (group_id, device_id, added_at)
        VALUES ($default_l2, $device_id, NOW())
        ON CONFLICT (device_id) DO UPDATE SET
            group_id = EXCLUDED.group_id, added_at = EXCLUDED.added_at
     d. DELETE FROM device_groups WHERE id = ANY($group_ids)
        （CASCADE 自动清理 device_group_members 旧记录）
  4. 事务提交
```

#### 设备移动（原子操作）

```
MoveDevices(ctx, deviceIDs, targetGroupID):
  1. 校验 target 组存在且 level = 2
  2. 使用 UPSERT 原子移动：
     INSERT INTO device_group_members (group_id, device_id, added_at)
     SELECT $target_group_id, unnest($device_ids), NOW()
     ON CONFLICT (device_id) DO UPDATE SET
         group_id = EXCLUDED.group_id, added_at = EXCLUDED.added_at
  3. 返回移动设备数
```

> **关键设计**：利用 `UNIQUE(device_id)` + `ON CONFLICT DO UPDATE` 实现设备在组之间的原子移动，无需先删后插。

---

## 4. 数据权限设计（DM-07）

### 4.1 权限模型

```
用户(User) ──N:M── 角色(Role) ──N:M── 设备组(DeviceGroup)
              via user_roles         via role_device_groups
```

**传递链**：用户 → 所有角色 → 角色关联的设备组 → 设备组下的设备

**权限粒度**：
- 关联一级组 = 可见其下**所有二级组**的设备
- 关联二级组 = **仅可见该二级组**的设备
- 超管（carrier=NULL 的用户）= **全部可见**，不受限

### 4.2 角色管理扩展

在现有 admin 模块的角色 CRUD 中扩展数据权限：

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/roles/:id/device-groups | 获取角色关联的设备组 ID 列表 |
| PUT | /api/v1/roles/:id/device-groups | 设置角色的数据权限（全量替换） |

**请求体**：

```json
{
  "group_ids": ["uuid1", "uuid2", "uuid3"]
}
```

**实现**：在 admin service 的 `CreateRole`/`UpdateRole` 中同步处理 `role_device_groups` 关联。

### 4.3 数据权限查询逻辑

```go
// GetUserVisibleGroupIDs 获取用户可见的设备组 ID 列表
// 返回 (groupIDs, isAll)，isAll=true 表示超管全部可见
func GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, bool) {
    // 1. 获取用户信息
    user := GetUser(ctx, userID)
    if user.Carrier == nil {
        return nil, true  // 超管，全部可见
    }

    // 2. 查 Redis 缓存
    cacheKey := fmt.Sprintf("user:visible_groups:%s", userID)
    if cached := redis.Get(cacheKey); cached != nil {
        return cached, false
    }

    // 3. 查询用户所有角色 → 角色关联的设备组
    // SELECT DISTINCT rdg.group_id
    // FROM user_roles ur
    // JOIN role_device_groups rdg ON rdg.role_id = ur.role_id
    // WHERE ur.user_id = $1
    directGroupIDs := queryRoleDeviceGroups(userID)

    // 4. 展开一级组 → 包含其下所有二级组
    level1IDs := filterLevel1(directGroupIDs)
    expandedL2 := queryChildGroups(level1IDs)  // level=2 AND parent_id IN (...)

    // 5. 合并去重：直接关联的二级组 + 展开的二级组
    allL2IDs := merge(filterLevel2(directGroupIDs), expandedL2)

    // 6. 缓存到 Redis，TTL 5 分钟
    redis.Set(cacheKey, allL2IDs, 5*time.Minute)

    return allL2IDs, false
}
```

### 4.4 设备查询权限注入

在 device 模块的 `DeviceFilter` 中新增数据权限字段：

```go
type DeviceFilter struct {
    // ... 现有过滤字段 ...

    // 数据权限过滤（由 handler 从 JWT context 注入）
    GroupID       *uuid.UUID    // 单组过滤（前端点击某个组）
    VisibleGroups []uuid.UUID   // 数据权限（nil=全部可见，空=无权限）
    model.ListRequest
}
```

**Repository SQL 注入**：

```go
// device_info_pg_repository.go ListDevicesWithInfo 中
if filter.GroupID != nil {
    // 前端选择了具体的组
    builder = builder.
        Join("device_group_members dgm ON dgm.device_id = d.id").
        Where(sq.Eq{"dgm.group_id": *filter.GroupID})
}
if filter.VisibleGroups != nil {
    if len(filter.VisibleGroups) == 0 {
        // 无任何数据权限 → 返回空
        builder = builder.Where("FALSE")
    } else {
        // 只返回权限范围内的设备
        builder = builder.Where(
            "d.id IN (SELECT device_id FROM device_group_members WHERE group_id = ANY(?))",
            filter.VisibleGroups,
        )
    }
}
```

**Handler 调用流程**：

```go
func (h *Handler) ListDevices(c *gin.Context) {
    filter := DeviceFilter{...}
    // ... 解析查询参数 ...

    // 注入数据权限
    userID := middleware.GetUserID(c)
    visibleGroups, isAll := h.permService.GetUserVisibleGroupIDs(c, userID)
    if !isAll {
        filter.VisibleGroups = visibleGroups
    }

    result, err := h.service.ListDevicesWithInfo(c, filter)
    // ...
}
```

### 4.5 缓存策略

| Key | TTL | 说明 |
|-----|-----|------|
| `user:visible_groups:{userID}` | 5 分钟 | 用户可见设备组 ID 列表 |

**缓存失效触发**：

| 场景 | 操作 |
|------|------|
| 修改角色的 device_group_ids | 清除该角色下所有用户的缓存 |
| 用户角色变更（分配/移除角色） | 清除该用户的缓存 |
| 设备组删除 | 清除所有用户的缓存 |

---

## 5. 设备注册增强设计（DM-01）

### 5.1 手动注册（SN 录入）

支持单个或批量输入 SN 进行预注册：

**API**：

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/device-registrations | 预注册设备（单个或批量 SN） |
| GET | /api/v1/device-registrations | 查询预注册列表 |
| DELETE | /api/v1/device-registrations/:id | 删除预注册记录 |

**请求体**：

```json
{
  "serial_numbers": ["SN001", "SN002", "SN003"],
  "group_id": "target-level2-group-uuid",
  "site_name": "北京朝阳站点",
  "remark": "2026Q1 部署批次"
}
```

**业务逻辑**：
1. 校验 `group_id` 为 level=2 的二级组
2. 校验 SN 格式合法性
3. 去重：已存在的 pending 状态 SN → 跳过并返回提示
4. 批量插入 `device_registrations`

### 5.2 Excel 批量导入

**API**：

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/device-registrations/template | 下载导入模板 |
| POST | /api/v1/device-registrations/import | 上传 Excel 批量导入 |
| GET | /api/v1/device-registrations/import/:batchId | 查询导入结果 |

**导入模板字段**：

| 字段 | 必填 | 类型 | 校验规则 |
|------|------|------|---------|
| 设备SN | 是 | 文本 | 非空，≤64 字符 |
| 安装经度 | 是 | 数值 | [-180, 180]，6位小数 |
| 安装纬度 | 是 | 数值 | [-90, 90]，6位小数 |
| 安装高度 | 是 | 数值 | [0, 99999999] |
| 基站名称 | 是 | 文本 | ≤128 字符 |
| 机械下倾角(°) | 是 | 数值 | [0, 9] |
| 垂直3dB波束宽度 | 是 | 数值 | [1, 9] |
| 水平方位角(°) | 是 | 数值 | [0, 359] |
| 备注 | 否 | 文本 | ≤512 字符 |

**导入流程**：
1. 上传文件 → 生成 `import_batch_id`
2. 逐行解析校验（格式、范围、SN 唯一性）
3. 合法行批量插入 `device_registrations`
4. 返回统计：成功 N 行、失败 M 行、失败详情

### 5.3 设备上线自动关联

修改 `RegisterFromInform` 流程：

```
RegisterFromInform(ctx, inform, carrier):
  // ... 现有逻辑 ...
  1. 创建 device 记录
  2. 查询是否有预注册记录：
     SELECT * FROM device_registrations
     WHERE serial_number = $sn AND status = 'pending'
  3. 如果有预注册记录：
     a. 将设备加入预注册指定的 group_id（二级组）
     b. 同步预注册信息到 device_info（site_name, longitude, latitude, height 等）
     c. 更新预注册状态 status = 'online', device_id = device.id
  4. 如果无预注册记录：
     a. 将设备加入默认二级组
  5. 发布 device.registered 事件
```

---

## 6. 设备列表增强设计（DM-03）

### 6.1 按组过滤

在 `DeviceFilter` 中新增 `GroupID` 字段（详见第 4.4 节），前端调用：

```
GET /api/v1/devices?group_id=xxx&page=1&page_size=20
```

复用现有 `ListDevicesWithInfo` 查询逻辑，通过 JOIN `device_group_members` 实现过滤。

### 6.2 列自定义

**API**：

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/column-configs/:pageKey | 获取用户列配置 |
| PUT | /api/v1/column-configs/:pageKey | 保存用户列配置 |

**请求体**：

```json
{
  "columns": [
    {"key": "serial_number", "visible": true, "width": 140, "order": 1},
    {"key": "device_name", "visible": true, "width": 160, "order": 2},
    {"key": "ip_address", "visible": true, "width": 120, "order": 3},
    {"key": "status", "visible": true, "width": 90, "order": 4}
  ]
}
```

**实现**：存储在 `user_column_configs` 表，page_key='device_list'。无配置时返回系统默认列。

### 6.3 数据导出

**API**：

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/devices/export | 条件导出（支持当前过滤条件） |

**导出策略**：
- ≤1 万行：同步导出，返回文件流
- \>1 万行：异步任务，返回 task_id，前端轮询下载
- 格式支持：CSV（字符流）、Excel（分片写入）
- 导出内容受数据权限控制

---

## 7. 涉及文件变更清单

### 7.1 数据库迁移（新建）

| 文件 | 迁移编号 | 说明 |
|------|---------|------|
| `000065_enhance_device_groups.up.sql` | 000065 | device_groups 增强 + 默认组种子数据 |
| `000065_enhance_device_groups.down.sql` | 000065 | 回滚 |
| `000066_create_role_device_groups.up.sql` | 000066 | 角色-设备组数据权限表 |
| `000066_create_role_device_groups.down.sql` | 000066 | 回滚 |
| `000067_create_device_registrations.up.sql` | 000067 | 设备预注册表 |
| `000067_create_device_registrations.down.sql` | 000067 | 回滚 |
| `000068_create_user_column_configs.up.sql` | 000068 | 用户列配置表 |
| `000068_create_user_column_configs.down.sql` | 000068 | 回滚 |

### 7.2 topology 模块（设备组管理改造）

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `model.go` | 修改 | DeviceGroup 新增字段，新增请求/响应结构 |
| `repository.go` | 修改 | 接口扩展（TreeWithCounts, BatchOps, Stats, CheckDelete 等） |
| `pg_repository.go` | 修改 | SQL 实现（新字段读写、计数查询、UPSERT 移动、事务删除） |
| `service.go` | 修改 | 业务逻辑重写（两级约束、默认组保护、设备移动事务） |
| `handler.go` | 修改 | 路由改为 /device-groups，新增 check-delete/move-devices/batch 等端点 |
| `service_test.go` | 新建 | 单元测试 |
| `handler_test.go` | 修改 | 端点测试 |

### 7.3 device 模块（设备列表增强）

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `repository.go` | 修改 | DeviceFilter 新增 GroupID + VisibleGroups |
| `device_info_pg_repository.go` | 修改 | ListDevicesWithInfo 新增 JOIN 过滤逻辑 |
| `handler.go` | 修改 | 解析 group_id 参数 + 注入数据权限 |
| `service.go` | 修改 | RegisterFromInform 加入设备归组逻辑 |
| `registration_handler.go` | 新建 | 预注册 CRUD + Excel 导入端点 |
| `registration_service.go` | 新建 | 预注册业务逻辑 |
| `registration_repository.go` | 新建 | 预注册持久层 |

### 7.4 admin 模块（数据权限扩展）

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `model.go` | 修改 | Role 新增 DeviceGroupIDs 字段 |
| `repository.go` | 修改 | 新增 RoleDeviceGroupReader/Writer 接口 |
| `pg_role_repository.go` | 修改 | 角色-设备组关联 CRUD SQL |
| `handler.go` | 修改 | 新增角色数据权限 GET/PUT 端点 |
| `service.go` | 修改 | CreateRole/UpdateRole 处理 device_group_ids |
| `permission_service.go` | 新建 | GetUserVisibleGroupIDs + 缓存 |

### 7.5 全局

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `global/consts.go` | 修改 | 新增默认设备组 UUID 常量 |
| `global/errors.go` | 修改 | 新增设备组相关错误码 |
| `cmd/app/router/router.go` | 修改 | 注册新路由组，更新 DI |

---

## 8. 实施阶段

| 阶段 | 内容 | 产出 | 涉及模块 |
|------|------|------|---------|
| **P1** | 数据库迁移 + 模型改造 | 000065~000068 迁移就绪，默认组种子数据 | migrations/ |
| **P2** | 设备组 CRUD 改造 | 两级树增删改查 + 默认组保护 + 同级唯一 | topology/ |
| **P3** | 设备归属管理 | 批量添加/移除/移动 + 删除联动 + 设备计数 | topology/ |
| **P4** | 设备列表按组过滤 | group_id 参数 + JOIN 查询 | device/ |
| **P5** | 数据权限 | role_device_groups + 查询注入 + 缓存 | admin/ + device/ |
| **P6** | 设备注册增强 | 手动注册 + Excel 导入 + 上线自动关联 | device/ |
| **P7** | 列自定义 + 数据导出 | 用户列配置 + CSV/Excel 导出 | device/ |
| **P8** | 测试与验证 | 单元测试 + E2E 测试覆盖 | 全部 |

---

## 9. 测试方案

### 9.1 单元测试

| 测试点 | 用例 |
|--------|------|
| 创建一级设备组 | 正常创建 / 名称重复拒绝 / 指定子组批量创建 |
| 创建二级设备组 | 正确父组 / 父组不存在拒绝 / 父组是二级拒绝 |
| 修改设备组 | 正常修改 / 默认组拒绝 / 名称重复拒绝 |
| 删除设备组 | 正常删除 / 默认组拒绝 / 有设备时移至默认组 / 删一级组级联 |
| 设备归属 | 只能加入二级组 / 一个设备一个组 / UPSERT 移动原子性 |
| 数据权限 | 超管全部可见 / 角色过滤 / 一级组展开 / 无权限返回空 |
| 预注册 | 正常注册 / SN 重复跳过 / Excel 校验（格式/范围/唯一） |
| 设备上线归组 | 有预注册 → 归指定组 / 无预注册 → 归默认组 |

### 9.2 E2E 测试场景

1. 创建一级组 + 同时创建 2 个二级子组 + 每个子组分配 3 个设备 → 树显示正确计数
2. 修改默认组名称 → 403；修改普通组名称 → 200
3. 删除有 5 个设备的二级组 → check-delete 返回 device_count=5 → 确认删除 → 设备出现在默认组
4. 删除含 2 个二级子组（共 10 设备）的一级组 → 全部移至默认二级组
5. 设置角色 A 关联设备组 X → 用户用角色 A 登录 → 只能看到组 X 的设备
6. 超管登录 → 可看到所有设备组和设备
7. 预注册 SN001~SN003 到组 Y → SN001 设备上线 → 自动归入组 Y
8. Excel 导入 100 行（90 合法 + 10 不合法）→ 返回成功 90、失败 10

---

## 10. 关键设计决策

| 决策 | 理由 |
|------|------|
| **严格两级树而非任意深度** | 需求文档明确指定两级；DB 约束保证数据一致性；简化前端渲染和权限计算 |
| **设备唯一归属（UNIQUE device_id）** | 业务需求：一个设备一个组；ON CONFLICT DO UPDATE 实现原子移动 |
| **设备计数不持久化** | LEFT JOIN COUNT 在设备组数量（<1000）规模下毫秒级；避免写入时维护计数器 |
| **数据权限用 IN 子查询** | device_group_members.device_id 已有索引；10 万设备规模下子查询性能充分 |
| **缓存可见组 5 分钟 TTL** | 平衡一致性和性能；权限变更时主动清除缓存 |
| **预注册表独立于 devices** | 预注册的设备尚未上线，不应混入 devices 主表；上线后通过 device_id 关联 |
| **列配置用 JSONB** | 灵活存储列顺序、宽度、可见性，无需为每种页面建独立表 |
| **默认组用固定 UUID** | 代码中引用常量而非查询，避免启动依赖；迁移脚本保证数据存在 |
| **路由从 /groups 迁移到 /device-groups** | 语义更清晰，与前端路由 `/device/group` 对应；旧路由暂时保留兼容 |

---

## 11. 与其他模块的关联

| 关联模块 | 关联方式 | 说明 |
|---------|---------|------|
| **ACS (F01)** | device.registered 事件 | Inform 注册新设备 → 自动归入默认组或预注册组 |
| **Provisioning (F09)** | device.sync.completed 事件 | 参数同步完成后启动自动开通 |
| **Dashboard (F06)** | groupRepo.GetTreeWithCounts | 仪表盘区域统计（已使用 groupRepo） |
| **Alarm (F04)** | 设备 → 设备组 → 区域告警统计 | 按组聚合告警数据 |
| **Admin RBAC** | role_device_groups | 角色数据权限决定设备可见范围 |
| **Northbound/OSS (F08)** | 数据导出 | 按权限范围导出设备数据给上游系统 |

---

## 12. Redis Key 规范

| Key | 类型 | TTL | 说明 |
|-----|------|-----|------|
| `user:visible_groups:{userID}` | List(UUID) | 5 min | 用户可见设备组 |
| `device:group:{deviceID}` | String(UUID) | 24 h | 设备归属组 ID 缓存 |
| `device_reg:sn:{serialNumber}` | Hash | — | 预注册设备缓存（启动预热） |

---

## 13. 设备操作扩展功能设计

> 以下功能来自原差距分析文档中的 G09-G27，前述章节未涵盖的详细实现方案。

### 13.1 射频开关（G09）

通过 TR069 SetParameterValues 下发射频参数。

**API 端点**：

| Method | Path | 说明 | 权限 |
|--------|------|------|------|
| PUT | /api/v1/devices/:id/rf-switch | 射频开关控制 | devices:write |

**请求体**：

```json
{"enabled": true}
```

**实现**：通过 `Carrier` 接口获取 RF 控制参数路径 → `SetParameters()` 下发 → 下发成功后更新 `device_info.rf_status`。

**Carrier 接口扩展**：

```go
GetRFControlPath(tech Technology) string
// CMCC LTE: "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_RadioEnable"
// CMCC NR:  "Device.Services.FAPService.1.CellConfig.NR.RAN.RF.X_COM_RadioEnable"
```

### 13.2 设备列表导出详细方案（G04）

**新增文件**：

```
internal/device/export_handler.go    — HTTP 处理
internal/device/export_service.go    — 导出逻辑（流式写入）
```

**实现要点**：
- 流式分页读取（每次 500 条），避免内存溢出
- CSV：`encoding/csv` 直接写 `http.ResponseWriter`
- Excel：`github.com/xuri/excelize/v2` 流式写入（`StreamWriter`）
- 导出字段包含 `devices` + `device_info` 合并数据
- 导出列可基于用户列配置（§6.2），无配置则使用默认列
- V1 同步处理（设 5 分钟超时），V2 再引入异步任务

### 13.3 设备详情聚合视图（G12）

**API**：`GET /api/v1/devices/:id/summary`

并行查询 device + device_info + alarm + kpi + license，用 `errgroup` 聚合返回：

```json
{
  "device": { /* devices + device_info 合并 */ },
  "alarm_summary": {"critical": 2, "major": 5, "total_active": 10},
  "latest_kpi": {"rsrp": -85.2, "sinr": 12.5},
  "license": {"status": "active", "expires_at": "..."}
}
```

**实现**：Service 层从 device_parameters 按前缀查询组装 DTO（MME Pool / License / Antenna / Cells），使用 `GetByGroup` 方法按 `param_group` 等值匹配代替 LIKE 查询（已实施，见 000064 迁移）。

### 13.4 日志收集 — 即时模式（G15）

**来源**：《日志收集功能逻辑设计文档》§3

**新增文件**：

```
internal/device/logcollect/
    model.go              — LogCollectTask 结构体、7 态状态机
    repository.go         — 接口定义
    pg_repository.go      — PostgreSQL 实现
    handler.go            — HTTP 处理
    service.go            — 业务逻辑（入队、状态流转、超时检测）
migrations/
    000XXX_create_log_collect_tasks.up.sql
```

**任务状态机**（7 态）：

```
PENDING(0) → QUEUED(1) → DOWNLOADING(2) → UPLOADING(3) → COMPLETED(4)
                                                      ├─→ FAILED(5)
                                                      └─→ TIMEOUT(6)
```

**API 端点**：

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/devices/:id/log-collect | 创建即时日志收集任务 |
| GET | /api/v1/devices/:id/log-collect | 查询设备日志收集记录 |
| GET | /api/v1/log-collect/tasks | 全局日志收集任务列表 |
| GET | /api/v1/log-collect/tasks/:task_id | 任务详情 |
| GET | /api/v1/log-collect/tasks/:task_id/file | 下载日志文件 |

**Carrier 接口扩展**：

```go
GetLogCollectParams(tech Technology) LogCollectParamPaths

type LogCollectParamPaths struct {
    FileType string  // Upload RPC 的 FileType 参数值
    URL      string  // 设备上传目标 URL 参数路径
    Username string  // FTP/HTTP 认证用户名参数路径
    Password string  // FTP/HTTP 认证密码参数路径
}
```

**实现要点**：
- 入队前检查设备在线状态和已有任务（避免重复）
- 生成 MinIO presigned URL 作为上传目标
- 通过 `cmdQueue` 下发 Upload RPC
- 监听 TransferComplete 事件标记完成
- 超时检测：定时扫描 QUEUED/DOWNLOADING 状态超过阈值的任务

### 13.5 日志收集 — 周期模式（G16）

**来源**：《日志收集功能逻辑设计文档》§4

**新增数据模型**：

```sql
CREATE TABLE log_collect_schedules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL,
    log_type    VARCHAR(32) NOT NULL,      -- running_log / security_log
    cron_expr   VARCHAR(64) NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_by  VARCHAR(64),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**实现要点**：
- 使用 `robfig/cron/v3` 调度
- 每次触发创建一条 `log_collect_tasks` 记录
- 复用 G15 的即时收集流程

### 13.6 日志收集 — 平台适配（G17）

**来源**：《日志收集功能逻辑设计文档》§6

**4G LTE 参数路径**：
```
FileType = "4 Vendor Log File"
URL      = "Device.DeviceInfo.VendorLogFile.1.URL"
Username = "Device.DeviceInfo.VendorLogFile.1.Username"
Password = "Device.DeviceInfo.VendorLogFile.1.Password"
```

**5G NR 参数路径**：
```
FileType = "4 Vendor Log File"
URL      = "Device.Services.FAPService.1.FAPControl.NR.LogFile.1.URL"
Username = "Device.Services.FAPService.1.FAPControl.NR.LogFile.1.Username"
Password = "Device.Services.FAPService.1.FAPControl.NR.LogFile.1.Password"
```

**改动文件**：三个运营商适配器（cmcc/ctcc/cucc adapter.go）实现 `GetLogCollectParams()`。

### 13.7 报文收集（G11）

厂商特定功能，通过参数下发触发抓包 + Upload 上传。

```
POST /api/v1/devices/:id/packet-capture
body: {"duration": 60}
```

需要 `Carrier` 接口适配器提供抓包控制参数路径。

### 13.8 固件升级回退（G18）

**来源**：《设备升级回退流程设计文档》§7

**API**：`POST /api/v1/upgrade-tasks/:id/rollback`

**回退流程**：

```
1. GetParameterValues(SoftwareImage.{i}.Active) 确认当前激活分区
2. GetParameterValues(SoftwareImage.{i}.RollbackEnable) 检查是否支持回退
3. SetParameterValues(SoftwareImage.{i}.RollbackEnable = true) 触发回退
4. 等待设备重启 + Inform (event=102 或 1 BOOT)
5. GetParameterValues 确认版本已恢复
```

**平台差异**：

| 平台 | 回退参数路径 |
|------|------------|
| 4G LTE | `Device.Services.FAPService.1.FAPControl.LTE.SoftwareImage.{i}.RollbackEnable` |
| 5G NR | `Device.Services.FAPService.1.FAPControl.NR.SoftwareImage.{i}.RollbackEnable` |

### 13.9 升级任务挂起/恢复/终止（G19）

**来源**：《设备升级回退流程设计文档》§8

| Method | Path | 说明 |
|--------|------|------|
| PUT | /api/v1/upgrade-tasks/:id/suspend | 挂起（不再向设备下发） |
| PUT | /api/v1/upgrade-tasks/:id/resume | 恢复（重新排入队列） |
| PUT | /api/v1/upgrade-tasks/:id/terminate | 终止（标记放弃） |

### 13.10 5G 升级完成事件处理（G20）

**来源**：《设备升级回退流程设计文档》§5.2

**改动文件**：`internal/device/inform_handler.go`

**实现**：
- InformHandler 解析事件码列表，检测 `102` 或 `M Download`
- 匹配到活跃的升级任务后标记为 COMPLETED
- 更新 `devices.firmware_version` 为新版本

### 13.11 异常重启日志 — BOOT 检测（G23）

**来源**：《设备异常重启日志功能流程规范文档》§4

**改动文件**：`internal/device/inform_handler.go`

**检测逻辑**：

```go
events := inform.Events // e.g. ["1 BOOT", "2 PERIODIC"]
hasBoot := slices.Contains(events, "1 BOOT")
hasBootstrap := slices.Contains(events, "0 BOOTSTRAP")

if hasBoot && !hasBootstrap {
    // 异常重启！提取原因
    mainReason := paramValues["Device.DeviceInfo.X_VENDOR_MainFaultReason"]
    detailReason := paramValues["Device.DeviceInfo.X_VENDOR_DetailFaultReason"]
    // 发布事件 device.reboot.abnormal
}
```

**平台差异（故障原因参数路径）**：

| 平台 | MainReason | DetailReason |
|------|-----------|-------------|
| 4G LTE | `Device.DeviceInfo.X_BAICELLS_MainFaultReason` | `Device.DeviceInfo.X_BAICELLS_DetailFaultReason` |
| 5G NR | `Device.Services.FAPService.1.FAPControl.NR.X_BAICELLS_MainFaultReason` | 同级 DetailFaultReason |

### 13.12 异常重启日志 — 自动/手动收集（G24/G25）

**来源**：《设备异常重启日志功能流程规范文档》§5-§6

**新增文件**：

```
internal/device/rebootlog/
    model.go              — RebootLogEntry 结构体
    repository.go         — 接口定义
    pg_repository.go      — PostgreSQL 实现
    service.go            — 自动/手动收集逻辑
    handler.go            — HTTP 处理
migrations/
    000XXX_create_device_reboot_logs.up.sql
```

**数据模型**：

```sql
CREATE TABLE device_reboot_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL,
    serial_number   VARCHAR(128) NOT NULL,
    reboot_time     TIMESTAMPTZ NOT NULL,
    main_reason     VARCHAR(256),
    detail_reason   TEXT,
    collection_mode VARCHAR(20) NOT NULL,  -- auto / manual
    log_file_path   VARCHAR(512),          -- MinIO 路径
    log_file_size   BIGINT,
    status          VARCHAR(20) NOT NULL,  -- detected / collecting / collected / failed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_reboot_logs_device ON device_reboot_logs (device_id, reboot_time DESC);
```

**自动收集流程**：
1. 订阅 `device.reboot.abnormal` 事件
2. 创建 `device_reboot_logs` 记录（status=detected）
3. 通过 GetParameterValues 获取 FaultLogFile URL
4. 如有日志文件 → Upload RPC 收集到 MinIO
5. TransferComplete → 更新 status=collected

**手动收集 API**：`POST /api/v1/devices/:id/reboot-logs/collect`

流程：SetParameterValues 设置 FaultLogURL → 设备主动上传 → TransferComplete 回调。

### 13.13 异常重启日志 — 列表/导出/清理（G26）

**来源**：《设备异常重启日志功能流程规范文档》§8-§11

| Method | Path | 说明 |
|--------|------|------|
| GET | /api/v1/reboot-logs | 异常重启日志列表（分页、过滤） |
| GET | /api/v1/reboot-logs/:id | 日志详情 |
| GET | /api/v1/reboot-logs/:id/file | 下载日志文件 |
| POST | /api/v1/reboot-logs/export | 批量导出 |

**清理策略**：
- 单设备日志上限（如 100 条），超出删除最早记录
- 全局总量上限（如 100,000 条）
- 磁盘使用率阈值（如 > 80% 时触发清理）
- 定时清理任务（cron）

### 13.14 逻辑删除（G13）

```sql
ALTER TABLE devices ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX idx_devices_deleted ON devices (deleted_at) WHERE deleted_at IS NULL;
```

- `pg_repository.go` 所有查询加 `WHERE deleted_at IS NULL`
- `Delete()` 改为 `UPDATE SET deleted_at = NOW(), status = 'decommissioned'`
- `device_info` 行跟随 `ON DELETE CASCADE` 或同步标记

### 13.15 敏感字段加密（G14）

**建议推迟到 V2**。理由：
1. 经纬度在 topology/geo 计算中使用，加密后无法做范围查询
2. 安全要求需与运营商确认后再定

### 13.16 STUN UDP — 心跳保活增强（G27）

**来源**：《STUN-UDPServer业务说明》§5

**当前状态**：`acs/stun/` 模块已实现基础 STUN 功能和 Store。

**增强需求**：
- UDP 心跳超时检测（超过 2×interval 未收到心跳标记设备 unreachable）
- 连接状态同步到 `device_info`（通过 InfoSyncer）

---

## 14. TR069 报文字段全景分析

> 基于《TR069报文全解析.md》中实际抓包的 252 个参数，分析每个字段的存储去向和利用方式。
> 原始分析来源：《TR069报文全解析.md》§7。

### 14.1 存储架构与设计原则

#### 核心原则：device_info 是 device_parameters 的「物化视图」

device_info 表**仅提取设备列表页必须展示/过滤/排序的少量字段**（控制在 25 列以内），
其余所有 TR069 参数统一存入 device_parameters K-V 表。设备详情页、配置下发、参数对比等场景直接从 device_parameters 读取。

**不在 device_info 中使用 JSONB 列存储 TR069 参数**。理由：

| # | 问题 | 说明 |
|---|------|------|
| 1 | **数据冗余** | device_parameters 已存储全量参数，JSONB 是重复副本，引入一致性风险 |
| 2 | **丢失参数路径** | JSONB 结构化后丢失 TR069 原始路径，配置下发时需反向查找 |
| 3 | **更新代价高** | 修改 JSONB 中单个字段需 读取→反序列化→修改→序列化→写回 整个 JSON |
| 4 | **违背 TR069 数据模型** | TR069 参数树天然是层级 K-V 结构，device_parameters 正是这种结构的自然映射 |
| 5 | **多实例不可控** | MME 16 组×3 字段=48 参数，License 4 组×8 字段=32 参数，小区数动态变化 |

#### 三层存储模型

```
┌──────────────────────────────────────────────────────────────┐
│  Tier 1: 快查列 — devices / device_info 表的具名列            │
│  用途：设备列表展示、过滤、排序（~26 列）                      │
│  准入标准：列表页必须展示 或 必须支持 WHERE/ORDER BY           │
├──────────────────────────────────────────────────────────────┤
│  Tier 2: 完整参数树 — device_parameters K-V 表                │
│  用途：所有 TR069 参数按原始路径 K-V 存储                      │
│  场景：设备详情页、参数配置/对比/审计、配置下发                  │
│  优化：已实现 Hash 32 分区 + param_group 等值查询（000064 迁移）│
├──────────────────────────────────────────────────────────────┤
│  Tier 3: 独立业务表 — alarms / license / etc                  │
│  用途：有独立生命周期的业务数据                                 │
└──────────────────────────────────────────────────────────────┘
```

#### 252 个参数的存储去向

| 存储层 | 参数数量 | 说明 |
|--------|---------|------|
| devices 表具名列 | ~8 | 设备身份/版本/IP，Inform 自动写入 |
| device_info 具名列 | ~18 | 列表展示/过滤必需的运行状态字段 |
| device_parameters K-V | **全部 252** | 所有参数完整保留原始路径 |
| 独立业务表 | 告警参数 | CurrentAlarm → alarm 模块独立管理 |

#### 容量估算（10 万设备）

| 方案 | 存储量 | 写入模式 |
|------|--------|---------|
| device_info 26 列 | ~50 MB（10 万行 × 500B） | 每 5 分钟 Inform 更新 ~13 列 |
| device_parameters 252 参数/设备 | ~5 GB（2500 万行 × 200B） | Inform + ACS 查询后批量 UPSERT |

### 14.2 参数来源分类

| 分类 | 来源 | 参数数量 | 特点 |
|------|------|---------|------|
| **A. Inform 自动携带** | 每次 Inform 设备主动上报 | 37 | 实时性最高，自动获取 |
| **B. ACS 主动查询** | GetParameterValues 按需获取 | 215 | 需 ACS 主动发起查询 |

### 14.3 Inform 参数分析（37 个）

#### 已捕获到快查列（8 个） ✅

| TR069 参数路径 | 存储位置 | 快查列 |
|---------------|---------|--------|
| `Device.DeviceInfo.SoftwareVersion` | devices | firmware_version |
| `Device.DeviceInfo.HardwareVersion` | device_info | hardware_version |
| `FAPService.1.FAPControl.LTE.RFTxStatus` | device_info | rf_status |
| `FAPService.1.FAPControl.LTE.OpState` | device_info | cell_status |
| `Device.DeviceInfo.FAP_adminstate` | device_info | cell_status |
| `FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList` | device_info | plmn |
| `Device.DeviceInfo.X_COM_STATION_RUN_Time` | device_info | run_time |
| `Device.ManagementServer.ConnectionRequestURL` | devices | connection_request_url |

#### 建议新增到快查列的 Inform 字段（2 个）

| 新增列 | 类型 | 来源参数 | 理由 |
|--------|------|---------|------|
| `gps_status` | VARCHAR(20) | `X_COM_GPS_Status` | GPS 状态是基站运维核心指标 |
| `alarm_severity` | VARCHAR(20) | `X_RADISYS_COM_AlarmStatus` | 设备最高告警级别视觉指示 |

#### cell_status 增强计算逻辑

```
FAP_adminstate = false                                          → "未激活"
FAP_adminstate = true && OpState = false                        → "故障"
FAP_adminstate = true && OpState = true && CellOpState = 0      → "退服"
FAP_adminstate = true && OpState = true && CellOpState = 1      → "正常"
```

### 14.4 GetParameterValues 关键参数组分析

#### 许可证（约 42 个参数）

| 存储目标 | 内容 | 说明 |
|---------|------|------|
| device_info 快查列 | `license_status` VARCHAR(20) | active / expiring / expired |
| device_parameters | 全部 42 个参数 K-V | 详情页按前缀 `X_COM_LICENSE.%` 查询 |

**license_status 计算逻辑**：

```go
func calcLicenseStatus(params map[string]string) string {
    hasActive, minRemain := false, math.MaxInt32
    for i := 1; i <= 32; i++ {
        prefix := fmt.Sprintf("...X_COM_LICENSE.Capacity.%d.", i)
        if params[prefix+"State"] == "1" {
            hasActive = true
            remain, _ := strconv.Atoi(params[prefix+"RemainingPeriod"])
            if remain < minRemain { minRemain = remain }
        }
    }
    switch {
    case !hasActive:      return "expired"
    case minRemain <= 30: return "expiring"
    default:              return "active"
    }
}
```

#### MME 池（约 48 个参数）

| 存储目标 | 内容 | 说明 |
|---------|------|------|
| device_info 快查列 | `mme_status` VARCHAR(20) | connected / partial / disconnected |
| device_parameters | 全部 48 个参数 K-V | 详情页按前缀 `MmePoolConfigParam.%` 查询 |

不使用 JSONB 存储 MME 池 — 配置下发需要原始 TR069 路径，JSONB 丢失了路径信息。

**mme_status 计算逻辑**：

```go
func calcMMEStatus(params map[string]string) string {
    activeCount := 0
    for i := 1; i <= 16; i++ {
        prefix := fmt.Sprintf("...MmePoolConfigParam.%d.", i)
        if params[prefix+"MME1Status"] == "1" { activeCount++ }
    }
    switch {
    case activeCount == 0: return "disconnected"
    case activeCount < 2:  return "partial"
    default:               return "connected"
    }
}
```

#### 载波聚合与多小区

| 参数 | 建议存储 | 说明 |
|------|---------|------|
| `FAPService.1.CellConfig.LTE.RAN.CA.PARAMS.NumOfCells` | device_info `num_of_cells` | 载波模式判断基础（SC/CA/DC/TC） |
| `FAPService.2/3.FAPControl.LTE.*` | device_parameters | 多实例参数，按 `FAPService.{n}.%` 前缀查询 |

**多小区参数展示方式**：设备详情页根据 `num_of_cells` 值决定展示哪些小区 Tab，每个 Tab 从 device_parameters 按前缀查询。不在 device_info 冗余存储。

#### 同步状态综合计算

```
tfcsSyncState + tfcsManagerPrimsrc + X_COM_GPS_Status + X_COM_BDS_Status + X_COM_1588_Status
→ 综合判定：GPS同步 / 北斗同步 / 1588同步 / NTP同步 / 未同步 / 异常
```

### 14.5 参数同步机制增强

#### 同步架构

```
CPE → Inform → device_parameters (全量 K-V 写入)
                    ↓
              InfoSyncer.SyncQuickColumns()
                    ↓
              device_info (仅更新快查列的计算值)
```

#### Inform 通用快查列映射（不依赖 Carrier 适配器）

```go
var informDirectMapping = map[string]string{
    "Device.DeviceInfo.X_COM_GPS_Status":                                  "gps_status",
    "Device.DeviceInfo.X_COM_STATION_RUN_Time":                            "run_time",
    "Device.Services.FAPService.1.FAPControl.X_RADISYS_COM_AlarmStatus":   "alarm_severity",
}
```

#### ACS 查询频率分级

| 频率 | 触发时机 | 参数组 | 写入目标 |
|------|---------|--------|---------|
| **每次 Inform** | 心跳时 | GPS 状态、同步源/同步状态、RadioEnable | device_parameters → 快查列计算 |
| **每日一次** | 定时任务 | License、MME 池、天线参数、ETH 速率 | device_parameters → 快查列计算 |
| **首次/变更时** | Bootstrap / ValueChange | 模块类型、经纬度/高度、NumOfCells | device_parameters → 快查列 |

#### 设备详情页数据组装（利用 param_group 分类）

| 详情页 Tab | param_group | 组装逻辑 |
|-----------|-------------|---------|
| MME 连接 | `mme_pool` | `GetByGroup(deviceID, "mme_pool")` → 过滤有效条目 |
| License | `license` | `GetByGroup(deviceID, "license")` → 按 Capacity.{i} 分组 |
| 天线参数 | `antenna` | `GetByGroup(deviceID, "antenna")` → 拍平为 AntennaDetail DTO |
| 小区 2/3 | — | `GetByFAPInstance(deviceID, 2)` / `GetByFAPInstance(deviceID, 3)` |
| GPS/同步 | `sync` | `GetByGroup(deviceID, "sync")` → 汇总为同步状态视图 |

### 14.6 device_info 表字段最终清单

基于「物化视图」原则，device_info 仅保留列表页必需的快查列。

**现有列（24 列）**：device_name, address, remark, project_status, height（手动 5 列）+ eci, pci, cell_id, freq_point, bandwidth, transmit_power, plmn, rf_status, cell_status, mme_status, sync_status, mac, hardware_version（自动 13 列）+ first_online_time, last_offline_time, run_time（时间 3 列）+ creator, updater, created_at, updated_at（审计 4 列）

**迁移 000063 新增列（4 列）**：

| 新增列 | 类型 | 来源 | 理由 |
|--------|------|------|------|
| `num_of_cells` | INTEGER DEFAULT 1 | `CA.PARAMS.NumOfCells` | 载波模式判断 |
| `gps_status` | VARCHAR(20) | `X_COM_GPS_Status` | GPS 运维核心指标 |
| `alarm_severity` | VARCHAR(20) | `X_RADISYS_COM_AlarmStatus` | 告警级别视觉指示 |
| `license_status` | VARCHAR(20) | License 参数计算 | 许可证整体状态 |

**不新增的字段及理由**：

| 曾考虑的字段 | 不新增的理由 |
|-------------|------------|
| `mme_pool` JSONB | 冗余 device_parameters，丢失原始路径 |
| `antenna_info` JSONB | 天线参数仅详情页展示，不需列表过滤 |
| `license_summary` JSONB | 42 个参数双写一致性风险 |
| `cell2_params` JSONB | device_parameters 天然支持前缀查询 |
| `license_expire_days` | 可在 `license_status` 中体现 |

**总计 device_info 列数**：29 列（含 PK），精简可控。

---

## 15. 扩展 Gap 项（G28-G36）

基于 TR069 报文分析（§14），补充以下差距项：

| # | 功能 | 优先级 | 复杂度 | 状态 | 说明 |
|---|------|--------|--------|------|------|
| G28 | device_info 表扩展 — 新增 4 列 | **P1** | 低 | ✅ 已实现 | 迁移 000063 + `device_info_model.go` 含 4 新列 |
| G29 | Inform 通用快查列同步 | **P1** | 低 | ✅ 已实现 | `info_sync.go:InfoSyncer` 调用全部 CalcXxx |
| G30 | mme_status 快查列计算逻辑 | **P1** | 中 | ✅ 已实现 | `info_calc.go:CalcMMEStatus` (L56-77) |
| G31 | license_status 快查列计算逻辑 | **P1** | 中 | ✅ 已实现 | `info_calc.go:CalcLicenseStatus` (L87-110) |
| G32 | cell_status 三维判定逻辑 | **P1** | 低 | ✅ 已实现 | `info_calc.go:CalcCellStatus` (L18-46) |
| G33 | ACS 查询频率分级策略 | **P2** | 中 | ❌ 未实现 | 每次 Inform / 每日 / 首次变更 分级 |
| G34 | 设备详情页复杂数据 DTO 组装 | **P2** | 中 | ❌ 未实现 | Service 层利用 GetByGroup 组装 MME/License/天线/多小区 DTO |
| G35 | 多小区参数处理（CA/DC/TC） | **P2** | 中 | ❌ 未实现 | 根据 num_of_cells 决定详情页展示哪些小区 Tab |
| G36 | sync_status 综合计算 | **P2** | 中 | ✅ 已实现 | GPS + BDS + GLONASS + 1588 + tfcsSync 五源判定 |

---

## 16. 完整实施路线图

> 综合 §8（设备组/权限/注册）与 §13-§15（设备操作/TR069 分析）的完整排期。

### Phase A：设备组基础（§8 P1-P3）

| 序号 | 功能 | 涉及模块 |
|------|------|---------|
| 1 | 数据库迁移 000065~000068 | migrations/ |
| 2 | 设备组两级树 CRUD 改造 | topology/ |
| 3 | 设备归属管理（批量/移动/删除联动/计数） | topology/ |

### Phase B：数据权限 + 注册（§8 P4-P6）

| 序号 | 功能 | 涉及模块 |
|------|------|---------|
| 4 | 设备列表按组过滤 | device/ |
| 5 | 角色-设备组数据权限 + 查询注入 + 缓存 | admin/ + device/ |
| 6 | 设备注册增强（手动 + Excel + 自动归组） | device/ |

### Phase C：快查列增强 + 核心操作（~10d）

| 序号 | 功能 | 工作量 | Gap | 状态 |
|------|------|--------|-----|------|
| 7 | device_info 新增 4 列 + 索引 | 0.5d | G28 | ✅ 已完成（000063 迁移） |
| 8 | 通用 Inform 快查列同步 | 0.5d | G29 | ✅ 已完成（info_sync.go） |
| 9 | cell_status 三维判定 | 0.5d | G32 | ✅ 已完成（info_calc.go） |
| 10 | mme_status/license_status 计算 | 1.5d | G30, G31 | ✅ 已完成（info_calc.go） |
| 11 | 射频开关 | 1d | G09 | ❌ 未实现 |
| 12 | 设备列表导出（CSV/Excel） | 2d | G04 | ❌ 未实现 |
| 13 | 设备详情聚合 + DTO 组装 | 2d | G12, G34 | ❌ 未实现 |
| 14 | 列自定义配置 | 1d | G05 | ❌ 未实现 |
| 15 | 5G 升级完成事件 | 1d | G20 | ❌ 未实现 |

### Phase D：日志收集体系（~12d）

| 序号 | 功能 | 工作量 | Gap |
|------|------|--------|-----|
| 16 | 日志收集 — 即时模式 | 3d | G15 |
| 17 | 日志收集 — 平台适配 | 1d | G17 |
| 18 | 日志收集 — 周期模式 | 2d | G16 |
| 19 | 异常重启 — BOOT 检测 | 1.5d | G23 |
| 20 | 异常重启 — 自动收集 | 2d | G24 |
| 21 | 异常重启 — 手动收集 | 1d | G25 |
| 22 | 异常重启 — 列表/导出/清理 | 1.5d | G26 |

### Phase E：升级增强（~5d）

| 序号 | 功能 | 工作量 | Gap |
|------|------|--------|-----|
| 23 | 固件升级回退 | 2d | G18 |
| 24 | 升级任务挂起/恢复/终止 | 1.5d | G19 |
| 25 | 报文收集 | 1.5d | G11 |

### Phase F：安全与增强（~7d）

| 序号 | 功能 | 工作量 | Gap | 状态 |
|------|------|--------|-----|------|
| 26 | 逻辑删除 | 1.5d | G13 | ❌ 未实现 |
| 27 | STUN 心跳增强 | 1d | G27 | ❌ 未实现 |
| 28 | sync_status 综合计算 | 1d | G36 | ✅ 已完成（info_calc.go） |
| 29 | ACS 查询频率分级 | 1d | G33 | ❌ 未实现 |
| 30 | 多小区参数处理 | 1d | G35 | ❌ 未实现 |
| 31 | 敏感字段加密（待运营商确认） | 1.5d | G14 | ❌ 未实现 |

---

## 17. 与设计文档的差异说明

| 设计文档内容 | 本方案处理 | 理由 |
|-------------|----------|------|
| `device_info` 单表存所有字段 | 拆分为 `devices`（核心）+ `device_info`（扩展） | `devices` 已有完整的 ACS 写入链路，不宜合并 |
| BIGINT 自增主键 | `device_info.device_id` 引用 `devices.id` (UUID) | 项目统一 UUID |
| `network_type`（2G/4G/5G） | 沿用 `devices.technology`（lte/nr） | 系统定位 4G/5G 小基站，无 2G |
| `connect_status`（在线/离线/异常/未激活/维护中） | 沿用 `devices.status` 7 态枚举 | 已有完整状态机 |
| `device_group` 字段在设备表 | 保持独立 `device_groups` 表（topology 模块） | 已实现层级分组 + 成员关联 |
| 逻辑删除 `is_delete` | 使用 `deleted_at` 时间戳 | 更灵活，可知删除时间 |
| 独立操作日志表 | 复用 `syslog/` 审计日志模块 | 避免重复建设 |
| 射频参数仅在主表 | 同时保留 `device_parameters` + `device_info` 快查列 | `device_parameters` 是 TR069 标准模型 |
| 外键 REFERENCES devices(id) | 无外键约束，应用层维护 1:1 | PG 分区表不支持被外键引用 |
| device_parameters LIKE 查询 | 已改造为 Hash 32 分区 + param_group 等值匹配 | 已实施（000064 迁移） |

---

## 18. 设计文档来源索引

| 文档 | 路径 | 涉及章节 |
|------|------|---------|
| 2/4/5G设备列表管理系统 — 后端开发设计文档 | `files/Back-end/2_4_5G设备列表管理系统 - 后端开发设计文档.md` | §13 G04-G14 |
| 日志收集功能逻辑设计文档 | `files/Back-end/日志收集功能逻辑设计文档.md` | §13.4-§13.6 (G15-G17) |
| 设备升级回退流程设计文档 | `files/Back-end/设备升级回退流程设计文档.md` | §13.8-§13.10 (G18-G20) |
| 设备注册功能说明 | `files/Back-end/设备注册功能说明.md` | §5 (DM-01) |
| 设备异常重启日志功能流程规范文档 | `files/Back-end/设备异常重启日志功能流程规范文档.md` | §13.11-§13.13 (G23-G26) |
| STUN-UDPServer业务说明 | `files/Back-end/STUN-UDPServer业务说明.md` | §13.16 (G27) |
| TR069报文全解析 | `files/Back-end/TR069报文全解析.md` | §14 (TR069 报文分析) |
| 系统管理模块开发设计文档 | `files/Back-end/系统管理模块开发设计文档.md` | §4 (admin RBAC) |
| 设备注册与设备分组管理系统开发设计文档 | `files/Back-end/设备注册与设备分组管理系统 开发设计文档.md` | §3, §5 |
| 设备组管理开发设计文档 | `files/Back-end/设备组管理开发设计文档.md` | §3 |

### 关联设计文档

| 文档 | 编号 | 关联 |
|------|------|------|
| 设备列表管理差距分析 | 已合并 | 本文 §13-§15 的详细来源，G01-G36 完整分析 |
| device_parameters 分区优化 | 已实施 | 000064 迁移，§14.5 中 GetByGroup 查询依赖此优化 |
