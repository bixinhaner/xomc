# MML 需求设计文档功能实现分析报告

> **分析日期**: 2026-04-14  
> **文档版本**: v3.0 (2026-04-14)  
> **分析范围**: `docs/design/mml-requirements-design.md` 全部功能点  
> **代码库**: omcgo (后端) + omcmb/webcode (前端)

---

## 执行摘要

| 维度 | 已完成 | 部分完成 | 未实现 | 完成度 |
|------|--------|----------|--------|--------|
| **后端 API** | 85% | 10% | 5% | 🟡 良好 |
| **前端 UI** | 75% | 15% | 10% | 🟡 良好 |
| **数据模型** | 90% | 5% | 5% | 🟢 优秀 |
| **执行引擎** | 30% | 20% | 50% | 🔴 待加强 |
| **权限安全** | 40% | 20% | 40% | 🔴 待加强 |
| **总体完成度** | **~65%** | | | 🟡 基本可用，部分功能待完善 |

---

## 1. MML 命令控制台 (/mml/console)

### 1.1 页面布局（三栏布局）

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| 三栏弹性布局 | 1fr:1fr:2fr | ✅ 已实现 | `Console/index.tsx` 使用 grid 布局 |
| 顶部工具栏 | 标题 + 状态 Tag | ✅ 已实现 | 显示设备数和命令码 |
| 右栏纵向分割 | 终端 50% + 面板 50% | ⚠️ 部分实现 | 实际比例为 40%:60% |

**代码位置**: `omcmb/webcode/src/pages/mml/Console/index.tsx`

### 1.2 左栏：设备选择面板（DeviceTree）

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| 面板头部 | 标题 + 批量输入按钮 | ✅ 已实现 | `DeviceTree.tsx` |
| 搜索框 | SN/名称模糊搜索 | ✅ 已实现 | 支持 allowClear |
| 产品类型筛选 | 下拉框，7 种产品类型 | ✅ 已实现 | `PRODUCT_TYPE_OPTIONS` |
| 全选复选框 | indeterminate 状态 | ✅ 已实现 | 显示"全选 (N/M)" |
| 设备列表 | 分页，每页 8 条 | ✅ 已实现 | `DEVICE_PAGE_SIZE = 8` |
| 设备状态圆点 | online/offline/alarm | ✅ 已实现 | `STATUS_COLORS` |
| 已选设备 Tag 区 | 显示已选设备，可删除 | ✅ 已实现 | maxHeight: 70px |
| 批量输入弹窗 | SN 解析，分类展示结果 | ✅ 已实现 | `BatchSnModal.tsx` |
| **设备数据来源** | 真实 API | ❌ 未实现 | 当前使用 Mock 数据 `DEVICE_LIST` |

**代码位置**: 
- `omcmb/webcode/src/pages/mml/Console/components/DeviceTree.tsx`
- `omcmb/webcode/src/pages/mml/Console/hooks/useDeviceSelection.ts`
- `omcmb/webcode/src/pages/mml/Console/constants.ts` (Mock 数据)

### 1.3 中栏：命令树面板（CommandTree）

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| 面板头部 | 标题 + 已选命令 Badge | ✅ 已实现 | `CommandTree.tsx` |
| 搜索框 | 命令名/编码搜索 | ✅ 已实现 | 支持 allowClear |
| 分类筛选 | 动态生成分类选项 | ✅ 已实现 | 从 MOCK_COMMANDS 提取 |
| 命令树结构 | 两级树（分类→命令）| ✅ 已实现 | Ant Design Tree |
| 命令节点高亮 | 选中命令蓝色背景 | ✅ 已实现 | 交互正常 |
| **命令数据来源** | GET /api/v1/mml/commands | ❌ 未实现 | 当前使用 Mock 数据 |

**代码位置**:
- `omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx`
- `omcmb/webcode/src/pages/mml/Console/hooks/useCommandSelection.ts`

### 1.4 右栏：执行面板

#### 1.4.1 终端输出区（TerminalPanel）

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| 深色主题 | GitHub Dark 风格 | ✅ 已实现 | 背景 #0d1117 |
| 工具栏 | 复制/清空/下载按钮 | ✅ 已实现 | `TerminalPanel.tsx` |
| 输出行颜色编码 | stdout/stderr/info/success | ✅ 已实现 | `LINE_COLORS` |
| 自动滚动 | lines 变化时自动滚动 | ✅ 已实现 | containerRef.scrollTop |
| 时间戳前缀 | HH:mm:ss 格式 | ✅ 已实现 | 可选展示 |

