#!/usr/bin/env bash
# =============================================================================
# smoke_mml.sh — F06 MML 控制台业务冒烟
#
# 覆盖：
#   - 命令检索读链路：commands 列表/详情/param-paths、commands/search、group-tree（树+flat）、sub-fields
#   - render + parse 纯函数闭环（不碰设备）：真实命令 render 出 MML 语句 → parse 回解析断言命令一致
#   - dangerous-check 正/负路径
#   - 兼容性查询：unsupported-paths、console/command-compatibility
#   - 任务读链路：tasks 列表/详情/results；start/pause/cancel/delete/export 仅负路径
#   - 脚本闭环：scripts 建→查→改→runs→删；start 仅负路径
#   - 模板 CRUD 闭环：templates 建→查→改→clone→删（含删后 404 验证）
#   - MML 后台治理：admin/standard-params、admin/commands、admin/groups 读 + group 闭环
#   - execute / execute-statements / structured / groups execute / tasks 创建：只测参数校验负路径
#
# 红线：绝不向任何设备 SN 真实下发 execute/start 等指令；所有 execute 系列
#       请求体都构造为「必然被参数校验拒绝」的形态（缺命令源 / 缺 device_sns /
#       空 statements / 不存在的 UUID），失败发生在任务创建之前。
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 MML 控制台" "$@"
smoke_login

NIL_UUID="00000000-0000-0000-0000-000000000000"

# ---------------------------------------------------------------------------
section "命令检索读链路（catalog builtin 字典）"
# ---------------------------------------------------------------------------
req GET "/api/v1/mml/commands?page=1&page_size=10"
check_ret_ok "MML 命令列表可查"
check_list_nonempty "命令列表非空（builtin catalog）" "data.items"
check_count_ge "命令总数 ≥ 1" "data.total" 1

# 搜索（console 联合 ILIKE，参数名是 q）
req GET "/api/v1/mml/commands/search?q=DEVICE_INFO&limit=20"
check_ret_ok "命令搜索 q=DEVICE_INFO 可查"
check_list_nonempty "搜索结果非空" "data.items"

# 从搜索结果挑一条 LST 命令，供后续 render/parse/dangerous-check/sub-fields 复用
CMD_ID=""
CMD_CODE=""
CMD_LOGICAL=""
CMD_OP=""
n=$(jlen "data.items")
i=0
while [ "$i" -lt "$n" ]; do
    op=$(jget "data.items.$i.operation_type")
    if [ "$op" = "LST" ]; then
        CMD_ID=$(jget "data.items.$i.command_id")
        CMD_CODE=$(jget "data.items.$i.command_code")
        CMD_LOGICAL=$(jget "data.items.$i.logical_code")
        CMD_OP="LST"
        break
    fi
    i=$((i + 1))
done
if [ -n "$CMD_ID" ]; then
    pass "选定真实 LST 命令：${CMD_CODE}（id=${CMD_ID}）"
else
    fail "选定真实 LST 命令" "搜索结果中没有 LST 命令"
fi

# 命令详情 + param-paths
if [ -n "$CMD_ID" ]; then
    req GET "/api/v1/mml/commands/$CMD_ID"
    check_ret_ok "命令详情可查"
    check_field "详情 id 回显" "data.id"
    check_field "详情 command_code 回显" "data.command_code"

    req GET "/api/v1/mml/commands/$CMD_ID/param-paths"
    check_ret_ok "命令 param-paths 可查"
else
    skip "命令详情/param-paths" "未选到命令"
fi

# 组树（console 3 列布局）
req GET "/api/v1/mml/group-tree"
check_ret_ok "命令组树可查"
check_list_nonempty "组树非空（chapter 章节）" "data.tree"

req GET "/api/v1/mml/group-tree?format=flat"
check_ret_ok "命令组树 format=flat 可查"

