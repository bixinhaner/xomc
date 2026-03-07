# 技术栈推荐 (Tech Stack Recommendation)

> OMC 统一网管系统 - 附录

## 概述

基于 OMC 统一网管系统的业务需求、团队能力和行业趋势，本文推荐完整的前端技术栈方案。

---

## 技术栈总览

| 层级 | 技术选择 | 版本 | 用途 |
|------|---------|------|------|
| 框架 | React | 18+ | UI 框架 |
| 语言 | TypeScript | 5.x | 类型安全 |
| UI 组件库 | Ant Design | 5.x | 基础组件 |
| 高级组件 | ProComponents | Latest | 高级表格/表单/布局 |
| 状态管理 | Zustand | 4.x | 全局状态 |
| 数据请求 | TanStack Query | 5.x | 服务端状态管理 |
| 图表 | ECharts | 5.x | 数据可视化 |
| 地图 | AMap (高德地图) | 2.x | 地理信息 |
| 拓扑 | AntV G6 | 5.x | 网络拓扑图 |
| 路由 | React Router | 6.x | 前端路由 |
| 构建工具 | Vite | 5.x | 构建和开发 |
| 国际化 | react-intl | 6.x | 多语言 (预留) |
| HTTP 客户端 | Axios | 1.x | API 请求 |
| CSS 方案 | CSS-in-JS (antd) + CSS Modules | - | 样式方案 |
| 代码规范 | ESLint + Prettier | Latest | 代码质量 |
| 测试 | Vitest + Testing Library | Latest | 单元/集成测试 |

---

## 1. 框架: React 18+

### 选择理由

```
1. 生态最大
   - NPM 包最丰富
   - Ant Design 原生 React
   - 社区支持最广

2. 团队能力匹配
   - React 是最主流的前端框架
   - 招聘 React 开发者更容易

3. 技术成熟度
   - React 18 引入 Concurrent Mode, 渲染性能更好
   - Suspense + Lazy 支持代码分割
   - Server Components 提供未来升级路径

4. TypeScript 集成
   - React + TypeScript 是行业标准组合
   - 类型推导和 IDE 支持最好
```

### 版本要求

```
React: >= 18.2.0
React DOM: >= 18.2.0
Node.js: >= 18.0.0 (开发环境)
```

---

## 2. UI 组件库: Ant Design 5.x

```
详见 03-component-library-evaluation.md

核心包:
  antd: ^5.x                    # 基础组件库
  @ant-design/pro-components    # 高级组件 (ProTable, ProForm, ProLayout)
  @ant-design/icons             # 图标库
  @ant-design/charts            # 图表组件 (可选, 或直接用 ECharts)
```

---

## 3. 状态管理: Zustand

### 选择理由

```
1. 轻量简洁
   - API 极简, 学习成本几乎为零
   - 包体积 < 2KB (gzipped)

2. 适合 OMC 场景
   - 全局状态不多 (用户信息、主题、通知、任务面板)
   - 大部分状态是服务端数据 (由 TanStack Query 管理)
   - 不需要 Redux 的复杂 middleware 体系

3. 与 React 完美集成
   - 基于 React hooks
   - 自动处理重渲染优化
   - 支持 DevTools
```

### 使用场景

```
Zustand 管理:
  - 用户认证状态 (token, userInfo, permissions)
  - UI 状态 (侧栏折叠、主题配置、通知列表)
  - 任务面板状态 (任务列表、展开/收起)
  - WebSocket 连接状态

TanStack Query 管理:
  - 所有服务端数据 (设备列表、告警列表、KPI 数据等)
  - 数据缓存和失效
  - 乐观更新
```

### 备选: Redux Toolkit

```
如果团队更熟悉 Redux, 推荐 Redux Toolkit (RTK):
  @reduxjs/toolkit + react-redux

RTK 优势:
  - 生态最大, 教程最多
  - RTK Query 内置数据请求方案
  - 适合复杂的全局状态管理

RTK 劣势:
  - Boilerplate 代码较多 (虽然 RTK 已大幅简化)
  - 包体积较大
  - 对于 OMC 的状态复杂度来说可能 over-engineering
```

---

## 4. 数据请求: TanStack Query (React Query)

### 选择理由

```
1. 服务端状态管理最佳实践
   - 缓存、失效、后台更新、乐观更新
   - 避免重复请求 (deduplication)
   - 窗口聚焦自动刷新

2. 完美支持 OMC 场景
   - Polling 轮询 (refetchInterval)
   - 分页查询 (useInfiniteQuery)
   - 并行查询 (useQueries)
   - Mutation + 自动刷新

3. 减少样板代码
   - 不需要手写 loading/error/data 状态
   - 不需要手写 useEffect 请求逻辑
```

### 使用示例

```typescript
// 设备列表查询
const { data, isLoading, error } = useQuery({
  queryKey: ['devices', { page, pageSize, filters }],
  queryFn: () => deviceApi.getList({ page, pageSize, ...filters }),
  staleTime: 30 * 1000, // 30s 内不重新请求
});

// 告警列表轮询
const { data: alarms } = useQuery({
  queryKey: ['alarms', 'current'],
  queryFn: alarmApi.getCurrentAlarms,
  refetchInterval: 30000, // 30s 轮询 (WebSocket 降级方案)
});

// 创建设备 Mutation
const createDevice = useMutation({
  mutationFn: deviceApi.create,
  onSuccess: () => {
    message.success('设备创建成功');
    queryClient.invalidateQueries({ queryKey: ['devices'] }); // 自动刷新列表
  },
});
```

