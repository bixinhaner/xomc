# 即插即用策略与参数配置交付审查

- 审查日期：2026-08-05
- 基线提交：`fa816ae31`
- 分支：`feat/plug-and-play-menu-visible`
- 范围：即插即用策略、参数模板编译、XML 下发、软件升级与许可证预安装、前端策略编辑
- 结论：PASS_WITH_WARNINGS

## 摘要

本次审查覆盖跨前后端 107 个文件。后端新增策略持久化、参数编译、XML 生成与下载、任务续跑及软件/许可证编排；前端接入真实策略 API，并补充 eNB、gNB、GSM 参数模板、快速配置和批量导入导出能力。

审查过程中发现 3 个前端一致性测试失败，已在提交前修复：

1. gNB 保存时，刷新工作簿字段会把顶层 `IPSEC_ENABLE` 覆盖为默认值。
2. LTE 模板字段覆盖清单遗漏 `NETWORK.NTP Enable`。
3. LTE 模板表头测试未同步新增的 `NTP Enable` 字段。

## 安全与正确性

- 公共 XML 下载入口使用每文件 UUID token，并以常量时间比较校验；未发现未授权直接读取 XML 内容的路径。
- 新增 SQL 使用参数化构造与占位符，未发现字符串拼接 SQL。
- 未发现运营商硬编码分支、明文生产凭据或新增裸 `panic`。
- 页面中的 `secret123` 为既有演示策略数据，并非本次新增的运行时凭据。
- YAML 文件未进入暂存区；`deployments/docker/docker-compose.yml` 保留为本地未提交改动。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./...`：本次涉及的 provider、ACS、backup、device、paramsync、provision、software 与 TR-069 包均通过；全量命令被未改动的 `internal/task` 用例 `TestService_PG_GetTask_TerminalTombstoneReturnsDurableDetails` 阻断，单独复跑仍失败，并伴随本机临时 Redis 端口连接失败。
- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb && npm test --workspace webcode -- --run`：初次发现 3 项一致性失败；修复后定向 29 项通过，全量复跑结果见最终交付记录。
- `git diff --cached --check`：通过。

## 发现

### WARNING

1. 后端全量测试存在与本次改动无路径交集的 `internal/task` 失败。该问题不影响本次变更包的编译和相关包测试，但应由任务模块维护者修复其 Redis/tombstone 测试隔离。
2. 仓库仅提供 `webcode` 源码；`webcode-v2`、`webcode-v3` 只有未跟踪构建产物，无法在本次源码提交中同步三套皮肤。现有 `npm run typecheck` 也只校验 `webcode`。

### INFO

1. 初始 schema 与 seed 直接更新，适用于新环境初始化；已部署环境需要确认是否另有增量迁移策略。
2. 新增 Excel 模板在文档归档目录和前端运行时目录各保存一份，文档已说明两份副本需保持哈希一致。

## 结论

未发现阻断提交的 CRITICAL 问题。除已记录的既有全量测试隔离问题外，本次改动具备提交条件。
