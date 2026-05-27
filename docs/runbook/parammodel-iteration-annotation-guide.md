# paramModel XML 迭代标注 supported="false" 操作指南

> 给后续 AI 会话使用：当某 paramModel 字典（如 BM.xml / BSC.xml / BaiBNQ.xml / MLN.xml / MLQ.xml / ENB_DEFAULT_098.xml / ENB_DEFAULT_181.xml / BTS.xml）出现 9005 fault 时，按本文档迭代标注 supported="false"，直至 path-b sync 全 batch 0 fault 收敛。
>
> **前提**：T-0105 Content-Length fix 已部署（commit `b1a59881`），单 sync 在 ~1.5s 内完成；T-0103 框架已就绪（commit `a89151b9`）。

---

## 0. T-0183 后的推荐路径(自动化优先)

T-0183 起 `omcctl device sweep-paths --apply` 在 omcctl 进程内一步完成:
1. GPV 探测设备 → 收集 9005 unsupported 集合
2. 写 `data/param-mappings/<paramModel>.xml` 标注 `supported="false"`(text-based 保格式)
3. UPDATE `param_mappings.is_supported=false` + `discovered_param_mappings.is_supported=false`
4. 清 Redis L2 + bump `parammodel:cache_version`

**优先用此路径**:
```bash
docker exec docker-worker-1 omcctl device sweep-paths "$SN" --apply
git -C /Users/cb/code/baicells/goomc diff omcgo/data/param-mappings/   # 审阅
git add omcgo/data/param-mappings/ && git commit -m "..."
```

XML 文件通过 docker-compose bind mount(`omcgo/data/param-mappings` → `/etc/omcgo/data/param-mappings`)
直接反映到宿主源码,容器重建 / 重启都不会丢标注。

**何时仍用本指南手工流程**:
- 9005 fault 指向对象前缀(如 `Device.X.Y.{i}.Z.`,带末尾点),需要 sed 批量标整个子树
- sweep-paths 探测覆盖不到的角落(如对象级 GPV,而非叶子)
- 多 batch 多 sample 收敛(每轮只暴露 1 个,多轮迭代)

下面的手工流程仍然有效,只是大多数单 leaf 场景已被 sweep-paths 自动化吸收。

---

---

## 1. 准备工作

### 1.1 确认 paramModel 已绑定产品

```sql
-- 查 product 与 paramModel 关联
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT p.name AS product, pm.name AS param_model, p.id, pm.id
FROM products p JOIN param_models pm ON p.param_model_id = pm.id
WHERE pm.name='BM';"   -- 改成目标 paramModel 名
```

### 1.2 确认有实测设备在线

```bash
# 找一台 productClass 路由命中该 paramModel 的真机
grep '"device_sn"' /Users/shangyingbin/project/goomc/run/logs/acs/acs.log | tail -3

# 也可以查 DB
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT serial_number, product_class, last_inform_at
FROM devices WHERE product_id='<product-uuid>' AND deleted_at IS NULL
ORDER BY last_inform_at DESC LIMIT 5;"
```

### 1.3 路径变量

```bash
GOOMC=/Users/shangyingbin/project/goomc/omcgo
XML=$GOOMC/data/param-mappings/<TARGET>.xml   # 例如 BM.xml
SN=<basestation-serial-number>                 # 例如 1202000240194DP0026
```

---

## 2. 单轮迭代流程

每轮 ~10 秒（不含基站 inform 等待）。

### 2.1 标 supported="false"

**单叶子参数**（一行 `<param name="...">`）：直接 Edit 工具改：
```xml
<param name="Device.X.Y" ... changeApplies="Immediate"/>
<!-- 改为 -->
<param name="Device.X.Y" ... changeApplies="Immediate" supported="false"/>
```

