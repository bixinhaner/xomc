#!/usr/bin/env bash
# =============================================================================
# smoke_backup.sh — F06 配置备份 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=backup 全部 read 路由 + 可逆 write 闭环）：
#   1. 读链路：tasks / schedules / ftp-configs / policy / restore-tasks /
#      config-snapshots / device-licenses 全读
#   2. 备份任务闭环：负路径校验 + 自建任务（目标 SN 不存在，不触达真实设备）
#      → 查详情 → cancel → 删除
#   3. 计划 schedules CRUD 闭环（cron_expr 字段，enabled=false 不会真触发）
#   4. FTP 配置闭环：建 → test（连不通也算端点可达，见下注）→ 改 → 删
#   5. 策略 policy：GET 留底 → PUT 原值回写（幂等）+ 非法值负路径
#   6. 配置快照：列表/过滤/不存在 SN 404、validate-sns 只读校验、
#      batch-delete 只测参数校验负路径
#   7. 设备 License 库：同快照结构的读 + validate-sns + batch-delete 负路径
#   8. restore 系列【红线】只测参数校验负路径，绝不真实下发恢复
#   9. 运营商规范别名（M4）/task/enb/config/backupRestore/*：
#      - 只读查询：queryTaskList / queryCellInfos / queryTaskDeviceList（空结果容忍）
#      - 写类红线（addBackupRestoreTask / terminateTask / single/importFile）
#        只测参数校验负路径，绝不真实下发任务/恢复
#
# 注：FTP test 端点 handler 永远 response.OK 包装探测结果（success=false 表示
#     连不通），tester 未注入时也是 200 stub —— 2xx 受理或 4xx/5xx 都算可达。
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 配置备份" "$@"
smoke_login

# 不存在的实体（带 SMOKE_TAG 前缀，绝不会命中真实设备）
NOPE_SN="${SMOKE_TAG}-NOPE-SN"
NOPE_UUID="00000000-0000-4000-8000-00000000dead"

# ---------------------------------------------------------------------------
section "0. 数据准备：取一台设备 SN（仅用于 validate-sns 只读校验）"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices?page=1&page_size=1"
check_ret_ok "设备列表可查（数据准备）"
DEV_SN=$(jget data.items.0.serial_number)
if [ -n "$DEV_SN" ]; then
    echo "  · 取到设备 SN=${DEV_SN}（仅作只读校验输入）"
else
    echo "  · 活栈无设备，validate-sns 正向断言将降级 skip"
fi

# ---------------------------------------------------------------------------
section "1. 读链路：tasks/schedules/ftp-configs/policy/restore-tasks/快照/License 全读"
# ---------------------------------------------------------------------------
req GET "/api/v1/backup/tasks"
check_list_or_empty "备份任务列表可查" "data.items"

req GET "/api/v1/backup/tasks?status=completed&task_type=config_only&page=1&page_size=5"
check_ret_ok "备份任务列表带 status/task_type 过滤可查"

req GET "/api/v1/backup/schedules"
check_list_or_empty "备份计划列表可查" "data.items"

req GET "/api/v1/backup/ftp-configs"
check_list_or_empty "FTP 配置列表可查" "data.items"

req GET "/api/v1/backup/policy"
check_ret_ok "备份策略可查（单例，空表时返回默认值）"
check_field "策略含 retention_days 字段" "data.retention_days"

req GET "/api/v1/backup/restore-tasks"
check_list_or_empty "恢复任务列表可查" "data.items"

req GET "/api/v1/backup/config-snapshots"
check_list_or_empty "配置快照列表可查" "data.items"

req GET "/api/v1/backup/device-licenses"
check_list_or_empty "设备 License 列表可查" "data.items"

# ---------------------------------------------------------------------------
section "2. 备份任务闭环（目标 SN 不存在 → executor 跳过，不触达任何真实设备）"
# ---------------------------------------------------------------------------
req POST "/api/v1/backup/tasks" '{}'
check_ret_fail "建任务缺必填 task_type/target_type 被拒"

req POST "/api/v1/backup/tasks" "{\"task_type\":\"config_only\",\"target_type\":\"device\",\"target_ids\":[\"${NOPE_SN}\"]}"
check_status "建备份任务（目标 SN 不存在）" 201
TASK_ID=$(jget data.id)
check_field "建任务返回 id" "data.id"

