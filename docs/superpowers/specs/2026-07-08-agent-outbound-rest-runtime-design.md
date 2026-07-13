# Agent Outbound REST Runtime Design

## Goal

Build the Agent integration as an outbound tool execution flow: Agent Studio handles Codex reasoning and conversation continuity, while OMC executes its own `/api/v1` APIs locally and returns tool results outbound. Agent Studio must not contain OMC business actions.

## Architecture

OMC starts the chat stream by calling Agent Studio. During a Codex run, Agent Studio exposes a generic `rest.request` CLI to Codex. The CLI sends a `tool_request` to Agent Studio's generic bridge and waits. Agent Studio forwards the request over the active SSE stream to OMC. OMC validates the request against its local policy and catalog, executes the local API with the current user's delegation identity, and posts a `tool_result` back to Agent Studio. The CLI returns the result to Codex, so the same run can continue with more API calls.

Network direction is OMC outbound only:

```text
OMC -> Agent Studio chat stream
OMC -> Agent Studio tool result callback
```

Agent Studio never calls OMC directly.

## OMC Responsibilities

- Provide the runtime chat proxy and outbound tool-result callback.
- Provide a generic REST gateway for `/api/v1`.
- Enforce thin hard boundaries:
  - Only configured HTTP methods are allowed.
  - Only `/api/v1` paths are allowed.
  - Configured blocked path prefixes are denied.
  - Model-supplied `Authorization`, `Host`, and arbitrary upstream URLs are ignored.
  - Response size and request timeout are configurable.
- Expose those boundaries in System Config -> Agent.
- Surface tool execution process in the Agent panel.
- Remove dependency on the old hand-written `/api/v1/agent-actions` action list for runtime use.

## Agent Studio Responsibilities

- Provide a generic bridge for `tool_request` and `tool_result`.
- Keep tool requests in the same Codex run so multi-step A/B/C/D/A querying remains coherent.
- Emit generic Agent stream events only. No OMC-specific code or labels.
- Keep prompt/skill editable so business behavior can be tuned without Agent Studio business code.

## Frontend Reference

The implementation must follow these generated reference images:

- `docs/design/agent-outbound-runtime/system-config-agent-policy.png`
- `docs/design/agent-outbound-runtime/agent-panel-tool-execution.png`

System Config adds a Security Policy card. The Agent panel shows REST tool execution cards and a read-only/write-disabled state using the existing panel style.

## Phasing

The first production slice supports read-only execution by default. Write methods can be enabled in configuration, but OMC still enforces the configured method allow-list and blocked path list. Rich business-specific risk logic remains in prompt/skill, not in OMC code.
