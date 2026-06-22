# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-22 |
| 提交 | 6a6b6902（base，本次为 working tree） |
| 作者 | wangyong |
| 范围 | fullstack-provision-device |
| 关联 Issue | Closes #549 |
| 跟进 Issue | #550 / #551 / #552（pre-existing failures，与本次无关） |
| 变更文件数 | 7（6 modified + 1 new） |
| 新增行数 | +94（含新增测试 ~45） |
| 删除行数 | -36 |

## 变更概要

修复 BSC（osmo-bsc）设备首次同步后，前端 QuickSettings 临区/TRX 表项仅显示部分 BTS 行的死锁问题。根因：（1）`hintFloor=32` 在 BSC cold-start 不足以触发 object 前缀展开，CPE 返回 ~1.5MB SOAP body 经 NATS 时被 `maximum payload exceeded` 截断；（2）sync GPV task 沿用全局 `ExpiresIn=120s`，慢设备 200+ task 后半批被 `ExpiredSweeper` 抢先标 expired；（3）`NumofTrxChannel` / `OmlConnectState` 应固定走 `Bts.0` 而非 `Bts.{i}`。三处一并根治；前端 hardcode workaround 改为 XML 固化后随之删除。

## 审查发现

### 🔴 CRITICAL (严重)

无。

### 🟡 WARNING (警告)

无。

### 🔵 INFO (建议)

- 三处常量的"为什么这个值"已用注释固化在代码内（`syncGPVTaskExpiresIn`、`hintFloor`、`expandThreshold` 联合关系），便于未来反推。建议保留这种长注释风格，不要为了简洁缩短。
- 三个 pre-existing failures 已分别开 #550 / #551 / #552 跟进，建议按 priority 排进下一批 ship。
- repo 记忆 `bsc-bts0-shared-status.md` 已记录"Bts.0 固定挂载"事实，本次 PR 行为与该记忆一致；XML 注释也复述了同一约定，前后端 + 数据字典三处协同明确。

## 详细分析

### `omcgo/internal/provision/sync.go`

新增常量 `syncGPVTaskExpiresIn = 1800`（30 分钟），并在 `EnqueueGPVBatches` 中显式设置每个 `CreateTaskRequest.ExpiresIn`。

- 数值选择有充分注释：1800s = 5~10 次 inform 周期 × 1 task/s × 250 task 余量
- 取值与全局 `default_expires_in_seconds=120` 解耦，避免下次有人调全局时误伤 sync 路径
- 仅影响 Path B / 手动 sync 链路，对交互式单 RPC 场景无副作用

### `omcgo/internal/provision/sync_pathb_expand.go`

`hintFloor: 32 → 256`，配合 `expandThreshold=600KB` 的联合过滤：
- BSC `DeviceGSM.Bts.{i}.*`（~70 字段）：`60×70×256=1.05MB > 600KB` → 展开 256 instance
- 中小对象（<40 字段）：`60×30×256=460KB < 600KB` → 仍不展开
- 上限 `maxHintCap=512` 不变 → DB 异常脏数据不会膨胀到几千 task
- 注释将"曾用 32 / 为什么 256 / 副作用如何被过滤"链条全部固化

### `omcgo/data/param-mappings/BSC.xml` + `omcgo/data/quicksettings/BSC.xml`

`DeviceGSM.Bts.{i}.NumofTrxChannel`、`OmlConnectState` 两 leaf 的 `standardPath` 固化为 `Bts.0`：
- 与 osmo-bsc 实际行为一致（`Bts.{i}.X` 非 0 索引一律返回 SoapFault 9005）
- 两个 XML 都加了同一段注释（背景 + `applyInstanceContext` 对不含 `{i}` 的 path 原样返回的语义）
- 前后端 + 数据字典三处一致

### `omcmb/webcode/src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx`

删除 `resolveReadPath` 内 hardcode 的 `Bts.{i}.<leaf> → Bts.0.<leaf>` rewrite。
- XML 固化后该 workaround 变多余
- 行为等价：删除后 `applyInstanceContext` 对不含 `{i}` 的 path（即新的 `Bts.0.X`）原样返回，所有 BTS 行仍读到同一根 leaf
- 注释更新到与 XML 一致的解释

### `omcgo/internal/provision/sync_test.go`（新增）

`TestEnqueueGPVBatches_UsesSyncGPVExpiresIn`：
- 验证混合 3 批（1 标量 + 2 object 前缀）task 全部使用 `syncGPVTaskExpiresIn`
- 额外断言常量值固定为 1800，防未来被悄悄改小

### `omcgo/internal/provision/sync_pathb_expand_test.go`（调整）

