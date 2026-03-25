# OMC 系统功能完善方案

> 创建时间：2026-03-19  
> 最后更新：2026-03-19（重新评估，修正初版误判）  
> 基于：全项目文档分析（docs/、features/、detailed-design/）+ 代码实际状态核查  
> 范围：omcgo/ 全模块

---

## 一、现状评估

### 1.1 整体完成度

| 维度 | 状态 | 说明 |
|------|------|------|
| 基础架构 | ✅ 完成 | 三进程 (acs/app/worker)、NATS/PG/Redis/MinIO 全部就绪 |
| Phase 1-4 计划项 | ✅ 完成 | 17 个 Sprint 全部交付，代码行数 ~41,500 行 |
| 编译状态 | ✅ **正常** | `go build ./...` 零错误，`go vet ./...` 零警告 |
| 测试状态 | ✅ **全部通过** | `go test ./...` 全部 PASS，无 FAIL |
| 功能完整度 | ⚠️ 待完善 | 设计文档中有明确 gap 尚未实现 |

### 1.2 已解决问题（初版误判，已更正）

初版方案将以下问题列为「编译阻断级」，经重新评估均已**实际解决**，当前代码库编译和测试均正常：

| # | 文件 | 初版描述 | 实际状态 |
|---|------|---------|----------|
| ~~1~~ | `internal/acs/handler.go` | Git 冲突标记未解决 | ✅ **已解决** — 当前版本已选择保留 `logger` 包，移除了 `upload` 导入；`upload` 功能的集成入口在 `server.go` 中完整保留 |
| ~~2~~ | `internal/acs/server.go` | Git 冲突标记未解决 | ✅ **已解决** — `ServerDeps` 已合并两版：同时含 `UploadHandler` 和 `RequestIDPrefix` 字段 |
| ~~3~~ | `internal/core/appconfig/config.go` | Git 冲突标记未解决 | ✅ **已解决** — `ACSConfig` 同时含 `MinIO`/`Upload` 和 `RequestIDPrefix` 字段，合并完整 |
| ~~4~~ | `internal/config/datamodel/registry.go` | 引用不存在的子包 `commonerrors` | ✅ **已解决（工作区未提交）** — 当前磁盘文件已修正为 `commonerrors "github.com/omcgo/omcgo/internal/core/errors"`，仍处于 git 未提交状态，需执行 `git add + commit` |

> **注意**：`internal/config/datamodel/registry.go` 的修复存在于工作区但尚未提交（`git status` 显示为已修改）。需执行一次 commit 固化。

### 1.3 当前实际待解决问题

**仅剩 1 项代码问题**，其余为功能完善性缺口：

| # | 文件 | 问题类型 | 详细描述 |
|---|------|---------|----------|
| P1 | `internal/config/datamodel/registry.go` | 未提交 | 包路径修复已在工作区，需要 commit 入版本控制 |

### 1.4 功能缺口清单（文档 vs 代码）

基于 `docs/design/tr069-parameter-implementation-proposal.md`、`docs/design/permission-system-design.md`、`docs/design/acs-file-upload-design.md` 与实际代码对比：

| # | 功能域 | 具体缺口 | 来源文档 |
|---|--------|---------|----------|
| G1 | 设备管理 | Device 模型缺少 9 个 TR069 字段（hardware_version/run_time/plmn/cell_status/rf_status/height/sync_source/sync_state/mme_pool） | tr069-parameter-implementation-proposal |
| G2 | 设备管理 | GPVResponse 参数提取逻辑完全未实现（仅有 Inform 参数提取） | tr069-parameter-implementation-proposal |
| G3 | 设备管理 | 多小区配置（CA/SC/DC/TC 载波模式）未支持 | tr069-parameter-implementation-proposal |
| G4 | 设备管理 | 设备许可证表（device_licenses）未建立（migrations 止于 000035/000046/000047） | tr069-parameter-implementation-proposal |
| G5 | 权限系统 | 仅有三角色（admin/operator/viewer）RBAC，缺少用户-部门-菜单的完整权限模型 | permission-system-design |
| G6 | 权限系统 | 数据权限（按运营商/站点隔离）仅在 context 注入，Repository 层未强制 | phase4-analysis-report §5.4 |
| G7 | 权限系统 | 菜单/按鈕级权限控制未实现 | permission-system-design |
| G8 | ACS 文件上传 | `acs/upload/` 已实现，`server.go` 集成正常，但 `handler.go` 中已**移除** upload 包导入（handler 层未引用）—需确认项目是否应在 handler 层进一步处理 UploadComplete 事件 | acs-file-upload-design |
| G9 | 安全加固 | JWT Secret 最小长度校验缺失 | phase4-analysis-report §5.2 |
| G10 | 安全加固 | 密码复杂度规则缺失 | phase4-analysis-report §5.2 |
| G11 | 安全加固 | /auth/login 无限流保护 | phase4-analysis-report §5.2 |
| G12 | 安全加固 | Refresh Token 黑名单（登出后立即失效）未实现 | phase4-analysis-report §5.2 |
| G13 | 北向接口 | Push 引擎仅支持 HTTP，缺少 gRPC/WebSocket 推送协议 | phase4-analysis-report §5.4 |
| G14 | 固件管理 | 升级任务超时自动失败机制未实现 | phase4-analysis-report §5.4 |
| G15 | 互操作测试 | 测试结果未持久化到数据库 | phase4-analysis-report §5.4 |
| G16 | 生产化 | Dockerfile/Helm Chart/CI Pipeline 缺失 | phase4-analysis-report §5.5 |
| G17 | 可观测性 | Alertmanager 规则文件缺失 | phase4-analysis-report §5.3 |

