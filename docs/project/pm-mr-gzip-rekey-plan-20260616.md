# PM/MR 压缩文件 `.xml.gz` 自描述 + 下载不再乱码

> 跟踪单子（持久真相源）。创建于 2026-06-16。
> 载体说明：本应建 GitHub Issue（项目任务源），但当前机器无 `gh`/无 token，暂用本仓库文档跟踪；
> gh 就绪后再镜像成 Issue 并在此回填编号。
> **续做方式**：读本文「进度 Checklist」+「文件改动清单」即可恢复，无需依赖对话上下文。

---

## 1. 背景 / 问题

`#321` 引入 `internal/core/rawarchive` 压缩回写：PM/MR 原始 XML 入库成功后异步 gzip 压缩、**按原对象键覆盖回写** MinIO 省盘，对象元数据带 `Content-Encoding: gzip`，DB 标 `raw_compressed=true`。

副作用：对象**键仍是 `.xml`，内容却是 gzip 字节**——"文件名撒谎"。导致：
- **web 页面下载**（后端代理流式透传）拿到 gzip 字节当 `.xml` → **乱码**。
- **直接进 MinIO console 拉对象** 同样是 `.xml` 名 gzip 内容 → 乱码、不自描述。

（入库读取侧 `compress.MaybeGunzip` 透明解压，所以系统内部解析不受影响——只有"下载/直拉"暴露问题。）

## 2. 决策（已锁定）

| 项 | 决策 |
|----|------|
| 压缩格式 | **gzip**（不用 zip）。zip 会连锁改造入库解析链路 + 真机 `.xml.gz`(gzip) 双格式并存，代价大；gzip 与真机统一、内部链路零改动 |
| 存储层对象键 | 改为 **`.xml.gz`**（自描述，直拉 MinIO 也正确） |
| web 下载 | 也给 **`.xml.gz`**（不设 `Content-Encoding: gzip`，避免浏览器自动解压与 .gz 名矛盾） |
| 范围 | **PM + MR 一起**改（同构链路） |
| 存量数据 | **不处理**（老对象保持 `.xml`/gzip 现状，web 下载靠切片 1 的魔数嗅探兜底；直拉老对象仍乱码，接受） |
| 提交/部署 | 暂不 commit、不部署，等用户明确指令（遵循"不主动 commit"约定） |

## 3. 切片划分

- **切片 1（下载层）**：下载/打包时嗅探 gzip 魔数，是压缩内容就把文件名补 `.gz`。独立、低风险、立即解决 web 下载乱码。
- **切片 2（存储层改键）**：压缩成功时把对象键从 `.xml` 改成 `.xml.gz`，DB `minio_path` 同步更新。解决"直拉 MinIO 自描述"。触及已上线 #321 + 跨 MinIO/DB 一致性。

---

## 4. 切片 1（下载层）—— ✅ 代码完成、build+test 绿、未提交

机制：`bufio.Peek(2)` 嗅探 gzip 魔数 → 是 gzip 则下载名补 `.gz`、Content-Type 设 `application/gzip`、**不设 `Content-Encoding`**；明文原样透传。

| 文件 | 改动 |
|------|------|
| `internal/pm/handler.go` `DownloadPMFile` | 单文件下载补 `.gz`（import 加 `bufio`/`compress`） |
| `internal/mr/handler.go` `DownloadFile` | MR 单文件下载同构（import 加 `bufio`/`strings`/`compress`） |
| `internal/bundle/service.go` `appendOne` | 批量 zip 条目名嗅探补 `.gz`（一处覆盖 PM/MR 四个 source；对固件/配置/license 非 gzip 零影响；`obj.Read`→`br.Read`） |

验证：`go build ./internal/pm/... ./internal/mr/... ./internal/bundle/...` 通过；`go test -short ./internal/mr/... ./internal/pm/...` + `go test ./internal/bundle/...` 全绿（22 包无回归）。

**已知盲区**：下载 handler 持有 `*minio.Client`（具体类型，非接口），纯单测注入不了"成功返回 gzip 内容"分支 → "下载成功 + 拿到 .gz"需**真栈手验**。

---

## 5. 切片 2（存储层改键）—— ✅ 代码完成、`go build ./...` exit 0；⏳ 待跑测试 + 待对账审查

