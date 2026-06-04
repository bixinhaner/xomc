# 三库「导入 XML」统一改造 + data 升级保全 — 设计文档

> 2026-06-04 立项。涉及 product/param-model、product/kpi-library、product/alarm-library 三页。
> 本文档为**已拍板决策**的落地设计,分 Phase 1（应用层）/ Phase 2（打包部署层）。

---

## 1. 已拍板决策（来自需求澄清）

| # | 决策 |
|---|---|
| D1 | 模型 B：整个 `omcgo/data` **不烤进镜像**，外置 + bind-mount；升级时**现网 data 目录覆盖合并进新版本 data 目录**（现网文件赢，新版只补充其没有的文件） |
| D2 | 外置范围 = 整个 `omcgo/data`（含 quicksettings / mml-catalog） |
| D3 | **单目录**：取消 `*-custom` 后缀目录，builtin 与 custom 同住一个目录；名称在**同一目录内唯一**，跨目录可重复（按现 XML 加载规律） |
| D4 | 三页各自以「名称」为唯一约束（三个独立命名空间） |
| D5 | 重复**硬拒**（提示改名）；更新 = 先删再传，**不再覆盖（force）** |
| D6 | 来源判定用 **sidecar 标记文件**（`X.xml.custom`），**不新增 DB 列**；`is_build_in` 保留（它是 indicator 指标级"系统标准"标记，与文件来源是两根轴） |
| D7 | 已提交 `ed568dc2`（IsDeletable=可删仅限 custom + DELETE 403 守门）作为地基 |

---

## 2. 来源判定模型（单目录 + sidecar）

```
indicator-library/
  enb/ALL.xml              ← builtin（无 sidecar）→ 不可删
  enb/MyKpi.xml            ← custom（有 sidecar）→ 可删
  enb/MyKpi.xml.custom     ← sidecar 标记（空文件）
  GSM.xml / GNB.xml        ← builtin 根级单文件
param-mappings/
  BTS.xml                  ← builtin
  MyModel.xml + MyModel.xml.custom   ← custom
alarm-definitions/
  ENB.xml                  ← builtin
  MyAlarm.xml + MyAlarm.xml.custom   ← custom
```

**判定规则**：`X.xml` 旁存在 `X.xml.custom` ⇒ custom（可删）；否则 builtin（不可删）。

**为什么 sidecar**：
- 来源信息**随文件走** → 扛过 D1 的"文件级反向合并升级"（custom 文件 + 其 sidecar 一起被现网目录保留）；
- 扛过 DB 重建（不依赖 pgdata）；
- 不改用户上传的 XML 内容（vs XML 根属性方案）；
- 不依赖目录前缀（D3 已取消 `-custom`）。

---

## 3. Phase 1：应用层改造（前端 + 后端，无 DB 迁移）

### 3.1 后端来源判定（`source.go` ×3）

签名由纯字符串改为带 baseDir（需文件系统判断 sidecar 是否存在）：

```go
const CustomMarkerSuffix = ".custom"

func IsCustom(baseDir, loadedFrom string) bool {
    if loadedFrom == "" { return false }
    _, err := os.Stat(filepath.Join(baseDir, loadedFrom) + CustomMarkerSuffix)
    return err == nil
}
func IsDeletable(baseDir, loadedFrom string) bool { return IsCustom(baseDir, loadedFrom) }
func ClassifySource(baseDir, loadedFrom string) Source {
    if loadedFrom == "" { return SourceUnknown }
    if IsCustom(baseDir, loadedFrom) { return SourceCustom }
    return SourceBuiltin
}
```

- 删除 `CustomDirPrefix` / `CustomDirSubdir` 相关常量与 dir-prefix 分支。
- `BuiltinDirSubdir` 保留为**唯一**数据子目录名。
- 调用方（DTO 填 source/deletable、DELETE 守门）改为传 baseDir —— handler 均持有 baseDir。
- ⚠️ 上一轮提交 `ed568dc2` 的 dir-prefix 版 `IsDeletable(loadedFrom)` + 其测试，在此被改写为 sidecar 版（预期内的演进，非返工）。

