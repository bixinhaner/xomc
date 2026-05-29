# PRD — param-model 自定义 XML 持久化与"内置不可删"分层目录（T-0178）

| 项 | 值 |
|---|---|
| Backlog | T-0178 |
| Type / Prio | feat / P2 |
| Sprint / Owner | sprint-13 / Claude+user |
| Est | M (~4.5d,详 §10 工期分解) |
| Deps | T-0098 ✅(`param_models` / `param_mappings` / `loaded_from` 字段已存在) |

---

## 1. 业务背景

T-0098 P1-06 落地后,9 个出厂 paramModel XML 与 `standard-model.xml` 通过 `dictloader` 启动期载入 `param_models` 表,镜像内固化路径 `/etc/omcgo/data/param-mappings/`。**问题来自不同运营商/集成商现场对私有扩展模型的差异化需求**:

- **现状**:`deployments/release/bundle/deploy/docker-compose.app.yml:85-90` app 容器 **未挂载** `data/param-mappings`,所有 XML 来自镜像层(`Dockerfile.app:47` `COPY /build/data → /etc/omcgo/data`)。
- **痛点 1**:现场运维若想为项目 A 加一个 `CBQQ.xml`(私有 paramModel),只能改源码 → 重发版镜像 → 不同项目无法保持各自定制。
- **痛点 2**:即便临时把 XML `docker cp` 进容器,**下次升级 `docker compose up -d` 重建容器即丢**(容器层无持久化)。
- **痛点 3**:现有 `重载 XML / 导入 XML` 按钮(上一轮已分析)只重扫现有目录,不解决"如何把新文件交付到容器"。

**业务目标**:让运维在 product/param-model 页面**直接上传** XML 文件,文件落 host 持久化目录,**升级镜像不丢失**;同时**严守安全边界** — 出厂内置 XML(随版本演进)与运维自定义 XML(随项目环境演进)物理隔离,不出现互相覆盖造成丢数据的事故。

## 2. 用户故事

### 2.1 网管运维 — 给现场项目加自定义 paramModel
> 作为网管运维,**我希望在 product/param-model 页面上传 `CBQQ.xml`**,文件持久化到 host(不在容器层),系统重载后字典里立刻能看到新模型,且**下次升级 OMC 版本时 CBQQ.xml 不丢**。

### 2.2 网管运维 — 同名时强制覆盖
> 作为网管运维,**我希望同名上传时弹出二次确认**,确认后可强制覆盖(旧版本自动备份为 `.bak.<ts>`),便于快速修订错误的私有 XML。

### 2.3 网管运维 — 误删保护
> 作为网管运维,**我希望出厂内置 XML 在 UI 上不可删除**(按钮置灰 + Tooltip 说明),避免误操作导致字典回退到错误状态;只对自己上传的自定义 XML 暴露删除按钮。

### 2.4 网管运维 — 删错可恢复
> 作为网管运维,**我希望删除自定义 XML 不是 unlink 而是 rename 备份**,误删后 30 天内可以从 host 文件系统恢复。

### 2.5 版本管理员 — 升级保护
> 作为升级维护人员,**我希望执行 `deploy.sh` 升级 OMC 镜像时,运维上传的自定义 XML 完全不动**,但出厂内置 XML 跟随新镜像自动更新;同名时由配置决定胜出方(默认 custom 胜,删 custom 后自动回退到 builtin)。

### 2.6 DBA / 排障 — 来源可审计
> 作为 DBA,**我希望 `param_models` 表的 `loaded_from` 列带目录前缀**(`param-mappings/BTS.xml` 或 `param-mappings-custom/CBQQ.xml`),可直接 SQL 筛选哪些是内置、哪些是运维上传。

## 3. 验收标准(Given-When-Then)

### GWT-1(自定义 XML 持久化跨升级)
- **Given** 项目 A 已部署 OMC v1,运维通过 UI 上传 `CBQQ.xml` 成功
- **When** 升级到 OMC v2(新镜像 tag、新版本 `data/param-mappings/` 不含 `CBQQ.xml`),执行 `bash deploy.sh`
- **Then** ① `/opt/omc/data/param-mappings-custom/CBQQ.xml` 仍存在;② 新容器启动后字典自动加载,`GET /param-models` 返回 CBQQ;③ 数据库 `param_models.loaded_from = 'param-mappings-custom/CBQQ.xml'`