if [ -n "$TASK_ID" ]; then
    req GET "/api/v1/backup/tasks/${TASK_ID}"
    check_ret_ok "查任务详情"
    check_field "任务详情 task_type 回显" "data.task_type"

    # executor 收事件后可能已把任务标 failed（目标设备不存在被跳过），
    # cancel 返回 200（仍 pending/running）或 400 biz_code 8100（已终态）都合法
    req POST "/api/v1/backup/tasks/${TASK_ID}/cancel"
    check_status_in "取消任务（已进终态时合法拒绝）" "200 400"

    req DELETE "/api/v1/backup/tasks/${TASK_ID}"
    check_ret_ok "删除自建任务（闭环清理）"
else
    skip "查/取消/删除任务闭环" "建任务未返回 id，无法继续闭环"
fi

req GET "/api/v1/backup/tasks/not-a-uuid"
check_ret_fail "非法任务 ID 格式被拒"

req GET "/api/v1/backup/tasks/${NOPE_UUID}"
check_status "查不存在任务 → 404" 404

# ---------------------------------------------------------------------------
section "3. 计划 schedules CRUD 闭环（cron_expr 字段；enabled=false 不会真触发）"
# ---------------------------------------------------------------------------
req POST "/api/v1/backup/schedules" '{"name":"x"}'
check_ret_fail "建计划缺必填 cron_expr/task_type 被拒"

SCHED_NAME="${SMOKE_TAG}-sched"
req POST "/api/v1/backup/schedules" "{\"name\":\"${SCHED_NAME}\",\"cron_expr\":\"0 3 * * *\",\"enabled\":false,\"task_type\":\"config_only\",\"target_type\":\"device\",\"target_ids\":[]}"
check_status "建备份计划（enabled=false）" 201
SCHED_ID=$(jget data.id)
check_field "建计划返回 id" "data.id"

if [ -n "$SCHED_ID" ]; then
    req PUT "/api/v1/backup/schedules/${SCHED_ID}" "{\"name\":\"${SCHED_NAME}-upd\",\"cron_expr\":\"30 4 * * *\",\"enabled\":false,\"task_type\":\"full\",\"target_type\":\"device\",\"target_ids\":[]}"
    check_ret_ok "更新计划（改名+改 cron_expr+改 task_type）"
    UPD_NAME=$(jget data.name)
    if [ "$UPD_NAME" = "${SCHED_NAME}-upd" ]; then
        pass "计划更新字段生效 (name=${UPD_NAME})"
    else
        fail "计划更新字段生效" "期望 name=${SCHED_NAME}-upd，实际 '${UPD_NAME}'"
    fi

    req DELETE "/api/v1/backup/schedules/${SCHED_ID}"
    check_ret_ok "删除计划（闭环清理）"

    req GET "/api/v1/backup/schedules?page=1&page_size=100"
    check_ret_ok "删除后计划列表仍可查"
    if printf '%s' "$BODY" | grep -q "$SCHED_NAME"; then
        fail "计划删除后列表无残留" "列表中仍能找到 ${SCHED_NAME}"
    else
        pass "计划删除后列表无残留"
    fi
else
    skip "计划 更新/删除 闭环" "建计划未返回 id，无法继续闭环"
fi

# ---------------------------------------------------------------------------
section "4. FTP 配置闭环（建 → test → 改 → 删；test 连不通也算端点可达）"
# ---------------------------------------------------------------------------
req POST "/api/v1/backup/ftp-configs" '{}'
check_ret_fail "建 FTP 配置缺必填 config_name/host/username 被拒"

FTP_NAME="${SMOKE_TAG}-ftp"
req POST "/api/v1/backup/ftp-configs" "{\"config_name\":\"${FTP_NAME}\",\"host\":\"127.0.0.1\",\"port\":2121,\"username\":\"smokeuser\",\"password_encrypted\":\"smokepass\",\"protocol\":\"FTP\",\"remote_path\":\"/smoke\",\"passive\":true,\"enabled\":false}"
check_status "建 FTP 配置" 201
FTP_ID=$(jget data.id)
check_field "建 FTP 配置返回 id" "data.id"

