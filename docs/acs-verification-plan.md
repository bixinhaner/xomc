# ACS 端到端验证方案

> 日期：2026-03-22
> 目标：验证 omcgo-acs 与 oam_tr069_worker（旧 CPE 模拟器）之间的完整 TR069 会话流程

---

## 0. 当前状态诊断

### 已确认正常

| 项目 | 状态 | 证据 |
|------|------|------|
| ACS 进程 | 运行中 | `omcgo-acs` PID 12378, 端口 8080 |
| App 进程 | 运行中 | `omcgo-app` PID 12346, 端口 8081 |
| Worker 进程 | 运行中 | `omcgo-worker` PID 12397 |
| CPE 连接 | 已建立 | TCP [::1]:52403 → [::1]:8080 |
| Inform 请求 | 200 OK | POST /smallcell/AcsService → InformResponse |
| 设备注册 | 已完成 | DB: serial=1202000588233HB0039, status=active |
| 会话关闭 | 正常 | 空请求 → 204 No Content → session closed |

### 发现的问题（需修复后再验证）

| # | 问题 | 严重度 | 详情 |
|---|------|--------|------|
| P1 | **device_tasks 表不存在** | 高 | 当前迁移版本=34，device_tasks 在 migration 48。Task API 返回 500，测试任务注入静默失败 |
| P2 | **Redis 心跳已过期** | 中 | `acs:heartbeat:1202000588233HB0039` TTL=-2（key 不存在）。Periodic Inform 未更新心跳 |
| P3 | **last_inform_at 未更新** | 中 | DB 中 last_inform_at 停留在 15:23:55（首次 BOOTSTRAP），后续 PERIODIC Inform 未刷新 |
| P4 | **last_inform_events 始终为 BOOTSTRAP** | 低 | 可能是 CPE 侧问题——旧 worker 每次都发 BOOTSTRAP 或 AUTONOMOUS TRANSFER COMPLETE |
| P5 | **RPC 任务未执行** | 高 | 每次 Inform 后只有 204 空响应，无 RPC 下发。因 device_tasks 表缺失 + 命令队列为空 |

---

## 1. 前置修复

### 1.1 执行 database migration 到最新版本

```bash
cd omcgo
# 检查当前版本
PGPASSWORD=omcgo123 psql -h localhost -U omcgo -d omcgo -c "SELECT * FROM schema_migrations;"

# 执行迁移到最新（包含 000048_create_device_tasks）
go run cmd/migrate/main.go -config cmd/app/etc/config.dev.yaml -direction up

# 验证 device_tasks 表已创建
PGPASSWORD=omcgo123 psql -h localhost -U omcgo -d omcgo -c "\d device_tasks"
```

### 1.2 重启三个服务

```bash
cd omcgo
# 使用一键重启脚本（如果有），否则手动重启
scripts/dev-restart.sh

# 或手动重启
kill $(pgrep omcgo-acs) $(pgrep omcgo-app) $(pgrep omcgo-worker)
sleep 2
bin/omcgo-app --config cmd/app/etc/config.dev.yaml &
bin/omcgo-acs --config cmd/acs/etc/config.dev.yaml &
bin/omcgo-worker --config cmd/worker/etc/config.dev.yaml &
```

### 1.3 验证服务恢复

```bash
# ACS 健康检查
curl -s http://localhost:8080/healthz
# 期望: ok

# App 登录
curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
# 期望: 返回 access_token
```

---

## 2. 验证矩阵

### 阶段一：Inform 事件处理验证

> 目标：确认 ACS 正确处理各类 Inform 事件，设备状态正确更新

| 测试编号 | 测试项 | 验证方法 | 预期结果 |
|----------|--------|----------|----------|
| T01 | Periodic Inform 处理 | 等待 60s 后检查 DB/Redis | last_inform_at 更新，Redis 心跳刷新 |
| T02 | 心跳超时检测 | 停止 CPE，等 2×interval | 设备 status 变为 offline |
| T03 | 重新上线 | 重启 CPE | status 恢复 active/online |
| T04 | Bootstrap 事件 | 清空 Redis+DB 设备记录，重启 CPE | 设备重新注册 |
| T05 | 并发 Inform | 记录多个 Inform 的 Session ID | 每个 Inform 独立 Session |

