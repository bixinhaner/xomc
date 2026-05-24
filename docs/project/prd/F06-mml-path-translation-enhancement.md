# PRD — MML Path 翻译方案完善（T-0168）

| 项 | 值 |
|---|---|
| PRD ID | F06-mml-path-translation-enhancement |
| Backlog | T-0168 |
| 功能域 | F06 (MML 控制台) + F02 (参数模型字典) + ops (Prometheus) |
| 优先级 | P2 |
| 状态 | DRAFT → REVIEW（S0 产出） |
| 起草日 | 2026-05-24 |
| 起草人 | Claude（用户 ULTRATHINK 4 轮校准） |
| 关联设计 | 本 PRD 附录"## 设计备忘"由 S2 阶段追加；不另起独立设计文档 |

---

## 1. 业务背景

MML 控制台允许运营商工程师在 Web UI 选择设备、勾选命令、提交后下发到 CPE。命令体里的"参数路径"在 OMC 内部用标准化路径（standardPath，TR-181 风格），但下发到 BAICELLS 等具体厂商 CPE 时必须翻译为厂商私有路径（privatePath，X_VENDOR_ 前缀）。

T-0123-P1 落地的 R-9.3 per-device 翻译机制已在 `internal/mml/console_validate.go:117 translateTaskPaths` 实现，链路为：

```
设备 product_class
  → ProductRegistry.MatchProductClass（正则匹配 product_class_patterns 表）
  → product.id
  → ParamRegistry.Translator(productID, swVersion)
  → Translator.ToPrivate(standardPath)
  → 写回 task.Commands 的 param_refs[].private_path + translation_source
```

机制已工作，但有 4 处缺口：
1. **productClass 解析失败时硬阻塞 fanout**（`mml_adapters.go:75-88` ErrOrphan → ErrProductClassUnresolved）。野设备 / 未登记 product_class 的设备完全无法用 MML，运维体验差。
2. **任务级翻译元数据未列存**。`mml_tasks` 表无 `product_resolved` / `matched_product_id` / `path_translation_source` 等字段，审计查询要 join + JSONB 解析，慢且容易漏。
3. **API 响应未把 per-path 翻译详情结构化**。前端只能拿到聚合的 `pathTranslationWarning`（"X 设备 Y 条 path 未翻译"），无法看到具体哪条 standardPath 翻译成什么 privatePath。
4. **前端只有聚合 Alert，没有详情入口**。任务详情 Modal 显 `paramPaths`（原始 standardPath）但不显 privatePath；也无独立的"路径转换详情"视图。
5. **缺 orphan 告警**。一旦激进路线开启 orphan_passthrough，野设备会静默累积，需 Prometheus 告警让运维兜底。

## 2. 用户故事

### 2.1 运营工程师 — 野设备紧急下发
> 作为一线运维工程师，当我新接入一台 product_class 还未在 OMC 注册（如新厂商首台 PoC 设备）时，**我希望 MML 命令能照常下发（原路径透传），并在任务详情明确看到"product 未识别 - 原路径下发"标记**，而不是被前端硬拦下"产品未识别，请先登记"，因为联调阶段我需要立刻试参数。

### 2.2 SRE — orphan 静默累积治理
> 作为 SRE，**我希望任何 orphan_passthrough 事件都触发 Prometheus 告警**，让我能 24h 内补登记产品；目前 R-9.3 静默生成 passthrough 但无指标，我不知道有多少野设备在用 MML。

### 2.3 网管管理员 — 翻译详情审计
> 作为网管管理员，**我希望在任务记录列表行直接展开看到每条 standardPath 翻译成什么 privatePath、来源（discovered/default/passthrough/orphan_passthrough）、状态 Tag**，目前我必须打开 task Modal、看 JSONB 字段或问开发，效率低。

## 3. 验收标准（Given-When-Then）

