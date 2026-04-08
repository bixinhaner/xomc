# 设备字段映射文档

> 本文档记录设备列表（Device List）和设备分组（Device Grouping）页面的字段定义、数据库映射关系以及字段变更时机。

---

## 1. 数据源概述

设备数据存储在三个主要表中：

| 表名 | 说明 | 关系 |
|------|------|------|
| `devices` | 设备核心信息表（按运营商分区） | 主表 |
| `device_info` | 设备运维扩展信息 | 与 devices 1:1 关联 |
| `device_groups` + `device_group_members` | 设备分组及成员关系 | 与 devices N:M 关联 |

**数据查询方式**：通过 `LEFT JOIN` 组合三表数据返回扁平化结构。

---

## 2. 设备列表页面字段

### 2.1 核心字段（devices 表）

| 前端字段 | 后端字段 | 数据库表 | 类型 | 说明 | 变更时机 |
|----------|----------|----------|------|------|----------|
| `sn` | `serial_number` | devices | VARCHAR(64) | 设备序列号（唯一标识） | 设备首次注册（TR069 Inform） |
| `connStatus` | `status` | devices | VARCHAR(20) | 连接状态 | ACS 会话状态变更、设备上线/离线 |
| `networkType` | `technology` | devices | VARCHAR(3) | 基站制式 | 设备注册时由数据模型解析确定 |
| `productType` | `product_class` | devices | VARCHAR(64) | 产品类型/产品类 | 设备注册时由 TR069 参数获取 |
| `platformType` | `platform_type` | devices | VARCHAR(64) | 平台类型 | 设备注册时获取 |
| `deviceModel` | `model_name` | devices | VARCHAR(128) | 设备型号 | 设备注册时获取 |
| `softwareVersion` | `firmware_version` | devices | VARCHAR(64) | 软件版本 | 固件升级完成 |
| `ipAddress` | `ip_address` | devices | INET | IP 地址 | 每次 Inform 更新 |
| `siteName` | `site_name` | devices | VARCHAR(128) | 站点名称 | 手动编辑或自动开站 |
| `longitude` | `longitude` | devices | DOUBLE PRECISION | 经度 | 手动编辑或 GPS 同步 |
| `latitude` | `latitude` | devices | DOUBLE PRECISION | 纬度 | 手动编辑或 GPS 同步 |
| `vendor` | `manufacturer` | devices | VARCHAR(128) | 厂商 | 设备注册时获取 |
| `stationId` | `site_id` | devices | VARCHAR(64) | 站点 ID | 手动编辑 |
| `createTime` | `created_at` | devices | TIMESTAMPTZ | 创建时间 | 设备首次注册 |
| `lastOnlineTime` / `lastInformTime` | `last_inform_at` | devices | TIMESTAMPTZ | 最后上线/Inform 时间 | 每次 Inform 更新 |

**connStatus 取值映射**：

| 前端显示 | 后端值 | 说明 |
|----------|--------|------|
| 在线 | `active` | 设备活跃运行中 |
| 离线 | `offline` | 设备离线 |
| 发现 | `discovered` | 设备刚发现，未注册 |
| 注册 | `registered` | 设备已注册 |
| 开通中 | `provisioning` | 正在开通配置 |
| 维护 | `maintenance` | 维护模式 |
| 退服 | `decommissioned` | 已退服 |

**networkType 取值**：

| 前端显示 | 前端值 | 后端值 | 说明 |
|----------|--------|--------|------|
| eNB (LTE) | `eNB` | `lte` | 4G LTE 基站 |
| gNB (NR) | `gNB` | `nr` | 5G NR 基站 |

### 2.2 运维扩展字段（device_info 表）

| 前端字段 | 后端字段 | 数据库表 | 类型 | 说明 | 变更时机 |
|----------|----------|----------|------|------|----------|
| `hostName` | `host_name` | device_info | VARCHAR(128) | 主机名 | TR069 参数同步 |
| `productName` | `product_name` | device_info | VARCHAR(128) | 产品名称 | TR069 参数同步 |
| `macAddress` | `mac` | device_info | VARCHAR(64) | MAC 地址 | TR069 参数同步 |
| `remark` | `remark` | device_info | TEXT | 备注 | 手动编辑 |
| `installAddress` | `address` | device_info | VARCHAR(256) | 安装地址 | 手动编辑 |
| `projectStatus` | `project_status` | device_info | VARCHAR(20) | 项目状态 | 手动编辑 |
| `gpsHeight` | `height` | device_info | DECIMAL(10,2) | GPS 高度(米) | TR069 参数同步 |
| `firstOnlineTime` | `first_online_time` | device_info | TIMESTAMPTZ | 首次上线时间 | 设备首次上线 |
| `offlineTime` | `last_offline_time` | device_info | TIMESTAMPTZ | 最后离线时间 | 设备离线时 |
| `onlineDuration` | `run_time` | device_info | BIGINT | 累计运行时长(秒) | 设备状态变更时累计 |

