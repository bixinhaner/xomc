# Goomc Agent Readonly Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the first production slice of the AI agent integration: a host-styled Agent Panel in goomc, a short-lived delegation-token bridge, read-only agent action APIs, and a generic `action_connector` integration in agent-studio without hardcoded goomc naming.

**Architecture:** goomc remains the business system and exposes a narrow agent-action contract. agent-studio remains the agent runtime and owns orchestration. Shared client/UI code in goomc is packaged as neutral AgentKit modules so another web project can adopt the same protocol, runtime client, and React panel with its own host styling. agent-studio integrates with business systems through a generic `action_connector` type whose display name, base URL, action paths, and policy come from integration-instance config.

**Tech Stack:** Go 1.25 + Chi + pgx/Squirrel in `omcgo`; React + TypeScript + Vite + Zustand + Axios in `omcmb`; Node/Express/TypeScript/Prisma + React/Ant Design in `agent-studio`; SSE for streaming; HMAC JWT for web-to-agent delegation tokens.

## Global Constraints

- Do not hardcode `goomc`, `omc`, or project-specific names in agent-studio route names, integration type behavior, UI labels, runtime service names, skill names, or tool names.
- agent-studio must use generic names: `action_connector`, `actions.search`, `actions.describe`, `actions.preview`, `actions.execute`.
- goomc-specific action IDs such as `device.search` and `alarm.active_summary` live only in goomc action metadata and goomc backend code.
- Agent Panel frontend must match each host skin's visual style. Generate imagegen mockups from the current v1/v2/v3 UI and get user approval before writing panel UI code.
- Frontend behavior changes must cover all three goomc skins: `webcode`, `webcode-v2`, and `webcode-v3`.
- Reuse web login state only through a short-lived delegation token. Do not send the normal web JWT to agent-studio.
- The first production slice is read-only. Write actions, CLI fallback, and sidecar packaging are reserved for later plans.
- Keep Docker and local uncommitted deployment changes untouched unless the task explicitly requires them.

---

## Task 1: Add Neutral AgentKit Protocol Types In Goomc Frontend Core

**Why:** The protocol must be reusable by goomc and other web products. Keeping it in frontend-core makes all three skins share one contract and avoids UI-specific drift.

**User impact:** Users get consistent streamed answers, tool previews, and errors across skins. Other projects can adopt the same component without rewriting message contracts.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/protocol.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/index.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/protocol.test.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/index.ts`

**Interfaces:**

```ts
export type AgentStreamEvent =
  | { type: 'start'; runId: string; conversationId: string }
  | { type: 'delta'; text: string }
  | { type: 'tool_call'; callId: string; toolName: string; title: string; input: unknown }
  | { type: 'action_preview'; callId: string; title: string; summary: string; risk: 'read' | 'low' | 'high'; preview: unknown }
  | { type: 'tool_result'; callId: string; status: 'ok' | 'error'; output?: unknown; error?: AgentError }
  | { type: 'done'; usage?: AgentUsage }
  | { type: 'error'; error: AgentError };

export interface AgentActionDescriptor {
  id: string;
  title: string;
  description: string;
  inputSchema: Record<string, unknown>;
  risk: 'read' | 'low' | 'high';
  scopes: string[];
}

export interface AgentRuntimeRequest {
  message: string;
  conversationId?: string;
  locale: string;
  timezone: string;
  context: AgentPageContext;
}
```

**Steps:**

- [ ] Create `src/agentkit/protocol.ts` with exported event, action, context, runtime request, runtime response, usage, and error types.
- [ ] Add a type guard `isAgentStreamEvent(value: unknown): value is AgentStreamEvent` that validates `type` and required fields only.
- [ ] Add `src/agentkit/index.ts` to re-export the public protocol.
- [ ] Export AgentKit from `frontend-core/src/index.ts`.
- [ ] Add unit tests for all stream event variants and invalid payload rejection.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/goomc/omcmb
npm run typecheck
```