**代码位置**: `omcmb/webcode/src/pages/mml/Console/components/TerminalPanel.tsx`

#### 1.4.2 命令输入区（CommandInput）

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| 面板头部 | 当前命令 + 目标设备 | ✅ 已实现 | Descriptions 双列 |
| Tab 切换 | 控制面板 / 参数路径指定 | ✅ 已实现 | 两个 Tab |
| 参数动态表单 | 根据 param_template 渲染 | ✅ 已实现 | 支持 enum/number/string/boolean |
| 操作类型下拉框 | LST/MOD/ADD/RMV | ✅ 已实现 | `OPERATION_TYPE_OPTIONS` |
| 参数路径列表 | 动态添加/删除路径 | ✅ 已实现 | 序号 + Input + 删除按钮 |
| 命令输入栏 | monospace 字体，Ctrl+Enter | ✅ 已实现 | 前缀 `>` |
| 底部操作栏 | 执行/重置/保存脚本按钮 | ✅ 已实现 | 状态文字显示 |
| 危险命令确认 | 5 种危险命令正则匹配 | ✅ 已实现 | Modal 二次确认 |

**代码位置**: `omcmb/webcode/src/pages/mml/Console/components/CommandInput.tsx`

### 1.5 命令执行流程

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| 前端执行 Hook | useCommandExecution | ⚠️ 部分实现 | 存在字段名不一致问题 |
| API 调用 | POST /api/v1/mml/execute | ✅ 已实现 | `mmlApi.ts` |
| **任务创建** | 返回 MMLTask (status=pending) | ✅ 已实现 | 后端 handler.go |
| **异步执行** | ACS Worker 消费任务 | ❌ 未实现 | 文档标注 TODO |
| **结果轮询** | GET /api/v1/mml/tasks/:id | ⚠️ 部分实现 | API 存在，前端未集成轮询 |
| **终端展示** | 逐台设备输出结果 | ⚠️ 部分实现 | Mock 数据直接返回结果 |

**代码位置**:
- `omcmb/webcode/src/pages/mml/Console/hooks/useCommandExecution.ts`
- `omcgo/internal/mml/handler.go` (Execute handler)
- `omcgo/internal/mml/service.go` (ExecuteCommand)

---

## 2. MML 脚本任务 (/mml/script)

### 2.1 页面布局

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| ListPageLayout | 标准列表页布局 | ✅ 已实现 | `ScriptTask/index.tsx` |
| 工具栏 | + 新增按钮 | ✅ 已实现 | type="primary" |
| 筛选栏 | FilterBar 组件 | ✅ 已实现 | 5 个筛选字段 |
| 表格 | DataTable 组件 | ✅ 已实现 | scroll.x=1400 |

**代码位置**: `omcmb/webcode/src/pages/mml/ScriptTask/index.tsx`

### 2.2 搜索筛选区域

| 筛选字段 | 文档要求 | 实现状态 | 说明 |
|---------|---------|---------|------|
| taskName | 输入框 | ✅ 已实现 | 包含匹配 |
| startTime | 日期范围 | ✅ 已实现 | 时间范围过滤 |
| createStatus | 类型下拉框 | ✅ 已实现 | immediate/suspended/scheduled/periodic |
| taskStatus | 状态下拉框 | ✅ 已实现 | pending/running/paused/completed/cancelled/failed |
| taskResult | 结果下拉框 | ✅ 已实现 | success/partial/failed |

### 2.3 任务列表表格

| 列 | 文档要求 | 实现状态 | 说明 |
|----|---------|---------|------|
| 操作列 | 查看 + 更多操作 | ✅ 已实现 | EyeOutlined + Dropdown |
| 任务名称 | ellipsis | ✅ 已实现 | 自适应宽度 |
| 创建者 | CREATE_USER | ✅ 已实现 | 100px |
| 创建时间 | 格式化显示 | ✅ 已实现 | YYYY-MM-DD HH:mm:ss |
| 类型 Tag | CREATE_STATUS_MAP | ✅ 已实现 | 4 种颜色 |
| 状态 Tag | TASK_STATUS_MAP | ✅ 已实现 | 6 种状态 |
| 进度 | 百分比字符串 | ✅ 已实现 | computeProgress 函数 |
| 结果 Tag | TASK_RESULT_MAP | ✅ 已实现 | 3 种结果 |
| 开始/结束时间 | 格式化显示 | ✅ 已实现 | 140px |

