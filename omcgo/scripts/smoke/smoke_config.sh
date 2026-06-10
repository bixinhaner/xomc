#!/usr/bin/env bash
# =============================================================================
# smoke_config.sh — F02 配置管理 业务冒烟
#
# 覆盖域：config（模板/基线/配置任务/邻区/配置同步通道）+ devparam（设备参数树/快速设置）
#   - 模板 CRUD 闭环（建→查→改→删→查无），dispatch 只测参数校验负路径，不真发
#   - 基线 CRUD 闭环（建→查→改→删→查无）
#   - config/tasks 与 neighbors 列表（POST tasks 无删除端点不做闭环，只测参数校验）
#   - config/sync push/pull 只测参数校验负路径；status 用真实设备 SN（任务队列按 SN 键控）
#   - devparam：parameters/tree/children/search/schema/sync-status 全读（容忍空数据）
#   - SPV(PUT parameters)/objects/add/sync-params/discover/config-file/sync
#     一律只打不存在设备 UUID 负路径，绝不向真实设备下发任何指令
#   - quicksettings/groups 读（原始 JSON 响应，非信封；依赖产品路由，容忍 404/422）
#
# 用法：bash smoke_config.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F02 配置管理" "$@"
smoke_login

# 不存在但格式合法的设备/模板/基线 UUID（负路径专用）
NONEXIST_ID="00000000-0000-0000-0000-00000000dead"

# ---------------------------------------------------------------------------
section "数据准备：取一台真实设备（devparam 读链路 / sync status 依赖）"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices?page=1&page_size=5"
check_ret_ok "设备列表可查（数据准备）"
DEVICE_ID=$(jget data.items.0.id)
DEVICE_SN=$(jget data.items.0.serial_number)
if [ -n "$DEVICE_ID" ]; then
    echo "  · 选用设备 id=${DEVICE_ID} sn=${DEVICE_SN}"
else
    echo "  · 活栈无任何设备，设备相关用例将降级 skip"
fi

# ---------------------------------------------------------------------------
section "配置模板 CRUD 闭环"
# ---------------------------------------------------------------------------
TPL_NAME="${SMOKE_TAG}-tpl"

req GET "/api/v1/templates"
check_list_or_empty "模板列表可查" "data.items"

req GET "/api/v1/templates?carrier=cmcc&technology=lte"
check_list_or_empty "模板列表按运营商/制式过滤可查" "data.items"

# product_class 用带 SMOKE_TAG 的不可匹配值，杜绝任何真实设备 bootstrap 匹配到本模板
req POST "/api/v1/templates" "{
  \"name\": \"${TPL_NAME}\",
  \"carrier\": \"cmcc\",
  \"technology\": \"lte\",
  \"product_class\": \"SMK-NOMATCH-${SMOKE_TAG}\",
  \"template_type\": \"provisioning\",
  \"parameters\": [{\"path\": \"Device.ManagementServer.PeriodicInformInterval\", \"value\": \"300\"}],
  \"priority\": 10,
  \"auto_dispatch\": false,
  \"description\": \"smoke 自建模板\"
}"
check_status "创建模板" 201
TPL_ID=$(jget data.id)
check_field "创建模板返回 id" "data.id"

if [ -n "$TPL_ID" ]; then
    req GET "/api/v1/templates/$TPL_ID"
    check_ret_ok "模板详情可查"
    check_field "模板名称回读一致" "data.name"
    TPL_NAME_GOT=$(jget data.name)
    if [ "$TPL_NAME_GOT" = "$TPL_NAME" ]; then
        pass "模板名称与创建值一致 (${TPL_NAME})"
    else
        fail "模板名称与创建值一致" "期望 ${TPL_NAME}，实际 ${TPL_NAME_GOT}"
    fi

    req PUT "/api/v1/templates/$TPL_ID" "{
      \"name\": \"${TPL_NAME}\",
      \"carrier\": \"cmcc\",
      \"technology\": \"lte\",
      \"product_class\": \"SMK-NOMATCH-${SMOKE_TAG}\",
      \"template_type\": \"provisioning\",
      \"parameters\": [{\"path\": \"Device.ManagementServer.PeriodicInformInterval\", \"value\": \"600\"}],
      \"priority\": 20,
      \"active\": false,
      \"auto_dispatch\": false,
      \"description\": \"smoke 更新后描述\"
    }"
    check_ret_ok "更新模板"
    DESC_GOT=$(jget data.description)
    if [ "$DESC_GOT" = "smoke 更新后描述" ]; then
        pass "模板描述更新生效"
    else
        fail "模板描述更新生效" "期望 'smoke 更新后描述'，实际 '${DESC_GOT}'"
    fi