### GWT-1（激进路线 — orphan 不阻塞）
- **Given** 一台设备 `serial=TEST-ORPHAN-01`，product_class=`UNKNOWN-2025`，product_class_patterns 表无匹配条目
- **When** 通过 MML 控制台对该设备执行 `LST CARRIER`
- **Then**
  - HTTP 响应 200（不是 422/400 拒绝）
  - 创建出 `mml_tasks` 行：`product_resolved=false`、`matched_product_id IS NULL`、`matched_product_class='UNKNOWN-2025'`、`path_translation_source='orphan_passthrough'`
  - `device_tasks` 行：`path_translation_source='orphan_passthrough'`、`has_path_translation_miss=true`
  - `device_tasks.params.param_refs[].private_path` == 对应 standardPath（passthrough）
  - `device_tasks.params.param_refs[].translation_source == 'orphan_passthrough'`
  - Prometheus `mml_path_translation_orphan_total{product_class="UNKNOWN-2025"}` 计数 +1

### GWT-2（正常路径 — discovered/default 翻译生效）
- **Given** 一台 BAICELLS BLQ 设备 `product_class=BaiBLQ_XXX`，patterns 匹配 product BLQ，存在 discovered_param_mappings(BLQ, sw=5.0.16) 记录
- **When** MML 控制台执行 `MOD CARRIER PCI=5`
- **Then**
  - `mml_tasks.product_resolved=true`、`matched_product_id=BLQ.id`、`path_translation_source='discovered'`（或 `mixed`，若有 mapping 未命中）
  - `device_tasks.params.param_refs[].private_path` 含 `InternetGatewayDevice.Services.FAPService.{i}.X_VENDOR_*` 等厂商路径
  - API `GET /mml/tasks/{id}/results` 返回的 `path_translations[]` 中，每条含 `standard_path` / `private_path` / `translation_source='discovered'`

### GWT-3（前端列表行展开）
- **Given** 任务记录列表页 `/mml/task-records`
- **When** 用户点击列表行 ▶ 展开按钮
- **Then**
  - 行内（不是 Modal）显示三个子区：命令信息（CommandSummary）/ 路径转换详情表 / 设备执行结果表
  - 路径转换详情表列：`standardPath` / `privatePath` / 来源 Tag / 状态 Tag（discovered/default → 绿"已转换"；passthrough → 黄"未转换 - mapping 缺失"；orphan_passthrough → 橙"未转换 - 产品未识别"）
  - 若 `product_resolved=false`，展开区顶部显示橙色 Banner："产品 `<product_class>` 未识别，本任务全部原路径下发"
  - 老 Modal 入口"查看"按钮保留（后期再决定是否删）

### GWT-4（Prometheus 告警生效）
- **Given** Alertmanager 配置含本 PRD 新增的 `MMLPathTranslationOrphan` 规则
- **When** 5 分钟窗口内 `mml_path_translation_orphan_total` 累计增长 > 0
- **Then**
  - 触发告警 severity=warning
  - 告警 labels 含 `product_class` 让运维定位
  - Alertmanager 经 T-0152 的 webhook → notification_history 落库邮件

### GWT-5（向后兼容 — passthrough 与 orphan_passthrough 区分）
- **Given** 一台已识别 product（BLQ）的设备，但某 standardPath 未在 param_mappings 表中（mapping 缺失）
- **When** MML 翻译该路径
- **Then**
  - `param_refs[].translation_source='passthrough'`（不是 `orphan_passthrough`）
  - `mml_path_translation_orphan_total` **不**累加（只有产品未识别才累加）
  - 前端 Tag 显示黄色"mapping 缺失"，与橙色"产品未识别"区分

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 说明 |
|---|---|---|---|---|
| MML 命令集 | 通用 | 通用 | 通用 | 三家共用 MML console；本任务**不引入运营商分支** |
| product_class 池 | BLQ / BNQ / MLN / MLQ | （目前少量） | （目前少量） | patterns 表已按厂商+产品装配，与运营商无关 |
| param_mappings 字典 | 同表 | 同表 | 同表 | 翻译机制不区分运营商，由 product.param_model_id 决定 |
| orphan 容忍度 | 高（CMCC 厂商种类多） | 中 | 中 | 决策与运营商无关，**激进路线对三家统一开放** |
| 告警接收方 | 运维（同上游 OSS） | 同 | 同 | 通过 T-0152 通知中心，已有按 severity 路由 |

**结论**：**无运营商差异**。本任务不需要 `Carrier` 接口扩展点。

