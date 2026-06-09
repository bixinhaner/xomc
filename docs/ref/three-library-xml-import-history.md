# 三库导入 XML 分层 — 历史与代码索引（ParamModel / Indicator / Alarm）

> 从 `omcgo/CLAUDE.md §5.3.1–5.3.3` 抽出。CLAUDE.md 只保留**当前范式**（单目录 + sidecar）的速记；本文件留：① 当前范式的完整代码索引；② 已被推翻的双目录历史（供考古，勿当现状）。
>
> **当前唯一真值（2026-06-04 定稿）= 单目录 + sidecar + 名称唯一 + 双重唯一硬拒 + data 外置 + 升级反向合并。** 任何 `*-custom` 双目录、`dir-prefix` 来源判定、`force` 覆盖的描述都是**历史背景**。

---

## 1. 当前范式（单目录 + sidecar）

三库（ParamModel / Indicator / Alarm）同范式：
- **单目录**：builtin 与 custom 同住一个目录（`data/param-mappings/`、`data/indicator-library/`、`data/alarm-definitions/`），取消 `*-custom` 后缀目录。
- **来源判定靠 sidecar**：`X.xml.custom` 空文件随文件走，扛过升级合并与 DB 重建。`source.go::IsCustom/IsDeletable(baseDir, loadedFrom)` 读 sidecar；仅 custom 可删，builtin/unknown → 403。
- **上传 = `name` 必填 + 双重唯一硬拒（无 force）**：文件名 `<name>.xml` 同目录唯一 + 内容主键唯一（param_models.name / indicator platform / alarm neType）。命中任一 → 409「请改名」。落盘 + 写 sidecar；DELETE 连带删 sidecar。
- **data 外置（Phase 2）**：整个 `data/` 不进镜像（Dockerfile 去 `COPY data`），bind-mount 进 app/worker（dev 挂源码树；prod 挂 `/opt/omc/data`，deploy.sh 首次播种 + 升级「现网赢、新版补充」`cp -an` 反向合并 + 快照回滚）。
- loader 单目录扫描（跳过 `*.custom`）；backup_cleanup 单目录 + 清孤儿 sidecar。

**三库差异**：

| 库 | 目录 | 内容主键 | tech 子目录 | 错误码段 |
|----|------|---------|------------|---------|
| ParamModel (T-0178) | `data/param-mappings/`（扁平） | `param_models.name` | 无 | 2030-2039 |
| Indicator (T-0180) | `data/indicator-library/{enb,gsm,gnb}/` + 根级 builtin | `platform`（按 tech 分表） | enb/gsm/gnb | 2040-2049 |
| Alarm | `data/alarm-definitions/`（扁平） | `neType` | 无 | 2050-2059 |

> Indicator 关键差异：上传按 `?tech=` 落 `{gsm,gnb}/` 子目录，不再覆盖根级出厂单文件；强制 `<indicatorModel platform deviceType>` 根元素，deviceType present 则须匹配 tech，absent 容忍（ENB legacy）。

**单实例假设**：Upload/Delete/单文件 Reload 通过 `acquireFileLock(basename)` 进程内 `sync.Map[name]*sync.Mutex` 互斥。**多实例横扩前必须**补 PG advisory lock `pg_try_advisory_xact_lock(...)`，否则同名 Upload 丢更新。

**当前代码索引**：

| 库 | 关键文件 |
|----|---------|
| ParamModel | `internal/config/parammodel/{source,filelist,upload,handler,backup_cleanup,loader}.go` |
| Indicator | `internal/pm/indicator/{source,filelist,upload,file_handler,file_repository,backup_cleanup,reload,loader,rest_handler}.go` |
| Alarm | `internal/alarm/definition/{source,upload,file_handler,file_repository,backup_cleanup,loader}.go` |
| 共用 | worker `cmd/worker/main.go::start*BackupCleanup`（cron `0 3 * * *`）；provider `cmd/app/provider/{pm,alarmdef}.go` EnsureBaseDir；部署 `deployments/docker/docker-compose.yml` + `release/bundle/deploy/{docker-compose.app.yml,deploy.sh}` bind mount |
| 前端 | `frontend-core/src/{types,services/api,hooks/api}/*`（paramModel / indicatorLibrary / alarmDefinition）+ `webcode/src/pages/product/{param-model,kpi-library,alarm-library}/` |

> 数据备份保留 30 天（`.deleted.<ts>`/`.bak.<ts>`，worker cron 清理）；`.tmp.<uuid>` 残留 1 小时即清。DELETE 备份失败 → 保守回滚（500，不删 DB）。

---

## 2. 已被推翻的历史（双目录方案，勿当现状）

2026-06-04 之前曾设计为 **builtin / custom 物理隔离两个目录**（`data/param-mappings/` + `data/param-mappings-custom/`），来源靠目录前缀（`loaded_from` 列写 `param-mappings/X.xml` vs `param-mappings-custom/X.xml`），上传支持 `force` 覆盖。该方案已被单目录 + sidecar 取代，相关 `*-custom` 目录均不存在。

历史端到端 commit 链路（仅供溯源，**不反映当前代码**）：
- **ParamModel（T-0178，8 commit）**：`58640fb3`（source 分类器）→ `944c6d4c`（Loader 双目录）→ `d1349c3c`（Delete 守门）→ `16f95d46`（Upload）→ `61a0e5aa`（三态 yaml）→ worker `2c27253b` → 部署 `aab7245b` → 前端 `84cf6950`；定稿 `11886b61` + `420b08f0`/`110c9bb2`（sidecar）。
- **Indicator（T-0180，9 commit）**：`35ee4d51`（立项）→ `69101bcb` → `351cf0b2` → `311bde18` → `a58f7833` → `c929a760` → worker `96ca95bd` → 部署 `1b742aa7` → 前端 `df2dd814`；定稿 `54e6c227`。
- **Alarm（3 commit）**：`3fcc1292`（定稿，单目录+sidecar）等。

> 历史注记里引用的 `migrations/000215`、`000217`、`seed/000006` 等迁移号在 2026-05-31 consolidation 后已不存在；当前迁移以 `omcgo/migrations/README.md` 为准。
