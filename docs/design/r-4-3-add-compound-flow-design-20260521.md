# R-4.3 ADD 复合流程 — 设计文档

> Spec：`omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md` §R-4.3
> 日期：2026-05-21
> 状态：**设计待审核**，实施分 3 个 commit

---

## 1. 需求摘要

### 1.1 spec 原文

> #### R-4.3 ADD
> - 仅当命令路径含 `{i}` 时可用；
> - Control Panel 列出该对象下的 RW (📝) 子参数作为新实例的初始值字段，勾选规则同 MOD；
> - 提交时调用 **AddObject(parentPath) 创建新实例，再用 SetParameterValues 写入勾选字段的初始值**（同一会话内）。

### 1.2 当前缺陷

- 用户在 ADD 模式填初始值 → submit
- 后端 `console_executor.go` ADD 分支 (L234-259) 仅派发 **AddObject**，`stmt.Values` 读取但**仅用作审计**（注释 L43-47 "1 MML=1 RPC 不做复合"）
- 用户体验：新实例创建成功，但**所有初始值字段保留 CPE 默认值**，spec 要求未达成
- 现状解决方案：用户手动新建 MOD MML 命令 + 手抄 InstanceNumber，UX 极差

---

## 2. 架构现状（关键事实）

| 维度 | 现状 | 对实施的影响 |
|---|---|---|
| `console_executor` ADD 分支 | 1 MML = 1 RPC（仅 AddObject）；values 仅审计 | **要改**：ADD with values 时生成 2 行 commands |
| Sequencer 任务链 | `sequencer.go OnTaskCompleted` 已实现回调驱动按 `cmd_idx` 推进 | **复用**：仅需加 result 反查 + 占位符替换钩子 |
| AddObject 响应解析 | `acs/handler.go` L710-726 已捕获 `<InstanceNumber>` → 写 `device_tasks.result.instance_number` | **零改动**：直接消费 |
| Translator | 纯静态 standardPath↔privatePath 映射，无运行时变量替换 | **不动**：占位符替换在 Sequencer 层完成 |
| `substituteInstanceSelectors` | 已有静态文本替换链路（处理 `.{i}.`）| **复用模式**：`{NEW}` 替换走同样思路 |
| Fanout `sequentialMode` | 已有 toggle，多行 commands 自动按 cmd_idx 链式入队 | **要验证**：Console execute path 是否启用 |

---

## 3. 架构决策

### 3.1 三个 Option 简对比（详见上轮分析）

| Option | 思路 | LOC | schema 变更 | 推荐 |
|---|---|---|---|---|
| **A** | ConsoleService 生成 2 行 commands + Sequencer 加 `{NEW}` 替换 | ~230 | 无 | ✅ |
| B | 前端拆 ADD → ADD+MOD 两 statement | ~400 | 无 | ❌（违反 spec "同一会话"） |
| C | Backend compound device_task params JSONB | ~500 | device_tasks.params 形态变 + ACS 改 | ❌ |

### 3.2 选 Option A 的关键拐点

1. **不破 "1 MML=1 RPC" 决策**：仍然 1 task = 1 RPC，只是 1 个 MML statement → 2 个 commands 数组 entry。用户 2026-05-20 冻结的决策原文是"不在单个 RPC 里塞两个动作"，本设计不违反。
2. **复用现有 Sequencer 链路**：fanout sequentialMode + sequencer.OnTaskCompleted 已 wire 好，仅需加一个 hook。
3. **InstanceNumber 已持久化**：ACS handler 已经把 AddObject 响应的 InstanceNumber 写入 `device_tasks.result.instance_number`，Sequencer 直接读。
4. **零 schema 变更**：与 P4.c reset 教训一致 — 最小 blast radius，最易回退。

---

## 4. Option A 详细设计

### 4.1 数据流

```
用户提交 ADD with values
   │
   │   POST /mml/console/execute-statements-structured
   │   { op: 'ADD', paths: [parent], values: { sub_path: value, ... }, instanceSelectors: {...} }
   ▼
ConsoleService.ExecuteStructuredStatements
   │
   │   console_executor.go executeStatement(stmt) — 修改后
   ▼
返回 []entry — 2 行：
   entry[0]: { rpc_method: AddObject, parameters: { object_name: "Device.X.1.Y." } }
   entry[1]: { rpc_method: SetParameterValues,
               parameters: { "Device.X.1.Y.{NEW}.Foo": "bar", "Device.X.1.Y.{NEW}.Baz": "qux" } }
   │
   ▼
fanout.go 入队 mml_task.commands[0,1] — sequentialMode 触发只入队 commands[0]
   │
   ▼
ACS dispatch AddObject → CPE 返回 <InstanceNumber>5</InstanceNumber>
   │
   ▼
acs/handler.go MarkTaskCompleted 写 device_tasks.result.instance_number=5
   │
   ▼
event: task_completed
   │
   ▼
sequencer.go OnTaskCompleted(taskA)
   │
   │   ⭐ 新逻辑：buildNextRequest 前从 taskA.result.instance_number 读取
   │      调 substituteNewInstance(nextCommand.parameters, instance_number)
   │      把所有 path 中 "{NEW}" 字面字符串替换为 "5"
   ▼
入队 commands[1] device_task — SPV 此时 paths 已是 "Device.X.1.Y.5.Foo"
   │
   ▼
ACS dispatch SetParameterValues 写入初始值
   │
   ▼
链完成
```

