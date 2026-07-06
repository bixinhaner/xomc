# goomc AI Agent 组件化接入设计

日期：2026-07-06

## 结论

goomc 的 AI Agent 采用“统一 Agent Runtime + goomc Action API + 可嵌入 AgentKit UI”的方案。

- Agent Runtime 由 `agent-studio` 或后续抽出的 `agent-gateway` 承担。
- goomc 不内置完整 runtime，只提供安全业务 Action API、前端上下文适配和 agent 侧专有 skill。
- goomc Web 入口是全局 Agent Panel，运行在当前业务页面上下文中。
- 不使用 MCP。goomc 业务侧通过 API-first 控制，CLI 只作为 sidecar/运维 fallback。
- 共享组件使用中性包名，不绑定公司或产品品牌。

## 设计目标

1. 用户在 goomc 当前页面即可获得懂 OMC 业务的助手，不需要跳到孤立聊天页。
2. Agent 调用必须继承当前用户权限、设备组可见范围、审计和业务校验。
3. 写操作必须先 preview，再由用户在 UI 中确认。
4. Agent UI、协议和 runtime client 可被 goomc、crest 或其他 Web 项目复用。
5. 私有化部署可把 runtime 作为 sidecar 放进 goomc 部署栈旁边，但协议不变。

## 非目标

1. 首期不做 goomc 内置完整 Agent Runtime。
2. 首期不把所有 goomc REST API 原样暴露给模型。
3. 首期不允许 Agent 自动执行高风险动作。
4. 首期不使用 MCP 作为 goomc 业务控制协议。

## 已参考项目

### agent-studio

`agent-studio` 适合作为统一 Agent Runtime/工作台：

- 管理会话、线程、模型/provider、skill、文件、流式输出、用量和审计。
- 已有 Codex runtime、provider 管理、skill 包和集成中心。
- 不应把这套 runtime 复制进 goomc。

### crest-v2

`crest-v2` 最值得复用的是 Action API 思路，而不是 MCP：

- 业务系统保留事实源、权限、状态机和审计。
- Agent 只调用语义 Action，不代理 raw endpoint。
- 写操作 preview -> confirmationToken -> execute。
- SSE 事件支持 `action_preview` 和 `ui_intent`。
- UI intent 可驱动页面跳转、刷新和高亮。

### goomc

goomc 已具备适合承载 Action 层的基础：

- 后端已有 JWT/API Key、Casbin、端点级权限、审计和设备组可见范围。
- `omcctl` 是 `X-API-Key` 调 `/api/v1` 的薄客户端，适合作为 CLI fallback。
- 前端是 `frontend-core` + 三套皮肤，Agent Panel 必须统一接入并保持三皮肤风格一致。

## 总体架构

```text
goomc Web
  └─ @agentkit/react AgentPanelHost
     └─ @agentkit/runtime-client
        └─ Agent Runtime (agent-studio / agent-gateway / sidecar)
           └─ @agentkit/action-skill
              ├─ goomc Action API (primary)
              └─ omcctl --output json (fallback)

goomc Backend
  ├─ POST /api/v1/agent/delegation
  └─ /api/v1/agent-actions/*
```

边界：

- Agent Runtime：线程、模型、skill、文件、流式事件、用量、runtime 侧审计。
- AgentKit：协议、client、React UI，项目无关。
- goomc Action API：安全业务动作，权限和审计在 goomc 内闭环。
- goomc Web Adapter：页面上下文采集和 `uiIntent` 执行。
- Action Skill：把自然语言任务映射到通用 Action API/CLI。Agent Runtime 侧不写死 goomc 名称。

## 共享组件包

包名使用中性 scope，scope 可按组织实际配置，不写死品牌。

建议命名：

- `@agentkit/protocol`
- `@agentkit/runtime-client`
- `@agentkit/react`
- `@agentkit/action-skill`

如果 `agentkit` scope 不可用，可替换为内部中性 scope，例如 `@internal-agent/*`。

### `@agentkit/protocol`

非 React 包，定义跨项目契约：

- `AgentStreamEvent`
- `AgentUiIntent`
- `ActionDefinition`
- `ActionPreview`
- `ActionExecuteRequest`
- `ActionExecuteResponse`
- `AgentArtifactRef`
- 标准错误结构

### `@agentkit/runtime-client`

封装 runtime 通信：

- 创建/恢复会话
- 发送消息
- 解析 SSE stream
- 取消 run
- 下载 artifact
- 提交 action confirmation
- 处理 token exchange 后的 delegation 凭证

### `@agentkit/react`

提供可嵌入 UI：

