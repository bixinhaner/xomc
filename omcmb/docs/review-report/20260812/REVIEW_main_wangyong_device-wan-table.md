# BLQ/MLQ WAN 表格改动审查报告

- 日期：2026-08-12
- 基线：`main`
- 分支：`fix/blq-mlq-wan-table`
- 范围：设备详情 → 快速设置 → 网络设置
- 结论：PASS

## 审查摘要

本次改动将 BLQ、MLQ 的 WAN 配置切换为与 MLN 一致的表格操作形式，同时由各站型 QuickSettings 配置决定实际行数和字段。BLQ、MLQ 使用 WAN 1–12；MLN、BLN 保持既有 WAN 1–4 和静态路由配置。BLQ 的“快速接口绑定”恢复为根据连接类型、启用 WAN 和 VLAN 动态生成选项。

## 检查结果

- CRITICAL：无。
- WARNING：无。
- INFO：WAN 表格固定约四行高度，超出部分内部滚动；这是明确的交互要求。
- INFO：设备实采数据确认 WAN1 不暴露启停参数，因此仅按业务规则显示默认 `ON`，不向设备下发虚构参数；WAN2 以后显示设备真实值。
- INFO：默认路由 DNS 添加区采用独立输入框和按钮，保留统一加号图标并增加间距。
- 类型安全：新增逻辑保持现有 `QuickSettingsGroup`、`QuickSettingsParam` 和 schema 类型约束，无 `any` 扩散。
- 安全：无 HTML 注入、凭据处理、鉴权或 SQL 变更。
- 兼容性：未修改 API；站型差异由 QuickSettings 数据驱动，MLQ 不额外暴露旧页面不支持的 TR-069 绑定项。

## 验证

- `cd omcmb && npm run typecheck`：通过。
- `cd omcgo && go test ./internal/quicksettings`：通过。
- `npm run build`：通过。
- Docker Compose 本地热部署：通过。
- `curl -I http://localhost:8081/`：`HTTP/1.1 200 OK`。
- BLQ/MLN 实机参数库核对：WAN1 无 Enable 参数；其余已支持 WAN 的 Enable 参数按设备实采结果保留。

## 风险与回滚

- 风险集中在不同设备实际上报的 WAN Enable/VLAN 值；当前处理与旧页面生成规则一致，并保留当前绑定值作为兼容选项。
- 可通过回滚本提交恢复原有卡片式展示。
