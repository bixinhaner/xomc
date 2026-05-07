---
type: review
date: 2026-05-07
author: kevin (shangyingbin)
reviewer: Claude (commit-skill simplified data-asset review)
scope: data
task: T-0098
verdict: PASS
---

# Review Report — omcgo/data/ XML 数据资产入库（T-0098）

## 0. Scope

| 项 | 值 |
|---|---|
| Commit type | `chore` |
| 变更文件数 | 30（全部新增） |
| 新增行 | 13977 |
| 总大小 | ~3.4 MB |
| Backlog Task | T-0098（沿用同批前一个 commit 登记的设计稿；hash 见 git log） |
| 审查模式 | 数据资产简化审查（格式 / 安全 / 与设计稿一致性） |

## 1. 文件清单

| 子目录 | 文件数 | 内容性质 |
|---|---|---|
| `omcgo/data/alarm-definitions/` | 7 XML | CPE / EGW / ENB / EPC / GNB / OMC / UPS 告警库（identifier / 严重级 / 名称 / 原因 / 建议） |
| `omcgo/data/indicator-library/` | 2 顶层 XML + `enb/` 8 XML | KPI Counter / KPI 定义 / 平台公式 / 功能集（GNB / GSM / ENB 系列） |
| `omcgo/data/param-mappings/` | 13 XML | 参数模型（9 paramModel）+ standard-model + 2 routing + products |

## 2. 格式与编码核查

| 检查项 | 结果 |
|---|---|
| 全部为 `*.xml` 文件 | ✅ 30/30 |
| `<?xml version="1.0" encoding="UTF-8"?>` 头部 | ✅（抽查 products.xml / CPE.xml） |
| 无二进制大文件 | ✅ 全部纯文本 XML |
| 无 .DS_Store / 临时文件 | ✅ |

## 3. 安全核查（无凭据 / 密钥 / Token 泄漏）

`grep -ril "password|secret|token|apikey|api_key"` 命中 5 个 XML，**逐一确认均为参数路径定义**（TR-069 数据模型 schema），非实际凭据值：

| 文件 | 命中样例 | 性质 |
|---|---|---|
| param-mappings/MLQ.xml 等 | `<param name="Device.DeviceInfo.X_COM_Localweb_password" .../>` | 参数路径声明，描述"该 path 上存在 password 参数"，无实际密码值 |
| param-mappings/MLQ.xml 等 | `<param name=".../TUNNEL_CONFIG_SECRETKEY" .../>` | 同上，IPSec 密钥的参数路径声明 |

**结论**：✅ 无凭据泄漏风险。

## 4. 与设计稿一致性核查（呼应 commit `e8aae25d`）

设计稿 §9 文件清单声明：
- `param-mappings/` 11 XML（9 paramModel + 1 standard + 1 products；旧 product-name-routing.xml + param-model-routing.xml **已合并为 products.xml**）
- `indicator-library/` 10 XML（8 ENB + GSM + GNB；BAIBLQ.xml **已合并入 BLQ.xml**）
- `alarm-definitions/` 7 XML

实际入库：
- `param-mappings/` **13 XML**（含 products.xml 已生成 + 旧 product-name-routing.xml / param-model-routing.xml 仍存在 → 与设计稿"已合并"陈述存在差异）
- `indicator-library/` **10 XML**（GNB + GSM + enb/ 下 8 XML，含 ALL/BLQ/BLX/BM/ENB_DEFAULT_098/181/MLN/MLQ）✅
- `alarm-definitions/` **7 XML** ✅

**Finding W1（WARNING）** — param-mappings 比设计稿描述多 2 个 routing XML：
- 现状：products.xml 已生成（替代 routing 表的设计目标），但旧 product-name-routing.xml / param-model-routing.xml 也一并入库
- 影响：当前阶段无影响（XML 入库不等于 loader 实际消费），但 followup 实施 loader 时需明确：以 products.xml 为权威源，旧两张 routing XML 仅作过渡参考或可删
- 建议：S5 拆 followup 实施 loader 时删除旧 routing XML，或在设计稿 §9 显式说明"过渡期保留"

## 5. 体积与 git 影响

| 指标 | 值 | 评估 |
|---|---|---|
| 总大小 | 3.4 MB | 可接受（远低于 git 单文件 100MB 上限，远低于触发 LFS 必要的规模） |
| 单文件最大 | standard-model.xml 316 KB / 2008 行 | 可读 diff 体验可接受 |
| 后续变更频率 | 低（启动期字典数据，业务变更才更新） | git 历史负担小 |

**结论**：直接 git 入库合理，无需 git LFS。

## 6. Findings

| 严重级 | 数量 | 详情 |
|---|---|---|
| CRITICAL | 0 | — |
| WARNING | 1 | W1（routing XML 与设计稿描述存在差异，followup 实施 loader 时再清理；不阻断本次入库） |
| INFO | 0 | — |

## 7. Verdict

**PASS** — XML 格式合法、无凭据泄漏、体积合适直接 git 入库；与设计稿 §9 文件清单一处差异已 finding 登记，留 followup 在实施 loader 时收口。

---

**注**：本审查报告对应 chore 类型数据资产入库；与同批前一个 commit（T-0098 设计稿）共同构成 S2 设计阶段交付物。
