# Review Report: PlugAndPlay Param Config Export Order

- Date: 2026-08-21
- Branch: `feat/plug-and-play-param-config-editor`
- Scope: `PlugAndPlay`
- Result: PASS

## Summary

本次审查覆盖即插即用参数自配置导入后编辑文件的下载/导出顺序修复，重点确认已编辑 workbook 的参数列顺序与当前下载模板一致。

## Findings

### CRITICAL

无。

### WARNING

无。

### INFO

- 下载已导入/编辑的配置时，导出上下文复用当前产品参数模型的 quick settings 与 mapping metadata；在 metadata 不完整时保持原有导出顺序，避免影响历史数据或异常场景。
- 自定义列不参与模板排序，保留原稳定顺序并追加在模板列之后，便于兼容用户导入文件中的额外字段。
- 浏览器自测中观察到既有 Ant Design `Descriptions` 废弃属性 warning，未影响本次即插即用页面打开与功能检查。

## Reviewed Areas

- `AddPolicyPage.tsx`
  - 单行下载与批量导出均传入同一份导出上下文。
  - 导出上下文包含 device type、product class、quick settings groups、parameter mappings 和 quick setting fields。
- `paramConfigWorkbook.ts`
  - 增加按模板 sheet/header 计算已编辑配置导出顺序的逻辑。
  - `Serial Number` 保持首列，模板内已知参数按模板顺序输出，未知/自定义列稳定后置。
- Tests
  - 增加 gNB CELL 导入后下载顺序回归用例。
  - 更新单行下载源码断言，确保使用完整导出上下文。

## Verification

- `npm test -- paramConfigWorkbook.test.ts specifiedParamConfigEditor.test.ts` in `omcmb/webcode` — passed, 63 tests
- `npm run typecheck` in `omcmb/webcode` — passed
- `npm run build` in `omcmb/webcode` — passed
- Docker Compose hot redeploy for local web stack — passed
- `curl -I --max-time 10 http://localhost:8081/` — returned `HTTP/1.1 200 OK`
- Browser smoke test for `/device/plug-and-play` and `/device/plug-and-play/add` — passed

## Residual Risk

低。改动集中在即插即用参数自配置 Excel 导出排序，已通过 workbook 单元测试、类型检查、构建、热部署和页面冒烟验证。