---

## 二、完善方案

按**优先级**和**依赖关系**划分为 5 个 Sprint。

---

### Sprint A：代码共夹整理（预计 0.5 人天）

**目标**：固化工作区已修复的内容，确保所有修复均进入版本控制。

#### A.1 提交 registry.go 修复（对应 P1）

`internal/config/datamodel/registry.go` 的包路径修复已在工作区，但尚未提交。

```bash
git add internal/config/datamodel/registry.go
git commit -m "fix(datamodel): 修正 registry.go 中错误的 commonerrors 包路径引用"
```

修复内容：

```go
// 修正前（包不存在）：
"github.com/omcgo/omcgo/internal/core/errors/commonerrors"

// 修正后（使用别名保持内部代码兼容）：
commonerrors "github.com/omcgo/omcgo/internal/core/errors"
```

#### A.2 确认编译和测试全部正常

```bash
go build ./...   # 应输出无错误
go vet ./...     # 应输出无警告
go test ./...    # 应全部 PASS
```

---

### Sprint B：设备 TR069 参数完善（预计 4-5 人天）

**目标**：实现设备模型字段补全、GPVResponse 参数提取、多小区支持。  
**来源**：`docs/design/tr069-parameter-implementation-proposal.md`

#### B.1 数据库迁移（新建 2 个迁移文件）

**文件 1：`migrations/000048_add_device_tr069_fields.up.sql`**

```sql
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS hardware_version VARCHAR(64),
    ADD COLUMN IF NOT EXISTS run_time        VARCHAR(32),
    ADD COLUMN IF NOT EXISTS plmn            VARCHAR(128),
    ADD COLUMN IF NOT EXISTS cell_status     BOOLEAN,
    ADD COLUMN IF NOT EXISTS rf_status       BOOLEAN,
    ADD COLUMN IF NOT EXISTS height          DOUBLE PRECISION,
    ADD COLUMN IF NOT EXISTS sync_source     INTEGER,
    ADD COLUMN IF NOT EXISTS sync_state      VARCHAR(16),
    ADD COLUMN IF NOT EXISTS mme_pool        JSONB,
    ADD COLUMN IF NOT EXISTS carrier_mode    VARCHAR(8) DEFAULT 'SC';

COMMENT ON COLUMN devices.hardware_version IS '硬件版本，来自 Inform';
COMMENT ON COLUMN devices.run_time IS '运行时间，格式如 "40d 4h 58m 29s"';
COMMENT ON COLUMN devices.plmn IS 'PLMN 列表，逗号分隔';
COMMENT ON COLUMN devices.cell_status IS '小区运行状态';
COMMENT ON COLUMN devices.rf_status IS '射频发射状态';
COMMENT ON COLUMN devices.height IS '天线高度（米）';
COMMENT ON COLUMN devices.sync_source IS '同步主源（0-7）';
COMMENT ON COLUMN devices.sync_state IS '同步状态';
COMMENT ON COLUMN devices.mme_pool IS 'MME 池配置 JSON 数组';
COMMENT ON COLUMN devices.carrier_mode IS '载波模式：CA/SC/DC/TC';
```

**文件 2：`migrations/000049_create_device_licenses.up.sql`**

