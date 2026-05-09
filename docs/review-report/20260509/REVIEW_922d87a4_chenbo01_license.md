# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-09 |
| 提交 | 922d87a4 |
| 作者 | chenbo01@baicells.com |
| 范围 | license |
| 变更文件数 | 12 |
| 新增行数 | +1358 |
| 删除行数 | -49 |

## 变更概要

T-0100-P4-B + T-0100-P4-C 双子任务合并提交：(1) **P4-C 签名强校验** 替换 P3 stub
为完整的 OEM 公钥加载 + RSA-PSS 验签链路，引入 `SignatureVerifier`、
`canonicalizeLicensePayload` 和 `ImportRequest.SignedLicenseJSON`；(2) **P4-B 6 月归档 cron**
新增 `LogArchiver`，把 license_logs 表里 ≥ 6 个月的行打包成 `license-logs/{YYYY-MM}.jsonl.gz`
写入 MinIO 后从 DB 删除，weekly schedule 注册到 Monitor。两子任务均仅后端 +
配置变更，无前端 / 数据库 schema 变更。

## 审查发现

### 🔴 CRITICAL (严重)

**1. 归档器同月跨 tick 覆盖导致数据丢失** —— `omcgo/internal/license/archiver.go:107-167`

每周 cron `0 3 * * 0` 跑一次，cutoff = `now.AddDate(0,-retentionMonths,0)`。
连续两个 tick 之间 cutoff 漂移 7 天，但 PutObject 用的对象键固定为
`license-logs/{YYYY-MM}.jsonl.gz`（按月）。当 cutoff 落在月中间时，会发生：

| Tick | cutoff | 当 tick 处理的"早于 cutoff"日志范围 | 写入的 MinIO 键 | 结果 |
|------|--------|-----------------------------------|----------------|------|
| 第 N 周（2026-05-10） | 2025-11-10 | 2025-11-01 ~ 2025-11-09 | `license-logs/2025-11.jsonl.gz` | 写入 9 天日志 |
| 第 N+1 周（2026-05-17） | 2025-11-17 | 2025-11-10 ~ 2025-11-16 | `license-logs/2025-11.jsonl.gz` | **覆盖**前一 tick 的 9 天日志 |

第 N tick 结束时 `DeleteBefore(2025-11-10)` 已把 11-01~11-09 日志从 DB 删除，
所以 N+1 tick 读不到这些行；MinIO 上 2025-11.jsonl.gz 又被原子覆盖。这会导致
**整月日志只保留最后一个 tick 的子区间**，违反 PRD §5.4.5「6 个月以上自动归档」
合规要求（等保 2.0 三级 8.1.4.7「重要操作日志保留 ≥ 6 个月」）。

**复现门槛**：周级 cron + 月内任意一天 + 该月起始有 license_logs 行即可触发。

**修复建议（任选一）**：

- (a) **唯一键** —— `license-logs/{YYYY-MM}/{tickTimestamp}.jsonl.gz`，每 tick 一个对象，
  下游审计工具按月前缀检索（最简单，零数据丢失风险）；
- (b) **append 真实现** —— PutObject 前先 GetObject 已存在对象，gunzip → append 新行 →
  gzip → PutObject 覆盖（archiveObjectIO 接口本来已经声明 StatObject + 加 GetObject 即可，
  但要处理对象不存在 / decompress 失败 / 大对象 OOM 等边界）；
- (c) **cutoff 月对齐** —— cron 改为「每月第一个周日」（`0 3 1-7 * 0`）+ cutoff 强制对齐
  到月末（`time.Date(y, m+1, 1, ...)`），保证一个月只在一次 tick 处理。

文件头部块注释（line 12-13）已经描述了 (b) 的 append 思路并写「简化处理：当前实现
直接覆盖（cron 同月只跑一次，理论上不会同月二次归档）」—— 这个推断**与周级 cron + 月
对齐 cutoff 缺失的事实矛盾**，docs/code 不一致独立标记为 WARNING（见下文 WARNING 项）。

