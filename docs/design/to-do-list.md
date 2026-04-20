# ~~消息队列合并可行性评估和实施方案~~ **已完成**

## 合并结果：CommandQueue + TaskService → 统一 TaskService

### 变更摘要

| 维度 | 合并前 | 合并后 |
|------|--------|--------|
| App/Worker 进程 | `RedisCommandQueue` | `BridgeQueue`（桥接 `TaskService`） |
| ACS 进程 | 双队列优先级级联 | 单一 `TaskService.PopTask` |
| Redis Key | `acs:cmdq:*` + `acs:taskq:*` | `acs:taskq:*`（统一） |
| PG 持久化 | 仅 TaskService 路径 | 全部任务 |
| Handler fallback | 有（commandQueue.Pop） | 无 |

### 修改文件

**DI 替换（3 进程）**：
- `omcgo/cmd/app/bootstrap.go` — 创建 TaskService + BridgeQueue
- `omcgo/cmd/app/provider/container.go` — CmdQueue 类型改为接口
- `omcgo/cmd/app/provider/modules.go` — initTaskModule 使用预创建的 TaskService
- `omcgo/cmd/app/main.go` — 传递 TaskSvc
- `omcgo/cmd/worker/bootstrap.go` — 创建 TaskService + BridgeQueue
- `omcgo/cmd/worker/main.go` — 使用 BridgeQueue
- `omcgo/cmd/acs/main.go` — 移除 cmdQueue，TaskService 必需（PG 必连）

**Handler 统一**：
- `omcgo/internal/acs/handler.go` — 移除 commandQueue 字段、双队列 fallback、nil 守卫
- `omcgo/internal/acs/server.go` — 移除 CommandQueue from ServerDeps/NewDefaultDeps
- `omcgo/internal/acs/handler_test.go` — 所有测试改用 TaskService mock

**文档**：
- `omcgo/CLAUDE.md` — Redis Key、接口列表、数据存储表更新

### 后续清理（非紧急）

- 移除 `internal/acs/cmdqueue/` 包中未使用的 `RedisCommandQueue` 实现（保留 `Command` struct 和接口定义）
- 迁移 ~22 个业务模块的 Push 调用点，从 `cmdqueue.Command` 改为 `task.CreateTaskRequest`
- 移除 `BridgeQueue` 中间层，让模块直接调用 `TaskService`


# MML bug 修复

## ~~1. 访问 /mml/script 触发 formatjs intl 报错~~ **已修复**

**现象**：访问 http://172.19.1.73:8081/mml/script 触发 `[@formatjs/intl] An 'id' must be provided to format a message`，之后切换到其它页面也报同样错误，刷新/重新登录后恢复正常。

**根因**：`ScriptTask/index.tsx` 中 column render 和详情弹窗使用 `t(SomeMap[val]?.key)` 模式，当后端返回的 status/executeType/result 值不在 TypeScript 联合类型定义中时（如 `'timeout'`、`'error'`、空字符串），map 查找返回 `undefined`，`?.key` 得到 `undefined`，传给 `t(undefined)` 触发 formatjs invariant 抛错。ErrorBoundary 捕获后影响整个 IntlProvider 上下文，导致切换页面也报错。

**修复**（两处）：
1. `useT.ts`：添加 `if (!id) return id` 防御，防止 falsy id 触发 formatjs invariant
2. `ScriptTask/index.tsx`：所有 map lookup 改为先检查 entry 是否存在，不存在时显示原始值而非传 undefined 给 `t()`

**修改文件**：
- `omcmb/webcode/src/hooks/useT.ts` — 添加 falsy id 防御
- `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx` — 6 处 map lookup 安全化（3 处 column render + 3 处详情弹窗）
