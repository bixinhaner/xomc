# Code Review — line 51 通知中心 DeleteObject 参数信息补全

| 字段 | 值 |
|------|------|
| 时间 | 2026-05-20 |
| 范围 | `omcgo/internal/notification/task_subscriber.go` + `task_subscriber_test.go` |
| 类型 | bugfix (用户反馈：消息中心删除对象通知未显示被删 path) |
| 审查者 | Claude (夜间无人监督自治) |
| 关联 | TODO.MD line 51 |
| 真机验证 | BLQ `1202000240194DP0026` InterFreq.Carrier.6 删除 → 标题 "Carrier.6" + 完整 path 显示 |

## 变更摘要

`renderNotifTitle` / `renderNotifContent` 加 `DeleteObject` 分支，新增 `extractObjectName` + `shortObjectName` 工具函数。

- 标题尾巴显示对象短名（末 2 段，如 `LTECell.14`），与已有 SetParameterValues 的"N 项"模式对称
- 内容显示完整 standardPath，便于运维定位

## 审查清单（Go 后端）

- [x] 命名规范：Go 标准 camelCase / PascalCase
- [x] 错误处理：`encoding/json` Unmarshal 失败返回空字符串（容错降级，不影响主流程）
- [x] 无 SQL 改动
- [x] 无 carrier 硬编码
- [x] 无 panic 风险
- [x] 无资源泄漏（纯字符串处理）
- [x] 测试覆盖：`Test_TaskSubscriber_DeleteObject_ShowsObjectPath`（端到端 task → notification）+ `Test_ShortObjectName`（工具函数 6 种边界）
- [x] 兼容性：仅扩展 switch case，不破坏现有 SetParameterValues / Reboot / FactoryReset 等分支

## 设计决策

| 决策 | 选项 | 理由 |
|------|------|------|
| 标题对象短名截取段数 | 末 2 段（`X.{i}`） | 既能识别"哪个对象的哪个实例"，又不会过长污染标题 |
| 内容是否含完整 path | 是 | 用户明确反馈"看不到具体修改信息"，完整 path 是核心需求 |
| AddObject 是否顺手补 | 否 | 守"夜间不顺手扩大范围"规则，TODO 已加备注待用户确认 |

## 结论

**PASS** — 0 CRITICAL / 0 WARNING / 0 INFO。改动局部、有测试、真机验证通过、风险极低。