#### T01: Periodic Inform 验证脚本

```bash
#!/bin/bash
# t01_periodic_inform.sh
# 验证 Periodic Inform 是否正确更新设备状态

DEVICE_SN="1202000588233HB0039"
PSQL="/usr/local/Cellar/postgresql@16/16.13/bin/psql"
DB_CONN="postgresql://omcgo:omcgo123@localhost:5432/omcgo"

echo "=== T01: Periodic Inform 验证 ==="
echo ""

# 1. 记录当前 last_inform_at
echo "[Step 1] 记录当前 last_inform_at..."
BEFORE=$($PSQL "$DB_CONN" -t -c "SELECT last_inform_at FROM devices WHERE serial_number='$DEVICE_SN';")
echo "  Before: $BEFORE"

# 2. 记录当前 Redis 心跳
echo "[Step 2] 检查 Redis 心跳..."
HB_BEFORE=$(redis-cli GET "acs:heartbeat:$DEVICE_SN")
HB_TTL=$(redis-cli TTL "acs:heartbeat:$DEVICE_SN")
echo "  Heartbeat value: $HB_BEFORE (TTL: ${HB_TTL}s)"

# 3. 等待一个 Inform 周期 (60s + 5s buffer)
echo "[Step 3] 等待 65 秒（1个 Periodic Inform 周期）..."
sleep 65

# 4. 检查 last_inform_at 是否更新
echo "[Step 4] 检查 last_inform_at 是否更新..."
AFTER=$($PSQL "$DB_CONN" -t -c "SELECT last_inform_at FROM devices WHERE serial_number='$DEVICE_SN';")
echo "  After: $AFTER"

if [ "$BEFORE" != "$AFTER" ]; then
    echo "  ✅ PASS: last_inform_at 已更新"
else
    echo "  ❌ FAIL: last_inform_at 未变化"
fi

# 5. 检查 Redis 心跳是否更新
echo "[Step 5] 检查 Redis 心跳..."
HB_AFTER=$(redis-cli GET "acs:heartbeat:$DEVICE_SN")
HB_TTL_AFTER=$(redis-cli TTL "acs:heartbeat:$DEVICE_SN")
echo "  Heartbeat value: $HB_AFTER (TTL: ${HB_TTL_AFTER}s)"

if [ -n "$HB_AFTER" ] && [ "$HB_TTL_AFTER" -gt 0 ]; then
    echo "  ✅ PASS: Redis 心跳存在且有 TTL"
else
    echo "  ❌ FAIL: Redis 心跳丢失或无 TTL"
fi

# 6. 检查 TR069Worker 日志
echo ""
echo "[Step 6] 最近 Inform 日志:"
tail -200 /Users/watermelon/Documents/code/oam-c/oaim/targetoxm/logs/TR069Worker.log \
  | grep -E "Inform sent successfully|Status: (200|204)" | tail -5
```

#### T02: 心跳超时检测脚本