### GWT-2(来源字段准确)
- **Given** 数据库已加载 9 个出厂 paramModel + 1 个自定义 paramModel(CBQQ)
- **When** 调用 `GET /api/v1/param-models`
- **Then** ① 10 行返回,每行含 `loaded_from / source / deletable` 三字段;② BLQ 等 9 行 `source=builtin, deletable=false`;③ CBQQ 行 `source=custom, deletable=true`

### GWT-3(内置不可删 — 后端硬拦截)
- **Given** 数据库存在内置 paramModel `BTS`(`loaded_from='param-mappings/BTS.xml'`)
- **When** 任何客户端调用 `DELETE /api/v1/param-models/BTS`
- **Then** 返回 `403 Forbidden`,响应 `code=40310 ErrCodeBuiltinNotDeletable`,审计日志写 `parammodel.delete.rejected_builtin`,数据库行**不被删除**,镜像 XML **不被删除**

### GWT-4(自定义可删 + 物理备份)
- **Given** 数据库存在自定义 paramModel `CBQQ`(`loaded_from='param-mappings-custom/CBQQ.xml'`)
- **When** 管理员调用 `DELETE /api/v1/param-models/CBQQ`
- **Then** ① 返回 200,响应含 `backup` 字段(形如 `CBQQ.xml.deleted.20260601030405`);② host 文件系统 `param-mappings-custom/CBQQ.xml` 消失,同目录出现 `CBQQ.xml.deleted.<14位ts>` 备份;③ DB 行被删除,关联 `param_mappings` CASCADE 删除,`products.param_model_id` 置 NULL;④ 审计 `parammodel.delete.custom`

### GWT-5(同名 self-healing 回退)
- **Given** 出厂内置 `BTS.xml` 已加载,运维上传同名 `BTS.xml`(覆盖,`custom_overrides_builtin=true`,`source=custom`)
- **When** 管理员调用 `DELETE /api/v1/param-models/BTS`
- **Then** ① custom 文件被 rename 为 `.deleted.<ts>` 备份;② DB 行删除;③ **触发 Reload 后**,Loader 从 builtin 目录重新发现 `BTS.xml`,DB 行重新出现,`source=builtin, deletable=false`

### GWT-6(同名上传强制覆盖)
- **Given** `param-mappings-custom/CBQQ.xml` 已存在(由上次上传写入)
- **When** 客户端再次 POST `/api/v1/param-models/upload-xml`(同名 + 无 `force=true`)
- **Then** 返回 `409 Conflict`,响应提示 `?force=true` 可覆盖
- **And When** 客户端补 `?force=true` 重试
- **Then** ① 返回 200;② 旧文件 rename 为 `.bak.<ts>`;③ 新文件落盘;④ 单文件 Reload 入库;⑤ 审计 `parammodel.upload.overwrite`

### GWT-7(备份失败 → 保守回滚)
- **Given** host `param-mappings-custom/` 目录因磁盘满或权限问题导致 rename 失败(`EACCES/ENOSPC/EROFS` 等非 ENOENT errno)
- **When** 管理员 DELETE 一个自定义 paramModel
- **Then** ① 返回 500,`code=50031 ErrCodeBackupFailed`;② host 原 XML **不被删除**;③ DB 行 **不被删除**;④ 审计 `parammodel.delete.aborted_backup_failed` 含 errno;⑤ Prometheus 告警 `ParamModelBackupFailedSurge` 触发

### GWT-8(备份 30 天清理)
- **Given** `param-mappings-custom/` 下存在 `CBQQ.xml.deleted.20260101030405`(40 天前的备份)
- **When** omcgo-worker 每天凌晨 3 点的 cron `parammodel-backup-cleanup` 触发
- **Then** ① 该文件被 unlink;② 30 天内的 `.deleted/.bak` 备份保留不删;③ 文件名 ts 解析失败的文件**不删**(用户私有备份保护);④ 指标 `parammodel_backup_cleanup_total{result="swept"}` +1

### GWT-9(安全 — 路径遍历拦截)
- **Given** 客户端构造恶意 multipart 文件名 `../../etc/passwd` 上传
- **When** 后端 Upload handler 解析
- **Then** ① 文件名白名单正则 `^[A-Za-z0-9_-]{1,64}\.xml$` 不通过;② 返回 400;③ 审计 `parammodel.upload.rejected_invalid_name`;④ host 文件系统**不被任何写入**

