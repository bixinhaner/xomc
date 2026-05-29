# F03 KPI 指标库页面重设计 + 自定义 XML 持久化(T-0180)

**Owner**: Claude + user
**Sprint**: sprint-13
**功能域**: F03 PM(指标库)
**对标方案**: T-0178(param-model 自定义 XML 分层目录) + T-0179(alarm-library 页面 drill-down)
**设计文档**: `docs/design/kpi-library-redesign-20260529.md`(468 行,12 节)

---

## 1. 业务背景

当前 `product/kpi-library` 页面与后端实现存在 4 个问题:

1. **UI 结构与 product/param-model 不一致** — Tabs 横排 4 个(ENB/GSM/GNB/单位定义),与 param-model 的 drill-down 主从模式割裂,运维认知成本高
2. **缺自定义 XML 持久化能力** — Loader 只扫 `data/indicator-library/` 镜像层目录,运维无法通过 UI 上传自有指标 XML;`docker cp` 进容器临时上传 → 升级 recreate 即丢
3. **"XML 导入/重载"按钮合并语义模糊** — 一个按钮承担"加法 UPSERT" + "destructive 全量重载"两种不可逆操作,用户无法明确告诉系统"我要哪种";对比 param-model 已拆分为独立两个按钮
4. **单位定义占用一个 Tab 空间** — 单位是跨制式共享资源(73 行),独立 Tab 占布局头部 25% 宽度;放抽屉更合适

T-0178 + T-0179 已沉淀分层目录 + drill-down 主从两套范式,本次直接复用,避免重新发明轮子。

---

## 2. 用户故事

| # | 角色 | 我想 | 以便 |
|---|------|------|------|
| US-1 | 运维 | 上传自定义指标 XML,且升级镜像后仍生效 | 接入新厂商/新版本指标无需修改主仓库代码 |
| US-2 | 运维 | 在 UI 上看到"内置 / 自定义"分类,且只能删除自己上传的 | 内置 XML 由产品发版决定,不能被误删 |
| US-3 | 运维 | "导入 XML" 与 "重载 XML" 两个独立按钮,各自带不可逆警告 | 区分加法补齐 vs 全量重置,降低误操作风险 |
| US-4 | 运维 | 一级页面列表态显示三个制式(ENB/GSM/GNB)各自指标数 + 来源分布 | 一眼看清全局 + 一键 drill-down 到细节 |
| US-5 | 运维 | "单位定义" 抽屉打开内含完整 CRUD | 单位是共享资源,不占主流程入口 |

---

## 3. 验收标准(Given-When-Then,11 条)

### XML 上传链路(GWT-1..3)

**GWT-1: 自定义 XML 上传成功 + 持久化**
- **Given**: 运维登录后访问 `/product/kpi-library`,host 上 `/opt/omc/data/indicator-library-custom/enb/` 为空
- **When**: 点击"上传 XML" → 选目标制式"ENB (LTE)" → 选 `MY_ENB.xml`(< 1MB,合法 `<indicatorModel>` 根元素)→ 确认
- **Then**: ① 后端返 200,文件落 `/opt/omc/data/indicator-library-custom/enb/MY_ENB.xml`; ② 自动触发 Loader.Reload,DB `perf_indicators_enb.loaded_from = 'indicator-library-custom/enb/MY_ENB.xml'`; ③ 主表 ENB 行"X 内置 / 1 自定义"计数 +1; ④ 二级表"来源"列显"自定义" Tag

**GWT-2: 上传同名文件触发 409 → force 覆盖**
- **Given**: `/opt/omc/data/indicator-library-custom/enb/MY_ENB.xml` 已存在
- **When**: 再次上传同名文件
- **Then**: ① 后端首次返 409 + 错误消息 `file already exists`; ② 前端弹 Modal 二次确认"覆盖? (旧文件备份为 .bak.<ts>)"; ③ 用户确认后 force=true 重试,后端原子写新文件 + `mv MY_ENB.xml MY_ENB.xml.bak.<ts>`