**整个对象子树**（多个 `Device.X.Y.{i}.Z.*` 叶子，CPE 报 `Device.X.Y.{i}.Z.` 前缀 fault）：用 sed 批量：
```bash
sed -i.bak 's|<param name="Device.X.Y.{i}.Z\.\([^"]*\)" \(.*\)/>|<param name="Device.X.Y.{i}.Z.\1" \2 supported="false"/>|g' $XML
rm $XML.bak
grep -c 'supported="false"' $XML   # 验证总数变化
```

**注意**：
- 对象前缀 fault（末尾带点的 path）说明整个子树不支持，标整树
- 单叶 fault（不带末尾点）只标该叶

### 2.2 部署 + 触发新 sync

**T-0183 后简化路径**(XML 在宿主 bind mount 内,无需 docker cp;reload 端点自动清缓存):

```bash
# 1. 触发后端 dictload 重读 XML(自动清 Redis L2 + bump cache_version)
curl -H "X-API-Key: $(cat /var/lib/omcgo/secrets/.api-key)" \
     -X POST http://app:8081/api/v1/admin/dictload/reload?name=param-model

# 2. 软删 device 触发 fresh sync
docker exec docker-postgres-1 psql -U omcgo -d omcgo -c "
UPDATE devices SET deleted_at=now() WHERE serial_number='$SN' AND deleted_at IS NULL RETURNING id;"
```

或通过 UI:进 `/system/dict-loader` 页面,点击"参数模型字典"的"重新加载"按钮。

---

**历史手工路径**(T-0183 前)— 仅供回滚或非容器化环境参考:

```bash
# 1. XML → 运行容器(T-0183 前需要,现在 bind mount 自动同步)
docker cp $XML omc-docker-app-1:/etc/omcgo/data/param-mappings/$(basename $XML)

# 2. 清 Redis L2 缓存(T-0183 后 reload 端点自动做)
docker exec omc-docker-redis-1 redis-cli --scan --pattern "parammodel:default:*" | xargs -I {} docker exec omc-docker-redis-1 redis-cli DEL {}
docker exec omc-docker-redis-1 redis-cli --scan --pattern "parammodel:discovered:*" | xargs -I {} docker exec omc-docker-redis-1 redis-cli DEL {}

# 3. 重启 app(T-0183 后 reload 端点替代,不需要全 app 重启)
cd /Users/shangyingbin/project/omc-docker && docker compose restart app
```

### 2.3 等基站 inform + 抓 fault sample

基站 PERIODIC inform 间隔通常 1 分钟。等下次 inform → stale cache → auto-register → ProvisioningEngine → path-b sync → 链式 8 batch 在 ~1.5s 内全部跑完。

```bash
# 监听新 fault
tail -F /Users/shangyingbin/project/goomc/run/logs/acs/acs.log | grep --line-buffered 'SOAP Fault'
```

抓本轮所有 fault sample（一次 sync 可能暴露多个）：

```bash
# 查 §2.2 重启后到现在的所有 fault
grep "SOAP Fault" /Users/shangyingbin/project/goomc/run/logs/acs/acs.log | \
  awk -F'"timestamp":"' '$2 >= "2026-MM-DDTHH:MM:00"' | \
  awk -F'"fault_msg":"' '{rest=$2; sub(/".*/,"",rest); print rest}'
```

### 2.4 验证本轮 batch 结果

```sql
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
SELECT status, count(*) FROM device_tasks
WHERE device_sn='$SN' AND created_at > 'YYYY-MM-DD HH:MM:00'
GROUP BY status;"
```

期望：完全收敛时 `completed=8`，`failed=0`，`sent=0`。

---

## 3. 收敛判断

- **continue**: `failed > 0` 或有新 fault sample → 标记新 sample，跳回 §2.1
- **converged**: `failed=0` 且 `sent=0` 且本轮全程无 9005 fault → 收敛达成

收敛后：把 paramModel.xml 的 supported="false" 标注 + 实测数据汇报到 modification.md，提交 commit。

---

## 4. 关于 sample 解读

BAICELLS（也代表其它常见厂商）9005 fault 只回一个 sample，格式：

```
Invalid Parameter Names [N], including: <PATH>
```

