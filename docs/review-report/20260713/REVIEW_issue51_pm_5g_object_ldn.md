# Issue #51 审查记录

## 范围

- 分支：`fix/51-pm-5g-object-ldn`
- 对比：`origin/main...HEAD`
- 规格来源：GitLab Issue #51
- 变更文件：
  - `docs/review-report/20260713/REVIEW_issue51_pm_5g_object_ldn.md`
  - `omcmb/frontend-core/src/i18n/en-US/index.ts`
  - `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
  - `omcmb/frontend-core/src/types/pmObject.ts`
  - `omcmb/frontend-core/src/types/__tests__/pmObject.test.ts`
  - `omcmb/webcode/src/pages/performance/PmDashboard/ChartCard.tsx`
  - `omcmb/webcode/src/pages/performance/PmDashboard/ChartCard.test.tsx`
  - `omcmb/webcode/src/pages/performance/PmDashboard/deviceListUtils.test.ts`

## Standards

未发现违反仓库标准的问题。

- 前端共享业务逻辑仍位于 `frontend-core`，PM 页面只消费共享 helper，符合根 `AGENTS.md` 的前端分层要求。
- 用户可见新增文案均进入 `frontend-core` 的中英文 i18n 词条，未在组件中硬编码。
- 未触及后端 SQL、运营商适配、安全敏感路径或迁移。
- `buildDeviceSeriesName` 现有 4G/GSM/缺失 `object_ldn` 行为由既有测试继续覆盖。
- 固定 tooltip 的新增交互局限在 PM 图表组件内，未扩宽公共接口为 `any` 或 `interface{}`。

## Spec

未发现规格缺口。

- 5G `object_ldn` 在多站图 series name 中改为原始字符串，保留 `CUID`、`DUID`、`PLMNID` 字段。
- 同 SN、同 NrCGI、同 PLMN 的 CU/DU 样本新增回归测试，断言 series name 不相等并包含 `CUID=1` / `DUID=1`。
- tooltip formatter 测试直接断言 HTML 中包含完整原始 5G `object_ldn`。
- 针对“对象很多时 hover tooltip 无法实际滚动”的后续验证问题，图表支持点击数据点固定 tooltip；固定浮层可滚动、可关闭，也可按 Escape 关闭。
- hover tooltip 顶部提示“对象较多，点击图表固定后可滚动查看”，避免用户不知道点击可固定；该提示已移到对象列表之前。
- hover tooltip 与固定 tooltip 的宽度、背景、边框、阴影、文字颜色和行布局保持同一视觉口径，固定态文字可读。
- 未重设计 `formatObjectLdn`，其他需要友好名的展示路径保持原口径。

## DoD 适用项

- 后端 `go build ./...`：本 MR 未触及后端；P5 记录为通过。
- 后端 `go test ./...`：本 MR 未触及后端；P5 记录为通过。
- 前端 `npm run typecheck`：通过。
- 前端 lint：触及文件 lint 通过；全量 `npm run lint` 的既有失败不由本次变更引入。
- 前端相关测试：`pmObject.test.ts`、`deviceListUtils.test.ts`、`ChartCard.test.tsx` 通过。
- Docker 部署：通过 `$docker-deploy` 方式部署到本机 `8081`。
- 前端真实后端浏览器冒烟：通过；在 `http://127.0.0.1:8081/performance/device-view` 写入 36 条 5G CU/DU 造假对象数据验证。固定 tooltip 可滚动，hover/固定样式一致，顶部可见点击固定提示。
- 浏览器证据：
  - `/Users/shangyingbin/project/.codex-tools/browser-control/issue51-tooltip-hint-top.png`
  - `/Users/shangyingbin/project/.codex-tools/browser-control/issue-51-pm-smoke-evidence.json`
- 新 REST 端点、迁移、安全敏感路径、指标/日志新增：N/A。

## 结论

P7 通过。未发现 CRITICAL、HIGH 或 MEDIUM 问题。
