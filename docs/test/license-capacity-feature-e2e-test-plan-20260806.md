# License 端到端测试用例集

> 本文件是 **License E2E 调测的详细用例集**（A 导入 / B 容量 / B-0 无 license / C 过期+累计 / D 功能 / E CPE / F 菜单 / G 硬件绑定 / H 累计+回拨）。
> **状态速查、前情提要、快速验证手段（SQL 伪造 license / 造假设备）、修 bug 文件索引** 见配套文档 [`license-refactor-recap.md`](./license-refactor-recap.md)。
> **实现状态**：核心能力（解码验签 / 容量 / 过期 / 无 license fail-closed / feature-check / 菜单显隐 / MAC+UUID 绑定 / 累计时长+时间回拨）均已实现，本文用例可直接执行。

## 1. 文档信息

| 项目 | 内容 |
|---|---|
| 测试对象 | 旧 TrueLicense `.lic` 的导入、容量、有效期、累计时长、功能授权、菜单显隐、硬件绑定 |
| 测试层级 | License 集成 + 管理面 API E2E + TR-069 CPE 接入模拟 |
| 目标环境 | Docker Compose dev/staging（app/acs/postgres/redis/web） |
| 参考实现 | `omcgo/internal/license/`、`omcgo/internal/device/`、`omcgo/internal/admin/`、`omcgo/scripts/cpe_simulator.py` |
| 编写日期 | 2026-08-06（重整） |

## 2. 重要边界与口径

### 2.1 License 文件格式
- 唯一格式：旧项目 TrueLicense `.lic` 二进制；HTTP 请求 JSON 只是传输封装：
  ```json
  {"raw_content":"<.lic 原文 Base64>","raw_content_encoding":"base64"}
  ```
- 测试用不同容量/功能/有效期/绑定的 `.lic` 由**受控旧签发环境**离线生成（仓库只有公钥库，无私钥，无法在线签）。
- **绕过签名的 dev 测试**：可直接 SQL 伪造 `system_license` 行（设 `feature_list.authorization_tree` / `expiry_date` / `devices_support`），见 recap §C。

### 2.2 容量口径
`EnforceCapacity` 把 `devices_support` 所有正容量求和作为总容量；设备计数是**全表 `SELECT COUNT(*) FROM devices`**（不分类型/状态/运营商）。→ 容量边界按全表设备数算；测试库要干净或算上存量。

### 2.3 过期口径
`EnforceExpiry` 三层：①`expiry_date` 日期过期；②累计使用时长 `>= time_limit_hours`；③系统时间回拨（`now < last_visited_time`）。NULL `expiry_date` = 永久（仅跳过①，②③仍生效）。

### 2.4 硬件绑定口径
license 的 `MACAddress` / `systemUUID` 非空才校验；docker 下 app 挂载宿主机 `/sys:/host/sys:ro`，校验读**物理机** MAC（`/host/sys/class/net/*/address`）+ UUID（`/host/sys/class/dmi/id/product_uuid`），归一化（去 `:`/`-`、大写）后比对；多值逗号、大小写不敏感。

### 2.5 CPE 接入 vs 管理面创建
`POST /api/v1/devices` → `CreateDevice` 是容量/过期/硬件的直接控制点；CPE `Inform` 自动注册目前**不经 enforcer**（见 E-04，南向可能绕过容量）。

## 3. 环境与前置

```bash
cd /home/zhanglu/goomc
export COMPOSE="docker compose -p omc --env-file deployments/docker/resources.env -f deployments/docker/docker-compose.yml"
$COMPOSE ps app acs worker web postgres redis
```
- 管理面 `http://localhost:18081`（容器 8081→宿主 18081）；ACS `http://localhost:8080/smallcell/AcsService`。
- App 挂载公钥库 `/etc/omc/license/omcPublicKey.store` + 宿主机 `/sys:/host/sys:ro`。
- `OMC_LICENSE_STORE_PASSWORD` 已默认 `bcb9omc6`（dev/prod config），开箱即用；自定义 keystore 用 env 覆盖。
- 启动日志出现 `License feature mapping loaded`、`legacy TrueLicense decoder configured`。
- 内部调测可用 `.api-key`：`KEY=$(docker exec omc-app-1 cat /var/lib/omcgo/secrets/.api-key)`，请求带 `-H "X-API-Key: $KEY"`。