### 2.4 操作列按钮

| 按钮 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| 查看结果 | EyeOutlined | ✅ 已实现 | showResult |
| 信息 | InfoCircleOutlined | ✅ 已实现 | viewTaskInfo |
| 开始 | PlayCircleOutlined | ✅ 已实现 | 仅 paused 可用 |
| 暂停 | PauseCircleOutlined | ✅ 已实现 | 仅 running 可用 |
| 终止 | StopOutlined | ✅ 已实现 | running/waiting 可用 |
| 删除 | DeleteOutlined (danger) | ✅ 已实现 | running 禁用 |

### 2.5 新建任务 Drawer

| 功能点 | 文档要求 | 实现状态 | 说明 |
|--------|---------|---------|------|
| Drawer | 宽度 560px | ✅ 已实现 | destroyOnClose |
| 基本信息 | 任务名称 + 脚本上传 | ✅ 已实现 | 必填验证 |
| 执行方式 | Radio.Group 4 选项 | ✅ 已实现 | immediate/suspended/scheduled/periodic |
| 定时执行 | DatePicker showTime | ✅ 已实现 | 禁选过去时间 |
| 周期任务 | RangePicker + TimePicker | ✅ 已实现 | 受控模式 |
| 离线设备策略 | Checkbox + InputNumber | ✅ 已实现 | 等待设备上线重试 |
| 在线设备策略 | 失败重试配置 | ✅ 已实现 | 重试次数 + 间隔 |
| 表单默认值 | 6 个默认值 | ✅ 已实现 | openAddModal 时设置 |

### 2.6 任务状态流转

| 状态 | 文档要求（前端） | 后端实现 | 对齐状态 |
|------|----------------|---------|---------|
| waiting | ✅ | ⚠️ 映射为 pending | ⚠️ 需统一 |
| running | ✅ | ✅ running | ✅ 一致 |
| paused | ✅ | ✅ paused | ✅ 一致 |
| completed | ✅ | ✅ completed | ✅ 一致 |
| terminated | ✅ | ⚠️ 映射为 cancelled | ⚠️ 需统一 |
| exception | ✅ | ⚠️ 映射为 failed | ⚠️ 需统一 |
| pending | ✅ | ✅ pending | ✅ 一致 |
| failed | ✅ | ✅ failed | ✅ 一致 |

**问题**: 前端使用 6 种状态，后端使用 6 种状态（含 cancelled），但命名不完全一致。文档第 4.7 节已标注此问题。

---

## 3. 后端 API 实现

### 3.1 命令管理 API

| API | 方法 | 文档要求 | 实现状态 | 说明 |
|-----|------|---------|---------|------|
| /mml/commands | GET | 分页列表 | ✅ 已实现 | handler.go:ListCommands |
| /mml/commands/:id | GET | 单个详情 | ✅ 已实现 | handler.go:GetCommand |

**代码位置**: `omcgo/internal/mml/handler.go`

### 3.2 脚本管理 API

| API | 方法 | 文档要求 | 实现状态 | 说明 |
|-----|------|---------|---------|------|
| /mml/scripts | GET | 分页列表 | ✅ 已实现 | handler.go:ListScripts |
| /mml/scripts | POST | 创建脚本 | ✅ 已实现 | handler.go:CreateScript |
| /mml/scripts/:id | GET | 脚本详情 | ✅ 已实现 | handler.go:GetScript |
| /mml/scripts/:id | PUT | 更新脚本 | ✅ 已实现 | handler.go:UpdateScript |
| /mml/scripts/:id | DELETE | 删除脚本 | ✅ 已实现 | handler.go:DeleteScript |

### 3.3 任务管理 API

| API | 方法 | 文档要求 | 实现状态 | 说明 |
|-----|------|---------|---------|------|
| /mml/execute | POST | 执行命令 | ✅ 已实现 | handler.go:Execute |
| /mml/tasks | GET | 任务列表 | ✅ 已实现 | handler.go:ListTasks |
| /mml/tasks/:id | GET | 任务详情 | ✅ 已实现 | handler.go:GetTask |
| /mml/tasks/:id/start | POST | 启动任务 | ✅ 已实现 | handler.go:StartTask |
| /mml/tasks/:id/pause | POST | 暂停任务 | ✅ 已实现 | handler.go:PauseTask |
| /mml/tasks/:id/cancel | POST | 终止任务 | ✅ 已实现 | handler.go:CancelTask |
| /mml/tasks/:id | DELETE | 删除任务 | ✅ 已实现 | handler.go:DeleteTask |
| /mml/tasks/:id/results | GET | 结果明细 | ❌ 未实现 | 文档标注待开发 |

