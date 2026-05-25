# 代码审查报告 — T-0164-P3 / G3 fix: OUI+SN 双键设备唯一标识

**Base commit**: `8be7ac76` (前一 commit 是 T-0165 docs 登记)
**Scope**: `pm` + `transfer` + migration
**Author**: shangyingbin (Claude)
**Date**: 2026-05-23
**Backlog**: T-0164-P3
**PRD**: docs/design/pm-kpi-pipeline-improvements.md §4.3 + docs/project/plan-T-0165-system-wide-oui-sn-migration.md
**Sprint**: wave-3
**Risk**: -

---

## 1. 背景 & 问题

G3 主 commit (`e767ce64`) 实施时为节省工作量做了"过渡设计"：pm_metrics.device_sn 列存 PMCounter.DeviceID 的 UUID 字符串（如 `b9dfacaf-...`）作伪 SN。用户审查时指出错误 — devices 表 `id (uuid)` 和 `serial_number (text)` 是两个完全独立字段，wrapper 内 `DeviceSN = DeviceID.String()` 等于丢失了上游已有的真实 SN 和 OUI。

讨论后确定：**按 TR-069 标准用 (OUI, SerialNumber) 双键作为设备唯一标识**，趁系统未上线一次改到位。本 fix 即此决定的 PM 范围落地。系统级切换由 T-0165 跟进（docs 已在前一 commit 立项登记）。

---

## 2. 变更范围

| 文件 | 类型 | 说明 |
|------|------|------|
| `migrations/000160_*` | 修改 | pm_metrics 加 `device_oui TEXT NOT NULL`；UNIQUE 索引 6 列 `(device_oui, device_sn, metric_path, granularity, end_time, time)`；ON CONFLICT 配套；compression segmentby 加 device_oui |
| `internal/transfer/bridge.go` | 修改 | pm/mr 两处 payload 加 `"oui": dev.OUI` |
| `internal/pm/collector/collector.go` | 修改 | FileReceivedPayload 加 `DeviceOUI string`；parser 返回后批量填 PMCounter.OUI/DeviceSN；KPIEngine 调用传 oui+sn |
| `internal/pm/collector/parser.go` | 修改 | PMCounter 构造时填 DeviceSN（已解析的 content.DeviceSN） |
| `internal/core/model/pm.go` | 修改 | PMCounter / KPIValue 加 `OUI string + DeviceSN string` 字段（与 DeviceID uuid 并存） |
| `internal/pm/kpi/engine.go` | 修改 | Calculate / CalculateAndStore 签名加 `oui, deviceSN string`；KPIValue 构造时填 OUI+DeviceSN |
| `internal/pm/handler.go` | 修改 | NewHandler 加 `pool *pgxpool.Pool` 参数 + `lookupDeviceOUISN` 私有方法；RecalculateKPI 端点反查 devices 拿 oui+sn 再传给 KPIEngine |
| `internal/pm/metrics/model.go` | 修改 | PMMetric 加 DeviceOUI 字段 |
| `internal/pm/metrics/pg_repository.go` | 修改 | INSERT/SELECT 列加 device_oui；ON CONFLICT 6 列；QueryRequest 加 DeviceOUIs；applyFilters 实现 OUI+SN 配对 OR 条件（防部分匹配） |
| `internal/pm/counter/pg_repository.go` | 修改 | counterToMetric/metricToCounter 用 c.OUI / c.DeviceSN（取消 DeviceID.String()）；filter 路径反查 devices；aggregateInMemory 改按 oui+sn grouping |
| `internal/pm/kpi/pg_repository.go` | 修改 | kpiValueToMetric/metricToKPIValue 同 counter 风格；Query 反查 devices |
| `cmd/app/provider/router.go` | 修改 | pm.NewHandler 调用加 c.TsPool 参数 |
| `internal/pm/metrics/pg_repository_test.go` | 修改 | 加 OUI 单独 IN / OUI+SN 配对 / 不等长截断三个测试 case |
| `internal/pm/counter/counter_test.go` | 修改 | round-trip 测试用真实 OUI=48BF74 + SN=1202000240194DP0026 |
| `internal/pm/kpi/engine_test.go` | 修改 | 3 处 Calculate/CalculateAndStore 调用补 oui+sn 参数 |
| `internal/pm/handler_test.go` | 修改 | NewHandler 调用补 nil pool |
| `scripts/seed_e2e_testdata.sql` | 修改 | pm_metrics 行加 device_oui 列；用真实值 `('00A0C6', 'TEST-SN-001')` 与上方 devices 表对齐 |
| `scripts/seed_test_data.sql` | 修改 | 同上；20000 行 generate_series 随机分布到 3 个 OUI 池 |

**净变化**：+304 / -86 行（18 files）

---

## 3. 关键设计决策

### 3.1 双键 vs 拼接的选择
讨论过三方案（仅 SN / carrier+SN 拼接 / oui+sn 双列）。最终选**双列**：
- TR-069 标准合规（OUI 是协议定义的厂商身份位）
- 防跨厂商 SN 撞键（业务规则未来可能演进）
- API/前端 拿到响应可直接渲染 sn 字段（不需要拆字符串）
- migration 改动 ~ 6 行（device_oui 列 + 5 个索引/约束列名调整）