**测试盲区**：现有 `archiver_test.go` 6 个用例均一次性塞入并归档完，未覆盖跨 tick 重复
归档同月场景；建议增加一个用例：
```go
// Tick 1: cutoff=2025-11-10 → 写 11-01..11-09
// Tick 2: cutoff=2025-11-17 → 不应覆盖 Tick 1 的归档对象
```

---

### 🟡 WARNING (警告)

**1. 归档器 doc/code 不一致** —— `archiver.go:8-15`

文件头部块注释列出步骤 4-a「StatObject 看 MinIO 上是否已有」和 4-b「append 模式」，但
实现路径里 `StatObject` 从未被调用，archiveObjectIO 接口签名声明了 StatObject 仅是
为后续可能的 append 实现预埋。**接口残留死方法 + 注释描述未实现行为**会让后续维护者
（包括审查 / 二次实现的人）误以为已有 append 逻辑。

**修复建议**：

- 选择 CRITICAL 项中的修复方案落地后，相应更新 doc 块注释；
- 或在不实现 append 时，从 archiveObjectIO 接口删除 StatObject（让接口名实义），
  并把 doc 改为「单 tick 内每月只 PutObject 一次（覆盖语义）；同月跨 tick 由 cutoff
  对齐保证不重复处理」（配合 CRITICAL 修复）。

---

**2. 业务错误码 9109 未在 `global/errors.go` 注册** —— `handler.go:687-689`

新增 `commonerrors.NewBusinessError(9109, ...)` 用于 strict 签名拒绝；但 license 模块
在 `global/errors.go:135` 注册的官方错误码区段是 **12000-12999**：

```
// License (12000-12999)
ErrCodeLicenseNotFound       = 12001
ErrCodeLicenseDuplicate      = 12002
...
```

9100-9108 是 P0/P1/P2/P3/P4-A 期间已经存在的"野"码（pre-existing tech debt），P4-C 的
9109 沿袭了同模式；CLAUDE.md 项目规范明确"9000-9999: 北向/OSS 接口"，9100~ 段不是
license 域的。

**修复建议**（不阻塞 P4-C，可作为 follow-up tech-debt）：
- 把 license handler 已用的 9100-9109 全部迁移到 12100-12109（保留 12001-12005 给基础
  错误用），同步 i18n 错误文案；
- 或在 `global/errors.go` 新增 `// License HTTP-internal (9100-9109)` 段并补全常量，
  让野码至少有名字。

---

**3. `pg_license_log_repository.go:212` ListBefore 默认 limit=10000 可能与 archiver
batchSize=10000 形成 N+1 隐患**

`LogArchiver.batchSize = 10000`，`ListBefore` 不传 limit 时默认也是 10000；这本身
没问题。隐患在于：

- 单次 tick 只能归档前 10000 条；超出部分留到下个 tick；
- 但 `DeleteBefore(cutoff)` **删除所有 `< cutoff` 的行（不限 10000）**；
- 也就是说，如果某 tick 有 25000 条 ≤ cutoff 的日志：
  - ListBefore 取出最早的 10000 条（按 created_at ASC）；
  - PutObject 写这 10000 条对应的月份归档；
  - **DeleteBefore 删掉全部 25000 条** —— 后 15000 条**没归档就被删了**。

复现门槛：单次 tick 积压 > batchSize（10000 条），需要前几次 tick 失败 / 系统停机
导致积压。在小规模部署不易触发，但在大压力 / 长时间无 cron 后第一次启动时可能踩雷。

**修复建议**：
- 把 `DeleteBefore(cutoff)` 改成 `DeleteByIDs(archivedIDs)`，只删本 tick 实际归档的
  行（接口加 `DeleteByIDs(ctx, ids []uuid.UUID)` 方法）；
