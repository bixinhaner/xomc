# Review: Quick Settings NR 枚举与三列布局

- 日期: 2026-07-11
- 分支: feat/quicksettings-nr-layout
- 范围: Quick Settings / BaiBNQ NR 小区参数
- 结论: PASS

## 暂存文件

- `omcgo/data/quicksettings/BaiBNQ.xml`
- `omcmb/frontend-core/src/i18n/en-US/index.ts`
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/MultiInstanceTable.tsx`

## 审查结论

未发现 CRITICAL / WARNING 问题。

## 检查项

- XML 枚举: `DLSubCarrierSpacing` / `ULSubCarrierSpacing` 增加 `0/1/2 -> 15kHz/30kHz/60kHz` 下拉元数据，loader 支持 `<option>` 结构。
- 字段顺序: BaiBNQ 小区参数按上下行成对排序，便于三列布局下扫描。
- 带宽联动: `DLCarrierBandWidth` / `ULCarrierBandWidth` 按对应 SCS 动态展示 RB 枚举，提交值仍为 RB 数值。
- 布局: 普通表单项从 2 列改为 3 列，整行表格类控件仍保持 `span=24`。
- 数据下发: Select 使用枚举 `value` 下发，不下发展示文案。
- NR 邻区校验: 新增 NR 邻区时按 SSB 频点反查 NR 异频邻频，未配置或未启用时阻止保存并提示用户。
- i18n: 新增 NR 异频邻频加载中、未配置、未启用三类中英文错误提示。

## 验证

- `cd omcmb && npm run typecheck` — 通过
- `cd omcgo && go test ./internal/quicksettings` — 通过
- `cd omcmb/webcode && npm run build` — 通过
- `OMC_PROJECT=goomc-local bash deployments/docker/dc.sh up -d --build web` — 通过
- `curl -I --max-time 10 http://localhost:8081/` — `HTTP/1.1 200 OK`

## 风险

- 三列布局会压缩单项宽度；当前实现保留了整行控件 `span=24`，普通输入框使用 Ant Design 栅格自适应。
- NR 邻区校验依赖异频邻频 schema 回读；加载中会提示用户稍后再保存，避免在依赖数据未就绪时误放行。
