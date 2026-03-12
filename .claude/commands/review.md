# 代码审查 (Code Review)

对暂存区或工作区中的变更文件执行代码审查，生成结构化审查报告。支持 Go 后端和 React/TypeScript 前端代码。

## 参数

- `$ARGUMENTS` — 可选。`staged` 仅审查暂存文件；`<commit-hash>` 审查指定 commit 的变更；留空则审查所有变更文件。

---

## 执行步骤

请按以下顺序严格执行，每步完成后报告状态：

### Step 1: 收集变更信息

并行执行以下命令，收集审查所需的元数据：

1. **变更文件列表**:
   - 如果 `$ARGUMENTS` 为 `staged`：`git diff --staged --name-only`
   - 如果 `$ARGUMENTS` 为某个 commit hash：`git diff <commit-hash>^..<commit-hash> --name-only`
   - 否则：`git diff --staged --name-only` + `git diff --name-only`（合并去重）
   - 如果没有任何变更文件，**停止并提示用户**

2. **作者**: `git config user.name`（用于报告文件名，空格替换为下划线，去除特殊字符）

3. **短哈希**: `git rev-parse --short HEAD`

4. **日期**: `date +%Y%m%d`（用于目录名）

### Step 2: 确定审查范围 (Scope)

根据变更文件的路径前缀推断 scope：

| 路径前缀 | Scope |
|----------|-------|
| `omcgo/internal/acs/` | acs |
| `omcgo/internal/config/` | config |
| `omcgo/internal/pm/` | pm |
| `omcgo/internal/alarm/` | alarm |
| `omcgo/internal/mr/` | mr |
| `omcgo/internal/device/` | device |
| `omcgo/internal/admin/` | admin |
| `omcgo/internal/topology/` | topology |
| `omcgo/internal/software/` | software |
| `omcgo/internal/backup/` | backup |
| `omcgo/internal/dashboard/` | dashboard |
| `omcgo/internal/ops/` | ops |
| `omcgo/internal/report/` | report |
| `omcgo/internal/mml/` | mml |
| `omcgo/internal/filemanager/` | filemanager |
| `omcgo/internal/syslog/` | syslog |
| `omcgo/internal/license/` | license |
| `omcgo/internal/nedirect/` | nedirect |
| `omcgo/internal/northbound/` | northbound |
| `omcgo/internal/provision/` | provision |
| `omcgo/internal/interop/` | interop |
| `omcgo/internal/carrier/` | carrier |
| `omcgo/internal/components/` | components |
| `omcgo/internal/middleware/` | middleware |
| `omcgo/internal/event/` | event |
| `omcgo/internal/model/` | model |
| `omcgo/internal/errors/` | errors |
| `omcgo/cmd/` / `omcgo/Makefile` / `omcgo/Dockerfile` / `omcgo/deployments/` | deploy |
| `omcgo/migrations/` | migration |
| `omcgo/scripts/` | scripts |
| `omcmb/webcode/src/services/api/` | api |
| `omcmb/webcode/src/hooks/` | hooks |
| `omcmb/webcode/src/pages/<name>/` | 使用 `<name>` 作为 scope |
| `omcmb/webcode/src/components/` | components |
| `omcmb/webcode/src/store/` | store |
| `omcmb/webcode/src/types/` | types |
| `omcmb/webcode/src/services/http.ts` | http |
| 根目录 `*.md` / `docs/` | docs |

**组合规则**:
- 单一 scope → 直接使用（如 `device`）
- 2-3 个 scope → 连字符连接（如 `device-alarm`）
- 超过 3 个 scope → 使用 `multi`
- 同时涉及 omcgo 和 omcmb → 前缀加 `fullstack-`（如 `fullstack-device`）

### Step 3: 读取并分析变更代码

读取每个变更文件的 diff 内容（使用 `git diff` 或 `git diff --staged`），逐文件进行审查。

#### Go 后端检查项

