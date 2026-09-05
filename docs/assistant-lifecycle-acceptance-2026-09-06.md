# 独立助手生命周期部署与联调验收（2026-09-06）

## 部署范围

按用户确认，xOMC 本地环境是用户本机 Mac；本次没有部署到 172.17.3.104。

- Agent Studio：PR #1 合入 main，生产版本 `74017cf8c871e380f7a35395b0119bfb900d035f`。标准发布脚本完成，admin/chat/public 三个健康入口均通过，助手租约迁移已落库。
- xOMC：PR #1 合入 `codex/active-intelligence`；功能代码 `cc7c82cd7ed906d7f2f2365eaaec56b15adedb40` 已构建并运行于本机 `omc-agent-test`。页面 `http://127.0.0.1:3000/system/active-intelligence`，后端 `http://127.0.0.1:18081`。主分支集成由 PR #2 留档。
- 两端发布前均完成数据库备份并验证备份目录可读取；保留既有本机设备、用户、配置及无关工作区改动。凭证未进入代码、文档或提交。

## 本轮修复

1. 新助手菜单和三个内置角色授权进入 MainReconcile，已有数据库升级可以补齐，无需重建数据库。
2. 发布版本冻结范围摘要，发布时核对成功试运行的范围；后续定时任务不能静默扩大授权。旧版本缺少摘要时失败关闭。
3. 取消后读取远端真实终态；远端已完成或失败的任务不会被伪造为 CANCELLED。旧租约结果不能覆盖新取消状态。
4. 所有主动智能 worker 和事件回调在路由注册、手册准备及 API 权限同步之后启动，消除 75/909 接口数量不一致的初始化竞态。重启后手册记录 `total_operations=909`，catalog `94be0a4dc5d3b6a7`。

## 真实环境证据

| 场景 | 证据 | 结果 |
| --- | --- | --- |
| 自然语言规划 | 私人助手 `db4894e2-5dd9-4ae6-a058-6a4e0afa498d`，草稿 v2 | 真实生产规划器生成每 5 分钟汇总设备数量的方案；不支持的在线状态未被编造 |
| 真实试运行 | Run `83c9d44a-f511-4d7d-9b38-35b57cb14a74` | 两端 COMPLETED，`get.devices` SUCCEEDED，事实引用 `tool:get.devices`，设备数量 1，页面呈现一致 |
| 发布门槛 | 草稿 v2 试运行完成后通过页面确认 | 发布 revision=2，服务器保存下次运行时间 |
| 私有隔离 | 另一个真实登录用户读取上述助手 | HTTP 404 |
| 取消 | Run `94a97ffb-8ed7-4c9a-8661-1b0c673343d4` | 从 CANCELLING 收敛，两端 CANCELLED，CANCELLED_BY_USER |
| 权限撤销 | Run `17c2149e-cebc-4bec-8ed6-c48983d06584` 入队后移除临时用户角色 | 本机 FAILED，ASSISTANT_ROLE_UNAVAILABLE，未继续业务查询 |
| 重启恢复 | Run `09292f6e-9e4a-41d5-a2d6-c88eef139483` RUNNING 时重启本机 app | 同一 Run ID 恢复并 COMPLETED，生产 runAttempt=1，`get.devices` SUCCEEDED，结果数量 1 |
| 关闭页面后的定时运行 | Run `b90091d7-b055-4930-ab77-568660b2b76d`，07:09 自动触发 | 两端 COMPLETED，生产 runAttempt=1，`get.devices` SUCCEEDED；重新打开页面显示 1 台，私人通知标记已落库 |
| 用量 | 首条试运行对应生产 usage_events.metadata.runId | success，inputTokens=191568，outputTokens=1375；不是未计量的旁路调用 |

模型用量是 runtime 累计口径，包含手册与上下文，并非只有用户消息的字数。此版本已验证功能闭环；大规模高频启用前仍需评估实际成本，不能把一次联调视作容量或成本验收。

## 自动化验证

- Agent Studio：API 构建、17 个文件 59 个测试（包含 5 个真实 PostgreSQL 集成测试）、新库 75 个 migration 均通过。
- xOMC：assistant/bridge/runtime/attention/provider 测试通过；真实 PostgreSQL 发布范围、取消终态以及基线重复协调回归通过。
- 前端：类型检查、10 个相关组件测试、构建通过。
- GitHub PR #2：手册契约、迁移严格 lint、真实新库 schema+seed 检查通过。

## 验收边界与已发现的仓库问题

不能宣称全量 CI 已通过：

- 既有 `cmd/migrate` 的 `TestParamModelContentHashIsFoldedIntoPreReleaseBaseline` 禁止兼容 ALTER，但原基线本身已包含该语句；本轮新增及真实升级回归通过。
- 既有 GitHub 前端工作流仍缓存不存在的 `omcmb/webcode/package-lock.json`，在 setup-node 阶段失败；真实锁文件位于 `omcmb/package-lock.json`，本机 workspace 类型检查和构建通过。
- gosec 固定旧工具版本，与当前 Go 工具链不兼容，构建报 invalid array length。
- govulncheck 和 Trivy 报告既有标准库及依赖漏洞，包括 excelize、x/crypto、grpc 等；本机 npm audit 也报告既有 axios、react-router、xlsx 等问题。此轮未通过禁用检查、忽略漏洞或广泛依赖升级掩盖这些问题。

本记录覆盖助手功能发布及本机联调，不代表 xOMC 通过面向公网生产的全量安全放行，也不代表长期稳定性、容量、邮件投递、团队共享或设备写操作已验收。

## 验收结束状态

测试助手已暂停，next_run_at 为空；两个临时测试账号已禁用并移除角色。历史结果和备份保留。本机 QUEUED/RUNNING/CANCELLING 助手任务数量为 0。