快速回归：
```bash
cd /home/zhanglu/goomc/omcgo && go test ./internal/license/... ./internal/admin/... ./internal/core/errors/...
```

## 4. 测试数据

### 4.1 `.lic` 样本矩阵（真实签名；离线生成）

| 样本 | devices_support | 到期 | 功能 | 绑定 | 用途 |
|---|---|---|---|---|---|
| L-LARGE | eNB/gNB/CPE=10000 等 | NULL(永久) | 全量 | 无 | 大容量+永久+全 feature |
| L-ONE | eNB=1 | 未来 | base | 无 | 1 台边界 |
| L-BOUNDARY | eNB=2 | 未来 | base+Alarm | 无 | 0/1/2/3 台边界 |
| L-MULTI | eNB=2,gNB=3,CPE=4 | 未来 | 部分 | 无 | 总容量 9 + per_type |
| L-EMPTY | 全 0 | 未来 | 最小 | 无 | 零容量语义 |
| L-30D/7D/1D | eNB=10 | +30/7/1 天 | base | 无 | 到期告警窗口 |
| L-EXPIRED | eNB=10 | -1 天 | base | 无 | 过期拒绝 |
| L-FEATURE-PARTIAL | 足够 | 未来 | 部分已知 path | 无 | 已授权/未授权相邻路径 |
| L-FEATURE-UNKNOWN | 足够 | 未来 | 含未知 ID/Code | 无 | 未知不默认授权 |
| L-GNB-FALLBACK | eNB=5,gNB=0 | 未来 | 含 CODE_GNB | 无 | gNB==0&&CODE_GNB 回退 eNB 容量 |
| L-MAC-BIND | 足够 | 未来 | base | MAC=本机 eno1 | 硬件匹配通过 |
| L-MAC-MISMATCH | 足够 | 未来 | base | MAC=不存在 | 硬件失配拒绝上传 |
| L-UUID-BIND | 足够 | 未来 | base | UUID=本机 | UUID 匹配通过 |
| L-CUMULATIVE | 足够 | 未来 | base | 无 | time_limit_hours=2，测累计超限 |
| L-ROLLBACK | 足够 | 未来 | base | 无 | 配合改系统时间测回拨 |

### 4.2 设备身份策略
每个场景唯一 SN 前缀（`LIC-E2E-ONE-` / `-BOUNDARY-` / `-MULTI-` 等），避免重复设备掩盖结果；测试结束按前缀清理或用独立库。

## 5. 核心端到端场景

### A. 旧 `.lic` 导入与替换

| 编号 | 操作 | 预期 |
|---|---|---|
| A-01 | 上传 L-LARGE（Base64）到 `POST /api/v1/system-license` | 201；`current`，`signature_status=verified`，容量/功能字段与样本一致 |
| A-02 | `GET /api/v1/system-license` | 同 `license_id`/`devices_support`/`feature_list`，`is_current=true` |
| A-03 | 相同 `.lic` 再传 | 409；不新增 current/history |
| A-04 | 改 1 字节后传 | 400；解密/验签失败；旧 current 不变 |
| A-05 | 错 Base64 / `encoding!=base64` | 400；错误不泄漏 STOREPWD |
| A-06 | 上传 L-BOUNDARY 替换 L-LARGE | 201；旧 License 进 history，新 License 成 current；enforcer 缓存立即失效 |
| A-07 | `GET /system-license/history` | 看到被替换旧 License，快照可读 |
| A-08 | 移除公钥库后传 | 400/500；公钥库未配置；旧 current 不变；不泄漏密码 |
| A-09 | alias 错误 | 400；alias 未找到；不覆盖旧 current |
| A-10 | STOREPWD 错误 | 400；JKS integrity 失败；不泄漏密码 |
| A-11 | 验签失败 + 已有有效 current | current 保持旧；新文件不入 history；无 `invalid` 成 current |
| A-12 | 查 current `signature_status` | 合法=verified；不存在 invalid/unverified 成 current |

