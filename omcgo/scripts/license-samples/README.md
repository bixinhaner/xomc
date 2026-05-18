# License 管理模块 — 手动测试套件

> 一份完整的 license 管理功能验证清单 + 8 个测试样本 + 一键灌入脚本。
> 适用于 dev/staging 环境快速验证。

---

## 1. 模块功能概览

| 功能 | 后端入口 | 前端入口 |
|------|---------|---------|
| 导入 (Import) | `POST /api/v1/licenses/import` | `/license/operations` Tab 1 |
| 激活 (Activate) | `POST /api/v1/licenses/activate` | `/license/operations` Tab 2 |
| 撤销 (Revoke) | `POST /api/v1/licenses/:id/revoke` | `/license/operations` Tab 3 |
| 单 license 导出 PDF | `GET  /api/v1/licenses/:id/export?format=pdf` | `/license/operations` Tab 4 |
| 批量 active 导出 CSV | `GET  /api/v1/licenses/export?format=csv` | `/license/operations` Tab 4 |
| 全量列表 | `GET  /api/v1/licenses` | `/license/list` |
| 配额查询 | `GET  /api/v1/licenses/quota` | 仪表盘 / 顶 bar |
| 审计日志 | `GET  /api/v1/licenses/logs` | `/license/logs` |

**核心策略**：
- **同维度互斥**：每个 `(device_type, region)` 组合**只允许一张 active license**；激活冲突时返 409，用户确认后 `force=true` 自动 revoke 旧的
- **容量告警**：默认阈值 60/80/95%，超过任一阈值触发 `capacity_alert` 审计日志
- **过期监控**：默认提前 7/3/1 天预警 (`expiry_alert`)，过期后 `monitor.go` 自动转 `auto_expire`
- **签名验证**：非 strict 模式下，未签名/签名失败仍允许导入，但 `signature_status` 字段反映真实状态

---

## 2. License 文件格式（JSON）

```json
{
  "license_name":  "<必填，VARCHAR 200>",
  "license_code":  "<必填，UNIQUE，VARCHAR 100>",
  "product_name":  "<必填，VARCHAR 200>",
  "license_type":  "subscription | perpetual | trial",
  "status":        "pending | active | revoked | expired",
  "max_devices":   <int>,
  "used_devices":  0,
  "features":      ["base", "alarm", "performance", ...],
  "issue_date":    "<必填，RFC3339>",
  "expiry_date":   "<RFC3339 或 null（永久）>",
  "licensor":      "Baicells OEM",
  "device_type":   "eNB | gNB | ...",
  "region":        "cmcc | ctcc | cucc | ...",
  "notes":         "..."
}
```

**可选签名字段**（走 `signed_license_json` 路径才生效）：
```json
{
  ...上面字段...,
  "signature":         "<base64 RSA-PSS over canonical(去 signature/signature_key_id 后字典序 JSON)>",
  "signature_key_id":  "<OEM 公钥 SHA-256 fingerprint hex>"
}
```

> 当前 sample 不带 signature，导入会被标记 `signature_status="unverified"`，但**不阻断**（strict=false 默认）。

---

## 3. 8 个测试场景

| 文件 | 场景 | 用途 |
|------|------|------|
| `01-perpetual-enb-cmcc.json` | 永久授权 / eNB / cmcc / 100 设备 | 测 perpetual 类型 + 设备类型/运营商组合 |
| `02-subscription-1year-enb-ctcc.json` | 1 年订阅 / eNB / ctcc / 500 设备 | 测 subscription 类型 + 与 01 不同维度共存 |
| `03-trial-30days-gnb-cucc.json` | 30 天试用 / gNB / cucc / 50 设备 | 测 trial 类型 + gNB |
| `04-expiring-soon-7days.json` | 即将过期（剩 7 天） | 测 `expiry_alert` 7d 阈值触发 |
| `05-already-expired.json` | 已过期 30 天 | 测 `monitor.go` 的 `auto_expire` + grace_period 宽限期 |
| `06-large-capacity-10k.json` | 10000 设备规模 | 测容量告警阈值 + 全特性 features |
| `07-conflict-same-dimension.json` | 与 01 同 (eNB, cmcc) | 测 activate 409 冲突 + force 强制激活 + auto_revoke |
| `08-minimal-1-device.json` | 边界 max_devices=1 | 测 enforcer 容量上限拒绝 |

---

## 4. 一键灌入

```bash
cd omcgo/scripts/license-samples

# 默认 localhost:8081 admin/admin123
./seed-licenses.sh

# 或自定义
API=http://172.21.175.129:8081 USER=admin PASS=OMC@123456 ./seed-licenses.sh
```

