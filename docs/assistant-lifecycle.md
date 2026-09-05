# 主动智能：独立助手生命周期 v1

配套 Agent Studio PR：https://github.com/bixinhaner/agent-studio/pull/1 。本次在 `codex/active-intelligence` 上增量实现，不替换原有聊天、业务 API 或四个系统场景。

## 用户流程

进入主动智能工作区，选择“新建助手”，用自然语言描述目标。创建器读取当前用户真正可用的能力目录，返回工作方案、必要问题或能力缺口，不再固定走告警级别问答。

方案保存在服务器，聊天和说明卡引用同一份草稿。可以通过继续对话或配置表单调整。点击试运行会创建真实后台 Run、调用正式只读 API 并保存结果；结果分为“有发现”“无变化”“数据不足”。最后一次当前草稿试运行成功且数据足够，才允许确认启用。试运行本身不会启用后台监测。

已发布助手继续使用不可变版本。编辑新草稿不覆盖正在运行的版本；新草稿需要重新试运行和发布。用户可以手动检查、暂停/恢复、取消任务、看执行步骤与历史，并从首页 Attention 打开自己的新结果。

## 简单的职责边界

```
xOMC
  Assistant + Draft + PublishedVersion
  能力目录 / 业务事件 / 定时触发
  用户当前权限 / 本地执行票据 / 正式 GET API
  私有 Run 记录 / 页面 / Attention
           |
           | 通用规划请求、不可变执行快照、持久工具通道
           v
Agent Studio
  通用 AssistantPlanner
  PostgreSQL Run 队列 / 租约 / 取消
  已有 ActionConnectorRuntimeService / Codex / UsageRecorder
  有证据引用的结构化结果
```

助手业务定义只由 xOMC 维护。Studio 不提供另一份可独立编辑的业务配置，而是在接收运行请求时冻结快照。无需新消息中间件、新 Harness、工作流画布或多 Agent 编排。

## 第一版支持范围

- 个人、GET-only、站内结果；不修改设备，不自动重试业务任务。
- 范围为创建者当前授权可见资源，或一台实际存在且有权限的设备。
- 通用触发包括手动、至少每 5 分钟的周期检查、指定星期/本地时间/IANA 时区、现有重要告警和任务失败事件。
- 能力目录基于运行中的 Handbook 和正式接口，第一版发布设备与告警领域的安全读取能力。没有提供的历史统计、设备组名字解析、性能指标、团队共享、邮件等外部投递，不会因为用户说一句话而自动获得。
- 连续离线时长、历史对比等需求必须有相应时间/历史证据。系统应返回能力缺口，不能偷偷用当前快照或一次采样替代。
- 初始最多 100 个个人助手。运行历史接口返回最近记录；归档删除、长期保留策略和大规模查询分页属于后续能力。

扩展新业务时，增加 xOMC 的能力语义、真实安全 API/绑定或事件描述即可；不要向 Studio 的新助手路径添加业务场景名称 switch。旧内置场景仍保留自己的兼容实现，不能据此宣称其所有历史问题已修复。

## 代码入口

| 模块 | 位置 |
| --- | --- |
| 助手模型、校验、定时计算 | `omcgo/internal/agentassistant/model.go` |
| 草稿/发布事务、版本、租约、私有记录 | `agentassistant/repository.go` |
| 当前身份、权限、范围与路径绑定 | `agentassistant/security.go` |
| 规划、触发与远端执行同步 | `agentassistant/service.go` |
| 当前运行 API 能力发布 | `agentruntime/assistant_capabilities.go` |
| 远端客户端与本地工具执行 | `agentbridge/assistant_client.go`、`tool_worker.go` |
| 页面与组件 | `omcmb/webcode/src/pages/system/ActiveIntelligence/` |
| API/types/文案 | `omcmb/frontend-core/src/` 中 assistant 相关文件 |

## 本地接口

前缀 `/api/v1/agent/assistants`，使用既有登录认证。操作对象必须属于当前用户；列表、结果和通知不会返回其他用户助手。

| 方法与路径 | 用途 |
| --- | --- |
| `GET /catalog` | 当前可用能力、事件和连接配置状态 |
| `GET /`、`POST /` | 列表、幂等创建空草稿（客户端 UUID） |
| `GET /:id` | 草稿、发布版本、运行状态 |
| `POST /:id/messages` | `{revision,message}`，调用真实规划器并 CAS 保存 |
| `PUT /:id/draft` | `{revision,definition}`，服务端验证配置后保存 |
| `POST /:id/publish` | `{revision}`，校验当前版本真实试运行门槛并发布 |
| `PATCH /:id/state` | `{state: active|paused}` |
| `POST /:id/runs` | `{revision,kind: trial|manual,requestId}` |
| `GET /:id/runs` | 私有运行历史 |
| `GET /runs/:runID` | 私有结果与工具进度 |
| `POST /runs/:runID/cancel` | 请求取消，保持 CANCELLING 直到远端确认 |
| `POST /runs/:runID/read` | 标记自己的结果已读 |

沿用统一 response 包装。409 版本冲突不会覆盖旧配置；前端保留未发送文本并要求重新确认。规划有独立的有界超时，超时不会发布或破坏草稿。

## 数据与安全

按本仓库未封版本规则修改 `omcgo/migrations/000001_init_schema.sql`，新增 `agent_assistants`、`agent_assistant_versions`、`agent_assistant_runs`，不新增 000002 迁移。菜单权限同步到既有基线 seed，不因此授予额外业务 API 权限。

后台执行绑定助手拥有者和确认时的角色、授权范围摘要。每次查询重新检查用户仍有效、角色仍分配、接口仍允许、范围仍成立。授权范围变化会阻止旧执行和旧结果读取，需要重新确认。单设备查询由服务端从可信票据绑定设备 ID，不信任模型提供的任意路径。原有 ToolExecutor 的 operationId 契约校验和正式 API 权限中间件仍执行。

本地和远端使用同一稳定 runId，重试不会创建第二份执行。已排队、运行或取消中的助手不重叠启动；定时到期和事件投递具有幂等键。取消请求先确保远端存在该幂等任务再取消，避免网络乱序导致取消后又创建执行。结果只进私人 ledger，不混入旧的全局 Finding 概览。站内提醒和数据访问权限分别处理。

`catalog.connected` 表示连接配置齐备，不等于远端模型健康；真正的连通性和数据可用性由规划/试运行验证。连接变化或断开不会生成固定的假成功结果。

## 验证与上线

```
bash scripts/ci/assistant-lifecycle.sh backend
bash scripts/ci/assistant-lifecycle.sh frontend
bash scripts/ci/assistant-lifecycle.sh browser
```

后端测试使用临时 PostgreSQL，验证草稿、版本、发布门槛、权限、幂等、租约、取消和时区。前端执行类型检查、目标单测和生产构建。Playwright 挂载真实生产组件，使用隔离的 API 测试夹具验证交互，并生成桌面、移动端、暗色主题截图；这些夹具不进入生产页面逻辑。

CI 通过不等于生产 Codex 联调通过。部署顺序为：先部署配套 Studio 与 Prisma migration，再部署 xOMC 并按既有基线 reconciliation 流程更新 schema/seed。勿仅部署一端。此次 PR 不合并、不自动部署生产。

发布负责人还需在真实环境验证：一个自然语言方案、一轮真实工具试运行、关闭浏览器后的定时执行、私人通知隔离、撤权后拒绝执行，以及取消后的终态保持。回滚前先暂停新助手并处理未完成任务，保留数据库记录用于排查；原系统场景可继续使用。