### B. 容量允许与边界拒绝

用 L-ONE/L-BOUNDARY，优先 `POST /api/v1/devices`。

| 编号 | 使用量+新增，max | 操作 | 预期 |
|---|---|---|---|
| B-01 | 0+1, max=1 | 创建唯一设备 | 成功；数量=1 |
| B-02 | 1+1, max=1 | 创建第二台 | 403 `ErrLicenseCapacityExceeded`；数量不变 |
| B-03 | 1+重复 SN | 再创第一台 | 设备已存在（非容量错）；语义稳定 |
| B-04 | max=2, 1+1 | 创建第二台 | 成功到边界 |
| B-05 | max=2, 2+1 | 创建第三台 | 403；数量保持 2 |
| B-06 | L-MULTI 多类型 | 创建到总容量 9 再创第 10 台 | 前 9 允许，第 10 拒绝；`quota.max_devices=9` |
| B-07 | 存量大量设备后传小容量 License | 上传 L-ONE 后创设备 | 按当前总数立即拒；不得靠换 License 绕过 |
| B-08 | 无 current License | 无 license 环境创设备 | 403 `license_unavailable`（详见 B-0） |

每个拒绝须同时确认：HTTP 403、设备数不变、日志/指标 `denied_capacity`/`audit=enforcement_capacity`。

### B-0. 完全没有 License

> 已实现 fail-closed：无 license 时受控业务 403 `ErrLicenseUnavailable`（metric `denied_no_license`）。

| 编号 | 操作 | 预期 |
|---|---|---|
| B-0-01 | `GET /system-license` | 404 / 12113（无当前 license） |
| B-0-02 | `feature-check?path=eNB.Monitor` | 404；不默认 authorized=true |
| B-0-03 | `POST /devices` 创建唯一设备 | 403 `ErrLicenseUnavailable`；数量不变；metric `denied_no_license`；不误记 denied_capacity/expired |
| B-0-04 | 连续创建多台 | 全 403；数量始终 0 |
| B-0-05 | 运行 CheckCapacity 周期 | 不发容量告警；license_unavailable 状态只读可见 |
| B-0-06 | 运行 CheckExpiringSoon | 不发到期告警 |
| B-0-07 | CPE Bootstrap Inform | 若自动注册创建设备，也应被拒（见 E-04） |
| B-0-08 | 打开 `/license` 页 | 空态；侧边栏只剩 License 菜单 |
| B-0-09 | 访问 `/dashboard`、`/device/list` | 回跳 `/license`；admin/超管也不能绕过 |

### C. 过期、永久、临界 + 累计时长 + 回拨

| 编号 | License | 操作 | 预期 |
|---|---|---|---|
| C-01 | L-LARGE expiry=NULL | 创建设备 | 成功；`days_remaining=-1` |
| C-02 | L-30D | 查 quota+运行 monitor | `days_remaining≈30`；`license_expiring_30d` warning |
| C-03 | L-7D | 运行 monitor | `license_expiring_7d` major |
| C-04 | L-1D | 运行 monitor | `license_expiring_1d` critical |
| C-05 | L-EXPIRED | 创建设备 | 403 `ErrLicenseExpired`；数量不变 |
| C-06 | L-EXPIRED | 读 current/history/设备/告警 | 只读接口仍可用 |
| C-07 | 到期前/后一秒 | 分别创建 | 前允许后拒绝；记录服务端 UTC 避免时区误判 |
| C-08 | 替换为未过期 License | 重创被拒设备 | Update 后 enforcer cache 立即失效，新 License 生效 |
| C-09 | L-CUMULATIVE (time_limit=2h) | 等累计 >2h 后创建设备（或 SQL 把 `system_license_usage.use_duration_hours` 调大） | 403 `ErrLicenseExpired`（累计超限）；日志 `denied_cumulative` |
| C-10 | L-ROLLBACK | 把系统时间往前调到 last_visited 之前，再创建设备 | 403 `ErrLicenseExpired`（时间回拨）；日志 `denied_rollback` |

> 累计/回拨当前只在 enforcement 点（device.create 等）推进/检查；菜单/页面不因累计超限受限（缺全局周期任务，见 §10 风险）。

