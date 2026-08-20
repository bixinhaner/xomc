# Review: PlugAndPlay LTE Bandwidth Import

## 结论

PASS

## 范围

- `webcode/src/pages/device/PlugAndPlay/EnbQuickSettingsCards.tsx`
- `webcode/src/pages/device/PlugAndPlay/paramConfigWorkbook.ts`
- `webcode/src/pages/device/PlugAndPlay/enbQuickSettingsFields.test.ts`
- `webcode/src/pages/device/PlugAndPlay/paramConfigWorkbook.test.ts`

## 关键检查

- LTE 带宽页面选项：产品模型返回 `25/50/75/100` 时，页面层统一规范为 `n25/n50/n75/n100`。
- 参数配置文件模板：BLN、BLQ、MLN、MLQ 的下行/上行带宽下拉值统一使用 `n` 前缀。
- 导入兼容性：导入校验接受新格式 `n50`，也兼容旧文件中的 `50`。
- 影响隔离：仅针对 LTE `DLBandwidth` / `ULBandwidth` TRPath 和 eNB 带宽字段名做特殊处理，未改变 NR 带宽联动逻辑。

## Findings

- CRITICAL: 无
- WARNING: 无
- INFO: 后端下发层已有 `n50` 与 `50` 双格式兼容，本次修复聚焦前端页面/Excel 导入层格式一致性。

## 验证

- `npm test -- --run src/pages/device/PlugAndPlay/enbQuickSettingsFields.test.ts src/pages/device/PlugAndPlay/paramConfigWorkbook.test.ts` — 通过，2 个文件 49 个用例
- `npm run typecheck -- --pretty false` — 通过

## Issue

- Closes #349