if [ -n "$FTP_ID" ]; then
    # handler 把探测结果包进 response.OK（success=false 表示连不通），
    # 2xx 受理或 4xx/5xx 连通失败都证明端点可达
    req POST "/api/v1/backup/ftp-configs/${FTP_ID}/test"
    check_status_in "FTP 连通性测试端点可达（127.0.0.1:2121 连不通属正常）" "200 201 202 400 408 500 502 503 504"

    req PUT "/api/v1/backup/ftp-configs/${FTP_ID}" "{\"config_name\":\"${FTP_NAME}\",\"host\":\"127.0.0.1\",\"port\":2121,\"username\":\"smokeuser\",\"password_encrypted\":\"smokepass\",\"protocol\":\"SFTP\",\"remote_path\":\"/smoke-upd\",\"passive\":false,\"enabled\":false}"
    check_ret_ok "更新 FTP 配置（改 protocol/remote_path）"

    req DELETE "/api/v1/backup/ftp-configs/${FTP_ID}"
    check_ret_ok "删除 FTP 配置（闭环清理）"

    req POST "/api/v1/backup/ftp-configs/${FTP_ID}/test"
    check_ret_fail "已删除 FTP 配置再 test 被拒（404）"
else
    skip "FTP test/更新/删除 闭环" "建 FTP 配置未返回 id，无法继续闭环"
fi

# ---------------------------------------------------------------------------
section "5. 策略 policy：GET 留底 → PUT 原值回写（幂等）+ 非法值负路径"
# ---------------------------------------------------------------------------
req GET "/api/v1/backup/policy"
check_ret_ok "策略 GET 留底"
POLICY_JSON=$(jget data)

if [ -n "$POLICY_JSON" ]; then
    req PUT "/api/v1/backup/policy" "$POLICY_JSON"
    check_ret_ok "策略原值回写（幂等 PUT）"
    check_field "回写后策略仍含 retention_days" "data.retention_days"
else
    skip "策略原值回写" "GET 未取到策略 data 对象"
fi

req PUT "/api/v1/backup/policy" '{"retention_days":-1}'
check_ret_fail "策略非法值（retention_days=-1）被 validatePolicy 拒绝"

# ---------------------------------------------------------------------------
section "6. 配置快照：列表过滤 / 不存在 SN 404 / validate-sns 只读校验 / batch-delete 负路径"
# ---------------------------------------------------------------------------
req GET "/api/v1/backup/config-snapshots?serial_number=${NOPE_SN}&page=1&page_size=10"
check_list_or_empty "快照列表按 serial_number 过滤可查" "data.items"

req GET "/api/v1/backup/config-snapshots/${NOPE_SN}"
check_status "查不存在 SN 的快照 → 404" 404

req GET "/api/v1/backup/config-snapshots/${NOPE_SN}/download"
check_status "下载不存在 SN 的快照 → 404" 404

req POST "/api/v1/backup/config-snapshots/validate-sns" '{}'
check_ret_fail "validate-sns 缺 serial_numbers 被拒"

req POST "/api/v1/backup/config-snapshots/validate-sns" "{\"serial_numbers\":[\"${NOPE_SN}\"]}"
check_ret_ok "validate-sns 只读校验（不存在 SN）"
check_list_nonempty "不存在 SN 落入 missing 列表" "data.missing"

if [ -n "$DEV_SN" ]; then
    req POST "/api/v1/backup/config-snapshots/validate-sns" "{\"serial_numbers\":[\"${DEV_SN}\"]}"
    check_ret_ok "validate-sns 只读校验（真实设备 SN）"
    check_list_nonempty "真实 SN 落入 existing 列表" "data.existing"
else
    skip "validate-sns 真实 SN 正向校验" "活栈无设备可用"
fi

req POST "/api/v1/backup/config-snapshots/batch-get" "{\"serial_numbers\":[\"${NOPE_SN}\"]}"
check_ret_ok "batch-get 批量查快照（只读）"
check_list_nonempty "batch-get 不存在 SN 报 missing" "data.missing"

# destructive：只测参数校验负路径，不真删任何快照
req POST "/api/v1/backup/config-snapshots/batch-delete" '{}'
check_ret_fail "batch-delete 缺 serial_numbers 被拒（不执行真删）"

# ---------------------------------------------------------------------------
section "7. 设备 License 库（与快照同构）：读 + validate-sns + batch-delete 负路径"
# ---------------------------------------------------------------------------
req GET "/api/v1/backup/device-licenses/${NOPE_SN}"
check_status "查不存在 SN 的 License → 404" 404