- `AgentPanelHost`
- `AgentChatPanel`
- `AgentMessageRenderer`
- `ActionPreviewCard`
- `UiIntentList`
- `ArtifactList`
- `TracePanel`

UI 必须通过 theme adapter/token 适配宿主项目，不携带固定视觉风格。

### goomc Web Adapter

goomc Web 侧适配代码属于 goomc 仓库，不属于 agent-studio 通用 runtime：

- 采集当前 route、页面类型、语言、当前角色。
- 采集选中设备、告警、任务、筛选条件。
- 把页面状态转换为 `AgentContext`。
- 消费 `uiIntent`，执行跳转、刷新、打开详情和列表高亮。

### `@agentkit/action-skill`

agent runtime 侧通用 action skill：

- 使用配置中的 Action API。
- sidecar/运维场景可 fallback 到 `omcctl --output json`。
- 不访问业务系统数据库。
- 不使用 MCP。
- 常驻工具只暴露 `actions.search`、`actions.describe`、`actions.preview`、`actions.execute`。
- 业务系统名称、显示名、base URL、delegation 路径和 action 路径都来自 integration instance 配置。

## Agent Studio 通用集成约束

agent-studio 侧不得新增 goomc 专用枚举、路由、组件、skill 名或工具前缀。

正确抽象是通用 Action Connector：

```ts
type ActionConnectorConfig = {
  connectorId: string;
  displayName: string;
  baseUrl: string;
  delegationExchangePath: string;
  actionSearchPath: string;
  actionDescribePath: string;
  actionPreviewPathTemplate: string;
  actionExecutePathTemplate: string;
  streamEventDialect?: 'agentkit-v1';
};
```

agent-studio 只识别 `action_connector` 这类通用集成类型。goomc、crest 或其他业务系统只是配置实例：

- `connectorId`: 机器可读实例 ID，例如 `omc-prod`，由管理员配置。
- `displayName`: UI 展示名，例如 `OMC Production`，由管理员配置。
- action 路径和 delegation 路径来自配置，不写死。
- runtime 注入的是 `@agentkit/action-skill`，不是 `goomc-skill`。
- 工具名保持 `actions.search/describe/preview/execute`，由请求参数中的 `connectorId` 定位业务系统。

goomc 侧可以在自己的仓库里保留 OMC 领域文案、页面上下文 adapter 和 Action 定义；这些内容不能进入 agent-studio 的通用代码路径。

## goomc Action API

新增后端模块建议放在 `omcgo/internal/agentaction/`，遵循现有 `handler -> service -> repository/model` 风格。

### Delegation

```http
POST /api/v1/agent/delegation
Authorization: Bearer <web-jwt>
```

用途：

- 前端复用 Web 登录态发起授权。
- goomc 验证当前用户、当前角色、session 状态和权限状态。
- goomc 签发短期 `agentDelegationToken`。
- 不把 Web JWT 原 token 交给 Agent Runtime。

token 约束：

- 短 TTL。
- 可撤销。
- 只允许访问 `/api/v1/agent-actions/*`。
- 绑定 user id、当前 role、session/version、过期时间。
- 可选绑定页面上下文或 allowlisted action scope。

### Action Catalog

```http
GET /api/v1/agent-actions/search?q=...
GET /api/v1/agent-actions/:id
```

ActionDefinition 至少包含：

```ts
type ActionDefinition = {
  id: string;
  title: string;
  description: string;
  module: 'device' | 'alarm' | 'pm' | 'mml' | 'trace' | 'ops' | 'software' | 'system';
  inputSchema: object;
  risk: 'read' | 'write' | 'destructive';
  requiresConfirmation: boolean;
  previewOnly?: boolean;
  requiredPermission?: {
    resource: string;
    action: string;
  };
  uiIntent?: AgentUiIntentTemplate;
};
```

### Preview

```http
POST /api/v1/agent-actions/:id/preview
Authorization: Bearer <agent-delegation-token>
Idempotency-Key: <uuid>
```

返回：

```ts
type ActionPreview = {
  ok: boolean;
  actionId: string;
  title: string;
  summary: string;
  risk: 'read' | 'write' | 'destructive';
  affectedResources: Array<{
    type: string;
    id: string;
    label?: string;
  }>;
  warnings: string[];
  confirmationToken?: string;
  expiresAt?: string;
  previewOnly?: boolean;
  nextAction?: string;
};
```

### Execute

```http
POST /api/v1/agent-actions/:id/execute
Authorization: Bearer <agent-delegation-token>
Idempotency-Key: <uuid>
```

请求：

```ts
type ActionExecuteRequest = {
  input: unknown;
  confirmationToken?: string;
};
```

响应：