| 检查项 | 说明 | 严重级别 |
|--------|------|---------|
| 命名规范 | 导出用 PascalCase，未导出用 camelCase，包名小写单数 | WARNING |
| 错误处理 | 必须 `fmt.Errorf("context: %w", err)` 包装，禁止裸 panic | CRITICAL |
| 运营商硬编码 | 禁止 `if carrier == "cmcc"` 类硬编码，必须通过 Carrier 接口 | CRITICAL |
| 接口定义 | 核心领域概念应定义接口（Repository、Service） | WARNING |
| SQL 安全 | 必须用 Squirrel 参数化构建，禁止字符串拼接 SQL | CRITICAL |
| 认证检查 | 受保护接口是否加了 auth middleware | CRITICAL |
| 敏感数据 | 禁止日志输出密码、token 等敏感信息 | CRITICAL |
| 资源泄漏 | 检查 goroutine、DB 连接、文件句柄是否正确关闭 | WARNING |
| 错误码使用 | 新错误码是否在 `global/errors.go` 的分配范围内 | WARNING |
| 测试覆盖 | 新增功能是否有对应 `_test.go` | INFO |

#### React/TypeScript 前端检查项

| 检查项 | 说明 | 严重级别 |
|--------|------|---------|
| 类型安全 | 禁止 `any`，必须定义明确的 interface/type | WARNING |
| API 服务模式 | 应遵循 `BackendXxx` → `mapBackendXxx` → `xxxApi` 模式 | WARNING |
| Hook 模式 | 必须使用 `useMock ? mockService : realApi` 开关 | WARNING |
| QueryKey | 必须使用层级式 `['resource', 'action', params]` | INFO |
| XSS 防护 | 检查 `dangerouslySetInnerHTML` 使用，确保输入已转义 | CRITICAL |
| Token 处理 | Token 不应出现在 URL 参数或 localStorage 明文中 | CRITICAL |
| 组件规范 | 使用函数组件 + TypeScript props 类型 | INFO |
| 不必要渲染 | 大列表/复杂计算是否使用了 useMemo/useCallback | INFO |

#### 通用检查项

| 检查项 | 说明 | 严重级别 |
|--------|------|---------|
| 代码重复 | 超过 10 行的相似代码块 | WARNING |
| 硬编码值 | 魔法数字、硬编码 URL/端口 | WARNING |
| 日志质量 | 关键操作是否有日志，日志是否包含上下文 | INFO |
| 文件大小 | 单文件超过 500 行建议拆分 | INFO |

#### 业务完整性检查

验证变更的业务逻辑是否完整闭环，不留残缺功能：

| 检查项 | 说明 | 严重级别 |
|--------|------|---------|
| Handler-Service-Repository 链路 | 新增 handler 必须有对应 service 和 repository 实现，禁止空壳 handler 直接操作 DB | CRITICAL |
| 路由注册 | 新增 handler 是否在 `router.go` 中注册了路由 | CRITICAL |
| 迁移文件配套 | 新增 model/repository 引用的表是否有对应 migration `.up.sql` + `.down.sql` | CRITICAL |
| 错误码注册 | 新增业务错误是否在 `global/errors.go` 中定义了错误码 | WARNING |
| API 服务配套 | 新增后端接口是否有对应的前端 `xxxApi.ts` 服务方法 | INFO |
| Hook 配套 | 新增���端 API 服务方法是否有对应的 React Query Hook | INFO |
| Mock 配套 | 新增前端 Hook 是否有对应的 mock service 实现 | INFO |
| 种子数据 | 新增表/接口是否在 `seed_e2e_testdata.sql` 中补充了测试数据 | INFO |
| E2E 测试用例 | 新增接口是否在 `e2e_verify.sh` 中补充了测试用例 | INFO |

#### 业务影响范围检查

评估本次变更对上下游模块的潜在影响，确保不引入隐式依赖问题：

| 检查项 | 说明 | 严重级别 |
|--------|------|---------|
| 接口签名变更 | 修改 Repository/Service 接口方法签名是否影响了所有调用方 | CRITICAL |
| 数据库 Schema 变更 | 表结构变更（加列/改类型/删列）是否影响现有查询和 model 映射 | CRITICAL |
| 事件契约变更 | EventBus 发布的事件结构变更是否通知了所有订阅方 | WARNING |
| 共享 model 变更 | `internal/model/` 下的共享类型修改是否影响引用它的模块 | WARNING |
| 中间件变更 | `middleware/` 变更是否影响所有经过该中间件的路由 | WARNING |
| 配置项变更 | `appconfig/` 或 `config.*.yaml` 变更是否需要同步更新部署配置和文档 | WARNING |
| 运营商适配器变更 | `carrier/` 下某运营商适配器变更是否需要同步其他运营商 | INFO |
| API 响应格式变更 | 后端 response 字段的增删改是否需要通知前端同步调整 | WARNING |
| 跨模块引用 | 新增的 import 是否引入了不合理的模块间依赖（如 pm → alarm 的循环引用） | WARNING |

