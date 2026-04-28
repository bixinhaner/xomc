# PRD: F06 License 容量 / 过期拦截

> **关联**: Backlog T-0015 / Risk R-103 / Sprint-03 / Wave 4 启动
> **作者**: Claude（代 Owner=电信业务专家）
> **创建**: 2026-04-28
> **状态**: 草案 → 实施

---

## 1. 业务背景

OMC 是商用网管系统，遵循 OEM 授权模式：
- **设备数量受 license 容量限制**（防止超卖；运营商按授权数付费）
- **license 有效期决定服务持续期**（过期需续期）
- **过期后限制写入**但不影响只读运维（保证生产网管不中断）

当前 `internal/license/` 模块已有完整 CRUD（Import / Activate / Revoke / List / Summary），`License.MaxDevices` / `License.UsedDevices` / `License.ExpiryDate` 字段齐全，但**无人调用拦截**：
- ❌ `device.CreateDevice` 不查 license 配额，可无限超卖
- ❌ 写操作不查 license 状态，过期后仍可写入
- ❌ 无定时器把 `expiry_date < now()` 的 license 转 `status=expired`
- ❌ 无容量阈值告警（80% / 90% / 95%）

R-103 风险登记册描述：「MaxDevices/ExpiryDate 字段有，但无超限拦截与自动禁用」。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| OEM 销售方 | 我希望客户实际部署设备数 ≤ license 授权数，超过时系统拒绝新增并提示 |
| 运营商运维 | 我希望 license 即将过期（30/7/1 天）时收到告警，以便提前续期 |
| 系统管理员 | 我希望仪表盘看到当前 license 容量使用率与剩余天数，量化授权状态 |
| 开发 / 测试人员 | 我希望本地 dev 环境无 license 时**不被拦截**（默认放行 + warning 日志） |

---

## 3. 验收标准（Given-When-Then）

### V1 — 容量未满放行
- **Given** 当前 active license `MaxDevices=100, UsedDevices=99`
- **When** `POST /api/v1/devices` 添加新设备
- **Then** 返回 200 + UsedDevices 更新为 100

### V2 — 容量超限拦截
- **Given** 当前 active license `MaxDevices=100, UsedDevices=100`
- **When** `POST /api/v1/devices` 添加新设备
- **Then** 返回 **403** + 错误码 9101 `license_capacity_exceeded`
- **And** Prometheus `license_enforcement_total{operation="device.create",result="denied_capacity"}` +1

### V3 — license 过期拦截写入
- **Given** 当前 active license `status=expired`（cron 自动转），且 `LicenseType != perpetual`
- **When** `POST /api/v1/devices` 添加新设备
- **Then** 返回 **403** + 错误码 9102 `license_expired`
- **And** GET 类操作（dashboard / report / alarm 查询）**不受影响**

### V4 — 过期前告警
- **Given** active license `expiry_date = now + 30 days`，类型 = subscription
- **When** daily cron 触发 expiry checker
- **Then** 写一条 alarm `severity=warning, alarm_identifier="license_expiring_30d"`，字段 `license_id` / `expiry_date` / `days_remaining=30`
- **过期前 7 天**：severity=major，alarm_identifier=`license_expiring_7d`
- **过期前 1 天**：severity=critical，alarm_identifier=`license_expiring_1d`

### V5 — 容量阈值告警去重
- **Given** UsedDevices 触发 80% 阈值，6h 前已发过同阈值告警
- **When** hourly cron 触发 capacity checker
- **Then** **不**重复发告警（last_capacity_alert_at 6h 内同阈值跳过）
- **And** 跨过下一阈值（90%）时立即发新告警，不等 6h

### V6 — 永久 license 跳过过期
- **Given** active license `LicenseType=perpetual, ExpiryDate=NULL`
- **When** daily expiry checker 跑
- **Then** 此 license 不被转 expired，无告警

### V7 — 无 active license 默认放行
- **Given** 数据库无任何 active license
- **When** `POST /api/v1/devices` 添加新设备
- **Then** 返回 200（不拦截）+ 启动时 warning 日志 + `license_active_count=0` metric
- **理由**：dev 环境友好；prod 部署 OEM Import license 后自动开始拦截

### V8 — 容量配额查询
- **Given** active license MaxDevices=100, UsedDevices=42
- **When** `GET /api/v1/licenses/quota`
- **Then** 返回 200 + JSON `{max_devices:100, used_devices:42, usage_ratio:0.42, days_remaining:N|null, license_type:"...", grace_period_days:0}`

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| License 颁发主体 | OEM 厂商（与运营商解耦）| 同 | 同 |
| 容量计算单位 | 按设备序列号去重 | 同 | 同 |
| 过期策略 | OEM 决定，OMC 不区分 | 同 | 同 |
| 实际差异 | **无** | **无** | **无**（一刀切）|