else
    skip "模板详情可查" "创建未返回 id"
    skip "模板更新" "创建未返回 id"
fi

# 模板创建参数校验负路径（缺必填 name/carrier）
req POST "/api/v1/templates" '{"technology":"lte"}'
check_ret_fail "创建模板缺必填字段被拒绝"

# ---------------------------------------------------------------------------
section "模板 dispatch 只测参数校验负路径（不真发）"
# ---------------------------------------------------------------------------
# 不存在模板 + 合法 body → 404 template not found（dispatcher 未配置时 503，同属拒绝）
req POST "/api/v1/templates/$NONEXIST_ID/dispatch" "{\"device_ids\":[\"$NONEXIST_ID\"]}"
check_ret_fail "dispatch 不存在模板被拒绝"

if [ -n "$TPL_ID" ]; then
    # 真实模板 + 空 device_ids → 400 must not be empty（handler 在 dispatch 前拦截，零下发）
    req POST "/api/v1/templates/$TPL_ID/dispatch" '{"device_ids":[]}'
    check_ret_fail "dispatch 空设备列表被拒绝"
else
    skip "dispatch 空设备列表被拒绝" "无自建模板 id"
fi

# ---------------------------------------------------------------------------
section "配置模板删除收尾（闭环）"
# ---------------------------------------------------------------------------
if [ -n "$TPL_ID" ]; then
    req DELETE "/api/v1/templates/$TPL_ID"
    check_ret_ok "删除模板"

    req GET "/api/v1/templates/$TPL_ID"
    check_ret_fail "删除后模板查无（404）"
else
    skip "删除模板" "创建未返回 id"
    skip "删除后模板查无" "创建未返回 id"
fi

# ---------------------------------------------------------------------------
section "配置基线 CRUD 闭环"
# ---------------------------------------------------------------------------
BL_NAME="${SMOKE_TAG}-baseline"

req GET "/api/v1/config/baselines"
check_list_or_empty "基线列表可查" "data.items"

req POST "/api/v1/config/baselines" "{
  \"baseline_name\": \"${BL_NAME}\",
  \"description\": \"smoke 自建基线\",
  \"device_type\": \"enb\",
  \"version\": \"v1\",
  \"params\": [{\"path\": \"Device.Time.NTPServer1\", \"value\": \"ntp.example.com\"}],
  \"status\": \"draft\"
}"
check_status "创建基线" 201
BL_ID=$(jget data.id)
check_field "创建基线返回 id" "data.id"

if [ -n "$BL_ID" ]; then
    req GET "/api/v1/config/baselines/$BL_ID"
    check_ret_ok "基线详情可查"
    BL_NAME_GOT=$(jget data.baseline_name)
    if [ "$BL_NAME_GOT" = "$BL_NAME" ]; then
        pass "基线名称与创建值一致 (${BL_NAME})"
    else
        fail "基线名称与创建值一致" "期望 ${BL_NAME}，实际 ${BL_NAME_GOT}"
    fi

    req PUT "/api/v1/config/baselines/$BL_ID" "{
      \"baseline_name\": \"${BL_NAME}\",
      \"description\": \"smoke 基线更新后描述\",
      \"device_type\": \"enb\",
      \"version\": \"v2\",
      \"params\": [{\"path\": \"Device.Time.NTPServer1\", \"value\": \"ntp2.example.com\"}],
      \"status\": \"draft\"
    }"
    check_ret_ok "更新基线"
    BL_VER_GOT=$(jget data.version)
    if [ "$BL_VER_GOT" = "v2" ]; then
        pass "基线版本更新生效 (v2)"
    else
        fail "基线版本更新生效" "期望 v2，实际 '${BL_VER_GOT}'"
    fi
else
    skip "基线详情可查" "创建未返回 id"
    skip "基线更新" "创建未返回 id"
fi