### 3.2 后端单目录合并（`filelist.go` / `loader.go` / `backup_cleanup.go` ×3）

- Loader 只扫**单目录**（取消 builtin+custom 双目录 merge）；扫描时**跳过 `*.custom` sidecar**（不作 XML 解析）。
- `loaded_from` 一律写 `<dir>/<file>`（不再有 `-custom/` 形态）。
- indicator 目录布局：custom 上传落 `indicator-library/<tech>/<name>.xml`（enb/gsm/gnb 子目录化）；builtin 维持 `enb/*.xml` + 根级 `GSM.xml`/`GNB.xml`。Loader 需扫 enb/gsm/gnb 三子目录 + 根级单文件。
- `backup_cleanup`：单目录扫描；顺带清理孤儿 `*.custom`（对应 XML 已不在时）。
- `appconfig` 中 alarm 的 `CustomDirectory`/`CustomOverrides` 等字段废弃清理。

### 3.3 后端上传（`upload.go` / `file_handler.go` UploadXML ×3）

- 入参新增 **`name`**（用户输入的唯一名称）；落盘文件名 = `<name>.xml`。
- 名称校验：非空、合法文件名字符集（`[A-Za-z0-9_-]`，禁 `..`/空格/点）、长度上限（如 64）。
- **唯一性硬校验**：目标目录已存在 `<name>.xml`（无论 builtin 还是 custom）→ **400/409 + "名称已存在，请改名"**；**去掉 force 覆盖路径**（D5）。
- 写 `<name>.xml` → 写空 sidecar `<name>.xml.custom` → 触发 reload。
- indicator 续用 tech 选择决定子目录；名称在该 tech 子目录内唯一（D3 跨目录可重复）。

### 3.4 后端删除（`DeleteFile` / `DeleteModel` ×3）

- 守门：`IsCustom(baseDir, loadedFrom)` 为假（builtin/无 sidecar）→ 403（沿用错误码 2030/2040/2050）。
- 删除时：备份并删除 `X.xml` **+ 同步删除 sidecar `X.xml.custom`**。

### 3.5 前端：三个上传 Modal（`UploadXmlModal` / `AlarmUploadXmlModal` / param-model 上传入口）

- 加**「名称」必填 Input**（唯一）+ 文件选择（indicator 保留 tech 选择）。
- 提交前用现有文件清单（ListFiles / summary）即时查重 → 命中则**内联报错"名称已存在，请改名"**（不再弹"覆盖确认"）。
- 后端 409 → 同样提示改名。
- 删除按钮置灰已在上一轮完成（读 `deletable`），sidecar 生效后自动正确。

### 3.6 测试

- `source_test`（×3）：改为 sidecar 版（temp dir + 建/不建 sidecar 断言 custom/builtin）。
- 上传测试：名称合法性 / 唯一性硬拒 / sidecar 写入。
- 删除测试：sidecar 守门 + 删除时 sidecar 一并清除。

### 3.7 Phase 1 不涉及 DB 迁移
sidecar + 保留 is_build_in ⇒ **无 schema 变更**。（存量 `*-custom/` 目录里的历史 custom 文件迁移到单目录 + 补 sidecar，归 Phase 2 / 一次性脚本；dev 环境该目录基本为空。）

---

## 4. Phase 2：打包与升级保全（部署层）