脚本输出形式：
```
→ 登录 ... 获取 access_token...
✓ token 已获取
→ 导入 01-perpetual-enb-cmcc.json
  ✓ imported  license_id=abc...  signature_status=unverified
→ 导入 02-...
  ✓ imported  license_id=...
...
════════════════════════════════════
  imported: 8   skipped: 0   failed: 0
════════════════════════════════════
→ 当前 licenses 列表（前 20 条）：
  total=8
    [pending] TEST-PERP-ENB-CMCC-100      ...
```

重复跑：已存在的会归到 `skipped`，不会重复导入。

---

## 5. 完整手动测试矩阵

### T1 — 导入路径（Tab 1: Import）

| 步骤 | 操作 | 预期 |
|------|------|------|
| T1.1 | 点 "导入许可证" → 切到 "文件导入" → 拖入 `01-perpetual-enb-cmcc.json` | 文件出现在拖拽区 |
| T1.2 | 点 "导入许可证" | 成功 Modal：license_code / license_name / 黄色 `unverified` Tag + signatureWarning |
| T1.3 | 切到 "粘贴模式" → 表单填写完整字段 → 点导入 | 同样成功 |
| T1.4 | 重复导入同一 license_code | 后端返 409 (重复 license_code)；前端 toast 失败 |
| T1.5 | 缺 license_code 字段提交 | 前端 binding required 校验报错 |

### T2 — 激活路径（Tab 2: Activate）

| 步骤 | 操作 | 预期 |
|------|------|------|
| T2.1 | 粘贴 `TEST-PERP-ENB-CMCC-100` 到激活码框 → 点激活 | 成功 toast；License 01 status: pending → active |
| T2.2 | 粘贴 `TEST-CONFLICT-ENB-CMCC-200` (07 同维度) → 点激活 | 弹 Modal "同维度冲突"，列出 License 01 |
| T2.3 | 在冲突 Modal 点 "强制激活" | 后端 auto_revoke License 01，激活 License 07 |
| T2.4 | 此时 License 01 status=revoked，License 07 status=active | 列表页确认状态变化 |
| T2.5 | 粘贴不存在的 code | toast 失败 |

### T3 — 撤销路径（Tab 3: Revoke）

| 步骤 | 操作 | 预期 |
|------|------|------|
| T3.1 | 下拉选择某 active license → 点撤销 | 弹二次确认 Modal |
| T3.2 | 确认 → 撤销成功 | 该 license status=revoked，下拉列表移除 |
| T3.3 | 当只剩 1 张 active license 时撤销 | Modal 内额外显示"撤销最后 1 张"警告 |

### T4 — 导出路径（Tab 4: Export）

| 步骤 | 操作 | 预期 |
|------|------|------|
| T4.1 | 选择某 active license → 点"导出 PDF" | 下载 `license-<id>.pdf` 文件 |
| T4.2 | 点"批量导出 CSV" | 下载 `licenses-active-YYYYMMDD.csv`，含全 active license 行 + BOM 头 |
| T4.3 | 无 active license 时 | 单 PDF 按钮 disabled；批量 CSV 仍可下载（空文件） |

### T5 — 审计日志（`/license/logs`）

| 步骤 | 操作 | 预期 |
|------|------|------|
| T5.1 | 进入审计日志页 | 列出 T1-T4 期间产生的所有 license_logs 行 |
| T5.2 | 按 log_type 过滤 `import` | 仅显示 import 类日志 |
| T5.3 | 按 result 过滤 `failed` | 显示 T1.4 / T1.5 / T2.5 的失败行 |
| T5.4 | 按时间范围过滤 | 范围内的日志正确显示 |
| T5.5 | 看 License 01 在 T2.3 被 auto_revoke 的日志 | log_type=`auto_revoke_by_activate`，actor=系统 |

### T6 — License 列表（`/license/list`）

| 步骤 | 操作 | 预期 |
|------|------|------|
| T6.1 | 列表按 status 过滤 | 4 个 status 切换都返回正确子集 |
| T6.2 | 按 license_type 过滤 | perpetual / subscription / trial 都过滤正确 |
| T6.3 | 按 device_type 过滤 | eNB / gNB 子集正确 |
| T6.4 | 点单 license 看详情 | 显示完整字段，含 features 数组 |

### T7 — 容量与过期监控

