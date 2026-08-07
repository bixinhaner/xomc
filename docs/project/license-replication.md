# OMC License 旧项目功能复刻留档

## 1. 范围与结论

OMC Go 的 License 目标是复刻旧 OMC 项目的 License 功能和行为，唯一 License 文件格式是旧项目二进制 `.lic`。

本实现不定义新的 JSON License 格式。HTTP 请求中的 JSON 只用于承载 `.lic` 原文的 Base64 字符串，不代表 License 文件格式。

复刻范围包括：

- TrueLicense `.lic` 解密、解压、XML 反序列化和签名验证。
- License ID、产品类型、用途、有效期、设备容量和功能授权字段解析。
- 旧功能 ID → Code → 功能名称 → 菜单路径映射。
- 当前 License 和历史 License 展示。
- License 功能检查和设备容量限制使用的授权快照。
- `.lic` 上传、校验、当前 License 替换和历史留档。
- 中英文 Feature List 展示及真实 `.lic` 下载。

## 2. 旧 License 运行资料

### 2.1 公钥库

运行时使用旧项目的 License 公钥库：

```text
license-run-time/keystore/omcPublicKey.store
```

该文件是 JKS 公钥库，不是 TLS client JKS，也不是签发端私钥库。

| 配置 | 值 |
|---|---|
| keystore 类型 | JKS |
| alias | `omcPublicKey` |
| 用途 | 读取 DSA 公钥证书，验证 `.lic` 签名 |
| SHA-256 | `1650aebbb63f320408ade0bc75128eec45d423d50a6ae61e4e368a30216f3331` |

生产部署中通过只读挂载提供给 App：

```text
/etc/omc/license/omcPublicKey.store
```

### 2.2 STOREPWD

旧项目的 `STOREPWD` 同时用于：

- TrueLicense `DefaultCipherParam`，解密 `.lic`。
- TrueLicense `DefaultKeyStoreParam`，读取 JKS 公钥库并校验 JKS integrity。

密码不得写入源码、镜像或普通日志。Docker 部署通过 `OMC_LICENSE_STORE_PASSWORD` 运行时注入。代码会去掉 Secret 文件末尾的 CR/LF，避免回车混入密码派生过程。

`extra.encryptKey` 是 `.lic` 解码后业务字段，用于旧项目 `check_au_info` 状态加密；它与 `STOREPWD` 不是同一个密码。

## 3. TrueLicense 协议

已通过旧 JAR 反编译和真实样本确认：

```text
PBEWithMD5AndDES
  salt = CE FB DE AC 05 02 19 71
  iterations = 2005
  key = final MD5 state[0:8]
  iv  = final MD5 state[8:16]
  ↓
PKCS#5 unpadding
  ↓
GZIP
  ↓
JavaBeans XMLDecoder XML
  ↓
GenericCertificate
  ↓
LicenseContent XML
  ↓
SHA1withDSA signature verification
```

DSA 签名覆盖 `GenericCertificate.encoded` 字符串的原始 UTF-8 字节，签名编码为 US-ASCII/Base64。

JKS integrity 摘要计算顺序为：

```text
UTF-16BE(STOREPWD) + "Mighty Aphrodite" + JKS 内容（不含尾部 20 字节摘要）
```

生产路径启用严格 JKS integrity 校验。真实样本、错误密码和篡改样本均有 Go 回归测试。

## 4. Go 实现位置

| 文件 | 职责 |
|---|---|
| `omcgo/internal/license/legacy_truelicense_decoder.go` | PBE/DES、GZIP、XMLDecoder 兼容解析、字段提取 |
| `omcgo/internal/license/legacy_truelicense_signature.go` | JKS 解析、X.509 DSA 公钥提取、SHA1withDSA 验签、JKS integrity |
| `omcgo/internal/license/legacy_license_claims.go` | 旧字段到 Go Claims、产品类型、日期、容量、功能 ID/Code 映射 |
| `omcgo/internal/license/legacy_feature_mapping.go` | 旧 ID/Code、功能主表、菜单关联和菜单路径标准化 |
| `omcgo/internal/license/system_license_service.go` | `.lic` Update、重复 ID 检查、当前替换、历史留档、读取时功能快照补算 |
| `omcgo/internal/license/feature_checker.go` | 当前授权功能路径检查 |
| `omcgo/cmd/app/provider/modules.go` | JKS 和功能映射资源注入 |
| `omcmb/webcode/src/pages/SystemLicense/FeatureListView.tsx` | 旧 OMC 风格功能分组、中英文名称和紧凑滚动布局 |

新版 RSA/JSON License 验签入口已移除，避免形成第二套 License 功能。

## 5. 功能映射资料

旧项目映射资料已一次性转换为仓库自有静态资产：

```text
omcgo/data/license-feature-mapping.json
```

旧项目 Java/SQL 只作为本次生成该 JSON 的参考输入，不进入仓库运行时，也不由 Go
运行时解析。JSON 内已打平 ID→Code、功能名称、父级和中英文菜单路径。

映射链路：

```text
License supportFeatureIdStr
  -> license-feature-mapping.json 的 id_to_code
  -> supportFeatureCode
  -> features 中的中文名 / 英文名 / 功能树父级 / 菜单路径
```

资料规模：

- ID → Code 映射：92 条。
- 功能树目录节点：124 个。
- 标准化功能项：103 个（包括只有菜单关联、没有功能主表记录的 Code）。

`CODE_TOOL_DHCP` 的已确认映射为：

```text
80 -> CODE_TOOL_DHCP -> DHCP -> 高级 / DHCP
```

