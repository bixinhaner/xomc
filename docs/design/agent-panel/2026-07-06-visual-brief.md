# AI Agent Panel Visual Brief

Date: 2026-07-06

## Scope

This brief is the visual source of truth for the first Agent Panel implementation in the three web skins.
Implementation must follow the approved mockups before adding or changing layout decisions.

## User Goal

Operators should be able to ask operational questions without leaving the current page.
The panel must keep the current dashboard context visible, show which read-only action is used, and make action preview/confirmation explicit before execution.

## Shared Interaction Rules

- Entry: a compact Agent icon button in the existing top or primary navigation area.
- Placement: right dock, not a modal. It should sit above the current page and preserve the user's current context.
- Width: about `380px` to `400px` on desktop, responsive to full-screen drawer on narrow viewports.
- States:
  - Chat: user message, assistant streaming response, tool call chip, structured result card.
  - Preview: read-only action card, parameters, policy explanation, execute/cancel controls, preview result schema.
- Safety copy: clearly state read-only/no network change for previewed actions.
- Composer: fixed bottom input with send icon. It must not resize the layout while streaming.
- Text: final visible labels must go through i18n. Mockup text is illustrative.
- Shared protocol/action code must remain product-neutral and must not hardcode business-specific project names.

## Skin-Specific Direction

### v1

Use the existing dark Ant Design-like surfaces:

- dark charcoal dock with thin border;
- blue primary action;
- compact table/result rows;
- top-bar icon uses the existing icon-button scale;
- no neon styling.

Mockups:

- `assets/mockup-v1-chat.png`
- `assets/mockup-v1-preview.png`

### v2

Use the existing light, quiet enterprise dashboard style:

- white dock with subtle border/shadow;
- blue primary action;
- green success and pale status chips;
- table/result cards should match existing dashboard cards;
- avoid dense cyber styling.

Mockups:

- `assets/mockup-v2-chat.png`
- `assets/mockup-v2-preview.png`

### v3

Use the existing neon control-console style:

- dark translucent dock with cyan border;
- monospaced compact labels;
- cyan primary action, yellow read-only badge;
- preserve severity colors already used by the dashboard;
- keep readability ahead of decoration.

Mockups:

- `assets/mockup-v3-chat.png`
- `assets/mockup-v3-preview.png`

## Implementation Guardrails

- The shared AgentKit primitives should live in `frontend-core`.
- Each skin may provide only thin shell styling and placement.
- The first UI slice should wire read-only actions only.
- Mutating actions remain unsupported in UI until policy, audit, and approval states are implemented.
- Do not add a new landing page, full-screen marketing page, or separate agent product surface.