# sub-fields（console 命令字段元数据）
if [ -n "$CMD_ID" ]; then
    req GET "/api/v1/mml/commands/$CMD_ID/sub-fields"
    check_ret_ok "命令 sub-fields 可查"
    check_list_or_empty "sub_fields 列表可查" "data.sub_fields"
else
    skip "命令 sub-fields" "未选到命令"
fi

# ---------------------------------------------------------------------------
section "render + parse 纯函数闭环（不碰设备）"
# ---------------------------------------------------------------------------
if [ -n "$CMD_ID" ]; then
    # render：空选择 LST → 裸语句（如 "LST DEVICE_INFO"），纯渲染不下发
    req POST "/api/v1/mml/render" "{\"command_id\":\"$CMD_ID\",\"operation_type\":\"$CMD_OP\",\"selected_sub_field_ids\":[],\"values\":{}}"
    check_ret_ok "render 渲染 MML 语句"
    MML_STR=$(jget "data.mml_string")
    check_field "render 产出 mml_string" "data.mml_string"
    case "$MML_STR" in
        *"$CMD_LOGICAL"*) pass "渲染语句含 logical_code（${MML_STR}）" ;;
        *) fail "渲染语句含 logical_code" "期望含 ${CMD_LOGICAL}，实际 '$MML_STR'" ;;
    esac

    # parse：把渲染产物解析回 statement，断言命令码/命令 ID 一致 —— 闭环
    if [ -n "$MML_STR" ]; then
        req POST "/api/v1/mml/parse" "{\"mml_string\":\"${MML_STR};\"}"
        check_ret_ok "parse 解析 MML 语句"
        perr=$(jlen "data.parse_errors")
        if [ "$perr" -eq 0 ]; then
            pass "parse 无解析错误"
        else
            fail "parse 无解析错误" "parse_errors=$(jget data.parse_errors)"
        fi
        PARSED_LOGICAL=$(jget "data.statements.0.logical_code")
        PARSED_OP=$(jget "data.statements.0.operation_type")
        PARSED_CMD_ID=$(jget "data.statements.0.command_id")
        if [ "$PARSED_LOGICAL" = "$CMD_LOGICAL" ] && [ "$PARSED_OP" = "$CMD_OP" ]; then
            pass "闭环：parse 回解析 op+logical_code 一致（$PARSED_OP ${PARSED_LOGICAL}）"
        else
            fail "闭环：parse 回解析 op+logical_code 一致" "期望 $CMD_OP ${CMD_LOGICAL}，实际 $PARSED_OP $PARSED_LOGICAL"
        fi
        if [ "$PARSED_CMD_ID" = "$CMD_ID" ]; then
            pass "闭环：parse lookup 回填 command_id 与原命令一致"
        else
            fail "闭环：parse lookup 回填 command_id 与原命令一致" "期望 ${CMD_ID}，实际 '$PARSED_CMD_ID'"
        fi
    else
        skip "parse 闭环" "render 未产出语句"
    fi
else
    skip "render+parse 闭环" "未选到真实命令"
fi

# render 负路径：不存在的命令 → 404
req POST "/api/v1/mml/render" "{\"command_id\":\"$NIL_UUID\",\"operation_type\":\"LST\"}"
check_ret_fail "render 不存在命令被拒绝"

# render 负路径：非法 operation_type → 400（binding oneof）
if [ -n "$CMD_ID" ]; then
    req POST "/api/v1/mml/render" "{\"command_id\":\"$CMD_ID\",\"operation_type\":\"DROP\"}"
    check_ret_fail "render 非法 operation_type 被拒绝"
fi

# parse 容错：垃圾语句 HTTP 200 但 parse_errors 非空
req POST "/api/v1/mml/parse" "{\"mml_string\":\"FOO NOT_A_REAL_CMD;\"}"
check_ret_ok "parse 垃圾语句仍 200（错误进 parse_errors）"
perr=$(jlen "data.parse_errors")
if [ "$perr" -ge 1 ]; then
    pass "垃圾语句累入 parse_errors（$perr 条）"