## 5. 非目标（Out of Scope）

- **不动 R-9.3 核心翻译机制**：`translateTaskPaths` / `MatchProductClass` / `Translator.ToPrivate` 实现保持不变
- **不修改 R-8.4**（device_sns product_class 一致性校验）—— R-8.4 与本任务正交
- **不做 product 自动学习**（看到野设备 productClass 自动 INSERT 到 patterns 表）—— 留 T-0142 处理
- **不删老的"查看 Modal"入口**—— 保留作为兜底，后期独立 task 评估
- **不做历史数据回填**（`mml_tasks.product_resolved` 历史默认 true，不回算）
- **不引入"按 device 单独翻译"**（D27 弹性保留，但本任务仍按 task 维度一次性翻译；device_tasks.path_translation_source 由 fanout 时从 task 复制）

## 6. 依赖

| 类型 | 标识 | 说明 |
|---|---|---|
| Backlog | T-0123-P1 ✅ | R-9.3 翻译框架已落地（done 2026-05-14） |
| Migration | 000114 ✅ | `device_tasks.has_path_translation_miss` 字段已加 |
| Component | `product.Registry` ✅ | `MatchProductClass` 工作中（T-0098-P2-01） |
| Component | `parammodel.Registry` ✅ | `Translator.ToPrivate` 工作中（T-0098-P2-02） |
| Component | `mmlPathTranslatorAdapter` ✅ | DI 接线已在 router（`cmd/app/provider/mml_adapters.go`） |
| 外部 | T-0152 ✅ | Alertmanager → notification_history 链路已通 |

无未结依赖。

## 7. 度量与可观测性

### 7.1 Prometheus 指标（新增 1）

| 指标 | 类型 | 标签 | 含义 |
|---|---|---|---|
| `mml_path_translation_orphan_total` | Counter | `product_class` | productClass 解析失败导致整 task 走 orphan_passthrough 的累计次数 |

### 7.2 Alertmanager 规则（新增 1）

```yaml
- alert: MMLPathTranslationOrphan
  expr: increase(mml_path_translation_orphan_total[5m]) > 0
  for: 0s
  labels:
    severity: warning
    domain: mml
  annotations:
    summary: "MML orphan productClass detected: {{ $labels.product_class }}"
    description: "野设备 product_class={{ $labels.product_class }} 在 MML 走 orphan_passthrough，请在 24h 内补 product_class_patterns 表登记。"
    runbook: "docs/operations/告警处置Runbook.md#MMLPathTranslationOrphan"
```

### 7.3 业务度量

- 任务列表行展开使用率（前端埋点可选）
- `mml_tasks.product_resolved=false` 比例（SQL：`SELECT COUNT(*) FILTER (WHERE product_resolved=false)::float / COUNT(*) FROM mml_tasks WHERE created_at > now()-interval '7d'`）
- 上线后 7 天内 orphan 告警计数 → 若 > 5 触发产品库治理 task（不在本任务范围）

### 7.4 日志

- `mml-translator-adapter` 在 orphan 时打 WARN：`zap.String("product_class", ...)` + `zap.Int("path_count", N)` + 已有 trace_id 注入（T-0157）

## 8. 验收清单（QA / 发布门）

- [ ] GWT-1 ~ GWT-5 全部 PASS
- [ ] `mml/console_validate_test.go` 新增 ≥ 1 个 orphan_passthrough table-driven 用例
- [ ] `scripts/e2e_verify.sh` 新增 ≥ 2 条 check_status 断言（GWT-1 + GWT-2）
- [ ] migration 000171 up/down 双向演练通过（goose v170↔171 至少一次）
- [ ] 新指标 `grep -rn mml_path_translation_orphan_total omcgo/` ≥ 2 处（注册 + 递增）
- [ ] 前端 `cd omcmb/webcode && npm run typecheck` 通过
- [ ] `omcmb/frontend-core/src/types/mml.ts` 类型与 backend API 响应字段对齐
- [ ] `omcmb/frontend-core/src/mock/` 对应 mock 数据更新
- [ ] Alertmanager 规则 `promtool check rules` 通过
- [ ] `docs/operations/告警处置Runbook.md` 加 `MMLPathTranslationOrphan` 一节
- [ ] DoD `docs/project/dod.md` 每项打勾或 N/A