### 机制（三步顺序，保证不丢文件）
压缩成功时：
1. `archiver.compress` 把 gzip 字节 **Put 到新键 `object+".gz"`**（不覆盖旧明文键），返回新键；
2. `onTerminal(ctx, old, new)` 回调 → `MarkCompressed(map[old]new)`：逐行 `UPDATE minio_path=new, raw_compressed=true WHERE minio_path=old`；
3. DB 更新成功后 `archiver.RemoveOld` **删旧明文键**。

不变式：**DB 永远指向已存在的对象**（先写新键、再切 DB、最后删旧），任何中途崩溃可由 Sweeper 重扫自愈（旧键还在、`raw_compressed` 仍 false → 重压），最坏只留个会被覆盖的孤儿，不丢文件、不 404。

### 接口签名变更（3 处）
- `Schedule` 的 `onTerminal`：`func(ctx, object)` → `func(ctx, oldObject, newObject)`
- `CompressNow`：返回 `bool` → `(bool, newObject string)`
- `RawFileRegistry.MarkCompressed` / `PMFileStore.MarkCompressed` / `MRStore.MarkCompressed`：`[]string` → `map[string]string`(old→new)

### 文件改动清单（生产）
| 文件 | 改动 |
|------|------|
| `internal/core/rawarchive/archiver.go` | 加 `gzSuffix` 常量；`RawStore` 接口 +`Remove`；`compress` 返回新键 + Put 到新键；`Schedule`/`CompressNow` 签名；新增 `RemoveOld` |
| `internal/core/rawarchive/minio_store.go` | 实现 `Remove`（`RemoveObject`，not-found 幂等） |
| `internal/core/rawarchive/sweeper.go` | `RawFileRegistry.MarkCompressed` 签名→map；`compressBatch` 返回 map；`sweepSource` 在 Mark 后对 old≠new 调 `RemoveOld` |
| `internal/pm/file_store.go` | `PMFileStore.MarkCompressed` 接口签名→map |
| `internal/pm/pg_file_store.go` | 实现改键（`pgx.Batch` 逐行 UPDATE minio_path+raw_compressed） |
| `internal/pm/collector/collector.go` | `markRawCompressed(old,new)`；Mark 成功后 old≠new 调 `archiver.RemoveOld` |
| `internal/mr/store.go` | `MRStore.MarkCompressed` 接口签名→map |
| `internal/mr/pg_store.go` | 实现改键（同 PM） |
| `internal/mr/collector/collector.go` | `markRawCompressed(old,new)` + `RemoveOld` |

### 文件改动清单（测试）
| 文件 | 改动 |
|------|------|
| `internal/core/rawarchive/archiver_test.go` | `fakeStore` +`Remove`/`removed` 字段；改键后 Put 断言→`.gz`；`onTerminal`/`CompressNow` 签名+断言 |
| `internal/core/rawarchive/sweeper_test.go` | `fakeRegistry.MarkCompressed`→map；put 断言→`.gz` + 旧键被删断言 |
| `internal/pm/pg_file_store_test.go` | `mockPMFileStore.MarkCompressed`→map |
| `internal/mr/handler_test.go` | `mockMRStore.MarkCompressed`→map |

### 注意事项
- `file_name` 字段**不改**（与 `device_sn` 有联合 UNIQUE；前端列表展示用它）。改键只动 `minio_path`。改键后 `file_name` 仍 `.xml`，前端显示 `.xml`、MinIO 实际 `.xml.gz`、下载/打包靠魔数嗅探补 `.gz` —— 各下游自洽。
- `minio_path` 无 UNIQUE 约束，改键安全。

---

## 6. 验证计划

- [x] 切片 2：`go test ./internal/core/rawarchive/... ./internal/pm/... ./internal/mr/...` 全绿（2026-06-16）
- [x] ultracode 对账审查（多 agent，21 agents/2026-06-16）：15 候选 → 9 确认 / 6 证伪，见 §10；据此修复（#6 bucket 已修）
- [ ] 真栈手验：起栈下载一个已压缩 PM/MR 文件，确认拿到可正常解压的 `.xml.gz`
- [ ] （切片 1）真栈手验下载成功分支