else
    fail "垃圾语句累入 parse_errors" "parse_errors 为空"
fi

# ---------------------------------------------------------------------------
section "dangerous-check 危险命令检查"
# ---------------------------------------------------------------------------
if [ -n "$CMD_CODE" ]; then
    # URL encode 空格（command_code 形如 "LST DEVICE_INFO"）
    CMD_CODE_ENC=$(printf '%s' "$CMD_CODE" | sed 's/ /%20/g')
    req GET "/api/v1/mml/dangerous-check?command_code=$CMD_CODE_ENC"
    check_ret_ok "dangerous-check 真实命令可查"
    check_field "返回 dangerous 字段" "data.dangerous"
else
    skip "dangerous-check 正路径" "未选到命令"
fi
req GET "/api/v1/mml/dangerous-check"
check_ret_fail "dangerous-check 缺 command_code 被拒绝"

# ---------------------------------------------------------------------------
section "兼容性查询（unsupported-paths / command-compatibility）"
# ---------------------------------------------------------------------------
req GET "/api/v1/mml/unsupported-paths"
check_ret_ok "unsupported-paths 无参可查（空集语义）"
check_list_or_empty "unsupported paths 列表" "data.paths"

# product_class 取活栈第一台设备；取不到用 catalog 常见值（T-0177：孤儿 class 也返 200）
req GET "/api/v1/devices?page=1&page_size=1"
DEV_PRODUCT_CLASS=$(jget "data.items.0.product_class")
DEV_SN=$(jget "data.items.0.serial_number")
[ -z "$DEV_PRODUCT_CLASS" ] && DEV_PRODUCT_CLASS="FAP-LTE-100"
PC_ENC=$(printf '%s' "$DEV_PRODUCT_CLASS" | sed 's/ /%20/g')
req GET "/api/v1/mml/console/command-compatibility?product_class=$PC_ENC"
check_ret_ok "command-compatibility 可查（product_class=${DEV_PRODUCT_CLASS}）"

req GET "/api/v1/mml/console/command-compatibility"
check_ret_fail "command-compatibility 缺 product_class 被拒绝"

if [ -n "$DEV_SN" ]; then
    SN_ENC=$(printf '%s' "$DEV_SN" | sed 's/ /%20/g')
    req GET "/api/v1/mml/unsupported-paths?device_sn=$SN_ENC"
    check_ret_ok "unsupported-paths 按 device_sn 可查（sn=${DEV_SN}）"
else
    skip "unsupported-paths 按 device_sn" "活栈无设备"
fi

# ---------------------------------------------------------------------------
section "任务读链路（任务由运行期 POST 创建，非种子；首次部署可为空）"
# ---------------------------------------------------------------------------
# mml_tasks 不是种子数据（任何 seed 文件都不插），由 POST /api/v1/mml/tasks 运行期创建。
# 首次部署 / 全新初始化的库该列表合法为空 —— 故只校验「可查 + 空集语义」，
# 有任务时再机会性地走详情/结果读链路（与本文件 scripts/unsupported-paths 同范式）。
req GET "/api/v1/mml/tasks"
check_ret_ok "MML 任务列表可查"
check_list_or_empty "任务列表可查（空集语义；任务非种子）" "data.items"
TASK_ID=$(jget "data.items.0.id")

if [ -n "$TASK_ID" ]; then
    req GET "/api/v1/mml/tasks/$TASK_ID"
    check_ret_ok "任务详情可查"
    check_field "任务详情 id 回显" "data.id"

    req GET "/api/v1/mml/tasks/$TASK_ID/results?page=1&page_size=20"
    check_ret_ok "任务执行结果可查"
else
    skip "任务详情/结果" "任务列表为空"
fi