**Expected result:** TypeScript compiles and tests prove malformed stream events are rejected before they reach UI state.

**Commit:** `feat(agentkit): 添加通用 agent 协议类型`

---

## Task 2: Implement AgentKit Runtime Client With SSE Parsing

**Why:** The UI should not know transport details. A small runtime client gives goomc and future projects the same streaming API and keeps retry, abort, and error handling centralized.

**User impact:** The panel can stop a run immediately, surface clear retryable errors, and avoid duplicate messages after network interruption.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/runtimeClient.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/runtimeClient.test.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/index.ts`

**Interfaces:**

```ts
export interface AgentRuntimeClientOptions {
  endpoint: string;
  getDelegationToken: () => Promise<string>;
  fetchImpl?: typeof fetch;
}

export interface AgentRuntimeClient {
  stream(request: AgentRuntimeRequest, handlers: AgentStreamHandlers, signal?: AbortSignal): Promise<void>;
}
```

**Steps:**

- [ ] Implement `createAgentRuntimeClient(options)` using `fetch` with `Authorization: Bearer <delegationToken>`.
- [ ] Parse `text/event-stream` frames and validate every JSON payload through `isAgentStreamEvent`.
- [ ] Convert HTTP 401/403/5xx and malformed event payloads into `AgentError` objects.
- [ ] Support `AbortSignal` so closing the panel or clicking stop cancels the request.
- [ ] Unit test normal streaming, malformed events, server errors, and cancellation.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/goomc/omcmb
npm run typecheck
```

**Expected result:** A transport-agnostic client streams typed events and never leaks malformed payloads into the UI.

**Commit:** `feat(agentkit): 添加 agent runtime sse 客户端`

---

## Task 3: Run The Frontend Visual Design Gate Before UI Code

**Why:** The user explicitly requires implementation to follow approved imagegen mockups. The design gate prevents the agent panel from drifting away from each skin's current visual system.

**User impact:** The agent feels like part of the existing system instead of a pasted-in foreign widget.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/2026-07-06-visual-brief.md`
- `/Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/assets/v1-panel.png`
- `/Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/assets/v1-preview.png`
- `/Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/assets/v2-panel.png`
- `/Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/assets/v2-preview.png`
- `/Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/assets/v3-panel.png`
- `/Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/assets/v3-preview.png`

**Steps:**

- [ ] Capture current browser screenshots of v1, v2, and v3 authenticated shells using the running local app.
- [ ] Use imagegen to generate six mockups:
  - [ ] v1 collapsed trigger + open panel on device list.
  - [ ] v1 action preview/result state.
  - [ ] v2 collapsed trigger + open panel on device list.
  - [ ] v2 action preview/result state.
  - [ ] v3 collapsed trigger + open panel on device list.
  - [ ] v3 action preview/result state.
- [ ] Save images under `docs/design/agent-panel/assets/`.
- [ ] Write a visual brief describing panel placement, spacing, typography, color tokens, empty state, streaming state, tool preview state, error state, and mobile behavior for all three skins.
- [ ] Present the six images and brief to the user.
- [ ] Do not start Task 4 UI implementation until the user approves the mockups.

**Imagegen prompt template:**

```text
Create a production UI mockup for an Operations Management web app agent panel.
Use the attached/current screenshot visual style exactly: spacing, typography, nav density, table density, colors, borders, and control shape.
Show a right-side assistant drawer opened from the app shell.
The drawer contains a compact message timeline, streaming response text, read-only action preview, cited device/alarm data chips, and a bottom composer.
No marketing text, no decorative gradient blobs, no unrelated branding, no oversized hero layout.
```

**Validation commands:**

```bash
ls -1 /Users/like/Desktop/baicells/Trae/goomc/docs/design/agent-panel/assets
```

**Expected result:** The user has approved visual references before implementation. UI code must match those references.

**Commit:** `docs(agent): 添加 agent panel 视觉方案`

---

## Task 4: Implement Skin-Agnostic Agent React State And Panel Primitives

**Why:** The reusable Agent Panel needs componentized internals, but each skin must still control its visual shell. State and behavior belong in frontend-core; skin-specific wrappers belong in each webcode package.

**User impact:** The same assistant behavior works in every skin while preserving each UI style.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/react/useAgentSession.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/react/AgentMessageList.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/react/AgentComposer.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/react/AgentActionPreview.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/react/index.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/agentkit/react/useAgentSession.test.tsx`