**GWT-3: 上传校验失败保守拒绝**
- **Given**: 准备上传一个不合法文件(含路径分隔符的文件名 / 非 XML 内容 / 超 1MB / 根元素非 `<indicatorModel>`)
- **When**: 点上传
- **Then**: 后端任一校验失败立即返 400 + 详细错误码 `ErrCodeIndicatorUploadInvalidName / InvalidXML / TooLarge / InvalidRoot`,不落盘;前端 message.error 提示

### XML 删除链路(GWT-4..6)

**GWT-4: 删除自定义 XML + 级联清理**
- **Given**: `indicator-library-custom/enb/MY_ENB.xml` 已上传,DB 中有 5 个 indicators + 8 条公式 + 3 条启用记录
- **When**: 详情顶部"管理 XML 文件" → 选 MY_ENB.xml 行 → 删除 → 确认
- **Then**: ① 物理 `mv MY_ENB.xml MY_ENB.xml.deleted.<ts>`; ② DB 事务删 `perf_indicators_enb` 5 行 + 级联删 `rela_platform_indicator_formula_enb` 8 条 + `enabled_pm_indicators_enb` 3 条; ③ `indicator_group_enb` 不动; ④ 主表行计数刷新; ⑤ 缓存版本号 INCR

**GWT-5: 内置 XML 删除被守门拒绝**
- **Given**: 用户访问"管理 XML 文件"看到 `indicator-library/enb/ALL.xml` (builtin) 行
- **When**: 该行删除按钮已置灰 + Tooltip "内置 XML 不可在线删除";若用户绕过 UI 直接 curl `DELETE /indicators/files/...?path=indicator-library/enb/ALL.xml`
- **Then**: 后端 `IsDeletable(loadedFrom)` 返 false → 返 403 + `ErrCodeIndicatorBuiltinNotDeletable`,文件与 DB 均不变

**GWT-6: 删除时备份失败保守回滚**
- **Given**: host 磁盘满或权限问题导致 `mv .deleted.<ts>` 失败
- **When**: 用户点删除
- **Then**: ① 不删 DB 任何行; ② 返 500 + `ErrCodeIndicatorBackupFailed`; ③ 物理文件原状保留; ④ 用户重试若空间恢复则成功(操作无残留半完成态)

### 一级 + 二级 UI(GWT-7..9)

**GWT-7: 一级列表态显三行 + 制式行点击下钻**
- **Given**: 用户访问 `/product/kpi-library`,无 URL 参数
- **When**: 页面首次渲染
- **Then**: ① 顶部 Card 含[搜索 制式/描述]+[单位定义][上传 XML][导入 XML][重载 XML][刷新缓存] 5 个按钮; ② 主区域显三行表格:ENB(LTE) / GSM / GNB(5G NR),各行有"指标数 / 分组数 / XML 文件(M 内置/N 自定义) / 平台 / [详情]"列; ③ 制式名 button.link 点击或行"详情"按钮 → 进入 drill-down 二级

**GWT-8: 二级详情 + URL 同步 + 返回按钮**
- **Given**: 用户从一级点 "ENB (LTE)" 进入二级
- **When**: 二级页面渲染
- **Then**: ① URL 同步成 `?tech=enb`; ② 顶部 Card 显[←返回][ENB (LTE) 的指标列表][管理 XML 文件][刷新缓存]; ③ F5 刷新仍在二级(URL state 留存); ④ 主区域显该制式所有指标(含"来源"列 Tag)

**GWT-9: 单位定义抽屉打开**
- **Given**: 用户在一级页面
- **When**: 点"单位定义"按钮
- **Then**: ① 右侧抽屉 width=720 打开; ② 内容 = 现 `IndicatorUnitsTab` CRUD; ③ 不影响主区域当前列表/详情态; ④ 抽屉关闭后状态恢复

### 重载语义(GWT-10..11)

**GWT-10: "导入 XML" 加法 UPSERT 不删孤儿**
- **Given**: DB 中存在某指标 `OLD.indicator` 来自历史 `indicator-library/enb/REMOVED.xml`(物理文件已删)
- **When**: 用户点"导入 XML" → Popconfirm 显加法语义描述 → 确认
- **Then**: ① 后端 mode=import,扫描当前 XML 全部 UPSERT 现存指标; ② `OLD.indicator` 不删(孤儿保留); ③ message.success 显"已导入 N 条"

