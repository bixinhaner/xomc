# Review Report

- Date: 2026-08-06
- Branch: `fix/gnb-ntp-server-visibility`
- Scope: `device-time`
- Result: PASS_WITH_WARNINGS

## Summary

本次改动只影响 gNB 快速设置里的 NTP 模式联动：切到 server 时隐藏 NTP 服务器 1-5，切回 client 时恢复显示，并在保存时跳过隐藏字段。

## Findings

### WARNING

1. `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx`
   - 设备时间模式的可见性依赖 `instanceContext.networkType` 与 `Enable` 下拉值联动；当前实现已经通过 `normalizeQuickSettingsNetworkType` 与页面内 `onChange` 做了同步刷新，行为上满足需求，但这条链路依赖多个状态源，后续改动需注意不要再只改标签不改显隐。

### INFO

1. `npm run typecheck` 在 `omcmb` 根目录失败，但错误来自既有的 [SystemLicense/History.tsx](/Users/wangyong/OBJECT/Codex/xomc/omcmb/webcode/src/pages/SystemLicense/History.tsx) 与 [SystemLicense/index.tsx](/Users/wangyong/OBJECT/Codex/xomc/omcmb/webcode/src/pages/SystemLicense/index.tsx) 的 `BlobPart` 类型问题，不在本次改动范围内。

## Validation

- `cd /Users/wangyong/OBJECT/Codex/xomc/omcmb/webcode && npm test -- --run src/pages/device/DeviceDetail/QuickSettingsTab/__tests__/deviceTimeMode.test.ts` — 通过，5/5 tests passed
- 本地浏览器自测：gNB 设备 [1202000534228JB0007](http://127.0.0.1:4173/device/detail/1202000534228JB0007?tab=quickSettings) 在 NTP Client 时显示 NTP 服务器 1-5，切到 NTP Server 后字段隐藏，切回 Client 后恢复显示

## Conclusion

PASS_WITH_WARNINGS
