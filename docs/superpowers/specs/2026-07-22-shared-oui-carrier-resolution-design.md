# 共享 OUI 运营商识别修复设计

## 问题

提交 `640369ef7` 为消除 Go map 遍历导致的随机运营商归属，改为按适配器注册顺序解析 OUI。app 和 worker 固定按 CMCC、CTCC、CUCC 注册；同时华为 `00E0FC`、中兴 `001E7E`、京信 `58FB96` 同时存在于 CMCC 与 CTCC 适配器。因此只按 OUI 取首项会把这些共享 OUI 稳定归入 CMCC。

## 业务规则

1. 新设备 Inform 同时具有 OUI 和 ProductClass，应按二者组合匹配运营商。
2. 组合身份只能在唯一匹配时自动归属。
3. OUI 只属于一个运营商时，允许保持现有 OUI 自动识别兼容性。
4. OUI 被多个运营商共享且 ProductClass 缺失、未知或仍然多义时，返回歧义错误，禁止回退到默认 CMCC。
5. CSV 预登记中显式 carrier 优先；未填写 carrier 且 OUI 多义时，要求用户填写 carrier。
6. 已存在设备以数据库中的 carrier 为准，不因本修复自动迁移或改写。
7. 完全未知 OUI 继续沿用现有部署默认 carrier，避免改变单运营商现场的历史接入策略。

## 实现边界

- 在 `internal/core/carrier` 增加设备身份解析结果，区分唯一匹配、未匹配和歧义。
- Inform 的四条自动注册入口传入 OUI 与 ProductClass，并在歧义时停止注册。
- 批量预登记保留显式 carrier 优先；只按 OUI 推断时仅接受唯一运营商。
- 不改变 carrier 适配器的 OUI/ProductClass 数据，不修改设备表，不增加迁移。
- 不涉及 PM 保留配置；该问题单独处理。

## 验证

- 共享 OUI + CMCC ProductClass 唯一解析为 CMCC。
- 共享 OUI + CTCC ProductClass 唯一解析为 CTCC。
- 共享 OUI 缺少或无法匹配 ProductClass 时返回歧义，不回退 CMCC。
- 单运营商独占 OUI 保持自动识别。
- 未知 OUI 保持历史默认 carrier 回退。
- 显式 carrier 的批量预登记不受影响。
- 运行 carrier、device 聚焦测试以及后端全量测试和编译。