#### 前后端一致性检查

**提醒级别 (INFO)**：检查前后端接口契约的一致性，确保两侧变更同步：

| 检查项 | 说明 | 严重级别 |
|--------|------|---------|
| 接口路径一致 | 后端 `RegisterRoutes` 注册的路径与前端 `xxxApi.ts` 调用路径是否一致 | INFO |
| 请求参数一致 | 后端 `ShouldBindJSON` 的 struct tag 与前端请求参数（经 snake_case 转换后）是否匹配 | INFO |
| 响应字段一致 | 后端 response struct 的 JSON tag 与前端 `BackendXxx` interface 字段是否匹配 | INFO |
| 分页参数一致 | 后端分页参数命名（page/page_size/sort_by/sort_dir）与前端 http.ts 拦截器的 `paramKeyMap` 是否一致 | INFO |
| 错误码处理 | 后端返回的业务错误码前端是否有对应的处理逻辑或展示文案 | INFO |
| 枚举值一致 | 后端定义的状态枚举（如 DeviceStatus、AlarmSeverity）与前端 types 中的定义是否一致 | INFO |
| 新接口双侧覆盖 | 本次新增的后端接口是否有对应的前端调用实现（或标注为后续 Sprint 实现） | INFO |

#### 代码质量回退检查 ⚠️

**十分严重 (CRITICAL)**：检查本次变更是否导致代码质量回退，以下任一情况视为质量回退：

| 检查项 | 说明 | 严重级别 |
|--------|------|---------|
| 删除测试用例 | 删除或注释掉已有的测试用例，但未替换为等价或更好的测试 | CRITICAL |
| 删除错误处理 | 移除已有的 error check 或将 `if err != nil` 改为忽略错误 `_ = xxx()` | CRITICAL |
| 降级安全措施 | 移除认证中间件、降低密码强度校验、关闭 CORS 检查等 | CRITICAL |
| 引入 any/interface{} | 将强类型改为 `any` 或 `interface{}`，丧失类型安全 | CRITICAL |
| 硬编码替代配置 | 将原本从配置读取的值改为硬编码常量 | CRITICAL |
| 删除日志 | 移除关键操作的日志记录（如删除审计日志写入） | WARNING |
| 简化校验逻辑 | 移除或弱化已有的输入校验（如去掉长度限制、格式检查） | CRITICAL |
| ORM 替代 Squirrel | 引入 GORM 等 ORM 替代已有的 Squirrel SQL 构建 | CRITICAL |
| 绕过接口抽象 | 将通过接口调用改为直接依赖具体实现，破坏依赖倒置 | WARNING |
| TODO/HACK 残留 | 新增代码中包含 `TODO`、`HACK`、`FIXME` 但无关联 issue 或计划 | INFO |

#### 配套更新提醒

**提醒级别 (INFO)**：代码变更后，检查以下配套产物是否需要同步更新：

| 检查项 | 触发条件 | 检查方式 |
|--------|---------|---------|
| 文档更新 | 修改了 API 接口签名/行为、新增功能模块、变更配置项、修改部署流程 | 检查 `CLAUDE.md`、`omcgo/CLAUDE.md`、`README.md`、`docs/` 下相关文档是否仍然准确；如 API 路径/参数/响应有变更，提醒更新接口文档 |
| 单元测试更新 | 修改了 service/repository 的业务逻辑、修复了 bug、重构了函数签名 | 检查对应 `_test.go` 是否覆盖了变更后的逻辑分支；如修改了函数入参/返回值，已有测试是否仍能编译通过；新增的 error path 是否有测试 |
| 端到端测试更新 | 新增/修改/删除了 REST API 端点、变更了请求/响应格式、修改了业务流程 | 检查 `omcgo/scripts/e2e_verify.sh` 中对应接口的测试用例是否需要更新；如新增端点，提醒在 E2E 脚本中补充测试；如变更响应格式，提醒更新断言 |

