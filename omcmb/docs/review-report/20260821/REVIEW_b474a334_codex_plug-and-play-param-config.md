# Review Report: PlugAndPlay Param Config Fixes

- Date: 2026-08-21
- Branch: `fix/plug-and-play-param-config`
- Scope: `PlugAndPlay`
- Result: PASS

## Summary

本次审查覆盖即插即用策略编辑页参数自配置相关改动，重点关注模块切换、多文件导入、网络快速配置抽屉、接口名称保存与下载导出链路。

## Findings

### CRITICAL

无。

### WARNING

无。

### INFO

- 单行下载对 `gNB INTERFACE.Interface Name` 增加了明确系统列白名单。该处理保持了原有未知列过滤策略，风险集中在接口名称这一既有页面字段的导出兼容性。
- 网络快速配置分组增加前端缓存，适合当前按参数模型复用的页面行为；若后续参数模型配置支持运行时热更新，可能需要补充缓存失效机制。

## Reviewed Areas

- `AddPolicyPage.tsx`
  - 模块切换改为 Form 字段驱动。
  - 多文件导入按文件和配置行展开预览。
  - 编辑已有策略时配置抽屉保存会立即调用策略 PUT。
- `CommonQuickSettingsNetworkCards.tsx`
  - 网络分组缓存减少抽屉打开等待。
  - 接口名称绑定到 `sheetParameters.INTERFACE[*]["Interface Name"]`。
- `paramConfigDetail.ts`
  - 合并保存时补齐 gNB 接口名称系统列。
  - 清理网络接口名称从 `customParams` 和 `sheetParameters` 双写导致的歧义。
- `paramConfigWorkbook.ts`
  - 导出时补充已有 `参数映射` 中缺失的系统映射。
  - 单行下载放行 `gNB INTERFACE.Interface Name`，避免保存值被映射过滤丢弃。
- Tests and docs
  - 增加参数详情、Workbook、网络配置卡片与页面源码回归用例。
  - 新增本次修复记录文档。

## Verification

- `npm test -- --run src/pages/device/PlugAndPlay/CommonQuickSettingsNetworkCards.test.tsx src/pages/device/PlugAndPlay/paramConfigDetail.test.ts src/pages/device/PlugAndPlay/specifiedParamConfigEditor.test.ts src/pages/device/PlugAndPlay/paramConfigWorkbook.test.ts` — 100 passed
- `npm run typecheck` in `omcmb` — passed
- `npm run build` in `omcmb/webcode` — passed
- `git diff --check` — passed
- Docker Compose hot redeploy for local web stack — passed
- Browser self-test:
  - module switching works
  - network drawer save persists interface name
  - single row download xlsx contains `INTERFACE.Interface Name = 1111`

## Residual Risk

低。改动集中在即插即用参数自配置前端链路和 workbook 读写逻辑，已有单元测试与真实浏览器下载验证覆盖用户反馈路径。