```bash
#!/bin/bash
# t02_heartbeat_timeout.sh
# 验证设备离线检测

DEVICE_SN="1202000588233HB0039"
PSQL="/usr/local/Cellar/postgresql@16/16.13/bin/psql"
DB_CONN="postgresql://omcgo:omcgo123@localhost:5432/omcgo"
OAM_DIR="/Users/watermelon/Documents/code/oam-c/oaim/targetoxm"

echo "=== T02: 心跳超时检测 ==="

# 1. 确认设备当前在线
STATUS=$($PSQL "$DB_CONN" -t -c "SELECT status FROM devices WHERE serial_number='$DEVICE_SN';")
echo "[Step 1] 当前状态: $STATUS"

# 2. 停止 CPE
echo "[Step 2] 停止 oam_tr069_worker..."
kill $(pgrep oam_tr069_worker)
echo "  已停止"

# 3. 等待超时（2 × inform_interval = 2 × 300 = 600s，但实际 inform_interval=60s）
# 按 ACS 默认配置 TTL 可能是 5min
echo "[Step 3] 等待 5 分钟（等待心跳超时）..."
echo "  提示：可用 redis-cli TTL acs:heartbeat:$DEVICE_SN 观察倒计时"
sleep 330

# 4. 检查状态
STATUS_AFTER=$($PSQL "$DB_CONN" -t -c "SELECT status FROM devices WHERE serial_number='$DEVICE_SN';")
echo "[Step 4] 超时后状态: $STATUS_AFTER"

if echo "$STATUS_AFTER" | grep -q "offline"; then
    echo "  ✅ PASS: 设备已标记为 offline"
else
    echo "  ❌ FAIL: 设备状态未变为 offline (当前: $STATUS_AFTER)"
fi

# 5. 重启 CPE
echo "[Step 5] 重启 oam_tr069_worker..."
cd "$OAM_DIR" && bin/oam_tr069_worker --log-level trace &
sleep 10

# 6. 检查重新上线
STATUS_BACK=$($PSQL "$DB_CONN" -t -c "SELECT status FROM devices WHERE serial_number='$DEVICE_SN';")
echo "[Step 6] 重启后状态: $STATUS_BACK"

if echo "$STATUS_BACK" | grep -qE "active|online"; then
    echo "  ✅ PASS: 设备已重新上线"
else
    echo "  ❌ FAIL: 设备未恢复在线 (当前: $STATUS_BACK)"
fi
```

---

### 阶段二：RPC 增删改查验证

> 目标：通过 REST API 创建 RPC 任务，验证 ACS 在下一次 Inform 时正确下发给 CPE

**前提：device_tasks 表已创建（见前置修复 1.1）**

#### 获取 Token（所有测试共用）

```bash
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")
echo "Token: ${TOKEN:0:20}..."
```

| 测试编号 | RPC 方法 | 验证要点 |
|----------|----------|----------|
| T10 | GetParameterValues | 读取设备参数，检查返回值 |
| T11 | SetParameterValues | 写入参数，检查 CPE 回复 status=0 |
| T12 | GetParameterNames | 枚举参数树，验证返回路径列表 |
| T13 | AddObject | 创建对象实例，返回 InstanceNumber |
| T14 | DeleteObject | 删除对象实例，返回 status=0 |
| T15 | Download | 下发固件下载，检查 TransferComplete |
| T16 | Upload (PM) | 触发 PM 文件上传，验证 MinIO 收到文件 |
| T17 | Reboot | 重启设备，验证 M Reboot 事件 |
| T18 | 批量任务 | 同时创建 5 个任务，验证按优先级执行 |
| T19 | 任务取消 | 创建后立即取消，验证未被执行 |
| T20 | 任务过期 | 创建极短 TTL 任务，验证自动过期 |

#### T10: GetParameterValues

```bash
#!/bin/bash
# t10_get_parameter_values.sh

DEVICE_SN="1202000588233HB0039"

# 获取 Token
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T10: GetParameterValues ==="

# 1. 创建 GetParameterValues 任务
echo "[Step 1] 创建 GetParameterValues 任务..."
RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GetParameterValues",
    "params": {
      "names": [
        "Device.DeviceInfo.SoftwareVersion",
        "Device.DeviceInfo.HardwareVersion",
        "Device.DeviceInfo.Manufacturer",
        "Device.DeviceInfo.SerialNumber"
      ]
    },
    "priority": 5,
    "max_retries": 3,
    "description": "T10: 读取设备基本信息"
  }')
echo "  Response: $RESULT"

TASK_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
echo "  Task ID: $TASK_ID"

# 2. 等待下一次 Inform（最长 60s）
echo "[Step 2] 等待 CPE 下次 Inform（最长 70s）..."
for i in $(seq 1 14); do
    sleep 5
    STATUS=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
      -H "Authorization: Bearer $TOKEN" \
      | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
    echo "  [${i}x5s] Task status: $STATUS"
    if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
        break
    fi
done

# 3. 检查任务结果
echo "[Step 3] 查询任务详情..."
curl -s "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

if [ "$STATUS" = "completed" ]; then
    echo "  ✅ PASS: GetParameterValues 执行成功"
else
    echo "  ❌ FAIL: 任务状态为 $STATUS"
fi
```