```sql
CREATE TABLE IF NOT EXISTS device_licenses (
    id            UUID        NOT NULL DEFAULT gen_random_uuid(),
    device_id     UUID        NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    license_code  VARCHAR(16) NOT NULL,
    version       INTEGER     NOT NULL,
    author        VARCHAR(64),
    generate_date VARCHAR(8),
    seq_num       INTEGER,
    max_capacity  INTEGER DEFAULT 32,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);
CREATE INDEX IF NOT EXISTS idx_device_licenses_device_id ON device_licenses(device_id);
```

#### B.2 更新 Device 模型

**文件：`internal/core/model/device.go`**

在 `Device` 结构体中新增：

```go
// TR069 扩展字段（来自 Inform ParameterList）
HardwareVersion string  `json:"hardware_version,omitempty" db:"hardware_version"`
RunTime         string  `json:"run_time,omitempty"         db:"run_time"`
PLMN            string  `json:"plmn,omitempty"             db:"plmn"`
CellStatus      bool    `json:"cell_status"                db:"cell_status"`
RFStatus        bool    `json:"rf_status"                  db:"rf_status"`
// TR069 扩展字段（来自 GetParameterValuesResponse）
Height          float64         `json:"height"                     db:"height"`
SyncSource      int             `json:"sync_source"                db:"sync_source"`
SyncState       string          `json:"sync_state,omitempty"       db:"sync_state"`
MMEPool         json.RawMessage `json:"mme_pool,omitempty"         db:"mme_pool"`
CarrierMode     string          `json:"carrier_mode"               db:"carrier_mode"`
```

#### B.3 扩展 InformHandler 参数提取

**文件：`internal/device/inform_handler.go`**

在 Inform 处理中补充提取：

```go
// 新增提取逻辑（在 extractFromInform 函数中）
device.HardwareVersion = findParamValue(params, "Device.DeviceInfo.HardwareVersion")
device.RunTime         = findParamValue(params, "Device.DeviceInfo.X_COM_STATION_RUN_Time")
device.PLMN            = findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList")
device.CellStatus      = findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.OpState") == "true"
device.RFStatus        = findParamValue(params, "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus") == "true"
```

#### B.4 实现 GPVResponse 参数提取

**新建文件：`internal/device/gpv_handler.go`**

实现 `HandleGPVResponse(ctx, deviceID, params)` 方法，提取：
- 模块型号 (`Device.DeviceInfo.X_COM_MODULE_TYPE` → `device.ModelName`)
- 经纬度 (`Device.FAP.GPS.LockedLatitude/Longitude`)
- 天线高度 (`Device.DeviceInfo.AntennaInfo.Height`)
- 同步源 (`Device.ManagementServer.tfcsManagerPrimsrc/tfcsSyncState`)
- MME 池配置（16 个参数 → JSON 数组）

#### B.5 多小区支持

**新建文件：`internal/device/cell_handler.go`**

根据 `carrier_mode` 决定保存哪些小区参数：
- `SC/CA`：仅主小区（FAPService.1）
- `DC`：主 + 小区2（FAPService.1 + FAPService.2）
- `TC`：三小区（FAPService.1-3）

#### B.6 验收标准

- `go build ./...` 零错误
- 新增迁移文件可正向/回滚执行
- Device API 返回新增字段
- 单元测试覆盖 GPVResponse 解析（≥5 个测试函数）

---

### Sprint C：权限系统完善（预计 5-6 人天）

**目标**：实现完整的用户-部门-菜单权限模型，强制数据权限隔离。  
**来源**：`docs/design/permission-system-design.md`、`docs/phase4-analysis-report.md §5.4`

#### C.1 数据库扩展（新建迁移文件）

**`migrations/000050_extend_permission_system.up.sql`**

```sql
-- 部门表
CREATE TABLE IF NOT EXISTS departments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(128) NOT NULL,
    parent_id   UUID REFERENCES departments(id),
    level       INTEGER NOT NULL DEFAULT 1,
    sort_order  INTEGER NOT NULL DEFAULT 0,
    leader_id   UUID,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 菜单/权限点表
CREATE TABLE IF NOT EXISTS menus (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id   UUID REFERENCES menus(id),
    name        VARCHAR(64)  NOT NULL,
    menu_type   VARCHAR(16)  NOT NULL,  -- 'menu' | 'button' | 'api'
    path        VARCHAR(256),
    component   VARCHAR(256),
    permission  VARCHAR(128),           -- 权限码如 "device:list"
    icon        VARCHAR(64),
    sort_order  INTEGER NOT NULL DEFAULT 0,
    is_visible  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 角色-菜单关联表
CREATE TABLE IF NOT EXISTS role_menus (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    menu_id UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, menu_id)
);

-- 用户-部门关联
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS dept_id UUID REFERENCES departments(id),
    ADD COLUMN IF NOT EXISTS data_scope VARCHAR(16) DEFAULT 'carrier'; -- carrier|dept|all

-- 角色数据权限范围
ALTER TABLE roles
    ADD COLUMN IF NOT EXISTS data_scope VARCHAR(16) DEFAULT 'carrier'; -- carrier|dept|all|custom
```