- `Dockerfile.app`：**移除** `COPY /build/data /etc/omcgo/data`（D1：data 不进镜像）。
- `docker-compose.yml`（dev）+ `docker-compose.app.yml`（prod）：app/worker **bind-mount 整个 data 目录**（RW，UID 10001 可写）。
- `build-release.sh`：data 仍随包发布（已在 `$STAGE/data`），作为新版本 builtin 基线。
- `deploy.sh`：
  - 首次部署：把包内 `data/` 播种到 `/opt/omc/data`。
  - 升级：**先快照** `/opt/omc/data → .bak.<ts>`（回滚）；**反向合并**=`cp -a 现网/opt/omc/data/. 新包data/` → 再把合并结果作为挂载源（D1：现网赢、新版补充）。
  - 清理死代码（`*-custom` 初始化段）。
- 启动期兜底：挂载缺失/为空 → app 字典加载将为空（模型 B 固有风险），deploy.sh 首次播种必须成功。
- 更新 `CLAUDE.md §5.3.1–5.3.3`：再次反转 2026-06-03 注记 → 记为「单目录 + sidecar + 名称唯一 + data 外置 + 反向合并升级」。

---

## 5. 风险与限制

| 级别 | 项 |
|---|---|
| HIGH | **名称唯一 ≠ 内容主键唯一**：KPI 按 XML 内 `platform`、告警按 `neType` 分组入库。两个不同「名称」文件可能内部 `platform`/`neType` 相同 → loader first-seen 取一，另一个被忽略。D4 只约束文件名，**挡不住这层**。建议 Phase 1 附加"内容主键也唯一"校验，或明确接受 first-seen。**待你定。** |
| HIGH | 模型 B：data 不在镜像，**挂载缺失/未播种 → 字典空、设备路由/KPI/告警失效**。deploy.sh 首次播种是硬依赖。 |
| MED | 升级合并方向（现网赢）：新版**修订过的既有 builtin 文件不会自动生效**（你已知情接受）。 |
| MED | 存量 `*-custom/` 历史文件迁移到单目录 + 补 sidecar（一次性）。 |
| MED | sidecar 与 XML 一致性：删 XML 漏删 sidecar / 反之 → 误判来源。靠 DELETE 同步 + backup_cleanup 清孤儿兜底。 |
| LOW | K8s（`deployments/k8s/*`）的 data 外置需 PVC，本期不做，标 TODO。 |
| LOW | 反转 2026-06-03（5 轮 ULTRATHINK）决策，需同步 CLAUDE.md，避免再次误导（已第三次）。 |

---

## 6. Phase 1 改动文件清单

**后端（Go）**
- `internal/{config/parammodel,pm/indicator,alarm/definition}/source.go` — sidecar 判定，删 dir-prefix
- `…/filelist.go` ×3 — 单目录扫描
- `…/loader.go` ×3 — 单目录、跳过 `*.custom`、loaded_from 去 -custom
- `…/upload.go` + `…/file_handler.go` ×3 — name 入参 + 唯一硬拒 + 写 sidecar；DELETE 守门改 sidecar + 删 sidecar
- `…/backup_cleanup.go` ×3 — 单目录 + 孤儿 sidecar
- `internal/core/appconfig/config.go` — 废弃 custom-dir 配置字段
- 各 `*_test.go` — sidecar 版断言

**前端（TS）**
- `webcode/src/pages/product/kpi-library/UploadXmlModal.tsx`
- `webcode/src/pages/product/alarm-library/AlarmUploadXmlModal.tsx`
- `webcode/src/pages/product/param-model/`（上传入口 Modal）
- i18n：名称必填 / 已存在改名 文案（zh + en）

**迁移**：无（Phase 1）。

---

## 7. 决策（已定）

1. **选项 ①（已定 2026-06-04）**：Phase 1 上传时**双重唯一校验** —— 文件名（同目录）唯一 **且** XML 内容主键唯一：
   - param-model：`param_models.name`（模型名）
   - kpi-library：`platform`
   - alarm-library：`neType`
   解析上传 XML 取内容主键，与现有库比对，撞车 → 硬拒（提示改名/换内容），杜绝 loader first-seen 静默丢失。
2. Phase 1 / Phase 2 **分两批提交**；Phase 1 内按模块纵向分块提交，可独立验证。