# 任务操作类全部只测负路径（start/pause/cancel/delete/export 不碰真实任务）
req POST "/api/v1/mml/tasks/$NIL_UUID/start" "{}"
check_ret_fail "start 不存在任务被拒绝"
req POST "/api/v1/mml/tasks/$NIL_UUID/pause" "{}"
check_ret_fail "pause 不存在任务被拒绝"
req POST "/api/v1/mml/tasks/$NIL_UUID/cancel" "{}"
check_ret_fail "cancel 不存在任务被拒绝"
req DELETE "/api/v1/mml/tasks/$NIL_UUID"
check_ret_fail "delete 不存在任务被拒绝"
req POST "/api/v1/mml/tasks/$NIL_UUID/export" "{}"
check_ret_fail "export 不存在任务被拒绝"
req GET "/api/v1/mml/tasks/$NIL_UUID/export/download"
check_ret_fail "下载不存在任务导出被拒绝"
req GET "/api/v1/mml/tasks/not-a-uuid"
check_ret_fail "任务详情非法 UUID 被拒绝"

# ---------------------------------------------------------------------------
section "脚本闭环（建→查→改→runs→删）"
# ---------------------------------------------------------------------------
req GET "/api/v1/mml/scripts"
check_ret_ok "脚本列表可查"
check_list_or_empty "脚本列表" "data.items"

SCRIPT_NAME="${SMOKE_TAG}-script"
req POST "/api/v1/mml/scripts" "{\"script_name\":\"$SCRIPT_NAME\",\"description\":\"smoke 自建脚本\",\"content\":\"LST DEVICE_INFO;\",\"tags\":[\"smoke\"]}"
check_ret_ok "创建脚本（${SCRIPT_NAME}）"
SCRIPT_ID=$(jget "data.id")
check_field "新脚本返回 id" "data.id"

if [ -n "$SCRIPT_ID" ]; then
    req GET "/api/v1/mml/scripts/$SCRIPT_ID"
    check_ret_ok "脚本详情可查"
    check_field "脚本名称回显" "data.script_name"

    req PUT "/api/v1/mml/scripts/$SCRIPT_ID" "{\"script_name\":\"$SCRIPT_NAME\",\"description\":\"smoke 更新描述\",\"content\":\"LST DEVICE_INFO;\",\"tags\":[\"smoke\",\"updated\"]}"
    check_ret_ok "更新脚本"

    req GET "/api/v1/mml/scripts/$SCRIPT_ID/runs"
    check_ret_ok "脚本历史执行 runs 可查"

    # start 是 destructive（会向设备派发）→ 只测不存在 ID 负路径，自建脚本绝不 start
    req POST "/api/v1/mml/scripts/$NIL_UUID/start" "{}"
    check_ret_fail "start 不存在脚本被拒绝"

    req DELETE "/api/v1/mml/scripts/$SCRIPT_ID"
    check_ret_ok "删除自建脚本（闭环清理）"
else
    skip "脚本闭环后续步骤" "脚本创建未返回 id"
fi

# ---------------------------------------------------------------------------
section "模板 CRUD 闭环（templates 自定义命令）"
# ---------------------------------------------------------------------------
req GET "/api/v1/mml/templates"
check_ret_ok "模板列表可查"
check_list_or_empty "模板列表" "data.items"

TPL_NAME="${SMOKE_TAG}-tpl"
req POST "/api/v1/mml/templates" "{\"command_name\":\"$TPL_NAME\",\"command_code\":\"LST SMK_${SMOKE_TAG}\",\"operation_type\":\"LST\",\"command_scope\":\"private\",\"category_group\":\"smoke\",\"parameters\":{},\"param_paths\":[\"Device.DeviceInfo.SoftwareVersion\"],\"description\":\"smoke 自建模板\"}"
check_ret_ok "创建私有模板（${TPL_NAME}）"
TPL_ID=$(jget "data.id")
check_field "新模板返回 id" "data.id"