- `[N]`：本 batch invalid 总数（仅参考）
- `<PATH>`：第一个 invalid path

**结尾带点**（如 `Device.X.Y.`） → 该路径是**对象前缀**，整个子树不支持，标整树（§2.1 sed 命令）。
**结尾不带点**（如 `Device.X.Y.Z`） → 该路径是**叶子参数**，只标这一个。

⚠️ 单 batch 内可能有 N 个 invalid path，但 fault 只暴露 1 个。所以一轮 sync 可能多 batch 各 1 fault → 多 sample；标完这些重跑下一轮还会暴露之前被遮蔽的 sample。这是正常收敛过程。

---

## 5. 关键代码 / 数据点

| 路径 | 说明 |
|------|------|
| `omcgo/data/param-mappings/*.xml` | 各 paramModel 字典源 |
| `omcgo/internal/config/parammodel/model.go` | `xmlParamEntry.Supported` 解析 |
| `omcgo/internal/config/parammodel/loader.go` | 写 DB `param_mappings.is_supported` 列 |
| `omcgo/internal/provision/sync_pathb.go` | `extractStorablePrefixes` 过滤 `IsStorable && IsSupported` |
| `omcgo/internal/acs/handler.go::sendSOAPResponse` | T-0105 Content-Length fix |
| `omcgo/migrations/000090_param_mappings_is_supported.sql` | 加 `is_supported` 列（DEFAULT TRUE） |

---

## 6. BLQ.xml 收敛案例（参考）

针对 BAICELLS BaiBLQ_5.0.16.1_1229 固件，10 轮迭代收敛到 85 个 supported="false" 标注（18 个 invalid path / 路径组）。详见 `~/Documents/notes/modification.md` 中"T-0105 ACS sendSOAPResponse Content-Length 修复 + iter#6-#10 BLQ.xml 收敛"章节。

收敛后业务指标：
- `device_parameters` 行数：240 → 9148（+3712%）
- 单 sync 完成时长：卡死 → 1.5 秒
- 9005 fault per sync：多 → 0

---

## 7. 单 paramModel 收敛后

1. **commit**：`feat(parammodel): <PARAM_MODEL> XML 标注 N 个 BAICELLS/华为/中兴 等固件不支持 path`
2. **更新 modification.md**：加一节"<PARAM_MODEL>.xml 迭代收敛实测"，含 iter 表 + 最终 supported="false" 标注清单
3. **更新 pending-issues.md**：在"其它 paramModel 迭代标注"清单划掉该项

---

## 附录 A：完整工作流脚本骨架

```bash
#!/bin/bash
# 自动化单轮迭代：编辑 XML 后调用本脚本
GOOMC=/Users/shangyingbin/project/goomc/omcgo
TARGET=${1:-BLQ.xml}
SN=${2:-1202000240194DP0026}
XML=$GOOMC/data/param-mappings/$TARGET

# 部署
docker cp $XML omc-docker-app-1:/etc/omcgo/data/param-mappings/$TARGET
docker exec omc-docker-redis-1 redis-cli --scan --pattern "parammodel:default:*" | \
  xargs -I {} docker exec omc-docker-redis-1 redis-cli DEL {}
docker exec omc-docker-redis-1 redis-cli --scan --pattern "parammodel:discovered:*" | \
  xargs -I {} docker exec omc-docker-redis-1 redis-cli DEL {}
cd /Users/shangyingbin/project/omc-docker && docker compose restart app

# 等 app 就绪
until docker logs --since 30s omc-docker-app-1 2>&1 | grep -q "routes registered"; do sleep 2; done

# 触发新 sync
docker exec omc-docker-postgres-1 psql -U omcgo -d omcgo -c "
UPDATE devices SET deleted_at=now() WHERE serial_number='$SN' AND deleted_at IS NULL RETURNING id;"

echo "iteration triggered, watch logs:"
echo "  tail -F /Users/shangyingbin/project/goomc/run/logs/acs/acs.log | grep 'SOAP Fault'"
```