- 或在 ArchiveOnce 内确认 `len(logs) < batchSize` 才执行 DeleteBefore，否则只删
  `<= logs[len-1].CreatedAt` 的范围（精度损失但不丢数据）。

---

**4. SignatureVerifier.LoadKeysFromDir 解析失败聚合后仍返 error 但 modules.go DI 仅
warn 不阻断启动** —— `modules.go:432-435`

DI 对 LoadKeysFromDir 的错误降级为 zap.Warn 不阻断进程启动：

```go
if loadErr := licenseVerifier.LoadKeysFromDir(dir); loadErr != nil {
    logger.Warn("license OEM public key load reported errors (non-fatal)", ...)
}
```

设计意图是 OK 的（部分 .pem 损坏不应让整个 app 起不来）。但 strict=true 部署下，
**所有 .pem 都解析失败 + verifier.KeyCount() == 0** → 后续每次 import 都会被 strict
拒绝，运维只能通过翻 zap 日志才能定位问题。

**修复建议**：

- 当 strict=true 且 KeyCount() == 0 时，启动时直接 `Fatal`（与 LoginCryptoConfig 自动
  生成密钥不同，OEM 公钥错失意味着 license 流程瘫痪，宜 fail-fast 让运维立即排查）；
- 或在 DI 后立即 RecordHealthCheck("license-signing", err) 让 /healthz 反映；
- 当前 prod 部署文档（应在 ops runbook 里）需要明确这个降级点。

---

**5. 归档器对 ctx cancellation 不敏感** —— `archiver.go:130-148`

`for month, monthLogs := range groups { ... }` 循环里没有 `select { case <-ctx.Done(): }`
检查；如果月数很多 + MinIO 慢，30 分钟超时到达后 ctx 已 cancel 但循环仍会继续 PutObject
（PutObject 内部应该会感知 ctx，但理论上 worst-case 仍可能写半个月就被 ctx kill），
导致部分月归档成功部分失败 → return error → DeleteBefore 不执行 → 下次 tick 重试 OK。

实际危害有限（幂等设计兜住了），但代码里显式的 `if ctx.Err() != nil { break }` 会让
失败更早可见 + 测试更易写。

**修复建议**：循环开头加 `if err := ctx.Err(); err != nil { return nil, err }`。

---

### 🔵 INFO (建议)

**1. signature.go canonicalize 未递归** —— `signature.go:370-405`

注释里写「嵌套对象不递归排序（约定：license JSON 顶层已扁平，仅 features 是数组）」；
这个约定写得清楚。但只要将来 license 结构演化加嵌套字段（例如 `device_type_quota`
子对象，PRD §16 已经预埋这个 future ext），canonicalize 就会因为嵌套 map key 顺序
不稳定（Go map iteration 不保证序）而产生**同一 license JSON 的两次 canonical 不一致**，
进而 verify 通过率随机失败。

**建议**：现在就添加一个 doc-level 校验：在 `VerifyLicenseJSON` 里 explicit
扫描 `doc[k]` 的嵌套结构，发现 nested object 时 log warn / fail；或者升级 canonicalize
为递归实现。前者零成本预警，后者一劳永逸。

---

**2. handler.go 引入 P3 stub `VerifySignature(nil)` 作为 fallback** —— `handler.go:686-688`

```go
} else {
    // 退化：保持 P3 stub 行为（让没接 verifier 的部署仍可导入）。
    sigStatus, sigNote = VerifySignature(nil)
}
```

P3 stub 函数（signature.go:304）现在变成"verifier nil 或 SignedLicenseJSON 空"两种情况
的 sentinel returns；功能上 OK，但代码里**两个独立路径返回 SignatureUnverified +
不同 note**让审计追溯起来稍麻烦。

**建议**（可选）：handler.Import 不再调 P3 stub `VerifySignature`，直接 inline
`sigStatus, sigNote = SignatureUnverified, "verifier or signed payload not provided"`，
然后把 stub 函数标记为 `// Deprecated: kept for tests; production path uses
SignatureVerifier.VerifyLicenseJSON.`

