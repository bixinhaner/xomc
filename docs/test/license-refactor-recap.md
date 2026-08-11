# License 重构 — 前情提要与验证调测计划

> 配套文档：[`license-capacity-feature-e2e-test-plan-20260806.md`](./license-capacity-feature-e2e-test-plan-20260806.md)（详细 E2E 用例集 A~H 组）。本文是"做到哪了 + 接下来怎么测/怎么调"的简明手册，重点是**可执行的验证手段**（含伪造 license / 伪造设备）。两者交叉引用：本文 §4 速测 ↔ e2e 各组。

## 1. 前情提要（本轮已完成）

OMC License 从旧 Java multi-license 模型重构为 Go singleton `system_license` 模型。本轮围绕容量/功能/菜单/硬件/时长控制，落地了：

1. **fail-closed**：无 license 时受控业务（设备创建等）返回 `ErrLicenseUnavailable`/403，对齐旧项目"无 license 安全拒绝"。顺带修了 `device_handler.CreateDevice` 把容量/过期拒绝错判成 500 的 bug（改走 `HTTPStatusFromError`，现在三类拒绝都 403）。
2. **feature-check 接通（选 B）**：上传/读取时从 feature code 前缀派生嵌套授权树（`feature_list.authorization_tree`），`CheckFeature` 解树后返回正确 `true/false`。派生规则：`CODE_<MODULE>[_<FEATURE>]` → `Module[.Feature]`，如 `CODE_ENB_MONITOR`→`eNB.Monitor`、`CODE_DASHBOARD`→`Dashboard`。
3. **P7-A 菜单按 license 显隐**：`menus` 表加 `feature_code text[]`（OR 语义，全量映射入库），后端 `GetUserMenuTree`/`GetUserMenuTreeByRole` 按 license 过滤。无 license→只剩 `/license`；有 license→按授权 feature 显隐；超管也受控。
4. **STOREPWD 开箱即用**：`config.dev.yaml`/`config.prod.yaml` 默认 `${OMC_LICENSE_STORE_PASSWORD:-bcb9omc6}`，别人构建 docker 不再因缺 env 而上传失败。
5. **MAC/UUID 硬件绑定**：上传时校验，docker 挂载宿主机 `/sys:/host/sys:ro` 读**物理机** MAC/UUID（归一化比对，多值逗号、大小写不敏感），失配→拒绝上传 403；为空=不绑定。
6. **累计使用时长 + 时间回拨**：`system_license_usage` 表 + enforcer compute-on-read（`time_limit_hours` 存 feature_list）；累计超限 / 系统时间回拨 → 403；持久化错误 fail-open。

## 2. 当前能力与边界

| 能力 | 状态 | 备注 |
|---|---|---|
| .lic 解码/验签（SHA1withDSA）| ✅ | TrueLicense 容器 |
| 设备容量 / 过期拦截 | ✅ | 管理面 `CreateDevice` + 南向 `RegisterFromInformEvent`（E-04 已修复）；无 license fail-closed |
| 无 license 菜单只剩 /license | ✅ | 后端裁剪 + 前端 `licenseOnlyMode` |
| feature 菜单显隐（P7-A） | ✅ | 机制 + 全量映射（63 菜单；`/license` 固定 NULL 防死锁；新模块菜单挂靠已有 code，可在 seed 调） |
| feature-check 查询 | ✅ | 按 code 派生 path |
| **API middleware 按 feature 拦（P7-B）** | ✅ 已完成 | 全部业务路由已挂 `RequireFeature`；仅 license 管理 / 北向集成 / bundle 批量下载 保持不挂（有意为之） |
| **MAC/UUID 硬件绑定** | ✅ | 上传时校验，不匹配拒绝（403）；复刻旧项目多值/大小写不敏感语义 |
| **累计使用时长 / 时间回拨** | ✅ | enforcer 过期检查 compute-on-read + `system_license_usage` 表；超限/回拨→403 |
| **南向 Inform 容量/过期拦截（E-04）** | ✅ | `RegisterFromInformEvent` 接入 enforcer，与管理面 `CreateDevice` 对齐；已注册设备更新不受影响 |
| **告警恢复自动清除** | ✅ | monitor 条件解除时调 `sink.Clear`（→`engine.AutoClear`）；上传新 license 后下一 tick 自动清旧告警 |
| **累计时长全局周期推进** | ✅ | monitor 每 5min 推进 `system_license_usage`；GetCurrent 响应含 `is_expired`/`cumulative_used_hours`/`cumulative_limit_hours` |
| 多实例 enforcer 跨实例失效 | ➖ 暂不做 | 当前单实例部署，无此问题；横扩时参照 ACS issue #65 Redis 化缓存失效 |

## 3. 环境与前置