**判断原则**：
- 改了代码但没改对应测试 → 提醒"建议同步更新单元测试"
- 改了接口但没改文档 → 提醒"建议同步更新接口文档"
- 改了 API 行为但没改 E2E → 提醒"建议同步更新端到端测试"
- 纯内部重构（不改外部行为）→ 不需要提醒文档和 E2E，但需提醒单元测试

### Step 4: 生成审查报告

1. 创建报告目录：
```bash
mkdir -p docs/review-report/$(date +%Y%m%d)
```

2. 将报告写入以下路径：
```
docs/review-report/<YYYYMMDD>/REVIEW_<short_hash>_<author>_<scope>.md
```

3. 报告使用以下模板：

```markdown
# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | YYYY-MM-DD HH:MM |
| 提交 | <short_hash> |
| 作者 | <author> |
| 范围 | <scope> |
| 变更文件数 | N |
| 新增行数 | +N |
| 删除行数 | -N |

## 变更概要

[用 2-3 句话概述本次变更的目的和内容]

## 审查发现

### 🔴 CRITICAL (严重)

> 必须在提交前修复的问题

[如无则写: 无]

### 🟡 WARNING (警告)

> 建议修复，不阻塞提交

[如无则写: 无]

### 🔵 INFO (建议)

> 改进建议，可选择性采纳

[如无则写: 无]

## 详细分析

### `<filepath>`

[逐文件列出具体发现，引用行号]

## 业务完整性检查

[检查 Handler-Service-Repository 链路、路由注册、迁移文件、错误码、API/Hook/Mock 配套等是否完整闭环]
[如全部完整则写: "业务链路完整，无遗漏"]

## 业务影响范围检查

[评估本次变更对上下游模块的影响：接口签名、DB Schema、事件契约、共享 model、中间件、配置、API 响应格式等]
[如无跨模块影响则写: "变更范围可控，未发现跨模块影响"]

## 前后端一致性检查

[检查前后端接口契约一致性：路径、请求参数、响应字段、分页参数、错误��、枚举值]
[如仅涉及单侧变更，提示是否需要另一侧同步]
[如无前后端同时变更则写: "本次变更仅涉及[后端/前端]，建议关注对应[前端/后端]的同步需求"]

## 代码质量回退检查

[检查是否存在质量回退：删除测试、移除错误处理、降级安全措施、引入 any、硬编码替代配置、简化校验等]
[如无回退则写: "未发现代码质量回退"]
[如发现回退，必须标记为 CRITICAL 并详细说明回退点]

## 配套更新提醒

[根据变更内容，提醒以下配套产物是否需要同步更新]

- **文档**: [是否需要更新 CLAUDE.md / 接口文档 / README / 设计文档？说明原因或写"无需更新"]
- **单元测试**: [是否需要新增或更新 _test.go？列出建议补充测试的函数/方法，或写"已有测试覆盖"]
- **端到端测试**: [是否需要更新 e2e_verify.sh？列出需要补充的 API 测试用例，或写"已有 E2E 覆盖"]

## 安全检查

[安全相关发现汇总，或 "未发现安全问题"]

## 性能检查

[性能相关发现汇总，或 "未发现性能问题"]

## 测试覆盖

[测试覆盖情况评估]

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | N |
| WARNING | N |
| INFO | N |

**审查结论**: `PASS` / `PASS_WITH_WARNINGS` / `NEEDS_FIX`

- **PASS**: 无 CRITICAL 和 WARNING
- **PASS_WITH_WARNINGS**: 无 CRITICAL，有 WARNING
- **NEEDS_FIX**: 存在 CRITICAL 问题
```

### Step 5: 输出审查结果

1. 打印审查报告的完整路径
2. 打印审查结论（PASS / PASS_WITH_WARNINGS / NEEDS_FIX）
3. 如有 CRITICAL 问题，逐条列出并建议修复方案
4. 打印总结统计（CRITICAL / WARNING / INFO 各多少个）

---

## 注意事项

- 审查应基于项目规范（参见根目录 `CLAUDE.md` 和 `omcgo/CLAUDE.md`）
- 对于纯文档变更（`.md` 文件），仅检查格式和内容准确性，不执行代码检查项
- 对于迁移文件（`migrations/`），检查 SQL 语法、���引、是否有对应 down 迁移
- 报告中的行号应引用 diff 中的实际行号，方便定位
- 如果变更文件过多（超过 30 个），优先审查核心业务文件，跳过自动生成文件
