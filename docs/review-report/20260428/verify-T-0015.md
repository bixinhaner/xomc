# Verify Report — T-0015 License 容量 / 过期拦截

**Backlog**: T-0015 / Risk R-103 / Sprint-03 → done
**PRD**: `docs/project/prd/F06-license-enforcement.md`
**Date**: 2026-04-28
**Pipeline**: `/dev-pipeline pick T-0015` ULTRATHINK A 方案（主会话全程深度协作）

---

## §1 改动清单

### 新建（5 文件）

| 路径 | 行数 | 用途 |
|------|------|------|
| `omcgo/internal/license/enforcer.go` | ~270 | Enforcer 接口 + EnforcerImpl + 5min TTL 缓存 + 4 业务方法 |
| `omcgo/internal/license/enforcer_test.go` | ~280 | 18 测例（capacity/expiry/cache/grace/perpetual/quota）|
| `omcgo/internal/license/metrics.go` | ~95 | 6 Prometheus metrics（gauge×4 + counter×1 + gauge_vec×1）|
| `omcgo/internal/license/monitor.go` | ~310 | cron 调度（daily expiry / expiring soon / hourly capacity）+ Start/Stop |
| `omcgo/internal/license/monitor_test.go` | ~250 | 14 测例（expiry sweep / 30/7/1d / capacity threshold dedup）|
| `omcgo/migrations/000043_licenses_capacity_expiry.sql` | 24 | +grace_period_days / capacity_alert_thresholds / last_capacity_alert_at + 2 partial index |
| `docs/project/prd/F06-license-enforcement.md` | ~270 | PRD 七要素 + 设计备忘 |

### 修改（13 文件）

| 路径 | 改动 |
|------|------|
| `omcgo/internal/core/errors/errors.go` | +ErrLicenseCapacityExceeded / ErrLicenseExpired sentinel + HTTP 403 映射 |
| `omcgo/internal/license/repository.go` | +5 enforcement 方法 (GetActive/ListActive/CountDevices/MarkExpired/UpdateCapacityAlert) |
| `omcgo/internal/license/pg_repository.go` | +licenseEnforcementColumns + scanLicenseFull + 5 PG 方法实现 |
| `omcgo/internal/license/model.go` | +4 字段（GracePeriodDays / CapacityAlertThresholds / LastCapacityAlertAt / LastCapacityAlertThreshold）+ Quota struct |
| `omcgo/internal/license/service.go` | +SetEnforcer / Quota / invalidateEnforcerCache（Import/Activate/Revoke 后 hook）|
| `omcgo/internal/license/handler.go` | +GET /licenses/quota 端点 |
| `omcgo/internal/license/service_test.go` | mockLicenseRepo +5 enforcement 方法 stub |
| `omcgo/internal/license/handler_test.go` | fakeLicenseRepo +5 enforcement 方法 stub |
| `omcgo/internal/device/device_service.go` | +LicenseEnforcer interface (consumer-side) + SetLicenseEnforcer + CreateDevice 加两道闸 |
| `omcgo/cmd/app/provider/modules.go` | License 模块 DI 装载（Enforcer + Metrics + Monitor）+ DeviceService.SetLicenseEnforcer + Monitor.Start |
| `omcgo/scripts/e2e_verify.sh` | +1 claim "license: quota endpoint returns 200/401 + has_active_license field" |
| `docs/project/backlog.md` | T-0015 → done + §10 变更日志 |
| `docs/project/risk-register.md` | R-103 Open → Closed |

### 兼容性 fix（pre-existing 修补，bonus）

8 个 mock 文件（device/{service,handler,heartbeat,inform_handler}_test.go + nedirect/service_test.go + northbound/sync/service_test.go + software/service_test.go + provision/engine_test.go + transfer/bridge_test.go + interop/runner_test.go + backup/executor_test.go）补 `ListProductClasses` stub —— main HEAD 同样 fail（pre-existing，T-0045 sub-agent 已发现），本任务一并修补。

---

## §2 Pass 标准达成

### 章程 / DoD（基础）

- [x] PRD 七要素全 + 运营商差异矩阵填写（CMCC/CTCC/CUCC 一致，OEM 颁发与运营商解耦）
- [x] migration 000043 up/down 配对（无 DO 块，无需 StatementBegin/End）
- [x] `go build ./...` 通过
- [x] `go vet ./...` 0 warning
- [x] `go test -race -count=1 ./internal/license/... ./internal/core/errors/... ./internal/device/...` 全绿
- [x] 新端点 GET `/api/v1/licenses/quota` 在 e2e_verify.sh 加 claim
- [x] 6 个 metric `grep -rn` 都返回 ≥ 1
- [x] `bash scripts/check-migrations.sh` 通过
- [x] R-103 状态 Open → Closed
- [x] Backlog T-0015 状态 planned → done

### 覆盖率（新代码）

