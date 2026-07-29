# Issue #213 多 PLMN 新增与按钮样式修复计划

**目标：** 修复 452/MLN 基站快捷设置中服务 PLMN 点击“新增一行”后立即消失的问题，并统一服务 PLMN 与 MME IP+PLMN 的新增按钮样式。

**根因：** PLMN 表格在 `onValuesChange` 时过早序列化，空白新增行被过滤；随后草稿回填把表格恢复成新增前状态。MME 表格保存完整行数组，因此没有同类问题。两个表格还分别声明了不同的 Ant Design Button 属性。

## 实施步骤

1. 为 PLMN 草稿行归一化、空白行保留和行数上限补回归测试。
2. 将 PLMN 草稿保存为完整行数组，只在保存设备参数时序列化。
3. 提取两个表格共用的虚线通栏新增按钮，并补 DOM 行为测试。
4. 运行定向测试、Quick Settings 测试集和 `npm run typecheck`。
5. 在独立分支 `codex/issue-213-plmn-add` 交付，不合并、不推送，不影响 IPsec 分支。