#### T11: SetParameterValues

```bash
#!/bin/bash
# t11_set_parameter_values.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T11: SetParameterValues ==="

# 1. 创建 Set 任务
echo "[Step 1] 创建 SetParameterValues 任务..."
RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "SetParameterValues",
    "params": {
      "values": [
        {
          "name": "Device.ManagementServer.PeriodicInformInterval",
          "value": "120",
          "type": "xsd:unsignedInt"
        }
      ]
    },
    "priority": 5,
    "description": "T11: 设置 Periodic Inform 间隔为 120 秒"
  }')
echo "  $RESULT"

TASK_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

# 2. 轮询任务状态
echo "[Step 2] 等待执行..."
for i in $(seq 1 14); do
    sleep 5
    STATUS=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
      -H "Authorization: Bearer $TOKEN" \
      | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
    echo "  [${i}x5s] Status: $STATUS"
    if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then break; fi
done

# 3. 验证：创建 Get 任务读回
echo "[Step 3] 创建 GetParameterValues 验证任务..."
VERIFY=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GetParameterValues",
    "params": {"names": ["Device.ManagementServer.PeriodicInformInterval"]},
    "priority": 1,
    "description": "T11-Verify: 读回 PeriodicInformInterval"
  }')

VERIFY_ID=$(echo "$VERIFY" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

for i in $(seq 1 14); do
    sleep 5
    RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$VERIFY_ID" \
      -H "Authorization: Bearer $TOKEN")
    VSTATUS=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
    if [ "$VSTATUS" = "completed" ]; then
        echo "  Verify result:"
        echo "$RESULT" | python3 -m json.tool
        echo "  ✅ PASS: SetParameterValues + 读回验证完成"
        break
    fi
done
```

#### T12: GetParameterNames

```bash
#!/bin/bash
# t12_get_parameter_names.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T12: GetParameterNames ==="

RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GetParameterNames",
    "params": {
      "path": "Device.DeviceInfo.",
      "next_level": true
    },
    "priority": 5,
    "description": "T12: 枚举 DeviceInfo 下一级参数名"
  }')

TASK_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
echo "Task ID: $TASK_ID"

for i in $(seq 1 14); do
    sleep 5
    DETAIL=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
      -H "Authorization: Bearer $TOKEN")
    STATUS=$(echo "$DETAIL" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
    echo "  [${i}x5s] Status: $STATUS"
    if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then
        echo "$DETAIL" | python3 -m json.tool
        break
    fi
done
```

#### T13-T14: AddObject / DeleteObject

```bash
#!/bin/bash
# t13_t14_add_delete_object.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T13: AddObject ==="

# 添加对象
ADD_RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "AddObject",
    "params": {"object_name": "Device.Services.FAPService."},
    "priority": 5,
    "description": "T13: 添加 FAPService 实例"
  }')
echo "  $ADD_RESULT"

ADD_ID=$(echo "$ADD_RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

# 等待结果
for i in $(seq 1 14); do
    sleep 5
    STATUS=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$ADD_ID" \
      -H "Authorization: Bearer $TOKEN" \
      | python3 -c "import sys,json; d=json.load(sys.stdin).get('data',{}); print(d.get('status',''), d.get('result',{}))" 2>/dev/null)
    echo "  [${i}x5s] $STATUS"
    if echo "$STATUS" | grep -qE "completed|failed"; then break; fi
done

echo ""
echo "=== T14: DeleteObject ==="

# 假设 AddObject 返回的 InstanceNumber 用于删除
# 实际中需要从 T13 的结果中提取 InstanceNumber
DEL_RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "DeleteObject",
    "params": {"object_name": "Device.Services.FAPService.99."},
    "priority": 5,
    "description": "T14: 删除 FAPService 实例 99"
  }')
echo "  $DEL_RESULT"
```

#### T16: Upload (PM 文件)