```ts
type ActionExecuteResponse = {
  ok: boolean;
  actionId: string;
  title: string;
  result?: unknown;
  auditId?: string;
  idempotencyKey?: string;
  reused?: boolean;
  uiIntent?: AgentUiIntent;
  error?: {
    code: string;
    message: string;
    nextAction?: string;
  };
};
```

## 首期动作范围

首期定位为“安全运维助手”，不是全自动运维员。

### 可直接执行的读动作

- 设备查询
- 设备详情摘要
- 告警分析
- PM/KPI 解释
- 任务状态查询
- MML 命令/参数解释
- 日志和 trace 查询

### 用户确认后可执行的低风险写动作

- 告警确认
- 告警清除
- 参数同步
- 诊断任务创建
- trace 抓包任务创建
- 低风险运维任务创建

### 首期只允许预览的高风险动作

- 批量重启
- 固件升级
- MML 执行
- 参数批量下发
- 删除类操作

这些动作返回 `previewOnly=true`，并引导用户到原业务页面完成。

## 前端入口与体验

### 入口形态

首期主入口是全局右侧抽屉/浮动 Agent Panel：

- 三套皮肤都接入同一个 `AgentPanelHost`。
- 用户在任意业务页面打开 Agent。
- 当前页面上下文自动注入，不要求用户重复描述“这个设备”“这个告警”。
- 独立 `/agent` 页面最多作为历史/管理入口，不作为首期主入口。

### 页面上下文

`AgentContext` 包含：

- 当前 path/search/hash。
- 页面模块：device/alarm/pm/mml/trace/ops/software/system。
- 当前选中资源：device id/sn、alarm id、task id 等。
- 当前筛选条件和表格选择项。
- 当前语言、时区和角色。

### UI Intent

`AgentUiIntent` 支持：

- `navigate`：跳转到页面。
- `show_records`：打开列表并高亮相关记录。
- `refresh_record`：刷新当前详情或列表。
- `toast`：提示操作结果。

示例：

```ts
type AgentUiIntent = {
  kind: 'navigate' | 'show_records' | 'refresh_record' | 'toast';
  route?: string;
  entity?: string;
  ids?: string[];
  query?: Record<string, string>;
  message?: string;
  sourceActionId?: string;
};
```

## 前端视觉设计硬约束

新增前端实现必须遵循以下规则：

1. Agent UI 必须和使用它的系统前端风格一致。goomc v1/v2/v3 三套皮肤分别按各自视觉语言适配，不允许引入独立、不协调的 Agent 风格。
2. 实现前必须基于当前前端使用 imagegen 生成前端示意图。如果涉及多页面、多组件、多层次交互，必须生成多个示意效果图，至少覆盖：
   - 默认收起/展开入口。
   - 聊天主面板。
   - Action Preview 确认卡片。
   - `uiIntent` 相关页面入口。
   - 错误/权限不足状态。
   - 移动端或窄屏状态。
3. 新增前端必须按已确认效果图还原实现，不能擅自实现。如果实现过程中发现效果图与实际页面约束冲突，必须先更新示意图并再次确认，再改代码。
4. 示意图必须以浏览器中的实际 goomc 页面为背景或参考，不能只画孤立组件。
5. 三皮肤实现必须共享交互结构，只通过 theme adapter/token、CSS 变量或皮肤级包装适配视觉。

## 安全设计

### 身份

- Web 用户身份通过 token exchange 复用。
- Agent Runtime 不持有 Web JWT。
- delegation token 只能调用 agent actions。
- 用户退出、切角色、强制下线或 token 过期后，Agent 调用必须失败并引导重新授权。

### 权限

- 每个 Action 声明 `requiredPermission`、`risk`、`resourceScope`、`requiresConfirmation`。
- goomc 后端执行 Action 时还原用户身份，继续走现有 Casbin 和业务权限。
- 设备级动作必须再次校验用户是否可见该设备组。
- system/API key fallback 只能进入受限 action 集，默认 read/diagnose。

### 确认与幂等

- 写操作必须 preview。
- preview 生成 `confirmationToken`，绑定 action id、输入摘要、用户、过期时间。
- execute 必须带 `confirmationToken` 和 `Idempotency-Key`。
- token 不匹配、输入被改、过期、用户权限变化，都拒绝执行并提示重新 preview。

### 审计

goomc 审计记录：

- 用户
- action id
- 风险等级
- 输入摘要
- 影响资源
- 执行结果
- runtime run id/thread id
- idempotency key

Agent Runtime 审计记录：

- 用户消息
- 模型/provider
- skill/tool 调用
- token 用量
- runtime 错误

两侧通过 `auditId`、`runId`、`threadId` 互相引用，不共享数据库。

