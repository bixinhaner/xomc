# xOMC 主动智能集成

本目录保存 xOMC 拥有的 Agent Studio 集成契约与业务 Pack。设计目标不是让模型直接控制 OMC，而是建立一条可追踪、可恢复、默认只读的主动分析链路：

`OMC 业务事件 → 可靠 Outbox → Agent Studio 后台 Run → OMC 本地只读 Tool Worker → Finding → Attention / Finding Drawer → 用户继续询问 Agent`

## 当前可运行范围

- 已打通：`omc.task.failed.v1` → `task-failure-analysis`。
- 已纳入 Pack、但运行时尚未启用：接入审核、严重告警解释、每日运维摘要。
- 后台调查只允许本地白名单中的 `GET` 操作；远端返回的字面路径不会直接执行。
- Finding 必须能从任务或设备资源解析出当前用户的数据可见范围，否则拒绝投递。
- Finding 只能生成分析与建议；任何写操作仍需进入 AgentPanel，由用户确认。

之所以只启用已具备完整事件、权限、证据和交付链的场景，是为了避免把“模型能生成文字”误当成“系统已经可靠主动分析”。其余场景应在各自业务事件 Outbox、最小只读操作集和可见性规则齐备后逐个启用。

## 配置与启动

1. 在 Agent Studio 数据库应用 `20260827120000_add_proactive_action_connector` 迁移。
2. 启动升级后的 Agent Studio API。
3. 在 OMC 管理端配置并同步 Agent Studio 地址和服务令牌。代码不再提供内置默认令牌；未配置时桥接 fail closed。
4. 应用 OMC 基线数据库结构并启动 OMC。事件转发、工具租约和 Finding 租约 Worker 常驻运行，配置同步后无需重启 OMC。

生产验收还必须验证：实际数据库迁移结果、两端进程健康、连接状态、真实任务失败事件、Tool 调用证据、Finding 可见范围，以及浏览器中的 Attention → Drawer → AgentPanel 全链路。

## 目录

- `contracts/`：两端共享的 JSON Schema、OpenAPI 和示例。
- `integration-pack/xomc/`：xOMC 场景、提示词、输出 Schema、展示规则和回归样例。

Pack 内容与 Starter Kit v1.0 保持逐字节一致；运行时通过已发布 ZIP 的 SHA-256 摘要固定版本。
