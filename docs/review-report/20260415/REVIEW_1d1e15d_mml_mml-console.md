# Code Review: MML 控制台前端 API 对接与 UI 优化

**日期**: 2026-04-15
**审查范围**: omcmb/webcode — MML Console 组件、hooks、mock 数据、API 层
**审查结论**: PASS_WITH_WARNINGS

---

## 审查文件

| # | 文件 | 行数 | 变更类型 |
|---|------|------|---------|
| 1 | `webcode/src/pages/mml/Console/components/CommandInput.tsx` | 563 | 修改 — 危险命令检查改为 API |
| 2 | `webcode/src/pages/mml/Console/components/CommandTree.tsx` | 1 | 修改 — placeholder 文本 |
| 3 | `webcode/src/pages/mml/Console/components/DeviceTree.tsx` | 2 | 修改 — placeholder 文本 |
| 4 | `webcode/src/pages/mml/Console/constants.ts` | -162 | 修改 — 移除硬编码 mock 数据 |
| 5 | `webcode/src/pages/mml/Console/hooks/useCommandExecution.ts` | 238 | 修改 — 异步任务轮询 |
| 6 | `webcode/src/pages/mml/Console/hooks/useDeviceSelection.ts` | 148 | 修改 — 服务端产品类型过滤 |
| 7 | `webcode/src/mock/data/mml.ts` | ~200 | 修改 — mock 命令目录扩展至 25 条 |
| 8 | `webcode/src/mock/services/mmlService.ts` | ~28 | 修改 — 新命令 mock 输出 |
| 9 | `webcode/src/services/api/deviceApi.ts` | 2 | 修改 — productType 查询参数映射 |

---

## 发现

### WARNING

**W1. checkDangerous 响应使用 PascalCase 键名**
- 文件: `mmlApi.ts`, `CommandInput.tsx`
- 返回类型 `{ dangerous: boolean; info: { Name, Desc } }` 使用 PascalCase，与项目其他 BackendXxx 接口不一致
- 需确认后端实际返回格式，否则 Name/Desc 可能为 undefined

**W2. 设备获取使用硬编码 pageSize: 1000**
- 文件: `useDeviceSelection.ts:19`
- 预存在问题，100K+ 设备规模下不可行
- 当前阶段可接受，后续需改为分页或搜索模式

**W3. placeholder 从 i18n key 改为硬编码中文**
- 文件: `DeviceTree.tsx`, `CommandTree.tsx`
- `t('common.search')` → `"设备SN,设备名称"` 丢失了国际化支持

### INFO

**I1. 死代码清理正确** — 162 行硬编码 DEVICE_LIST/MOCK_COMMANDS 已移除

**I2. 任务轮询架构良好** — useMMLTaskPolling 使用 refetchInterval + terminal state 停止

**I3. 服务端过滤优化正确** — productType 通过 API 参数传递，减少传输量

**I4. Mock 数据与 DB 种子保持一致** — 25 条命令覆盖 7 个标准分类

---

## 审查清单

- [x] TypeScript 编译通过
- [x] 无 `any` 类型
- [x] 无硬编码密钥
- [x] 无 XSS 风险
- [x] API 服务模式一致
- [x] Hook 模式一致（useQuery/useMutation）
- [x] 死代码已清理
- [ ] i18n 回归（placeholder 硬编码中文）