# 基线创建参数校验负路径（缺必填 baseline_name）
req POST "/api/v1/config/baselines" '{"device_type":"enb"}'
check_ret_fail "创建基线缺必填字段被拒绝"

# 更新不存在基线负路径
req PUT "/api/v1/config/baselines/$NONEXIST_ID" "{\"baseline_name\":\"${SMOKE_TAG}-ghost\"}"
check_ret_fail "更新不存在基线被拒绝"

if [ -n "$BL_ID" ]; then
    req DELETE "/api/v1/config/baselines/$BL_ID"
    check_ret_ok "删除基线"

    req GET "/api/v1/config/baselines/$BL_ID"
    check_ret_fail "删除后基线查无"
else
    skip "删除基线" "创建未返回 id"
    skip "删除后基线查无" "创建未返回 id"
fi

# ---------------------------------------------------------------------------
section "配置任务与邻区列表"
# ---------------------------------------------------------------------------
req GET "/api/v1/config/tasks"
check_list_or_empty "配置任务列表可查" "data.items"

req GET "/api/v1/config/tasks?status=pending"
check_ret_ok "配置任务按状态过滤可查"

# POST /config/tasks 无删除端点不做闭环，只测参数校验（缺必填 task_name/task_type）
req POST "/api/v1/config/tasks" '{"creator":"smoke"}'
check_ret_fail "创建配置任务缺必填字段被拒绝"

req GET "/api/v1/config/neighbors"
check_list_or_empty "邻区列表可查" "data.items"

# ---------------------------------------------------------------------------
section "配置同步通道（push/pull 只测参数校验，status 走真实设备）"
# ---------------------------------------------------------------------------
# push：不存在设备 + 空参数 body → binding min=1 拒绝（零下发）
req POST "/api/v1/config/sync/push/no-such-dev-${SMOKE_TAG}" '{"parameters":[]}'
check_ret_fail "push 配置空参数被拒绝"

req POST "/api/v1/config/sync/push/no-such-dev-${SMOKE_TAG}" '{}'
check_ret_fail "push 配置缺 parameters 字段被拒绝"

# pull：缺 parameter_names → 400（零下发）
req POST "/api/v1/config/sync/pull/no-such-dev-${SMOKE_TAG}" '{}'
check_ret_fail "pull 配置缺 parameter_names 字段被拒绝"

# status：任务队列按设备 SN 键控（handler 将 :deviceId 直接作 DeviceSN 查队列长度）
if [ -n "$DEVICE_SN" ]; then
    req GET "/api/v1/config/sync/status/$DEVICE_SN"
    check_ret_ok "设备配置同步状态可查"
    check_field "同步状态含 device_id" "data.device_id"
    PENDING=$(jget data.pending_count)
    if [ -n "$PENDING" ]; then
        pass "同步状态含 pending_count (=${PENDING})"
    else
        fail "同步状态含 pending_count" "字段缺失"
    fi
else
    skip "设备配置同步状态可查" "活栈无设备"
    skip "同步状态含 device_id" "活栈无设备"
    skip "同步状态含 pending_count" "活栈无设备"
fi

# ---------------------------------------------------------------------------
section "设备参数树读链路（devparam，容忍设备无已同步参数）"
# ---------------------------------------------------------------------------
if [ -n "$DEVICE_ID" ]; then
    req GET "/api/v1/devices/$DEVICE_ID/parameters"
    check_list_or_empty "已同步参数值列表可查" "data.items"

    req GET "/api/v1/devices/$DEVICE_ID/parameters/tree"
    check_ret_ok "参数树可查"

    req GET "/api/v1/devices/$DEVICE_ID/parameters/tree?format=flat"
    check_list_or_empty "参数树扁平格式可查" "data.items"

    req GET "/api/v1/devices/$DEVICE_ID/parameters/children?path_prefix=Device."
    check_ret_ok "参数子节点懒加载可查（path_prefix=Device.）"

    req GET "/api/v1/devices/$DEVICE_ID/parameters/search?q=Device"
    check_list_or_empty "参数搜索可查（q=Device）" "data.items"

    req GET "/api/v1/devices/$DEVICE_ID/parameters/schema"
    check_ret_ok "参数 schema 可查"
    SCHEMA_TOTAL=$(jget data.total)
    if [ -n "$SCHEMA_TOTAL" ]; then
        pass "参数 schema 含 total (=${SCHEMA_TOTAL})"
    else
        fail "参数 schema 含 total" "字段缺失"
    fi

    req GET "/api/v1/devices/$DEVICE_ID/parameters/sync-status"
    check_ret_ok "参数同步状态可查"
    check_field "同步状态含 status 字段" "data.status"
