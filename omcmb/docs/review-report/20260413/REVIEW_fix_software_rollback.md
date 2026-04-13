# Code Review: 版本回退页面修复与清理

| 项目 | 详情 |
|------|------|
| 日期 | 2026-04-13 |
| 审查人 | Claude AI |
| Scope | software, components |
| 结论 | **PASS** |

---

## 变更文件

- `webcode/src/components/DataTable/index.tsx` (+8/-2)
- `webcode/src/pages/software/VersionRollback/index.tsx` (+19/-142)

---

## 审查发现

### INFO (5)

1. **DataTable 分页数据切片** — 新增 `pagedData` useMemo，按 `currentPage`/`pageSize` 对 `filteredData` 切片后传给 Table，修复分页与列表显示不联动的问题
2. **移除未使用代码** — 清理 7 个未使用 state、4 个未使用 computed、4 个未使用导入、2 个未使用类型，共减少 ~130 行
3. **表格优化** — 任务列表和设备列表均添加 `showRowNumber`/`rowNumberTitle`、`scroll.y` 纵向滚动，与版本升级页面一致
4. **DatePicker 类型修复** — `current < new Date()` (Dayjs vs Date) 改为 `current.isBefore(new Date(), 'day')`
5. **移除弃用属性** — 3 处 `Card` 组件的 `bordered={false}` 已在 Ant Design 5 中弃用，移除

---

## 前端专家审查清单

- [x] 类型安全：修复 Dayjs 比较的类型错误，无 `any` 使用
- [x] 组件复用：DataTable 分页逻辑统一在组件层处理
- [x] 无 XSS 风险
- [x] 国际化：无新增用户可见文本
