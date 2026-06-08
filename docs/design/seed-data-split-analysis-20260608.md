# 种子数据拆分分析(部署数据 vs 测试数据)

> 状态:**已实施 + 全新库验证通过**(见 §11)
> 日期:2026-06-08
> 范围:`omcgo/migrations/`(schema + seed)、`omcgo/scripts/*seed*`、dictloader 启动加载
> 目标:把"部署一个全新系统必须内置的数据"与"测试样例数据"彻底分离,使生产部署得到一套**干净、最小、可开箱即用**的基础数据。

---

## 1. 背景与目标

当前 `migrations/seed/000001_init_seed.sql`(6.8 万行)是**从测试环境 `pg_dump` 出来的合并 baseline**,其头部注释自陈同时包含:

- 部署必需:admin / 字典 / 菜单 / 权限 / RBAC / 数据模型字典;
- 测试数据:**33,750 个赞比亚测试设备**、设备分组成员、设备参数、设备任务等运行时业务数据。

也就是说——**部署基础数据和测试数据混在同一个种子文件里**。新系统部署时会被强行灌入 3 万多个测试设备,这是要解决的核心问题。

**目标产物**:
1. **部署数据集(A 类)**:全新系统跑起来、能登录、菜单/权限/字典/数据模型完整所必需的最小数据。
2. **测试数据集(B 类)**:设备、告警、性能、拓扑等运行时业务样例,仅开发/E2E/压测使用,生产不加载。

---

## 2. 现状:数据来源全景

| 来源 | 文件 | 性质 | 加载方式 |
|------|------|------|---------|
| DDL | `migrations/000001..000029_*.sql`(206 张表) | 表结构 | goose 自动 |
| **主种子** | `migrations/seed/000001_init_seed.sql` | **混合(部署+测试)** | goose 自动 |
| 增量种子 | `migrations/seed/000002..000028_*.sql` | 多为部署(菜单调整、PM 内置任务、告警定义、字典单位…) | goose 自动 |
| 数据模型字典 | `datamodels/`(XML) → dictloader | 部署(products/param_models/param_mappings/product_class_patterns/indicator_*) | 应用启动期 UPSERT(`auto_load_on_startup`,test 环境默认 false) |
| E2E 测试 | `scripts/seed_e2e_testdata.sql` | 测试(UUID 前缀 `e2e0000*` 隔离) | 手动 |
| 规模测试 | `scripts/seed_test_data.sql` | 测试(generate_series,UUID 前缀 `a0000000*`,约 7 万行) | 手动 |
| 旧链路(疑似已废) | `scripts/seed.sh` → `datamodels/seed/carrier_defaults/*.json` → `data_model_definitions` 表 | — | 手动 |

> ⚠️ **`scripts/seed.sh` 与 `data_model_definitions` 表疑似为已下线的旧 datamodel JSON 链路**(CLAUDE.md:"原 JSON seed 链路已下线,所有种子收敛进 migrations/seed/";T-0098 旧 datamodel 包已下线)。`data_model_definitions` 不在当前 206 张表清单内。**待确认是否可一并清理**(见 §7-Q7)。

**关键结论**:真正的"部署 vs 测试"拆分点,几乎全部集中在 **`migrations/seed/000001_init_seed.sql` 这一个文件内部** —— 把它里面的测试数据剥出去即可。增量种子 `000002..000028` 基本都是部署数据,基本不用动。

---

## 3. 三分类框架(206 张表)

| 类别 | 含义 | 部署时 | 举例 |
|------|------|--------|------|
| **A 部署必需** | 系统跑起来必须内置 | ✅ 必须 | users/roles/menus/api_endpoints/字典/数据模型字典/告警定义/指标 |
| **B 测试样例** | 运行时业务数据,仅测试用 | ❌ 不加载 | devices/device_parameters/alarms/pm_metrics/mr/topo |
| **C 运行时空表** | 部署时为空,运行中生成 | ❌ 空表 | 日志/任务/outbox/快照/审计/dead_letters |

