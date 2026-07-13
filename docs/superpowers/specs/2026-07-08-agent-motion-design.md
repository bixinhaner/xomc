# Agent Motion Design

## Goal

Add restrained, state-driven motion to the embedded Agent panel so users can see when the agent is connected, thinking, running actions, streaming an answer, and finished.

## Design Direction

Use real runtime state, not decorative animation. Motion appears only when the agent is active:

- `connected`: small live status dot near the connection badge.
- `streaming/thought`: active avatar ring, thought block pulse, and a thin activity line.
- `process/tool`: process rows show active/completed visual states.
- `delta`: answer text shows a compact streaming caret.
- `done/error`: active motion stops, leaving static status and result affordances.

## Frontend Scope

The implementation applies to all three skins:

- `webcode`: dark enterprise drawer, blue activity accents.
- `webcode-v2`: light enterprise drawer, blue/emerald status accents.
- `webcode-v3`: cyber operations console, cyan scan/pulse accents.

Shared protocol and controller data remain unchanged. The UI consumes existing `controller.isStreaming`, assistant message `status`, `thoughts`, `process`, and activity `status`.

## Interaction Rules

- No animation should resize the panel, message bubbles, toolbar, composer, or process rows.
- Active motion must stop when a request completes or fails.
- Users with `prefers-reduced-motion: reduce` get static status indicators only.
- The component must stay usable on mobile drawer width.

## Visual References

Generated UI mockups were used as implementation references for:

- v1 dark operations panel: `docs/superpowers/specs/agent-motion-mockups/v1-dark-operations.png`.
- v2 light operations panel: `docs/superpowers/specs/agent-motion-mockups/v2-light-operations.png`.
- v3 cyber operations console: `docs/superpowers/specs/agent-motion-mockups/v3-cyber-operations.png`.

## Testing

- Run agentkit unit tests for event/state regressions.
- Run `npm run typecheck`.
- Run `npm run skin-parity`.
- Use browser verification against `http://localhost:8081/` if layout risk appears after implementation.