```bash
#!/bin/bash
# t16_upload_pm.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T16: Upload PM 文件 ==="

# 1. 创建 Upload 任务（ACS 会让 CPE 上传 PM 文件到 FileUploadService）
RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "Upload",
    "params": {
      "file_type": "2 Vendor Configuration File",
      "url": "http://localhost:8080/smallcell/FileUploadService?fileType=PM&filename='"$DEVICE_SN"'_test.xml.gz"
    },
    "priority": 5,
    "description": "T16: 触发 PM 文件上传"
  }')
echo "  $RESULT"

TASK_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

# 2. 等待执行
for i in $(seq 1 14); do
    sleep 5
    STATUS=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
      -H "Authorization: Bearer $TOKEN" \
      | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
    echo "  [${i}x5s] Status: $STATUS"
    if [ "$STATUS" = "completed" ] || [ "$STATUS" = "failed" ]; then break; fi
done

# 3. 检查 MinIO 是否收到文件（如果有 mc 客户端）
echo "[Check MinIO]"
mc ls local/pm-files/ 2>/dev/null | tail -5 || echo "  (mc 未安装，跳过 MinIO 检查)"
```

#### T18: 批量任务优先级验证

```bash
#!/bin/bash
# t18_batch_priority.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T18: 批量任务优先级 ==="

# 创建 5 个不同优先级的任务
echo "[Step 1] 创建 5 个任务（优先级 50, 10, 30, 1, 20）..."

for P in 50 10 30 1 20; do
    curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{
        \"method\": \"GetParameterValues\",
        \"params\": {\"names\": [\"Device.DeviceInfo.Manufacturer\"]},
        \"priority\": $P,
        \"description\": \"T18: Priority=$P 测试\"
      }" > /dev/null
    echo "  Created task with priority=$P"
done

# 等待并观察执行顺序
echo "[Step 2] 等待执行并查看任务历史..."
sleep 120  # 等 2 个 Inform 周期

curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN&page_size=10" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json
data = json.load(sys.stdin)
items = data.get('data', {}).get('items', data.get('items', []))
print(f'Total tasks: {len(items)}')
for t in items:
    print(f'  [{t.get(\"status\",\"?\")}] priority={t.get(\"priority\",\"?\")} method={t.get(\"method\",\"?\")} sent_at={t.get(\"sent_at\",\"N/A\")}')
"
```

---

### 阶段三：任务生命周期验证

| 测试编号 | 测试项 | 验证方法 |
|----------|--------|----------|
| T19 | 任务取消 | 创建任务后立即 DELETE，检查 status=cancelled |
| T20 | 任务过期 | 创建 expires_in=10 的任务，等 15s 后验证 status=expired |
| T21 | 任务重试 | 创建必然失败的任务，验证 retry_count 递增直到 max_retries |
| T22 | 任务统计 | GET /tasks/stats 验证各状态计数正确 |

#### T19: 任务取消

```bash
#!/bin/bash
# t19_cancel_task.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T19: 任务取消 ==="

# 1. 创建任务
RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GetParameterValues",
    "params": {"names": ["Device.DeviceInfo.Manufacturer"]},
    "priority": 99,
    "description": "T19: 待取消的任务"
  }')
TASK_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
echo "  Created: $TASK_ID"

# 2. 立即取消
echo "[Step 2] 取消任务..."
curl -s -X DELETE "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool

# 3. 验证状态
STATUS=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
echo "  Final status: $STATUS"

if [ "$STATUS" = "cancelled" ]; then
    echo "  ✅ PASS"
else
    echo "  ❌ FAIL (expected 'cancelled', got '$STATUS')"
fi
```

#### T20: 任务过期

```bash
#!/bin/bash
# t20_task_expiry.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T20: 任务过期 ==="

# 创建一个 10 秒后过期的任务
RESULT=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "GetParameterValues",
    "params": {"names": ["Device.DeviceInfo.Manufacturer"]},
    "priority": 99,
    "expires_in": 10,
    "description": "T20: 10秒后过期的任务"
  }')
TASK_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)
echo "  Created: $TASK_ID"

# 等待过期
echo "  等待 15 秒..."
sleep 15

# 检查状态（可能需要等下一次 Inform 时 ACS 检查过期）
sleep 70  # 等一个 Inform 周期让 ACS pop 并检查过期

STATUS=$(curl -s "http://localhost:8081/api/v1/devices/tasks/$TASK_ID" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
echo "  Final status: $STATUS"

if [ "$STATUS" = "expired" ]; then
    echo "  ✅ PASS"
else
    echo "  ⚠️  Status is '$STATUS' (may need ACS to actively check expiry)"
fi
```