req GET "/api/v1/backup/device-licenses/${NOPE_SN}/download"
check_status "下载不存在 SN 的 License → 404" 404

req POST "/api/v1/backup/device-licenses/validate-sns" "{\"serial_numbers\":[\"${NOPE_SN}\"]}"
check_ret_ok "License validate-sns 只读校验"
check_list_nonempty "License validate-sns 不存在 SN 报 missing" "data.missing"

req POST "/api/v1/backup/device-licenses/batch-delete" '{}'
check_ret_fail "License batch-delete 缺 serial_numbers 被拒（不执行真删）"

# ---------------------------------------------------------------------------
section "8. 恢复 restore【红线：只测参数校验负路径，绝不真实下发恢复】"
# ---------------------------------------------------------------------------
req POST "/api/v1/backup/restore" '{}'
check_ret_fail "restore 缺必填 bucket/object_path/target_device_sns 被拒"

req POST "/api/v1/backup/restore" "{\"bucket\":\"firmware\",\"object_path\":\"x.xml\",\"target_device_sns\":[\"${NOPE_SN}\"]}"
check_ret_fail "restore 非法 bucket（只允许 config_backup）被拒"

req POST "/api/v1/backup/restore" "{\"bucket\":\"config_backup\",\"object_path\":\"smoke/${SMOKE_TAG}-none.xml\",\"target_device_sns\":[\"${NOPE_SN}\"]}"
# 期望：源对象不存在 → 精确 404。#145-B 修复后硬断言。
# 历史根因：CanonicalRestoreBucket="config_backup" 含下划线违反 S3/MinIO 桶命名，
# StatObject 返回 InvalidBucketName；translateMinIONotFound 原只翻 NoSuchKey/NoSuchBucket，
# 漏了 InvalidBucketName → 落 default 500。修复已把 InvalidBucketName/XMinioInvalidObjectName
# 纳入 minioNotFoundCodes → 翻译为 ErrNotFound → handler 映射 404。
check_status "restore 源对象不存在 → 精确 404（#145-B 修复后硬断言）" 404

req POST "/api/v1/backup/restore/by-task-id" '{}'
check_ret_fail "restore/by-task-id 缺必填字段被拒"

req POST "/api/v1/backup/restore/by-task-id" "{\"backup_task_id\":\"${NOPE_UUID}\",\"target_device_sns\":[\"${NOPE_SN}\"]}"
check_ret_fail "restore/by-task-id 不存在任务被拒"

req POST "/api/v1/backup/restore/by-snapshot" '{}'
check_ret_fail "restore/by-snapshot 缺 target_device_sns 被拒"

req POST "/api/v1/backup/restore/by-snapshot" "{\"target_device_sns\":[\"${NOPE_SN}\"]}"
check_ret_fail "restore/by-snapshot 无快照 SN 整体拒绝（404 带 missing）"

req GET "/api/v1/backup/restore-tasks/not-a-uuid"
check_ret_fail "非法恢复任务 ID 格式被拒"

req GET "/api/v1/backup/restore-tasks/${NOPE_UUID}"
check_status "查不存在恢复任务 → 404" 404

# ---------------------------------------------------------------------------
section "9a. 规范别名 只读查询：queryTaskList / queryCellInfos / queryTaskDeviceList"
# ---------------------------------------------------------------------------
# body 为 ListTasksAliasRequest（嵌入 model.ListRequest，无 json tag，
# encoding/json 按字段名大小写不敏感匹配 Page/PageSize；空 body 走默认分页）
req POST "/api/v1/task/enb/config/backupRestore/queryTaskList" '{"Page":1,"PageSize":10}'
check_ret_ok "规范别名 queryTaskList 可达（显式分页）"
check_list_or_empty "queryTaskList 返回任务列表" "data.items"

req POST "/api/v1/task/enb/config/backupRestore/queryTaskList" '{}'
check_ret_ok "规范别名 queryTaskList 空 body 走默认分页"

# queryCellInfos：按运营商/产品类型/关键字过滤基站列表，分页返回（只读，data.items）。
# deviceReader 在 modules.go 已注入（SetDeviceReader）；未注入时 handler 返回 503。
req POST "/api/v1/task/enb/config/backupRestore/queryCellInfos" '{}'
check_ret_ok "规范别名 queryCellInfos 可达（空 body 走默认分页）"
check_list_or_empty "queryCellInfos 返回基站列表" "data.items"