---

**3. archiver.go encodeLogsJSONLGz 未限制单对象大小** —— `archiver.go:184-198`

如果某个月有 100 万条日志（极端值，但理论可能），单 PutObject 会上传一个非常大的
gzip 对象（甚至触达 MinIO 默认单对象限制）。MinIO 单对象上限默认 5GB（multipart
上传），实际单月日志通常 < 100MB 不会撞顶；但仍是个未来 scale-out 需要观察的点。

**建议**：metrics 监控 `omc_license_archive_bytes_per_month` + 在 ArchiveResult 新增
`PerMonthSize map[string]int64` 字段方便运维诊断。本期可以延后。

---

**4. `Monitor.SetArchiver` 在 `Start()` 之后调用会无效**

DI 里 SetArchiver 在 modules.go 的 `c.miscDeps.licenseMonitor.Start(...)` 之前调用
（modules.go:438 `Start` 早于 P4-B 注入位置），所以**当前生效**。但接口设计上没有
guard：如果以后有人 reorder DI 顺序，archive cron 就静默不注册。

**建议**：在 `Start` 里加 `if m.archiver != nil` 之外的 sanity log；或者把 SetArchiver
设计成「未启动时才允许」（cron != nil 时 panic / return error）。

---

**5. 测试 `TestSignatureVerifier_LoadKeysFromDir_HappyPath` 用 0o600 写 PEM 文件**

测试用 `os.WriteFile(path, pemBytes, 0o600)` 写 PEM；mode 0o600 对真实公钥目录是 OK，
但**测试里其实可以用 0o644**（不影响测试逻辑），保留 0o600 让人误以为这是 prod
强制权限模型。

**建议**（选）：测试用 0o644 即可；prod 部署文档里再额外强调 `configs/oem_public_keys/*.pem`
的权限要求（敏感度只有公钥，0o644 即满足合规）。

---

## 详细分析

### `omcgo/internal/license/signature.go`（+406 行）

整页重写，从 P3 33 行 stub 变成 405 行完整 verifier。结构清晰：
- `SignatureVerifier{ mu, keys, strict }` 线程安全 + RW lock 分离 read/write 路径；
- `LoadKeysFromDir` 子文件失败聚合 + 整体不中断（合理）；
- `VerifyLicenseJSON` 5 个 fast-fail 早返（nil verifier / 0 keys / empty raw / parse / no signature field）+ 2 个匹配路径（key_id 精确 / try-all 兜底）；
- `canonicalizeLicensePayload` 顶层字典序排序 + delete signature/key_id —— 简洁正确；
- `parsePublicKeysPEM` 双格式（PKIX + PKCS1）兼容；
- `fingerprintRSAPublicKey` SHA-256 hex，与 OpenSSL CLI 一致 —— OEM 工具链可复核。

**Strict 模式半连续语义**：未签名（unverified）和签名错（invalid）在 strict 下都返
error，这是审慎的（防止 strict 模式被攻击者用未签名 license 绕过）；非 strict 下 unverified
和 invalid 用 status 字段区分 —— 设计一致。

**已有测试**：13 个用例覆盖 verified / tampered / strict 拒绝 / missing-sig (含 strict)
/ no-keys (含 strict) / wrong-key try-all / LoadDir (含 .txt 忽略 / nonexistent /
empty) / nil-verifier-stub / canonical-order-invariant 全部 race PASS。

---

### `omcgo/internal/license/archiver.go`（+198 行 / new）

- 接口窄定义 `archiveObjectIO`（PutObject + StatObject）—— consumer-side 模式正确，
  但 StatObject 是死方法（见 WARNING #1）；
- `ArchiveOnce` 「先持久化后删」语义清晰，幂等性靠 PutObject 覆盖兜住 —— **但同月
  跨 tick 覆盖丢数据**（CRITICAL #1）；
