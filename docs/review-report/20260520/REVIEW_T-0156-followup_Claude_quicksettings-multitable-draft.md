# 代码审查报告 — T-0156 followup: 列表表 draft 持久化 + 异频载波列表改名

| 字段 | 值 |
|------|------|
| 任务 | T-0156（umbrella）— MultiInstanceTable 部分 + 异频载波列表标签改名 |
| 范围 | frontend (store + quicksettings page) + omcgo/data (quicksettings XML) |
| 改动量 | 6 文件（1 store / 1 component / 4 XML）|
| 审查结论 | **PASS** |

---

## 1. 变更概览

### 1.1 列表表 draft 持久化（MultiInstanceTable）

T-0156 之前已把 CellParameterForm 的 draft 提升到 zustand `quickSettingsFeedbackStore`，但 `MultiInstanceTable`（异频载波列表 / 异频邻区列表等"列表类参数"）的 `rowEdits` 仍在组件 `useState`，DeviceDetail 跨顶层 TabBar 卸载时丢失。本次同模式扩展到 list table：

- **store**: 新增 `clearDraftPrefix(key, prefix)` 方法，按前缀清理 drafts（单行 save/delete 后只清该行，保留其他行未保存编辑）
- **MultiInstanceTable**:
  - 订阅 `drafts[fbKey]`
  - 每个 cell `setCellValue` 同步写 `setDraftField(fbKey, '${instId}.${leaf}', value)`
  - useEffect 在 schema ready 时把 draft 拆 instId/leaf 还原到 rowEdits（仅在 rowEdits 空时填，避免覆盖会话内编辑；实例已被删除则丢弃）
  - `handleSaveRow` 成功后 + `handleDelete` 入队后清该行 draft

### 1.2 异频载波列表改名

`<group id="enb-neighbor-freq">` 在 BLQ/BM/MLN/MLQ 四个 paramModel quicksettings XML 中的 titleZh/titleEn：

| 旧值 | 新值 |
|------|------|
| 异频邻区频点列表 | 异频载波列表 |
| Inter-Frequency Neighbor Carrier List | Inter-Frequency Carrier List |

group `id="enb-neighbor-freq"` 不变，不影响后端 fbKey / 前端路由。

## 2. 设计决策

| 决策 | 理由 |
|------|------|
| draft 名格式 `${instId}.${leaf}` | 单一 fbKey 命名空间内打平存储；leaf 不含点，解析无歧义 |
| 仅在 `rowEdits.size === 0` 时还原 draft | 避免覆盖用户当前会话已编辑的值（用户在某行编辑时另一行的 draft 不该回写） |
| 删除实例时同步清 draft + rowEdits | 实例不存在后保留 draft 是僵尸数据，重挂载会尝试还原已删实例 |
| 改名"异频邻区频点"→"异频载波" | TR-069 `IdleMode.InterFreq.Carrier` 对应 3GPP 36.331 InterFreqCarrierFreqList，行业术语为"异频载波"（华为/中兴/大唐 OMC 通用）；"邻区"语义偏离（邻区指特定 cell，列表项实际是 frequency）|

## 3. 审查检查项

| 项 | 结果 | 备注 |
|----|------|------|
| 前端 typecheck | ✅ | `npm run typecheck` PASS |
| XML 良构 | ✅ | 4 个 quicksettings XML `xmllint --noout` 通过 |
| 类型安全 | ✅ | RowEditState/draft 全精确类型，无 `any` |
| 无 XSS | ✅ | 数据皆来自 schema/store，无 dangerouslySetInnerHTML |
| 无内存泄漏 | ✅ | zustand store 自动清理，draft 通过 clearDraftPrefix 主动清 |
| 无运营商硬编码 | ✅ | 通用机制 |
| 兼容旧 sessionStorage | ✅ | `drafts: Record<string, Record<string, string>>` 形态未变 |
| group id 稳定 | ✅ | 改名只改 titleZh/titleEn，id 不变 |

## 4. 部署验证（已完成）

| 测试 | 结果 |
|------|------|
| 异频载波列表第 1 行 PMax 改 3→5 | sessionStorage drafts 立即写入 `{"2.PMax": "5"}` ✅ |
| 切到"详情"tab → 切回"快速设置" | PMax 仍显示 5（不再回退到 schema 原值 3）✅ |

## 5. Out of scope

- CellParameterForm 已有 draft 持久化（T-0156 主体），本次未动
- group id `enb-neighbor-freq` 不重命名（保留向后兼容；如未来要改 group id，需同步 fbKey 计算 + 历史 sessionStorage 数据迁移）
- 后端 `internal/quicksettings/` 模块不动（XML 由 Go loader 读取，无 hardcoded 字符串引用）

---

**审查人**：Claude Opus 4.7  
**日期**：2026-05-20  
**结论**：PASS