#### C.2 Repository 层数据权限强制

**修改各模块 pg_repository.go**，在查询 SQL 中注入运营商/部门过滤条件：

```go
// 在 Repository 构造函数中注入 carrier 过滤器
type deviceRepository struct {
    pool    *pgxpool.Pool
    carrier model.CarrierCode // 非空时强制过滤
}

// 所有 List 查询自动追加 WHERE carrier = $N
func (r *deviceRepository) List(ctx context.Context, req model.ListRequest) ([]model.Device, int64, error) {
    q := squirrel.Select("*").From("devices")
    if r.carrier != "" {
        q = q.Where(squirrel.Eq{"carrier": r.carrier})
    }
    // ...
}
```

受影响模块：`device/`、`alarm/`、`pm/`、`mr/`、`topology/`、`software/`

#### C.3 菜单权限 API

**新建：`internal/admin/menu_handler.go`** — CRUD 菜单和权限点  
**新建：`internal/admin/menu_model.go`** — 菜单数据结构  
**新建：`internal/admin/pg_menu_repository.go`** — 菜单持久化  
**新建：`internal/admin/dept_handler.go`** — 部门管理 CRUD  

#### C.4 前端权限码约定

在 `global/consts.go` 中定义权限码常量：

```go
const (
    PermDeviceList   = "device:list"
    PermDeviceCreate = "device:create"
    PermDeviceUpdate = "device:update"
    PermDeviceDelete = "device:delete"
    PermAlarmList    = "alarm:list"
    PermAlarmAck     = "alarm:ack"
    PermPMView       = "pm:view"
    // ... 全部权限码
)
```

#### C.5 验收标准

- 三大系统角色权限矩阵与设计文档一致
- Repository 层 carrier 过滤通过单元测试验证
- 菜单 API 支持树形结构返回
- 部门 CRUD 接口可用

---

### Sprint D：安全加固（预计 2-3 人天）

**目标**：修复遗留的安全缺陷，提升生产安全等级。  
**来源**：`docs/phase4-analysis-report.md §5.2`

#### D.1 JWT Secret 校验

**修改：`internal/admin/jwt.go`**（在 `NewJWTService` 中）：

```go
func NewJWTService(cfg appconfig.JWTConfig, logger *zap.Logger) (*JWTService, error) {
    if len(cfg.Secret) < 32 {
        return nil, fmt.Errorf("JWT secret too short: minimum 32 bytes required, got %d", len(cfg.Secret))
    }
    // ...
}
```

#### D.2 密码复杂度校验

**修改：`internal/admin/service.go`**（在 `CreateUser`/`ChangePassword` 中）：

```go
func validatePassword(password string) error {
    if len(password) < 8 { return ErrPasswordTooShort }
    // 需包含大写字母、小写字母、数字、特殊字符各至少一个
    if !hasUpper(password) || !hasLower(password) || !hasDigit(password) || !hasSpecial(password) {
        return ErrPasswordComplexity
    }
    return nil
}
```

#### D.3 登录限流

**修改：`cmd/app/server.go` 或 `internal/admin/middleware.go`**：

```go
// 在 /auth/login 路由上添加限流中间件
authGroup.POST("/login", loginRateLimiter(5, time.Minute), authHandler.Login)
```

使用 Redis 存储计数：每 IP 每分钟最多 5 次登录尝试，超限返回 429 + 错误响应。

#### D.4 Refresh Token 黑名单

**修改：`internal/admin/jwt.go`** 和 **`internal/admin/service.go`**：

