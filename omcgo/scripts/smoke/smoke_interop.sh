#!/usr/bin/env bash
# =============================================================================
# smoke_interop.sh — F10 互操作测试 业务冒烟
#
# 覆盖路由（omcgo/internal/interop/handler.go RegisterRoutes）：
#   GET  /api/v1/interop/test-cases          读：内置一致性用例集（5 类，启动期代码注册）
#   POST /api/v1/interop/run                 写：对设备跑一致性测试 → 只测负路径
#   POST /api/v1/interop/run/:category       写：按分类跑 → 只测负路径
#   POST /api/v1/interop/run/report          读：CSV 报告导出（同 run 执行路径）→ 负路径 + 格式校验
#   POST /api/v1/interop/validate/:deviceId  写：数据模型校验 → 只测负路径
#
# 红线：run/validate 会向设备派发 RPC 任务（含 X_CMCC_Reboot 探测等故障注入用例），
#       本脚本绝不使用真实存在的设备 SN / 设备 ID，全部用保证不存在的标识做负路径。
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F10 互操作测试" "$@"
smoke_login

# 保证不存在的设备标识（带 SMOKE_TAG，绝不与真实设备冲突）
NOSN="SMK-NOEXIST-${SMOKE_TAG}"
NOUUID="deadbeef-0000-4000-8000-000000000000"

# ---------------------------------------------------------------------------
section "测试用例列表（builtin 用例集，5 类）"
# ---------------------------------------------------------------------------
req GET "/api/v1/interop/test-cases"
check_ret_ok "用例列表可查"
# 5 类用例均为启动期代码注册（cmd/app/provider/modules.go initInteropModule），必有
check_list_nonempty "protocol 协议类用例非空" "data.protocol"
check_list_nonempty "datamodel 数据模型类用例非空" "data.datamodel"
check_list_nonempty "rpc 类用例非空" "data.rpc"
check_list_nonempty "inform 类用例非空" "data.inform"
check_list_nonempty "fault_inject 故障注入类用例非空" "data.fault_inject"
check_field "用例含 id 字段" "data.protocol.0.id"
check_field "用例含 name 字段" "data.protocol.0.name"
check_field "用例含 category 字段" "data.protocol.0.category"
check_field "用例含 steps 步骤" "data.protocol.0.steps.0.action"

# ---------------------------------------------------------------------------
section "run 一致性测试（红线：只测负路径，不向任何真实设备下发）"
# ---------------------------------------------------------------------------
req POST "/api/v1/interop/run" '{}'
check_ret_fail "缺 device_sn（binding required）被拒绝"

req POST "/api/v1/interop/run" "{\"device_sn\":\"${NOSN}\",\"categories\":[\"bogus\"]}"
check_status "非法 category → 400" 400

req POST "/api/v1/interop/run" "{\"device_sn\":\"${NOSN}\"}"
check_ret_fail "不存在设备 SN 被拒绝"
RUN_NF_CODE="$HTTP_CODE"
if [ "$RUN_NF_CODE" = "500" ]; then
    known_bug "run 不存在设备返回 500 而非 404" "runner.RunAll 的 device not found 用裸 fmt.Errorf，未 wrap commonerrors.ErrNotFound，HTTPStatusFromError 落默认 500"
fi

# ---------------------------------------------------------------------------
section "run/:category 按分类执行（负路径）"
# ---------------------------------------------------------------------------
req POST "/api/v1/interop/run/bogus" "{\"device_sn\":\"${NOSN}\"}"
check_status "非法分类路径 → 400" 400

req POST "/api/v1/interop/run/protocol" '{}'
check_ret_fail "protocol 分类缺 device_sn 被拒绝"

req POST "/api/v1/interop/run/protocol" "{\"device_sn\":\"${NOSN}\"}"
check_ret_fail "protocol 分类不存在设备被拒绝"

# ---------------------------------------------------------------------------
section "run/report 报告导出（格式校验 + 负路径）"
# ---------------------------------------------------------------------------
req POST "/api/v1/interop/run/report?format=pdf" "{\"device_sn\":\"${NOSN}\"}"
check_status "不支持的导出格式 pdf → 400" 400

req POST "/api/v1/interop/run/report" '{}'
check_ret_fail "report 缺 device_sn 被拒绝"

# 200 CSV 需要真实设备且会真跑用例（派发 RPC），红线禁止 → 只断言不存在设备被合理拒绝
req POST "/api/v1/interop/run/report?format=csv" "{\"device_sn\":\"${NOSN}\"}"
check_ret_fail "report 不存在设备被拒绝（不对真实设备跑导出）"

# ---------------------------------------------------------------------------
section "validate/:deviceId 数据模型校验（负路径）"
# ---------------------------------------------------------------------------
req POST "/api/v1/interop/validate/not-a-uuid?carrier=cmcc&tech=lte"
check_status "非法 deviceId（非 UUID）→ 400" 400

req POST "/api/v1/interop/validate/${NOUUID}?carrier=bogus&tech=lte"
check_status "非法 carrier → 400" 400

req POST "/api/v1/interop/validate/${NOUUID}?carrier=cmcc"
check_status "缺 tech → 400" 400

req POST "/api/v1/interop/validate/${NOUUID}?carrier=cmcc&tech=lte"
check_ret_fail "不存在设备的 validate 被拒绝"
VAL_NF_CODE="$HTTP_CODE"
if [ "$VAL_NF_CODE" = "500" ]; then
    known_bug "validate 不存在设备返回 500 而非 404" "GetByID 未命中返回 (nil,nil)，validator.resolveExpectedParams 报 'device productClass missing' 裸错误 → 默认 500，且报错语义误导"
fi

smoke_summary
