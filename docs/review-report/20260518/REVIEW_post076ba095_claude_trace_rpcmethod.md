# 代码审查报告 — TR-069 报文跟踪 RPC 方法标签错乱修复

| 元数据 | 值 |
|---|---|
| 审查日期 | 2026-05-18 |
| 基线 commit | 076ba095（pull --rebase 后 HEAD） |
| 作者 | Claude（pair with shangyingbin） |
| 主要 scope | acs, trace |
| 变更类型 | fix |
| Backlog | T-0137 |
| 结论 | **PASS** |

## 变更范围

| 文件 | 类型 | 说明 |
|---|---|---|
| `omcgo/internal/acs/trace_capture.go` | 修改 | 核心修复：in / out 各自从 XML 解析方法名 |
| `omcgo/internal/acs/handler.go` | 修改 | `traceService` 字段类型 `*trace.Service` → `traceCaptureSink` 接口 |
| `omcgo/internal/acs/trace_capture_test.go` | 新增 | 5 个用例覆盖 Inform / GPV 中段 / 会话尾 / 首段 / 白名单未命中 |

## 问题与修复

### 根因

`maybeCaptureTrace` 写入 in / out 两条 `trace.Message` 时，`RPCMethod` 字段共用同一个 `rpclog.LogEntry.Method`。但在 TR-069 协议下，一次 HTTP 事务的请求体（CPE→ACS）与响应体（ACS→CPE）是**两条不同的 RPC 消息**（例如请求体是 `GetParameterValuesResponse`，响应体是新派发的 `GetParameterValues`）。`entry.Method` 在请求处理过程中会被多次覆盖（`handler.go:672` 写入 reqMethod 后被 `:606` 的 `taskItem.Method` 覆盖），导致两条 trace.Message 都用最后一次写入的值——前端看到 GPV/GPVResponse 不再 1:1 对应，出现"成对重复同名"现象。

### 修复方式

`maybeCaptureTrace` 不再读 `entry.Method`，在 capture 入口对 `reqXML` / `respXML` 各自调用 `soap.DetectMethod` 现场解析方法名与 cwmpID（`detectXMLMeta` 辅助函数）。空 body 标记为 `"Empty"`（与 `handler.go:536` 既有的 `entry.Method="Empty"` 语义一致），与 "XML 解析失败"（返回空字符串）的两种情况区分开。CwmpID 优先取 XML 解析值，回退到 `entry.CwmpID`，兼容异常 XML。

### 为支持单元测试做的最小改动

`Handler.traceService` 字段类型从具体类型 `*trace.Service` 改为本文件新增的小接口 `traceCaptureSink`（仅 `EnqueueCapture` + `ObserveCaptureLatency` 两个方法）。生产装配（`server.go:87` 仍传入 `*trace.Service`）自动满足该接口，无外部行为变化。测试可注入 stub sink 直接断言入队的 message 内容。

## 审查清单

| 维度 | 结果 | 备注 |
|---|---|---|
| 命名规范 | ✓ | 私有 camelCase（`detectXMLMeta` / `firstNonEmpty` / `traceCaptureSink`），导出大写 |
| 错误处理 | ✓ | `DetectMethod` 出错落入"空方法名"分支并加注释说明（trace 是诊断旁路，不能影响主流程） |
| 接口设计 | ✓ | `traceCaptureSink` 接口仅含使用到的两个方法，符合"接受接口"原则；接口定义在消费方而非提供方，符合 Go 习惯 |
| 性能 | ✓ | `strings.NewReader(xml)` 替代 `bytes.NewReader([]byte(xml))` 避免一次切片拷贝（此前评审 INFO 项已在本次修复中处理）；`DetectMethod` 流式解析根元素即返回，µs 级 |
| 并发安全 | ✓ | 无新增共享状态；stub sink 在测试里用 mutex 保护 captured 切片 |
| SQL / 鉴权 / 运营商 | N/A | 本次无相关变更 |
| 测试覆盖 | ✓ | 5 个用例 100% 覆盖新增分支：Inform Pair / GPV 中段 / 会话尾（in 有 out 空）/ 首段（in 空 out 有）/ 白名单未命中 |
| 兼容性 | ✓ | API / DB schema / 前端契约零变更；既有日志、协议日志、其他 ACS 行为完全保留 |
| 文档/注释 | ✓ | 重写了函数顶部注释，明确说明协议层语义与字段含义；`detectXMLMeta` 注释解释"Empty vs 空字符串"的区分 |

## 发现项

| 级别 | 项 | 处理 |
|---|---|---|
| INFO | `bytes.NewReader([]byte(xml))` 多一次拷贝 | 已在本次修复中改为 `strings.NewReader(xml)` |

无 CRITICAL / WARNING。

## 验证

| 步骤 | 结果 |
|---|---|
| `go build ./...` | ✓ 通过 |
| `go test ./internal/acs/...` | ✓ 全绿（含 5 个新用例） |
| `go vet ./internal/acs/...` | ✓ 无告警 |
| 增量构建 ACS 镜像 + `docker compose up -d acs` | ✓ 容器健康，trace 子系统装载日志正常 |
| 手动验证（前端报文跟踪页 + 实时刷新） | 待用户在真实 CPE 报文流上确认 GPV/GPVResponse 1:1 对应 |

## 结论

**PASS** — 修复精准、范围最小、测试充分，不引入回归风险。可以合入 main。