if [ -n "$TPL_ID" ]; then
    req GET "/api/v1/mml/templates/$TPL_ID"
    check_ret_ok "模板详情可查"
    check_field "模板名称回显" "data.command_name"

    req PUT "/api/v1/mml/templates/$TPL_ID" "{\"command_name\":\"$TPL_NAME\",\"command_code\":\"LST SMK_${SMOKE_TAG}\",\"operation_type\":\"LST\",\"command_scope\":\"private\",\"category_group\":\"smoke\",\"parameters\":{},\"param_paths\":[\"Device.DeviceInfo.SoftwareVersion\"],\"description\":\"smoke 更新描述\"}"
    check_ret_ok "更新模板"
    DESC=$(jget "data.description")
    if [ "$DESC" = "smoke 更新描述" ]; then
        pass "模板更新生效（description 回读一致）"
    else
        fail "模板更新生效" "期望 'smoke 更新描述'，实际 '$DESC'"
    fi

    req POST "/api/v1/mml/templates/$TPL_ID/clone"
    check_ret_ok "克隆模板"
    CLONE_ID=$(jget "data.id")
    if [ -n "$CLONE_ID" ] && [ "$CLONE_ID" != "$TPL_ID" ]; then
        pass "克隆产生新 id（${CLONE_ID}）"
        req DELETE "/api/v1/mml/templates/$CLONE_ID"
        check_ret_ok "删除克隆模板（闭环清理）"
    else
        fail "克隆产生新 id" "clone id='$CLONE_ID'"
    fi

    req DELETE "/api/v1/mml/templates/$TPL_ID"
    check_ret_ok "删除自建模板（闭环清理）"

    req GET "/api/v1/mml/templates/$TPL_ID"
    check_ret_fail "删除后模板详情应 404"
else
    skip "模板闭环后续步骤" "模板创建未返回 id"
fi

# 模板创建负路径：缺必填字段 → 400
req POST "/api/v1/mml/templates" "{\"command_name\":\"${SMOKE_TAG}-bad\"}"
check_ret_fail "创建模板缺必填字段被拒绝"

# ---------------------------------------------------------------------------
section "MML 后台治理（admin/standard-params · commands · groups）"
# ---------------------------------------------------------------------------
req GET "/api/v1/mml/admin/standard-params?page=1&page_size=10"
check_ret_ok "standard-params 路径字典可查"
check_list_nonempty "standard-params 非空（builtin 字典）" "data.items"
SP_ID=$(jget "data.items.0.id")
if [ -n "$SP_ID" ]; then
    req GET "/api/v1/mml/admin/standard-params/$SP_ID"
    check_ret_ok "standard-param 详情可查"
    check_field "standard_path 回显" "data.standard_path"
else
    skip "standard-param 详情" "列表未返回 id"
fi

req GET "/api/v1/mml/admin/commands?page=1&page_size=10"
check_ret_ok "admin 命令列表可查"
check_list_nonempty "admin 命令非空（builtin catalog）" "data.items"
ADMIN_CMD_ID=$(jget "data.items.0.id")
# 既有命令的 group_id 是合法 FK（已挂在真实 param_version 下的分组）；自建命令复用它，
# 避开第 2 轮刚修的 param_version FK 422 路径（命令本身无 param_version 字段，FK 仅 group_id）。
ADMIN_CMD_GROUP_ID=$(jget "data.items.0.group_id")
if [ -n "$ADMIN_CMD_ID" ]; then
    req GET "/api/v1/mml/admin/commands/$ADMIN_CMD_ID"
    check_ret_ok "admin 命令详情可查"

    req GET "/api/v1/mml/admin/commands/$ADMIN_CMD_ID/sub-fields"
    check_ret_ok "admin 命令 sub-fields 可查"
else
    skip "admin 命令详情/sub-fields" "列表未返回 id"
fi

req GET "/api/v1/mml/admin/groups"
check_ret_ok "admin 分组列表可查"
check_list_nonempty "admin 分组非空（chapter 章节）" "data.items"
# param_version 有 FK 约束（mml_param_groups_param_version_fkey），必须复用既有版本号
PARAM_VERSION=$(jget "data.items.0.param_version")

