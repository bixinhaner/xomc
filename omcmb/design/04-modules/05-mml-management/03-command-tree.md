# MML 命令树 (Command Tree)

## 页面用途

定义 MML 命令控制台左栏中的命令导航树组件规格。树结构分为 eNB Configuration、Radio Configuration、Customized 三个顶级分类，每个分类下包含具体的 MML 命令。支持搜索、添加/删除自定义命令。

## 布局模板

内嵌于 MML 命令控制台左栏，位于设备选择器下方。

## 线框描述

```
┌──────────────────────────┐
│ 🔍 搜索命令码或名称...     │
│ ──────────────────────── │
│                          │
│ ▼ eNB Configuration      │
│   ├─ LST CELLPARA        │ ← 点击选中
│   ├─ MOD CELLPARA        │
│   ├─ ADD CELLPARA        │
│   ├─ DEL CELLPARA        │
│   ├─ LST ENODEB          │
│   ├─ MOD ENODEB          │
│   ├─ LST ALMDATA         │
│   ├─ LST BOARDSTATE      │
│   └─ ... (展开显示全部)    │
│                          │
│ ▶ Radio Configuration    │
│   ├─ LST FREQPARA        │
│   ├─ MOD FREQPARA        │
│   ├─ LST POWERPARA       │
│   ├─ MOD POWERPARA       │
│   └─ ...                 │
│                          │
│ ▼ Customized             │
│   ├─ MY_CMD_01           │ ← 用户自定义
│   └─ MY_CMD_02           │
│                          │
│ ──────────────────────── │
│ [+ 添加自定义] [- 删除]    │
└──────────────────────────┘
```

## 树组件规格

### 整体配置

| 属性           | 值                              |
|---------------|--------------------------------|
| 组件类型       | Ant Design Tree                |
| 高度           | flex: 1 (填充剩余空间)          |
| 溢出处理       | overflow-y: auto               |
| 默认展开       | 展开第一个分类                   |
| 选中模式       | 单选                           |
| 搜索           | 顶部搜索框, 实时过滤 (debounce 200ms) |

### 顶级分类 (Category Node)

| 分类名称            | 图标                    | 说明               |
|-------------------|-----------------------|---------------------|
| eNB Configuration | `SettingOutlined`     | eNB 配置相关命令     |
| Radio Configuration| `WifiOutlined`       | 射频配置相关命令      |
| Customized        | `StarOutlined`        | 用户自定义命令       |

### 分类节点样式

| 属性         | 值                                    |
|-------------|---------------------------------------|
| 字体         | 13px, font-weight: 600               |
| 颜色         | #262626                              |
| 图标尺寸     | 14px                                 |
| 展开/折叠图标 | ▼ (展开) / ▶ (折叠), 旋转动画 200ms   |
| 背景色       | transparent                           |
| Hover 背景   | #F5F5F5                              |
| 高度         | 32px                                 |

### 命令节点 (Command Node)

| 属性         | 默认状态                    | 选中状态                   |
|-------------|---------------------------|---------------------------|
| 字体         | 12px, regular              | 12px, medium              |
| 颜色         | #595959                    | #1890FF                   |
| 背景色       | transparent                | #E6F7FF                   |
| 高度         | 28px                       | 28px                      |
| 缩进         | padding-left: 24px         | padding-left: 24px        |
| Hover 背景   | #F5F5F5                    | -                         |
| 左侧指示条   | 无                         | 2px solid #1890FF         |

### 命令节点显示格式

```
LST CELLPARA          ← 命令码 (命令名称 hover 时 Tooltip 显示)
```

| 属性         | 值                               |
|-------------|----------------------------------|
| 主文字       | 命令码 (如 "LST CELLPARA")       |
| Tooltip      | 命令码 + 命令名称 (如 "LST CELLPARA - 查询小区参数") |
| 命令码过长时  | ellipsis 截断, 全文在 Tooltip     |

## 搜索功能

### 搜索框规格

| 属性         | 值                              |
|-------------|--------------------------------|
| 位置         | 树组件顶部                      |
| Placeholder  | "搜索命令码或名称..."           |
| 图标         | `SearchOutlined` 前缀           |
| 清除按钮     | 有内容时显示 × 清除按钮         |
| Debounce     | 200ms                          |

### 搜索行为

| 行为         | 说明                                      |
|-------------|------------------------------------------|
| 搜索范围     | 命令码 + 命令名称 (任一匹配即显示)          |
| 匹配方式     | 不区分大小写的模糊匹配                     |
| 结果展示     | 自动展开包含匹配命令的分类, 隐藏不匹配的节点  |
| 高亮匹配     | 匹配文字部分黄色高亮                        |
| 无匹配       | 显示 "未找到匹配的命令"                     |
| 清空搜索     | 恢复完整树结构                             |

## 自定义命令管理

### 添加自定义命令

| 属性     | 值                           |
|---------|------------------------------|
| 触发     | 点击底部 [+ 添加自定义] 按钮  |
| 弹窗标题  | 添加自定义命令               |
| 宽度     | 480px                       |

**表单字段：**

| 字段名称   | 组件类型   | 必填 | 校验规则                      |
|----------|---------|------|------------------------------|
| 命令码    | Input   | 是   | 非空, 大写字母+空格, 系统内不重复 |
| 命令名称  | Input   | 否   | 最大 64 字符                   |
| 命令脚本  | TextArea | 是  | 非空, MML 语法格式             |

### 删除自定义命令

| 属性     | 值                                   |
|---------|--------------------------------------|
| 触发     | 选中自定义命令后点击 [- 删除]          |
| 确认     | Popconfirm "确定删除此自定义命令？"    |
| 范围     | 仅 Customized 分类下的命令可删除       |
| 系统命令  | eNB Configuration / Radio 下的命令不可删除 |

## 数据接口

```typescript
// 获取命令树
GET /api/mml/command-tree?productType={type}

interface CommandTreeData {
  categories: CommandCategory[];
}

interface CommandCategory {
  key: string;              // 分类标识
  name: string;             // 分类名称
  icon: string;             // 图标名
  commands: CommandNode[];  // 命令列表
}

interface CommandNode {
  code: string;             // 命令码
  name: string;             // 命令名称
  isCustom: boolean;        // 是否自定义命令
}

// 添加自定义命令
POST /api/mml/commands/custom
{
  code: string;
  name: string;
  script: string;
}

// 删除自定义命令
DELETE /api/mml/commands/custom/{code}
```

## 状态处理

| 状态             | 处理方式                                |
|-----------------|----------------------------------------|
| 命令树加载中     | Skeleton 骨架 (3 组矩形条)              |
| 命令树加载失败   | "命令列表加载失败" + 重试按钮            |
| 搜索无结果       | "未找到匹配的命令" 文字提示              |
| 添加成功         | Toast "自定义命令已添加", 树自动刷新     |
| 删除成功         | Toast "已删除", 如删除的是当前选中项则清空中栏 |

## 跨模块导航

本组件为 MML 命令控制台的内嵌子组件，不涉及跨模块导航。选中命令后联动中栏显示命令详情。