**Interfaces:**

```ts
export interface UseAgentSessionOptions {
  client: AgentRuntimeClient;
  getPageContext: () => AgentPageContext;
  locale: string;
  timezone: string;
}

export interface AgentPanelPrimitiveProps {
  messages: AgentMessage[];
  pendingPreview?: AgentActionPreviewState;
  isStreaming: boolean;
  error?: AgentError;
  onSubmit: (message: string) => void;
  onStop: () => void;
  onRetry: () => void;
}
```

**Steps:**

- [ ] Implement `useAgentSession` to maintain messages, stream state, action previews, cancellation, retry, and current conversation ID.
- [ ] Keep primitives unstyled except semantic class names and ARIA attributes.
- [ ] Do not import Ant Design from frontend-core primitives.
- [ ] Add keyboard behavior: Enter sends, Shift+Enter inserts newline, Escape stops the current stream.
- [ ] Add tests for event ordering, stop behavior, retry behavior, and error display state.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/goomc/omcmb
npm run typecheck
```

**Expected result:** Agent behavior is shared and testable independent of the three skin shells.

**Commit:** `feat(agentkit): 添加 agent panel 共享交互状态`

---

## Task 5: Add Goomc Delegation Token And Read-Only Action API

**Why:** agent-studio needs temporary authority to call business actions on behalf of the logged-in user, but it must not receive the user's normal web JWT. A dedicated delegation token narrows scope, TTL, and auditability.

**User impact:** The assistant can answer with real device and alarm data while preserving existing permission boundaries. If the agent token leaks, it expires quickly and only works on agent-action endpoints.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/admin/jwt_service.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/admin/jwt_service_test.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentaction/model.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentaction/delegation.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentaction/registry.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentaction/service.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentaction/handler.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/internal/agentaction/handler_test.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/cmd/app/provider/container.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/cmd/app/provider/modules.go`
- `/Users/like/Desktop/baicells/Trae/goomc/omcgo/cmd/app/provider/router.go`

**Interfaces:**

```go
type ActionDescriptor struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	InputSchema map[string]any         `json:"inputSchema"`
	Risk        string                 `json:"risk"`
	Scopes      []string               `json:"scopes"`
}

type ExecuteRequest struct {
	ActionID string         `json:"actionId"`
	Input    map[string]any `json:"input"`
	DryRun   bool           `json:"dryRun"`
}

type ExecuteResponse struct {
	ActionID string `json:"actionId"`
	Status   string `json:"status"`
	Result   any    `json:"result"`
}
```

**Routes:**

```text
POST /api/v1/agent/delegation
GET  /api/v1/agent-actions/actions
POST /api/v1/agent-actions/actions/search
POST /api/v1/agent-actions/actions/describe
POST /api/v1/agent-actions/actions/preview
POST /api/v1/agent-actions/actions/execute
```

**Steps:**

- [ ] Add `GenerateAgentDelegationToken` and `ValidateAgentDelegationToken` to `admin.JWTService`.
- [ ] Use JWT subject `agent_delegation`, TTL 5 minutes, user ID, username, roles, visible permission context, and scope `agent-actions`.
- [ ] Make `POST /api/v1/agent/delegation` require normal web JWT context. Reject API-key-only callers because API keys are server credentials, not user sessions.
- [ ] Add agent-action middleware that validates delegation tokens and restores existing context keys used by permission services.
- [ ] Register read-only actions:
  - [ ] `device.search`: query visible devices by serial number, name, IP, status, and group filters.
  - [ ] `device.summary`: return one visible device summary plus key parameter snapshot.
  - [ ] `alarm.active_summary`: return visible active alarm counts and top active alarm rows.
  - [ ] `system.health`: return service health, version, and current time.