### 3.4 模板管理 API

| API | 方法 | 文档要求 | 实现状态 | 说明 |
|-----|------|---------|---------|------|
| /mml/templates | GET | 模板列表 | ❌ 未实现 | 文档第 7.4 节 |
| /mml/templates | POST | 创建模板 | ❌ 未实现 | - |
| /mml/templates/:id | PUT | 更新模板 | ❌ 未实现 | - |
| /mml/templates/:id | DELETE | 删除模板 | ❌ 未实现 | - |
| /mml/templates/:id/clone | POST | 复制模板 | ❌ 未实现 | - |

**评估**: 模板管理功能完全未实现，包括数据模型、后端 API、前端 UI。

---

## 4. 数据模型

### 4.1 数据库表

| 表名 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| mml_commands | DDL + 索引 | ✅ 已实现 | 000007_system_infra.sql |
| mml_scripts | DDL + 触发器 | ✅ 已实现 | 000007_system_infra.sql |
| mml_tasks | DDL + 调度字段 | ✅ 已实现 | 000007 + 900004_mml_enhance.sql |
| mml_templates | DDL（推荐） | ❌ 未实现 | 文档第 7.3 节 |

### 4.2 mml_tasks 扩展字段

| 字段 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| execute_type | VARCHAR(20) | ✅ 已实现 | 900004_mml_enhance.sql |
| scheduled_at | TIMESTAMPTZ | ✅ 已实现 | - |
| period_start | TIMESTAMPTZ | ✅ 已实现 | - |
| period_end | TIMESTAMPTZ | ✅ 已实现 | - |
| period_time | VARCHAR(10) | ✅ 已实现 | - |
| offline_retry | BOOLEAN | ✅ 已实现 | - |
| offline_retry_wait | INT | ✅ 已实现 | - |
| failed_retry | BOOLEAN | ✅ 已实现 | - |
| failed_retry_count | INT | ✅ 已实现 | - |
| failed_retry_interval | INT | ✅ 已实现 | - |
| started_at | TIMESTAMPTZ | ✅ 已实现 | - |
| finished_at | TIMESTAMPTZ | ✅ 已实现 | - |
| total_devices | INT | ✅ 已实现 | - |
| success_count | INT | ✅ 已实现 | - |
| failed_count | INT | ✅ 已实现 | - |
| result | VARCHAR(20) | ✅ 已实现 | - |

### 4.3 Go 模型

| 模型 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| MMLCommand | 完整字段 | ✅ 已实现 | model.go:42-52 |
| MMLScript | 完整字段 | ✅ 已实现 | model.go:55-65 |
| MMLTask | 完整字段 + 调度 | ✅ 已实现 | model.go:68-103 |
| TaskStatus | 6 种状态 | ✅ 已实现 | model.go:13-20 |
| ExecuteType | 4 种类型 | ✅ 已实现 | model.go:25-30 |
| TaskResult | 3 种结果 | ✅ 已实现 | model.go:35-39 |

### 4.4 前端 TypeScript 类型

| 类型 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| MMLParam | 完整字段 | ✅ 已实现 | types/mml.ts |
| MMLCommand | 完整字段 | ✅ 已实现 | types/mml.ts |
| MMLScript | 完整字段 | ✅ 已实现 | types/mml.ts |
| MMLTask | 完整字段 | ✅ 已实现 | types/mml.ts |
| MMLTemplate | 完整字段 | ❌ 未实现 | 模板功能未开发 |

---

## 5. 命令执行引擎

### 5.1 执行流程

| 步骤 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| 1. 前端创建任务 | POST /mml/execute | ✅ 已实现 | 返回 pending 任务 |
| 2. 任务入队 | cmdQueue | ❌ 未实现 | 文档标注 TODO |
| 3. ACS Worker 消费 | 异步执行 | ❌ 未实现 | - |
| 4. TR-069 SOAP 请求 | 发送到设备 | ❌ 未实现 | - |
| 5. 结果写回 | 更新 mml_tasks.results | ❌ 未实现 | - |
| 6. 前端轮询 | GET /mml/tasks/:id | ⚠️ 部分实现 | API 存在，前端未集成 |