## 9. 风险

| ID | 风险 | 等级 | 缓解 |
|---|---|---|---|
| R-NEW-T0168-1 | 激进路线让野设备静默接受命令，可能下发到非预期硬件 | 中 | Prometheus 告警 + Alertmanager webhook 邮件，运维 24h 内补 patterns |
| R-NEW-T0168-2 | mml_tasks 新增 4 列对历史查询性能影响 | 低 | 4 列默认值 NOT NULL DEFAULT 或 NULL，无索引（ALTER TABLE 元数据操作）；监控 query latency |
| R-NEW-T0168-3 | 前端列表行 expandable 数据加载延迟（按需懒加载 path_translations） | 低 | path_translations 从 device_tasks.params JSONB 提取，只在展开时拉，不影响列表渲染 |
| R-NEW-T0168-4 | 老 Modal 入口保留可能导致用户行为分裂 | 低 | 老 Modal 标注"单设备深入"，新展开行覆盖 80% 场景；S7 收尾登记 followup task 评估老 Modal 去留 |

## 10. 实施路线

| 阶段 | 制品 | 工作量 |
|---|---|---|
| S2 设计 | 本 PRD 附录"## 设计备忘"（接口签名 / 迁移 up/down / 前端组件层次） | 0.2d |
| S3 实施 | 后端 ~350 行 + 前端 ~120 行 + migration 000171 + Alertmanager 规则 | 0.8d |
| S4 验证 | golangci-lint + go test -race + npm typecheck + migrate up/down + e2e claim | 0.1d |
| S5 审查 | /review + /simplify | 0.05d |
| S6 提交 | 单 commit 含完整 footer | 0.02d |
| S7 收尾 | backlog 状态 + done 归档 + changelog | 0.03d |
| **合计** | | **~1.2d** |

---

## 设计备忘（S2 阶段产出 — 2026-05-24）

### S2.1 待定点决策

| ID | 决策 | 理由 |
|---|---|---|
| **D1** | `mml.PathTranslator` 接口签名升级为返 `*TranslationOutcome`（含 Paths/ProductID/ProductClass/Resolved/Source）。原 `TranslateForDevice(ctx, productClass, swVersion, paths) ([]TranslatedPath, error)` 改造为 `TranslateForDevice(ctx, productClass, swVersion, paths) (*TranslationOutcome, error)` | 现 caller 只有 `translateTaskPaths` 一处，重构成本低；避免"两次调用 ResolveProduct + TranslateForDevice"的冗余 |
| **D2** | `device_tasks.path_translation_source` 由 fanout 时从 task 上算好的字段直接复制下来，**不**在 fanout 时 per-device 单独翻译 | R-8.4 保证 task 内 product_class 一致；per-device 翻译是未来弹性（D27 已保留），首版无需 |

### S2.2 后端接口契约

#### S2.2.1 `mml.TranslationOutcome` 新结构（mml/service.go）

```go
// TranslationOutcome 是 PathTranslator.TranslateForDevice 的完整返回结果。
// 它把"产品解析"与"路径翻译"两步的结果合一，避免上层 caller 二次调用。
type TranslationOutcome struct {
    // Paths 是逐条 standardPath → privatePath 的翻译结果（与输入 standardPaths 顺序对齐）。
    Paths []TranslatedPath

    // ProductResolved 表示 productClass 是否成功匹配到 product。
    //   true  → MatchProductClass 命中，所有 Paths.Source ∈ {"discovered","default","passthrough"}
    //   false → MatchProductClass 返 ErrOrphan，所有 Paths.Source == "orphan_passthrough"
    ProductResolved bool

    // ProductID 是命中的 product.id（uuid）；ProductResolved=false 时为 uuid.Nil。
    ProductID uuid.UUID

    // ProductClass 透传调用方传入的 productClass，便于上层写审计列。
    ProductClass string

    // AggregateSource 是任务级翻译来源汇总（基于 Paths[].Source 推导）：
    //   - 全部 discovered / 全部 default / 全部 passthrough / 全部 orphan_passthrough → 取该单一值
    //   - 否则（任意混合）→ "mixed"
    AggregateSource string
}
```