### GWT-10(安全 — XML 内容校验)
- **Given** 客户端上传一个语法合法但根元素非 `paramModel` 的 XML(例如 HTML / `products.xml` 风格)
- **When** 后端解析校验
- **Then** ① 返回 400 `invalid_xml_root`;② 不入 custom 目录;③ 审计记录拒绝原因

### GWT-11(同名占位文件冲突)
- **Given** 客户端上传文件名为 `standard-model.xml` 或 `products.xml` 或 `param-model-routing.xml`(4 个保留名之一)
- **When** 后端校验
- **Then** ① 返回 400 `reserved_filename`;② 不入 custom 目录(避免误盖出厂保留文件语义)

## 4. 运营商差异矩阵

**无运营商差异**。自定义 XML 持久化机制对 cmcc/ctcc/cucc 一视同仁;运营商差异仍由各运营商自己上传的 XML 内容承载(本任务不引入运营商特化逻辑)。

## 5. 非目标

- **不做**:多 app 实例并发协调。当前依赖 `sync.Map[filename]*sync.Mutex` 进程内互斥;横扩多实例时再补 PG advisory lock(`omcgo/CLAUDE.md §5.3` 注明 TODO)
- **不做**:Upload 时自动派生 `param_mappings`(已由 Loader 单事务 UPSERT 处理,无需在 handler 内重复)
- **不做**:`standard-model.xml` 的 Upload(运维不该改 standard 表)
- **不做**:批量上传(单次单文件,UI 不暴露 multi-file)
- **不做**:`product-name-routing.xml` / `param-model-routing.xml` 上传(历史遗留 routing 已废弃)
- **不做**:Custom XML 编辑 / 在线 diff(查看与下载留后续 PRD)
- **不做**:跨 host 同步(每台 host 独立维护自己的 custom XML,符合"不同项目保持不同 XML"语义)

## 6. 依赖

- T-0098 ✅ `param_models` / `param_mappings` schema(本任务复用 `loaded_from` 列)
- T-0098 P1-06 ✅ `parammodel.Loader.Reload` 已支持 `mode=reload` 孤儿清理(本任务只增量加 custom 目录扫描)
- T-0098 P2-01 ✅ `ProductRegistry` 已不再依赖 `devices.param_model_id`(本任务删除 paramModel 时 `SET NULL` 不影响运行路径)
- 现有 `dictloader.Registry.ReloadOne` ✅
- 现有 `audit_logs` 表 ✅(`omcgo/internal/admin/` 提供 `AuditService`)
- 现有 RBAC `RequirePermission` 中间件 ✅
- 现有 omcgo-worker cron 注册框架(`robfig/cron/v3`)✅
- **新增依赖**:GW(管理 API)层暴露 Prometheus alert 规则到 `omcgo/monitoring/prometheus/alerts.yml`(若不存在则同步新建)

## 7. 度量

| 指标 | 目标值 |
|------|--------|
| Upload API p99 latency | < 2s(含 1MiB XML + Reload + Registry refresh) |
| Delete API p99 latency | < 500ms |
| Builtin 误删拦截成功率 | 100%(`parammodel.delete.rejected_builtin` audit / `DELETE` 调用总数) |
| Backup cleanup 每日扫描覆盖率 | 100% |
| Self-healing 回退命中率 | GWT-5 场景下重启或 Reload 后 `loaded_from` 切回 builtin 比例 = 100% |
| custom XML 升级保留率 | 升级前后 `ls param-mappings-custom/*.xml` diff = 空集 |
| Prometheus 监控覆盖 | 至少 3 个指标:`parammodel_upload_total` / `parammodel_delete_total` / `parammodel_backup_cleanup_total` |
| E2E 用例覆盖 | 新增 ≥ 11 个 `check_status` 断言(GWT-1 → GWT-11 各 ≥1) |

## 8. 风险

