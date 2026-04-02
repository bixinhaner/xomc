# Device 模块重构方案

> 文档编号: 0003
> 创建日期: 2026-04-02
> 更新日期: 2026-04-02
> 状态: 待审批

---

## 目录

1. [问题分析](#1-问题分析)
2. [重构目标](#2-重构目标)
3. [后端重构方案](#3-后端重构方案)
4. [API 层同步调整](#4-api-层同步调整)
5. [前端同步调整](#5-前端同步调整)
6. [测试同步调整](#6-测试同步调整)
7. [执行计划](#7-执行计划)
8. [风险评估与验收标准](#8-风险评估与验收标准)
9. [附录](#附录)

---

## 1. 问题分析

### 1.1 当前文件命名问题

| 问题类型 | 当前文件 | 问题描述 |
|---------|---------|---------|
| 命名过于通用 | `repository.go` | 无法识别操作的是 `devices` 表 |
| 命名过于通用 | `handler.go` | 无法识别处理的是设备 CRUD |
| 命名过于通用 | `service.go` | 无法识别是设备核心服务 |
| 命名过于通用 | `types.go` | 包含 `BatchOperationResult` 等类型，命名不明确 |
| 前缀不一致 | `device_info_*.go` vs `pg_*.go` | 混合命名风格：实体前缀 vs 实现前缀 |
| 缩写歧义 | `column_config_*.go` | 应为 `user_column_config_*` 明确用户专属 |
| 注册相关歧义 | `registration_*.go` | 可能被误解为设备注册流程，实际是预注册功能 |

### 1.2 Model 定义问题

| 问题类型 | 当前情况 | 问题描述 |
|---------|---------|---------|
| 位置不统一 | `DeviceInfo` 在 `device_info_model.go`，`Device` 在 `internal/core/model/` | 核心实体位置不一致 |
| 字段无注释 | `DeviceInfo.RFStatus` 等状态字段无枚举说明 | 开发者不知道有哪些可能值 |
| 命名歧义 | `DeviceWithInfo.InfoAddress` JSON tag 是 `device_address` | 字段名与 JSON tag 不一致 |
| 文件混合 | `device_info_model.go` 包含多个不相关类型 | `DeviceInfo`, `UpdateDeviceInfoRequest`, `DeviceWithInfo`, `DeviceInfoSyncUpdate` 混在一起 |

### 1.3 状态枚举定义不完整

| 字段 | 当前类型 | 可能值 | 计算逻辑位置 |
|------|---------|--------|-------------|
| `rf_status` | `string` | on, off, error | `info_calc.go:CalcRFStatus` |
| `cell_status` | `string` | normal, inactive, fault, decommissioned | `info_calc.go:CalcCellStatus` |
| `mme_status` | `string` | connected, partial, disconnected | `info_calc.go:CalcMMEStatus` |
| `sync_status` | `string` | gps, beidou, ntp, error | `info_calc.go:CalcSyncStatus` |
| `gps_status` | `string` | normal, abnormal, no_signal | `info_calc.go:CalcGPSStatus` |
| `license_status` | `string` | active, expiring, expired | `info_calc.go:CalcLicenseStatus` |

### 1.4 影响范围分析

| 层级 | 涉及文件数 | 影响程度 | 说明 |
|------|-----------|---------|------|
| **后端 Model** | 5 | 高 | 实体定义变更，影响所有使用者 |
| **后端 Repository** | 10 | 高 | 文件重命名 + 合并 |
| **后端 Service** | 3 | 中 | 导入路径可能变化 |
| **后端 Handler** | 8 | 中 | 导入路径可能变化 |
| **后端测试** | 8 | 中 | 测试文件需要同步调整 |
| **前端 Types** | 2 | 低 | 类型定义保持兼容 |
| **前端 API** | 2 | 低 | API 调用保持兼容 |
| **前端 Hooks** | 1 | 低 | 无需修改 |
| **E2E 测试** | 1 | 低 | API 端点不变，测试无需修改 |

---

## 2. 重构目标

### 2.1 核心目标

| 目标 | 衡量标准 |
|------|---------|
| **文件命名规范化** | 一眼看出文件对应的数据库表和职责 |
| **Model 定义清晰化** | 核心实体统一位置，字段有完整注释 |
| **状态枚举类型化** | 所有状态字段有类型安全的常量定义 |
| **Repository 合并** | 消除 `pg_*.go` 文件，接口与实现合并 |
| **SQL 日志可控** | 通过配置控制 SQL 日志记录 |
| **前后端兼容** | API 接口不变，前端无需修改 |

### 2.2 不变项

以下内容保持不变，确保向后兼容：

| 项目 | 说明 |
|------|------|
| API 端点 | `/api/v1/devices/*` 路由不变 |
| JSON 字段名 | 响应体字段名不变（除了修正 `InfoAddress` → `Address`） |
| 数据库表结构 | 无 Schema 变更 |
| 前端 API 调用 | `deviceApi.ts` 方法签名不变 |

---

## 3. 后端重构方案

### 3.1 文件命名规范

采用 `{entity}_{role}.go` 模式：

```
{table_name}_{role}.go

entity: devices, device_info, device_registrations, device_parameters, user_column_configs
role:   model, repository, handler, service, types
```

### 3.2 文件重命名映射

| 当前文件 | 重命名后 | 操作 |
|---------|---------|------|
| `repository.go` | — | 合并到 `device_repository.go` |
| `pg_repository.go` | `device_repository.go` | 重命名 + 合并接口 |
| `handler.go` | `device_handler.go` | 重命名 |
| `service.go` | `device_service.go` | 重命名 |
| `types.go` | `device_types.go` | 重命名 |
| `device_info_model.go` | 拆分 | → `device_info_model.go` + `device_info_dto.go` + `device_info_update.go` |
| `registration_*.go` | `device_registration_*.go` | 统一前缀 |
| `param_repository.go` | `device_param_repository.go` | 统一前缀 |
| `column_config_*.go` | `user_column_config_*.go` | 明确用户级别 |

### 3.3 Repository 合并方案

**合并前**：
```
repository.go           (接口, ~123 行)
pg_repository.go        (实现, ~810 行)
```

**合并后**：
```
device_repository.go    (接口 + 实现, ~930 行)
```

**合并后文件结构**：
```go
// device_repository.go
package device

// ===== 接口定义 =====

type DeviceFilter struct { ... }
type DeviceReader interface { ... }
type DeviceWriter interface { ... }
type DeviceRepository interface { ... }

// ===== PostgreSQL 实现 =====

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type PgDeviceRepository struct {
    pool *pgxpool.Pool
}

func NewPgDeviceRepository(pool *pgxpool.Pool) *PgDeviceRepository { ... }
func (r *PgDeviceRepository) Create(ctx context.Context, device *model.Device) error { ... }
// ... 其他方法实现
```

### 3.4 Model 字段注释规范

为 `DeviceInfo` 添加完整字段注释：

```go
// DeviceInfo 存储设备的扩展运维信息。
// 对应数据库 device_info 表，与 devices 表一对一关系。
type DeviceInfo struct {
    DeviceID uuid.UUID `json:"device_id"`

    // ===== 运维标识（操作员手动填写）=====

    // DeviceName 设备名称，操作员自定义的易读名称
    DeviceName string `json:"device_name"`

    // Address 安装地址，详细地址描述
    Address string `json:"address"`

    // ===== 状态字段（从 TR069 参数计算）=====

    // RFStatus 射频状态
    // 可能值: RFStatusOn("on"), RFStatusOff("off"), RFStatusError("error")
    // 计算逻辑: CalcRFStatus()
    RFStatus string `json:"rf_status"`

    // CellStatus 小区状态
    // 可能值: CellStatusNormal, CellStatusInactive, CellStatusFault, CellStatusDecommissioned
    CellStatus string `json:"cell_status"`

    // ... 其他字段类似
}
```

### 3.5 状态枚举类型化

新增 `device_status_types.go`：

```go
package device

// RFStatus 射频状态
type RFStatus string

const (
    RFStatusOn    RFStatus = "on"
    RFStatusOff   RFStatus = "off"
    RFStatusError RFStatus = "error"
)

func (s RFStatus) IsValid() bool {
    switch s {
    case RFStatusOn, RFStatusOff, RFStatusError:
        return true
    }
    return false
}

// CellStatus 小区状态
type CellStatus string

const (
    CellStatusNormal         CellStatus = "normal"
    CellStatusInactive       CellStatus = "inactive"
    CellStatusFault          CellStatus = "fault"
    CellStatusDecommissioned CellStatus = "decommissioned"
)

// MMEStatus, SyncStatus, GPSStatus, LicenseStatus 类似...
```

### 3.6 SQL 日志记录方案

通过 pgx Tracer + Zap 实现：

**配置**：
```yaml
database:
  postgres:
    sql_log:
      enabled: false              # 开关
      level: debug                # 日志级别阈值
      slow_query_threshold_ms: 1000  # 慢查询阈值
      include_params: true        # 是否记录参数
```

**实现要点**：
- 实现 `pgx.QueryTracer` 接口
- 慢查询始终记录（无论日志级别）
- 支持动态开关（`kill -USR1 <pid>`）
- 生产环境默认关闭，`include_params: false`

---

## 4. API 层同步调整

### 4.1 当前 API 结构

| 文件 | 职责 | 调整内容 |
|------|------|---------|
| `handler.go` → `device_handler.go` | 设备 CRUD | 重命名 |
| `device_info_handler.go` | DeviceInfo 更新 | 无需修改 |
| `registration_handler.go` → `device_registration_handler.go` | 预注册 | 重命名 |
| `param_handler.go` → `device_param_handler.go` | 参数管理 | 重命名 |
| `column_config_handler.go` → `user_column_config_handler.go` | 列配置 | 重命名 |
| `export_handler.go` | 导出 | 无需修改 |
| `inform_handler.go` | ACS Inform | 无需修改 |

### 4.2 API 端点（不变）

| 方法 | 端点 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/v1/devices` | `device_handler.go` | 设备列表 |
| GET | `/api/v1/devices/:id` | `device_handler.go` | 设备详情 |
| POST | `/api/v1/devices` | `device_handler.go` | 创建设备 |
| PUT | `/api/v1/devices/:id` | `device_handler.go` | 更新设备 |
| DELETE | `/api/v1/devices/:id` | `device_handler.go` | 删除设备 |
| POST | `/api/v1/devices/batch-delete` | `device_handler.go` | 批量删除 |
| GET | `/api/v1/devices/:id/parameters` | `device_param_handler.go` | 参数列表 |
| PUT | `/api/v1/devices/:id/info` | `device_info_handler.go` | 更新信息 |
| POST | `/api/v1/registrations` | `device_registration_handler.go` | 预注册 |

### 4.3 Handler 导入调整

**调整前**：
```go
// router.go
import "github.com/omcgo/omcgo/internal/device"

deviceHandler := device.NewHandler(deviceService)
```

**调整后**（无变化，包名不变）：
```go
// router.go
import "github.com/omcgo/omcgo/internal/device"

deviceHandler := device.NewHandler(deviceService)
```

> **注意**：Go 包名（`package device`）不变，仅文件名变化，import 路径无需修改。

---

## 5. 前端同步调整

### 5.1 前端文件结构

```
omcmb/webcode/src/
├── types/
│   ├── device.ts              # 设备类型定义（无需修改）
│   └── deviceParameter.ts     # 参数类型定义（无需修改）
├── services/api/
│   ├── deviceApi.ts           # 设备 API（无需修改）
│   └── deviceParameterApi.ts  # 参数 API（无需修改）
├── hooks/api/
│   ├── useDevices.ts          # 设备 Hooks（无需修改）
│   └── useDeviceParameters.ts # 参数 Hooks（无需修改）
├── mock/
│   ├── data/devices.ts        # Mock 数据（无需修改）
│   └── services/deviceService.ts # Mock 服务（无需修改）
└── pages/device/
    ├── DeviceList/             # 设备列表页
    ├── DeviceDetail/           # 设备详情页
    ├── DeviceGrouping/         # 设备分组
    ├── DeviceRegistration/     # 预注册
    └── ...                     # 其他页面
```

### 5.2 类型定义调整（可选优化）

**当前 `device.ts` 类型定义**：
```typescript
// Backend 响应类型
interface BackendDevice {
  id: string;
  serial_number: string;
  // ...
  rf_status?: string;  // 可选优化：改为联合类型
  cell_status?: string;
}
```

**优化后（可选）**：
```typescript
// 状态类型定义（与后端枚举对应）
export type RFStatus = 'on' | 'off' | 'error';
export type CellStatus = 'normal' | 'inactive' | 'fault' | 'decommissioned';
export type MMEStatus = 'connected' | 'partial' | 'disconnected';

interface BackendDevice {
  id: string;
  serial_number: string;
  // ...
  rf_status?: RFStatus;
  cell_status?: CellStatus;
  mme_status?: MMEStatus;
}
```

### 5.3 状态枚举常量（新增，可选）

```typescript
// types/deviceStatus.ts
export const RF_STATUS = {
  ON: 'on',
  OFF: 'off',
  ERROR: 'error',
} as const;

export const CELL_STATUS = {
  NORMAL: 'normal',
  INACTIVE: 'inactive',
  FAULT: 'fault',
  DECOMMISSIONED: 'decommissioned',
} as const;

// 状态显示映射
export const RF_STATUS_LABELS: Record<RFStatus, string> = {
  on: '开启',
  off: '关闭',
  error: '故障',
};
```

### 5.4 前端影响总结

| 组件 | 是否需要修改 | 说明 |
|------|-------------|------|
| `deviceApi.ts` | ❌ 否 | API 端点不变 |
| `useDevices.ts` | ❌ 否 | Hook 逻辑不变 |
| `device.ts` (types) | ⚠️ 可选 | 可添加状态类型，但不强制 |
| 页面组件 | ❌ 否 | 渲染逻辑不变 |
| Mock 服务 | ❌ 否 | 数据结构不变 |

---

## 6. 测试同步调整

### 6.1 后端测试文件

| 测试文件 | 调整内容 |
|---------|---------|
| `handler_test.go` → `device_handler_test.go` | 重命名 |
| `inform_handler_test.go` | 无需修改 |
| `service_test.go` → `device_service_test.go` | 重命名 |
| `*_test.go` | 导入路径可能需要调整 |

### 6.2 E2E 测试

E2E 测试脚本 `scripts/e2e_verify.sh` **无需修改**：
- API 端点不变
- 请求/响应格式不变
- 测试数据不变

### 6.3 测试覆盖要点

| 测试类型 | 覆盖内容 |
|---------|---------|
| 单元测试 | 状态枚举 `IsValid()` 方法 |
| 单元测试 | `Calc*Status()` 函数返回类型 |
| 集成测试 | Repository CRUD 操作 |
| E2E 测试 | 设备列表、详情、更新接口 |

---

## 7. 执行计划

### Phase 1: 状态枚举定义（低风险）

**预计时间**：1 小时

```bash
# 1. 创建状态类型文件
touch omcgo/internal/device/device_status_types.go

# 2. 定义所有状态枚举
# - RFStatus
# - CellStatus
# - MMEStatus
# - SyncStatus
# - GPSStatus
# - LicenseStatus

# 3. 编译验证
cd omcgo && go build ./...

# 4. 运行测试
go test ./internal/device/...
```

### Phase 2: Repository 文件合并（中等风险）

**预计时间**：2 小时

```bash
# 1. 合并 device_repository
cd omcgo/internal/device

# 备份
cp repository.go repository.go.bak
cp pg_repository.go pg_repository.go.bak

# 合并：将 repository.go 的接口定义移动到 pg_repository.go 顶部
# 然后重命名为 device_repository.go

# 2. 编译验证
go build ./...

# 3. 对其他 repository 重复相同操作
# - device_info_repository.go
# - device_registration_repository.go
# - device_param_repository.go
# - user_column_config_repository.go
```

### Phase 3: 文件重命名（低风险）

**预计时间**：1 小时

```bash
cd omcgo/internal/device

# Handler/Service 重命名
git mv handler.go device_handler.go
git mv service.go device_service.go
git mv types.go device_types.go

# Registration 相关
git mv registration_model.go device_registration_model.go
git mv registration_handler.go device_registration_handler.go
git mv registration_service.go device_registration_service.go

# Param 相关
git mv param_handler.go device_param_handler.go

# Column Config 相关
git mv column_config_model.go user_column_config_model.go
git mv column_config_handler.go user_column_config_handler.go

# Info 相关
git mv info_calc.go device_info_calc.go

# 编译验证
go build ./...
```

### Phase 4: Model 拆分与注释（低风险）

**预计时间**：2 小时

```bash
# 1. 创建新文件
touch omcgo/internal/device/device_info_dto.go
touch omcgo/internal/device/device_info_update.go

# 2. 迁移类型
# - DeviceWithInfo → device_info_dto.go
# - UpdateDeviceInfoRequest → device_info_update.go
# - DeviceInfoSyncUpdate → device_info_update.go

# 3. 为 DeviceInfo 添加字段注释

# 4. 编译验证
go build ./...
```

### Phase 5: SQL 日志实现（中等风险）

**预计时间**：2 小时

```bash
# 1. 创建 Tracer 实现
mkdir -p omcgo/internal/components/postgres
touch omcgo/internal/components/postgres/tracer.go

# 2. 添加配置结构
# 编辑 omcgo/internal/appconfig/config.go

# 3. 集成到连接池
# 编辑 omcgo/internal/components/postgres/pool.go

# 4. 编写测试
touch omcgo/internal/components/postgres/tracer_test.go

# 5. 编译验证
go build ./...
go test ./internal/components/postgres/...
```

### Phase 6: 验证与清理

**预计时间**：1 小时

```bash
# 1. 完整测试
cd omcgo
make test

# 2. E2E 测试
bash scripts/e2e_verify.sh http://localhost:8080

# 3. Lint 检查
make lint

# 4. 清理备份文件
rm internal/device/*.bak

# 5. 提交
git add .
git commit -m "refactor(device): 重构 device 模块 - 文件命名规范化 + Repository 合并 + 状态枚举类型化 + SQL 日志"
```

---

## 8. 风险评估与验收标准

### 8.1 风险评估

| 风险 | 级别 | 缓解措施 |
|------|------|---------|
| 文件重命名导致 IDE 索引失效 | 低 | 重新打开项目或刷新索引 |
| Repository 合并导致文件过大 | 低 | 800-1000 行在可接受范围内 |
| 遗漏文件引用更新 | 低 | 编译器会报错，逐个修复 |
| JSON 字段名变化影响前端 | 中 | `InfoAddress` → `Address` 需同步前端 |
| SQL 日志影响性能 | 低 | 可配置关闭，生产默认关闭 |
| SQL 日志泄露敏感数据 | 中 | 生产环境 `include_params: false` |

### 8.2 验收标准

#### 后端

- [ ] 所有文件命名遵循 `{entity}_{role}.go` 模式
- [ ] 无 `pg_*.go` 前缀的独立实现文件（已合并）
- [ ] 所有 Model 字段有中文注释说明
- [ ] 所有状态字段有对应的类型安全常量
- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 测试通过
- [ ] `make lint` 无新增警告

#### 前端

- [ ] API 调用正常（端点不变）
- [ ] 页面渲染正常（数据结构不变）
- [ ] 类型检查通过 `npx tsc --noEmit`

#### 集成

- [ ] E2E 测试通过
- [ ] SQL 日志功能正常
- [ ] SQL 日志可通过配置开关控制

---

## 附录

### 附录 A: 状态枚举完整定义

| 状态字段 | 类型 | 可能值 | 含义 | 计算来源 |
|---------|------|--------|------|---------|
| `DeviceStatus` | `global.DeviceStatus` | discovered, registered, provisioning, active, maintenance, offline, decommissioned | 设备生命周期状态 | 状态机转换 |
| `RFStatus` | `device.RFStatus` | on, off, error | 射频发射状态 | FAPControl.AdminState + RFTxStatus |
| `CellStatus` | `device.CellStatus` | normal, inactive, fault, decommissioned | 小区服务状态 | AdminState + OpState + CellOpState |
| `MMEStatus` | `device.MMEStatus` | connected, partial, disconnected | MME 池连接状态 | MmePoolConfigParam.*.MME1Status |
| `SyncStatus` | `device.SyncStatus` | gps, beidou, ntp, error | 时钟同步源状态 | GPS/BDS/1588 Status + tfcsSyncState |
| `GPSStatus` | `device.GPSStatus` | normal, abnormal, no_signal | GPS 定位状态 | X_COM_GPS_Status |
| `LicenseStatus` | `device.LicenseStatus` | active, expiring, expired | 许可证状态 | X_COM_LICENSE.Capacity.*.State |
| `RegistrationStatus` | `global.RegistrationStatus` | pending, online, expired | 预注册状态 | 首次 Inform / 超时 |

### 附录 B: 表与 Model 对应关系

| 数据库表 | Model 文件 | Repository 文件 |
|---------|-----------|----------------|
| `devices` | `internal/core/model/device.go` | `device_repository.go` |
| `device_info` | `device_info_model.go` | `device_info_repository.go` |
| `device_registrations` | `device_registration_model.go` | `device_registration_repository.go` |
| `device_parameters` | `internal/core/model/parameter.go` | `device_param_repository.go` |
| `user_column_configs` | `user_column_config_model.go` | `user_column_config_repository.go` |

### 附录 C: 决策总结

| 需求 | 推荐方案 | 理由 |
|------|---------|------|
| MySQL/PostgreSQL 切换 | **Dialect 抽象层** | 保持 TimescaleDB 兼容，避免 ORM 开销 |
| 消除 `pg_*.go` 文件 | **接口 + 实现合并** | 减少文件碎片，逻辑内聚 |
| ORM 替换 pgx | **不采用** | TimescaleDB 不兼容，违反架构决策 |
| SQL 日志记录 | **pgx Tracer + Zap** | 原生支持，性能好，与日志系统集成 |

### 附录 D: SQL 日志配置速查

| 环境 | enabled | level | slow_query_threshold_ms | include_params |
|------|---------|-------|------------------------|----------------|
| **开发** | `true` | `debug` | `500` | `true` |
| **测试** | `true` | `debug` | `1000` | `true` |
| **生产** | `false` | — | `1000` | `false` |

### 附录 E: 数据库表字段注释方案

#### E.1 概述

为所有设备相关表添加 PostgreSQL `COMMENT ON COLUMN` 语句，使通用数据库工具（DBeaver、pgAdmin、DataGrip 等）可直接查看字段说明。

#### E.2 实施方式

创建迁移文件 `migrations/000070_add_device_table_comments.up.sql`：

```sql
-- ============================================================
-- 000070_add_device_table_comments.up.sql
-- 为设备相关表添加字段注释，便于数据库工具查看
-- ============================================================

-- ===== devices 表 =====
COMMENT ON TABLE devices IS '设备核心表：存储 TR069 设备身份与连接信息';

COMMENT ON COLUMN devices.id IS '设备唯一标识 (UUID)';
COMMENT ON COLUMN devices.serial_number IS '设备序列号 (SN)，全局唯一';
COMMENT ON COLUMN devices.oui IS '组织唯一标识符 (Organizationally Unique Identifier)，厂商标识';
COMMENT ON COLUMN devices.product_class IS '产品型号，如 "AirScale ASN01"';
COMMENT ON COLUMN devices.manufacturer IS '厂商名称，如 "Nokia"、"华为"';
COMMENT ON COLUMN devices.model_name IS '设备型号名称';
COMMENT ON COLUMN devices.carrier IS '运营商代码：cmcc(移动)/ctcc(电信)/cucc(联通)';
COMMENT ON COLUMN devices.technology IS '无线制式：lte(4G)/nr(5G)';
COMMENT ON COLUMN devices.data_model_id IS '关联的数据模型定义 ID';
COMMENT ON COLUMN devices.status IS '设备生命周期状态：discovered/registered/provisioning/active/maintenance/offline/decommissioned';
COMMENT ON COLUMN devices.firmware_version IS '当前固件版本';
COMMENT ON COLUMN devices.ip_address IS '设备 IP 地址 (INET 类型)';
COMMENT ON COLUMN devices.connection_request_url IS 'TR069 Connection Request URL，用于反向唤醒设备';
COMMENT ON COLUMN devices.last_inform_at IS '最后一次 Inform 时间';
COMMENT ON COLUMN devices.last_inform_events IS '最后一次 Inform 事件码列表 (JSONB)：["2 PERIODIC", "6 CONNECTION REQUEST"]';
COMMENT ON COLUMN devices.inform_interval IS 'Inform 心跳间隔 (秒)，默认 300';
COMMENT ON COLUMN devices.site_name IS '站点名称';
COMMENT ON COLUMN devices.site_id IS '站点 ID';
COMMENT ON COLUMN devices.latitude IS '纬度';
COMMENT ON COLUMN devices.longitude IS '经度';
COMMENT ON COLUMN devices.extension_data IS '扩展数据 (JSONB)，存储运营商定制字段';
COMMENT ON COLUMN devices.created_at IS '创建时间';
COMMENT ON COLUMN devices.updated_at IS '最后更新时间';
COMMENT ON COLUMN devices.deleted_at IS '软删除时间 (NULL 表示未删除)';

-- ===== device_info 表 =====
COMMENT ON TABLE device_info IS '设备运维扩展信息：运维人员手动填写 + TR069 参数自动同步';

COMMENT ON COLUMN device_info.device_id IS '关联 devices.id，1:1 关系';
COMMENT ON COLUMN device_info.device_name IS '设备名称，运维人员自定义';
COMMENT ON COLUMN device_info.address IS '安装地址';
COMMENT ON COLUMN device_info.remark IS '备注信息';
COMMENT ON COLUMN device_info.project_status IS '项目状态：building(在建)/delivered(交付)/operating(商用)/deactivated(退服)';
COMMENT ON COLUMN device_info.height IS '安装高度 (米)';

-- 无线参数
COMMENT ON COLUMN device_info.eci IS 'E-UTRAN 小区标识符 (28位)：PLMN + CellID';
COMMENT ON COLUMN device_info.pci IS '物理小区标识 (Physical Cell ID)：LTE(0-503)/NR(0-1007)';
COMMENT ON COLUMN device_info.cell_id IS '逻辑小区 ID';
COMMENT ON COLUMN device_info.freq_point IS '频点号：LTE(EARFCN)/NR(NRARFCN)';
COMMENT ON COLUMN device_info.bandwidth IS '载波带宽 (MHz)';
COMMENT ON COLUMN device_info.transmit_power IS '发射功率 (dBm)';
COMMENT ON COLUMN device_info.plmn IS '公众陆地移动网标识：MCC+MNC，如 46000';

-- 状态字段
COMMENT ON COLUMN device_info.rf_status IS '射频状态：on(开启)/off(关闭)/error(故障)';
COMMENT ON COLUMN device_info.cell_status IS '小区状态：normal(正常)/inactive(未激活)/fault(故障)/decommissioned(退服)';
COMMENT ON COLUMN device_info.mme_status IS 'MME 连接状态：connected(已连接)/partial(部分连接)/disconnected(断开)';
COMMENT ON COLUMN device_info.sync_status IS '时钟同步源：gps(北斗)/beidou(北斗)/ntp(网络)/error(失步)';
COMMENT ON COLUMN device_info.kpi_status IS 'KPI 状态 (保留)';
COMMENT ON COLUMN device_info.gps_status IS 'GPS 状态：normal(正常)/abnormal(异常)/no_signal(无信号)';
COMMENT ON COLUMN device_info.license_status IS '许可证状态：active(有效)/expiring(即将过期)/expired(已过期)';
COMMENT ON COLUMN device_info.alarm_severity IS '当前最高告警级别：critical/major/minor/warning';

-- 硬件信息
COMMENT ON COLUMN device_info.mac IS 'MAC 地址';
COMMENT ON COLUMN device_info.hardware_version IS '硬件版本号';

-- 时间信息
COMMENT ON COLUMN device_info.first_online_time IS '首次上线时间';
COMMENT ON COLUMN device_info.last_offline_time IS '最后离线时间';
COMMENT ON COLUMN device_info.run_time IS '累计运行时长 (秒)';

-- 审计
COMMENT ON COLUMN device_info.creator IS '创建人';
COMMENT ON COLUMN device_info.updater IS '最后更新人';
COMMENT ON COLUMN device_info.created_at IS '创建时间';
COMMENT ON COLUMN device_info.updated_at IS '最后更新时间';

-- ===== device_parameters 表 =====
COMMENT ON TABLE device_parameters IS 'TR069 参数值存储：每设备每参数一条记录';

COMMENT ON COLUMN device_parameters.device_id IS '关联 devices.id';
COMMENT ON COLUMN device_parameters.parameter_path IS 'TR069 参数路径，如 Device.FAPControl.RFTxStatus';
COMMENT ON COLUMN device_parameters.parameter_value IS '参数值';
COMMENT ON COLUMN device_parameters.parameter_type IS '参数类型：string/int/unsignedInt/boolean/dateTime';
COMMENT ON COLUMN device_parameters.writable IS '是否可写';
COMMENT ON COLUMN device_parameters.last_updated_at IS '最后更新时间';

-- ===== device_registrations 表 =====
COMMENT ON TABLE device_registrations IS '设备预注册：批量导入待上线设备的 SN 及规划信息';

COMMENT ON COLUMN device_registrations.id IS '预注册记录 ID';
COMMENT ON COLUMN device_registrations.serial_number IS '设备序列号 (SN)';
COMMENT ON COLUMN device_registrations.group_id IS '预分配的设备组 ID';
COMMENT ON COLUMN device_registrations.device_id IS '实际上线后关联的设备 ID';
COMMENT ON COLUMN device_registrations.status IS '状态：pending(待上线)/online(已上线)/expired(已过期)';

-- 规划信息
COMMENT ON COLUMN device_registrations.site_name IS '规划站点名称';
COMMENT ON COLUMN device_registrations.device_name IS '规划设备名称';
COMMENT ON COLUMN device_registrations.longitude IS '规划经度';
COMMENT ON COLUMN device_registrations.latitude IS '规划纬度';
COMMENT ON COLUMN device_registrations.height IS '规划安装高度 (米)';
COMMENT ON COLUMN device_registrations.azimuth IS '水平方位角 [0-359°]';
COMMENT ON COLUMN device_registrations.tilt_angle IS '机械下倾角 [0-9°]';
COMMENT ON COLUMN device_registrations.beam_width IS '垂直3dB波束宽度 [1-9]';
COMMENT ON COLUMN device_registrations.remark IS '备注';

-- 审计
COMMENT ON COLUMN device_registrations.created_by IS '创建人';
COMMENT ON COLUMN device_registrations.import_batch_id IS '批量导入批次 ID';
COMMENT ON COLUMN device_registrations.created_at IS '创建时间';
COMMENT ON COLUMN device_registrations.updated_at IS '最后更新时间';

-- ===== device_groups 表 =====
COMMENT ON TABLE device_groups IS '设备分组：树形拓扑管理';

COMMENT ON COLUMN device_groups.id IS '分组 ID';
COMMENT ON COLUMN device_groups.name IS '分组名称';
COMMENT ON COLUMN device_groups.parent_id IS '父分组 ID (NULL 表示根分组)';
COMMENT ON COLUMN device_groups.carrier IS '所属运营商 (NULL 表示通用分组)';
COMMENT ON COLUMN device_groups.description IS '分组描述';
COMMENT ON COLUMN device_groups.sort_order IS '排序序号';
COMMENT ON COLUMN device_groups.created_at IS '创建时间';
COMMENT ON COLUMN device_groups.updated_at IS '最后更新时间';

-- ===== device_group_members 表 =====
COMMENT ON TABLE device_group_members IS '设备分组成员：设备与分组的多对多关系';

COMMENT ON COLUMN device_group_members.group_id IS '分组 ID';
COMMENT ON COLUMN device_group_members.device_id IS '设备 ID';
COMMENT ON COLUMN device_group_members.added_at IS '加入时间';

-- ===== user_column_configs 表 =====
COMMENT ON TABLE user_column_configs IS '用户自定义列配置：列表页面列显示/隐藏/排序';

COMMENT ON COLUMN user_column_configs.user_id IS '用户 ID';
COMMENT ON COLUMN user_column_configs.page_key IS '页面标识：device_list/alarm_list/pm_list 等';
COMMENT ON COLUMN user_column_configs.columns IS '列配置 JSON 数组：[{"key":"sn","visible":true,"width":120}]';
COMMENT ON COLUMN user_column_configs.created_at IS '创建时间';
COMMENT ON COLUMN user_column_configs.updated_at IS '最后更新时间';
```

#### E.3 表字段完整说明

##### devices 表

| 字段 | 类型 | 说明 | 示例值 |
|------|------|------|--------|
| `id` | UUID | 设备唯一标识 | `550e8400-e29b-41d4-a716-446655440000` |
| `serial_number` | VARCHAR(64) | 设备序列号，全局唯一 | `SN123456789` |
| `oui` | VARCHAR(6) | 组织唯一标识符 (OUI) | `00A1B2` |
| `product_class` | VARCHAR(64) | 产品型号 | `AirScale ASN01` |
| `manufacturer` | VARCHAR(128) | 厂商名称 | `Nokia` |
| `model_name` | VARCHAR(128) | 设备型号名称 | `ASN01-1900` |
| `carrier` | VARCHAR(4) | 运营商代码 | `cmcc`/`ctcc`/`cucc` |
| `technology` | VARCHAR(3) | 无线制式 | `lte`/`nr` |
| `data_model_id` | UUID | 数据模型定义 ID | |
| `status` | VARCHAR(20) | 生命周期状态 | `active` |
| `firmware_version` | VARCHAR(64) | 固件版本 | `1.2.3-build456` |
| `ip_address` | INET | 设备 IP 地址 | `192.168.1.100` |
| `connection_request_url` | VARCHAR(256) | TR069 反向唤醒 URL | `http://...` |
| `last_inform_at` | TIMESTAMPTZ | 最后 Inform 时间 | |
| `last_inform_events` | JSONB | Inform 事件码列表 | `["2 PERIODIC"]` |
| `inform_interval` | INTEGER | 心跳间隔(秒) | `300` |
| `site_name` | VARCHAR(128) | 站点名称 | `北京朝阳站点` |
| `site_id` | VARCHAR(64) | 站点 ID | `BJCY001` |
| `latitude` | DOUBLE | 纬度 | `39.9042` |
| `longitude` | DOUBLE | 经度 | `116.4074` |
| `extension_data` | JSONB | 扩展数据 | `{}` |
| `created_at` | TIMESTAMPTZ | 创建时间 | |
| `updated_at` | TIMESTAMPTZ | 更新时间 | |
| `deleted_at` | TIMESTAMPTZ | 软删除时间 | |

##### device_info 表

| 字段 | 类型 | 说明 | 可能值 |
|------|------|------|--------|
| `device_id` | UUID | 关联 devices.id | |
| `device_name` | VARCHAR(128) | 设备名称（手动填写） | `朝阳1号基站` |
| `address` | VARCHAR(256) | 安装地址 | `北京市朝阳区...` |
| `remark` | TEXT | 备注 | |
| `project_status` | VARCHAR(20) | 项目状态 | `building`/`delivered`/`operating`/`deactivated` |
| `height` | DECIMAL(10,2) | 安装高度(米) | `15.5` |
| `eci` | VARCHAR(64) | E-UTRAN 小区标识 | `46000-12345` |
| `pci` | VARCHAR(64) | 物理小区标识 | `100` |
| `cell_id` | VARCHAR(64) | 逻辑小区 ID | `1` |
| `freq_point` | VARCHAR(32) | 频点号 | `38050` |
| `bandwidth` | DECIMAL(8,2) | 带宽(MHz) | `20.0` |
| `transmit_power` | DECIMAL(8,2) | 发射功率(dBm) | `43.0` |
| `plmn` | VARCHAR(32) | PLMN 标识 | `46000` |
| `rf_status` | VARCHAR(20) | 射频状态 | `on`/`off`/`error` |
| `cell_status` | VARCHAR(20) | 小区状态 | `normal`/`inactive`/`fault`/`decommissioned` |
| `mme_status` | VARCHAR(20) | MME 连接状态 | `connected`/`partial`/`disconnected` |
| `sync_status` | VARCHAR(32) | 时钟同步源 | `gps`/`beidou`/`ntp`/`error` |
| `kpi_status` | VARCHAR(20) | KPI 状态 | |
| `gps_status` | VARCHAR(20) | GPS 状态 | `normal`/`abnormal`/`no_signal` |
| `license_status` | VARCHAR(20) | 许可证状态 | `active`/`expiring`/`expired` |
| `alarm_severity` | VARCHAR(20) | 最高告警级别 | `critical`/`major`/`minor`/`warning` |
| `mac` | VARCHAR(64) | MAC 地址 | `00:1A:2B:3C:4D:5E` |
| `hardware_version` | VARCHAR(64) | 硬件版本 | `v2.0` |
| `first_online_time` | TIMESTAMPTZ | 首次上线时间 | |
| `last_offline_time` | TIMESTAMPTZ | 最后离线时间 | |
| `run_time` | BIGINT | 累计运行时长(秒) | `86400` |
| `creator` | VARCHAR(64) | 创建人 | `admin` |
| `updater` | VARCHAR(64) | 更新人 | `admin` |

##### device_registrations 表

| 字段 | 类型 | 说明 | 可能值 |
|------|------|------|--------|
| `id` | UUID | 预注册记录 ID | |
| `serial_number` | VARCHAR(64) | 设备序列号 | `SN123456789` |
| `group_id` | UUID | 预分配设备组 ID | |
| `device_id` | UUID | 上线后关联设备 ID | |
| `status` | VARCHAR(16) | 状态 | `pending`/`online`/`expired` |
| `site_name` | VARCHAR(128) | 规划站点名称 | |
| `device_name` | VARCHAR(128) | 规划设备名称 | |
| `longitude` | DOUBLE | 规划经度 | |
| `latitude` | DOUBLE | 规划纬度 | |
| `height` | DECIMAL(10,2) | 规划高度(米) | |
| `azimuth` | SMALLINT | 水平方位角 | `0-359` |
| `tilt_angle` | SMALLINT | 机械下倾角 | `0-9` |
| `beam_width` | SMALLINT | 垂直波束宽度 | `1-9` |
| `remark` | TEXT | 备注 | |
| `created_by` | VARCHAR(64) | 创建人 | |
| `import_batch_id` | UUID | 批量导入批次 ID | |

##### device_parameters 表

| 字段 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `device_id` | UUID | 设备 ID | |
| `parameter_path` | VARCHAR(512) | TR069 参数路径 | `Device.FAPControl.RFTxStatus` |
| `parameter_value` | TEXT | 参数值 | `on` |
| `parameter_type` | VARCHAR(20) | 参数类型 | `string`/`int`/`boolean` |
| `writable` | BOOLEAN | 是否可写 | `true` |
| `last_updated_at` | TIMESTAMPTZ | 最后更新时间 | |

##### device_groups 表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID | 分组 ID |
| `name` | VARCHAR(128) | 分组名称 |
| `parent_id` | UUID | 父分组 ID |
| `carrier` | VARCHAR(4) | 所属运营商 |
| `description` | TEXT | 分组描述 |
| `sort_order` | INTEGER | 排序序号 |

##### user_column_configs 表

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | UUID | 用户 ID |
| `page_key` | VARCHAR(64) | 页面标识 |
| `columns` | JSONB | 列配置 |

#### E.4 状态字段值速查

| 状态字段 | 可能值 | 中文含义 | 计算来源 |
|---------|--------|---------|---------|
| `status` (devices) | discovered | 已发现 | 首次 Inform |
| | registered | 已注册 | 预注册匹配 |
| | provisioning | 配置中 | 配置下发 |
| | active | 活跃 | 正常运行 |
| | maintenance | 维护中 | 手动设置 |
| | offline | 离线 | 心跳超时 |
| | decommissioned | 已退服 | 手动退服 |
| `rf_status` | on | 射频开启 | FAPControl.AdminState=1 |
| | off | 射频关闭 | FAPControl.AdminState=0 |
| | error | 射频故障 | RFTxStatus=fault |
| `cell_status` | normal | 正常服务 | AdminState=1 & OpState=2 |
| | inactive | 未激活 | AdminState=0 |
| | fault | 故障 | OpState=3 |
| | decommissioned | 已退服 | 手动设置 |
| `mme_status` | connected | 已连接 | 2+ MME 连接正常 |
| | partial | 部分连接 | 仅 1 个 MME 连接 |
| | disconnected | 断开 | 无 MME 连接 |
| `sync_status` | gps | GPS 同步 | GPS 锁定 |
| | beidou | 北斗同步 | 北斗锁定 |
| | ntp | 网络同步 | NTP/1588v2 |
| | error | 失步 | 无同步源 |
| `gps_status` | normal | GPS 正常 | X_COM_GPS_Status=1 |
| | abnormal | GPS 异常 | X_COM_GPS_Status=2 |
| | no_signal | 无信号 | X_COM_GPS_Status=0 |
| `license_status` | active | 有效 | RemainingPeriod > 30 |
| | expiring | 即将过期 | RemainingPeriod ≤ 30 |
| | expired | 已过期 | RemainingPeriod ≤ 0 |
| **生产排障** | `true` | `info` | `1000` | `false` |

### 附录 F: 重构后目录结构

```
internal/device/
├── device_handler.go              # 设备 CRUD 处理器
├── device_service.go              # 设备核心业务逻辑
├── device_repository.go           # ✅ 接口 + PostgreSQL 实现（合并后）
├── device_types.go                # 设备相关类型
├── device_status_types.go         # ✅ 状态枚举类型定义（新增）
│
├── device_info_model.go           # DeviceInfo 实体
├── device_info_dto.go             # ✅ DeviceWithInfo 等 DTO（新增）
├── device_info_update.go          # ✅ UpdateDeviceInfoRequest 等（新增）
├── device_info_repository.go      # ✅ 接口 + 实现（合并后）
├── device_info_handler.go         # DeviceInfo 处理器
├── device_info_sync.go            # TR069 参数同步
├── device_info_calc.go            # 状态计算函数
│
├── device_registration_model.go   # DeviceRegistration 实体 + DTO
├── device_registration_repository.go  # ✅ 接口 + 实现（合并后）
├── device_registration_handler.go
├── device_registration_service.go
│
├── device_param_repository.go     # ✅ 接口 + 实现（合并后）
├── device_param_handler.go        # 参数管理处理器
├── param_classify.go              # 参数分类
│
├── user_column_config_model.go    # ColumnConfig 实体
├── user_column_config_repository.go  # ✅ 接口 + 实现（合并后）
├── user_column_config_handler.go
│
├── detail_dto.go                  # 详情页 DTO
├── detail_assembler.go            # 详情组装器
├── state_machine.go               # 设备状态机
├── heartbeat.go                   # 心跳监控
├── device_cache.go                # 设备缓存
├── batch_processor.go             # 批量处理
├── metrics.go                     # Prometheus 指标
├── query_tier.go                  # 查询分层
├── inform_handler.go              # ACS Inform 处理
├── export_handler.go              # 导出处理器
├── export_service.go              # 导出服务
│
└── *_test.go                      # 测试文件
```

### 附录 G: 前端相关文件清单

```
omcmb/webcode/src/
├── types/
│   ├── device.ts                  # 设备类型（可选优化状态类型）
│   └── deviceParameter.ts         # 参数类型
│
├── services/api/
│   ├── deviceApi.ts               # 设备 API（无需修改）
│   └── deviceParameterApi.ts      # 参数 API（无需修改）
│
├── hooks/api/
│   ├── useDevices.ts              # 设备 Hooks（无需修改）
│   └── useDeviceParameters.ts     # 参数 Hooks（无需修改）
│
├── mock/
│   ├── data/devices.ts            # Mock 数据（无需修改）
│   └── services/deviceService.ts  # Mock 服务（无需修改）
│
└── pages/device/
    ├── DeviceList/                # 设备列表（无需修改）
    ├── DeviceDetail/              # 设备详情（无需修改）
    ├── DeviceGrouping/            # 设备分组（无需修改）
    ├── DeviceRegistration/        # 预注册（无需修改）
    ├── Commissioning/             # 开站（无需修改）
    ├── OnlineMonitoring/          # 在线监控（无需修改）
    ├── ImportExport/              # 导入导出（无需修改）
    ├── RecycleBin/                # 回收站（无需修改）
    └── ...
```