下面只列 `init_seed` 里**实际有数据**的 90 张表(行数为近似值,已扣除 pg_dump 头行误差)。

---

## 4. A 类:部署必需数据清单

### 4.1 RBAC 核心(必须完整,否则无法登录/鉴权)

| 表 | 行数 | 内容 | 备注 |
|----|------|------|------|
| `users` | 2 | ① `admin`(超管,`builtIn`)② `system`(系统内部账号,`builtIn`,禁密码登录) | 口令为 bcrypt,首登强制改密机制兜底 |
| `roles` | 3 | `admin`(全权限,`is_builtin=true`)/ `operator` / `viewer` | admin = "拥有系统全部权限的角色" |
| `user_roles` | 2 | admin 用户→admin 角色;system 用户→admin 角色 | |
| `role_inheritance` | 2 | admin → operator → viewer(继承链) | 全权限的实现方式之一 |
| `role_menus` | ~464 | admin 角色的全量菜单授权 | 决定登录后能看到的菜单 |
| `role_api_permissions` | ~30 | 角色的 API 权限 | |
| `role_device_groups` | 0 | (未播种) | 设备组级数据权限,默认空 |

> ✅ 你要求的"超级管理员账号 / 拥有全部权限的角色"**已存在且为内置**。超管 = `admin` 用户(`20000000-…-001`);全权角色 = `admin` 角色(`10000000-…-001`)。

### 4.2 菜单与 API 权限(必须完整)

| 表 | 行数 | 内容 |
|----|------|------|
| `menus` | ~221 | 全量系统菜单树 |
| `api_endpoints` | ~663 | 全量 API 端点(权限点) |
| `seed_menu_show_status_backups` | ~10 | 菜单显隐迁移的内部备份表 —— **建议从部署集剔除**(非业务数据) |

### 4.3 系统字典与配置

| 表 | 行数 | 内容 |
|----|------|------|
| `sys_dictionaries` | ~14 | 字典分类 |
| `sys_dictionary_details` | ~61 | 字典项(含单位等) |
| `sys_configs` | ~19 | 系统级配置项 |

### 4.4 数据模型字典(TR-069 参数/产品)

| 表 | 行数 | 内容 | 备注 |
|----|------|------|------|
| `standard_params` | ~2101 | 标准参数定义 | dictloader 启动也会 UPSERT |
| `param_mappings` | ~4741 | 参数映射(标准↔私有路径) | 同上 |
| `param_models` | ~9 | 参数模型 | 同上 |
| `products` | ~15 | 产品装配件 | 同上 |
| `product_class_patterns` | ~29 | productClass 匹配规则 | 同上 |
| `standard_commands` | **0** | 未播种 | **待确认是否需要**(见 Q6) |

> ⚠️ 这 5 张数据模型表**与 dictloader 启动加载重叠**。需决策:部署集是**保留这份 SQL** 还是**只靠 dictloader 从 `datamodels/` XML 加载**(见 §7-Q5)。

### 4.5 MML 命令目录

| 表 | 行数 | 内容 |
|----|------|------|
| `mml_commands` | ~242 | MML 命令 |
| `mml_command_groups` | ~22 | 命令分组 |
| `mml_command_sub_fields` | ~1190 | 命令子字段 |
| `mml_param_versions` | ~1 | 参数版本 |

### 4.6 告警定义

| 表 | 行数 | 内容 |
|----|------|------|
| `alarm_definitions` | ~442 | 告警定义字典 |
| `alarm_severity_levels` | ~4 | 告警级别 |

### 4.7 性能指标 / KPI 字典

| 表 | 行数 | 内容 |
|----|------|------|
| `perf_indicators_enb/gnb/gsm` | ~1411 / ~282 / ~73 | 各制式性能指标 |
| `enabled_pm_indicators_enb/gnb/gsm` | 同量级 | 启用的 PM 指标 |
| `rela_platform_indicator_formula_enb/gnb/gsm` | ~5910 / ~282 / ~73 | 指标公式关系 |
| `indicator_unit` | ~27 | 指标单位 |
| `indicator_group_enb/gnb/gsm` | ~23 / ~16 / ~9 | 指标分组 |
| `kpi_definitions` | ~12 | KPI 定义 |
| `mr_indicators` | ~5 | MR 指标 |
| `ufte_task_types` | ~6 | 升级/文件传输任务类型字典 |