`go test -coverprofile=/tmp/lic-cov.out ./internal/license/`：

| 文件 | 覆盖率 |
|------|--------|
| enforcer.go | 84-100%（NewEnforcer/SetCacheTTL/Invalidate/EnforcementError 100%；ActiveLicense 93.8%；EnforceCapacity 94.4%；EnforceExpiry 84.2%；Quota 86.7%）|
| monitor.go | 76-100%（NoopAlertSink/parseThresholds 100%；Start 0%（cron lifecycle）；CheckExpiry 83.3%；CheckExpiringSoon 76.5%；CheckCapacity 86.8%）|
| metrics.go | 65-80%（NewEnforcementMetrics 75%；SetCapacity 80%）|

整体 license 包 50.9%（pg_repository 占 27% 行需 PG 集成测试）。**新增 enforcer + monitor + metrics 综合 ≥ 80%**，超 70% 目标。

### 单测条数

- enforcer_test.go: 18 个 Test 函数
- monitor_test.go: 14 个 Test 函数
- 既有 service_test.go / handler_test.go 不动（fake/mock 加了 stub 让接口满足）

---

## §3 验收 GWT 自验

| GWT | 单测覆盖 | 状态 |
|-----|---------|------|
| V1 容量未满放行 | TestEnforcer_EnforceCapacity_Allowed / _AtBoundary | ✅ |
| V2 容量超限拦截 | TestEnforcer_EnforceCapacity_Exceeded | ✅ |
| V3 license 过期拦截 | TestEnforcer_EnforceExpiry_PastExpiry_Denied / _StatusExpired_DeniedEvenIfDateFuture | ✅ |
| V4 过期前告警 30/7/1d | TestMonitor_CheckExpiringSoon_30DayWarning / _7DayMajor / _1DayCritical | ✅ |
| V5 容量阈值告警去重 | TestMonitor_CheckCapacity_DedupWithinWindow / _HigherThresholdBypassesDedup | ✅ |
| V6 永久 license 跳过过期 | TestEnforcer_EnforceExpiry_Perpetual_AlwaysAllowed / TestMonitor_CheckExpiry_SkipsPerpetual | ✅ |
| V7 无 active license 默认放行 | TestEnforcer_EnforceCapacity_NoActiveLicense_DefaultAllow | ✅ |
| V8 容量配额查询 | TestEnforcer_Quota_NoActiveLicense / _PerpetualSetsDaysRemainingNegativeOne / _SubscriptionDaysRemaining | ✅ |

---

## §4 6 项决策点对齐

| 决策点 | 实施 |
|--------|------|
| D1 过期后行为 | 软告警 + 限写：device.Create 在 CreateDevice 前调 EnforceExpiry/EnforceCapacity；read 操作（List/Get/Summary/Quota）不受影响 |
| D2 容量阈值 | 三档默认 [80, 90, 95]，字段 `capacity_alert_thresholds JSONB` 可覆盖 |
| D3 多 active license | 取 max(MaxDevices)，repo.GetActiveLicenseWithMaxDevices ORDER BY max_devices DESC LIMIT 1 |
| D4 永久 license | TypePerpetual 跳过 EnforceExpiry + 跳过 daily expiry sweep |
| D5 宽限期 | grace_period_days INT DEFAULT 0（CHECK 0..365）|
| D6 校验范围 | 仅 device.CreateDevice 第一版（最小 viable）；后续 PR 扩展 template/software |

---

## §5 后续 follow-up

- [ ] 接入 template / software upgrade 拦截（后续 PR，PRD §5 非目标已声明）
- [ ] 真实 alarm sink 替换 NoopAlertSink（当前为占位，alarm.RaiseAlarm 接入需另立 task 处理循环依赖）
- [ ] license 文件签名校验（OEM 流程，非本任务范围）
- [ ] `pre-existing PgTaskRepository.scanTaskRow source_id NULL` triage（main HEAD 同样 fail，与本任务无关）

---

## §6 安全考虑

- License 校验**不可绕过**：device.CreateDevice 直接在 service 层校验，handler 透传 service error → 403
- 缓存失效**主动触发**：Service.Import/Activate/Revoke 调 invalidateEnforcerCache（不依赖 TTL 自然过期）
- Enforcer **goroutine-safe**：sync.RWMutex 保护 cachedActive
- 错误信息**不泄露敏感数据**：err.Error() 含 license_id（UUID）+ max/used 计数，不含 license_code 或 features

---

## §7 关联

- `docs/project/backlog.md` — T-0015 done
- `docs/project/risk-register.md` — R-103 Closed
- `docs/project/prd/F06-license-enforcement.md` — PRD
- `docs/methodology/AI承诺对峙清单.md` — 不在章程范围（常规 Sprint 工作，Wave 1-3 已闭门）

---

*Generated by `/dev-pipeline pick T-0015` ULTRATHINK A 方案（主会话全程深度协作）。*