**GWT-11: "重载 XML" destructive 删孤儿**
- **Given**: 同 GWT-10 初始状态
- **When**: 用户点"重载 XML" → 红色 Popconfirm 显 destructive + 不可撤销描述 → 确认
- **Then**: ① 后端 mode=reload,记录 start = NOW(); ② UPSERT 所有 XML 内指标; ③ DELETE FROM perf_indicators_enb WHERE updated_at < start; ④ 级联删 rela_platform_indicator_formula_enb + enabled_pm_indicators_enb; ⑤ 返回孤儿删除数,前端 message.success 显"已重载 N 条 + 删 M 个孤儿"; ⑥ Prometheus `indicator_reload_orphans_deleted_total{tech="enb"}` += M

---

## 4. 运营商差异

**本条目运营商无差异** — KPI 指标库是 OMC 内部数据字典,与运营商 carrier 维度正交。三大运营商使用同一份指标定义。

CMCC/CTCC/CUCC 一致。

---

## 5. 非目标(本次不做)

1. 不重写指标的启用/禁用 / 公式 / 平台映射模型(独立任务链)
2. 不动 `perf_indicators_{enb,gsm,gnb}` 三表分表结构(三表合一是长期重构)
3. 不引入"指标行级"删除 — 删除粒度统一在 XML 文件层(对齐 param-model)
4. 不动 `indicator-units` 表与 CRUD(只迁移 UI 容器到抽屉)
5. 不补 `indicator_group_*` 的 loaded_from(分组跨文件共享,删文件不动分组)
6. 不做指标公式 XML 化(公式只能 UI 单条编辑)
7. 不做单位定义 XML 化
8. 不做多实例横扩兼容(单实例假设,与 T-0178 一致)

---

## 6. 依赖

- **T-0098** ✅ 已 done — ProductRegistry + ParamRegistry 基础设施
- **T-0178** ✅ 已 done — param-model 分层目录 source.go / upload.go / filelist.go / handler.go / backup_cleanup.go / Loader 双目录范式(完整复用)
- **T-0179** ✅ 已 done — alarm-library drill-down 范式 + URL 同步(useSearchParams)前端模板

---

## 7. 度量(成功指标)

| 指标 | 类型 | 期望阈值 |
|------|------|---------|
| `indicator_upload_total{result}` | counter | 7 天内 ≥ 1 次"成功"上传(运维灰度验证) |
| `indicator_delete_total{source,result}` | counter | builtin source 误调用次数 = 0(守门有效) |
| `indicator_reload_orphans_deleted_total{tech}` | counter | reload 调用一次后立即可见 |
| `indicator_backup_cleanup_total{kind,result}` | counter | worker cron 每日 03:00 触发一次 |
| 后端 P99 latency(/indicators/summary) | gauge | < 200ms(单 SQL 三表 GROUP BY) |
| 前端首屏 LCP | 浏览器实测 | 一级页面 < 1.5s |
| 三皮肤 typecheck | CI | webcode + webcode-v2 + webcode-v3 全过 |

---

## 8. 5 项设计决策(2026-05-29 用户 L1=A3 模式 1 轮锁定)

| ID | 决策 | 选择 | 依据 |
|----|------|------|------|
| **D1** | 保留名白名单 | **不设保留名,custom 可覆盖内置同名** | 沿用 T-0178 `custom_overrides_builtin=true`;文档 §3.1 自述"沿用" |
| **D2** | 自定义 XML 根元素强制属性 | **`platform` 必填 + `deviceType` 必须与上传 `?tech=` 一致** | 防呆,小代码成本换大 UX 改善;文档 §10 倾向"防止用户传错制式" |
| **D3** | "重载 XML" 是否 dry-run preview | **简化(沿用 T-0178 destructive 语义,红色 danger Popconfirm + 描述提示)** | 文档 §8 自述"沿用 param-model 红色 danger";dry-run 是 feature creep |
| **D4** | 单位定义容器 | **抽屉(右侧 width=720)** | 文档 §2.4 自述"本文档采用" |
| **D5** | drill-down 详情是否按 XML 文件分组折叠 | **A 平表 + 来源列 + 管理 XML Modal** | 文档 §10 自述"倾向 A",实现简单,与 param-model 一致 |