> 注:PM 内置采集任务(`pm_tasks` 的 builtin)由增量种子 `seed/000010_seed_pm_builtin_tasks.sql` 提供,已是部署数据,不在 init_seed 内。

---

## 5. B 类:测试样例数据清单(部署应排除)

| 表 | 行数 | 性质 |
|----|------|------|
| `devices_cmcc/ctcc/cucc/other` | **约 33,800** | 赞比亚测试设备(核心测试数据) |
| `device_group_members` | ~9967 | 设备分组成员 |
| `device_groups` | ~21 | 设备分组(**灰色**,见 Q1) |
| `device_info` | 2 | 设备扩展信息 |
| `device_parameters_p00..p31` | 合计数千 | 设备参数实例 |
| `device_tasks_p05 / p13` | ~56 / ~1 | 设备任务实例 |
| `alarms_active` | 2 | 活动告警实例 |
| `provisioning_tasks` | ~2 | 开站任务实例 |
| `parameter_discovery_log` | ~1 | 参数发现日志 |
| `pm_dashboards` | ~12 | 仪表盘(**灰色**,见 Q2) |
| `northbound_servers` | 3 | 北向 OSS 端点(**灰色**,见 Q3) |
| `ops_templates` | ~6 | 运维模板(**灰色**,见 Q4) |

### 安全敏感 / 运行时(无条件排除)

| 表 | 行数 | 原因 |
|----|------|------|
| `api_keys` | 2 | system 用户的全权限(`{*}`)内部 key —— **凭证,绝不能进部署 baseline**,应运行时生成 |
| `async_jobs` | ~1 | 运行时任务执行记录(pm 聚合 succeeded) |
| `async_jobs_cron_state` | ~2 | cron 运行状态(**灰色**,见 Q8) |
| `audit_logs` | ~4 | 审计日志(运行时) |

> 另有 `scripts/seed_e2e_testdata.sql`(E2E,~90 行/35 表,`e2e0000*` 前缀)和 `scripts/seed_test_data.sql`(规模,~7 万行/25 表,`a0000000*` 前缀)两份**已经独立的测试脚本**,本就不随 goose 自动跑,无需改动——它们正是测试数据应该待的地方。

---

## 6. 拆分落地方案(候选)

### 方案 A(推荐):重建纯净部署 baseline + 测试数据归位

1. 用 `pg_dump --data-only --inserts` **只导出 §4 的 A 类表**,生成新的 `migrations/seed/000001_init_seed.sql`(纯部署),替换现有混合版。
2. 现 init_seed 里的 B 类测试数据(33,800 设备等)**剥离**:需要时统一用现有 `scripts/seed_test_data.sql` 生成,或新增一份 `scripts/seed_dev_baseline.sql` 承载这份赞比亚设备集。
3. 生产部署:goose 只跑 `migrations/`(含纯净 seed)→ 得到干净基础数据;开发/测试:额外手动跑测试脚本。
4. 安全:剔除 `api_keys`、`async_jobs*`、`audit_logs`、`seed_menu_show_status_backups`。

**优点**:与现有"goose 自动跑 seed + scripts 手动跑测试"的机制天然契合,生产零测试数据;**缺点**:需重新生成 baseline 并核对外键顺序。

### 方案 B:环境开关控制

goose 迁移本身**无法条件执行**(它顺序全跑)。因此"开关"只能落在应用/脚本层:把测试数据从 goose 链路移出(同方案 A 第 2 步),用 `OMCGO_ENV` / 启动参数决定是否加载测试脚本。本质仍需先做方案 A 的剥离,故**方案 A 是前提**。

> **倾向方案 A**。

---

## 7. 待确认问题(请逐条拍板)

