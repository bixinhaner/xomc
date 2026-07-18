# Review: notifications message routing

- Date: 2026-07-18
- Author: wangyong
- Scope: notifications
- Result: PASS_WITH_WARNINGS

## Summary

This change fixes notification-center item navigation so a top-bar notification opens the notification center message detail instead of jumping to the backend related device link. It also makes the message detail drawer deep-linkable through `messageId`, clears that URL state when the drawer is closed, and aligns the notification message refresh controls with the device list refresh behavior.

## Files Reviewed

- `webcode/src/components/NotificationCenter/index.tsx`
- `webcode/src/components/NotificationCenter/notificationRoute.test.ts`
- `webcode/src/pages/notifications/MessageList.tsx`

## Findings

### CRITICAL

None.

### WARNING

- `MessageDetailDrawer` still uses the deprecated Ant Design Drawer `width` prop. Existing browser smoke tests report `Warning: [antd: Drawer] width is deprecated. Please use size instead.` This is non-blocking and unrelated to the routing or close behavior fixed here.

### INFO

- The click route now uses an encoded `messageId` query parameter and intentionally ignores backend `link` values for top-bar notification item clicks.
- The detail drawer auto-open effect is guarded by `dismissedMessageIdRef` so a user-closed drawer is not immediately reopened by a stale `messageId`.
- The message list manual refresh and auto-refresh buttons now share the same spinner timing pattern as the device list.

## Verification

- `cd omcmb/webcode && npm run typecheck` - passed.
- `cd omcmb/webcode && npm run test -- NotificationCenter/notificationRoute.test.ts` - passed, 2 tests.
- `cd omcmb/webcode && npm run build` - passed with existing Vite deprecation/chunk-size warnings.
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web` - passed, web container restarted.
- `curl -I --max-time 10 http://localhost:8081/` - returned `200 OK`.
- Browser smoke: notification detail opens at `/notifications?tab=messages&messageId=...`, drawer close clears `messageId`, drawer is hidden.
- Browser smoke: message-list refresh button enters loading then recovers; auto-refresh menu can select `15秒` and the button becomes primary.

## Risk

Low. The change is limited to the v1 notification center message UI and adds a small route helper test. It does not change backend APIs or persisted data.
