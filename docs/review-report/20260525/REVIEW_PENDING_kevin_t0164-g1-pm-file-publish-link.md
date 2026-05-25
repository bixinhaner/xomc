# 代码审查报告 — T-0164 G1 PM 文件解析链路打通

| 字段 | 值 |
|------|----|
| 审查时间 | 2026-05-25 |
| 审查人 | kevin (AI 辅助) |
| Commit Hash | _PENDING_ |
| Backlog | T-0164 (G1 真机闭环 publish 链路) |
| Scope | pm, acs, deploy |
| Type | fix |
| 审查结论 | **PASS_WITH_WARNINGS** |

---

## 背景

T-0164 G1 真机验证暴露：MinIO 已收 8+ 个 PM XML 文件，但 `pm_metrics` / `pm_metrics_hourly` / `pm_metrics_daily` 三表全部 0 行。

根因（已 Explore agent 定位）：`internal/acs/upload/handler.go` ACS 上传 handler 收到 FileType=PM 文件后 **未 publish 任何事件**（DataModel / Config / Log 都 publish 了，唯独 PM 漏）；`pm.Collector` 订阅的 `pm.file.received` 永远收不到 → 不解析 → `pm_metrics` 永空。

另一条 `transfer.Bridge` 路径（订阅 `device.inform.autonomous_transfer_complete`）逻辑正确，但当前 CPE 走的是直接 HTTP POST `/smallcell/FileUploadService?fileType=PM` 上传，不经过 bridge。

本次修复补齐缺口：ACS 收到 FileType=PM 后 publish 瘦 payload，collector 用 `DeviceLookup` 反查回填 UUID/OUI/Carrier/Technology。

## 变更范围

| 文件 | 性质 | 行数 |
|------|------|------|
| `omcgo/internal/acs/upload/handler.go` | 修改 — 新增 PM 事件发布 + SN 提取 | +91 |
| `omcgo/internal/pm/collector/collector.go` | 修改 — 新增 DeviceLookup 接口 + resolveDevice 助手 | +88/-10 |
| `omcgo/cmd/worker/main.go` | 修改 — wiring SetDeviceLookup | +3 |
| `omcgo/internal/acs/upload/handler_test.go` | 修改 — 4 个新测（empty SN / thin payload / SN 提取 6 用例 / publish err 不 panic） | +139 |
| `omcgo/internal/pm/collector/resolve_device_test.go` | **新增** — 6 用例（fat passthrough / thin fill / no lookup err / empty SN err / not found err / lookup err prop） | +109 |

合计：5 文件，+311 / -10。

## 审查发现

### CRITICAL — 无

### WARNING

1. **publish 失败时静默丢弃数据**（`handler.go:publishPMFileReceivedEvent`）
   - 行为：NATS 不可用时 publish 失败 → 仅 log error，upload 仍返回 200 → collector 永远收不到这次的事件 → 这一份 PM 数据丢失
   - 设计意图：注释里已说明（"5xx on a successful MinIO write is worse than a silently-skipped event"），且 CPE 下次周期会重新上传
   - 风险：NATS 持续故障期间 PM 数据持续丢失（依赖 CPE 周期重传容错）。如果改为返回 5xx，CPE 可能进入异常重试链路
   - 结论：可接受，但需在 release-gate 前补 NATS 健康度告警（Prometheus alert: `nats_outbound_lag > N` 或 publish error rate > 0）

2. **`pmFilenameSNRe` 仅覆盖两种厂商命名**（`handler.go`）
   - 当前：Baicells `A{ts}_{OUI}.{SN}.xml(.gz)?` + simulator `pm-{SN}.xml(.gz)?`
   - 风险：商用化后新增厂商 CPE 文件名不匹配 → SN 提取失败 → publish skip → 数据丢失（且只有 WARN log，无监控）
   - 缓解：代码注释明确"新 vendor 走分支扩展"，下一个 vendor 接入前补
   - 建议：补一个 Prometheus 计数器 `pm_publish_skipped_total{reason="empty_sn"}` 让 SN 提取失败可观测

3. **DeviceLookup 失败走 retry+DLQ，依赖 DLQ 监控就绪**（`collector.go:resolveDevice`）
   - 行为：device 未注册（CPE registration 还没完成）/ PG 抖动 → 返回 error → runner.Wrapper retry 3 次 → 进 DLQ
   - 注释里说"transient retries usually succeed"，符合 inform/registration race 场景
   - 真实风险：DLQ 里堆积的 PM 文件需要人工 replay，需确认 DLQ 监控 + replay 机制就绪（T-0164 G8 框架应该已经覆盖）

### INFO

1. **`handler_test.go` 末尾 `_ = time.Millisecond` 是 keep-import-alive hack**
   - 既然测试没真用 `time` 包，应该直接删 `import "time"` 而不是用 `_=` 占位
   - 不阻塞合并，但下次顺手清掉

2. **`FileReceivedPayload` 用 `omitempty` 共享 thin/fat 两种 payload — 干净**
   - 两条 publish 路径（transfer.Bridge 胖 / acs.upload 瘦）共享同一结构，靠 `omitempty` + collector 端 resolveDevice 兜底，没有引入新类型，降低维护成本

### 规范对照

| 维度 | 结论 |
|------|------|
| 命名规范 | ✓ exported/unexported 一致，包名/接口名符合规范 |
| 错误处理 | ✓ `fmt.Errorf("...: %w", err)` wrap，无裸 panic |
| 接口优先 | ✓ `DeviceLookup` 新接口（最小，仅 1 方法），避免 collector 依赖 device 包内部 |
| SQL 安全 | N/A（本次无 SQL 改动） |
| 运营商硬编码 | ✓ 无 `if carrier == "cmcc"`；正则区分 vendor 是文件名格式区分，不是 carrier 区分 |
| 测试覆盖 | ✓ 成功 + 失败两路径全覆盖（10 用例：4 handler + 6 collector） |
| 日志质量 | ✓ 结构化 `zap.String/Int64`，含 device_sn / path / size 关键字段 |

## 真机回归证据

- 部署：~16:53 BJ `bash /Users/shangyingbin/project/omc-docker/docker-run.sh`
- 首次上传：17:00:01 BJ（CPE 15 分钟周期）
- ACS 侧：✓ `published pm.file.received device_sn=1202000240194DP0015 size=166892`
- Collector 侧：✓ `parsed PM file counters=2048`
- INSERT 侧：✗ 撞 BUG-6 `SQLSTATE 21000 ON CONFLICT DO UPDATE command cannot affect row a second time`
  - BUG-6 是独立问题（`pm_metrics` 自然键漏 `object_ldn`），由下一个 commit 闭环

## 结论

**PASS_WITH_WARNINGS** — 可合入。WARNING 项中 #1 和 #2 需要在 release-gate 前补可观测性（NATS 告警 + skip 计数器），但不阻塞本次提交。BUG-6 是下游问题，与本 commit 解耦。

## Follow-up

- [ ] BUG-6：`pm_metrics` 自然键加 `object_ldn`（下一 commit）
- [ ] 补 Prometheus 计数器 `pm_publish_skipped_total{reason}` 让 SN 提取失败可观测
- [ ] NATS publish error rate 告警进 release-gate.md §8.5
- [ ] 商用化前列 CPE 厂商 PM 文件名格式清单（Baicells / Huawei / ZTE / ...）
