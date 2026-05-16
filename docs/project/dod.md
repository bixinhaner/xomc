# Definition of Done（完成定义）

> **性质**：硬约束。PR 合入前必须每项打勾（N/A 需注明）。  
> **源头**：`CLAUDE.md §10 质量关卡` + `docs/project/process-design-20260420.md §5.4`  
> **守护人**：QA/发布经理（`CLAUDE.md §16.12`）

---

## 通用 DoD（所有 PR 必过）

### 编译与类型
- [ ] 后端 `go build ./...` 通过
- [ ] 后端 `go test ./...` 全绿
- [ ] 后端 `golangci-lint run` 无新增告警
- [ ] 前端（如涉及）`cd omcmb/webcode && npx tsc --noEmit` 通过
- [ ] 前端（如涉及）`npm run lint` 无新增告警

### 测试
- [ ] 新增/修改的代码包含对应单元测试
- [ ] 测试覆盖**成功路径 + 失败路径两条**（不止 happy path）
- [ ] 新 REST 端点：E2E 脚本（`scripts/e2e_verify.sh`）已增补用例
- [ ] 修复 bug：回归测试（先写失败测试 → 改代码让它过）
- [ ] 测试覆盖率不低于合入前水平（local `make test-coverage` 对比）
- [ ] 禁止禁用失败测试；确需删除需在 PR 说明理由

### 迁移与数据
- [ ] 新迁移文件编号**连续递增**（`bash scripts/check-migrations.sh` 通过）
- [ ] `up/down` 配对，`down` 真正可回滚（不是 `-- noop`）
- [ ] 破坏性变更（删字段/改类型/非空约束）有数据迁移脚本
- [ ] 涉及 TimescaleDB hypertable 的变更已在 staging 验证

### 代码规范
- [ ] 无遗留 `TODO` / `FIXME` / `panic("not implemented")`，如保留需附 issue 链接
- [ ] 错误处理符合 `CLAUDE.md §8.2`：`fmt.Errorf("context: %w", err)`，无裸 panic
- [ ] SQL 构建使用 Squirrel，无字符串拼接
- [ ] 运营商差异通过 `Carrier` 接口适配，无 `if carrier == "cmcc"` 硬编码
- [ ] 前端：后端响应定义 `BackendXxx` → `mapBackendXxx` → 前端 `Xxx`，无 `any`
- [ ] 新增日志字段采用 `zap.String/Int/Error` 结构化形式

### 文档
- [ ] PR 说明填写了 **Why**（为什么做），不只是 **What**
- [ ] **关联 Backlog Task（`T-NNNN`，见 `docs/project/backlog.md`）必填**；无对应 Task → 先用 `/dev-pipeline backlog add "<title>"` 登记再提交
- [ ] 关联 PRD（如为 feat/P0/P1）：`docs/project/prd/F{NN}-*.md`
- [ ] 如涉及约定变更，`CLAUDE.md` 或 `omcgo/CLAUDE.md` 已同步更新
- [ ] 如涉及对外接口变更，Swagger/API 文档已更新

### 流水线闭环（S6/S7 硬门）
- [ ] commit footer **四元组齐全**（见 `.claude/commands/dev-pipeline.md §D3`）：
  - `PRD:` 路径 或 `N/A (type=<bugfix|docs|...>)`（必填一项）
  - `Sprint:` `sprint-NN`（必填，无则 `N/A (out-of-sprint hotfix)`）
  - `Risk:` `R-NNN` 或 `-`（必填，`-` 表示无关联风险）
  - `Backlog:` `T-NNNN`（**必填，不允许空**）
  - `Review:` 审查报告路径 或 `N/A (skipped per §C <type>)`
- [ ] backlog.md Task 状态已回写（`in_review → done` / 填 `Closed` / §6 追加 Evidence）
- [ ] 如走快速通道（hotfix/bugfix 裁剪 S0-S2）→ postmortem 或裁剪说明已补

