# 快速设置带宽与设备终态刷新审查

## 结论

PASS，无 CRITICAL 或 WARNING 发现，可以提交并进入 GitLab MR 审查。

## 审查范围

- BLN、MLN、BLQ、MLQ、BM 及默认 LTE 型号快速设置的上下行带宽展示。
- LTE 上下行带宽独立输入、TDD 一致性校验及即插即用参数镜像编译。
- 快速设置在设备任务成功、失败、超时或取消终态后的 schema、参数列表、搜索结果和参数树刷新。
- 设备详情与即插即用页面共用的 5G NR 带宽选项，移除设备不支持的 5/15/25MHz。

## 关键判断

1. 产品参数映射继续作为设备 wire value 的唯一来源；参数编译器只根据 `mirrorWith` 补齐对端路径，不改写不同型号的枚举编码。
2. 页面同时展示上下行带宽，由用户分别配置；TDD 场景在提交前校验两者相等，避免静默覆盖用户输入。
3. 设备任务进入任一终态后统一失效快速设置关联查询；失败终态立即回读设备值并清理本次失败草稿，避免页面继续显示未生效值。
4. NR 带宽选项已提取为设备页面共享定义，设备详情和即插即用不再分别维护，降低配置再次漂移的风险。

## 安全与兼容性

- 未新增接口、数据库迁移、鉴权绕过、SQL 拼接、敏感信息或公共 `any` 类型。
- 保留 BLN/BLQ/BM/MLN/MLQ 与 ENB_DEFAULT 型号各自的 LTE 带宽 wire value；无存量数据迁移。
- webcode-v2/webcode-v3 中没有对应快速设置页面，本次页面级改动仅适用于当前 webcode；共享 i18n 位于 frontend-core。

## 验证证据

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：通过，包含 E2E 与 integration 包。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb && npm test --workspace webcode -- --run src/pages/device/DeviceDetail/QuickSettingsTab/__tests__ src/pages/device/PlugAndPlay/gnbQuickSettingsFields.test.ts`：20 个测试文件、105 条用例通过。
- `git diff --cached --check`：通过。
- 本地 Docker Compose 重新构建并热替换 web/app/acs，`http://localhost:8081/` 返回 200。
- 浏览器实测 30kHz 上下行带宽均只显示 10–100MHz，5/15/25MHz 不再出现；设备失败终态页面回读行为已由组件测试覆盖。

## 风险与回滚

- 主要风险是设备型号枚举编码或 `mirrorWith` 配置不完整；型号级 XML 测试、编译器镜像测试和快速设置加载测试已覆盖。
- 回滚本提交即可恢复原行为，不涉及数据库或设备数据迁移。