# admin group 闭环（admin 来源 group 可删，catalog builtin 不动）
if [ -n "$PARAM_VERSION" ]; then
    GRP_CODE="${SMOKE_TAG}_grp"
    req POST "/api/v1/mml/admin/groups" "{\"group_code\":\"$GRP_CODE\",\"group_name_zh\":\"smoke 分组\",\"group_name_en\":\"smoke group\",\"param_version\":\"$PARAM_VERSION\",\"display_order\":999}"
    check_ret_ok "创建 admin 分组（${GRP_CODE}）"
    GRP_ID=$(jget "data.id")
    if [ -n "$GRP_ID" ]; then
        req PATCH "/api/v1/mml/admin/groups/$GRP_ID" "{\"group_name_zh\":\"smoke 分组改名\"}"
        check_ret_ok "更新 admin 分组"
        req DELETE "/api/v1/mml/admin/groups/$GRP_ID"
        check_ret_ok "删除 admin 分组（闭环清理）"
    else
        skip "admin 分组闭环后续步骤" "分组创建未返回 id"
    fi
else
    skip "admin 分组闭环" "既有分组未带 param_version，无合法 FK 值可用"
fi

# 创建分组负路径：不存在的 param_version → FK violation(23503) 翻业务级 422，
# 不外泄裸 SQL 约束名（issue #125-mml 问题 1 已修复，精确断言 422）
req POST "/api/v1/mml/admin/groups" "{\"group_code\":\"${SMOKE_TAG}_badgrp\",\"param_version\":\"smoke-no-such-version\"}"
check_status "创建分组不存在 param_version 被拒绝（422）" 422
case "$BODY" in
    *"foreign key constraint"*|*"23503"*|*"_fkey"*)
        fail "param_version 错误不外泄裸 SQL" "body 含 SQL 约束细节: $(printf '%s' "$BODY" | head -c 160)" ;;
    *) pass "param_version 错误不外泄裸 SQL（无约束名/SQLSTATE）" ;;
esac

# ---------------------------------------------------------------------------
section "admin/commands CRUD 闭环（建→改→sub-field→删，admin 来源可删）"
# ---------------------------------------------------------------------------
# 自建 admin 来源命令（source=admin, catalog_protected=false → 可删）。command_code
# 全局唯一带 $SMOKE_TAG 自隔离；group_id 复用既有合法 FK（见上方 ADMIN_CMD_GROUP_ID 注释）。
A_CMD_NAME="${SMOKE_TAG} admcmd"
A_CMD_CODE="LST SMK_${SMOKE_TAG}"
A_LOGICAL="SMK_${SMOKE_TAG}"
if [ -n "$ADMIN_CMD_GROUP_ID" ]; then
    A_CMD_BODY="{\"command_name\":\"$A_CMD_NAME\",\"command_code\":\"$A_CMD_CODE\",\"operation_type\":\"LST\",\"group_id\":\"$ADMIN_CMD_GROUP_ID\",\"logical_code\":\"$A_LOGICAL\",\"category\":\"query\",\"description\":\"smoke 自建命令\"}"
else
    # 既有命令无 group_id 时退化为不挂分组（group_id 可空）
    A_CMD_BODY="{\"command_name\":\"$A_CMD_NAME\",\"command_code\":\"$A_CMD_CODE\",\"operation_type\":\"LST\",\"logical_code\":\"$A_LOGICAL\",\"category\":\"query\",\"description\":\"smoke 自建命令\"}"
fi
req POST "/api/v1/mml/admin/commands" "$A_CMD_BODY"
check_ret_ok "创建 admin 命令（${A_CMD_CODE}）"
A_CMD_ID=$(jget "data.id")
check_field "新命令返回 id" "data.id"
A_CMD_SOURCE=$(jget "data.source")
if [ "$A_CMD_SOURCE" = "admin" ]; then
    pass "新命令 source=admin（可删，不动 builtin catalog）"
else
    fail "新命令 source=admin" "实际 source='$A_CMD_SOURCE'"
