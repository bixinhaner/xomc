# MML 控制台重构计划

## 1. 目标

将当前 1319 行的单文件重构为 IDE 风格布局，提升代码可维护性和用户体验。

## 2. 当前问题

| 问题 | 影响 |
|------|------|
| 单文件 1319 行 | 维护困难，职责不清 |
| 两种模式（向导/专业）| 代码重复，切换不明显 |
| 组件未拆分 | 无法复用，测试困难 |

## 3. 新设计方案

### 3.1 布局结构

```
┌─────────────────────────────────────────────────────────────────┐
│  MML 控制台                              [设备: 3] [命令: LST]  │
├───────────────┬─────────────────────────────────────────────────┤
│               │                                                 │
│  📁 设备      │           终端输出                              │
│   ├─ 🔍 搜索  │  ┌─────────────────────────────────────────────┐│
│   ├─ eNB (5)  │  │ > LST BASIC_INFO                            ││
│   │  ├─ ENB01 │  │ --- 设备: ENB00001 ---                      ││
│   │  └─ ENB02 │  │   设备名称: 北京朝阳基站01                    ││
│   ├─ gNB (3)  │  │   状态: 在线                                 ││
│   └─ GSM (2)  │  │   ...                                        ││
│               │  └─────────────────────────────────────────────┘│
│ ───────────── │  ───────────────────────────────────────────────│
│               │                                                 │
│  📁 命令      │  参数配置 / 命令输入                             │
│   ├─ 总览     │  ┌─────────────────────────────────────────────┐│
│   │  └─ 基本信息│  │ 命令: LST BASIC_INFO                        ││
│   ├─ 快速设置 │  │ 参数: [无]                                   ││
│   └─ 参数配置 │  │                                             ││
│               │  │ > _                                          ││
│               │  └─────────────────────────────────────────────┘│
│               │  [执行] [保存脚本] [清空]                         │
└───────────────┴─────────────────────────────────────────────────┘
```

### 3.2 布局比例

- 左侧面板：20%（可折叠）
- 右侧上方：60%（终端输出）
- 右侧下方：40%（参数配置 + 命令输入）

## 4. 组件拆分计划

### 4.1 目录结构

```
src/pages/mml/Console/
├── index.tsx                    # 主入口，组装各组件
├── components/
│   ├── DeviceTree.tsx           # 设备选择树（带搜索、筛选）
│   ├── CommandTree.tsx          # 命令选择树（按分类）
│   ├── TerminalPanel.tsx        # 终端输出面板
│   ├── CommandInput.tsx         # 命令输入栏（支持 @ / 快捷输入）
│   ├── ParamConfigPanel.tsx     # 参数配置面板
│   ├── SelectionSummary.tsx     # 已选设备/命令摘要
│   └── BatchSnModal.tsx         # 批量输入SN弹窗
├── hooks/
│   ├── useDeviceSelection.ts    # 设备选择状态管理
│   ├── useCommandSelection.ts   # 命令选择状态管理
│   ├── useCommandExecution.ts   # 命令执行逻辑
│   └── useTerminalHistory.ts    # 终端历史记录
├── constants.ts                 # 常量（设备列表、命令定义）
└── types.ts                     # 类型定义
```

### 4.2 组件职责

| 组件 | 职责 | 行数估计 |
|------|------|---------|
| `index.tsx` | 组装布局，协调状态 | ~100 |
| `DeviceTree.tsx` | 设备搜索、筛选、多选 | ~150 |
| `CommandTree.tsx` | 命令分类树、搜索 | ~120 |
| `TerminalPanel.tsx` | 封装 TerminalOutput | ~80 |
| `CommandInput.tsx` | 命令输入、智能提示 | ~150 |
| `ParamConfigPanel.tsx` | 动态参数表单 | ~100 |
| `SelectionSummary.tsx` | 摘要展示 | ~60 |
| `BatchSnModal.tsx` | 批量输入弹窗 | ~80 |
| hooks (4个) | 状态管理逻辑 | ~200 |

**总计**: ~1040 行（比原 1319 行减少 21%，且结构清晰）

## 5. 实现步骤

### Phase 1: 基础架构（2h）
1. 创建目录结构
2. 提取常量和类型
3. 创建自定义 hooks

### Phase 2: 组件拆分（3h）
1. 实现 `DeviceTree.tsx`
2. 实现 `CommandTree.tsx`
3. 实现 `TerminalPanel.tsx`
4. 实现 `CommandInput.tsx`
5. 实现 `ParamConfigPanel.tsx`
6. 实现 `SelectionSummary.tsx`
7. 实现 `BatchSnModal.tsx`

### Phase 3: 主入口组装（1h）
1. 重构 `index.tsx`
2. 使用 CSS Grid 实现新布局
3. 协调各组件状态

### Phase 4: 测试与优化（1h）
1. 功能测试
2. 响应式适配
3. 键盘快捷键支持

## 6. 关键技术点

### 6.1 布局实现

```tsx
// 使用 CSS Grid 实现 IDE 风格布局
<div style={{
  display: 'grid',
  gridTemplateColumns: sidebarCollapsed ? '0 1fr' : '240px 1fr',
  height: '100%',
  gap: 12,
}}>
  {/* 左侧面板 */}
  <aside>
    <DeviceTree />
    <CommandTree />
  </aside>

  {/* 右侧区域 */}
  <main style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
    <TerminalPanel style={{ flex: '1 1 60%' }} />
    <ParamConfigPanel style={{ flex: '1 1 40%' }} />
  </main>
</div>
```

### 6.2 命令输入智能提示

```tsx
// 支持 @设备 /命令 语法
const parseInput = (input: string) => {
  const deviceMatch = input.match(/@(\S+)/g);  // @ENB00001
  const commandMatch = input.match(/\/(\S+)/g); // /LST
  return { devices: deviceMatch, command: commandMatch };
};
```

### 6.3 状态管理

```tsx
// 使用 useReducer 或 zustand 管理复杂状态
interface MMLState {
  devices: Device[];
  selectedDevices: string[];
  selectedCommand: MMLCommand | null;
  outputLines: TerminalLine[];
  paramValues: Record<string, unknown>;
}
```

## 7. 兼容性考虑

- 保留原有 API 接口
- 保留 i18n 键值
- 渐进式重构，不破坏现有功能

## 8. 预期收益

| 指标 | 当前 | 重构后 |
|------|------|--------|
| 单文件行数 | 1319 | ~100 |
| 组件数量 | 1 | 8+ |
| 代码复用性 | 低 | 高 |
| 可测试性 | 困难 | 容易 |
| 用户操作步骤 | 4步 | 2步 |