- [ ] Ensure every action applies the same permission resolution as existing device/alarm pages.
- [ ] Audit action execution with user ID, action ID, input hash, result status, and request ID.
- [ ] Return structured errors with stable codes: `UNAUTHORIZED`, `FORBIDDEN`, `ACTION_NOT_FOUND`, `VALIDATION_FAILED`, `UPSTREAM_ERROR`.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/goomc/omcgo
go test ./internal/admin ./internal/agentaction ./cmd/app/provider
go build ./...
```

**Expected result:** The backend can issue a scoped delegation token and execute read-only actions with existing user permissions.

**Commit:** `feat(agent): 添加只读 agent action api`

---

## Task 6: Add Goomc Frontend API Bindings For Delegation And Runtime

**Why:** The web app should hide token exchange and runtime details from the panel components. Existing user token refresh remains in the normal HTTP client; the agent-specific token is requested only when a run starts.

**User impact:** The assistant works from the current login session without asking users to paste keys or re-authenticate.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/services/api/agent.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/hooks/useAgentRuntimeClient.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/i18n/locales/en-US/agent.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/i18n/locales/zh-CN/agent.ts`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/services/api/index.ts`

**Interfaces:**

```ts
export interface DelegationTokenResponse {
  token: string;
  expiresAt: string;
}

export interface AgentRuntimeConfig {
  endpoint: string;
  connectorId: string;
}
```

**Steps:**

- [ ] Implement `requestAgentDelegationToken()` using the existing authenticated Axios client.
- [ ] Implement `useAgentRuntimeClient()` that creates `AgentRuntimeClient` with the configured agent-studio runtime endpoint.
- [ ] Read runtime endpoint and connector ID from environment/config, not hardcoded constants.
- [ ] Add Chinese and English i18n keys for assistant labels, empty state, stop, retry, tool preview, and error actions.
- [ ] Unit test that the hook requests a new delegation token per run and never exposes the normal web access token to runtime client headers.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/goomc/omcmb
npm run typecheck
```

**Expected result:** The panel can start a secure runtime stream without knowing web-token internals.

**Commit:** `feat(agent): 添加前端 agent 授权绑定`

---

## Task 7: Mount Agent Panel In All Three Goomc Skins After Mockup Approval

**Why:** Shell integration is where visual consistency and route context are enforced. Each skin needs its own wrapper, while the shared AgentKit state remains central.

**User impact:** Users can open the assistant anywhere without navigating away, and the assistant can understand the current page, selected device, filters, and locale.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/components/agent/AgentPanelHost.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/components/agent/AgentPanelHost.css`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode/src/components/Layout/index.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v2/src/components/agent/AgentPanelHost.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v2/src/components/agent/AgentPanelHost.css`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v2/src/components/layout/AppShell.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v3/src/components/agent/AgentPanelHost.tsx`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v3/src/components/agent/AgentPanelHost.css`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/webcode-v3/src/components/shell/BridgeShell.tsx`

**Steps:**

- [ ] Implement a skin-specific floating trigger and right drawer for v1 matching approved v1 mockups.
- [ ] Implement the same behavior for v2 matching approved v2 mockups.
- [ ] Implement the same behavior for v3 matching approved v3 mockups.
- [ ] Build `getPageContext()` per skin using route pathname, query params, visible page title, selected rows when available, and current locale.
- [ ] Ensure drawer has stable dimensions, no text overflow, no overlap with existing shell controls, and mobile behavior matching the approved visual brief.
- [ ] Add accessible labels and focus management: trigger opens drawer, focus moves to composer, close returns focus to trigger.
- [ ] Keep user-visible copy in i18n.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/goomc/omcmb
npm run skin-parity
npm run typecheck
```

