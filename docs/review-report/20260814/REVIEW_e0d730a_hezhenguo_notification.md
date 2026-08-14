# Issue #283 告警与 KPI 邮件通知审查报告

- 日期：2026-08-14
- 基线：`origin/main@5232194bb`
- 审查提交：`e0d730a59`
- 分支：`codex/email-sms-notification-implementation`
- 范围：告警邮件、KPI 周期报表邮件、统一 SMTP 传输、通知历史、V1 前端与 `000001` 基线
- 结论：`PASS_WITH_WARNINGS`

## 结论

未发现 CRITICAL 问题。实现符合最终确认范围，可以推送 feature 分支并创建 Draft MR 供维护者审阅；在真实阿里企业邮箱验收和主线全量门禁恢复前，不应合入或开启生产 SMTP。

## 审查范围

- 告警规则 `notify_email` 的收件人、有效期、产生/清除生命周期和处理建议。
- app 入队、worker 异步执行、逐收件人投递、幂等抢占、失败重试与 dead-letter。
- KPI Query 私有模板的 15 分钟/小时/天自然窗口、CSV 导出、MinIO 附件读取和停机恢复。
- SMTP implicit TLS / STARTTLS、MIME 附件、大小限制、Header 注入与日志脱敏。
- 查询模板所有权、JWT/RBAC、通知历史超级管理员边界和敏感字段裁剪。
- PostgreSQL schema/seed 折回 `000001` 基线及前端 V1 i18n/权限交互。
- 运维手册、范围说明、开发方案和回退步骤。

## 重点检查

- **SQL 与事务**：新查询使用 Squirrel + pgx；运行状态与订阅状态在同一事务更新，没有字符串拼接外部输入或 ORM。
- **权限与数据暴露**：API 位于现有 JWT/feature group 下；订阅写操作限定超级管理员或私有模板所有者；通知历史仅超级管理员可读，普通读者的收件人与错误详情会被裁剪。
- **凭据与传输**：SMTP 密码仅从配置/环境变量注入，不进入 Git、邮件历史、错误正文或结构化日志；TLS 最低版本为 1.2，STARTTLS 不可用时拒绝降级。
- **输入安全**：邮箱地址归一化并限制最多 50 个；主题拒绝 CR/LF；附件文件名拒绝路径和 Header 注入；附件按声明大小和实际流量双重限流。
- **可靠性**：告警接收链路只冻结快照并入队，SMTP 失败不阻塞告警入库；逐地址幂等键、原子接管 stale pending、成功地址跳过和最终 dead-letter 均有测试。SMTP DATA 成功后的 QUIT 失败不触发重发。
- **KPI 口径**：调度按系统 PM 时区取最近已闭合自然窗口，恢复时折叠过期窗口，避免停机后的邮件风暴；CSV 使用既有导出与对象存储链路。
- **迁移边界**：只修改主库和 seed 的 `000001`，未新增 `000002+`；迁移检查通过。
- **前端**：只维护 V1；新增文案已进入中英文 catalog；告警规则和 KPI 模板操作使用既有权限点，删除两处不可达的旧通知设置页面。
- **范围控制**：未实现 Office365/OAuth、短信、FTP、Webhook 改造、老 OMC 全局邮件开关、默认收件人叠加或 Redmine #98018 汇总邮件。

## 验证结果

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go vet ./...`：通过。
- `cd omcgo && go test ./... -count=1`：除主线既有 `internal/product` BLN XML 元数据计数外，其余包（含 E2E/集成）通过；失败测试和 `BLN.xml` 与 `origin/main` 无差异。
- `cd omcgo && go test -race -count=1 ./internal/notification ./internal/pm/reportsubscription`：通过。
- 本次告警邮件与筛选相关测试的 `-race` 聚焦运行：通过。
- `cd omcgo && bash scripts/check-migrations.sh`：通过；脚本仅提示仓库约定的三个 `000001` 基线同号现状。
- `cd omcmb/webcode && npm run typecheck`：通过。
- `cd omcmb/webcode && npm run lint`：退出码 0，无 error；主线现存 3604 条 warning。
- `cd omcmb/webcode && npm run build`：通过；存在 Vite 配置弃用和大 chunk 警告。
- 前端通知/告警/KPI 聚焦 Vitest：2 个文件、26 个测试通过。
- 全量 Vitest：302/304 个文件、2047/2049 个测试通过；IPSec 快速设置和 KPI Query 各一项并发运行超时，KPI Query 单文件复跑全绿。
- 本地模拟 SMTP：告警产生/去重/清除得到 2 封邮件、1 条历史、0 条活动告警；KPI 15 分钟自然窗口生成 66 字节 CSV 并随邮件投递。
- 本地 Docker app/worker/web 构建、启动和健康检查：通过；测试数据与 SMTP 开关已清理并恢复为禁用。
- `omcgo/test/e2e` 与 `omcgo/test/integration` 全量通过，但新增 KPI 订阅 REST 路由尚未加入 `scripts/e2e_verify.sh` 的专用端点 claim。

## WARNING / 未完成项

1. **真实阿里企业邮箱未验**：当前拿不到账号、授权码、白名单收件人和生产出站网络。合入/部署前保持 `notification.smtp.enabled=false`，取得凭据后按 `docs/runbook/email-notification.md` 验收认证、TLS、发件人和真实收件。
2. **主线全量 Go 测试非全绿**：`internal/product` 的 BLN XML 元数据计数期望 380、实际 329；本分支未修改相关文件，应由主线责任项修复。
3. **全量前端测试存在并发超时**：两项失败均为超时；本功能涉及的 KPI Query 单文件和告警聚焦测试复跑通过。MR 中不得把全量 Vitest 勾成全绿。
4. **全包 alarm race 有历史测试夹具竞态**：`expedited_integration_test` 异步写 mock map 时测试线程直接读取；本功能聚焦 race 通过，不在本次需求中扩修历史夹具。
5. **本机缺少 golangci-lint**：已执行 `go vet ./...`，但 DoD 中 `golangci-lint run` 需由 CI 或安装了标准工具链的维护者补验。
6. **新增 REST 端点缺专用 E2E claim**：服务授权、参数校验和业务链路已有单元/集成验证，但 `/pm/query-templates/:id/report-subscription` 系列路由尚未纳入 `scripts/e2e_verify.sh`；Draft MR 转 Ready 前需补齐或由维护者明确接受替代证据。

## 回退与发布建议

- 业务回退先禁用 KPI 订阅和告警邮件规则，再将 app/worker 的 `notification.smtp.enabled` 置为 `false`；不影响告警入库、PM 采集或 KPI 查询。
- 当前软件未封版，数据库变化通过重建 `000001` 基线回退，不新增补丁迁移。
- 本次应创建 Draft MR；真实 SMTP、全量门禁和维护者审阅完成后再转 Ready。