fi

if [ -n "$A_CMD_ID" ]; then
    req GET "/api/v1/mml/admin/commands/$A_CMD_ID"
    check_ret_ok "新命令详情可查"
    check_field "命令 command_code 回显" "data.command_code"

    # PATCH 改 description（command_code/operation_type 为锁定字段，不改）
    req PATCH "/api/v1/mml/admin/commands/$A_CMD_ID" "{\"description\":\"smoke 更新描述\"}"
    check_ret_ok "更新 admin 命令（description）"
    A_DESC=$(jget "data.description")
    if [ "$A_DESC" = "smoke 更新描述" ]; then
        pass "命令更新生效（description 回读一致）"
    else
        fail "命令更新生效" "期望 'smoke 更新描述'，实际 '$A_DESC'"
    fi

    # sub-field 闭环：取一个真实 standard_param 的 id 作为 param_id，挂一条 sub-field 再删
    req GET "/api/v1/mml/admin/standard-params?page=1&page_size=1"
    SP_PARAM_ID=$(jget "data.items.0.id")
    if [ -n "$SP_PARAM_ID" ]; then
        req POST "/api/v1/mml/admin/commands/$A_CMD_ID/sub-fields" "{\"param_id\":\"$SP_PARAM_ID\",\"mml_code\":\"SMK_FIELD\",\"sort_order\":1}"
        check_ret_ok "命令挂载 sub-field"
        A_SF_ID=$(jget "data.id")
        check_field "新 sub-field 返回 id" "data.id"
        if [ -n "$A_SF_ID" ]; then
            req PATCH "/api/v1/mml/admin/commands/$A_CMD_ID/sub-fields/$A_SF_ID" "{\"mml_code\":\"SMK_FIELD2\"}"
            check_ret_ok "更新 sub-field（mml_code）"
            req DELETE "/api/v1/mml/admin/commands/$A_CMD_ID/sub-fields/$A_SF_ID"
            check_ret_ok "删除 sub-field（闭环清理）"
        else
            skip "sub-field 改/删" "sub-field 创建未返回 id"
        fi
    else
        skip "命令 sub-field 闭环" "standard-params 未返回 id"
    fi

    # DELETE 自建命令（闭环清理）
    req DELETE "/api/v1/mml/admin/commands/$A_CMD_ID"
    check_ret_ok "删除自建 admin 命令（闭环清理）"

    # 删除后 GET：not-found 应翻 404（issue #145 E 已修：IsErrNotFound 纳入
    # commonerrors.ErrNotFound，admin commands GetByID 的裸 sentinel 不再落 500）。
    req GET "/api/v1/mml/admin/commands/$A_CMD_ID"
    check_status "删除后命令详情应 404（not-found）" 404
else
    skip "admin 命令 CRUD 后续步骤" "命令创建未返回 id"
fi

# 创建命令负路径：缺必填 command_code/operation_type → 400（binding required）
req POST "/api/v1/mml/admin/commands" "{\"command_name\":\"${SMOKE_TAG}-badcmd\"}"
check_ret_fail "创建命令缺必填字段被拒绝"

# 创建命令负路径：非法 operation_type → 400（binding oneof=LST MOD ADD RMV）
req POST "/api/v1/mml/admin/commands" "{\"command_name\":\"${SMOKE_TAG}-badop\",\"command_code\":\"DROP X\",\"operation_type\":\"DROP\"}"
check_ret_fail "创建命令非法 operation_type 被拒绝"

# PATCH/DELETE 不存在命令负路径（issue #145 E 已修：not-found 翻 404，不再 500）
req PATCH "/api/v1/mml/admin/commands/$NIL_UUID" "{\"description\":\"x\"}"
check_status "PATCH 不存在命令应 404" 404
req DELETE "/api/v1/mml/admin/commands/$NIL_UUID"
check_status "DELETE 不存在命令应 404" 404
req PATCH "/api/v1/mml/admin/commands/not-a-uuid" "{\"description\":\"x\"}"
check_status "PATCH 命令非法 UUID 被拒绝（400）" 400