- `SmallObjectsKeptAsIs`：固化 fields=5 + hintFloor=256 下"不展开"门槛（76800 < 600KB）
- `FirstTimeSyncUsesHintFloor`：改造为 BSC cold-start 真实场景（70 字段，断言展开数 = `hintFloor` 常量）
- 新增 `FirstTimeSyncMidSizedObjectNotExpanded`：fields=30 仍不展开，验证 hintFloor=256 不误伤中小对象
- `DBLookupErrorFallsBackToHintFloor`：换 BSC 真实 path，断言改用 `hintFloor` 常量而非裸数字

测试覆盖完备：常量回退、cold-start 大对象、中等对象、DB 失败兜底四象限齐全。

## 业务完整性检查

- Handler-Service-Repository 链路：N/A（无新增 handler/service/repository）
- 路由注册：N/A
- 迁移文件配套：N/A（无新表）
- 错误码注册：N/A（无新错误码）
- API 服务配套：N/A（无新 API）
- 种子数据：N/A
- E2E 测试用例：N/A（无新端点；BSC 同步触发链路已有 e2e 覆盖）

**结论**：业务链路完整，本次为既有链路的常量/数据字典调优 + 死代码删除。

## 业务影响范围检查

- 接口签名变更：无（`SyncService` 公共方法签名未动，仅添加常量 + 设置 `ExpiresIn`）
- 数据库 Schema 变更：无
- 事件契约变更：无
- 共享 model 变更：无
- 中间件变更：无
- 配置项变更：`syncGPVTaskExpiresIn` 是包内常量而非配置项，无需同步 yaml/部署文档
- 运营商适配器变更：本次调整在通用 `provision` 包，对 osmo-bsc 之外的设备影响评估：
  - `hintFloor=256` + `expandThreshold=600KB` 联合过滤保证小对象（<40 字段）不被误展开 → 对 enodeb/wifi 等典型小对象设备无副作用
  - 大对象（≥40 字段）确实会展开到 256 个 instance，但已配合 ACS `tryRecoverGPVFault` 容错路径
  - `syncGPVTaskExpiresIn=1800s` 对所有设备生效，但仅影响"已 expired 被 sweeper 清掉"的边界，慢设备受益，快设备无影响
- API 响应格式变更：无
- 跨模块引用：无新增 import

**结论**：变更范围可控。BSC 之外的影响经联合阈值过滤已收敛。

## 前后端一致性检查

- 接口路径一致：N/A
- 请求参数一致：N/A
- 响应字段一致：N/A
- 分页参数一致：N/A
- 错误码处理：N/A
- 枚举值一致：N/A
- 新接口双侧覆盖：N/A
- **数据字典一致性**（本次主要一致性面）：
  - `omcgo/data/param-mappings/BSC.xml` 与 `omcgo/data/quicksettings/BSC.xml` 中 `Bts.0` 路径写法完全一致
  - 前端 `CellParameterForm.tsx` 删除 hardcode rewrite 后，行为依赖 XML 固化的 `Bts.0`，已与数据字典同步
  - XML 注释 + 前端注释 + repo 记忆三处描述一致

**结论**：跨前后端 + 数据字典三处一致，无悬空契约。

## 代码质量回退检查

- 删除测试用例：无（仅调整测试数据让其更贴近 BSC 真实场景，且新增 1 个测试覆盖中等对象不展开 + 新增 1 个测试防 `ExpiresIn` 回退）
- 删除错误处理：无
- 降级安全措施：无（无 auth / 中间件改动）
- 引入 any/interface{}：无
- 硬编码替代配置：⚠️ 表面上 `syncGPVTaskExpiresIn = 1800` 是硬编码，但本质是**业务常量**（与 ACS 串行 push 速率 + inform 周期数学耦合），不属于运行时可调项；注释已固化推导链。同理 `hintFloor=256` 对应 BSC 物理上限 256 BTS。两者均非"应从配置读取"的可变值。
- 删除日志：无
- 简化校验逻辑：无
- ORM 替代 Squirrel：无
- 绕过接口抽象：无
- TODO/HACK 残留：无

**结论**：未发现代码质量回退。`syncGPVTaskExpiresIn` 与 `hintFloor` 形式上是常量但属于业务推导值，注释固化了推导过程，符合"魔法数字必须有理由"的原则。

## 配套更新提醒

- 文档：repo 记忆 `bsc-bts0-shared-status.md` 已记录 Bts.0 约定 → 本次 PR 行为一致，无需补充。建议在 PR merge 后追加"hintFloor=256 / ExpiresIn=1800"的 ADR 注记到 `docs/adr/`（INFO 级别，非阻塞）。
- 单元测试：✅ 已配套（`sync_test.go` 新增 + `sync_pathb_expand_test.go` 调整）
- E2E：本次无新端点，e2e 沿用既有 BSC sync 链路覆盖。验收手段为线上/测试环境观察 BSC 首次同步后 `device_parameters` 表覆盖全部 254 BTS 实例（runbook 级验证，PR body 已注明）。

## 总结

- 0 CRITICAL / 0 WARNING / 3 INFO
- 通过审查，建议直接进 P8 提交