---

## 5. 图表: ECharts 5.x

### 选择理由

```
1. 中国地图支持最好
   - 内置中国省市地图数据
   - 支持地图下钻
   - 高德地图/百度地图集成

2. 功能最丰富
   - 30+ 图表类型
   - 交互丰富 (缩放、框选、数据视图)
   - 动画效果好

3. 性能优秀
   - Canvas/SVG 双引擎
   - 大数据量优化 (dataset, sampling)
   - 增量渲染

4. 生态完善
   - 中文文档完善
   - 社区活跃
   - Apache 开源
```

### React 集成

```
推荐: echarts-for-react 或直接 useRef + init

// echarts-for-react 用法
import ReactECharts from 'echarts-for-react';

<ReactECharts
  option={chartOption}
  style={{ height: 400 }}
  notMerge
  lazyUpdate
/>
```

---

## 6. 地图: 高德地图 (AMap) JS API 2.0

### 选择理由

```
1. 中国地图数据最准确、最完整
2. API 文档中文, 技术支持好
3. 免费额度满足内部系统需求
4. 支持覆盖物、热力图、轨迹等功能
5. React 集成方案: @amap/amap-jsapi-loader 或 react-amap
```

---

## 7. 拓扑图: AntV G6

### 选择理由

```
1. 阿里出品, 与 Ant Design 设计体系一致
2. 专注图可视化 (关系图、拓扑图、流程图)
3. 内置布局算法 (Force、Dagre、Circular 等)
4. 支持大规模节点 (GPU 加速渲染)
5. 丰富的交互 (拖拽、缩放、框选、右键菜单)
6. TypeScript 类型完整
```

### 适用场景

```
- 网络拓扑图 (网元连接关系)
- 设备关系图 (设备-小区-邻区关系)
- 告警关联图 (告警传播路径)
```

---

## 8. 路由: React Router 6

### 关键特性

```
- 嵌套路由 (Nested Routes) 适合 OMC 多层级页面
- 数据加载 (Loader) 路由级数据预取
- 延迟加载 (Lazy) 路由级代码分割
- URL 参数同步 (筛选条件持久化)
```

---

## 9. 构建工具: Vite

### 选择理由

```
1. 开发体验极佳
   - 冷启动 < 1s (ESM 按需编译)
   - HMR 热更新 < 100ms
   - 相比 Webpack: 启动快 10-100x

2. 生产构建优秀
   - 基于 Rollup 打包
   - Tree Shaking
   - 代码分割
   - 压缩优化

3. 生态成熟
   - 官方插件丰富 (@vitejs/plugin-react)
   - 社区插件生态完善
   - 支持所有主流框架
```

---

## 10. 国际化: react-intl

### 选择理由

```
- ICU 消息格式标准
- 支持日期、数字、复数等复杂格式化
- 与 React 深度集成
- 首期以中文为主, 预留英文国际化能力

备选: i18next + react-i18next
  - 更灵活的翻译文件管理
  - 支持命名空间, 适合大型项目
  - 社区最大
```

---

## 11. 项目结构推荐

```
src/
├── api/                    # API 接口定义
│   ├── device.ts
│   ├── alarm.ts
│   └── ...
├── assets/                 # 静态资源 (图片、SVG)
├── components/             # 通用组件
│   ├── Layout/             # 全局布局
│   ├── TaskPanel/          # 任务面板
│   └── ...
├── hooks/                  # 自定义 Hooks
├── pages/                  # 页面组件 (按模块组织)
│   ├── dashboard/
│   ├── device/
│   ├── alarm/
│   ├── performance/
│   ├── config/
│   ├── software/
│   ├── backup/
│   ├── mml/
│   ├── mr/
│   └── system/
├── router/                 # 路由配置
├── store/                  # 全局状态 (Zustand)
├── styles/                 # 全局样式
├── theme/                  # Ant Design 主题配置
├── types/                  # TypeScript 类型定义
├── utils/                  # 工具函数
├── App.tsx
└── main.tsx
```

---

## 12. 浏览器兼容性

```
最低要求:
  Chrome: >= 90
  Firefox: >= 90
  Edge: >= 90
  Safari: >= 15

不支持:
  IE 11 (已停止支持)
  旧版移动端浏览器

注意:
  OMC 系统主要在 PC 端使用 (Chrome 为主)
  移动端适配为低优先级
```

---

## 13. 性能目标

```
首屏加载:
  - FCP (First Contentful Paint): < 1.5s
  - LCP (Largest Contentful Paint): < 2.5s
  - Bundle Size (gzipped): < 500KB (首屏)

运行时:
  - 表格渲染 1000 行: < 500ms
  - 路由切换: < 300ms
  - 搜索响应: < 100ms (前端) / < 500ms (后端)

优化策略:
  - 路由级代码分割 (React.lazy)
  - 组件级按需加载 (antd Tree Shaking)
  - 虚拟滚动 (大数据量表格)
  - 图片懒加载
  - 接口数据缓存 (TanStack Query)
```