## 7. 回滚

未提交，回滚=丢弃工作区改动（`git checkout -- <files>`）。已列全部改动文件，可精确还原。切片 1 与切片 2 相互独立，可单独回滚。

---

## 8. 进度 Checklist（续做看这里）

- [x] 切片 1 代码：pm/mr handler + bundle
- [x] 切片 1 验证：build + test 绿
- [x] 切片 2 代码：rawarchive(archiver/minio_store/sweeper) + PM/MR(store接口/pg实现/collector)
- [x] 切片 2 测试适配：4 个测试文件签名/断言
- [x] `go build ./...` exit 0
- [x] 切片 2 跑测试（rawarchive+pm+mr 全绿，2026-06-16）
- [x] ultracode 对账审查（9 确认/6 证伪，见 §10）
- [x] 修复 #6 bucket 不一致（build+test 绿）
- [x] 修复 #2 RemoveOld ctx 解耦（build+test 绿）
- [x] 范围决策：本切片只修 #2；B(测试盲区) + C(#3/#5/#4) 转 follow-up（见 §11）
- [x] 真栈手验①（切片2 改键，sweeper 路径，2026-06-16）：明文对象→`.xml.gz`(合法gzip 1f8b、解压还原原文) + DB `minio_path` 改键 + `raw_compressed=true` + 旧 `.xml` 键删除 + metric `compressed=1` 无 `remove_old_error`。真栈/真MinIO/真DB/真编译我的代码（含 #6/#2 修复）
- [ ] 真栈手验②（切片1 下载 HTTP：`.xml.gz` + 无 Content-Encoding）——待 admin 密码或你浏览器自验
- [ ] （待用户指令）commit 分切片
- [x] 部署 app + worker（`dc.sh up -d --build app worker`，HEALTH: OK，2026-06-16 18:08）—— 本地 omc 栈现跑改动代码；未 commit
- [ ] （gh 就绪后）镜像成 GitHub Issue，回填编号：`#____`

## 9. 当前状态一句话

切片 1+2 代码全改完、测试绿、ultracode 审查（9确认/6证伪）、#6+#2 已修、真栈手验①(改键)通过、**已部署 app+worker(本地 omc 栈，HEALTH OK)**；剩：②下载 HTTP 校验(待 admin 密码/浏览器) + commit(待用户指令)。**未 commit**。

---

## 10. ultracode 对账审查结果（2026-06-16，21 agents）

完整报告（含逐条 reasoning）：`tasks/w1nudnl8f.output`。15 候选 → **9 确认 / 6 证伪**。

**关键背景（来自一条被证伪的发现，重要）**：MinIO 桶有 ILM 生命周期（PM/MR 桶默认 60 天按对象年龄回收，`minio_ilm.go`）+ MR 另有 3 天 cleaner（`mr/task/cleaner.go`）。故"崩溃/超时残留的旧明文键"**不是永久孤儿**，会被自动回收 → 大幅降低 #2/#5 的危害。

### 9 条确认 + 分诊