**结论**：F06 license 拦截**对运营商透明**，不需要 Carrier 适配点。OEM 颁发 license 时已决定容量与有效期，OMC 仅做执行层。

> **未来扩展点**：若 OEM 将来要求按设备类型（pico/femto/micro）分别授权，可在 `License.Features` JSONB 字段加 `device_type_quota` 子字段，由 Carrier 接口的 `MapDeviceTypeToQuotaKey()` 适配 — **本任务不做**。

---

## 5. 非目标

- ❌ 不实现 license 数字签名验证（OEM 颁发流程的事，本任务只查数据库 license 表状态）
- ❌ 不实现 license 文件离线导入解析（已有 `Import` API 走 JSON 输入）
- ❌ 不实现按运营商分别 license（一份 license 跨运营商）
- ❌ 不实现 license 续期自助流程（线下走 OEM）
- ❌ 不接入 template / software upgrade 拦截（**第一版仅 device.Create**，最小 viable，后续 PR 扩展）

---

## 6. 依赖

| 依赖 | 用途 |
|------|------|
| `internal/license/` | 自身（Service / Repo / Model）|
| `internal/device/` | DeviceService.CreateDevice 装载 LicenseEnforcer setter（小接口） |
| `github.com/robfig/cron/v3` | daily expiry checker + hourly capacity checker（已有 dep）|
| `internal/alarm/` | RaiseAlarm API（按需触发告警）— 通过 callback 接口注入避免循环依赖 |
| `internal/core/errors/` | 新增 ErrLicenseCapacityExceeded / ErrLicenseExpired sentinel |
| migrations/000043 | licenses 表加 `grace_period_days` / `capacity_alert_thresholds` / `last_capacity_alert_at` |

---

## 7. 度量（Prometheus 指标）

| 指标 | 类型 | 标签 | 说明 |
|------|------|------|------|
| `license_active_count` | gauge | — | 当前 active license 数（=0 时 warning）|
| `license_capacity_used_devices` | gauge | — | 当前 used devices |
| `license_capacity_max_devices` | gauge | — | 最大授权设备数（多 license 取 max）|
| `license_capacity_usage_ratio` | gauge | — | used / max（0-1）|
| `license_expiry_days_remaining` | gauge | `license_id` | 剩余天数（perpetual = -1）|
| `license_enforcement_total` | counter | `operation,result` | result ∈ {allowed, denied_capacity, denied_expired, no_active_license}|

---

## 8. 设计备忘（S2）

### 8.1 接口契约

**license 包定义**（实现侧）:
```go
// internal/license/enforcer.go
type Enforcer interface {
    EnforceCapacity(ctx context.Context, additional int) error
    EnforceExpiry(ctx context.Context, operation string) error
    ActiveLicense(ctx context.Context) (*License, error)
    Quota(ctx context.Context) (*Quota, error)
}

type Quota struct {
    MaxDevices       int
    UsedDevices      int
    UsageRatio       float64
    DaysRemaining    int    // -1 = perpetual / no expiry
    LicenseType      string
    GracePeriodDays  int
    HasActiveLicense bool
}
```

**device 包消费侧定义**（小接口原则）:
```go
// internal/device/device_service.go 新增
type LicenseEnforcer interface {
    EnforceCapacity(ctx context.Context, additional int) error
    EnforceExpiry(ctx context.Context, operation string) error
}

func (s *DeviceService) SetLicenseEnforcer(e LicenseEnforcer) {
    s.licenseEnforcer = e
}
```

DeviceService.CreateDevice 在 ctx check existing 之后、写入 DB 之前加：
```go
if s.licenseEnforcer != nil {
    if err := s.licenseEnforcer.EnforceExpiry(ctx, "device.create"); err != nil {
        return nil, err
    }
    if err := s.licenseEnforcer.EnforceCapacity(ctx, 1); err != nil {
        return nil, err
    }
}
```

### 8.2 错误码

`internal/core/errors/errors.go` 新增 sentinel：
```go
var (
    ErrLicenseCapacityExceeded = errors.New("license capacity exceeded")
    ErrLicenseExpired          = errors.New("license expired")
)
```

`codes.go` 9000-9099 段（与 northbound 9000-9999 共享，预留 9100-9199 给 license）：
- 9101: license_capacity_exceeded → HTTP 403
- 9102: license_expired → HTTP 403

errors.go AbortWithError 加 case 把上述 sentinel 映射到 HTTP 403（`ErrForbidden`）。

### 8.3 DB 改动 (migrations/000043_licenses_capacity_expiry.sql)