---

## 9. 设计备忘(S2)

### 9.1 目录与表结构

**新目录**:
- 镜像层(builtin): `data/indicator-library/{enb/*.xml, GSM.xml, GNB.xml}`(沿用现状)
- host 持久化(custom): `/opt/omc/data/indicator-library-custom/{enb,gsm,gnb}/`(新增)

**migration 000217 ALTER 三张表加 loaded_from 列**:

```sql
ALTER TABLE perf_indicators_enb ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);
ALTER TABLE perf_indicators_gsm ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);
ALTER TABLE perf_indicators_gnb ADD COLUMN IF NOT EXISTS loaded_from VARCHAR(256);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_enb_loaded_from ON perf_indicators_enb(loaded_from);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gsm_loaded_from ON perf_indicators_gsm(loaded_from);
CREATE INDEX IF NOT EXISTS idx_perf_indicators_gnb_loaded_from ON perf_indicators_gnb(loaded_from);
```

无回填,历史数据 NULL → SourceUnknown → 不可删(安全)。Reload 后自动写入。

### 9.2 source 分类器(P1.1 起手 commit)

`omcgo/internal/pm/indicator/source.go` 镜像 `parammodel/source.go` 73 LOC 结构:

```go
const (
    BuiltinDirPrefix = "indicator-library/"
    CustomDirPrefix  = "indicator-library-custom/"
    BuiltinDirSubdir = "indicator-library"
    CustomDirSubdir  = "indicator-library-custom"
)

type Source string
const (
    SourceBuiltin Source = "builtin"
    SourceCustom  Source = "custom"
    SourceUnknown Source = "unknown"
)

func ClassifySource(loadedFrom string) Source { /* 前缀判定 */ }
func IsDeletable(loadedFrom string) bool      { /* 仅 custom = true */ }
```

`source_test.go` 镜像 parammodel/source_test.go 三个测试函数(ClassifySource 11 case + IsDeletable 6 case + SourceConstants 5 断言),共 22 个表驱动 case。

### 9.3 后端 API 6 个端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/indicators/summary` | GET | 新增,三行制式汇总 |
| `/indicators/import-directory?mode=import\|reload` | POST | 改造,加 mode 参数 |
| `/indicators/upload-xml?tech=enb\|gsm\|gnb&force=` | POST | 新增,multipart 上传 |
| `/indicators/files?tech=` | GET | 新增,列出该制式 XML 文件 |
| `/indicators/files/<path>` | DELETE | 新增,删除自定义 XML |
| `/indicator-units[/*]` | GET/POST/PUT/DELETE | 保持现状 |

### 9.4 错误码(global/errors.go,Data Model 块 20XX 段后续)

| 错误码 | 含义 |
|--------|------|
| `ErrCodeIndicatorBuiltinNotDeletable = 2040` | DELETE 内置返 403 |
| `ErrCodeIndicatorBackupFailed = 2041` | DELETE 备份失败保守回滚返 500 |
| `ErrCodeIndicatorUploadInvalidTech = 2042` | upload-xml 的 tech 不在 enb/gsm/gnb |
| `ErrCodeIndicatorUploadInvalidName = 2043` | 文件名违规 |
| `ErrCodeIndicatorUploadInvalidRoot = 2044` | XML 根元素非 indicatorModel |
| `ErrCodeIndicatorUploadTooLarge = 2045` | 文件 > 1MiB |

(具体段号在写 errors.go 时按可用值定,本表数字仅为占位估算)

### 9.5 前端文件清单

```
omcmb/webcode/src/pages/product/kpi-library/
├── index.tsx                 重写:列表↔详情 + 顶部统一 toolbar
├── SummaryTab.tsx            新建:三行制式表
├── IndicatorsByTech.tsx      新建:drill-down 详情
├── XMLFilesModal.tsx         新建:管理 XML 文件 Modal
├── UploadXmlModal.tsx        新建:上传 XML 表单
├── UnitsDrawer.tsx           新建:抽屉壳
└── IndicatorUnitsTab.tsx     保留:抽屉内容
```