| ID | 风险 | 缓解 |
|----|------|------|
| R-NEW-T0178-1 | 客户端上传超大 XML 触发 OOM | Gin `MaxMultipartMemory = 4 MiB` + 单文件 `≤ 1 MiB` 双重门禁;`xml.Decoder.Strict=true` 拒绝畸形 |
| R-NEW-T0178-2 | 上传成功但 Loader 解析失败 → DB 与文件状态不一致 | Upload handler 在 Reload 失败时反向 rename(`.tmp` 删除 / `.bak` 还原),保证最终一致 |
| R-NEW-T0178-3 | 备份失败导致用户反复点击 DELETE 全部失败 | 守护规则触发 Prometheus alert `ParamModelBackupFailedSurge`(rate > 0/10m,for 5m),Telegram/邮件通知运维 |
| R-NEW-T0178-4 | host 目录 `param-mappings-custom` 在 deploy.sh 首次部署被遗漏初始化 → app 启动期 Loader 扫描 non-existent dir 报错 | Loader 已对 `os.Stat(customDir).IsNotExist` 做容忍(`§3.1 verbiage`);deploy.sh 显式 `mkdir -p` 双保险 |
| R-NEW-T0178-5 | bool 字段 `custom_overrides_builtin` 默认值陷阱(Go zero value false ≠ 默认意图 true) | 改用 `*bool`,access method `CustomOverridesEnabled() bool { return c == nil || *c }`,yaml 不写 → 走默认 true |
| R-NEW-T0178-6 | 多实例横扩时同时上传同名文件竞态 | P1 不实现 PG advisory lock;`omcgo/CLAUDE.md §5.3` 写明"单实例假设",横扩前必须先补 |
| R-NEW-T0178-7 | Upload 过程中容器 OOM-Killed 留下 `.tmp.<uuid>` 残留 | BackupCleanup cron 扩展:扫到 `.tmp.<uuid>` 且 mtime > 1h 即删 |
| R-NEW-T0178-8 | builtin 同名 XML 与 custom XML 同时存在,且 `custom_overrides_builtin=false`(运维误改) | 启动期 Loader 日志 WARN 记录冲突文件名清单;UI Source 列 builtin 行展示原因 Tooltip "其他同名 custom 文件被压制" |

## 9. 实施细节(S2 设计备忘)

### 9.1 分层目录契约

```
<XMLBaseDir>/
├── param-mappings/             ← 镜像层只读, builtin
│   ├── BLQ.xml ... MLN.xml     ← 9 个出厂 paramModel
│   ├── standard-model.xml      ← 单独 Pass 2
│   ├── products.xml            ← product Loader 用
│   └── *-routing.xml           ← 历史保留,跳过
└── param-mappings-custom/      ← host bind mount, 可读写, 持久化
    ├── CBQQ.xml ... <运维自定义>
    ├── *.deleted.<14位ts>       ← DELETE 备份
    ├── *.bak.<14位ts>           ← UPLOAD 同名覆盖备份
    └── *.tmp.<uuid>             ← 进行中的 Upload(失败残留由 cleanup 清)
```

### 9.2 source 判定唯一函数

```go
// omcgo/internal/config/parammodel/source.go (新文件)
const (
    builtinPrefix = "param-mappings/"
    customPrefix  = "param-mappings-custom/"
)

func ClassifySource(loadedFrom string) Source { ... }
func IsDeletable(loadedFrom string) bool {
    return ClassifySource(loadedFrom) == SourceCustom
}
```

唯一真值源,前端不重新推导(API 直接返 `deletable` 布尔)。

### 9.3 Loader 合并算法

```
1. scan builtin → builtinFiles  ([]absPath)
2. scan custom  → customFiles   ([]absPath,容忍 dir 不存在)
3. if customOverrides:
     for f in customFiles: replace builtin entry with same basename(f)
4. for absPath in mergedFiles:
     loadParamModelFile(absPath)        # 已有逻辑不变
     入库时 loaded_from = filepath.Rel(base, absPath) | ToSlash
```

### 9.4 Upload handler 关键约束

| 校验 | 实现 |
|------|------|
| 扩展名 | `strings.HasSuffix(file.Filename, ".xml")` |
| 文件名白名单 | 正则 `^[A-Za-z0-9_-]{1,64}\.xml$` |
| 保留名 | 不在 `{standard-model.xml, products.xml, *-routing.xml}` |
| 大小 | `file.Size <= 1 << 20`(1 MiB) |
| XML 合法 | `xml.Decoder{Strict: true}` 试解析 |
| 根元素 | 必须 `<paramModel>`(beevik/etree 反射 root) |
| 路径反查 | `filepath.Rel(customDir, target)` 不含 `..` |
| 同名冲突 | `?force=true` 才覆盖,默认 409 |
| 写盘原子 | tmp + rename;defer Remove(tmp) |
| 失败回滚 | `.bak` rename / `.tmp` 删除 |
| 锁 | per-filename `sync.Map[name]*sync.Mutex`,Upload + Delete + ReloadOne 共用 |

