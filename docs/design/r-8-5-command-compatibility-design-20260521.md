# R-8.5 命令兼容性警告 — 设计文档

> Spec：`omcgo/规范/移动/南向数据模型/cmcc-tdlte-southbound-data-model-v2.3.md` §R-8.5
> 日期：2026-05-21
> 状态：**设计待审核**，实施分 2 个 commit（backend → frontend）

---

## 1. 需求摘要（spec 原文 + 解读）

> §R-8.5 与命令兼容性提示（可选 P2）：当用户已选中某 `product_class` + 命令，但该 `product_class` 经 `ProductRegistry` 解析到的 `product → param_model` 不包含该命令路径时，命令节点旁显示警告图标 + Tooltip "该产品类型不支持本命令"（不阻塞，仅提示；执行时由后端 Translator 缺失映射时报错）。

**核心规则**：

- 输入：当前选中的 `product_class`（DeviceTree 顶部下拉）
- 处理：对每条 command，检查其 `tree_node_refs`（或 `target_paths` fallback）是否全部位于该 product 对应的 `param_mappings.standardPath` 集合
- 输出：每条命令的 `unsupported: bool`
- UI：unsupported=true 的命令叶子尾部加 ⚠️ 黄色图标 + Tooltip
- 不阻塞：用户仍可点击 / 执行；最终由后端 Translator 兜底报错

---

## 2. 数据现状（dev DB 实测）

| 项 | 数据量 | 备注 |
|---|---|---|
| `products`（含 `param_model_id`） | 15 / 15 全部 | 数据健全 |
| `param_mappings` | 4782 行 / 9 distinct ParamModel | 充足 |
| `mml_commands.tree_node_refs` 非空 | 190 / 1183 | v2 catalog Loader 出来 |
| `mml_commands.target_paths` 非空 | 563 / 1183 | v1 catalog Loader 出来 |
| `product_class_patterns` | 8+ 正则 pattern | `^FAP/MLN/SC$` 等 |

**关键决策**：路径来源直接读 `tree_node_refs`（v2 唯一路径来源）。v1 `target_paths` 路径已随 v1 catalog 下线一起删除（2026-05-22）。

---

## 3. 架构决策

### 3.1 API 形态：独立新端点（不动 group-tree）

```
GET /api/v1/mml/console/command-compatibility?product_class=<class>&lang=<lang>

Response:
{
  "data": {
    "product_class": "FAP/MLN/SC",
    "product_id": "<uuid>",
    "param_model_id": "<uuid>",
    "unsupported_command_ids": ["<uuid1>", "<uuid2>", ...]
  }
}

Errors:
  404 — product_class 无任何 product 匹配
  400 — product_class 缺失
  500 — 内部错误
```

**为什么独立端点（而非扩 group-tree）**：
- 不动现有 group-tree 契约（缓存策略不变）
- 兼容性计算独立可缓存（per product_class）
- 易于 rollback（删端点即可）
- 与 P4.c reset 教训一致：blast radius 最小化

### 3.2 兼容性判定逻辑

```go
// pseudocode
supportedSet := makeStandardPathSet(paramRegistry.GetByParamModel(product.ParamModelID))
for _, cmd := range allCommands {
    paths := cmd.TreeNodeRefs
    if len(paths) == 0 {
        paths = cmd.TargetPaths
    }
    if len(paths) == 0 {
        continue  // 命令无路径声明 → 视为支持（不阻塞）
    }
    for _, p := range paths {
        if _, ok := supportedSet[p]; !ok {
            unsupported = append(unsupported, cmd.ID)
            break
        }
    }
}
```

**边界**：
- 命令 paths 全空 → 支持（不警告）
- product 找不到 → 404（前端降级为不显示警告）
- product.ParamModelID 是 nil → 返回所有 commands 为 unsupported（spec 行为）
- product 有 ParamModel 但 default mappings 全空 → 返回所有 commands 为 unsupported