业务层:`frontend-core/src/services/api/indicatorLibraryApi.ts` + `hooks/api/useIndicatorsLibrary.ts` 加 5 个 API/hook(summary/reloadDirectory/uploadXml/listFiles/deleteFile)+ types/indicatorLibrary.ts 加 5 个 type。

### 9.6 部署改动

- `deployments/docker/docker-compose.app.yml` app + worker 加 volume `/opt/omc/data/indicator-library-custom:/etc/omcgo/data/indicator-library-custom:rw`
- `deployments/release/bundle/deploy/deploy.sh` 在"建立目录结构"步加 `mkdir -p /opt/omc/data/indicator-library-custom/{enb,gsm,gnb}` + `chown 1000:1000`
- worker cron `BackupCleanup` 扩展扫 `indicator-library-custom/**/*.{deleted,bak,tmp}.*`(30 天 / 1 小时阈值同 T-0178)

---

## 10. 实施分阶段(P1-P5,合计 ~6 天 / 9 commits)

| 阶段 | 内容 | 估时 | 阻塞 |
|------|------|------|------|
| **立项** | PRD + backlog + 设计文档 incl. | 0.1d | - |
| **P1.1** | source.go + tests | 0.5d | - |
| **P1.2** | Loader 双目录合并 + filelist.go + migration 000217 + loaded_from 写入 | 1d | P1.1 |
| **P1.3** | Delete 守门 + 错误码 + handler 端点 | 1d | P1.2 |
| **P1.4** | Upload + 验证器 + summary + files 端点 | 1d | P1.3 |
| **P1.5** | import-directory 加 mode=reload 分支 + 三表孤儿删除 | 0.5d | P1.4 |
| **P2** | worker BackupCleanup 扩展 + 新指标 | 0.3d | P1.4 |
| **P3** | docker-compose volume + deploy.sh mkdir | 0.3d | P2 |
| **P4** | 前端业务层 + UI 6 文件 | 1.5d | P1.5 全后端上线 |
| **P5** | docs 收尾(CLAUDE.md §5.3.2 + risk-register + DoD) | 0.3d | P4 |

每阶段独立 commit,共 9 commits。

---

## 11. 风险登记(6 条 R-NEW-T0180-*)

| 风险 ID | 描述 | 等级 | 缓解 |
|---------|------|------|------|
| R-NEW-T0180-1 | 三表 enb/gsm/gnb 分表导致 handler 重复代码 | P3 | 抽 tech 参数 + 共用 helper,接受 ~20% 重复换 schema 简单 |
| R-NEW-T0180-2 | loaded_from NULL 历史数据被 reload 误删 | P2 | reload 仅删 updated_at < start 的行,从未被 Loader 写入 loaded_from 的旧行也满足 < start → 被删;**接受**(运维通过先 reload 一次回填) |
| R-NEW-T0180-3 | reload 操作误伤运维手动启用的孤儿指标 | P2 | UI 红色 Popconfirm + 详细描述 + 操作不可撤销提示;沿用 T-0178 |
| R-NEW-T0180-4 | 删除时备份失败导致 DB 与文件不一致 | P2 | 保守回滚:备份失败立即 return 500,不动 DB |
| R-NEW-T0180-5 | 单实例假设下同名 Upload 并发竞态 | P2 | `acquireFileLock(basename)` per-filename mutex;多实例需补 PG advisory lock |
| R-NEW-T0180-6 | 三皮肤 alarm-library/kpi-library 一致性漂移 | P2 | 本次仅改 webcode 主皮肤;webcode-v2/v3 由 PgM 决策是否同步 |

---

## 12. 关联

- T-0178 PRD: `docs/project/prd/F02-param-model-custom-xml.md`
- T-0179 PRD: `docs/project/prd/F04-alarm-library-redesign.md`
- 本任务设计文档: `docs/design/kpi-library-redesign-20260529.md`
- 现状代码:
  - 后端: `omcgo/internal/pm/indicator/`
  - 前端: `omcmb/webcode/src/pages/product/kpi-library/`
- 参考样板:
  - `omcgo/internal/config/parammodel/source.go`(P1.1 起手镜像目标)
  - `omcgo/internal/config/parammodel/upload.go`
  - `omcgo/internal/config/parammodel/filelist.go`
  - `omcmb/webcode/src/pages/product/param-model/index.tsx`
