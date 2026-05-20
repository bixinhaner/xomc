# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-20 09:43 |
| 提交 | f08a5e48 |
| 作者 | zhanglu |
| 范围 | device |
| 变更文件数 | 2 |
| 新增行数 | +391 |
| 删除行数 | -32 |

## 变更概要

本次变更把设备详情参数树相关的三个读取端点统一到“产品绑定的默认参数模型 + device_parameters 当前值叠加”的显示契约，避免 UI 只看到已发现参数而缺失 OMC 内置参数清单。
同时补充了聚焦测试，覆盖 tree、children 和 schema 三条读链路中 model-only 参数的可见性。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

无

## 详细分析

### `omcgo/internal/device/device_param_handler.go`

- `GetParameterTree`、`GetDirectChildren`、`GetParameterSchema` 现在统一通过 `resolveDefaultMappingValidator` 和 `buildDisplayParameters` 构造展示集，和参数树 UI 约定保持一致。
- 设备不存在、仓库读取失败等路径均保留了明确的错误返回，未引入新的资源泄漏或越权风险。
- `LastUpdatedAt` 的零值处理从固定格式化改为条件输出，避免向前端暴露无意义的零时间戳字符串。

### `omcgo/internal/device/device_param_handler_test.go`

- 测试覆盖了 tree、children、schema 三个端点对 model-only 参数的展示行为，验证默认参数模型叠加契约已落地且三条链路一致。
- 断言同时覆盖了空值参数、可写性和类型字段，能够防止后续再次回退到“只显示已发现参数”的行为。

## 业务完整性检查

- Handler-Service-Repository 链路：通过，未引入空壳路径或跨层短路。
- 路由注册：通过，本次未新增路由，仅修正既有路由的显示逻辑。
- 迁移文件配套：不涉及 schema 变更，无需 migration。
- 错误码注册：不涉及新增业务错误。
- API / Hook / Mock 配套：不涉及前端接口契约变更。
- 单元测试配套：通过，已补齐同切面的 focused test。
- E2E 配套：本次为后端读取契约修正，聚焦单测已覆盖核心行为。

## 业务影响范围检查

- 接口签名变更：无。
- 数据库 Schema 变更：无。
- 事件契约变更：无。
- 共享 model 变更：无。
- 配置项变更：无。
- API 响应格式变更：无新增字段；`total/items/tree/parameters` 仍保持原有结构，仅内容来源收敛为默认参数模型叠加。

## 验证记录

- `cd omcgo && go test ./internal/device -run 'TestParameterTreeHandler_UsesDefaultParamModelForTreeAndChildren' -count=1` ✅

## 审查结论

PASS