# ---------------------------------------------------------------------------
section "脚本治理操作只测负路径（红线：自建脚本绝不 pause/cancel 真实脚本）"
# ---------------------------------------------------------------------------
# pause/cancel 是 destructive（会改真实脚本运行态 / 向设备派发中止）→ 只测不存在脚本 ID。
req POST "/api/v1/mml/scripts/$NIL_UUID/pause" "{}"
check_ret_fail "pause 不存在脚本被拒绝"
req POST "/api/v1/mml/scripts/$NIL_UUID/cancel" "{}"
check_ret_fail "cancel 不存在脚本被拒绝"
req POST "/api/v1/mml/scripts/not-a-uuid/pause" "{}"
check_status "pause 脚本非法 UUID 被拒绝（400）" 400
req POST "/api/v1/mml/scripts/not-a-uuid/cancel" "{}"
check_status "cancel 脚本非法 UUID 被拒绝（400）" 400

# ---------------------------------------------------------------------------
section "execute 系列只测参数校验负路径（红线：不触达任何设备）"
# ---------------------------------------------------------------------------
# 缺 device_sns（binding required）→ 400，发生在任务创建之前
req POST "/api/v1/mml/execute" "{\"command_code\":\"LST DEVICE_INFO\"}"
check_ret_fail "execute 缺 device_sns 被拒绝"

# 有 device_sns 但命令源全空（command_code/script_id/commands/param_paths 均缺）→ 400
req POST "/api/v1/mml/execute" "{\"device_sns\":[\"SMK-NO-SUCH-DEV\"],\"task_name\":\"${SMOKE_TAG}-never\"}"
check_ret_fail "execute 无命令源被拒绝"

# POST /mml/tasks（脚本任务登记）同样的双重校验
req POST "/api/v1/mml/tasks" "{\"task_name\":\"${SMOKE_TAG}-never\"}"
check_ret_fail "tasks 创建缺 device_sns 被拒绝"
req POST "/api/v1/mml/tasks" "{\"device_sns\":[\"SMK-NO-SUCH-DEV\"],\"task_name\":\"${SMOKE_TAG}-never\"}"
check_ret_fail "tasks 创建无命令源被拒绝"

# execute-statements：空 body / 空数组（binding min=1）→ 400
req POST "/api/v1/mml/execute-statements" "{}"
check_ret_fail "execute-statements 空 body 被拒绝"
req POST "/api/v1/mml/execute-statements" "{\"statements\":[],\"device_sns\":[]}"
check_ret_fail "execute-statements 空 statements 被拒绝"

# 结构化执行：空 statements → 400
req POST "/api/v1/mml/console/execute-statements-structured" "{\"statements\":[],\"device_sns\":[]}"
check_ret_fail "structured 执行空 statements 被拒绝"

# 分组批量执行：非法 UUID → 400；不存在分组（全零 UUID）→ 404 + 如实文案
# （issue #125-mml 问题 2 已修复：not-found 翻 404，不再 500 + 误导「group_id required」）
req POST "/api/v1/mml/groups/not-a-uuid/execute" "{\"device_sns\":[\"SMK-NO-SUCH-DEV\"]}"
check_status "groups execute 非法 UUID 被拒绝（400）" 400
req POST "/api/v1/mml/groups/$NIL_UUID/execute" "{\"device_sns\":[\"SMK-NO-SUCH-DEV\"]}"
check_status "groups execute 不存在分组被拒绝（404）" 404
case "$BODY" in
    *"group_id required"*)
        fail "groups execute not-found 文案如实" "仍含误导性 group_id required: $(printf '%s' "$BODY" | head -c 160)" ;;
    *"not found"*) pass "groups execute not-found 文案如实（含 not found）" ;;
    *) pass "groups execute not-found 文案不误导（无 group_id required）" ;;
esac

smoke_summary