## CLI Fallback

CLI fallback 只用于：

- sidecar 私有化部署。
- 容器内运维。
- 无浏览器或无 Web session 的受控环境。

约束：

- 默认调用 `omcctl --output json`。
- 只允许受限 action 集。
- 高风险动作仍必须 preview-only 或要求人工到业务页面完成。
- stderr 必须转换为结构化错误，不直接展示给用户。

## 错误处理

- 鉴权失败：提示重新授权，保留当前输入。
- 权限不足：说明缺少哪个动作权限，并给只读替代建议。
- preview 失败：展示校验失败原因和下一步。
- execute 失败：说明是否已产生影响，并提供任务或审计入口。
- runtime 失败：保留页面上下文和用户输入，可重试。
- CLI fallback 失败：展示结构化错误和下一步，不暴露原始命令噪声。

## 测试验收

### AgentKit

- `@agentkit/protocol`：schema 兼容性、事件解析、错误结构。
- `@agentkit/runtime-client`：SSE 流、取消、重试、artifact 下载、确认提交。
- `@agentkit/react`：消息渲染、确认卡片、uiIntent、无障碍、响应式。

### goomc 后端

- delegation token 签发、过期、撤销。
- action search/describe/preview/execute。
- Casbin 权限。
- 设备组隔离。
- confirmation token 校验。
- idempotency。
- 审计串联。

### goomc 前端

- 三皮肤 Agent Panel 挂载。
- 当前页面上下文注入。
- `uiIntent` 跳转、刷新、高亮。
- Action Preview 确认卡片。
- 权限不足、过期、runtime 失败状态。

### 集成验收场景

- 在设备页问“分析这个设备最近告警”。
- 在告警页确认单条告警。
- 在 trace 场景创建抓包任务 preview。
- 在 MML 场景对高风险执行只返回 preview-only。
- 在 sidecar 场景通过 `omcctl --output json` 完成只读查询。

## 里程碑

### M1 协议和 UI 骨架

- AgentKit 三层包。
- goomc 三皮肤全局 Panel。
- runtime-client SSE 接通。
- 基于 imagegen 的三皮肤示意图确认。

### M2 goomc Action API

- delegation token。
- search/describe/preview/execute。
- 首批 read actions。

### M3 安全写操作

- confirmation token。
- idempotency。
- 低风险 confirmed write。
- 审计串联。

### M4 sidecar/CLI fallback

- `@agentkit/action-skill` 支持 API-first。
- `omcctl --output json` fallback。
- 私有化部署适配。

### M5 扩展到其他 Web 项目

- 用另一个项目接入同一 AgentKit。
- 验证 AgentKit 不包含 goomc 假设。

## 实施顺序建议

1. 先抽协议和 React UI 骨架，不接真实动作。
2. 为 goomc 生成并确认 v1/v2/v3 Agent Panel 示意图。
3. 实现 goomc delegation 和 read-only actions。
4. 接入低风险 write 的 preview/confirm。
5. 最后补 CLI fallback 和 sidecar 部署。

## 实施切片

这个设计覆盖的是完整方向，不应一次性塞进一个实现计划。

首个 implementation plan 只覆盖 M1 到 M2 的只读闭环：

- AgentKit protocol/runtime-client/react-ui 骨架。
- goomc 三皮肤 Agent Panel 入口和 imagegen 示意图确认。
- agent-studio 通用 `action_connector` 集成，不出现 goomc 专用代码路径。
- goomc delegation token。
- 首批 read-only actions。
- 从 goomc 页面发起一次查询并通过 SSE 返回答案。

后续单独拆计划：

- M3：低风险写操作、confirmation token、idempotency 和审计串联。
- M4：sidecar/CLI fallback。
- M5：其他 Web 项目接入验证。

## 开放问题

当前设计已确认以下决策：

- 统一 runtime，预留 sidecar 部署。
- API-first，CLI fallback。
- 首期安全运维助手，不做全自动运维员。
- AgentKit 拆为 protocol/runtime-client/react-ui。
- 共享包使用中性命名，不绑定品牌。
- agent-studio 侧使用通用 `action_connector`，不写死 goomc 名称。
- goomc Web 主入口为全局 Agent Panel。
- Web 登录态通过 token exchange 换 delegation token，不直接复用 Web JWT。
- 前端实现必须先基于 imagegen 效果图确认，再按图还原。

未在本设计中展开的细节留到 implementation plan：

- 每个首批 Action 的具体 schema。
- delegation token 的签名算法和存储策略。
- AgentKit 包的仓库位置和发布方式。
- 三皮肤效果图的具体提示词和验收截图。
