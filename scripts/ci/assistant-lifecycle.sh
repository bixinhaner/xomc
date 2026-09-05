#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../.."
case "${1:-}" in
  backend)
    cd omcgo
    go test -p 2 ./internal/agentassistant ./internal/agentbridge ./internal/agentruntime ./internal/attention ./cmd/app/provider
    ;;
  frontend)
    cd omcmb
    npm ci --no-audit --no-fund
    npm run typecheck --workspace webcode
    npm run test --workspace webcode -- src/pages/system/ActiveIntelligence src/pages/dashboard/components/AttentionBar.test.tsx
    npm run build --workspace webcode
    ;;
  browser)
    cd omcmb/webcode
    npx playwright install --with-deps chromium
    npx playwright test --config playwright.assistants.config.ts
    ;;
  *) echo 'usage: assistant-lifecycle.sh backend|frontend|browser' >&2; exit 2;;
esac
