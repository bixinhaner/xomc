<!--
  ⛔ 整改 Wave 1-3 期间（2026-04-27 ~ 2026-08-03）额外约束：
    1. 任何"新功能"PR 必须挂在 W1.1~W1.8 / Block A-I 子任务，否则关闭
    2. 测试覆盖率单调不降；E2E 用例数单调不降
    3. CI 红灯不绕过（不允许 // nolint:all、--no-verify）
    4. 详见：docs/project/整改路线图-2026Q2.md / docs/methodology/AI承诺对峙清单.md
-->

<!--
  PR 模板 — 本文件由 docs/project/process-design-20260420.md §6.1 定义。
  所有 PR 必须填写本模板。留空或略过会被 QA/发布经理退回。
-->

## 关联

- **Issue**: #___  （必填。无 Issue 的变更请先开 tech-debt Issue）
- **PRD**（如为新功能）: `docs/project/prd/F{NN}-{slug}.md`
- **Milestone / Sprint**: `docs/project/sprint/sprint-NN.md`
- **Risk**（如关闭风险）: R-NNN

## 变更摘要

### Why（必填 — 业务/技术动机）
<!-- 用 2-4 句话说明"为什么做"。不是"这个 PR 做了什么"。 -->


### What（具体变更）
<!-- 分点列出关键变更：新增了什么 / 修改了什么 / 删除了什么 -->
-
-

### 影响面
<!-- 哪些模块、端点、数据库表、前端页面会受影响？ -->
-

---

## Definition of Done 清单

> **说明**：详细见 [`docs/project/dod.md`](../docs/project/dod.md)。每项必须 `[x]`，不适用请注明 `N/A（原因）`。

### 编译与类型
- [ ] 后端 `go build ./...` 通过
- [ ] 后端 `go test ./...` 全绿
- [ ] 后端 `golangci-lint run` 无新增告警
- [ ] 前端（如涉及）`tsc --noEmit` + `npm run lint` 通过

### 测试
- [ ] 新增/修改代码有单元测试（成功路径 + 失败路径）
- [ ] 新端点有 E2E 用例（`scripts/e2e_verify.sh`）
- [ ] 修复 bug 有回归测试
- [ ] 测试覆盖率不低于合入前

### 迁移与数据（如涉及）
- [ ] `bash omcgo/scripts/check-migrations.sh` 通过
- [ ] `up/down` 配对且可回滚
- [ ] 破坏性变更有数据迁移脚本

### 代码规范
- [ ] 无残留 `TODO`/`FIXME`/`panic` 未关联 Issue
- [ ] 错误处理符合 `fmt.Errorf("context: %w", err)`
- [ ] SQL 使用 Squirrel 构建（无字符串拼接）
- [ ] 运营商差异通过 `Carrier` 接口适配
- [ ] 前端无 `any`，有 `BackendXxx → mapBackendXxx → Xxx` 映射

### 文档
- [ ] PR 说明含 Why
- [ ] CLAUDE.md 如涉及约定变更已同步
- [ ] Swagger/API 文档如涉及已更新

### 安全
- [ ] 新端点有 JWT + RBAC 中间件
- [ ] 输入校验齐全，SQL 参数化
- [ ] 日志/错误不含密钥或认证信息

### 可观测性
- [ ] 关键路径有 Prometheus 指标
- [ ] 日志携带 `request_id`

---

## 测试证据

<!-- 贴出：单测输出摘要、E2E 运行结果、手工验证步骤与截图 -->

```
# 粘贴命令输出
```

---

## 回滚计划（如涉及迁移/接口变更）

<!-- 遇到线上问题怎么回滚？步骤是什么？回滚脚本在哪？ -->


---

## 对审阅人的说明

<!-- 需要特别关注什么？有什么未决的设计权衡？ -->


---

<!-- Claude/AI 补充自审摘要（如适用） -->
<!--
  本 PR 由 Claude 协作完成时，请在此处简述 AI 审查通过项与人工需复核项。
  例：已通过领域专家（TR069 + Go 工程）审查；待 QA 人工复核 E2E 覆盖。
-->