### D. 功能授权与 Feature Check

> 已实现（选 B）：`Update`/读取 enrich 时从 feature code 前缀派生嵌套授权树 `feature_list.authorization_tree`，`CheckFeature` 解树后走 `HasFeature`。path 派生：`CODE_<MODULE>[_<FEATURE>]`→`Module[.Feature]`（`CODE_ENB_MONITOR`→`eNB.Monitor`、`CODE_ENB_UPGRADE_IMAGE`→`eNB.UpgradeImage`、`CODE_DASHBOARD`→`Dashboard`）。未识别 code 不入树（feature-check=false）。legacy license 无模块级 `"All"`，每个授权 feature 是显式叶子。

| 编号 | License | 操作 | 预期 |
|---|---|---|
| D-01 | 全量 feature | `feature-check?path=eNB.Monitor` | authorized=true |
| D-02 | L-FEATURE-PARTIAL | 查已授权叶子 | authorized=true |
| D-03 | L-FEATURE-PARTIAL | 查相邻未授权叶子 | authorized=false |
| D-04 | 任一 | 查不存在/空路径 | authorized=false 或 400；不默认 true |
| D-05 | L-FEATURE-UNKNOWN | 查未知 code/ID | authorized=false；不把数字 ID 当功能名 |
| D-06 | L-LARGE | 打开 `/license` Feature List | 中文名/菜单路径/分组与映射一致；无裸 ID |
| D-07 | L-LARGE | 切中英文刷新 | 名称随 locale 变；原始字段不变 |
| D-08 | 替换 License 后看 history | 当前与历史快照各自对应上传时授权 |

### E. CPE 模拟器接入

先在容量足够 License 下确认 ACS 基线，再把容量/过期断言放管理面创建链路。

| 编号 | 操作 | 预期 |
|---|---|---|
| E-01 | `cpe_simulator.py --event bootstrap --sn LIC-E2E-BOOT-001 --once` | 进程退出 0；Inform 报文正常；设备列表出现；计数与 quota 口径一致 |
| E-02 | `--count 5 --base-sn LIC-E2E-MULTI-` | 5 个唯一 SN 独立落库；无重复/串线 |
| E-03 | `--event periodic --interval 30`（timeout 150s） | ≥3 次周期 Inform；设备仍 1 条；计数不增长 |
| E-04 | L-ONE(1 台) 下先 E-01 接 1 台，再 E-02 观察 | 探针：CPE Inform 是否仍自动注册超额（=绕过 enforcer，登记缺陷或明确策略）；同时 `POST /devices` 第 2 台确认 403 |

CPE 命令见 recap §B-3。

### F. 菜单按 license 显隐（P7-A）

> 已实现：`menus.feature_code text[]` + 后端 `GetUserMenuTree`/`GetUserMenuTreeByRole` 按 license 过滤；无 license→只剩 `/license`；超管也受控；菜单↔feature 全量映射见 `seed/000001_init_seed.sql`。

| 编号 | 场景 | 预期 |
|---|---|---|
| F-01 | License 授权 CODE_ENB_MONITOR | `/device/list` 可见；直访不被守卫拦 |
| F-02 | 未授权 CODE_ALARM_VIEW | `/alarm/current` 不渲染；直访→`/403` |
| F-03 | 超管+未授权某 feature | 该菜单对超管也隐藏 |
| F-04 | 无 License | 后端菜单只剩 `/license`；前端 licenseOnlyMode 重定向非 `/license` 路由 |
| F-05 | 替换 License 撤销某 feature | 刷新/重登后该菜单消失 |
| F-06 | `/device/list`(多 feature OR) 只授权 CPE monitor | 菜单仍可见 |
| F-07 | 系统管理菜单 | 按 `CODE_SYSTEM_*` 受控；仅 `/license` 始终可见 |
| F-08 | `feature-check?path=eNB.Monitor` 与 `/device/list` 可见性 | 结论一致（同源） |
| F-09 | 目录可见性 | 子菜单全未授权→父目录隐藏；至少一个授权→显示 |

F 组须用**非超管账号**测（超管在 RBAC 层 bypass 路由守卫，但 P7-A 菜单过滤对超管也生效，F-03 专验）。