**projectStatus 取值**：

| 值 | 说明 |
|----|------|
| `building` | 建设中 |
| `delivered` | 已交付 |
| `operating` | 运营中 |
| `deactivated` | 已停用 |

### 2.3 无线参数字段（device_info 表）

| 前端字段 | 后端字段 | 数据库表 | 类型 | 说明 | 变更时机 |
|----------|----------|----------|------|------|----------|
| `eci` | `eci` | device_info | VARCHAR(64) | E-UTRAN 小区标识符 | TR069 参数同步 |
| `pci` | `pci` | device_info | VARCHAR(64) | 物理小区标识 | TR069 参数同步 |
| `cellId` | `cell_id` | device_info | VARCHAR(64) | 逻辑小区 ID | TR069 参数同步 |
| `freqPoint` | `freq_point` | device_info | VARCHAR(32) | 频点号 | TR069 参数同步 |
| `bandwidth` | `bandwidth` | device_info | DECIMAL(8,2) | 载波带宽(MHz) | TR069 参数同步 |
| `txPower` | `transmit_power` | device_info | DECIMAL(8,2) | 发射功率(dBm) | TR069 参数同步 |
| `plmnId` | `plmn` | device_info | VARCHAR(32) | PLMN 标识 | TR069 参数同步 |
| `tac` | `tac` | device_info | VARCHAR(64) | TAC | TR069 参数同步 |
| `band` | `band` | device_info | VARCHAR(64) | 频段 | TR069 参数同步 |
| `dlEarfcn` | `dl_earfcn` | device_info | VARCHAR(64) | 下行 EARFCN | TR069 参数同步 |
| `ulEarfcn` | `ul_earfcn` | device_info | VARCHAR(64) | 上行 EARFCN | TR069 参数同步 |

### 2.4 状态字段（device_info 表）

| 前端字段 | 后端字段 | 数据库表 | 类型 | 说明 | 变更时机 |
|----------|----------|----------|------|------|----------|
| `rfStatus` | `rf_status` | device_info | VARCHAR(20) | 射频状态 | TR069 参数同步 |
| `cellStatus` | `cell_status` | device_info | VARCHAR(20) | 小区状态 | TR069 参数同步 |
| `mmeStatus` | `mme_status` | device_info | VARCHAR(20) | MME 连接状态 | TR069 参数同步 |
| `syncStatus` | `sync_status` | device_info | VARCHAR(32) | 时钟同步源状态 | TR069 参数同步 |
| `kpiStatus` | `kpi_status` | device_info | VARCHAR(20) | KPI 状态 | KPI 采集计算 |
| `gpsStatus` | `gps_status` | device_info | VARCHAR(20) | GPS 状态 | TR069 参数同步 |
| `alarmLevel` | `alarm_severity` | device_info | VARCHAR(20) | 最高告警级别 | 告警产生/清除 |
| `licenseStatus` | `license_status` | device_info | VARCHAR(20) | 许可证状态 | TR069 参数同步 |
| `opState` | `op_state` | device_info | VARCHAR(20) | 运营状态 | TR069 参数同步 |
| `ueCount` | `ue_count` | device_info | INTEGER | UE 数量 | TR069 参数同步 |
| `adminState` | `admin_state` | device_info | VARCHAR(20) | 管理状态 | 手动配置 |

**rfStatus 取值**：`on` / `off` / `error`

**cellStatus 取值**：`normal` / `fault` / `unconfigured` / `decommissioned`

### 2.5 设备分组字段（关联查询）

| 前端字段 | 后端字段 | 数据来源 | 类型 | 说明 | 变更时机 |
|----------|----------|----------|------|------|----------|
| `groupName` | `group_name` | device_groups.name | VARCHAR(128) | 设备分组名称 | 设备移动到分组时 |

### 2.6 硬件信息字段

| 前端字段 | 后端字段 | 数据库表 | 类型 | 说明 | 变更时机 |
|----------|----------|----------|------|------|----------|
| `hardwareVersion` | `hardware_version` | device_info | VARCHAR(64) | 硬件版本 | TR069 参数同步 |

### 2.7 扩展字段（extension_data JSONB 或额外列）

以下字段存储在 `devices.extension_data` JSONB 字段或通过其他表关联：