```go
// Logout 时将 Refresh Token JTI 写入 Redis 黑名单（TTL = refresh_token_ttl）
func (s *JWTService) RevokeRefreshToken(ctx context.Context, tokenID string, ttl time.Duration) error {
    key := fmt.Sprintf("revoked_token:%s", tokenID)
    return s.redis.Set(ctx, key, "1", ttl).Err()
}

// 验证 Refresh Token 时检查黑名单
func (s *JWTService) ValidateRefreshToken(ctx context.Context, tokenStr string) (*Claims, error) {
    claims, err := s.parseToken(tokenStr)
    if err != nil { return nil, err }
    // 检查黑名单
    key := fmt.Sprintf("revoked_token:%s", claims.ID)
    if exists, _ := s.redis.Exists(ctx, key).Result(); exists > 0 {
        return nil, ErrTokenRevoked
    }
    return claims, nil
}
```

#### D.5 验收标准

- JWT Secret < 32 字节时服务启动失败（而非 panic）
- 弱密码创建用户返回 400 + 明确错误信息
- 登录 IP 超限返回 429
- 登出后 Refresh Token 无法再次使用（单元测试验证）

---

### Sprint E：生产化与可观测性（预计 3-4 人天）

**目标**：补全生产就绪所需基础设施。  
**来源**：`docs/phase4-analysis-report.md §5.3 §5.5`

#### E.1 固件升级超时机制

**修改：`internal/software/service.go`**

新增后台 goroutine，定时扫描 `pending/downloading` 状态超过阈值的升级任务并转为 `failed`：

```go
func (s *SoftwareService) StartTimeoutChecker(interval, upgradeTimeout time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for range ticker.C {
            s.checkAndFailTimeoutTasks(context.Background(), upgradeTimeout)
        }
    }()
}
```

#### E.2 互操作测试结果持久化

**新增迁移：`migrations/000051_create_interop_results.up.sql`**

```sql
CREATE TABLE IF NOT EXISTS interop_test_results (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices(id),
    device_sn   VARCHAR(64) NOT NULL,
    category    VARCHAR(32) NOT NULL,
    case_id     VARCHAR(64) NOT NULL,
    case_name   VARCHAR(128) NOT NULL,
    passed      BOOLEAN NOT NULL,
    duration_ms INTEGER,
    error_msg   TEXT,
    detail      JSONB,
    run_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON interop_test_results(device_sn, run_at DESC);
```

**修改：`internal/interop/runner.go`**，执行完毕后写入数据库。

#### E.3 Dockerfile 补全

**新建 3 个文件**（多阶段构建，最终镜像基于 `gcr.io/distroless/base`）：

- `deployments/docker/Dockerfile.acs`
- `deployments/docker/Dockerfile.app`
- `deployments/docker/Dockerfile.worker`

示例（Dockerfile.acs）：

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/omcgo-acs ./cmd/acs

# Final stage
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/omcgo-acs /omcgo-acs
ENTRYPOINT ["/omcgo-acs"]
```

#### E.4 Alertmanager 规则文件

**新建：`deployments/monitoring/alertmanager-rules.yaml`**

覆盖以下告警维度：
- ACS 活跃会话 > 阈值（实例容量告警）
- 设备离线率 > 5%
- 告警处理积压（NATS consumer lag > 1000）
- PostgreSQL 连接池耗尽
- KPI 计算失败率 > 1%

#### E.5 CI/CD Pipeline

**新建：`.github/workflows/ci.yml`**

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.23' }
      - run: go mod download
      - run: go build ./...
      - run: go vet ./...
      - run: go test ./... -race -coverprofile=coverage.out
      - run: go tool cover -func=coverage.out
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golangci/golangci-lint-action@v6
        with: { version: latest }
```

#### E.6 可观测性增强（各 handler 添加指标）

在关键路径添加 Prometheus 指标：

```go
// 告警处理
alarmProcessed.WithLabelValues(carrier, result).Inc()

// 固件升级状态转换
upgradeTransition.WithLabelValues(from, to).Inc()

// NE Direct 请求
nedirectRequests.WithLabelValues(endpoint, status).Inc()

// 登录失败计数（用于暴力破解监控）
loginFailures.WithLabelValues(ip).Inc()
```

#### E.7 验收标准

- 固件升级任务在配置时限（默认 30 分钟）后自动标记 `failed`
- 互操作测试结果可通过 API 查询历史记录
- 3 个服务的 Docker 镜像可成功构建
- CI Pipeline 在 main/develop 分支触发自动运行
- Alertmanager 规则文件语法正确（`promtool check rules`）

---

## 三、Sprint 执行计划

### 3.1 依赖关系