**Expected result:** All three skins expose an approved Agent Panel with equivalent routes and behavior.

**Commit:** `feat(agent): 在三套皮肤接入 agent panel`

---

## Task 8: Add Generic action_connector Support In agent-studio API

**Why:** agent-studio should support many business systems through one generic integration type. Hardcoding one project name repeats the `crest_crm` coupling problem and makes reuse expensive.

**User impact:** Administrators can connect goomc now and later connect other systems with the same agent runtime path.

**Files:**

- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/center/types.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/center/action-connector-adapter.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/center/service.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/center/action-connector-adapter.test.ts`

**Interfaces:**

```ts
export const actionConnectorConfigSchema = z.object({
  displayName: z.string().min(1),
  baseUrl: z.string().url(),
  healthPath: z.string().default('/healthz'),
  actionListPath: z.string().default('/api/v1/agent-actions/actions'),
  actionSearchPath: z.string().default('/api/v1/agent-actions/actions/search'),
  actionDescribePath: z.string().default('/api/v1/agent-actions/actions/describe'),
  actionPreviewPath: z.string().default('/api/v1/agent-actions/actions/preview'),
  actionExecutePath: z.string().default('/api/v1/agent-actions/actions/execute'),
  delegationHeader: z.string().default('Authorization'),
  policy: z.object({
    allowReadActions: z.boolean().default(true),
    allowLowRiskActions: z.boolean().default(false),
    allowHighRiskActions: z.boolean().default(false),
  }),
});
```

**Steps:**

- [ ] Add `action_connector` to integration type validation and TypeScript unions.
- [ ] Implement a validation adapter that validates generic config shape and optional health reachability.
- [ ] Register the adapter in integration-center service without singleton behavior.
- [ ] Ensure names and log messages say `action_connector` or `action connector`, not a business-system name.
- [ ] Add tests for valid config, invalid URL, missing path, and non-singleton multiple instances.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/agent-studio/agent-api
npm test -- --run action-connector
```

**Expected result:** agent-studio can store and validate generic action connector instances.

**Commit:** `feat(integrations): 添加通用 action connector`

---

## Task 9: Add Generic action_connector UI In agent-studio

**Why:** Integration setup must remain business-neutral. Admins configure a named connector instance, not a goomc-specific product screen.

**User impact:** The same setup screen can connect multiple business systems by URL and action paths.

**Files:**

- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-ui/src/features/integration-center/types.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-ui/src/features/integration-center/ActionConnectorIntegrationView.tsx`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-ui/src/features/integration-center/IntegrationCenterShell.tsx`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-ui/src/features/integration-center/ActionConnectorIntegrationView.test.tsx`

**Steps:**

- [ ] Add `action_connector` to UI integration type union and tab metadata.
- [ ] Build a generic form with display name, base URL, path fields, and risk policy switches.
- [ ] Do not include goomc-specific placeholder text. Use examples like `https://ops.example.com`.
- [ ] Show validation status and next action: save, test connection, activate.
- [ ] Add tests that assert no rendered text contains `goomc` or `OMC`.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/agent-studio/agent-ui
npm run build
```

**Expected result:** agent-studio administrators can configure a generic action connector without project-specific labels.

**Commit:** `feat(integrations): 添加 action connector 配置界面`

---

## Task 10: Add agent-studio Generic Action Runtime Bridge

**Why:** The agent runtime needs a stable way to discover and execute business actions through the selected connector while remaining independent of any one business system.

**User impact:** A goomc user can ask a question in the web panel, agent-studio can call only permitted read actions, and the response streams back with action visibility.

**Files:**

- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/client.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/runtime.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/routes.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/index.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/src/integrations/action-connector/runtime.test.ts`

**Routes:**

```text
POST /api/action-connectors/:connectorId/chat/stream
```

**Request:**

```ts
interface ActionConnectorChatRequest {
  message: string;
  conversationId?: string;
  locale: string;
  timezone: string;
  context: Record<string, unknown>;
}
```