#### S2.2.2 `mml.PathTranslator` 接口升级

```go
// 旧签名：
//   TranslateForDevice(ctx, productClass, softwareVersion string, standardPaths []string) ([]TranslatedPath, error)
//
// 新签名：
type PathTranslator interface {
    TranslateForDevice(
        ctx context.Context,
        productClass, softwareVersion string,
        standardPaths []string,
    ) (*TranslationOutcome, error)
}
```

**ErrProductClassUnresolved 的处置变化**：
- 旧：`MatchProductClass` 返 ErrOrphan → 适配器返 `&ErrProductClassUnresolved{}` → 阻塞 fanout
- 新：`MatchProductClass` 返 ErrOrphan → 适配器仍构造 `TranslationOutcome`，标 `ProductResolved=false` / `ProductID=uuid.Nil` / 所有 Path 走 `Source="orphan_passthrough"` → **不**返 error
- 适配器**保留**返 error 的场景：未注入（products==nil / params==nil）、context cancel、Registry 内部 IO 错误
- `mml.ErrProductClassUnresolved` sentinel 类型本任务**保留**（向后兼容），但 `mml_adapters.go` 不再触发它

#### S2.2.3 `MMLTask` 结构扩展（mml/model.go）

```go
type MMLTask struct {
    // ... 现有字段 ...

    // T-0168 翻译元数据（持久化到 mml_tasks 表 4 列）
    ProductResolved       bool       `json:"product_resolved"`
    MatchedProductID      *uuid.UUID `json:"matched_product_id,omitempty"`
    MatchedProductClass   string     `json:"matched_product_class,omitempty"`
    PathTranslationSource string     `json:"path_translation_source,omitempty"` // discovered/default/passthrough/orphan_passthrough/mixed
}
```

#### S2.2.4 `translateTaskPaths` 改造（mml/console_validate.go）

```go
func (s *Service) translateTaskPaths(ctx context.Context, task *MMLTask) error {
    if s.pathTranslator == nil { return nil }
    if len(task.DeviceSNs) == 0 || s.deviceLookup == nil { return nil }

    firstDev, err := s.deviceLookup.GetBySerialNumber(ctx, task.DeviceSNs[0])
    if err != nil || firstDev == nil || firstDev.ProductClass == "" { ... }

    standardPaths := collectStandardPaths(task.Commands)
    if len(standardPaths) == 0 { return nil }

    outcome, err := s.pathTranslator.TranslateForDevice(ctx,
        firstDev.ProductClass, firstDev.FirmwareVersion, standardPaths)
    if err != nil {
        // 仅 Registry IO / DI 错误才到这里；orphan 已被适配器消化
        return fmt.Errorf("translate paths for product_class %s: %w", firstDev.ProductClass, err)
    }

    // 写回任务级翻译元数据
    task.ProductResolved = outcome.ProductResolved
    task.MatchedProductClass = outcome.ProductClass
    if outcome.ProductResolved {
        id := outcome.ProductID
        task.MatchedProductID = &id
    }
    task.PathTranslationSource = outcome.AggregateSource

    // 写回 task.Commands（与旧实现一致）
    trans := make(map[string]TranslatedPath, len(outcome.Paths))
    for _, p := range outcome.Paths {
        trans[p.Standard] = p
    }
    for i := range task.Commands {
        applyTranslationToCommandEntry(task.Commands[i], trans)
    }
    return nil
}
```

#### S2.2.5 `mml_adapters.go::TranslateForDevice` 改造（cmd/app/provider）