```
Sprint A（紧急修复）
    └── Sprint B（设备参数）   # 需要 A 完成才能编译验证
    └── Sprint C（权限系统）   # 需要 A 完成才能编译验证
    └── Sprint D（安全加固）   # 需要 A，依赖 C 的 JWT 基础
Sprint E（生产化）             # 相对独立，可与 B/C/D 并行
```

### 3.2 时间估计

| Sprint | 预估工作量 | 建议执行顺序 |
|--------|-----------|------------|
| A 代码共夹整理 | 0.5 人天 | **第一**（commit 工作区已修复） |
| B 设备参数 | 4-5 人天 | 第二（A 完成后） |
| C 权限系统 | 5-6 人天 | 第三（A 完成后，与 B 并行） |
| D 安全加固 | 2-3 人天 | 第四（A/C 完成后） |
| E 生产化 | 3-4 人天 | 可与 B/C/D 并行 |
| **合计** | **15-18.5 人天** | |

### 3.3 里程碑验收

| 里程碑 | 条件 |
|--------|------|
| M1：编译恢复 | `go build ./...` 零错误，`go test ./...` 全 PASS |
| M2：设备能力完整 | GPVResponse 提取可用，Device API 返回 TR069 扩展字段 |
| M3：安全就绪 | 权限系统完整，安全加固项全部实现 |
| M4：生产就绪 | CI Pipeline 绿色，Docker 镜像可部署，监控告警完整 |

---

## 四、附录：文件变更清单

### 4.1 Sprint A 涉及文件

| 操作 | 文件路径 |
|------|----------|
| 提交（已在工作区修正） | `internal/config/datamodel/registry.go` |

### 4.2 Sprint B 涉及文件

| 操作 | 文件路径 |
|------|---------|
| 新建 | `migrations/000048_add_device_tr069_fields.up.sql` |
| 新建 | `migrations/000048_add_device_tr069_fields.down.sql` |
| 新建 | `migrations/000049_create_device_licenses.up.sql` |
| 新建 | `migrations/000049_create_device_licenses.down.sql` |
| 修改 | `internal/core/model/device.go` |
| 修改 | `internal/device/service.go` |
| 修改 | `internal/device/inform_handler.go` |
| 新建 | `internal/device/gpv_handler.go` |
| 新建 | `internal/device/gpv_handler_test.go` |
| 新建 | `internal/device/cell_handler.go` |
| 修改 | `internal/device/pg_repository.go` |

### 4.3 Sprint C 涉及文件

| 操作 | 文件路径 |
|------|---------|
| 新建 | `migrations/000050_extend_permission_system.up.sql` |
| 新建 | `migrations/000050_extend_permission_system.down.sql` |
| 新建 | `internal/admin/menu_model.go` |
| 新建 | `internal/admin/menu_handler.go` |
| 新建 | `internal/admin/menu_handler_test.go` |
| 新建 | `internal/admin/pg_menu_repository.go` |
| 新建 | `internal/admin/dept_model.go` |
| 新建 | `internal/admin/dept_handler.go` |
| 新建 | `internal/admin/dept_handler_test.go` |
| 新建 | `internal/admin/pg_dept_repository.go` |
| 修改 | `global/consts.go`（添加权限码常量） |
| 修改 | `internal/device/pg_repository.go`（数据权限） |
| 修改 | `internal/alarm/pg_store.go`（数据权限） |
| 修改 | `internal/pm/pg_task_repository.go`（数据权限） |

### 4.4 Sprint D 涉及文件

| 操作 | 文件路径 |
|------|---------|
| 修改 | `internal/admin/jwt.go` |
| 修改 | `internal/admin/service.go` |
| 新建 | `internal/admin/middleware_ratelimit.go` |
| 新建 | `internal/admin/middleware_ratelimit_test.go` |

### 4.5 Sprint E 涉及文件

| 操作 | 文件路径 |
|------|---------|
| 修改 | `internal/software/service.go` |
| 新建 | `migrations/000051_create_interop_results.up.sql` |
| 新建 | `migrations/000051_create_interop_results.down.sql` |
| 新建 | `internal/interop/pg_result_repository.go` |
| 修改 | `internal/interop/runner.go` |
| 新建 | `deployments/docker/Dockerfile.acs` |
| 新建 | `deployments/docker/Dockerfile.app` |
| 新建 | `deployments/docker/Dockerfile.worker` |
| 新建 | `deployments/monitoring/alertmanager-rules.yaml` |
| 新建 | `.github/workflows/ci.yml` |
