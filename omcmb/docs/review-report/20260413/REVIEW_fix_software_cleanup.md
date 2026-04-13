# Code Review: FirmwareUpload 页面清理与类型修复

| 项目 | 详情 |
|------|------|
| 日期 | 2026-04-13 |
| 审查人 | Claude AI |
| Scope | software |
| 结论 | **PASS** |

---

## 变更文件

- `webcode/src/pages/software/FirmwareUpload/index.tsx`

---

## 审查发现

### INFO (6)

1. **移除未使用导入** — `Table`, `Tag`, `Divider`, `Checkbox`, `useUploadSoftwareVersion` 均未在组件中使用，清理合理
2. **移除未使用常量** — `fileTypeMap` 和 `versionTypeMap` 已无引用，移除正确
3. **修复未使用变量** — `form.validateFields().then((vals) =>` 中 `vals` 未使用，改为 `() =>`
4. **移除多余 useMemo** — `columns` 的 `useMemo(..., [])` 依赖为空数组，无记忆化效果，移除合理
5. **优化表格滚动** — `scroll.x` 从固定 `1100` 改为 `'max-content'`，新增 `scroll.y` 纵向滚动，添加行号列
6. **修复 TS 类型错误** — `collapseTags`(Element UI API) → `maxTagCount="responsive"`(Ant Design 5 API)

---

## 前端专家审查清单

- [x] 类型安全：修复了 `collapseTags` 类型错误，无 `any` 使用
- [x] 组件复用：使用 `DataTable`、`FilterBar` 项目组件
- [x] 无 XSS 风险：无 `dangerouslySetInnerHTML`
- [x] 国际化：无新增用户可见文本
