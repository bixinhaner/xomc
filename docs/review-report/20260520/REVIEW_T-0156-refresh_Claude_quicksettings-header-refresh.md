# 代码审查报告 — T-0156 followup: 头部"刷新"按钮接入快速设置

| 字段 | 值 |
|------|------|
| 任务 | T-0156（umbrella）— DeviceDetail 头部刷新按钮按 activeTab 失效对应资源 |
| 范围 | frontend (store + DeviceDetail page) |
| 改动量 | 3 文件 |
| 审查结论 | **PASS** |

---

## 1. 变更概览

之前 `DeviceDetail/index.tsx::handleHeaderRefresh` 只覆盖 `parameters` / `alarms` tab，`quickSettings` tab 缺 case → 点刷新只 refetch 设备信息但不刷新快速设置 group / schema，UI 上"刷新无效"。

本次：
- **store 加 `refreshTicks: Record<string, number>` 状态 + `bumpRefreshTick(deviceId)` action**
- **handleHeaderRefresh 加 case 'quickSettings'**：
  - `clearByDevice(deviceId)` — 清 drafts + 反馈 Tags（方案 B：刷新=重置到服务器状态）
  - `bumpRefreshTick(deviceId)` — 触发子组件 remount
  - invalidate `['quicksettings','groups',deviceId]` + `['devices','parameter-schema',deviceId]`
- **QuickSettingsTab/index.tsx**: 订阅 `refreshTicks[deviceId]`，拼进子组件 `key={'${group.id}::${refreshTick}'}` → tick 变化触发 CellParameterForm / MultiInstanceTable 整体 remount，组件内 form.touched / rowEdits state 全归零

## 2. 设计决策

| 决策 | 理由 |
|------|------|
| 方案 B（刷新清 drafts） | 通信网管"刷新"工业惯例 = 拉服务器最新 + 丢前端临时状态（类似浏览器 F5）；用户主动点说明想看现状不该被自己 draft 遮住 |
| 用 refreshTick 当 key 强制 remount | 比手动遍历每个子组件调 form.resetFields / setRowEdits(new Map) 简单可靠；React 处理所有清理 |
| 不加 confirm dialog | 与同 tab 的 parameters / alarms 分支对称，那两者本来就直接 invalidate 不弹确认 |
| 不动 clearByDevice 现有语义 | 已有"卸载时全清"用法不受影响，drafts + entries 同步清是符合刷新语义 |

## 3. 审查检查项

| 项 | 结果 | 备注 |
|----|------|------|
| 前端 typecheck | ✅ | `npm run typecheck` PASS |
| 类型安全 | ✅ | refreshTicks 精确 `Record<string, number>`，bumpRefreshTick 签名清晰 |
| sessionStorage 兼容 | ✅ | 新增字段为可选，旧版 storage 反序列化时 refreshTicks 自动取默认 `{}` |
| 无内存泄漏 | ✅ | refreshTicks 仅按 deviceId 累加；离线 device 残留无副作用 |
| 子组件 unmount 副作用 | ✅ | CellParameterForm/MultiInstanceTable 内部无 setInterval / 长连接，remount 安全 |

## 4. 部署验证（已完成）

| 测试 | 结果 |
|------|------|
| 改 MultiInstance PMax 3→9，drafts 写入 `{"2.PMax":"9"}` | ✅ |
| 点头部"刷新"按钮 | drafts 清空 `{}`，refreshTicks 变为 `{deviceId: 1}` ✅ |
| 刷新后 PMax 回到 schema 服务器值 **3** | ✅（remount 生效）|

## 5. Out of scope

- `parameters` / `alarms` 既有分支不动
- 不加二次确认弹窗（如未来用户反馈误点丢编辑，可加 confirm）
- license 头部按钮已隐藏（line 692 注释），不动

---

**审查人**：Claude Opus 4.7  
**日期**：2026-05-20  
**结论**：PASS