req POST "/api/v1/task/enb/config/backupRestore/queryCellInfos" '{"Page":1,"PageSize":5,"operator_code":"CMCC","product_type":"NOPE-PRODUCT","search":"NOPE-KEYWORD"}'
check_ret_ok "queryCellInfos 带 operator_code/product_type/search 过滤可查"
check_list_or_empty "queryCellInfos 过滤后列表（无命中容忍空）" "data.items"

# queryTaskDeviceList：按 task_id 列出目标设备视图（task body 必填 → uuid → GetTask）。
# 负路径：缺 task_id / 非法 uuid / 不存在任务（不触达任何设备执行）。
req POST "/api/v1/task/enb/config/backupRestore/queryTaskDeviceList" '{}'
check_ret_fail "queryTaskDeviceList 缺 task_id 被拒"

req POST "/api/v1/task/enb/config/backupRestore/queryTaskDeviceList" '{"task_id":"not-a-uuid"}'
check_ret_fail "queryTaskDeviceList 非法 task_id 格式被拒"

req POST "/api/v1/task/enb/config/backupRestore/queryTaskDeviceList" "{\"task_id\":\"${NOPE_UUID}\"}"
check_status "queryTaskDeviceList 查不存在任务 → 404" 404

# 正向只读：用 §2 自建并已清理的任务无法复用，改用 queryTaskList 取一条现存任务 id（若有）
req POST "/api/v1/task/enb/config/backupRestore/queryTaskList" '{"Page":1,"PageSize":1}'
EXIST_TASK_ID=$(jget data.items.0.id)
if [ -n "$EXIST_TASK_ID" ]; then
    req POST "/api/v1/task/enb/config/backupRestore/queryTaskDeviceList" "{\"task_id\":\"${EXIST_TASK_ID}\"}"
    check_ret_ok "queryTaskDeviceList 现存任务只读查询（device 视图）"
    check_field "queryTaskDeviceList 回显 task_id" "data.task_id"
else
    skip "queryTaskDeviceList 现存任务正向查询" "活栈无现存备份任务可用"
fi

# ---------------------------------------------------------------------------
section "9b. 规范别名 写类【红线】只测参数校验负路径，绝不真实下发任务/恢复"
# ---------------------------------------------------------------------------
# addBackupRestoreTask 别名 = CreateTask（与 POST /backup/tasks 同 handler）。
# 只测缺必填被拒，绝不用真实 SN 自建任务（避免误触发 executor 真备份）。
req POST "/api/v1/task/enb/config/backupRestore/addBackupRestoreTask" '{}'
check_ret_fail "addBackupRestoreTask 缺必填 task_type/target_type 被拒"

# terminateTask 别名 = CancelTask（从 POST body 读 task_id）。
# 只测缺字段 / 非法 uuid / 不存在任务被拒，绝不取消任何真实运行中任务。
req POST "/api/v1/task/enb/config/backupRestore/terminateTask" '{}'
check_ret_fail "terminateTask 缺 task_id 被拒"

req POST "/api/v1/task/enb/config/backupRestore/terminateTask" '{"task_id":"not-a-uuid"}'
check_ret_fail "terminateTask 非法 task_id 格式被拒"

req POST "/api/v1/task/enb/config/backupRestore/terminateTask" "{\"task_id\":\"${NOPE_UUID}\"}"
check_status "terminateTask 终止不存在任务 → 404" 404

# single/importFile 别名 = CreateRestore（配置恢复下发红线）。
# 只测缺必填 / 非法 bucket 被拒，绝不真实下发恢复到任何设备。
req POST "/api/v1/task/enb/config/backupRestore/single/importFile" '{}'
check_ret_fail "single/importFile 缺必填 bucket/object_path/target_device_sns 被拒"

req POST "/api/v1/task/enb/config/backupRestore/single/importFile" "{\"bucket\":\"firmware\",\"object_path\":\"x.xml\",\"target_device_sns\":[\"${NOPE_SN}\"]}"
check_ret_fail "single/importFile 非法 bucket（只允许 config_backup）被拒"

smoke_summary
