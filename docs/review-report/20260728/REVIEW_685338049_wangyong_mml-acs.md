# Review Report: MML ACS Standard Path Fix

Date: 2026-07-28
Author: wangyong
Scope: mml-acs
Base: 685338049
Conclusion: PASS

## Summary

This review covers the staged changes for preserving standard TR-069 paths through MML-to-ACS execution, request-aware ACS response name translation, MML Console result parsing safeguards, and command tree visibility filtering.

## Findings

No CRITICAL findings.

No WARNING findings.

INFO:
- The frontend large XML parser intentionally uses a lightweight regex path only for large GPV payloads or caller-limited summary parsing. Full parser semantics remain unchanged when `maxParams` is omitted.
- The ACS response fallback intentionally keeps the private response path when a standard template still contains unresolved `{i}` placeholders, avoiding fabricated standard names.
- Command tree visibility now relies on supported sub-field/path intersection instead of product technology string heuristics; this may reveal commands that were previously hidden by naming assumptions but are supported by the param model.

## Verification

- `git diff --staged --check` — passed.
- `cd omcgo && go build ./...` — passed.
- `cd omcgo && go test ./internal/acs ./internal/config/parammodel ./internal/mml ./cmd/app/provider` — passed.
- `cd omcmb && npm run typecheck` — passed.
- `cd omcmb/webcode && npm run test -- src/pages/mml/Console/__tests__/objectPathResults.test.ts src/pages/mml/Console/__tests__/buildDeviceRows.test.ts src/pages/mml/Console/__tests__/useConsoleHistory.test.ts ../frontend-core/src/utils/__tests__/mmlResultParser.test.ts` — passed.
- `cd omcmb/webcode && npm run build` — passed before submission review for the same staged code.
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --no-deps --build acs app web` — passed before submission review.
- `curl -I --max-time 10 http://localhost:8081/` — returned `HTTP/1.1 200 OK`.

## Risk Notes

- Full compose migration is still affected by the existing TSDB migration issue around `public.pm_measurement_anchors`; this submission used a targeted no-deps rebuild for `acs`, `app`, and `web`.