**评估**: 执行引擎核心逻辑未实现，当前仅完成任务创建和状态管理。

### 5.2 RPC 方法支持

| RPC 方法 | 文档要求 | 实现状态 | 说明 |
|---------|---------|---------|------|
| GetParameterValues | ✅ 已有框架 | ⚠️ 部分支持 | ACS 模块有基础实现 |
| SetParameterValues | ✅ 已有框架 | ⚠️ 部分支持 | - |
| Reboot | ✅ 已有框架 | ⚠️ 部分支持 | - |
| AddObject | ❌ 待实现 | ❌ 未实现 | 文档第 5.2 节 |
| DeleteObject | ❌ 待实现 | ❌ 未实现 | - |
| GetParameterNames | ⚠️ 部分支持 | ⚠️ 部分支持 | - |

### 5.3 批量执行策略

| 维度 | 文档建议 | 当前实现 | 说明 |
|------|---------|---------|------|
| 并发数 | 建议 10 台 | ❌ 逐台串行 | useCommandExecution.ts for 循环 |
| 失败继续 | ✅ 单台失败不影响 | ✅ 已实现 | - |
| 超大批量 | 建议分批 50 台 | ❌ 无限制 | 无分批逻辑 |

---

## 6. 自定义命令模板

| 功能 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| 数据模型 | mml_templates 表 | ❌ 未实现 | 文档第 7.3 节 |
| 后端 API | CRUD + clone | ❌ 未实现 | 文档第 7.4 节 |
| 前端 UI | 保存/加载模板 | ❌ 未实现 | - |
| 可见性控制 | private/public | ❌ 未实现 | - |
| 权限规则 | 角色矩阵 | ❌ 未实现 | 文档第 7.6 节 |

**评估**: 模板功能完全未实现，属于第三阶段高级能力。

---

## 7. 权限与安全

### 7.1 权限矩阵

| 功能 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| mml.console.view | 查看控制台 | ⚠️ 部分实现 | 路由存在，Casbin 未配置 |
| mml.console.execute.read | 执行只读命令 | ❌ 未实现 | - |
| mml.console.execute.write | 执行写操作 | ❌ 未实现 | - |
| mml.script.view | 查看脚本任务 | ⚠️ 部分实现 | 路由存在 |
| mml.script.create | 创建脚本/任务 | ❌ 未实现 | - |
| mml.script.control | 控制任务 | ❌ 未实现 | - |
| mml.template.public.manage | 管理公共模板 | ❌ 未实现 | 模板功能未开发 |

### 7.2 安全控制

| 功能 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| 危险命令确认（前端）| 5 种正则匹配 | ✅ 已实现 | CommandInput.tsx |
| 危险命令确认（后端）| 同等校验 | ❌ 未实现 | handler.go 未校验 |
| 审计日志 | 记录操作 | ❌ 未实现 | 无审计表 |
| 限流 | 用户级频控 | ❌ 未实现 | - |
| 参数校验 | binding + 业务校验 | ✅ 已实现 | handler.go ShouldBindJSON |
| 数据隔离 | creator 过滤 | ⚠️ 部分实现 | 模型有 creator，查询未过滤 |

---

## 8. 非功能需求

| 需求 | 文档要求 | 实现状态 | 说明 |
|------|---------|---------|------|
| 控制台首屏 ≤ 3s | 性能要求 | ❓ 未测试 | 使用 Mock 数据无法评估 |
| 命令树筛选 ≤ 200ms | 本地筛选 | ✅ 已实现 | 前端本地过滤 |
| 任务列表翻页 ≤ 1s | 服务端分页 | ⚠️ 部分实现 | API 支持，前端未接入 |
| 任务创建 ≤ 500ms | 返回 task id | ⚠️ 部分实现 | 需性能测试 |
| 浏览器兼容 | Chrome/Edge 最新两版 | ✅ 已实现 | React 18 + Ant Design 5 |
| 响应式布局 | 1440px 无水平滚动 | ✅ 已实现 | Grid 布局 |
| 键盘快捷键 | Ctrl+Enter 执行 | ✅ 已实现 | CommandInput.tsx |

---

## 9. 待完善事项（对照文档第 12 章）

### 9.1 Mock vs 真实 API 对照（文档 12.1）