---

### 阶段四：文件上传通道验证

| 测试编号 | 测试项 | 验证方法 |
|----------|--------|----------|
| T30 | FileUploadService 直连 | 用 curl 直接 POST 文件到 ACS |
| T31 | CPE 触发上传 | 通过 Upload RPC 让 CPE 上传 |
| T32 | 大文件上传 | 上传接近 max_file_size 的文件 |

#### T30: FileUploadService 直连测试

```bash
#!/bin/bash
# t30_file_upload_direct.sh

echo "=== T30: FileUploadService 直连测试 ==="

# 创建一个测试 PM 文件
echo '<?xml version="1.0"?><measCollec><measData><measInfo><measType>RRC.ConnEstabAtt</measType><measValue>42</measValue></measInfo></measData></measCollec>' > /tmp/test_pm.xml
gzip /tmp/test_pm.xml 2>/dev/null || true

# 上传到 ACS FileUploadService
echo "[Step 1] 上传 PM 文件..."
UPLOAD_RESP=$(curl -sv "http://localhost:8080/smallcell/FileUploadService?fileType=PM&filename=test_device_20260322_150000.xml.gz" \
  -u "upload_user:secure_upload_password_123" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @/tmp/test_pm.xml.gz 2>&1)

HTTP_CODE=$(echo "$UPLOAD_RESP" | grep "< HTTP" | awk '{print $3}')
echo "  HTTP Status: $HTTP_CODE"

if [ "$HTTP_CODE" = "200" ]; then
    echo "  ✅ PASS: 文件上传成功"
else
    echo "  ❌ FAIL: HTTP $HTTP_CODE"
    echo "$UPLOAD_RESP" | tail -10
fi

rm -f /tmp/test_pm.xml /tmp/test_pm.xml.gz
```

---

### 阶段五：ACS 测试任务注入验证

> ACS 配置中 `enable_test_task_injection: true`，每次 Inform 会自动注入 3-10 个随机 RPC 任务

| 测试编号 | 测试项 | 验证方法 |
|----------|--------|----------|
| T40 | 自动注入 | 修复 device_tasks 表后，检查 Inform 后是否有 RPC 下发 |
| T41 | RPC 执行 | 观察 TR069Worker 日志，确认 CPE 正确响应各类 RPC |
| T42 | 任务完成率 | 统计 completed/failed 比率 |

#### T40: 测试任务注入验证

```bash
#!/bin/bash
# t40_test_injection.sh

DEVICE_SN="1202000588233HB0039"
TOKEN=$(curl -s http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

echo "=== T40: 测试任务注入验证 ==="
echo "(前提: device_tasks 表已创建，ACS 已重启)"

# 1. 记录当前任务数
BEFORE=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN&page_size=1" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('total', 0))" 2>/dev/null)
echo "[Step 1] 当前任务数: $BEFORE"

# 2. 等待 Inform + RPC 执行（2 个周期）
echo "[Step 2] 等待 130 秒（2 个 Inform 周期）..."
sleep 130

# 3. 检查新任务
AFTER=$(curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN&page_size=1" \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c "import sys,json; print(json.load(sys.stdin).get('total', 0))" 2>/dev/null)
echo "[Step 3] 现在任务数: $AFTER"

NEW_TASKS=$((AFTER - BEFORE))
if [ "$NEW_TASKS" -gt 0 ]; then
    echo "  ✅ PASS: 注入了 $NEW_TASKS 个新任务"
else
    echo "  ❌ FAIL: 没有新任务产生"
fi

# 4. 查看任务详情
echo "[Step 4] 最近任务列表:"
curl -s "http://localhost:8081/api/v1/devices/tasks?device_sn=$DEVICE_SN&page_size=20" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "
import sys, json
data = json.load(sys.stdin)
items = data.get('items', data.get('data', {}).get('items', []))
for t in items[:20]:
    print(f'  [{t[\"status\"]:10s}] {t[\"method\"]:30s} priority={t.get(\"priority\",\"?\")} | {t.get(\"description\",\"\")}')
"

# 5. 检查 TR069Worker 日志中的 RPC 交互
echo ""
echo "[Step 5] CPE 侧 RPC 日志:"
tail -500 /Users/watermelon/Documents/code/oam-c/oaim/targetoxm/logs/TR069Worker.log \
  | grep -E "GetParameterValues|SetParameterValues|GetParameterNames|AddObject|DeleteObject|Download|Upload|Reboot|FactoryReset" \
  | tail -20
```