**Steps:**

- [ ] Resolve `connectorId` to an active `action_connector` integration instance.
- [ ] Read the delegation token from the incoming `Authorization` header and forward it only to the configured connector action endpoints.
- [ ] Implement `ActionConnectorClient` with generic methods: `search`, `describe`, `preview`, `execute`.
- [ ] Add runtime policy enforcement: first slice allows `risk === 'read'` only.
- [ ] Stream events using the shared event names: `start`, `delta`, `tool_call`, `action_preview`, `tool_result`, `done`, `error`.
- [ ] Integrate with the existing agent execution path by injecting generic connector tool instructions and tool callbacks. The injected tool names must be `actions.search`, `actions.describe`, `actions.preview`, `actions.execute`.
- [ ] Add tests using a fake connector server to verify token forwarding, read-only enforcement, event order, and no business-specific names in runtime logs.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/agent-studio/agent-api
npm test -- --run action-connector
```

**Expected result:** agent-studio streams a generic action-enabled conversation and calls only read actions through the configured connector.

**Commit:** `feat(runtime): 添加通用 action connector agent bridge`

---

## Task 11: Add End-To-End Local Configuration And Smoke Test

**Why:** A production slice is not done until the two repositories work together locally with clear configuration and repeatable checks.

**User impact:** The user can run the app locally and verify the assistant against real local data without editing source code.

**Files:**

- `/Users/like/Desktop/baicells/Trae/goomc/docs/agent-local-runbook.md`
- `/Users/like/Desktop/baicells/Trae/goomc/omcmb/frontend-core/src/config/agent.ts`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-api/.env.example`
- `/Users/like/Desktop/baicells/Trae/agent-studio/agent-ui/.env.example`

**Steps:**

- [ ] Add goomc frontend config keys for agent-studio runtime URL and connector ID.
- [ ] Add agent-studio example env keys for action connector runtime, without goomc-specific variable names.
- [ ] Write a runbook that explains:
  - [ ] Start goomc backend/frontend.
  - [ ] Start agent-studio API/UI.
  - [ ] Create an `action_connector` instance pointing at local goomc.
  - [ ] Set the connector ID in goomc frontend config.
  - [ ] Open a device page and ask a read-only question.
- [ ] Include expected SSE event sequence and common failure fixes for 401, 403, CORS, and connector health failure.

**Validation commands:**

```bash
cd /Users/like/Desktop/baicells/Trae/goomc/omcgo
go test ./...
cd /Users/like/Desktop/baicells/Trae/goomc/omcmb
npm run skin-parity
npm run typecheck
cd /Users/like/Desktop/baicells/Trae/agent-studio/agent-api
npm test -- --run action-connector
cd /Users/like/Desktop/baicells/Trae/agent-studio/agent-ui
npm run build
```

**Expected result:** Local goomc can open the Agent Panel, receive a streamed answer from agent-studio, and show read-only tool activity backed by goomc action APIs.

**Commit:** `docs(agent): 添加本地 agent 联调说明`

---

## Implementation Order

1. Task 1
2. Task 2
3. Task 5
4. Task 8
5. Task 10
6. Task 6
7. Task 3
8. Task 4
9. Task 7
10. Task 9
11. Task 11

This order makes the backend and runtime contract real before panel implementation, while preserving the required visual approval gate before UI code.

## Definition Of Done

- goomc backend issues scoped delegation tokens and exposes read-only action APIs.
- agent-studio supports generic `action_connector` instances and uses generic action tool names.
- goomc frontend uses AgentKit protocol/client/state from `frontend-core`.
- All three goomc skins include a mockup-approved Agent Panel with equivalent behavior.
- Normal web JWT is never sent to agent-studio.
- agent-studio codebase contains no new hardcoded `goomc` or `OMC` labels for the connector/runtime implementation.
- Required validation commands pass, or failures are documented with exact command output and next fix.

