# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-24 |
| 基准提交 | 6efa53f |
| 作者 | watermelon |
| Scope | acs, provision |
| 审查结论 | **PASS** |

## 变更概要

修复两阶段参数同步（GPN→GPV）在真实基站联调中发现的 3 个 Bug，使 GPN 阶段能正确收集实例路径并转入 GPV 阶段。

## 变更文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/acs/session.go` | 新增字段 | Session 增加 `LastCommandParams` 存储 RPC 命令参数 |
| `internal/acs/handler.go` | Bug 修复 | 发送 RPC 时保存命令参数，响应事件中提取 path 用于 GPN 关联 |
| `internal/provision/engine.go` | Bug 修复 | 不再跳过空 GPN 响应，确保 PendingGPNCount 正确递减 |
| `internal/provision/sync.go` | Bug 修复 | 保存 sync plan 前先清除旧条目，避免 Redis Sorted Set 累积 |

## 审查清单

### Go 工程专家
- [x] 错误处理：`fmt.Errorf` 包装，无裸 panic
- [x] 并发安全：Session 更新通过 sessionStore，无竞态
- [x] Context 传递：正确传播
- [x] 资源释放：无新 goroutine/连接
- [x] 命名规范：PascalCase 导出，camelCase 未导出

### TR-069 协议栈专家
- [x] 会话状态：LastCommandParams 在两个 HandleEmpty 分支中均设置
- [x] RPC 关联：命令参数随会话持久化，响应时正确提取 path
- [x] 空响应处理：某些 CPE 路径返回空子节点，不应被丢弃

### 数据与存储专家
- [x] Redis 操作：Clear + Push 避免 Sorted Set 成员累积
- [x] 幂等性：Clear 后 Push 是原子语义上安全的（单设备串行处理）
- [x] TTL：sync plan 通过 command queue 的现有 TTL 机制管理

## 发现

### WARNING

1. **sync plan 的 Clear+Push 非原子** (`sync.go:421-430`)
   - `Clear` 和 `Push` 是两次独立 Redis 操作，极端情况下中间可能有读取
   - **风险**：低。同一设备的 GPN 响应由 provision engine 串行处理
   - **建议**：当前可接受，后续可考虑 Lua 脚本原子化或改用 Redis STRING

### INFO

2. **path 日志使用 fmt.Sprintf** (`handler.go:1252`)
   - `zap.String("path", fmt.Sprintf("%v", payload["path"]))` 可简化为类型断言
   - 不影响功能，属于风格优化

## 结论

3 个 Bug 修复均针对真实基站联调中发现的问题，修改精准、范围小。无 CRITICAL 问题，代码质量良好。