### 9.5 Delete handler 守门顺序

```
1. GetParamModelByName(name) → ErrNotFound → 404
2. !IsDeletable(loaded_from) → 403 ErrCodeBuiltinNotDeletable
3. acquire per-filename lock
4. rename(absPath, backupPath):
     ENOENT      → log Warn, 容忍,继续
     other errno → 500 ErrCodeBackupFailed, 不删 DB, 严格回滚 ←保守语义
5. DELETE param_models WHERE name=$1 (事务,FK CASCADE 生效)
     failed → 反向 rename(backup → original) + 500
6. audit + registry.Refresh
7. 返回 {deleted:true, backup: filename}
```

### 9.6 30 天 BackupCleanup(omcgo-worker)

- 文件名正则 `\.(deleted|bak)\.(\d{14})$` 解析 ts
- `<name>.tmp.<uuid>` 单独分支:mtime > 1h 即删
- cron 表达式默认 `0 3 * * *`,从 `dict_loader.param_model.backup_cleanup_cron` 读
- 启动期延迟 30s 跑一次 catch-up(防 worker 长期宕机后堆积)
- 错误隔离:单文件失败 continue,不退出 cleanup 协程
- 指标 `parammodel_backup_cleanup_total{result="swept|error|skipped"}`

### 9.7 数据迁移 — `loaded_from` 前缀回填

```sql
-- omcgo/migrations/000NNN_param_models_loaded_from_prefix.sql
-- +goose Up
UPDATE param_models
   SET loaded_from = 'param-mappings/' || loaded_from
 WHERE loaded_from IS NOT NULL
   AND loaded_from <> ''
   AND loaded_from NOT LIKE 'param-mappings/%'
   AND loaded_from NOT LIKE 'param-mappings-custom/%';

-- +goose Down
UPDATE param_models
   SET loaded_from = regexp_replace(loaded_from, '^param-mappings(-custom)?/', '')
 WHERE loaded_from LIKE 'param-mappings%/%';
```

幂等(`NOT LIKE` 已守门),新机/老机都安全。`NNN` 取 backlog 实施期合入时的最大版本号 + 1。

### 9.8 部署改动

**`deployments/release/bundle/deploy/docker-compose.app.yml`** app + worker 容器 volumes 段加:

```yaml
app:
  volumes:
    - /opt/omc/data/param-mappings-custom:/etc/omcgo/data/param-mappings-custom

worker:
  volumes:
    - /opt/omc/data/param-mappings-custom:/etc/omcgo/data/param-mappings-custom
    # 不写 :ro — cleanup 需要 unlink
```

**`deployments/release/bundle/deploy/deploy.sh`** Step 5.x 加目录初始化:

```bash
if [ ! -d /opt/omc/data/param-mappings-custom ]; then
  mkdir -p /opt/omc/data/param-mappings-custom
  chown 10001:10001 /opt/omc/data/param-mappings-custom   # 与 Dockerfile USER 一致
  chmod 0750         /opt/omc/data/param-mappings-custom
fi
```

升级时**不动**已存在目录的权限。

**`omcgo/cmd/app/etc/config.{dev,prod}.yaml`** dict_loader 段补:

```yaml
dict_loader:
  param_model:
    directory: "param-mappings"
    custom_directory: "param-mappings-custom"
    custom_overrides_builtin: true           # *bool — yaml 不写 = nil = 默认 true
    backup_retention_days: 30
    backup_cleanup_cron: "0 3 * * *"
```

顺手修上一轮分析发现的 prod.yaml 缺 `dict_loader` 潜在 bug(LATENT BUG)。

### 9.9 前端改造点