- `groupLogsByMonth` 用 `time.UTC().Format("2006-01")` 作为 key —— 注意字符 `time.Time.Format`
  的伪造 reference value（`2006-01-02 15:04:05`），代码正确；
- `encodeLogsJSONLGz` 用 `json.NewEncoder(gw).Encode(&l)` 实现流式压缩 —— 正确；defer
  路径的 gzip writer Close 错误处理也对；
- 唯一风险点：批 10000 条 + DeleteBefore 不限 → 大积压数据丢失（WARNING #3）。

---

### `omcgo/internal/license/handler.go`（+27 / -16 行）

- `Handler.sigVerifier` + `SetSignatureVerifier` 注入点设计与 logRepo / logWriter 同
  pattern —— 一致性 OK；
- `ImportRequest.SignedLicenseJSON` 用 `omitempty` JSON tag 保持向后兼容（旧客户端不传
  仍走 stub 路径）—— 兼容性 OK；
- `Import` handler 内 sigVerifier nil-check + req.SignedLicenseJSON empty-check 双保险，
  fallback 到 P3 stub —— dev 友好，但 stub fallback 可以 inline（INFO #2）；
- strict 拒绝路径：写一条 `LogTypeImport`/`LogResultFailed` 审计 + AbortWithError 9109
  —— 正确，但 9109 不在官方错误码段（WARNING #2）。

---

### `omcgo/internal/license/monitor.go`（+19 / -3 行）

- `Monitor` 加两个字段 `archiver` + `archiveSchedule` —— 字段对齐打破了原有视觉对齐
  （`archiver         *LogArchiver` 比上面多一空格），影响可读性，gofmt 不报错但建议
  两边对齐；
- `SetArchiver(a, schedule)` 默认 schedule=`0 3 * * 0`（周日 03:00 UTC）—— 与 PRD
  §5.4.5「DB 清理 cron 周级跑一次」一致；
- `Start()` 里 archive cron 注册带 30min 超时 —— generous 上限合理；
- 启动 log 改为动态 fields slice 把 archive_schedule 拼入 —— 干净。

---

### `omcgo/internal/license/pg_license_log_repository.go`（+47 行）

- `ListBefore`：squirrel `Where(sq.Lt{...})` + `OrderBy("created_at ASC")` + `Limit(uint64(limit))`
  —— 参数化查询，安全；
- `DeleteBefore`：squirrel `Delete(...).Where(sq.Lt{...})` + 返回 RowsAffected —— 安全；
- 接口契约同步：mocks (memLogRepo / failingLogRepo) 都补全了 ListBefore / DeleteBefore
  —— OK。

---

### `omcgo/cmd/app/provider/modules.go`（+34 行）

- 两段 DI 块清晰隔离 P4-C / P4-B；
- bucket fallback 链：`License.LogArchive.MinIOBucket → MinIO.Buckets.Logs` —— 合理；
- 三态禁用：`RetentionMonths > 0 && bucket != "" && minio != nil` —— dev 友好；
- 但 strict + 0 keys 的运维不可见性见 WARNING #4。

---

### `omcgo/internal/core/appconfig/config.go`（+51 行）

- `LicenseConfig` 带 `Signing` + `LogArchive` 子结构，mapstructure tags 完整 ——
  与同文件其他 module config 风格一致；
- 注释说明默认值（PublicKeyDir=空 / Strict=false / RetentionMonths=0 / Schedule=空）
  和 prod 推荐 —— OK。

---

### `omcgo/internal/license/{signature,archiver}_test.go`（+250 行 / new）

- `signature_test.go` 13 用例包含 happy / tampered / strict / missing / no-keys /
  wrong-key / canonical-order-invariant / nil-verifier-stub —— 覆盖良好；