功能主表没有登记但菜单关联存在的 Code 也已写入 JSON，并使用菜单路径生成展示名称；例如
`CODE_TOOL_DHCP` 映射为 `高级 / DHCP`。

## 6. 旧项目业务规则

### 6.1 产品、用途和 UI

| 字段 | 旧值 | 语义 |
|---|---:|---|
| `productType` | `1` | Cloud |
| `productType` | `2` | Local |
| `useFor` | `1` | Commercial |
| `useFor` | 其他 | 旧代码按 Test 处理 |
| `uiType` | `0` | NoLogo |
| `uiType` | `1` | BaiCells |
| `uiType` | `2` | RY |
| `uiType` | `3` | TIANYI |
| `uiType` | `4` | RH |
| `uiType` | `5` | FengHuo |

未知枚举保留原值，不自行扩展旧项目语义。

### 6.2 设备容量

| License 字段 | 展示类型 |
|---|---|
| `maxDeviceCapability` | eNB |
| `gnbMaxNum` | gNB |
| `cpeMaxNum` | CPE |
| `epcMaxNum` | EPC |
| `egwMaxNum` | EGW/WCG |
| `upsMaxNum` | UPS |

`gnbMaxNum` 缺失或为零时，旧逻辑在部分场景回退到 eNB 容量。容量为零、空值和不限制必须保持区分，不能擅自解释。

### 6.3 功能展示

当前 License 的功能展示按旧 OMC 页面组织：

```text
Dashboard / All
MAP / All
eNB
  Monitor
  Maintenance
  Upgrade&Rollback
  Inventory
gNB
CPE
  Monitor
  Maintenance
  Upgrade
  Inventory
EGW / UPS
Alarm
Performance
Advanced
System
```

功能 Code 前缀用于设备归属兜底：

- `CODE_ENB_*` → eNB。
- `CODE_GNB_*` 或 `CODE_GNB` → gNB。
- `CODE_CPE_*` → CPE。
- `CODE_EGW` → EGW。
- `CODE_UPS` → UPS。
- `CODE_ALARM_*` → Alarm。
- `CODE_PERFORMANCE_*` → Performance。
- `CODE_SYSTEM_*` → System。
- `CODE_ADVANCE_*` → Advanced。

旧功能树中没有 Code 的目录节点不作为功能项展示；未知叶子值不能默认授权，也不应直接把数字 ID作为主要名称。

### 6.4 gNB 说明

当前代码目录样本 `NO2022-03-14002` 中：

- gNB 容量为 10,000。
- 具体 gNB 功能 Code 数量为 0。
- 只有 `CODE_GNB` 总能力标志。

因此页面显示 gNB 容量但没有具体 gNB 功能项，是 License 数据本身的授权结果，不是映射丢失。

## 7. API 和存储行为

上传请求使用 JSON HTTP envelope，但 License 内容必须是 `.lic` 原文 Base64：

```json
{
  "raw_content": "<base64 of .lic>",
  "raw_content_encoding": "base64"
}
```

数据库中保存 Base64 原文，以保持当前字符串存储契约；下载时前端解码回原始二进制并保存为 `.lic`。

读取当前 License 和历史 License 时，会根据原始 `legacy_feature_ids` / `legacy_feature_codes` 动态重算 `features`，因此旧快照无需重新上传即可获得最新名称和菜单路径。数据库原始快照不被读取操作修改。

## 8. Docker 部署

Compose 项目必须使用 `omc` 和资源配置：

```bash
export COMPOSE="docker compose -p omc --env-file deployments/docker/resources.env -f deployments/docker/docker-compose.yml"
```

App 需要：

- 挂载 `license-run-time/keystore/omcPublicKey.store` 到 `/etc/omc/license/omcPublicKey.store`。
- 注入 `OMC_LICENSE_STORE_PASSWORD`。
- `legacy_verify_integrity: true`。

常用部署：

```bash
$COMPOSE build app web
OMC_LICENSE_STORE_PASSWORD='<通过 Secret 注入>' $COMPOSE up -d --no-deps --force-recreate app web
```

管理页面地址为 `http://localhost:8081`；`8080` 是 ACS/TR-069 入口。

## 9. 验证记录

已通过：

```bash
cd omcgo
go test ./internal/license ./cmd/app/provider
```

覆盖内容：

- 真实 `.lic` 解密和 GZIP/XML 解析。
- JKS integrity 和 DSA 验签。
- 错误密码拒绝。
- 单字节篡改拒绝。
- 旧项目多份 License 样本解码。
- ID `80` 到 `CODE_TOOL_DHCP`、`高级 / DHCP` 的映射。
- 未知 Code 保留且不授权。
- 旧数据库快照读取时动态补算功能展示。
- 中英文 Feature List 展示。

前端完整 typecheck 当前仍受仓库既有的 `rehype-katex`、`remark-math`、`mermaid` 缺失依赖影响；本 License 改动文件编辑器检查通过，Docker 内 Vite 构建已通过。

## 10. 已知后续工作

以下内容不阻塞当前 `.lic` 解码和展示主链路，但仍需后续按旧 Java 行为补齐：

- `check_au_info` 的累计时长/最后访问时间状态迁移。
- MAC/System UUID 绑定校验的完整业务接入。
- `notBefore`、标准 `notAfter`、`omcNotAfter` 和时间回拨策略的完整运行时接入。
- ID/Code 冲突、自动补码和 Cloud 特殊清洗规则的完整行为向量。
- 原始 Base64 License 的大小、编码和 SHA-256 独立存储元数据。
- Preflight/Commit token、SHA 绑定和原子回滚协议。
- 完整功能检查与页面功能树之间的授权快照统一。

这些工作必须继续以旧项目行为为基线，不得引入新的 License 文件格式。