```go
func (a *mmlPathTranslatorAdapter) TranslateForDevice(
    ctx context.Context, productClass, softwareVersion string, standardPaths []string,
) (*mml.TranslationOutcome, error) {
    if a.products == nil || a.params == nil {
        return nil, fmt.Errorf("mml-translator-adapter: ProductRegistry or ParamRegistry not wired")
    }

    matchRes, err := a.products.MatchProductClass(ctx, productClass)
    if err != nil && !errors.Is(err, product.ErrOrphan) {
        return nil, fmt.Errorf("mml-translator-adapter: match product_class %s: %w", productClass, err)
    }
    if errors.Is(err, product.ErrOrphan) || matchRes == nil || matchRes.Product == nil {
        // 激进路线：全部 path passthrough，标记 orphan
        a.logger.Warn("product_class unresolved, all paths orphan_passthrough",
            zap.String("product_class", productClass), zap.Int("path_count", len(standardPaths)))
        a.metrics.orphanInc(productClass)  // 新增指标
        paths := make([]mml.TranslatedPath, 0, len(standardPaths))
        for _, p := range standardPaths {
            paths = append(paths, mml.TranslatedPath{Standard: p, Private: p, Source: "orphan_passthrough"})
        }
        return &mml.TranslationOutcome{
            Paths:           paths,
            ProductResolved: false,
            ProductID:       uuid.Nil,
            ProductClass:    productClass,
            AggregateSource: "orphan_passthrough",
        }, nil
    }

    translator, err := a.params.Translator(ctx, matchRes.Product.ID, softwareVersion)
    if err != nil {
        // ParamRegistry 失败：全部 passthrough，但 ProductResolved=true（产品识别了，只是 mapping 拿不到）
        ...
    }

    src := string(translator.Source())  // "discovered" or "default"
    paths := make([]mml.TranslatedPath, 0, len(standardPaths))
    hasHit, hasMiss := false, false
    for _, p := range standardPaths {
        r := translator.ToPrivate(p)
        if r.Found {
            paths = append(paths, mml.TranslatedPath{Standard: p, Private: r.Translated, Source: src})
            hasHit = true
        } else {
            paths = append(paths, mml.TranslatedPath{Standard: p, Private: p, Source: "passthrough"})
            hasMiss = true
        }
    }
    aggregate := src
    if hasHit && hasMiss { aggregate = "mixed" }
    if !hasHit && hasMiss { aggregate = "passthrough" }

    return &mml.TranslationOutcome{
        Paths: paths, ProductResolved: true,
        ProductID: matchRes.Product.ID, ProductClass: productClass,
        AggregateSource: aggregate,
    }, nil
}
```

#### S2.2.6 `device_tasks.path_translation_source` 填充（mml/fanout.go）

fanout 时从 task 复制（D2 决策）：

```go
// 在创建 device_task 行时：
dtask := DeviceTask{
    // ... 现有字段 ...
    PathTranslationSource: task.PathTranslationSource,
}
```

#### S2.2.7 Results 响应增强（mml/handler.go）

新增结构：

```go
type PathTranslationView struct {
    StandardPath      string `json:"standard_path"`
    PrivatePath       string `json:"private_path"`
    TranslationSource string `json:"translation_source"` // discovered/default/passthrough/orphan_passthrough
    Translated        bool   `json:"translated"`         // source != "passthrough" && source != "orphan_passthrough"
}
```

`GET /mml/tasks/{id}/results` 响应增加 `path_translations []PathTranslationView` 字段：
- 从首条 device_task 的 `params.param_refs[]` JSONB 提取（R-8.4 保证一致）
- 去重（同 standardPath 只展示一次）

### S2.3 数据库迁移草案（migrations/000171）