| # | 维度 | 严重度 | 问题 | 分诊 |
|---|------|--------|------|------|
| 6 | path-parity | minor | MR 内联 `RemoveOld` 用 `c.bucket`，与 Schedule 写入的 `payload.Bucket` 可能不一致→删错桶 | **✅ 已修**（onTerminal 透传 bucket，correct-by-construction，PM/MR 都覆盖；build+test 绿） |
| 2 | crash-recovery | minor | `RemoveOld` 与压缩共用 30s ctx + 关停 cancel→偶发跳过删旧键留孤儿（ILM 60d 兜底回收） | **✅ 已修**（`RemoveOld` 用 `WithoutCancel`+10s 独立短超时，解耦压缩/关停 ctx；build+test 绿） |
| 1 | tests-edges | **major** | 下载层(切片1) gzip 嗅探+改名**成功分支三处全无单测**（minio.Client 具体类型测不到） | B 类·测试加固：抽小接口注入 fake，补 PM/MR/bundle 三处 table 测试（断言补 .gz / 无 Content-Encoding / 已 .gz 不重复） |
| 7 | tests-edges | minor | `minioStore.Remove` not_found 幂等分支无测试 | B 类·测试加固（小） |
| 8 | tests-edges | minor | `markRawCompressed` 改键回调（old≠new→删、标记失败→不删）无单测 | B 类·测试加固（小，护住改键不变式） |
| 9 | tests-edges | nit | `gzSuffix` HasSuffix 防御分支 + MarkCompressed 空 map 无断言 | B 类·测试加固（nit，几行） |
| 3 | minio-db-consistency | minor | NATS 重投旧键→`GetObject` 404→DLQ/反复 NAK（数据已入库，无丢失；MR 其实未接 DLQ 只 NAK） | C 类·建议另立 follow-up：下载/解析对 NoSuchKey 做幂等 ack（触发窄、可恢复） |
| 5 | minio-db-consistency | minor | 并发下载流 vs `RemoveOld` TOCTOU 窗口（极窄、可重试、bundle 已有 .error.txt 占位） | C 类·建议另立 follow-up：下载对 NoSuchKey 按新键重试，或延迟删旧键到 sweeper grace |
| 4 | minio-db-consistency | minor | `file_name`(.xml) 与 `minio_path`(.xml.gz) 长期不一致，下载靠魔数嗅探兜底，contract 脆弱 | C 类·需决策：同步改 file_name 加 .gz（影响前端展示+唯一约束）或给 DB 投影加 raw_compressed 字段 |

**6 条证伪**（均为过度定性/前提不成立，reasoning 充分）：永久孤儿无 GC（实有 ILM）、PM 撞键数据丢失（文件名内嵌 SN+改键前后等价）、pgx.Batch 部分失败（pgx 隐式事务全或无+自愈）、pgx.Batch 测试缺失判 major（实为 nit）、bundle .gz 注释口径、前端 RFC5987。

### 分诊汇总
- **A 类（本切片现修）**：#2 ✅ 已修 + #6 ✅ 已修
- **B 类（测试加固，护回归）→ 转 follow-up §11**：#1(major) #7 #8 #9
- **C 类（另立 follow-up §11）**：#3 #5（NoSuchKey 家族，触发窄/可恢复/ILM 兜底）、#4（需产品决策）

---

## 11. Follow-up（另立单，本切片不做；用户决策 2026-06-16：本切片只修 #2）

> 这些都是 ultracode 审查确认的真实项，但触发窄/无数据丢失/属测试或产品决策，刻意推迟。记录在此防丢。

**B 类 — 测试盲区（护回归，建议优先做）**
- **#1（major）下载层成功分支零单测**：PM/MR `DownloadFile` + bundle `appendOne` 的 gzip 嗅探+改名成功分支无测试（minio.Client 是具体类型）。修法：抽最小 `objectGetter` 接口注入 fake，补三处 table 测试，断言「压缩→补 .gz / Content-Type application/gzip / **响应头无 Content-Encoding** / 已 .gz 不重复 / 明文不改名」。
- **#7** `minioStore.Remove` not_found 幂等分支补单测（仿 `backup/policy_monitor_test.go` TestClassifyMinIORemoveErr）。
- **#8** `markRawCompressed` 改键回调补单测：new==old 不删、MarkCompressed 失败不删（护改键不变式）。
- **#9（nit）** `gzSuffix` HasSuffix 防御分支 + MarkCompressed 空 map 补断言。

**C 类 — 运营硬化 / 产品决策**
- **#3（minor）NATS 重投旧键 404**：重投撞已删旧键→download/parse 报错→PM 灌 DLQ / MR NAK 到上限。修法：collector 对 NoSuchKey 做幂等 ack（查 DB 已入库则 return nil），别当瞬时错误重试。
- **#5（minor）并发下载 TOCTOU**：下载流读到一半旧键被 RemoveOld 删→可能 404。修法：下载对 NoSuchKey 按当前 minio_path 重试一次，或延迟删旧键到 sweeper grace。
- **#4（minor，需产品决策）file_name 与 minio_path 不一致**：下载靠魔数嗅探兜底，contract 脆弱。两选：(a) MarkCompressed 同步把 file_name 也改 .xml.gz（影响前端列表展示 + (device_sn,file_name) 唯一约束，需确认）；(b) 给 PMFileInfo/MRFileInfo 加 `raw_compressed` 投影字段，下载据此判定。