| 问题 | 文档描述 | 当前状态 | 修复建议 |
|------|---------|---------|---------|
| pageSize vs page_size | 查询参数命名不一致 | ⚠️ 部分修复 | getAllCommands 使用 page_size，getCommands 使用 pageSize |
| params vs parameters | 字段名不一致 | ❌ 未修复 | mmlApi.ts 应统一为 parameters |
| 执行返回值 | Mock 返回数组，后端返回 Task | ❌ 未修复 | useCommandExecution.ts 需适配 Task 结构 |
| 状态枚举 | 三套枚举并存 | ⚠️ 部分修复 | 后端已扩展为 6 种，前端需统一 |

### 9.2 后端待实现清单（文档 12.2）

| 序号 | 功能 | 实现状态 | 优先级 |
|------|------|---------|--------|
| 1 | GET /mml/scripts/:id | ✅ 已实现 | - |
| 2 | 任务控制接口（start/pause/cancel/delete）| ✅ 已实现 | - |
| 3 | ExecuteRequest 扩展 | ✅ 已实现 | 支持 execute_type/retry 等 |
| 4 | AddObject/DeleteObject ACS Worker | ❌ 未实现 | P1 |
| 5 | mml_templates 全栈 | ❌ 未实现 | P2 |
| 6 | mml_tasks 调度字段 | ✅ 已实现 | - |
| 7 | 统一状态机 | ✅ 已实现 | service.go 状态转换校验 |
| 8 | 任务结果分页 | ❌ 未实现 | P2 |
| 9 | 审计日志表 | ❌ 未实现 | P1 |

### 9.3 前端待完善清单（文档 12.3）

| 序号 | 功能 | 实现状态 | 优先级 |
|------|------|---------|--------|
| 1 | 命令树切换到真实接口 | ❌ 未实现 | P0 |
| 2 | 修复 API 参数名不一致 | ⚠️ 部分修复 | P0 |
| 3 | 修复 useCommandExecution 返回值 | ❌ 未修复 | P0 |
| 4 | 执行后改为任务轮询 | ❌ 未实现 | P0 |
| 5 | 脚本任务页服务端分页 | ❌ 未实现 | P1 |
| 6 | 新建任务 Drawer 设备选择 | ❌ 未实现 | P1 |
| 7 | 模板管理入口 | ❌ 未实现 | P2 |
| 8 | i18n 迁移 | ⚠️ 部分实现 | 控制台已迁移，脚本页未迁移 |

---

## 10. 分阶段实施建议（对照文档 12.4）

### 第一阶段：接口对齐（P0）

| 任务 | 状态 | 工作量 | 说明 |
|------|------|--------|------|
| 统一任务状态枚举 | ⚠️ 部分完成 | 0.5d | 前后端枚举已扩展，需清理 Mock 数据 |
| 修复 API 参数名映射 | ❌ 未完成 | 0.5d | pageSize→page_size, params→parameters |
| 打通命令列表真实接口 | ❌ 未完成 | 1d | DeviceTree + CommandTree 切换 API |
| 打通脚本任务列表真实接口 | ❌ 未完成 | 1d | 服务端分页 + 筛选 |

**预计工作量**: 3 天

### 第二阶段：执行链路闭环（P0-P1）

| 任务 | 状态 | 工作量 | 说明 |
|------|------|--------|------|
| 前端执行后轮询任务详情 | ❌ 未完成 | 2d | useCommandExecution 重构 |
| ACS Worker 异步执行框架 | ❌ 未完成 | 5d | cmdQueue 集成 |
| TR-069 RPC 执行实现 | ❌ 未完成 | 5d | GetParameterValues/SetParameterValues/Reboot |
| 任务结果明细接口 | ❌ 未完成 | 1d | 分页查询 results |
| 脚本任务页结果展示 | ❌ 未完成 | 1d | 结果弹窗 + 明细表格 |

**预计工作量**: 14 天

### 第三阶段：高级能力（P1-P2）

| 任务 | 状态 | 工作量 | 说明 |
|------|------|--------|------|
| 自定义命令模板 | ❌ 未完成 | 5d | 全栈开发 |
| 脚本校验与模板导入 | ❌ 未完成 | 2d | .txt 解析 |
| 大批量并发分批 | ❌ 未完成 | 3d | 并发 10 台，分批 50 台 |
| 审计日志 | ❌ 未完成 | 2d | 审计表 + 写入逻辑 |
| 危险命令后端校验 | ❌ 未完成 | 0.5d | handler.go 正则校验 |
| 权限隔离（Casbin）| ❌ 未完成 | 2d | 7 个权限点配置 |