else
    skip "已同步参数值列表可查" "活栈无设备"
    skip "参数树可查" "活栈无设备"
    skip "参数树扁平格式可查" "活栈无设备"
    skip "参数子节点懒加载可查" "活栈无设备"
    skip "参数搜索可查" "活栈无设备"
    skip "参数 schema 可查" "活栈无设备"
    skip "参数同步状态可查" "活栈无设备"
fi

# 读端点参数校验负路径（与设备存在与否无关）
req GET "/api/v1/devices/not-a-uuid/parameters"
check_ret_fail "非法设备 id 查参数被拒绝"

if [ -n "$DEVICE_ID" ]; then
    req GET "/api/v1/devices/$DEVICE_ID/parameters/children"
    check_ret_fail "children 缺 path_prefix 被拒绝"

    req GET "/api/v1/devices/$DEVICE_ID/parameters/search"
    check_ret_fail "search 缺 q 被拒绝"
else
    req GET "/api/v1/devices/$NONEXIST_ID/parameters/children"
    check_ret_fail "children 缺 path_prefix 被拒绝"

    req GET "/api/v1/devices/$NONEXIST_ID/parameters/search"
    check_ret_fail "search 缺 q 被拒绝"
fi

# ---------------------------------------------------------------------------
section "SPV/AddObject/同步类只测不存在设备负路径（红线：绝不真发）"
# ---------------------------------------------------------------------------
# SPV（PUT parameters）：合法 body + 不存在设备 → 404，零下发
req PUT "/api/v1/devices/$NONEXIST_ID/parameters" '{"parameters":[{"path":"Device.X.Y","value":"1","type":"string"}]}'
check_ret_fail "SPV 不存在设备被拒绝"

req PUT "/api/v1/devices/$NONEXIST_ID/parameters" '{"parameters":[]}'
check_ret_fail "SPV 空参数列表被拒绝"

req POST "/api/v1/devices/$NONEXIST_ID/objects/add" '{"object_path":"Device.Services.FAPService."}'
check_ret_fail "AddObject 不存在设备被拒绝"

req POST "/api/v1/devices/$NONEXIST_ID/objects/add" '{}'
check_ret_fail "AddObject 缺 object_path 被拒绝"

req POST "/api/v1/devices/$NONEXIST_ID/objects/delete" '{"object_path":"Device.Services.FAPService.1."}'
check_ret_fail "DeleteObject 不存在设备被拒绝"

req POST "/api/v1/devices/$NONEXIST_ID/sync-params" '{}'
check_ret_fail "sync-params 不存在设备被拒绝"

req POST "/api/v1/devices/$NONEXIST_ID/parameters/discover"
check_ret_fail "参数发现 不存在设备被拒绝"

req POST "/api/v1/devices/$NONEXIST_ID/config-file/sync"
check_ret_fail "配置文件同步 不存在设备被拒绝"

# ---------------------------------------------------------------------------
section "快速设置分组元数据（quicksettings）"
# ---------------------------------------------------------------------------
# 注意：该端点返回原始 JSON（非 {ret,msg,data} 信封），断言走 HTTP 状态 + 字段
req GET "/api/v1/quicksettings/groups"
check_ret_fail "快速设置缺 device_id 被拒绝"

req GET "/api/v1/quicksettings/groups?device_id=not-a-uuid"
check_ret_fail "快速设置非法 device_id 被拒绝"

if [ -n "$DEVICE_ID" ]; then
    req GET "/api/v1/quicksettings/groups?device_id=$DEVICE_ID"
    # 200=有分组元数据；404=设备 productClass 未路由到 product；422=product 未配置 paramModel
    # 后两种属活栈数据形态（产品装配件未配齐），不是接口故障
    check_status_in "快速设置分组可查（真实设备）" "200 404 422"
    if [ "$HTTP_CODE" = "200" ]; then
        check_field "快速设置响应含 param_model" "param_model"
    fi
else
    skip "快速设置分组可查（真实设备）" "活栈无设备"
fi

smoke_summary
