# Active Intelligence Beginner Flow Design QA

- Selected ImageGen source: `.codex/design-references/beginner-ai-config-conversation-first.png`
- Current OMC implementation: `.codex/design-references/active-intelligence-beginner-final-1440x1024.png`
- Side-by-side comparison: `.codex/design-references/active-intelligence-beginner-comparison.png`
- Verified viewport: 1440 x 1024 in the Codex in-app browser

## Visual comparison

- Reproduces the selected conversation-first layout: a large guided conversation area, a live plain-language helper summary, quick choices, example requests, a safe preview, and progressively disclosed professional settings.
- Preserves the real OMC V1 dark shell and its functional tab bar. The ImageGen reference omitted the tab bar, but the implementation keeps it because direct-link and refresh persistence are required product behavior.
- Uses plain user language on the primary surface. Event names, allowed tools, deduplication, and rate-limit details only appear after the user opens `需要专业设置？`.
- P0: none.
- P1: none.
- P2: none.
- P3: minor typography, icon artwork, and spacing differences caused by the live OMC shell and Ant Design rendering.

## Interaction verification

- Direct navigation to `/system/active-intelligence`: passed.
- Refresh keeps the `主动智能` tab selected: passed.
- `系统管理` dynamic menu contains `主动智能` with the robot icon: passed.
- `一起处理` -> `先只给我看` -> `先试试看`: passed.
- Preview modal opens and `再聊聊` closes it without changing configuration: passed.
- `需要专业设置？` opens the progressive-disclosure drawer: passed.
- Final browser console check after reload: no new entries.
- The final activation action was intentionally not executed during visual QA because it changes scenario rollout state and remains an explicit user confirmation.

final result: passed