| 步骤 | 操作 | 预期 |
|------|------|------|
| T7.1 | 激活 License 08（max=1）后注册 1 台设备 | OK |
| T7.2 | 注册第 2 台 | enforcer 拒绝，写 `enforcement_capacity` 审计 |
| T7.3 | 激活 License 04（剩 7 天），等 monitor 周期跑 | 触发 `expiry_alert` 审计（warning 级） |
| T7.4 | 激活 License 05（已过期），等 monitor 周期 | 触发 `auto_expire`，status: active → expired |
| T7.5 | License 06（10K 容量）注册到 60%/80%/95% | 各触发一次 `capacity_alert` 审计 |

### T8 — 高频调用回归（faf2507c 修过的 bug）

| 步骤 | 操作 | 预期 |
|------|------|------|
| T8.1 | 打开 `/license/operations`，开浏览器 Network 面板 | `/licenses/logs` 仅调 1 次（不是高频毫秒级递增） |
| T8.2 | Tab 间切换 / 表单输入 / 上传文件 | 不再触发新的 `/licenses/logs` 请求 |

---

## 6. 用 curl 直接测试单个 API

```bash
# 登录拿 token
TOKEN=$(curl -sS -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c 'import json,sys; print(json.load(sys.stdin).get("data",{}).get("access_token"))')

# 导入
curl -X POST http://localhost:8081/api/v1/licenses/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d @01-perpetual-enb-cmcc.json

# 激活（用 license_code，不是 id）
curl -X POST http://localhost:8081/api/v1/licenses/activate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"license_code":"TEST-PERP-ENB-CMCC-100","force":false}'

# 列表 + 过滤
curl "http://localhost:8081/api/v1/licenses?status=active&page=1&page_size=50" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 审计日志
curl "http://localhost:8081/api/v1/licenses/logs?log_type=import&log_type=activate&page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 撤销（用 id，不是 code）
LIC_ID="<从列表里拿的 UUID>"
curl -X POST "http://localhost:8081/api/v1/licenses/$LIC_ID/revoke" \
  -H "Authorization: Bearer $TOKEN"

# 单 PDF 导出
curl -OJ "http://localhost:8081/api/v1/licenses/$LIC_ID/export?format=pdf" \
  -H "Authorization: Bearer $TOKEN"

# 批量 CSV 导出
curl -OJ "http://localhost:8081/api/v1/licenses/export?format=csv" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 7. 验收 Checklist（按本目录 8 sample 完整跑一遍）

- [ ] T1 全 5 项通过（导入 / 重复 / 校验）
- [ ] T2.1-T2.5 通过，**T2.2 弹同维度冲突 Modal**，T2.3 强制激活成功 auto_revoke 01
- [ ] T3 全 3 项通过（撤销 / 二次确认 / 最后 1 张警告）
- [ ] T4 全 3 项通过（PDF / CSV / disabled 状态）
- [ ] T5 审计日志能看到 T1-T4 产生的所有 log_type，含 `auto_revoke_by_activate`
- [ ] T6 列表过滤全维度正确
- [ ] T7 容量/过期监控四个子项均触发（需等 monitor 周期）
- [ ] T8 `/licenses/logs` 不再高频调用（commit `766b461c` 修复回归）

---

## 8. 清理（测试后回滚）

```bash
# 用 psql 清除 TEST- 开头的全部 license + 关联日志
psql -h localhost -U omcgo -d omcgo <<EOF
DELETE FROM license_logs
WHERE license_id IN (SELECT id FROM licenses WHERE license_code LIKE 'TEST-%');
DELETE FROM licenses WHERE license_code LIKE 'TEST-%';
EOF
```

或单条删（如果有 admin API）：
```bash
curl -X DELETE "http://localhost:8081/api/v1/licenses/$LIC_ID" \
  -H "Authorization: Bearer $TOKEN"
```

---

## 9. 已知限制

- **签名验证**：本目录所有 sample **不带 signature 字段**。导入时 `signature_status="unverified"`，UI 显示黄色警告 Tag，但不阻断。要测真验证：
  1. 生成 RSA 密钥对，公钥放 `configs/oem_public_keys/*.pem`
  2. 用私钥对 license 做 RSA-PSS 签名 → 嵌 signature 字段
  3. 后端 `SignatureVerifier` 自动加载公钥并验证
  4. 想拒绝未签名 → 启动时 `strict=true`
- **前端 file upload bug**：`importSignedLicense` 只传 `signed_license_json` 字段，**不传结构化字段**，而后端 `binding:"required"` 的 license_name/license_code/product_name/issue_date 4 字段会让请求 400 失败。**这是已知 bug，暂时绕过**：用粘贴模式或 curl 直接调 `/licenses/import` 走结构化路径。
- **配额/告警监控**：`monitor.go` 默认周期是 1 分钟（可配），不会即时触发。手动测 T7 时给 monitor 留 1-2 分钟落地时间。