### G. 硬件绑定（MAC / UUID）

> 已实现：上传时 `validateHardwareBinding`，license MAC/UUID 非空且与本机不匹配→拒绝上传 403 `ErrLicenseHardwareMismatch`；为空=不绑定。

| 编号 | License | 操作 | 预期 |
|---|---|---|---|
| G-01 | L-MAC-BIND（MAC=本机 eno1） | 上传 | 201；成为 current |
| G-02 | L-MAC-MISMATCH（MAC=不存在） | 上传 | 403 `ErrLicenseHardwareMismatch`；不成为 current；旧 current 不变 |
| G-03 | L-UUID-BIND（UUID=本机） | 上传 | 201 |
| G-04 | MAC 多值（含本机一个） | 上传 | 201（任一匹配即通过） |
| G-05 | license MAC/UUID 为空 | 上传 | 201（不绑定，任意硬件放行） |

取证：上传响应、`SELECT mac... from system_license`（不存 MAC/UUID 到库？实际不落库，只校验）、app 日志硬件校验记录、容器内 `/host/sys/class/net/eno1/address` 与 license MAC 归一化对比。

### H. 累计使用时长 + 时间回拨

> 已实现：`system_license_usage` 表 + enforcer compute-on-read；`time_limit_hours` 存 feature_list。

| 编号 | 场景 | 预期 |
|---|---|---|
| H-01 | L-CUMULATIVE(time_limit=2h)，正常使用（累计<2h） | 创建设备成功 |
| H-02 | 累计推进到 ≥2h（SQL 调大 `use_duration_hours` 或等待） | 创建设备 403 `ErrLicenseExpired`；日志 `denied_cumulative` |
| H-03 | 系统时间往前拨到 last_visited 之前 | 创建设备 403 `ErrLicenseExpired`；日志 `denied_rollback` |
| H-04 | 持久化错误（usage 表不可用） | fail-open（仅告警），不锁死全部业务 |
| H-05 | perpetual(expiry=NULL) + time_limit>0 | 日期不过期但累计/回拨仍生效 |

## 6. 监控与可观测性

### 6.1 日志断言
- `audit=enforcement_capacity`、`result=denied_capacity`、`used_devices`、`total_capacity`
- `audit=enforcement_expiry`、`result=denied_expired` / `denied_cumulative` / `denied_rollback`
- `result=denied_no_license`（无 license）
- `audit=capacity_alert`/`expiry_alert` + identifier
- 硬件失配上传日志
- 不得出现 STOREPWD / 完整 License Base64 / 私钥 / Token

### 6.2 指标断言
| 指标 | 关键标签 |
|---|---|
| `license_enforcement_total` | `result=allowed` / `denied_capacity` / `denied_expired` / `denied_no_license` / `denied_cumulative` / `denied_rollback` |
| `license_capacity_used_devices` | 与设备查询数量一致 |
| `license_capacity_max_devices` | 与 `quota.max_devices` 一致 |
| `license_capacity_usage_ratio` | `used/max`；无 license 时 max=0、受控业务 fail-closed |
| `license_expiry_days_remaining` | 永久=-1 |

## 7. API 请求模板（端口 18081）

内部调测可用 `.api-key`（`-H "X-API-Key: $KEY"`）替代 Bearer Token。

```bash
# 上传 .lic
RAW=$(base64 -w 0 L-ONE.lic)
curl -X POST http://localhost:18081/api/v1/system-license -H "X-API-Key: $KEY" \
  -H 'Content-Type: application/json' -d "{\"raw_content\":\"$RAW\",\"raw_content_encoding\":\"base64\"}"

# 查 current / history
curl -H "X-API-Key: $KEY" http://localhost:18081/api/v1/system-license | python3 -m json.tool
curl -H "X-API-Key: $KEY" 'http://localhost:18081/api/v1/system-license/history?page=1&page_size=50'

# feature-check
curl -G -H "X-API-Key: $KEY" http://localhost:18081/api/v1/system-license/feature-check --data-urlencode 'path=eNB.Monitor'

# 创建设备（字段以 CreateDeviceRequest 为准，别把字段校验失败误判为 license 拒绝）
# ⚠️ oui 是 6 位 hex（varchar(6)），如 48BF74；manufacturer 才填 Baicells。
curl -X POST http://localhost:18081/api/v1/devices -H "X-API-Key: $KEY" -H 'Content-Type: application/json' \
  -d '{"serial_number":"LIC-E2E-ONE-1","oui":"48BF74","product_class":"Femto-Classic","manufacturer":"Baicells","carrier":"cmcc","technology":"lte"}'
```