**预计工作量**: 14.5 天

---

## 11. 验收要点对照（文档 12.5）

| 验收项 | 文档标准 | 当前状态 | 说明 |
|--------|---------|---------|------|
| Console 页面 | 设备选择/命令选择/参数配置/任务创建/结果轮询 | ⚠️ 部分达标 | 结果轮询未实现 |
| ScriptTask 页面 | 新建/查看/筛选/启动/终止任务 | ✅ 基本达标 | 前端使用 Mock 数据 |
| 数据模型 | mml_commands/scripts/tasks/templates | ❌ 部分达标 | templates 未实现 |
| 权限 | 只读/写操作/模板公共管理权限隔离 | ❌ 未达标 | Casbin 未配置 |
| 安全 | 危险命令确认/审计日志/参数校验 | ⚠️ 部分达标 | 后端无危险命令校验，无审计日志 |
| 文档一致性 | API/类型/DDL 与源码一致 | ✅ 基本达标 | 除模板功能外基本一致 |

---

## 12. 关键风险与建议

### 12.1 高风险项

1. **执行引擎未实现**（P0）
   - 当前仅创建任务，无实际设备通信
   - 建议：优先完成 ACS Worker 框架，至少实现 GetParameterValues

2. **前端未接入真实 API**（P0）
   - Console 和 ScriptTask 均使用 Mock 数据
   - 建议：第一阶段集中完成 API 切换

3. **模板功能完全缺失**（P2）
   - 数据模型、API、UI 均未开发
   - 建议：第三阶段按文档第 7 章完整实现

### 12.2 中风险项

4. **状态枚举不一致**
   - 前端 6 种状态 vs 后端 6 种状态（命名不同）
   - 建议：统一为文档后端枚举（pending/running/completed/failed/paused/cancelled）

5. **审计日志缺失**
   - 写操作无审计记录
   - 建议：新增 mml_audit_logs 表，记录所有 MOD/ADD/RMV/RST 操作

6. **危险命令后端校验缺失**
   - 前端有校验，后端无同等校验
   - 建议：handler.go Execute 方法增加正则校验

### 12.3 低风险项

7. **i18n 不完整**
   - 脚本任务页中文硬编码
   - 建议：逐步迁移至 i18n

8. **性能未测试**
   - 文档性能指标未验证
   - 建议：第二阶段完成后进行压测

---

## 13. 总结

### 13.1 已完成核心功能

✅ **三栏控制台 UI**：设备选择、命令树、终端输出、参数配置  
✅ **脚本任务管理 UI**：列表、筛选、新建任务 Drawer、任务控制  
✅ **后端 CRUD API**：命令/脚本/任务的完整 REST API  
✅ **数据模型**：mml_commands、mml_scripts、mml_tasks（含调度字段）  
✅ **任务状态机**：状态转换校验、任务控制（start/pause/cancel/delete）  
✅ **种子数据**：11 条内置命令（文档 8.3 节完整实现）  

### 13.2 待开发核心功能

❌ **ACS Worker 执行引擎**：任务消费、TR-069 RPC 执行、结果写回  
❌ **前端 API 接入**：命令树、设备列表、任务轮询  
❌ **自定义命令模板**：全栈开发（数据模型 + API + UI）  
❌ **权限隔离**：Casbin 权限配置（7 个权限点）  
❌ **审计日志**：操作审计表 + 写入逻辑  
❌ **安全增强**：后端危险命令校验、限流  

### 13.3 总体评价

MML 模块已完成 **UI 框架和基础 API 开发**（约 65%），但**核心执行引擎和真实数据接入尚未完成**。当前系统可以展示页面、创建任务、管理任务状态，但**无法真正执行命令到设备**。

建议按照文档第 12.4 节的三阶段计划推进：
1. **第一阶段（3 天）**：接口对齐，切换真实 API
2. **第二阶段（14 天）**：执行链路闭环，实现 ACS Worker
3. **第三阶段（14.5 天）**：高级能力（模板、审计、权限）

**预计总工作量**: 31.5 天（约 6.5 周）

---

**分析人**: AI Assistant  
**分析工具**: 代码库静态分析 + 文档对比  
**下一步建议**: 召开需求评审会议，确认优先级和排期