| # | 问题 | 我的建议 |
|---|------|---------|
| **Q1** | `device_groups`(~21)是否需要在部署集保留一个默认根分组? | 倾向**全部归测试**(迁移 `seed/000017_clear_default_group_memberships` 已清理默认成员关系);如业务要求有默认分组再单独 seed 1 条 |
| **Q2** | `pm_dashboards`(~12)是内置仪表盘还是用户创建? | 倾向**测试**(`seed/000011_drop_pm_builtin_dashboards` 已删内置);确认后定 |
| **Q3** | `northbound_servers`(3)是默认 OSS 端点还是测试配置? | 倾向**测试**(含具体地址/凭证) |
| **Q4** | `ops_templates`(~6)是内置运维模板还是测试? | 倾向**部署**(预置模板),需你确认内容 |
| **Q5** | 数据模型字典(standard_params/param_mappings/param_models/products/product_class_patterns)是**保留 SQL seed** 还是**只靠 dictloader 从 XML 加载**? | 倾向**保留 SQL**(开箱即用 + 生产首次确定性),但须保证与 `datamodels/` XML 一致,避免双份漂移 |
| **Q6** | `standard_commands`(0 行)新系统是否需要? | 待你确认该表是否在用;不用则忽略 |
| **Q7** | `scripts/seed.sh` + `data_model_definitions`(旧 JSON 链路)是否可删除? | 倾向**删除**(已下线) |
| **Q8** | `async_jobs_cron_state`(~2)是否含必须的内置 cron 调度状态? | 倾向**排除**(运行时自建);确认后定 |
| **Q9** | 超管默认口令策略:部署 baseline 继续用现有 bcrypt 默认 hash + 首登强制改密,还是部署时随机生成? | 倾向**沿用首登改密**(与现状一致),文档提示运维 |

---

## 8. 风险与注意点

- **外键依赖顺序**:重建 baseline 须保证插入顺序(pg_dump 默认按依赖排序 + `--disable-triggers`)。
- **TimescaleDB**:seed 用 `timescaledb_pre_restore()/post_restore()` 包裹,重建时保留。
- **dictloader 重叠**:若 Q5 决定保留 SQL,需与 XML 对齐;若决定只靠 dictloader,生产须确保 `auto_load_on_startup` 行为正确。
- **凭证安全**:`api_keys` 必须排除;超管口令靠首登改密兜底。
- **`is_builtin` / `builtIn` 标志**:users/roles 用它区分内置与测试账号,清理测试数据时按"非 builtin"删,避免误删超管。
- **增量种子**:`seed/000002..000028` 基本是部署数据(菜单/告警/PM 内置任务/字典),拆分时**保持不动**,只动 `000001`。

---

## 9. 确认后下一步(实施预告,本次不做)

1. 按确认结论生成纯净 `000001_init_seed.sql`(仅 A 类)。
2. 把赞比亚设备等 B 类剥离到 `scripts/` 测试脚本。
3. 更新 `migrations/seed/README.md` 说明"部署 seed vs 测试 seed"边界。
4. 跑 `e2e_verify.sh` + 全新库 goose 迁移验证开箱可登录、菜单/权限完整。

> **请针对 §7 的 Q1–Q9 给出结论,我再据此实施拆分。**

---

## 10. 确认结论与最终拆分清单(2026-06-08 已确认)

### 10.1 决策记录

| # | 决策 |
|---|------|
| Q1 | device_groups **只保留系统内置组**:`未分组设备`(`…002`,`is_builtin=true`)+ 其依赖的 level-1 父组 `默认设备组`(`…001`,`is_builtin=true`)。其余示例组(移动/电信/联通设备域、各地区组、测试组、自动分组等,均 `is_builtin=false`)**全删** |
| Q2 | `pm_dashboards` → 测试,**移除** |
| Q3 | `northbound_servers` → 测试,**移除** |
| Q4 | `ops_templates` → 部署,**保留** |
| Q5 | 数据模型/字典 **改由 dictloader 从 XML 加载**,从 SQL seed **移除全部 dictloader 管理表**(见 10.2) |
| Q7 | `scripts/seed.sh` + 旧 `data_model_definitions` JSON 链路 **删除** |
| Q8 | `async_jobs_cron_state` → **移除**(运行时自建) |
| Q9 | 超管口令沿用 **bcrypt 默认 + 首登强制改密**,部署不随机生成 |