### 4.2 ConsoleService / console_executor 改造

**位置**：`omcgo/internal/mml/console_executor.go` `executeStatement` 函数 ADD 分支（L234-259）。

**改造前**：
```go
case "ADD":
    // ... substitute instance selectors ...
    params := map[string]interface{}{"object_name": targetObject}
    for k, v := range stmt.Values {
        params[k] = v  // 审计透传
    }
    entry["rpc_method"] = "AddObject"
    entry["parameters"] = params
    // returns []entry with 1 item
```

**改造后**：
```go
case "ADD":
    // ... 现有 substituteInstanceSelectors 逻辑 ...
    addEntry := buildAddObjectEntry(targetObject)
    entries = append(entries, addEntry)
    
    // R-4.3 复合：若 stmt.Values 非空，生成第 2 行 SPV 命令
    if len(stmt.Values) > 0 {
        spvParams, err := buildSpvParamsWithNewPlaceholder(stmt, cmd, targetObject)
        if err != nil {
            return nil, fmt.Errorf("ADD compound SPV: %w", err)
        }
        if len(spvParams) > 0 {
            entries = append(entries, map[string]interface{}{
                "rpc_method": "SetParameterValues",
                "parameters": spvParams,
            })
        }
    }
    // returns []entry with 1-2 items
```

**新函数 `buildSpvParamsWithNewPlaceholder`**：
```go
// 对每个 (mml_code, value)：
//   1. 取 sub_field.tr069_path（如 Device.Services.FAPService.{i}.PLMNList.{i}.PLMNID）
//   2. 用 stmt.InstanceSelectors 替换前 N 个 {i}（与 MOD 一致）
//   3. 剩余的最后一个 {i}（对应新实例）替换为字面字符串 "{NEW}"
//   4. 输出 { "Device.Services.FAPService.1.PLMNList.{NEW}.PLMNID": "...", ... }
//
// 算法：targetObject 中有 N 个 {i}（已通过 InstanceSelectors 替换为具体数字），
//      sub_field.tr069_path 应有 N+1 个 {i}（最后 1 个为新实例）。
//      验证不变量；不满足则返回 err。
```

### 4.3 Sequencer 改造

**位置**：`omcgo/internal/mml/sequencer.go` `OnTaskCompleted` → `buildNextRequest`

**改造前**：构造 next_task 时只按 `cmd_idx` 推进，不读 task A.result。

**改造后**：
```go
func (s *Sequencer) buildNextRequest(
    ctx context.Context,
    mmlTask *MMLTask,
    deviceSN string,
    deviceIdx int,
    nextIdx int,
    prevTask *task.Task,  // ⭐ 新增参数：上一个 task，用于读 result
) (*task.CreateTaskRequest, error) {
    nextCmd := mmlTask.Commands[nextIdx]
    
    // R-4.3：若上一 task 是 AddObject 且本次 cmd 是 SPV，应用 {NEW} 替换
    if prevTask != nil &&
       prevTask.RpcMethod == "AddObject" &&
       nextCmd.RpcMethod == "SetParameterValues" {
        instanceNumber, ok := extractInstanceNumber(prevTask.Result)
        if !ok {
            return nil, fmt.Errorf("R-4.3 chain: prev AddObject task %s has no instance_number in result", prevTask.ID)
        }
        nextCmd = substituteNewInstance(nextCmd, instanceNumber)
    }
    
    // ... 现有 buildNextRequest 逻辑 ...
}

// 把 cmd.Parameters 中所有 key 包含 "{NEW}" 的，替换为具体 instance number 字符串
func substituteNewInstance(cmd Command, instanceNumber int) Command { ... }

// 从 task.Result JSON 提取 result.instance_number
func extractInstanceNumber(resultJSON []byte) (int, bool) { ... }
```

### 4.4 占位符语义：`{NEW}`

