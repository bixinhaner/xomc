# Review — 即插即用任务筛选与配置展示调整

## 结论

PASS。未发现 CRITICAL 或 WARNING 问题。

## 审查范围

- 即插即用与快速设置移除“参考子载波间隔”字段及无用中英文文案。
- 执行状态模块按钮尺寸与筛选控件对齐。
- 执行状态工具栏复用设备列表的“刷新 + 自动刷新”交互。
- 时间范围按系统时区钟面序列化为 RFC3339，避免浏览器时区造成筛选偏移。
- 新增 UTC、Asia/Shanghai 与空时间范围回归测试。

## 重点检查

- 类型安全：时间范围保持 `[Dayjs, Dayjs] | null`，序列化结果显式为可选字符串。
- Hook 行为：自动刷新默认关闭，间隔默认 30 秒；启用或切换间隔时立即刷新一次。
- API 兼容：继续使用既有 `started_after` / `started_before` 查询契约，无接口或数据结构变更。
- 时区一致性：DatePicker 值作为系统时区钟面处理，与列表的系统时间展示规则一致。
- 配置一致性：即插即用字段定义、快速设置 XML、i18n 与字段顺序测试同步删除。
- 安全性：无 HTML 注入、鉴权、Token、SQL 或敏感信息变更。

## 验证

- `cd omcmb && npm run typecheck` — PASS。
- `npm run test -- src/pages/device/PlugAndPlay/taskTimeFilter.test.ts src/pages/device/PlugAndPlay/gnbQuickSettingsFields.test.ts` — 2 files / 5 tests PASS。
- `npm run test -- ../frontend-core/src/services/api/__tests__/provisionApi.test.ts` — 7 tests PASS。
- `go test ./internal/quicksettings ./internal/provision -run 'TestBuildProvisioningTaskListSQL|TestHandlerList|Test.*Quick' -count=1` — PASS。
- `npm run build` — PASS。
- 本地 Docker Compose 热部署 — PASS；Web/App/ACS 为 Up，`http://localhost:8081/` 返回 200。

## 风险与影响

- 影响范围限于主皮肤即插即用页面、共享中英文文案及 BaiBNQ 快速设置 XML。
- 无数据库迁移，无后端接口变更，无兼容性风险。