| 文件 | 改动 |
|------|------|
| `omcmb/frontend-core/src/types/paramModel.ts` | 加 `source: 'builtin'\|'custom'\|'unknown'`、`loadedFrom: string`、`deletable: boolean` |
| `omcmb/frontend-core/src/services/api/paramModelApi.ts` | 新增 `uploadXML(file, force)` / `deleteCustomXML(filename)`(后者由后端复用 `DeleteParamModel`,前端只需 hook 既有 `useDeleteParamModel`) |
| `omcmb/frontend-core/src/hooks/api/useParamModels.ts` | 新增 `useUploadXML()` |
| `omcmb/webcode/src/pages/product/param-model/index.tsx` | toolbar 加 `<Upload>` 按钮(导入 XML 左侧) |
| `omcmb/webcode/src/pages/product/param-model/ModelsTab.tsx` | "来源" 列;操作列删除按钮按 `row.deletable` 渲染(`<a>` vs `<span style="cursor:not-allowed">` + `<Tooltip>`) |
| `omcmb/frontend-core/src/mock/services/paramModelService.ts` | mock `uploadXML` / list 返回 `source/deletable` |
| `omcmb/webcode/src/services/http.ts`(或拦截器) | 按 `code=40310/50031` 渲染友好弹窗 |

### 9.10 错误码

`omcgo/global/errors.go` 加:

```go
ErrCodeBuiltinNotDeletable = 40310   // F02:内置 XML 不可删除
ErrCodeBackupFailed        = 50031   // F02:删除前备份失败,流程已回滚
```

### 9.11 Prometheus 监控 + 告警

新增 3 个 counter:
- `parammodel_upload_total{result="success|rejected|error", reason="..."}`
- `parammodel_delete_total{result="success|rejected_builtin|aborted_backup_failed"}`
- `parammodel_backup_cleanup_total{kind="deleted|bak|tmp", result="swept|error|skipped"}`

告警规则:
```yaml
- alert: ParamModelBackupFailedSurge
  expr: rate(parammodel_delete_total{result="aborted_backup_failed"}[10m]) > 0
  for: 5m
  annotations:
    summary: "param-model delete backup failures on {{ $labels.instance }}"
```

## 10. 实施清单(工期分解)

| Phase | 内容 | 工期 | Owner |
|-------|------|------|-------|
| P1 后端 | source 模块 + Delete 守门 + Upload handler + `*bool` config + 迁移 `000NNN_param_models_loaded_from_prefix.sql` | 1.5d | Claude |
| P2 worker | `BackupCleanup` + worker cron 注册 + compose mount + 30s 启动 catch-up | 0.5d | Claude |
| P3 部署 | `docker-compose.app.yml` + `deploy.sh` 目录初始化 + `config.prod.yaml` 补 `dict_loader` + alerts.yml | 0.5d | Claude |
| P4 前端 | Upload 按钮 + Source Tag + Tooltip 置灰 + 同名 Modal.confirm + 类型 + mock | 1.0d | Claude |
| P5 E2E | `scripts/e2e_param_model_custom.sh` 11 个用例覆盖 GWT-1 ~ GWT-11 + 嵌入 `e2e_verify.sh` 主链 | 0.5d | Claude |
| P6 文档 | `omcgo/CLAUDE.md §5.3` 加分层目录 + Self-healing 说明;`docs/operations/OMC 内网离线部署手册(运维侧).md` 加 custom XML 章节;backlog/sprint 日志 | 0.5d | Claude |

**合计 4.5d**;sprint-13 内单 PR 提交(或按 P1-P3 / P4 / P5-P6 拆 3 PR review)。

## 11. 实测产物(S5 实施完毕后填)

```
(待 S5 完成填写,字段示例:
- 后端新增文件数 / 改动行数
- 前端 typecheck / vitest 通过情况
- E2E 11 用例全部 PASS
- 升级演练:dev 环境模拟 upload → 重打镜像 → recreate 容器 → CBQQ.xml 仍存且字典正确加载
- builtin 误删拦截:curl DELETE BTS → 403 确认
- backup 失败回滚:chmod 0500 custom dir → DELETE → 500 + 文件保留确认
- 30 天清理:touch X.deleted.20260101000000 → 跑一次 cleanup → 文件消失确认
)
```

## 12. 风险登记同步

提案合入后,`docs/project/risk-register.md` 加 8 条 `R-NEW-T0178-*`,Owner=Claude+user。

---

**版本历史**:2026-05-29 v1.0 起草(S0 → S2 设计备忘合并;5 轮 ULTRATHINK 决策已对齐:① `custom_overrides_builtin=*bool 默认 true` ② builtin 行置灰+Tooltip ③ 30 天备份 ④ 备份失败保守回滚 ⑤ source 唯一真值源在后端)
