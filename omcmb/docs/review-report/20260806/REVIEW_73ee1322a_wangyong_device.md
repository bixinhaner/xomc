# Issue #258：BM 站 Band 识别修复审查报告

- 审查日期：2026-08-06
- 审查范围：`omcmb/webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.ts`、`omcmb/webcode/src/pages/device/DeviceList/deviceRadioFieldSupport.test.ts`
- 结论：PASS_WITH_WARNINGS

## 变更摘要

- 设备列表的 GSM band 展示新增带宽样式值识别，`GSM200K` / `200` 会改为基于 ARFCN 推导真实频段。
- 设备级 RF 状态仍按产品型号区分，BSC 继续显示 `-`，BTS/LTE 保留自身值。
- 新增单测覆盖带宽样式 band、BTS 发射功率和未知型号的缺失语义。

## 审查结果

### CRITICAL

无。

### WARNING

- `formatGsmBandFromArfcn()` 中 `PCS1900` 分支目前被 `DCS1800` 的更宽范围完全覆盖，实际不可达。虽然这不影响本次 Issue #258 的主修复，但如果后续要扩展 GSM band 推导，建议重新梳理 ARFCN 区间或补充分支判定依据。

### INFO

- 当前实现只在 `networkType === 'GSM' && field === 'band'` 时改写展示值，作用范围窄，未影响其他 radio 字段。
- 未引入 SQL、接口契约、权限或运营商硬编码改动。
- 单测覆盖了本次新增的主路径，能稳定回归 `GSM200K` 的展示修复。

## 验证记录

- `cd omcmb/webcode && npx vitest run src/pages/device/DeviceList/deviceRadioFieldSupport.test.ts`：8 个用例全部通过。
- `cd omcmb && npm run typecheck`：未通过，但失败来自仓库既有的 `src/pages/SystemLicense/History.tsx` 和 `src/pages/SystemLicense/index.tsx` 的 `BlobPart` 类型问题，与本次改动无关。

## 风险与回滚

- 风险集中在 GSM band 推导规则本身；当前仅对带宽样式值做兜底转换，未影响非 GSM 或非 band 字段。
- 回滚可直接撤销本次提交，不涉及数据迁移或外部接口变更。
