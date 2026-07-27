# Review: #168 Agent 默认连接、运行时契约与文件分析

## 结论

**PASS_WITH_WARNINGS**

未发现 CRITICAL 或阻断合入的问题。改动限定在 Agent 配置、运行时手册、工具执行与系统配置页，没有改变原有 Web UI 业务 API 调用路径。

## 范围

- 默认 Agent Studio 地址、开发期 Service Token 与受管 Connector 实例
- 运行时 API 手册生成、关联操作、空结果语义与参数契约
- 工具请求的 operationId、方法、路径、查询参数和枚举校验
- OMC 下载响应到 Agent Studio 会话附件的文件传输
- 系统配置页 Connector 实例只读展示

## 审查结果

### 安全

- 文件大小限制为 25 MiB，超限和空文件均拒绝。
- 文件名经过路径剥离和控制字符清理，避免路径穿越。
- 上传使用 Agent Studio Service Token，并携带当前 OMC 用户与 conversationId。
- 上传响应校验文件大小和 SHA-256，完整性不匹配时不向模型投递。
- OMC 既有只读/写方法策略和禁止路径策略继续生效。

### 正确性

- 运行时手册以当前实际路由为准，并在生成阶段校验 operationId 唯一性、请求/响应覆盖和关联操作完整性。
- Handler 参数作为执行允许列表，OpenAPI 只补充描述，避免文档参数与真实处理器漂移。
- 文件响应仅由 Content-Disposition、显式文件名或 octet-stream 识别，普通 JSON/文本响应不会误判。
- CSV、XLSX、TXT、XML 真实端到端场景已验证；无文件场景不会编造结果。

### 测试

- `go build ./...`：通过。
- Agent 相关 Go 包：通过。
- `go test ./...`：使用现有本地库时，`internal/paramsync` 因数据库基线缺表失败；使用全新 schema 库后该包通过。
- 全新 schema 库的 MML 集成测试仍依赖未被同一 goose 版本表执行的旧 seed 基线，属于主线已有迁移/seed 测试环境问题，与本 MR 无关。
- `npm run typecheck`：通过。
- 本 MR 涉及的前端文件 ESLint：通过。
- 全量 ESLint 有 3 个主线既有错误，分别位于 `runtimeClient.test.ts`、`mmlConsoleStore.ts` 和 `DeviceDetail/kpiSeries.ts`。
- `make agent-handbook-check`：通过。
- `golangci-lint`：本机未安装，未执行。

## Warning

当前按开发/POC 决策将共享 Service Token 作为默认值写入源码。这会让有仓库读取权限的人获得该凭证。商业化或扩大仓库访问范围前，必须迁移为部署环境密钥并轮换 Agent Studio 共享 Token。

## DoD

- [x] 后端构建通过
- [x] Agent 相关单元测试覆盖成功与失败路径
- [x] 前端类型检查通过
- [x] 变更文件 lint 通过
- [x] API 手册生成物与源码一致
- [x] 无新增 REST 路由或数据库迁移
- [x] MR 说明包含 Why、影响和测试证据
- [x] 关联 GitLab Issue #168
- [x] 安全边界已审查
- [x] 真实端到端文件场景已验证