> Q6(`standard_commands`):0 行且非 dictloader 管理,无需处理。

### 10.2 从 seed 移除 → 改由 dictloader/XML 提供(已代码核实)

dictloader 在 app 启动期(`auto_load_on_startup`,**dev/prod 均为 `true` 已核实**)UPSERT 以下表,SQL seed 不再内置(单源于 `datamodels/` XML):

```
param_models, param_mappings, standard_params, discovered_param_mappings,
products, product_class_patterns,
indicator_group_enb/gsm/gnb,
perf_indicators_enb/gsm/gnb,
rela_platform_indicator_formula_enb/gsm/gnb,
enabled_pm_indicators_enb/gsm/gnb,
alarm_definitions,
mml_param_versions, mml_command_groups, mml_commands,
mml_command_sub_fields, mml_group_param_rel
```

代码佐证:`internal/config/parammodel/loader.go`、`internal/product/loader.go`、`internal/pm/indicator/loader.go`、`internal/alarm/definition/loader.go`、`internal/config/parammodel/mmlstandardloader/seed_importer.go`。

### 10.3 从 seed 移除 → 测试/运行时/安全

```
devices_cmcc/ctcc/cucc/other(约 3.38 万),device_group_members,device_info,
device_parameters_p00..p31,device_tasks_p05/p13,alarms_active,
provisioning_tasks,parameter_discovery_log,
pm_dashboards(Q2),northbound_servers(Q3),
api_keys(凭证!),async_jobs,async_jobs_cron_state(Q8),audit_logs,
seed_menu_show_status_backups(内部备份表),
device_groups 中 is_builtin=false 的全部示例组(Q1)
```

### 10.4 保留在 seed(部署必需,最终清单)

| 域 | 表 |
|----|----|
| RBAC | `users`(admin+system)、`roles`、`user_roles`、`role_inheritance`、`role_menus`、`role_api_permissions` |
| 菜单/API | `menus`、`api_endpoints` |
| 系统字典 | `sys_dictionaries`、`sys_dictionary_details`、`sys_configs` |
| 告警级别 | `alarm_severity_levels`(dictloader 只读它,seed 必须提供) |
| 指标辅助 | `indicator_unit`(seed 初始化)、`kpi_definitions`、`mr_indicators` |
| 任务字典 | `ufte_task_types` |
| 运维模板 | `ops_templates`(Q4) |
| 设备组 | `device_groups` 仅 `默认设备组`(`…001`)+ `未分组设备`(`…002`) |

> 增量种子 `migrations/seed/000002..000028` 基本是部署数据(菜单/PM 内置任务/字典调整),**保持不动**。但其中涉及 dictloader 管理表的增量(若有)也需一并清理,实施时逐个核对。

### 10.5 实施步骤(确认后执行)

1. 重建 `migrations/seed/000001_init_seed.sql`:删除 10.2 + 10.3 全部表的 INSERT 块,保留 10.4;device_groups 仅留两行内置组。保留 TimescaleDB pre/post-restore 包裹。
2. 删除 `scripts/seed.sh`(Q7);确认无残留 `data_model_definitions` 引用。
3. 测试数据归位:33,800 设备等如开发需要,用 `scripts/seed_test_data.sql` 生成(已存在)。
4. 更新 `migrations/seed/README.md` 写明"部署 seed vs 测试 seed"边界。
5. 全新库验证:goose 迁移 → 启动 app(dictloader 自动灌字典)→ 能登录、菜单/权限/字典/数据模型完整。

### 10.6 新增风险(实施前必读)