---

## 3. 一键全量验证脚本

```bash
#!/bin/bash
# acs_full_verify.sh — ACS 全量验证（需在前置修复完成后运行）

set -e

DEVICE_SN="1202000588233HB0039"
PSQL="/usr/local/Cellar/postgresql@16/16.13/bin/psql"
DB_CONN="postgresql://omcgo:omcgo123@localhost:5432/omcgo"
ACS_URL="http://localhost:8080"
APP_URL="http://localhost:8081"
PASS=0
FAIL=0
SKIP=0

get_token() {
    curl -s "$APP_URL/api/v1/auth/login" \
      -H "Content-Type: application/json" \
      -d '{"username":"admin","password":"admin123"}' \
      | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])"
}

report() {
    if [ "$1" = "pass" ]; then
        PASS=$((PASS+1)); echo "  ✅ PASS: $2"
    elif [ "$1" = "fail" ]; then
        FAIL=$((FAIL+1)); echo "  ❌ FAIL: $2"
    else
        SKIP=$((SKIP+1)); echo "  ⏭️  SKIP: $2"
    fi
}

echo "============================================"
echo "  OMC ACS 端到端验证"
echo "  设备: $DEVICE_SN"
echo "  时间: $(date)"
echo "============================================"
echo ""

# --- 前置检查 ---
echo "▶ 前置检查"

# ACS 健康
HEALTH=$(curl -s "$ACS_URL/healthz")
[ "$HEALTH" = "ok" ] && report pass "ACS 健康检查" || report fail "ACS 健康检查: $HEALTH"

# device_tasks 表
DT=$($PSQL "$DB_CONN" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_name='device_tasks';" 2>/dev/null | tr -d ' ')
[ "$DT" = "1" ] && report pass "device_tasks 表存在" || report fail "device_tasks 表不存在（请先运行 migration）"

# 设备存在
DEV_STATUS=$($PSQL "$DB_CONN" -t -c "SELECT status FROM devices WHERE serial_number='$DEVICE_SN';" 2>/dev/null | tr -d ' ')
[ -n "$DEV_STATUS" ] && report pass "设备已注册 (status=$DEV_STATUS)" || report fail "设备未注册"

# Token
TOKEN=$(get_token 2>/dev/null)
[ -n "$TOKEN" ] && report pass "API 认证" || report fail "获取 Token 失败"

echo ""
echo "▶ Inform 处理 (等待 70 秒)"

BEFORE_INFORM=$($PSQL "$DB_CONN" -t -c "SELECT last_inform_at FROM devices WHERE serial_number='$DEVICE_SN';" 2>/dev/null)
HB_BEFORE=$(redis-cli GET "acs:heartbeat:$DEVICE_SN" 2>/dev/null)
sleep 70
AFTER_INFORM=$($PSQL "$DB_CONN" -t -c "SELECT last_inform_at FROM devices WHERE serial_number='$DEVICE_SN';" 2>/dev/null)
HB_AFTER=$(redis-cli GET "acs:heartbeat:$DEVICE_SN" 2>/dev/null)

[ "$BEFORE_INFORM" != "$AFTER_INFORM" ] && report pass "last_inform_at 更新" || report fail "last_inform_at 未更新 ($BEFORE_INFORM → $AFTER_INFORM)"
[ -n "$HB_AFTER" ] && report pass "Redis 心跳存在 ($HB_AFTER)" || report fail "Redis 心跳丢失"

echo ""
echo "▶ RPC 任务创建与执行"

# GetParameterValues
create_and_wait_task() {
    local METHOD="$1"
    local PARAMS="$2"
    local DESC="$3"
    local TOKEN=$(get_token)

    RESULT=$(curl -s "$APP_URL/api/v1/devices/tasks?device_sn=$DEVICE_SN" \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"method\":\"$METHOD\",\"params\":$PARAMS,\"priority\":5,\"description\":\"$DESC\"}")

    TASK_ID=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('id',''))" 2>/dev/null)

    if [ -z "$TASK_ID" ]; then
        report fail "$DESC — 创建失败: $RESULT"
        return
    fi

    for i in $(seq 1 16); do
        sleep 5
        STATUS=$(curl -s "$APP_URL/api/v1/devices/tasks/$TASK_ID" \
          -H "Authorization: Bearer $TOKEN" \
          | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))" 2>/dev/null)
        if [ "$STATUS" = "completed" ]; then
            report pass "$DESC"
            return
        elif [ "$STATUS" = "failed" ]; then
            report fail "$DESC — status=failed"
            return
        fi
    done
    report fail "$DESC — 超时 (status=$STATUS)"
}

create_and_wait_task "GetParameterValues" \
  '{"names":["Device.DeviceInfo.SoftwareVersion","Device.DeviceInfo.HardwareVersion"]}' \
  "GetParameterValues"

create_and_wait_task "GetParameterNames" \
  '{"path":"Device.DeviceInfo.","next_level":true}' \
  "GetParameterNames"

create_and_wait_task "SetParameterValues" \
  '{"values":[{"name":"Device.ManagementServer.PeriodicInformInterval","value":"120","type":"xsd:unsignedInt"}]}' \
  "SetParameterValues"

echo ""
echo "============================================"
echo "  结果汇总: ✅ $PASS 通过 | ❌ $FAIL 失败 | ⏭️  $SKIP 跳过"
echo "============================================"
```