### 安全
- [ ] 新增 API 端点有 JWT 验证中间件
- [ ] 敏感操作有 RBAC 校验
- [ ] 输入参数有校验；SQL 参数化；避免路径遍历
- [ ] 日志/错误**不打印**密码、Token、CPE 密钥
- [ ] 文件上传验证类型与大小

### 可观测性
- [ ] 关键路径注册 Prometheus 指标（计数器/直方图）
- [ ] 错误日志携带 `request_id` 或 `trace_id`
- [ ] 长耗时操作支持 context 取消

---

## 模块特定 DoD

### ACS / TR-069（`internal/acs/**`）
- [ ] SOAP 信封命名空间声明完整（cwmp/soap/xsd/xsi）
- [ ] 请求/响应 `cwmp:ID` 匹配
- [ ] 会话状态转换覆盖异常路径，安全降级
- [ ] Empty HTTP Response（204/空 body）正确释放 CPE 会话
- [ ] 新 RPC 方法：`cpe_simulator.py` 能模拟成功

### 前端（`omcmb/webcode/**`）
- [ ] API 服务连真实后端（而非仅 mock）
- [ ] Hook 模式：`useMock ? mockService : realApi`
- [ ] 查询键层级：`['domain', 'action', params]`
- [ ] 用户可见文本通过 `react-intl`
- [ ] 错误有页面级 ErrorBoundary 或提示
- [ ] **新页面 → `menus` 表 seed migration**（T-0137 教训）：动态菜单守卫开启时，仅注册路由不够，必须在 `omcgo/migrations/seed/000NNN_*.sql` 注入菜单行 + role_menus 绑定，否则 PrivateRoute 重定向 /403
- [ ] **新组件用到的 `t('xxx.yyy')` key 必须存在于 `frontend-core/src/i18n/{zh-CN,en-US}/index.ts`**（T-0137 教训）：表格"操作"列约定用 `table.action`，DataTable / 通用对话框等模式参考既有页面，不要凭直觉自创 `common.action` 这类未在 catalog 注册的 key

### 告警（`internal/alarm/**`）
- [ ] 去重键符合 `alarm:active:{device_sn}:{code}` 模式
- [ ] 生命周期状态转换合法（激活/确认/清除）
- [ ] 规则 action="notify" 能触达通知渠道（邮件/短信/Webhook 至少一种）

### 性能/告警数据（`internal/pm/**`、`internal/mr/**`、TimescaleDB）
- [ ] 时序表采用 hypertable
- [ ] 批量写入使用 `CopyFrom` 或批量 INSERT
- [ ] 查询有时间范围限制（防全表扫描）

### 数据库迁移（`migrations/**`）
- [ ] 编号严格连续（当前已知缺 000010、000015-000018，新迁移必须衔接下一个可用编号）
- [ ] 含 `-- +goose Up` / `-- +goose Down` 段
- [ ] 索引命名 `idx_{table}_{columns}` 或 `uq_{table}_{columns}`
- [ ] 大表变更用 `CONCURRENTLY` 创建索引

---

## DoD 勾选规范

**PR 模板** 自动展开本文件的通用 DoD 清单。作者提交 PR 时：
- 每项逐项判断，打勾 `[x]` 或注明 `N/A（原因）`
- 未完成项应在 PR 说明列出"跟进 issue"
- 评审人对 N/A 理由进行交叉检查

**Claude 的职责**：在 PR review（`/review` Skill）时，按本清单逐项核查，未勾选项必须被指出。

---

## DoD 演进

- 每 sprint 回顾时检视一次，剔除形同虚设的项、补充漏掉的坑
- 演进记录在 `docs/project/sprint/sprint-NN.md` 的 "DoD 调整" 段
- 重大修订推 PR 到本文件，由 QA/发布经理审批

**当前版本**：v1.0（2026-04-20）