## 8. 测试执行顺序

1. 环境健康 + 快速 Go 回归 + 启动日志（mapping/decoder）。
2. A 组（导入真实 `.lic`），再 B/C/D/G/H；先清旧 License 再比容量。
3. E-01/E-02 确认 ACS 基线 → B 容量边界 → E-04 探 CPE Inform 绕过。
4. C 过期/累计/回拨（必要时直接调 monitor / SQL 调时间，避免真等）。
5. D + F（API + 浏览器 Feature List / 菜单）。
6. 采集 API 响应/设备数/日志/指标/CPE 报文。
7. 清理测试数据（不误删非测试设备/真实 License）。

## 9. 通过标准

- A~H 各组必测用例有明确 PASS/FAIL。
- 容量/过期/累计/回拨/无 license/硬件失配拒绝都能区分（错误码 + 日志），不都归 500。
- 永久 / 无 license / 零容量 / 绑定 / 累计 语义有实测结论；与旧项目不一致登记缺陷，不在脚本里默默放宽。
- feature-check 与菜单显隐对同一路径结论一致；未知功能不授权。
- CPE 多设备 Inform 不产生重复设备；周期 Inform 不增计数。
- 日志/报文/导出无 STOREPWD / 私钥 / Token / 完整敏感 license。

## 10. 当前风险与后续

| 风险 | 状态/建议 |
|---|---|
| CPE Inform 自动注册绕过 enforcer | 待产品决策；要全链路拦截需接 `RegisterFromInform`（E-04） |
| 容量是各类型配额之和，无 per-type gating | 增 `device_type` 维度 enforcer + 独立计数（Phase 7 RBAC） |
| monitor 用真实 cron，30/7/1 天慢 | 注入 clock / 直调 monitor 检查方法 |
| 测试签发工具不在仓库 | 受控外部签发环境生成 fixture |
| 前端 typecheck 受既有依赖影响 | 浏览器验收与 Docker Vite 构建分别记录 |
| enforcer 无跨实例失效（单实例 5min 内存缓存） | 横扩前补 Redis 通知 / PG advisory lock |
| **累计时长缺全局周期推进** | 当前仅 enforcement 点推进；菜单/页面不因累计超限受限。要全局 isExpired 需加周期任务 |
| P7-B（API middleware 按 feature 拦截）未做 | 前端隐藏≠后端拦；新增 `RequireFeature` 中间件挂 `permGroup` |
| dev 库落后基线（provision/param-sync/storage-protection 等缺失） | 重置 dev 库 或 封版本后增量迁移补齐 |

## 11. 清理与证据归档

归档到不入 Git 目录：`/tmp/omc-license-e2e/<timestamp>/{api,cpe-logs,app.log,acs.log,metrics.txt,sample-index.txt,result.md}`。清理前保存样本 SHA-256、current/history 的 ID/容量/功能数、各拒绝的 HTTP 状态+业务码+时间、CPE 每 SN 的 Inform 次数、告警字段。清理后复查 `LIC-E2E-*` 设备不存在、测试 License 不再 current、不影响非测试数据。

## 12. 关联资料

- 速查/前情/快速验证：[`license-refactor-recap.md`](./license-refactor-recap.md)
- `docs/project/license-replication.md`、`docs/project/prd/F06-system-license-redesign.md`
- `omcgo/internal/license/{enforcer,monitor,hardware,pg_system_license_usage,legacy_feature_mapping}.go`、`omcgo/internal/admin/service.go`、`omcgo/internal/device/device_service.go`
- `omcgo/scripts/cpe_simulator.py`、`omcgo/data/license-feature-mapping.json`、`migrations/seed/000001_init_seed.sql`