---

## 4. 观测工具

### 实时监控命令

```bash
# 实时观察 CPE 日志
tail -f /Users/watermelon/Documents/code/oam-c/oaim/targetoxm/logs/TR069Worker.log | grep --color -E "Inform|Status:|ERROR|RPC|GetParam|SetParam"

# 实时观察 Redis 变化
redis-cli MONITOR | grep -E "acs:|cmdq:|heartbeat:"

# 观察设备状态变化
watch -n5 'PGPASSWORD=omcgo123 /usr/local/Cellar/postgresql@16/16.13/bin/psql -h localhost -U omcgo -d omcgo -t -c "SELECT status, last_inform_at, last_inform_events FROM devices WHERE serial_number='"'"'1202000588233HB0039'"'"';"'

# 查看 ACS 进程日志（如有 stdout 输出）
# 如果启动时没有重定向，可能需要重启并重定向 stdout
```

### 数据库查询

```sql
-- 设备完整信息
SELECT * FROM devices WHERE serial_number = '1202000588233HB0039';

-- 任务历史（按创建时间降序）
SELECT id, method, status, priority, created_at, sent_at, completed_at, error_message
FROM device_tasks
WHERE device_sn = '1202000588233HB0039'
ORDER BY created_at DESC
LIMIT 20;

-- 任务状态统计
SELECT status, COUNT(*) FROM device_tasks
WHERE device_sn = '1202000588233HB0039'
GROUP BY status;

-- 最近完成的任务及结果
SELECT method, result, completed_at
FROM device_tasks
WHERE device_sn = '1202000588233HB0039' AND status = 'completed'
ORDER BY completed_at DESC
LIMIT 10;
```

---

## 5. 验证优先级

| 优先级 | 行动 | 原因 |
|--------|------|------|
| **P0** | 执行 migration 到 48+ | device_tasks 表缺失导致任务系统完全不可用 |
| **P1** | 验证 Periodic Inform 更新 last_inform_at | 当前每次 Inform 后 DB 未更新，说明 InformHandler 可能有 bug |
| **P2** | 验证 Redis 心跳刷新 | 心跳 key 已过期，设备离线检测不工作 |
| **P3** | 验证 RPC 任务下发完整链路 | 核心业务——增删改查全部依赖此通道 |
| **P4** | 验证文件上传 | PM/MR 数据采集依赖此通道 |