| 前端字段 | 后端字段 | 存储位置 | 类型 | 说明 |
|----------|----------|----------|------|------|
| `halobFlag` | `halob_enabled` | device_info | BOOLEAN | HALOB 功能开关 |
| `ipsecAddr` | `ipsec_addr` | extension_data | VARCHAR | IPSec 地址 |
| `gpsSatelliteCount` | `gps_satellite_count` | device_info | INTEGER | GPS 卫星数量 |
| `networkModel` | `network_model` | device_info | VARCHAR | 网络模式 |

---

## 3. 设备分组页面字段（右侧设备列表面板）

设备分组页面的设备列表是一个简化版本，仅显示关键字段：

| 前端字段 | 后端字段 | 数据来源 | 类型 | 说明 |
|----------|----------|----------|------|------|
| `id` | `id` | devices | UUID | 设备 ID（隐藏列，用于操作） |
| `connStatus` | `status` | devices | VARCHAR(20) | 连接状态 |
| `engStatus` | - | 计算字段 | VARCHAR | 工程状态（来自 project_status 映射） |
| `sn` | `serial_number` | devices | VARCHAR(64) | 设备序列号 |
| `name` | `site_name` | devices | VARCHAR(128) | 站点名称 |
| `macAddress` | `mac` | device_info | VARCHAR(64) | MAC 地址 |
| `groupName` | `group_name` | device_groups | VARCHAR(128) | 设备分组名称 |
| `longitude` | `longitude` | devices | DOUBLE PRECISION | 经度 |
| `latitude` | `latitude` | devices | DOUBLE PRECISION | 纬度 |
| `gpsHeight` | `height` | device_info | DECIMAL(10,2) | GPS 高度 |
| `offlineDays` | - | 计算字段 | INTEGER | 离线天数（当前时间 - last_offline_time） |
| `remark` | `remark` | device_info | TEXT | 备注 |

---

## 4. 前后端字段映射规则

### 4.1 命名转换

前端使用 **camelCase**，后端使用 **snake_case**。HTTP 拦截器自动转换：

```typescript
// 前端 → 后端
connStatus → conn_status → status
networkType → network_type → technology
groupName → group_name

// 后端 → 前端
serial_number → serialNumber → sn
rf_status → rfStatus
```

### 4.2 特殊映射

| 前端字段 | 前端值 | 后端字段 | 后端值 | 说明 |
|----------|--------|----------|--------|------|
| `networkType` | `eNB` | `technology` | `lte` | 前端显示名称到后端代码映射 |
| `networkType` | `gNB` | `technology` | `nr` | 前端显示名称到后端代码映射 |
| `connStatus` | `online` | `status` | `active` | 连接状态映射 |
| `connStatus` | `offline` | `status` | `offline`/`discovered`/`registered` | 多值映射 |

---

## 5. 字段变更触发器

### 5.1 TR069 参数同步（InfoSyncer）

设备扩展信息通过 `InfoSyncer` 从 TR069 参数自动同步：

1. 设备发送 Inform（含 ParameterList）
2. ACS 解析参数并更新 devices 表核心字段
3. InfoSyncer 根据运营商参数映射规则提取扩展字段
4. 写入/更新 device_info 表

**同步字段列表**（由 `GetInfoParamMapping(tech)` 定义）：
- `mac` / `hardware_version`
- `eci` / `pci` / `cell_id` / `freq_point`
- `bandwidth` / `transmit_power` / `plmn`
- `rf_status` / `cell_status` / `mme_status` / `sync_status`
- `gps_status` / `gps_satellite_count` / `height`
- `ue_count` / `run_time`

### 5.2 手动编辑

以下字段支持手动编辑：
- `remark`（备注）- 设备列表行内编辑
- `address`（安装地址）- 设备详情页
- `project_status`（项目状态）- 设备详情页
- `site_name`（站点名称）- 设备详情页
- `latitude` / `longitude`（经纬度）- 设备详情页

### 5.3 系统计算

以下字段由系统自动计算：
- `alarm_severity`（最高告警级别）- 告警模块关联计算
- `offline_days`（离线天数）- 前端实时计算
- `online_duration`（在线时长）- 后端根据 run_time 累计

---

## 6. 筛选条件映射

设备列表页面的筛选条件与后端查询参数映射：

| 前端筛选 | 前端参数名 | 后端参数名 | 数据库字段 | 说明 |
|----------|------------|------------|------------|------|
| 名称/序列号搜索 | `name` / `searchText` | `search` | `serial_number` / `site_name` | 模糊搜索 |
| 序列号 | `sn` | `sn` | `serial_number` | 精确搜索 |
| 厂商 | `vendor` | `oui` | `oui` | 精确匹配 |
| 基站制式 | `networkType` | `technology` | `technology` | 值映射：eNB→lte, gNB→nr |
| 设备分组 | `groupId` | `group_id` | `device_group_members.group_id` | 关联查询 |
| 连接状态 | `connStatus` | `status` | `status` | 值映射：1→active, 0→offline |
| 运营状态 | `opState` | `op_state` | `device_info.op_state` | 精确匹配 |
| 产品类型 | `productModel` | `product_class` | `product_class` | 精确匹配 |