- docker compose 起 `app/acs/postgres/redis/web`；管理面 `http://localhost:18081`。
- STOREPWD 现已默认，**无需手动设 env**。
- 内部调测可用 `.api-key`：`KEY=$(docker exec omc-app-1 cat /var/lib/omcgo/secrets/.api-key)`，请求带 `-H "X-API-Key: $KEY"`。
- ⚠️ **dev 库整体落后于基线**：`provisioning_tasks.policy_id`/`parameter_sync.admission_queued_at`/`storage_protection_policies` 等缺失（日志吵，与 license 无关）。要彻底干净建议重置一次 dev 库（封版本后用增量迁移补齐）。

### 3.1 当前环境现状快照（接手时参考；随测试可能变动）

- **app 运行最新代码**（2026-08-10 重建）：fail-closed / feature-check / 菜单显隐 / MAC+UUID 绑定 / 累计+回拨 / **E-04 南向拦截** / **P7-B 全路由** / **告警恢复清除** / **累计周期推进** 全在线（`omc-app-1` 含 `/sys:/host/sys:ro` 挂载）。
- **dev 库（`omc-postgres-1`，db=`omcgo`）已手动补**：`menus.feature_code` 列 + 全量映射 + `system_license_usage` 表。这些已折回 `migrations/000001`；**全新 `down -v && up` 会由 schema+seed 自动生成，无需手动**。仅当前这个旧 dev 库是手动 catch-up 的。
- **license 当前已安装**：`NO2026-08-10007`（Commercial，expiry 2027-01-02，eNB:200000 等）。验证无 license 前先 `DELETE FROM system_license;`，并**重启 app 清 enforcer 5min 内存缓存**（见 §4.A 末尾）。
- **本机物理 MAC**：`eno1 = a4:bb:6d:bb:fe:d5`（`a4bb6dbbfed5`）——申请/构造硬件绑定 license 时填这个；查：`docker exec omc-app-1 cat /host/sys/class/net/eno1/address`。
- 容器 eth0 的 MAC 是 docker 动态分配的（重建会变），**不要用它绑定**。
- **ACS 端口**：`http://localhost:7557/smallcell/AcsService`（cpe_simulator 用这个）。

## 4. 验证清单（可直接执行）

### A. 无 license 行为

**快速卸载 license（后台一行命令，无 DELETE API，直接 SQL）：**
```bash
docker exec omc-postgres-1 psql -U omcgo -d omcgo -c \
  "DELETE FROM system_license; SELECT count(*) AS remaining FROM system_license;"
```
- 删的是 current 行；`system_license_history`（审计）保留不动。
- **enforcer 有 5min 内存缓存**：删后菜单/license 端点（实时读库）立即变无 license；但 `device.create` 的容量/过期裁决可能最长 5min 内仍看到旧 license。要立即全清，重启 app：`docker compose -p omc --env-file deployments/docker/resources.env -f deployments/docker/docker-compose.yml restart app`。

验证无 license：
- `GET /api/v1/system-license` → 404 / biz_code 12113
- `GET /api/v1/auth/menus` → 只剩 `/license`
- `POST /api/v1/devices` → 403 `ErrLicenseUnavailable`（fail-closed）

### B. 容量测试（构造假设备）

> 直接控制点是**管理面 `POST /api/v1/devices`**。南向 CPE Inform 自动注册**已接入 enforcer**（E-04 已修复，两条路径均受 license 约束，见 B-3）。

**B-1 设小容量 license（SQL 伪造，无需签名）：**
```sql
DELETE FROM system_license;
INSERT INTO system_license (license_id, license_type, issued_at, expiry_date,
                            devices_support, feature_list, raw_content, signature_status, is_current)
VALUES ('CAP-TEST','Commercial',now(),'2099-01-01',
        '{"eNB":2}'::jsonb,
        '{"legacy_feature_codes":[],"authorization_tree":{}}'::jsonb,
        '', 'verified', true);
```
容量口径：`devices_support` 所有正容量求和 = 总容量（这里=2）。设备计数是**全表 `COUNT(*)`**，注意先清/算上存量设备。

**B-2 管理面造假设备到边界（脚本循环）：**
```bash
KEY=$(docker exec omc-app-1 cat /var/lib/omcgo/secrets/.api-key)
for i in $(seq 1 3); do
  SN="CAP-TEST-$(date +%s)-$i"
  echo "== create $SN =="
  curl -s -o /dev/null -w "%{http_code}\n" -X POST -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
    http://localhost:18081/api/v1/devices \
    -d "{\"serial_number\":\"$SN\",\"oui\":\"48BF74\",\"product_class\":\"Femto-Classic\",\"manufacturer\":\"Baicells\",\"carrier\":\"cmcc\",\"technology\":\"lte\"}"
done
# 预期：前 2 台 201，第 3 台 403 ErrLicenseCapacityExceeded；SELECT count(*) FROM devices 仍为 2
```
⚠️ `oui` 是 6 位 hex（`devices.oui` 是 varchar(6)），如 `48BF74`；**`BAICELLS` 是制造商不是 oui，会触发 `value too long for type character varying(6)`**。`manufacturer` 字段才填 `Baicells`。
（请求体字段以当前 `CreateDeviceRequest` 为准；若 4xx 是字段校验失败而非容量，先补全合法请求体，别误判。）

