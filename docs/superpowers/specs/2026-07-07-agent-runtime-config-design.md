# Agent Runtime Configuration Design

## Goal

Let an OMC administrator configure the AI Agent integration from `System Management -> System Configuration -> Agent`, without rebuilding the frontend, editing server environment variables, or manually creating an Action Connector in Agent Studio.

## Product Decision

Use goomc as the source of truth for the business system integration. Agent Studio remains generic and exposes a token-protected provision API that can create or update an `action_connector` instance for any external business system.

Why: the operator is already administering the business system in goomc. Sending them to Agent Studio to finish a backend connector is a split workflow and creates failure modes that are hard to diagnose.

User impact: one page in goomc controls enablement, remote endpoint, connector provisioning, and connection validation.

## Architecture

### goomc

- Store Agent integration settings in `sys_configs` under category `agent`.
- Add a focused backend service and handler for Agent settings instead of exposing raw KV editing for the secret and provision flow.
- Add an authenticated runtime config endpoint consumed by the Agent panel after login.
- Keep existing `VITE_AGENT_*` support only as a development fallback.
- Keep the current short-lived delegation token flow. The browser still obtains a web-token-derived delegation token from goomc and sends it to Agent Studio runtime.

### agent-studio

- Add a generic external provision endpoint for `action_connector`.
- Protect the provision endpoint with the existing service token middleware.
- Upsert connector instances by `slug`; do not hardcode goomc or any brand name.
- Reuse the existing `action_connector` config schema, runtime endpoint, validation adapter, and integration repository.

## Data Flow

1. Admin opens goomc `System Configuration -> Agent`.
2. UI reads `GET /api/v1/admin/agent-config`.
3. Admin enters Agent Studio API URL, service token, public goomc API URL, and enables Agent.
4. UI calls `POST /api/v1/admin/agent-config/sync`.
5. goomc saves non-secret fields and the masked/write-only service token server-side.
6. goomc calls Agent Studio `POST /api/integrations/action-connectors/provision` with service token.
7. Agent Studio creates or updates a generic `action_connector` and returns `connectorId`.
8. goomc stores `connector_id`, derived runtime stream URL, status, last validation time, and error message.
9. Agent panel reads `GET /api/v1/agent/config` at runtime and streams to Agent Studio without build-time env config.

## Configuration Keys

All keys use category `agent`.

- `enabled` (`bool`): enables the Agent panel runtime.
- `agent_studio_base_url` (`string`): Agent Studio API origin, for example `https://agent.example.com`.
- `agent_studio_service_token` (`string`): write-only secret used by goomc backend to provision the connector.
- `omc_public_base_url` (`string`): public goomc API origin reachable by Agent Studio.
- `connector_slug` (`string`): stable slug used for idempotent provisioning.
- `connector_id` (`string`): returned Agent Studio connector id.
- `runtime_stream_url` (`string`): derived endpoint used by the frontend panel.
- `status` (`string`): `not_configured`, `disabled`, `connected`, `error`.
- `last_validated_at` (`string`): RFC3339 timestamp.
- `last_error` (`string`): most recent validation/provision error, empty on success.

## Security

- The Agent Studio service token is never returned in full to the frontend.
- Read responses only expose `serviceTokenConfigured: boolean`.
- Updating the token is optional. Empty token in an update means "keep existing token".
- Provision calls happen only from goomc backend to Agent Studio.
- Runtime browser calls never receive the provision token. They use short-lived delegation tokens issued from the current web user token.
- Connector policy remains read-only for this slice: `allowReadActions=true`, `allowLowRiskActions=false`, `allowHighRiskActions=false`.

## UI Design

The frontend adds one Agent tab to all three skins.

- v1: Ant Design structured form, matching existing `SystemConfig` tab style.
- v2: shadcn/Tailwind compact settings panel, matching the current editable KV page.
- v3: dark HUD table-like editor, matching the existing neon system config surface.

The generated mockups define the layout and hierarchy:

- Enable Agent switch.
- Agent Studio API URL.
- Service Token masked input plus replace action.
- Public API URL.
- Connector ID readonly field with copy.
- Runtime Stream URL readonly field with copy.
- Connection status, last validation time, and test result.
- Footer actions: reset, test connection, save and sync.

No Baicells, goomc, or product-specific brand names are introduced in new Agent Studio code. goomc UI may use existing OMC wording already present in the app shell.

## Reuse Boundary

This implementation moves toward a reusable Agent integration shape:

- `frontend-core/src/agentkit` remains protocol/client/state-machine oriented.
- `useAgentRuntimeClient` consumes a runtime config provider instead of only reading build env.
- goomc backend keeps business actions in `agentaction`, but connector configuration becomes generic and reusable.
- Agent Studio provision API accepts display name, slug, base URL, paths, and policy, so other business systems can reuse the same endpoint.

## Error Handling

- Invalid URLs return HTTP 400 with actionable messages.
- Missing Agent Studio token returns HTTP 400 before a remote call.
- Remote provision failures store `status=error` and `last_error`, then return a failed response to the UI.
- The Agent panel shows disabled/disconnected state if runtime config is missing or disabled.

## Testing

### goomc

- Unit tests for Agent config service:
  - masks token on read.
  - keeps existing token when update omits token.
  - builds runtime stream URL from Agent Studio base URL and connector id.
  - handles provision success and failure.
- Frontend tests for runtime config resolution:
  - API runtime config wins over Vite env.
  - Vite env remains fallback.
- Run:
  - `cd omcgo && go test ./internal/agentconfig ./internal/agentaction ./internal/admin`
  - `cd omcmb && npm run typecheck`
  - `cd omcmb && npm run skin-parity`

### agent-studio

- Router/service tests for generic action connector provision:
  - rejects missing service token.
  - creates connector by slug.
  - updates existing connector by slug.
  - returns runtime stream URL.
- Run relevant API tests and typecheck.

## Out of Scope

- Low-risk or high-risk write actions.
- Cross-system OAuth setup between goomc and Agent Studio.
- Multi-tenant connector selection in goomc UI.
- MCP integration.
