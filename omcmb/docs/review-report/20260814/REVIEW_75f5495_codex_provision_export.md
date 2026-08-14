# 参数配置导出字段去重与编排文案审查报告

## 审查范围

- `frontend-core/src/i18n/en-US/index.ts`
- `frontend-core/src/i18n/zh-CN/index.ts`
- `frontend-core/src/i18n/__tests__/provisionMessages.test.ts`
- `webcode/src/pages/device/PlugAndPlay/paramConfigWorkbook.ts`
- `webcode/src/pages/device/PlugAndPlay/paramConfigWorkbook.test.ts`

## 结论

PASS

未发现 CRITICAL 或 WARNING 级问题。

## 审查摘要

- 参数配置工作簿按 `sheet + TRPath` 识别同一参数，避免旧表头与新表头同时导出为重复列。
- 导出列优先保持模板顺序，其次遵循参数映射顺序；重复别名会保留首个非空值，避免数据丢失。
- 隐藏参数映射表采用相同去重口径，保证重新导入时映射唯一。
- 空 `TRPath` 的映射按表头分别保留，避免未映射参数被错误合并。
- 编排步骤 `verify_online` 的中英文展示改为校验小区激活，与实际业务动作一致。
- 回归测试覆盖重复 TRPath、映射顺序、别名值回填及中英文文案。
- 未涉及接口、数据库、权限、Token 或迁移变更。

## 验证

- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb/webcode && npm test -- --run src/pages/device/PlugAndPlay/paramConfigWorkbook.test.ts ../frontend-core/src/i18n/__tests__/provisionMessages.test.ts`：2 个文件、33 项测试通过。
- `cd omcmb && npx eslint --no-warn-ignored <本次 5 个变更文件>`：通过。
- `git diff --check`：通过。

## 风险与影响

- 影响范围限于参数配置 Excel 导出与共享编排步骤文案。
- 若同一 TRPath 存在多个历史表头，导出仅保留按模板/映射顺序选出的首列表头，并从别名中回填首个非空值；该行为符合去重目标。