**为什么用字面字符串 `{NEW}` 而非新字段**：
- 与 spec / 现有 `.{i}.` 占位符同风格，可观察 / debug 友好
- DB 存储不变（device_tasks.params JSONB 里就含字面 `{NEW}`），日志 / 审计可见
- 替换逻辑只需 `strings.Replace(path, "{NEW}", strconv.Itoa(n), -1)` 一行

**约定**：
- `{NEW}` 只出现在 SPV 命令的 parameters 的 key 中（path）
- AddObject 命令永远不含 `{NEW}`
- 一个 SPV path 至多包含一个 `{NEW}`（spec ADD 单层创建实例）

### 4.5 SequentialMode 启用确认

**已有事实**：fanout.go L99-102 提供 `SetSequentialMode(enabled bool)`。Sprint B Q-V3-3 决议启用 — 但默认 OFF。

**待验证**（实施时）：Console execute path 是否调用了 SetSequentialMode(true)？
- 若是 → ADD with values 自动按 sequential 走，无需额外开关
- 若否 → 在 Console fanouter 启用时设 sequentialMode=true（per-task），或全局打开

### 4.6 失败处理矩阵

| 场景 | 行为 | 用户可见 |
|---|---|---|
| AddObject 失败 | Sequencer.OnTaskCompleted 检测 task A failed → 不入队 task B | task A 失败状态；新实例未创建；初始值未设 |
| AddObject 成功 + SPV 失败 | task A 已成功（实例已创建），task B 失败 | task 列表两条：A 成功 / B 失败；用户可手动 MOD 重试或 RMV 清理 |
| AddObject 成功 + result 缺 instance_number | Sequencer 写警告日志 + 不入队 SPV；task A 标 "compound_chain_broken" 标签 | task A 成功但用户看到链未完成提示 |
| Multi `{NEW}` in single path | 防御性 fallback：替换所有 `{NEW}` 为同一 instance_number（应不会出现，但不阻塞）| — |
| stmt.Values 空 | 退化为单行 AddObject（与现状一致） | 老 ADD 行为保留 |

**不做的事**：
- ❌ 不自动 DeleteObject 回滚（spec 未要求；避免双失败级联）
- ❌ 不在前端阻塞 ADD 提交（用户已勾选 + 填值就 submit；后端兜底）

### 4.7 与"1 MML=1 RPC"决策的关系

用户 2026-05-20 决策原文（console_executor.go L43-47）：
> "1 MML = 1 RPC，不做复合。stmt.Values 透传到 parameters 仅为审计；BuildTR069Params(AddObject) 只用 object_name。"

**本设计如何对齐这个决策**：
- 仍然 **1 task = 1 RPC**（task A 是 1 个 AddObject RPC，task B 是 1 个 SetParameterValues RPC）
- 但 **1 个 MML statement → 2 个 commands 数组 entry**（这是 MML scripting 层概念，不是 RPC 层）
- 类比：用户在 textbox 里手写 "ADD X; MOD X.5.Foo=bar"（两行 MML）走 sequentialMode 时与本设计行为完全一致

**决策注解需要更新吗？**
- L43-47 的注释保留（仍然描述 1 task=1 RPC 原则）
- 新增注释说明 ADD 复合路径在 ConsoleService + Sequencer 协作下实现

---

## 5. 实施清单

### Phase 1 — Backend ConsoleService（commit 1）

| 文件 | 改动 | 估计 LOC |
|---|---|---|
| `omcgo/internal/mml/console_executor.go` | ADD 分支生成 2 行 commands；新增 `buildSpvParamsWithNewPlaceholder` 函数 | ~80 |
| `omcgo/internal/mml/console_executor_test.go` | 测试：ADD with values → 2 行；ADD without values → 1 行；多层 `{i}` 边界 | ~60 |
| **小计** | | ~140 |

### Phase 2 — Backend Sequencer（commit 2）

| 文件 | 改动 | 估计 LOC |
|---|---|---|
| `omcgo/internal/mml/sequencer.go` | `buildNextRequest` 加 prevTask 参数；新增 `extractInstanceNumber` / `substituteNewInstance` 辅助 | ~80 |
| `omcgo/internal/mml/sequencer_test.go` | 测试：链 AddObject→SPV 时 `{NEW}` 替换；缺 instance_number 时 break；非 ADD 链不动 | ~80 |
| （可选）fanout.go | Console execute path 启用 sequentialMode | ~5 |
| **小计** | | ~165 |

### Phase 3 — Impl plan 同步（commit 3）

| 文件 | 改动 |
|---|---|
| `docs/design/cmcc-tdlte-v2.3-implementation-plan-20260521.md` | R-4.3 状态从 ⚪ 改 ✅；Gap P1 清单移除 R-4.3 |
| 注释 console_executor.go L43-47 | 加注释说明 R-4.3 在 ConsoleService + Sequencer 协作实现 |