```sql
-- +goose Up
-- ============================================================
-- 000171_mml_tasks_path_translation_audit.sql
-- T-0168 — mml_tasks / device_tasks 加路径翻译审计列（与 000114 互补）
--
-- 设计：T-0168 PRD §3 GWT-1/GWT-2 — 任务级翻译来源 + 产品解析状态列存
-- 化，避免审计查询 join + JSONB 解析；与 000114 has_path_translation_miss
-- （per-device 路径未命中标记）正交。
-- ============================================================

ALTER TABLE mml_tasks
    ADD COLUMN IF NOT EXISTS product_resolved        BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS matched_product_id      UUID NULL,
    ADD COLUMN IF NOT EXISTS matched_product_class   VARCHAR(64) NULL,
    ADD COLUMN IF NOT EXISTS path_translation_source VARCHAR(32) NULL;

COMMENT ON COLUMN mml_tasks.product_resolved IS
    'T-0168: 设备 product_class 是否通过 ProductRegistry.MatchProductClass 命中 product。'
    'false = orphan（激进路线下 path 走 orphan_passthrough 原路径下发，触发 Prometheus 告警 mml_path_translation_orphan_total）。'
    '历史默认 true（不回填，假设旧任务非 orphan）。';

COMMENT ON COLUMN mml_tasks.matched_product_id IS
    'T-0168: 命中的 product.id（UUID）。NULL = product_resolved=false 或非 MML 翻译路径。'
    '便于审计查询 join products 表拿厂商/参数模型信息。';

COMMENT ON COLUMN mml_tasks.matched_product_class IS
    'T-0168: 翻译时使用的设备 product_class 字符串（取 device_sns[0].product_class）。'
    '冗余存储避免 join devices 表；R-8.4 保证 task 内一致。';

COMMENT ON COLUMN mml_tasks.path_translation_source IS
    'T-0168: 任务级翻译来源汇总。枚举：'
    'discovered（全部走 discovered_param_mappings） / '
    'default（全部走 param_mappings 默认表） / '
    'passthrough（mapping 缺失，原路径下发） / '
    'orphan_passthrough（product 未识别，激进路线下发） / '
    'mixed（任务内多种来源混合）。NULL = 非 MML 翻译路径或 PathTranslator 未注入。';

ALTER TABLE device_tasks
    ADD COLUMN IF NOT EXISTS path_translation_source VARCHAR(32) NULL;

COMMENT ON COLUMN device_tasks.path_translation_source IS
    'T-0168: per-device 翻译来源（同 mml_tasks.path_translation_source 枚举）。'
    '首版由 fanout 从 task 直接复制（R-8.4 保证 task 内 product_class 一致）；'
    'D27 弹性保留：未来若引入 per-device swVersion 差异化翻译，本列由 per-device translate 重写。';

-- 不加索引：4 列均无高频 WHERE 过滤需求（按 task_id PK 已足够覆盖）

-- +goose Down
ALTER TABLE device_tasks DROP COLUMN IF EXISTS path_translation_source;
ALTER TABLE mml_tasks
    DROP COLUMN IF EXISTS path_translation_source,
    DROP COLUMN IF EXISTS matched_product_class,
    DROP COLUMN IF EXISTS matched_product_id,
    DROP COLUMN IF EXISTS product_resolved;
```

### S2.4 前端组件层次

```
omcmb/webcode/src/pages/mml/TaskRecord/index.tsx
├── ListPageLayout
│   └── DataTable
│       ├── expandable.expandedRowRender(record) → ▶ 展开
│       │   ├── (若 record.productResolved === false)
│       │   │   └── <Alert type="warning" message="产品未识别，本任务全部原路径下发" /> Banner
│       │   ├── CommandSummary (现有组件复用)
│       │   ├── ★ PathTranslationTable (新组件，本任务新增)
│       │   │   └── Table 列：standardPath / privatePath / 来源 Tag / 状态 Tag
│       │   └── DeviceResultsTable (现 resultRows 抽出)
│       └── columns 加 "查看" 按钮（保留老 Modal 入口兜底）
```

**新增组件**：`omcmb/webcode/src/pages/mml/TaskRecord/components/PathTranslationTable.tsx`

```tsx
type PathTranslationItem = {
  standardPath: string;
  privatePath: string;
  translationSource: 'discovered' | 'default' | 'passthrough' | 'orphan_passthrough';
  translated: boolean;
};

const SOURCE_TAG_CONFIG: Record<string, { color: string; text: string }> = {
  discovered:         { color: 'green',  text: '已转换（设备实测）' },
  default:            { color: 'green',  text: '已转换（标准映射）' },
  passthrough:        { color: 'gold',   text: '未转换（mapping 缺失）' },
  orphan_passthrough: { color: 'orange', text: '未转换（产品未识别）' },
};
```

**frontend-core 类型扩展**：`omcmb/frontend-core/src/types/mml.ts`

```ts
export interface MMLTask {
  // ... 现有字段 ...
  productResolved?: boolean;
  matchedProductClass?: string;
  matchedProductId?: string;
  pathTranslationSource?: 'discovered' | 'default' | 'passthrough' | 'orphan_passthrough' | 'mixed';
}

export interface BackendMMLTask {
  // ... 现有字段 ...
  product_resolved?: boolean;
  matched_product_class?: string;
  matched_product_id?: string;
  path_translation_source?: string;
}

export interface MMLTaskResultsResponse {
  // ... 现有字段 ...
  path_translations?: BackendPathTranslation[];
}
```