---

## 7. 数据库表结构速查

### 7.1 devices 表（分区表，按 carrier 分区）

```sql
CREATE TABLE devices (
    id                     UUID PRIMARY KEY,
    serial_number          VARCHAR(64) NOT NULL UNIQUE,
    oui                    VARCHAR(6) NOT NULL,
    product_class          VARCHAR(64),
    manufacturer           VARCHAR(128),
    model_name             VARCHAR(128),
    carrier                VARCHAR(4) NOT NULL,      -- cmcc/ctcc/cucc
    technology             VARCHAR(3) NOT NULL,      -- lte/nr
    data_model_id          UUID,
    status                 VARCHAR(20) NOT NULL DEFAULT 'discovered',
    firmware_version       VARCHAR(64),
    ip_address             INET,
    connection_request_url VARCHAR(256),
    last_inform_at         TIMESTAMPTZ,
    last_inform_events     JSONB,
    inform_interval        INTEGER DEFAULT 300,
    site_name              VARCHAR(128),
    site_id                VARCHAR(64),
    latitude               DOUBLE PRECISION,
    longitude              DOUBLE PRECISION,
    extension_data         JSONB,
    created_at             TIMESTAMPTZ,
    updated_at             TIMESTAMPTZ
) PARTITION BY LIST (carrier);
```

### 7.2 device_info 表

```sql
CREATE TABLE device_info (
    device_id         UUID PRIMARY KEY REFERENCES devices(id),
    device_name       VARCHAR(128),
    address           VARCHAR(256),
    remark            TEXT,
    project_status    VARCHAR(20),
    height            DECIMAL(10,2),
    eci               VARCHAR(64),
    pci               VARCHAR(64),
    cell_id           VARCHAR(64),
    freq_point        VARCHAR(32),
    bandwidth         DECIMAL(8,2),
    transmit_power    DECIMAL(8,2),
    plmn              VARCHAR(32),
    rf_status         VARCHAR(20),
    cell_status       VARCHAR(20),
    mme_status        VARCHAR(20),
    sync_status       VARCHAR(32),
    kpi_status        VARCHAR(20),
    gps_status        VARCHAR(20),
    alarm_severity    VARCHAR(20),
    license_status    VARCHAR(20),
    op_state          VARCHAR(20),
    ue_count          INTEGER,
    mac               VARCHAR(64),
    hardware_version  VARCHAR(64),
    first_online_time TIMESTAMPTZ,
    last_offline_time TIMESTAMPTZ,
    run_time          BIGINT DEFAULT 0,
    num_of_cells      INTEGER DEFAULT 1,
    creator           VARCHAR(64),
    updater           VARCHAR(64),
    created_at        TIMESTAMPTZ,
    updated_at        TIMESTAMPTZ
);
```

### 7.3 device_groups / device_group_members 表

```sql
CREATE TABLE device_groups (
    id          UUID PRIMARY KEY,
    name        VARCHAR(128) NOT NULL,
    parent_id   UUID REFERENCES device_groups(id),
    remark      TEXT,
    is_default  BOOLEAN DEFAULT FALSE,
    level       INTEGER DEFAULT 1,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

CREATE TABLE device_group_members (
    id         UUID PRIMARY KEY,
    group_id   UUID REFERENCES device_groups(id),
    device_id  UUID REFERENCES devices(id),
    added_at   TIMESTAMPTZ,
    UNIQUE(group_id, device_id)
);
```

---

## 8. 相关文件索引

| 文件 | 说明 |
|------|------|
| `omcgo/migrations/000001_create_devices.up.sql` | devices 表结构 |
| `omcgo/migrations/000062_create_device_info.up.sql` | device_info 表结构 |
| `omcgo/migrations/000063_extend_device_info.up.sql` | device_info 扩展字段 |
| `omcgo/migrations/000008_create_device_groups.up.sql` | device_groups 表结构 |
| `omcgo/migrations/000009_create_device_group_members.up.sql` | device_group_members 表结构 |
| `omcgo/internal/device/device_info_dto.go` | DeviceWithInfo 结构体定义 |
| `omcgo/internal/device/device_info_pg_repository.go` | 设备列表查询 SQL |
| `omcmb/webcode/src/services/api/deviceApi.ts` | 前端 API 映射 |
| `omcmb/webcode/src/pages/device/DeviceList/index.tsx` | 设备列表页面 |
| `omcmb/webcode/src/pages/device/DeviceGrouping/DeviceListPanel.tsx` | 设备分组页面设备列表 |