- `archiver_test.go` 6 用例包含 NoMinIO 短路 / EmptyDB / 单月 / 三月 / PutObject 失败
  / RetentionDefault；
- **测试盲区**：缺 (a) 跨 tick 同月覆盖（CRITICAL #1）；(b) 单 tick 积压超 batchSize
  导致 DeleteBefore 误删未归档（WARNING #3）；(c) ListBefore 失败 / DeleteBefore 失败
  路径；(d) gzip Close 错误路径。

---

### `omcgo/internal/license/{log_writer,service}_test.go`（+44 行）

- memLogRepo + failingLogRepo 同步加 ListBefore / DeleteBefore —— 接口契约维护。

---

### `docs/project/backlog.md`（+3 行）

- T-0100-P4 拆 P4-A done / P4-B done / P4-C done —— 状态正确；
- §6 Done 表加 P4-C / P4-B 两条详细 closing evidence —— 信息完整。

## 业务完整性检查

- ✅ Handler-Service-Repository 链路：`ImportRequest.SignedLicenseJSON` → `handler.Import`
  → `SignatureVerifier.VerifyLicenseJSON` 已贯通；归档侧 `Monitor.archiver.ArchiveOnce`
  → `LogArchiver` → `LicenseLogRepository.{ListBefore,DeleteBefore}` + `archiveObjectIO.PutObject`
  全链路完整。
- ✅ 路由注册：本次未新增端点（仅扩展 Import 字段），无需 `RegisterRoutes` 改动。
- ✅ 迁移文件：本次零 schema 变更（archiver 仅 SELECT/DELETE 既有 license_logs 表），
  无需 migrations。
- ⚠️ 错误码：9109 未在 `global/errors.go` 注册（详见 WARNING #2）。
- ➖ 前端配套：本次纯后端 + 配置变更；P4-C 的 `signed_license_json` 字段前端尚未
  消费（前端 Tab 1 Import 表单还是手填，没接文件上传 → 解析 → 提取 signature 链路）。
  PRD §5.3.1 明确要求"上传文件 → 后端解析 + 验证签名 + 入库"，但当前前端只在 Tab 1
  顶部放了一个 Dragger UI 而没真正读文件内容传给后端。**这是 P4-C 的功能闭环缺口**，
  应作为 P4-C 的扫尾子项（前端读取文件 → base64 → POST.body.signed_license_json）补上。
- ✅ Mock 配套：licenseService mock 不变（不影响）。
- ➖ 种子数据：归档场景在 e2e 不易模拟（要伪造 6 月前的 created_at），可作为后续 tech
  debt 不强制。
- ➖ E2E 测试用例：`scripts/e2e_verify.sh` 没补 license export / signature import
  cases；P4 整体 GA 前应补。

## 业务影响范围检查

- ✅ 接口签名变更：仅添加可选字段 `SignedLicenseJSON`、新方法 `LoadKeysFromDir` /
  `AddKey` / `KeyCount` / `VerifyLicenseJSON` / `ListBefore` / `DeleteBefore` /
  `SetArchiver` / `SetSignatureVerifier`；既有签名零变更。
- ✅ 数据库 Schema：零变更。
- ✅ 事件契约：`LogArchiver` 不发 event，仅 zap Info/Warn 日志；`SignatureVerifier`
  也不发事件。
- ✅ 共享 model：没改 `internal/core/model/`。
- ✅ 中间件：未触碰。
- ⚠️ 配置项变更：新增 `appconfig.LicenseConfig`（嵌套两级），需要同步部署侧 `config.dev.yaml`
  / `config.prod.yaml` 加 `license:` section。本次提交**未同步更新这些 yaml 文件**
  （检查 `omcgo/cmd/app/etc/`），prod 部署如果想启用 P4-C strict 或 P4-B 归档，运维
  需要手动加配置。建议下次 commit 把 `config.prod.yaml` 加默认 license section（
  `signing.strict=true` + `log_archive.retention_months=6` + 注释说明）。