**B-3 CPE Inform 路径（E-04 已修复 + 端到端验证 ✅）：**
```bash
python3 omcgo/scripts/cpe_simulator.py --acs http://localhost:7557/smallcell/AcsService \
  --count 5 --base-sn CAP-INFORM- --event bootstrap --once
```
`RegisterFromInformEvent` 已接入 `EnforceExpiry`/`EnforceCapacity`（`device_service.go`，operation=`device.inform.register`）。容量满后 Inform 自动注册同样被拒（返回 license 错误 → 事件重试，直到 license 允许）。**已注册设备**的 Inform 更新不走 enforcer，不会因 license 过期/容量满被断开。
> **2026-08-10 验证结论**：容量=2 时发 5 台 bootstrap Inform，设备数仍=2，日志 `"license capacity exceeded — device.create denied"` 确认南向拦截生效。

### C. 单 feature 控制（构造 license）

> **核心技巧：直接 SQL 伪造 `system_license` 行，手设 `feature_list.authorization_tree`，测菜单/feature-check，无需离线签发工具**（仓库只有公钥库，无私钥）。

**C-1 只授权 Alarm.View：**
```sql
DELETE FROM system_license;
INSERT INTO system_license (license_id, license_type, issued_at, expiry_date,
                            devices_support, feature_list, raw_content, signature_status, is_current)
VALUES ('FEAT-ALARM','Commercial',now(),'2099-01-01',
        '{"eNB":100}'::jsonb,
        '{"legacy_feature_codes":["CODE_ALARM_VIEW"],"authorization_tree":{"Alarm":{"View":"All"}}}'::jsonb,
        '', 'verified', true);
```
预期：
- `GET /system-license/feature-check?path=Alarm.View` → `authorized=true`；`?path=eNB.Monitor` → `false`
- 菜单：`/alarm/current` 可见；`/device/list`、`/topology/canvas`、`/dashboard` 等带 `feature_code` 的被隐藏
- `feature_code=NULL` 的菜单（MML/系统配置等，映射待补）仍可见

**C-2 path 派生对照（构造 `authorization_tree` 用）：**
| feature code | path | 树片段 |
|---|---|---|
| `CODE_ALARM_VIEW` | `Alarm.View` | `{"Alarm":{"View":"All"}}` |
| `CODE_ENB_MONITOR` | `eNB.Monitor` | `{"eNB":{"Monitor":"All"}}` |
| `CODE_TOPO` | `Topo` | `{"Topo":"All"}` |
| `CODE_SYSTEM_USERS_USER` | `System.Users` | `{"System":{"Users":"All"}}` |
| `CODE_DASHBOARD` | `Dashboard` | `{"Dashboard":"All"}` |

多 feature 组合：`{"Alarm":{"View":"All"},"eNB":{"Monitor":"All"},"Topo":"All"}`。

**C-3 测真实签名 `.lic`**：需离线签发环境（私钥 + 旧 Java 签发工具），仓库内无法签。

### D. 过期 / fail-closed 区分
```sql
UPDATE system_license SET expiry_date = now() - interval '1 day' WHERE is_current;
```
- `POST /devices` → 403 `ErrLicenseExpired`（与容量 403 错误码可区分）
- `GET /system-license` → `is_expired: true`（日期过期 + 累计超限均触发）
- 累计超限验证：`UPDATE system_license SET feature_list = jsonb_set(feature_list, '{time_limit_hours}', '1') WHERE is_current;`，等 5min 周期推进后 `cumulative_used_hours >= cumulative_limit_hours` → `is_expired: true`

### E. 告警恢复自动清除
- 触发告警：`UPDATE system_license SET expiry_date = now() + interval '5 days' WHERE is_current;` → 等 daily tick（01:00 UTC）发 `license_expiring_7d`
- 恢复：`UPDATE system_license SET expiry_date = now() + interval '365 days' WHERE is_current;` → 下一 daily tick 自动清 `license_expiring_*`（`engine.AutoClear`）

## 5. 修 bug 查这里（关键文件）

