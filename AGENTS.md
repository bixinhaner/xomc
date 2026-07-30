# AGENTS.md — OMC 项目根级指导

本文件每次 AI 会话都会加载，只保留高频、易错、犯错代价高的规则。能通过扫描代码、README 或 docs 索引发现的信息，不写在这里。

## 项目语境

OMC 是面向小基站运维的系统，核心协议是 TR069/CWMP，按运营商级质量处理。

## 仓库与提交边界

- 仓库根目录是 `xomc/`，所有 git 操作在根目录执行。
- `omcgo/`、`omcmb/` 不是独立 git 仓库。
- `.codex/`、本机私有插件缓存、个人临时配置默认不提交，除非用户明确要求。

## 高风险代码规则

- 后端 SQL 使用 Squirrel + pgx，禁止 ORM 和字符串拼接 SQL。
- 后端错误要包装上下文：`fmt.Errorf("context: %w", err)`。
- 运营商差异走 `internal/core/carrier/`，禁止散落 `if carrier == "cmcc"`。
- 当前软件未封版本，`omcgo/migrations/` 只允许维护三个基线文件：`000001_init_schema.sql`、`seed/000001_init_seed.sql`、`tsdb/000001_tsdb_schema.sql`。不允许新增 `000002+` 迁移；新增 DB / seed / tsdb 变化必须折回对应 `000001`。软件封版本后，先更新本文件和 `omcgo/migrations/README.md`、`omcgo/migrations/seed/README.md`，再允许从 `000002` 开始追加迁移。

## 前端易错规则

- 当前只维护 V1 单皮肤；前端改动不要恢复或维护旧 V2/V3 皮肤。
- 用户可见文案必须走 i18n，不要在组件或业务逻辑中硬编码。
- 页面、表单、toast、Modal、空态、错误信息等交互改动必须用真实浏览器验证，不能只按源码推断。

## PM 性能管理知识库

处理 PM 指标、KPI/counter、聚合、导出、设备性能查看、指标查询、`pm_metrics`、`pm_adhoc_aggregation_results`、`statis_type`、15 分钟点、hourly/daily/weekly/monthly 口径问题时，必须先阅读：

- `docs/ref/pm-metrics-knowledge.md`

先确认统计口径和入口矩阵；涉及页面行为必须用浏览器验证真实请求参数。

## 验证与 Git

- 验证命令按改动范围现查；不要假装跑过，失败要说明命令和原因。
- 不主动 `git fetch` / `git pull` / `git push`，除非用户明确要求或收尾流程需要。
- 不执行 `git reset --hard`、`git clean -f`、强推、`--no-verify`，除非用户明确要求且风险已说明。
- 不回滚用户已有改动；遇到无关脏文件直接忽略。
- 提交信息使用 Conventional Commits，中文描述。

## 工作方式

- 先读现有代码和文档，再动手。
- 小步修改，保持可验证。
- 保持项目既有模式，避免无关重构。
- 高风险共享逻辑、跨模块契约或用户可见流程要补测试。
- 复杂问题最多尝试三种方案；仍不通时停下来记录现象、假设和下一步。