- ✅ 运营商适配：P4-B/P4-C 都不与 carrier 交互，纯运维侧能力。
- ➖ API 响应格式：handler.Import 响应 `ImportResponse` 字段未变（已经在 P3 加了
  signature_status / signature_note），P4-C 只让这两个字段开始反映真实状态。前端
  `licenseApi.importLicense` 已经处理 `signature_status` `'verified'|'unverified'|'invalid'`
  三态，无需变更。
- ✅ 跨模块引用：archiver.go 引入 `compress/gzip` + `github.com/minio/minio-go/v7`
  —— gzip 是 stdlib，minio-go 项目已用；signature.go 全部 stdlib。零新增第三方依赖。

## 前后端一致性检查

本次纯后端变更。已确认：
- 后端 `ImportResponse{ License, SignatureStatus, SignatureNote }` 与前端
  `licenseApi.ImportLicenseResult` 字段完全一致（P3 已对齐，P4-C 不改）。
- **前后端不一致点**：后端 P4-C 已经支持 strict 拒绝路径（HTTP 400 + 9109），但前端
  `LicenseOperations` Tab 1 Import 没消费这个路径 —— 当 strict 拒绝时前端显示的是
  通用 error message 而不是清晰的"签名校验失败：xxx"业务文案。建议前端：
  - 在 `extractErrorMessage` 之外新增 `parseImportRejection`，识别 9109 + 渲染"OEM
    签名验证失败"专用 Modal；
  - 或在导入失败 Modal 直接展示 `signature_status` + `signature_note` 字段（如果
    后端在 400 响应里也带这两个字段的话；当前 9109 path 没有，要加）。

**改后端但未改前端的契约缺口** = P4-C 后端就绪但前端未消费 = 应作为 P4-C 扫尾。

## 代码质量回退检查

未发现质量回退。逐项核对：
- ❌ 删除测试用例：未发现，反而新增 19 个（13 signature + 6 archiver）；
- ❌ 删除错误处理：未发现，新代码全部 `fmt.Errorf("...: %w", err)` 包装；
- ❌ 降级安全措施：反向加强（P3 stub → P4-C 真实 RSA-PSS 验签 + strict 模式）；
- ❌ 引入 any/interface{}：未发现；强类型保持；
- ❌ 硬编码替代配置：反向（PublicKeyDir/Strict/RetentionMonths/MinIOBucket/Schedule
  全部从 config 读）；
- ❌ 删除日志：反向新增 6 处 zap.Info/Warn；
- ❌ 简化校验：反向加强（strict 模式下三态拒绝）；
- ❌ ORM 替代 Squirrel：未发现，ListBefore / DeleteBefore 用 squirrel；
- ❌ 绕过接口抽象：反向新增 `SignatureVerifier`、`LogArchiver`、`archiveObjectIO` 三个
  抽象；
- ❌ TODO/HACK 残留：grep 0 命中。

**结论**：未发现代码质量回退。

## 配套更新提醒

- **文档**:
  - 建议更新 `omcgo/CLAUDE.md` §5（开发规范）补 license 模块的「签名格式约定 +
    OEM 公钥准备步骤」section（需要让运维知道怎么生成 .pem + 怎么签 license JSON）；
  - 建议更新 `docs/project/prd/F06-license.md` §5.3.1 / §5.4.5 把"待 P4 实现"措辞
    改为"P4-C/P4-B 已实现"，并附实现链接（archiver.go / signature.go）；
  - 建议 `docs/operations/`（如果存在）添加 OEM 公钥滚动 / 归档对象保留期 runbook。
- **单元测试**:
  - 必加：跨 tick 同月覆盖测试（CRITICAL #1 mitigations）；
  - 必加：积压超 batchSize 测试（WARNING #3）；
  - 选加：ctx cancel 中途生效测试（WARNING #5）；
  - 选加：嵌套字段 canonicalize 行为测试（INFO #1）。