- 容量/过期/fail-closed：`omcgo/internal/license/enforcer.go`
- 设备创建拦截 + 错误映射：`omcgo/internal/device/device_service.go:2087`（管理面 `CreateDevice`）、`RegisterFromInformEvent`（南向 Inform，E-04）、`device_handler.go:268`
- 菜单 license 过滤：`omcgo/internal/admin/service.go`（`applyLicenseMenuGate`/`applyLicenseFeatureGate`/`ensureLicenseMenuVisible`/`filterLicenseOnlyMenus`）
  - 无 license 时 `filterLicenseOnlyMenus` 只保留 `/license`；有 license 时 `applyLicenseFeatureGate` 按 feature 过滤 + `ensureLicenseMenuVisible` 确保 `/license` 始终可见
- feature 派生树：`omcgo/internal/license/legacy_feature_mapping.go`（`AuthorizationTree`/`codeToPath`）+ `feature_checker.go`
- license 拒绝 biz_code 映射：`omcgo/internal/core/errors/errors.go`（`bizCodeFromSentinel` + `AbortWithError`）+ `global/errors.go`（12116）
- P7-B feature 中间件：`omcgo/internal/license/feature_middleware.go`（`RequireFeature`）+ `cmd/app/provider/router.go`（`featGroup`）
- 告警恢复清除：`omcgo/internal/license/monitor.go`（`clearExpiryAlerts`/`clearCapacityAlerts`/`clearCumulativeAlerts`）+ `cmd/app/provider/license_alert_sink.go`（`Clear` → `engine.AutoClear`）
- 累计周期推进：`omcgo/internal/license/monitor.go`（`CheckCumulativeUsage`，5min cron）+ `system_license_service.go`（`enrichCumulativeStatus`，填充 `is_expired` 等 transient 字段）+ `pg_system_license_usage.go`（`CurrentUsage` 只读）
- 菜单↔feature 映射数据：`migrations/seed/000001_init_seed.sql`（feature_code 回填）；dev 库已手动 `ALTER TABLE menus ADD COLUMN feature_code text[]` 补列

## 6. 后续计划

1. ~~补全菜单↔feature 映射~~ ✅ 已完成：63 个菜单全部映射（`/license` 固定 NULL）；新模块菜单（产品中心/文件传输/运维等）按最近语义挂靠已有 code，可在 seed 调整。
2. ~~复刻旧项目 feature code 补全规则~~ ✅ 已完成：`ExpandFeatureCodes` 复刻 sysList/gnbList/getOrtherCode/delAdvanceControl/delAdvancePlug，真实 `.lic` 菜单表现与旧项目对齐。
3. ~~**P7-B（API middleware 按 feature 拦截）**~~ ✅ 已完成。`license.RequireFeature` 中间件已挂到全部业务路由（设备/告警/性能/拓扑/仪表板/MML/配置模板/下发/同步/固件/文件传输/运维/备份/日志/报文跟踪/通知/报表等）。未授权 feature → 403 biz_code 12114。有意不挂的三类：`systemLicenseHandler`（license 管理入口须永远可访问）、`nbRouter`（北向系统集成）、bundle 批量下载（粒度碎，RBAC 数据权限兜底）。
4. ~~**E-04 决策**：CPE Inform 自动注册接入 enforcer~~ ✅ 已完成：`RegisterFromInformEvent` 加 `EnforceExpiry`/`EnforceCapacity`（operation=`device.inform.register`），已注册设备更新不受影响。新增 3 个测试（容量超限/过期/已注册设备不拦）。
5. **dev 库重置 / 封版本增量迁移**：补齐 provision/param-sync/storage-protection 等缺失表列。
6. ~~MAC/UUID 绑定 / 累计时长 / 时间回拨~~ ✅ 已完成：硬件绑定上传时校验（403）；累计时长+回拨在 enforcer 过期检查里 compute-on-read。
7. ~~**告警恢复自动清除**~~ ✅ 已完成：`AlertSink` 加 `Clear` 方法；monitor 在 license 恢复/容量回落时遍历清所有 identifier（`engine.AutoClear` 幂等）。上传新 license 后下一 daily/hourly tick 自动清旧告警。
8. ~~**（新缺口）累计时长的全局周期推进**~~ ✅ 已完成：monitor 每 5min 周期推进 `system_license_usage.total_used`（`CheckCumulativeUsage`）；超限发 `license_cumulative_exceeded` 告警，恢复自动清除。`GetCurrent` 响应填充 `is_expired`（日期 + 累计）、`cumulative_used_hours`、`cumulative_limit_hours` 三个 transient 字段，页面/前端可直接读取。enforcement 点（device.create/inform.register）的即时推进不受影响。
9. **多实例 enforcer 跨实例失效**：➖ 暂不做（当前单实例部署）。横扩 app 副本时需参照 ACS issue #65 把缓存失效通知迁 Redis pub/sub。
