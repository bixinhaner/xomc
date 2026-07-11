# REVIEW 7b43279 codex quicksettings

## 结论

PASS

本次审查未发现 CRITICAL 阻塞问题。

## 范围

- `omcgo/data/param-mappings/BaiBNQ.xml`
- `omcgo/data/quicksettings/BaiBNQ.xml`
- `omcgo/internal/config/parammodel/model_test.go`
- `omcgo/internal/device/device_param_handler.go`
- `omcgo/internal/device/device_param_handler_test.go`
- `omcmb/frontend-core/src/i18n/en-US/index.ts`
- `omcmb/frontend-core/src/i18n/zh-CN/index.ts`
- `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/MultiInstanceTable.tsx`

## 发现

### WARNING

无。

### INFO

- BaiBNQ LTE 异系统重选列表从 ConnMode 路径切到 IdleMode 路径，并新增范围约束。该行为依赖设备侧实际支持 `NR.RAN.Mobility.IdleMode.EUTRA.Carrier` AddObject/SPV；已通过 XML/handler 单元测试覆盖 schema 与约束，但仍建议在真实设备上继续观察 AddObject 后实例号和回读路径一致性。
- 前端新增阶段会在 AddObject 入队后立即关闭弹窗并展示中间状态，后续 SPV 仍在后台串行等待实例号。该体验改善不改变 TR-069 下发窗口，最终回显仍受设备 Inform/Connection Request 时机影响。

## 验证

- `cd omcmb && npm run typecheck` — 通过，含 skin-parity、webcode、webcode-v2、webcode-v3。
- `cd omcgo && go build ./...` — 通过。
- `cd omcgo && go test ./internal/device ./internal/config/parammodel` — 通过。
- `cd omcgo && go test ./...` — 通过。

## 审查说明

- 后端 schema merge 改动未引入 SQL 拼接或认证面变化。
- 新增路径实例匹配逻辑限定 `{i}` 数量和数字实例段，未放宽到任意路径。
- 前端状态机保留失败通知、自动回滚和草稿清理路径；AddObject 中间态不会触发成功回读。