- **端到端测试**:
  - `scripts/e2e_verify.sh` 当前 226 断言未覆盖 license 导出 / 签名导入；建议在
    GA 前补：(a) 用 stub OEM key 签一个 fixture license + POST /licenses/import
    带 signed_license_json + 验证 signature_status='verified'；(b) 全量 CSV 导出
    返 200 + Content-Type=text/csv；(c) 单条 PDF 导出返 200 + body 以 `%PDF-` 开头。

## 安全检查

- ✅ 签名 strict 模式提供反伪造能力；
- ✅ 公钥 fingerprint hex 与 OpenSSL CLI 输出一致，便于跨工具复核；
- ✅ 验证失败 try-all 兜底带 downgrade 防护：声明 key_id 命中但 verify 失败 → 直接
  invalid，不 fallback 到 try-all（防止攻击者通过伪造 key_id 绕过严格匹配，让 verifier
  退回到弱 key try-all 路径）；这个语义在 signature.go:269-282 实现得很谨慎，赞。
- ✅ 归档对象写到 MinIO `logs` bucket，复用既有访问控制；归档对象本身不含敏感字段
  扩散（license_logs 列已过 IP / UA 等是合规审计本就需要保留的）。
- ⚠️ 数据丢失风险（CRITICAL #1）属于完整性而非机密性问题，但合规层面同样严重。

## 性能检查

- ✅ ListBefore/DeleteBefore 用 squirrel 参数化 + WHERE created_at < ?，依赖既存
  `idx_license_logs_created_at` 索引（migration 000073 已建）；
- ✅ encodeLogsJSONLGz 流式 gzip，不会把整月日志先读全到内存（除了 buf.Bytes 最后
  一次性返）；不过 buf 仍然 in-memory，大月份可能 OOM —— 量级风险见 INFO #3；
- ✅ canonicalizeLicensePayload 顶层 sort + linear marshal，license JSON 字段数 ~20
  以内，O(n log n) 不是瓶颈；
- ⚠️ archiver tick 30 分钟超时偏长，假如 MinIO 阻塞，会让下次 tick 排队 —— 实际
  cron 不会并行（cron lib 默认串行同 entry），但 tick 内部的 PutObject 失败重试可能
  把整个 tick 拉长。建议把 PutObject 自身 ctx 加更短超时（5 分钟级别），让单月失败
  能更快感知。

## 测试覆盖

- 新增 19 个测试用例（13 signature + 6 archiver）+ 修补 mocks 实现 ListBefore / DeleteBefore；
- race PASS，2.969s 总耗时合理；
- **关键盲区**（已在 CRITICAL #1 / WARNING #3 列出）：跨 tick 同月覆盖、积压超
  batchSize 时 DeleteBefore 误删；这两个补完后覆盖率才算真正可发；
- 全量 `go test ./...` 仅 internal/task 2 个 pre-existing NULL source_id scan 失败，
  与 license 无关（git log 已验证）。

## 总结

| 级别 | 数量 |
|------|------|
| CRITICAL | 1 |
| WARNING | 5 |
| INFO | 5 |

**审查结论**: `NEEDS_FIX`

CRITICAL 问题（同月跨 tick 覆盖丢数据）必须修复后才能合入 GA 路径；P4-B 当前状态的归档
在小积压 / 月对齐场景下能用，但合规风险使其不能简单标 done。建议在 backlog 加一个
`T-0100-P4-B1: 归档对象键唯一化（防同月跨 tick 覆盖）` 子任务并立刻拉起，本期 P4-B
状态调整为 partially-done 等修复后再 done。

WARNING 项里的 #2（错误码注册）可作为 follow-up tech debt；#3（DeleteBefore 误删）
建议同 P4-B1 一起修；#4（strict 0 keys 静默）+ #1（doc/code 不一致）+ #5（ctx 不敏感）
是单独的 polish。