### 3.3 数据源

**只查 default mappings**（`param_mappings`），不查 discovered（`discovered_param_mappings`）：

- 兼容性是"产品类型"级判断，与具体设备无关
- discovered 是 per-device per-software_version，无 product_class 维度
- spec 原文 "param_model 不包含该命令路径" 精确指 default

调用：`ParamRegistry.GetByParamModel(paramModelID) → MappingSet`

### 3.4 前端 productClassFilter 状态 lift

**当前**：`useDeviceSelection` 内部 `useState`，CommandTree 兄弟组件无法订阅。

**改造**：lift 到 `mmlConsoleStore`（Zustand）：

```typescript
// mmlConsoleStore.ts 新增
productClassFilter: string;
setProductClassFilter: (cls: string) => void;
```

**同步点**：`Console/index.tsx` 在 `dev.productTypeFilter` 变化时 `setProductClassFilter` 写入 store。useDeviceSelection 内部 useState 保留（设备列表逻辑用），单向同步到 store。

### 3.5 UI 形态

CommandTree 命令叶子 `renderOpLeafTitle()` 加分支：

```tsx
{unsupported && (
  <Tooltip title={t('mml.console.commandTree.unsupportedForProductClass')}>
    <WarningOutlined style={{ color: '#faad14' }} aria-label="unsupported" />
  </Tooltip>
)}
```

- Antd `WarningOutlined` 黄色 #faad14（警告语义）
- 位置：OP Tag → CodeOutlined → 命令名 → **WarningOutlined**（最末尾）
- Tooltip 文案 i18n key `mml.console.commandTree.unsupportedForProductClass`

### 3.6 缓存策略

- React Query staleTime: 5 分钟（product / ParamModel 数据稳定）
- queryKey 包含 productClass + lang，切产品类型自动重新 fetch
- 后端无显式缓存（依赖 ParamRegistry 已有的 L1 + Redis L2）

### 3.7 明确不做的事

❌ 不阻塞执行（spec 明文："不阻塞，仅提示"）
❌ 不在 sub-field path 层做警告（仅 command 级）
❌ 不动 Translator / DB schema / migration
❌ 不改现有 `/mml/group-tree` 契约
❌ 不查 discovered mappings（按 product 级判断）

---

## 4. 实施清单

### Phase 1 — Backend (commit 1)

| 文件 | 改动 | 估计 LOC |
|---|---|---|
| `omcgo/internal/mml/command_compatibility.go`（新建） | repo SQL + `ComputeCommandCompatibility` 纯函数 + 类型 | ~90 |
| `omcgo/internal/mml/console_service.go` | 加 `ProductRegistry` / `ParamRegistry` 依赖 + `GetCommandCompatibility(ctx, productClass, lang)` 方法 | ~40 |
| `omcgo/internal/mml/console_handler.go` | 加 `GetCommandCompatibility` handler + 路由注册 | ~30 |
| `omcgo/cmd/app/provider/modules.go` | `NewConsoleService` 调用加 2 个 registry 参数 | ~3 |
| `omcgo/internal/mml/command_compatibility_test.go`（新建） | 表驱动测试覆盖：normal / empty paths / product not found / param model nil / mixed v1+v2 paths | ~120 |

**总计**：~280 LOC，1 commit。

### Phase 2 — Frontend (commit 2)

| 文件 | 改动 | 估计 LOC |
|---|---|---|
| `omcmb/frontend-core/src/store/mmlConsoleStore.ts` | 加 `productClassFilter` + setter | ~5 |
| `omcmb/webcode/src/pages/mml/Console/index.tsx` | `dev.productTypeFilter` 变化时同步到 store | ~5 |
| `omcmb/frontend-core/src/services/api/mmlApi.ts` | `getCommandCompatibility(productClass, lang)` API client | ~20 |
| `omcmb/frontend-core/src/hooks/api/useMmlConsole.ts` | `useCommandCompatibility(productClass)` React Query hook | ~25 |
| `omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx` | 订阅 + 装饰命令叶子 `<WarningOutlined>` | ~30 |
| `omcmb/frontend-core/src/i18n/{zh-CN,en-US}/index.ts` | 1 个新 key `mml.console.commandTree.unsupportedForProductClass` | ~4 |
| `omcmb/webcode/src/pages/mml/Console/components/__tests__/CommandTree.test.tsx`（新或扩展） | 测试警告图标渲染 | ~30 |