### S2.5 Carrier 差异点

**无**。详见 PRD §4 运营商差异矩阵。

### S2.6 观测埋点清单

| 名称 | 类型 | 文件 | 说明 |
|---|---|---|---|
| `mml_path_translation_orphan_total` | Counter(product_class) | `internal/mml/metrics.go` | 新增；orphan 计数 |
| `mml_path_translation_miss_total` | Counter(...) | 已存在 | 不动 |
| WARN log `product_class unresolved, all paths orphan_passthrough` | zap.Warn | `cmd/app/provider/mml_adapters.go` | 新增；字段 product_class / path_count / trace_id（T-0157 自动注入） |
| Alertmanager rule `MMLPathTranslationOrphan` | YAML | `deployments/monitoring/alerts/omc-rules.yml` | 新增；详 PRD §7.2 |

### S2.7 测试设计

**单元测试**（`internal/mml/console_validate_test.go` 新增）：

```go
func TestService_translateTaskPaths_OrphanPassthrough(t *testing.T) {
    // Given: PathTranslator mock 返 ProductResolved=false / AggregateSource="orphan_passthrough"
    // When:  translateTaskPaths
    // Then:  task.ProductResolved=false / task.MatchedProductID=nil
    //        task.PathTranslationSource="orphan_passthrough"
    //        task.Commands[i].param_refs[].translation_source 全为 "orphan_passthrough"
}

func TestService_translateTaskPaths_MixedSource(t *testing.T) {
    // mock：第一条 path 命中 discovered，第二条未命中
    // 验证 task.PathTranslationSource="mixed"
}
```

**适配器单测**（`cmd/app/provider/mml_adapters_test.go` 新增/补充）：

```go
func TestMMLPathTranslatorAdapter_TranslateForDevice_Orphan(t *testing.T) {
    // mock products.MatchProductClass 返 ErrOrphan
    // 验证 outcome.ProductResolved=false / 所有 path source="orphan_passthrough"
    // 验证 metrics.orphanInc(productClass) 被调用一次
    // 验证不返 error
}
```

**E2E**（`scripts/e2e_verify.sh` 新增 ≥ 2 条 claim）：

```bash
# claim mml-translation-1: 正常路径 product_resolved=true
# claim mml-translation-2: orphan 路径 product_resolved=false + path_translation_source=orphan_passthrough
```

### S2.8 实施顺序（S3 推进步骤）

1. migration 000171（schema 先行，本地 up/down 演练 1 轮）
2. `mml/model.go` MMLTask + DeviceTask 加字段 + `mml/pg_repository.go` 读写列
3. `mml/service.go` TranslationOutcome + PathTranslator 接口升级
4. `mml/console_validate.go` translateTaskPaths 改造
5. `mml_adapters.go` 适配器改造 + orphan 分支 + metrics 调用
6. `mml/metrics.go` 加 `mml_path_translation_orphan_total`
7. `mml/handler.go` results 响应增加 path_translations[]
8. 后端单测（console_validate_test + mml_adapters_test）
9. Alertmanager 规则 + Runbook 一节
10. 前端 frontend-core 类型 + Mock
11. 前端 PathTranslationTable + TaskRecord 接 expandable
12. E2E 脚本断言

### S2.9 出口门核查

| 出口门 | 状态 |
|---|---|
| 接口契约明确 | ✅ S2.2 全列 |
| 迁移草案 | ✅ S2.3 含 up/down |
| Carrier 差异点列出 | ✅ S2.5 无差异 |
| 观测埋点名字列出 | ✅ S2.6 |
| 待定点 < 3 | ✅ 全敲定（S2.1） |

---

**版本历史**：
- 2026-05-24 v1.0 起草（S0 阶段产出，PM + 电信 + MML 专家三方过目）
- 2026-05-24 v1.1 追加设计备忘（S2 阶段产出，2 个待定点全敲定）