```sql
-- +goose Up
ALTER TABLE licenses
    ADD COLUMN IF NOT EXISTS grace_period_days INT NOT NULL DEFAULT 0
        CHECK (grace_period_days >= 0 AND grace_period_days <= 365),
    ADD COLUMN IF NOT EXISTS capacity_alert_thresholds JSONB NOT NULL
        DEFAULT '[80, 90, 95]'::jsonb,
    ADD COLUMN IF NOT EXISTS last_capacity_alert_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_capacity_alert_threshold INT;

CREATE INDEX IF NOT EXISTS idx_licenses_active_status
    ON licenses(status) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_licenses_expiry_date
    ON licenses(expiry_date) WHERE expiry_date IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_licenses_expiry_date;
DROP INDEX IF EXISTS idx_licenses_active_status;
ALTER TABLE licenses
    DROP COLUMN IF EXISTS last_capacity_alert_threshold,
    DROP COLUMN IF EXISTS last_capacity_alert_at,
    DROP COLUMN IF EXISTS capacity_alert_thresholds,
    DROP COLUMN IF EXISTS grace_period_days;
```

### 8.4 cron 设计

`monitor.go`:
- daily @ `0 0 * * *` UTC：扫所有 active license，过期（now > expiry_date + grace）→ status=expired + 写 system_logs
- daily @ `0 1 * * *` UTC：检查过期前 30/7/1 天阈值 → 触发对应 severity 告警
- hourly @ `0 * * * *`：检查容量阈值（80/90/95%）→ 去重后触发告警

cron lib `github.com/robfig/cron/v3` 已在 go.mod。`cmd/app/provider/modules.go` 启动时注册 cron job。

### 8.5 缓存策略

进程内缓存 active license：
```go
type cachedActive struct {
    mu        sync.RWMutex
    license   *License  // 取 max(MaxDevices) 那条
    cachedAt  time.Time
    ttl       time.Duration  // 5 min
}
```

Import / Activate / Revoke 时主动 Invalidate（设 cachedAt = zero）。

容量计数（CountActiveDevices）实时查 DB（PG `SELECT COUNT(*) FROM devices` < 1ms 索引下），不缓存。

### 8.6 多 active license 处理

repo `GetActiveLicenseWithMaxDevices()` 返回 active 中 MaxDevices 最大的一条 license 作为 enforcement 依据：
```sql
SELECT * FROM licenses WHERE status = 'active'
ORDER BY max_devices DESC LIMIT 1;
```

剩余 active license 的 expiry / type 仅供查询，**不参与 enforcement 决策**（避免歧义）。

如果存在多 active license 但取 max 后另一条已过期，不影响：实际拦截依据是被选中的那条 license 状态。

### 8.7 测试策略

| 文件 | 用例数 | 覆盖 |
|------|--------|------|
| `enforcer_test.go` | ≥ 12 | EnforceCapacity 边界 / EnforceExpiry 边界 / Permanent 跳过 / NoLicense 放行 / 缓存 TTL / 缓存 Invalidate |
| `monitor_test.go` | ≥ 8 | daily expiry checker 路径 / 30/7/1 天告警 / hourly capacity 三档阈值 / 6h 去重 |
| `service_test.go` 增量 | +3 | ActiveLicense / Quota / Invalidate hook |
| `device_service_test.go` 增量 | +3 | LicenseEnforcer setter / Create 拦截路径 / nil enforcer 跳过 |

整体覆盖率 ≥ 70%（与 W2 标准一致）。

---

## 9. 验收（DoD 对齐 `docs/project/dod.md`）

- [ ] PRD 七要素全 + 运营商差异矩阵填写
- [ ] migration 000043 up/down 配对，goose StatementBegin/End 不需要（无 DO 块）
- [ ] `go build ./...` 通过
- [ ] `go test -race -count=1 ./internal/license/... ./internal/device/...` 全绿
- [ ] 覆盖率 license/ ≥ 70%
- [ ] `golangci-lint run ./internal/license/... ./internal/device/...` 0 error
- [ ] 新端点 GET `/api/v1/licenses/quota` 在 e2e_verify.sh 加 ≥ 1 claim
- [ ] 6 个 metric `grep -rn` 都返回 ≥ 1
- [ ] `risk-register.md` R-103 状态 Open → Closed
- [ ] backlog T-0015 状态 planned → done

---

## 10. 关联

- `internal/license/` — 现有 1588 行（CRUD 完整，本任务加 Enforcer + Monitor）
- `internal/device/device_service.go:961` — `CreateDevice` 接入点
- `docs/project/risk-register.md#R-103` — 关闭依据
- `docs/project/backlog.md` T-0015 — 状态回写
- `docs/methodology/AI承诺对峙清单.md` — 不在此章程范围（Wave 1-3 已闭门，本属常规 Sprint 工作）

---

*本 PRD 由 dev-pipeline /pick T-0015 ULTRATHINK A 方案生成，主会话全程深度协作。*
