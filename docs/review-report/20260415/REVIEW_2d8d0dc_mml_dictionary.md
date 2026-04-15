# Code Review: MML 控制台字典驱动产品类型与命令分类

**日期**: 2026-04-15
**审查范围**: omcgo (migration seed) + omcmb (adminApi, hooks, Console 组件)
**审查结论**: PASS

---

## 审查文件

| # | 文件 | 变更类型 |
|---|------|---------|
| 1 | `omcgo/migrations/seed/900004_mml_enhance.sql` | 追加字典种子 (product_type + mml_command_category) |
| 2 | `omcmb/webcode/src/services/api/adminApi.ts` | 新增 findDictionaryByType + Dictionary 类型扩展 |
| 3 | `omcmb/webcode/src/hooks/api/useSystem.ts` | 新增 useDictionary hook |
| 4 | `omcmb/webcode/src/pages/mml/Console/types.ts` | 删除 PRODUCT_TYPE_OPTIONS |
| 5 | `omcmb/webcode/src/pages/mml/Console/components/DeviceTree.tsx` | useDictionary('product_type') 替换硬编码 |
| 6 | `omcmb/webcode/src/pages/mml/Console/hooks/useCommandSelection.ts` | useDictionary('mml_command_category') 驱动分类 |
| 7 | `omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx` | 新增 categoryOptions props |
| 8 | `omcmb/webcode/src/pages/mml/Console/index.tsx` | 传递 categoryOptions |

---

## 发现

### CRITICAL — 无

### WARNING — 无

### INFO

**I1. 字典种子 SQL 幂等性** — 字典 INSERT 使用 `ON CONFLICT DO NOTHING`，详情 INSERT 使用子查询关联，重复执行安全

**I2. useDictionary hook 设计良好** — `enabled` 守卫防止空查询，`staleTime: 5min` 避免频繁请求，query key 层级式 `['dictionary', dictType]`

**I3. categoryOptions 有 fallback** — 字典未加载时从命令数据动态提取，不影响用户体验

**I4. +goose Down 完整** — 正确删除详情和字典，顺序正确（先子后父）

---

## 审查清单

- [x] SQL 无字符串拼接
- [x] 无硬编码密钥
- [x] TypeScript 编译通过
- [x] 无 `any` 类型
- [x] Hook 模式一致 (useQuery + enabled)
- [x] 死代码已清理 (PRODUCT_TYPE_OPTIONS)
- [x] 迁移幂等 (ON CONFLICT DO NOTHING)
