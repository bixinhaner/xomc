# Review: PlugAndPlay 参数配置映射与只读详情

- 结论: PASS
- 作者: wangyong
- 基线: 9b9a1c36f
- 范围: `omcmb/webcode/src/pages/device/PlugAndPlay`
- 日期: 2026-08-20

## 变更概览

- 将导入工作簿中的字段映射反向应用到参数详情表单，覆盖 5G DEVICE/CELL/PLMN/1588/IPSEC、4G CELL/NETWORK 等常见别名。
- 参数详情抽屉从禁用态调整为只读态，并在有权限时提供显式编辑入口。
- 网络快捷参数按 TR path 保存多实例值，避免同名字段在不同实例或子对象之间互相覆盖。
- 增加 PlugAndPlay 参数导入、映射、只读展示和多实例网络参数回归测试。

## 审查结果

### CRITICAL

- 无。

### WARNING

- 无。

### INFO

- 本次改动范围集中在前端 PlugAndPlay 参数配置逻辑，无后端接口、数据库迁移或鉴权链路变更。
- 当前环境未关联 GitHub Issue；提交 footer 将使用 `Issue: N/A`。

## 审查要点

- 类型安全: 新增映射 helper 对 `unknown` 值做空值和字符串归一化处理，未引入 `any` 扩散。
- 表单行为: 只读模式使用 `readOnly` 和隐藏增删按钮，保留字段展示和编辑入口，不再依赖整表单 disabled。
- 数据一致性: 保存时优先写回原始 workbook mapping 列，缺失 mapping 时才回退到模板默认列或现有别名列。
- 多实例网络参数: multi-instance 字段使用绝对 TR path 的 `networkParameterValues`，并从现有 TR path 推断实例数量。
- 安全性: 未新增 HTML 注入、token 处理、动态 SQL、后端调用或敏感配置。

## 验证

- `cd omcmb/webcode && npm run test -- src/pages/device/PlugAndPlay/paramConfigDetail.test.ts src/pages/device/PlugAndPlay/paramConfigWorkbook.test.ts src/pages/device/PlugAndPlay/specifiedParamConfigEditor.test.ts src/pages/device/PlugAndPlay/CommonParameterConfigPanel.test.tsx src/pages/device/PlugAndPlay/CommonQuickSettingsNetworkCards.test.tsx` - 通过，5 个文件 88 个用例。
- `cd omcmb && npm run typecheck` - 通过。
- `git diff --check` - 通过。
