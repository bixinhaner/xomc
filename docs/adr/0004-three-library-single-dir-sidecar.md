# 0004 — 三库导入 XML 用单目录 + sidecar，不用 builtin/custom 双目录物理隔离

## Status

Accepted（追认 retroactive）。决策于 2026-06-04 定稿（在此之前曾两度反转），本目录成立时补记。

> 本决策建立在 ADR 0003（三库统一为字典 + dictloader 范式）之上。

## Context

三库（ParamModel `data/param-mappings/`、Indicator `data/indicator-library/`、Alarm `data/alarm-definitions/`）都需要在出厂内置（builtin）字典之外，支持用户上传自定义（custom）XML，并要回答两个问题：**怎么判定一个文件是 builtin 还是 custom（决定能否删除）**，以及**升级时怎么保住现网的 custom 文件**。

2026-06-04 之前曾设计为 **builtin / custom 双目录物理隔离**：`data/param-mappings/` + `data/param-mappings-custom/`，来源靠目录前缀判定（`loaded_from` 列写 `param-mappings/X.xml` vs `param-mappings-custom/X.xml`），上传支持 `force` 覆盖。该方案的问题：

- 来源信息绑在目录结构上，**扛不住"data 外置 + 升级反向合并"**（现网目录覆盖合并进新版本时，custom 文件需连同其来源标记一起被保留）。
- `force` 覆盖让"更新"语义隐式、易误覆盖出厂文件。
- 双目录使 Loader 需做 builtin + custom 双目录 merge，逻辑分叉。

需要一个**来源信息随文件走**、不依赖目录结构、不改用户 XML 内容、且能扛过 DB 重建与升级合并的方案。

## Decision

定稿范式 = **单目录 + sidecar + 名称唯一 + 双重唯一硬拒 + data 外置 + 升级反向合并**：

1. **单目录**：builtin 与 custom 同住一个目录，取消所有 `*-custom` 后缀目录。名称在**同一目录内**唯一（跨目录可重复）。
2. **来源靠 sidecar 空文件**：`X.xml` 旁存在 `X.xml.custom` ⇒ custom（可删）；否则 builtin（不可删）。`source.go::IsCustom/IsDeletable(baseDir, loadedFrom)` 读 sidecar；仅 custom 可删，builtin/unknown → 403。来源信息随文件走，扛过升级合并与 DB 重建，且不改用户上传的 XML 内容。
3. **上传双重唯一硬拒（无 force）**：上传必填 `name`，落盘 `<name>.xml`；文件名同目录唯一 **且** 内容主键唯一（ParamModel `param_models.name` / Indicator `platform` / Alarm `neType`），命中任一 → 409「请改名」。内容主键也校验，杜绝 loader first-seen 静默丢失。更新 = 先删再传。落盘后写空 sidecar；DELETE 连带删 sidecar。
4. **data 外置**：整个 `data/` 不进镜像（Dockerfile 去 `COPY data`），bind-mount 进 app/worker（dev 挂源码树，prod 挂 `/opt/omc/data`）。升级时 deploy.sh 先快照回滚，再**反向合并**（`cp -an`，现网赢、新版只补其没有的文件）。
5. **三库差异**：ParamModel / Alarm 扁平目录；Indicator 按 `?tech=` 落 `{enb,gsm,gnb}/` 子目录 + 根级 builtin 单文件。错误码段 ParamModel 2030 / Indicator 2040 / Alarm 2050。

## Consequences

**正向**：

- 来源标记随文件走 → 与"data 外置 + 现网赢反向合并升级"天然兼容，custom 文件 + sidecar 一起被保留，不丢。
- 不依赖目录结构、不依赖 DB 列（`is_build_in` 是另一根轴，保留）、不改用户 XML，扛过 DB 重建。
- 去掉 `force`，更新语义显式（先删再传）；双重唯一硬拒堵住 loader first-seen 静默丢失。
- 三库同范式，Loader 单目录扫描（跳过 `*.custom`），逻辑收敛。

**代价 / 约束**：

- **名称唯一 ≠ 内容主键唯一**已用上传期双重校验兜住，但要求上传时解析 XML 取内容主键比对，增加上传路径复杂度。
- **data 外置是硬依赖**：挂载缺失 / 未播种 → 字典空、设备路由 / KPI / 告警失效，deploy.sh 首次播种必须成功。
- **单实例假设**：Upload/Delete/Reload 走进程内 `acquireFileLock`（`sync.Map[name]*sync.Mutex`）；多实例横扩前必须补 PG advisory lock，否则同名 Upload 丢更新。
- sidecar 与 XML 需保持一致（删 XML 同步删 sidecar，backup_cleanup 清孤儿 sidecar 兜底）。
- 升级「现网赢」意味着新版修订过的既有 builtin 文件不会自动生效（已知情接受）。

> 当前范式速记见 `omcgo/CLAUDE.md §4.4`。完整代码索引 + 被推翻的双目录历史见 `docs/ref/three-library-xml-import-history.md`；落地设计见 `docs/design/three-library-xml-import-redesign-20260604.md`。