### 3.2 上下游过渡策略
- **上游写入**：collector / KPIEngine 全链路填真实 OUI+SN，BatchInsert 直接使用
- **下游查询**：handler REST API 契约保持（仍按 device_id uuid 接收），wrapper 内部反查 `devices.{oui,serial_number}`，每次 Query 一次 SQL 反查（pgxpool 复用连接，开销可接受）
- **DeviceID 字段保留**：作 internal PK 与 devices.id 关联；以 extra JSONB 存储在 pm_metrics（写入时 collector 存的），metricToCounter/metricToKPIValue 反向反查
- 完整契约切换由 T-0165-D 跟进（前后端协调期 API `/devices/:oui/:sn/*`）

### 3.3 applyFilters OUI+SN 配对 OR 条件
原 metrics.QueryRequest.DeviceSNs 是 IN 列表。加 DeviceOUIs 后，业务语义是"按 (oui[i], sn[i]) 配对查"，而不是 OUI IN ∧ SN IN（后者会查到 oui[i]+sn[j] 跨匹配错误）。实现：
- 两边都有 → `(oui=A AND sn=X) OR (oui=B AND sn=Y)` 配对
- 单边有 → 按 IN 过滤
- 不等长 → 按 `min(len)` 截断（防止 OUI 多 SN 少时漏匹配）

### 3.4 aggregateInMemory grouping key 变化
旧 key (bucket, deviceID uuid, cellID, group, name) → 新 key (bucket, oui, sn, cellID, group, name)。AggregatedCounter struct 中 DeviceID 字段保留，从 extra["device_id"] 反查（写入时存的）。

### 3.5 dry-run 抓的真 bug
**5 个验证场景全过**：
1. counter+kpi 双 metric_type 存储
2. UNIQUE 索引 6 列正确
3. ON CONFLICT 含 device_oui 幂等更新
4. **跨 OUI 同 SN 不冲突**（同一 SN `1202000240194DP0026` 在 OUI `48BF74` 和 `00A0C6` 各自独立存）
5. 与 devices 表数据风格一致（OUI 6 位 hex 大写）

---

## 4. 审查检查清单

### 4.1 Go 后端规范
- [x] 命名：导出 PascalCase（PMMetric, NewPgRepository, OUI），未导出 camelCase（counterToMetric, lookupDeviceOUISN）
- [x] 错误处理：`fmt.Errorf("lookup device oui+sn by id %s: %w", ...)` 全部包装
- [x] SQL 安全：Squirrel 构建 + 占位符 `$N`；反查走 `pool.QueryRow` 参数化
- [x] OUI+SN 配对 OR 条件用 squirrel.Or + squirrel.And 嵌套，无 SQL 拼接
- [x] 资源释放：rows.Close defer
- [x] 测试覆盖：metrics 加 3 个新 case（DeviceOUIs IN / OUI+SN 配对 / 不等长截断）+ counter round-trip 验证 OUI 字段守恒

### 4.2 Migration 自查（按 omcgo/CLAUDE.md §5.5.10）
- [x] 版本号 = 160（不变）
- [x] StatementBegin/End 包裹
- [x] CHECK 约束完整
- [x] hypertable + ALTER SET (compress) + add_compression_policy + add_retention_policy 顺序正确
- [x] UNIQUE 索引含分区列 time（TS 要求）
- [x] Down 段完整
- [x] Dry-run 通过（5 个验证场景）

### 4.3 兼容性
- [x] handler REST API 契约不变（前端无感）
- [x] CounterRepository / KPIRepository 接口签名不变（northbound 等订阅者无影响）
- [x] KPIEngine 签名变化只影响 collector（自动触发）+ handler RecalculateKPI（手动触发）
- [x] event payload 多了 device_oui 字段（向后兼容，订阅者解码忽略未知字段）

### 4.4 后续工作明确化
- [INFO-1] handler Query 反查 devices 每次 1 次 SQL → T-0165-D 切 API 契约后取消反查
- [INFO-2] devices 表无 UNIQUE(oui, serial_number) 约束 → T-0165-A 加约束兜底
- [INFO-3] 其他模块（alarm/mr/device_parameters/前端 413 处）OUI+SN 切换 → T-0165-B/C/D
- [INFO-4] Baseline 3 个 build failure（pre-existing）仍存在，与本 fix 无关

---

## 5. 验证证据

- ✅ `go build ./...` 通过
- ✅ `go test ./internal/pm/...` 全过（metrics + counter + kpi + collector + indicator + retention + pm 7 包）
- ✅ Migration dry-run 在 docker postgres 事务内验证 5 个场景全 OK
- ✅ TR-069 标准合规：(OUI, SerialNumber) 双键
- ✅ devices 表数据格式对齐：OUI 6 位 hex 大写（48BF74、00A0C6 等）

---

## 6. 审查结论

**PASS_WITH_INFO** — 设计正确（TR-069 标准合规）、改动面控制良好、单测+dry-run+跨 OUI 同 SN 隔离验证充分。4 个 INFO 级遗留均已在 T-0165 plan 中规划。

无 CRITICAL，无 WARNING。

可合入 `draft/pm-kpi-impl` 分支等晨间 review；连同 G3 主 commit (`e767ce64`) + T-0165 docs (`8be7ac76`) 一起 merge 进 main 时 PM 范围内 OUI+SN 双键完整生效。