**总计**：3 commits，~310 LOC（实现 ~160 + 测试 ~140 + 文档 ~10）

---

## 6. 验证计划

### 6.1 单元测试覆盖

- **ConsoleService.executeStatement(ADD with values)** → 返回 2 entries，第 2 entry 含 `{NEW}` 占位符
- **ConsoleService.executeStatement(ADD without values)** → 返回 1 entry（无回归）
- **buildSpvParamsWithNewPlaceholder** 边界：
  - sub_field path 与 targetObject `{i}` 数不一致 → error
  - instanceSelectors 不全 → error
  - 多 sub_field path 转换 → 全部含 `{NEW}` 且 prefix 一致
- **Sequencer.buildNextRequest** 边界：
  - prevTask 是 AddObject + nextCmd 是 SPV + has instance_number → 替换成功
  - prevTask 无 instance_number → 写警告 + return error（chain break）
  - prevTask 是 LST/MOD/RMV → 跳过替换逻辑（无回归）
- **extractInstanceNumber** 边界：
  - result JSON 含 `instance_number: 5` → 5, true
  - result 缺字段 → 0, false
  - result 是 null 或非 JSON → 0, false

### 6.2 集成验证

实施完成后通过：
- `docker compose build app + restart`
- 浏览器 `localhost:8081/mml/console` 选含 `{i}` 命令（如 PLMNList ADD），填 PLMNID = "46000"
- 提交 → 任务列表看到 2 个 device_tasks：AddObject 成功（instance #N） + SetParameterValues 成功
- 设备端查 path 验证 PLMNID 已写入新实例

---

## 7. 风险与回退

| 风险 | 评估 | 缓解 / 回退 |
|---|---|---|
| Sequencer hook 影响非 ADD 链路 | 中 — 改 buildNextRequest 加参数 | 严格 if prevTask.RpcMethod=='AddObject' 才走新分支；测试覆盖回归 |
| SequentialMode 默认 OFF 影响 ADD with values | 低 — fanout sequentialMode 已有 toggle | 实施时显式启用 / commit 2 加 deferred ticket |
| AddObject 成功但 SPV 失败导致"半完成"状态 | 中 — spec 未规定回滚 | task 列表显式区分 task A 成功 / task B 失败；用户能看到 |
| InstanceNumber 解析失败 | 低 — acs/handler.go 已实现，dev 环境不易复现 | 写警告日志 + chain break；task A 状态正常 |
| 多层嵌套 ADD（嵌套实例） | 低 — spec 未涉及，目前所有 ADD 都是单层新实例 | 不在本期范围；future spec 升级时处理 |

**回退路径**（如果上线后发现严重问题）：
- Revert commit 1 + commit 2 → ADD 退回 1 RPC 仅 AddObject（与现状等价）
- 无 schema / data 残留
- task 列表中 commit 期间产生的孤立 task B 行：用 admin SQL 标 `status='superseded_by_revert'` 不显示

---

## 8. 不在范围

- ❌ 失败回滚（自动 DeleteObject）— spec 未要求
- ❌ 嵌套 ADD（一次创建多层实例）— spec 未涉及
- ❌ 批量 ADD（一次创建多个实例）— spec 未涉及
- ❌ ADD 与 R-4.4 RMV 的合并优化 — 互不影响
- ❌ Customized 模板的 ADD 复合 — 待 admin UI 单独立项（R-5 隔离）

---

## 9. 待审决策点

1. **`{NEW}` 占位符字面 vs 结构化字段**：选字面 `"{NEW}"` 字符串（推荐：观察友好），或单独 `new_instance_placeholder: true` 元数据字段？
2. **失败回滚策略**：AddObject 成功 + SPV 失败时，**不自动回滚**（推荐）或**自动 DeleteObject**？
3. **SequentialMode 启用**：本期实施时**仅对 Console 通道启用**，还是**全局打开**（影响 MML 脚本任务）？
4. **commit 拆分**：backend ConsoleService / Sequencer / impl plan 三个 commit（推荐），还是合并？
5. **测试投资**：表驱动单测覆盖 ConsoleService + Sequencer 两侧（推荐），还是只一侧？

---

## 10. 实施前置确认（reviewer checklist）

- [ ] 同意 Option A（不破"1 MML=1 RPC"决策）的解读？
- [ ] 同意 `{NEW}` 字面占位符语义？
- [ ] 同意失败处理矩阵（不自动回滚）？
- [ ] 同意三阶段 commit 拆分？
- [ ] 同意单测覆盖范围？

读完后告诉我：
- **GO** → 按本设计实施 Phase 1 → 2 → 3
- 或具体调整哪个决策点