- **E2E/test 环境 `auto_load_on_startup` 未设置(默认 false)**:`config.test.yaml` 没开自动加载。从 seed 移除字典后,test 环境若不启 dictloader 将**字典全空**。实施时需二选一:① `config.test.yaml` 显式 `auto_load_on_startup: true`;② E2E 已通过运行真实 app 触发 dictloader(需验证)。**这是本次拆分最大的连带风险点。**
- dictloader 启动顺序/失败处理:字典从"迁移时就位"变为"app 启动后就位",首次启动到 dictloader 完成之间字典短暂为空,确认无依赖该窗口的逻辑。
- `device_group_members` 全删后,被删的示例组无成员,无 FK 悬挂(设备本身也删了)。
- `mml_command_sub_fields` 有 AFTER INSERT 触发器(重算 target_paths),改由 dictloader 写时触发器照常生效,无影响。

---

## 11. 实施完成与验证(2026-06-08)

### 11.1 落地改动

| 文件 | 改动 |
|------|------|
| `omcgo/migrations/seed/000001_init_seed.sql` | 68589 → 2208 行。剥离 §10.2(dictloader→XML)+ §10.3(测试/运行时/安全)全部数据段;`device_groups` 过滤为 2 个内置组(`默认设备组`+`未分组设备`)。保留 §10.4 部署必需 18 张表 + TimescaleDB 包裹 + goose 标记 |
| `omcgo/cmd/app/etc/config.test.yaml` | 新增 `dict_loader` 段,`auto_load_on_startup: true`(Option A,与 dev/prod 一致) |
| `omcgo/scripts/seed.sh` | 删除(Q7,旧 JSON 链路;`datamodels/seed/` 已空) |
| `omcgo/migrations/seed/README.md` | 重写,说明部署 seed vs 测试 seed 边界 + 正确执行方式 |

### 11.2 全新库端到端验证(临时 timescaledb 容器)

**正确执行方式**(生产同款,独立版本表):
```bash
go run ./cmd/migrate up --path migrations                                          # schema
GOOSE_TABLE=goose_db_version_seed go run ./cmd/migrate up --path migrations/seed   # seed
```
结果:schema → v29 OK,seed → v28 OK(含此前在错误验证方式下 FK 失败的 `000022`)。

验证数据(全新库 omcgo_v2):
- **部署必需有数据**:users 2(admin+system)、roles 3、menus 228、role_menus 472、api_endpoints 663、role_api_permissions 30、sys 字典齐、device_groups 2、ops_templates 6、ufte_task_types 11 ✓
- **dictloader 表全为 0**:param_mappings / standard_params / products / alarm_definitions / perf_indicators_enb / mml_commands / mml_command_sub_fields —— 启动期由 XML 灌 ✓
- **测试/安全数据全为 0**:devices_cmcc / device_group_members / api_keys ✓

### 11.3 验证方法的坑(重要)

`make migrate-up`(`--paths migrations,migrations/seed`)对两个目录共用**默认**版本表 `goose_db_version`;schema 跑完后 version_id 1..N 已占,seed 的 `000001` 等因撞号被当"已应用"**跳过** → `init_seed` 不执行 → 后续增量 seed FK 落空。**全新库验证必须用独立 `GOOSE_TABLE=goose_db_version_seed`**(生产 compose 即如此)。原始混合 baseline 在该错误方式下同样"失败",证明与本次拆分无关。

### 11.4 未尽事项 / 提示

- **赞比亚 3.3 万测试设备**已从 baseline 移除;如 MML 控制台产品解析测试需要该特定数据集,从 git `7d4ae90f^` 取回或用 `scripts/seed_test_data.sql` 生成。
- **生产首启时序**:字典从"迁移即就位"变为"app 启动后由 dictloader 就位";首启到 dictloader 完成间字典短暂为空(auto_load=true 已保证会加载)。
- `data_model_definitions` 在 3 个 Go 文件(`core/appconfig`、`core/model/parameter`、`core/event/subjects`)仍有遗留引用,但表已不在 schema 中,属历史死引用,**本次未触碰**(超出种子拆分范围)。