**总计**：~120 LOC，1 commit。

---

## 5. 验证计划

### 5.1 Backend
- `go build ./...` 通过
- `go vet ./internal/mml/...` 通过
- `go test -race -count=1 ./internal/mml/...` 通过（含 command_compatibility_test.go 新单测）
- curl 验证：
  - `GET /api/v1/mml/console/command-compatibility?product_class=^FAP/MLN/SC$&lang=zh-CN` → 返回 unsupported_command_ids list
  - `GET /api/v1/mml/console/command-compatibility?product_class=NONEXISTENT` → 404

### 5.2 Frontend
- `cd omcmb/webcode && npm run typecheck` 通过
- 现有测试 0 回归
- 浏览器烟测（`http://localhost:8081/mml/console`）：
  1. 默认产品类型加载完，CommandTree 部分命令有 ⚠️ 图标
  2. 切换产品类型，警告图标随之更新（hook re-fetch）
  3. Hover ⚠️ → Tooltip "该产品类型不支持本命令" / "Not supported by this product class"
  4. 点击 unsupported 命令 → 不阻塞，可进入 Control Panel（spec 要求）

---

## 6. 风险与回退

| 风险 | 评估 | 回退路径 |
|---|---|---|
| 后端 SQL 性能：1183 行 × 路径数 × hash 查找 | 低 — 单次 ~120K 次 hash lookup，毫秒级 | 不需 |
| 前端 hook re-fetch 风暴：用户高频切产品类型 | 低 — React Query 5min cache 兜底 | 调大 staleTime |
| ParamRegistry 启动期未 ready（dictload 阻塞） | 低 — `Refresh()` 在 init 阶段已完成 | 早退 503 |
| `tree_node_refs` 与 `target_paths` 同时为空 | 兼容 — fallback 到 "视为支持" | 不需 |
| 用户切产品类型时 CommandTree 已 unmount | 低 — React Query 自动 abort | 不需 |

**回退路径**：
- 后端独立端点 → 直接删 handler + route + service method
- 前端独立 hook + UI 装饰 → 直接 revert CommandTree 改动 + hook 文件

不动 schema / migration，无遗留状态。

---

## 7. 与 spec 其他条款的关系

| 关联条款 | 关系 |
|---|---|
| R-8.3 单一性 | 切换 product_class 时清空设备 → CommandTree 兼容性查询也随之 re-key |
| R-8.4 后端兜底 | 与本条款独立 — R-8.4 是 execute-statements 入参校验，本条款是 UI 提示 |
| R-9.3 Translator passthrough | passthrough 时 spec 标"不阻塞"，与本条款一致 — 警告仅提示 |
| R-2.5 tree_node_refs | 本条款主依赖；fallback target_paths 保 v1 兼容 |

---

## 8. 待审决策点

请确认：

1. **API 路径**：`GET /mml/console/command-compatibility`？还是别的路径？
2. **路径 fallback 顺序**：tree_node_refs 优先 + target_paths 回退？还是仅认 tree_node_refs？
3. **UI 图标位置**：命令叶子末尾（在 OnReboot Tag 之后）？还是命令名前？
4. **commit 拆分**：backend / frontend 两个 commit 节奏？还是一个 commit？
5. **测试投资**：表驱动 backend 单测 + 前端 1-2 个渲染测试？还是只 backend？

读完后告诉我是 GO（按本设计实施 backend → frontend 两个 commit），还是需要调整哪些拐点。
