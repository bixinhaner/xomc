#!/usr/bin/env bash
# =============================================================================
# smoke_topology.sh — 拓扑与设备分组(F06) 业务冒烟
#
# 覆盖：
#   - 分组树/统计（设备页强依赖）：/device-groups/tree、/device-groups/stats
#   - 分组 CRUD 闭环：建 L1/L2 → GET → PUT 改名 → sort → check-delete → DELETE
#   - 设备进出组：POST :id/devices 批量加入 → GET 验证 → legacy 单设备出/入 →
#     批量移除 → move-devices 移回原分组（可逆，结束后还原设备归属）
#   - legacy /groups 兼容路由：GET 树 / GET :id / 单设备加入移除
#   - 站点：GET /sites、GET /sites/:id（自备数据缺失时打负路径）
#   - 拓扑图读链路：nodes/edges/graph/geo/statistics 全读
#   - 拓扑节点/边 CRUD 闭环（自建自删，${SMOKE_TAG} 命名）
#   - 负路径：非法 UUID / 缺必填字段 / 不存在 ID / sync 非法参数
#
# 危险禁区：POST /topology/sync 与 /topology/nodes/batch 会批量写拓扑节点，
#           只测参数校验负路径，不真实触发同步。
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "拓扑与设备分组(F06)" "$@"
smoke_login

NOEXIST_ID="00000000-0000-0000-0000-00000000dead"

# ---------------------------------------------------------------------------
section "分组树与统计（设备页强依赖）"
# ---------------------------------------------------------------------------
req GET "/api/v1/device-groups/tree"
check_ret_ok "分组树可查(GET /device-groups/tree)"
check_field "分组树带统计对象" "data.stats.total_groups"

req GET "/api/v1/device-groups/stats"
check_ret_ok "分组统计可查(GET /device-groups/stats)"
check_field "统计含 total_groups" "data.total_groups"
check_field "统计含 ungrouped_devices" "data.ungrouped_devices"

# ---------------------------------------------------------------------------
section "分组 CRUD 闭环（建 L1 → 建 L2 → 查 → 改名 → 排序）"
# ---------------------------------------------------------------------------
L1_NAME="${SMOKE_TAG}-L1"
L2_NAME="${SMOKE_TAG}-L2"
L1_ID=""
L2_ID=""

req POST "/api/v1/device-groups" "{\"name\":\"$L1_NAME\",\"remark\":\"smoke 自建 L1，跑完即删\"}"
check_ret_ok "创建 L1 分组"
L1_ID=$(jget data.id)
check_field "L1 分组返回 id" "data.id"

if [ -n "$L1_ID" ]; then
    req POST "/api/v1/device-groups" "{\"name\":\"$L2_NAME\",\"parent_id\":\"$L1_ID\",\"remark\":\"smoke 自建 L2\"}"
    check_ret_ok "创建 L2 子分组(parent_id=L1)"
    L2_ID=$(jget data.id)
    check_field "L2 分组返回 id" "data.id"
    lvl=$(jget data.level)
    if [ "$lvl" = "2" ]; then pass "L2 分组 level=2"
    else fail "L2 分组 level=2" "实际 level='$lvl'"; fi
else
    skip "创建 L2 子分组" "L1 创建失败，无 parent_id 可用"
fi

if [ -n "$L2_ID" ]; then
    req GET "/api/v1/device-groups/$L2_ID"
    check_ret_ok "按 ID 查询 L2 分组"
    v=$(jget data.name)
    if [ "$v" = "$L2_NAME" ]; then pass "L2 分组名称回读一致"
    else fail "L2 分组名称回读一致" "期望 ${L2_NAME}，实际 '$v'"; fi

    req PUT "/api/v1/device-groups/$L2_ID" "{\"name\":\"${L2_NAME}-renamed\"}"
    check_ret_ok "PUT 改名 L2 分组"
    req GET "/api/v1/device-groups/$L2_ID"
    v=$(jget data.name)
    if [ "$v" = "${L2_NAME}-renamed" ]; then pass "改名后名称回读一致"
    else fail "改名后名称回读一致" "期望 ${L2_NAME}-renamed，实际 '$v'"; fi

    # 批量排序正路径：借自建 L1/L2 两个临时分组（结束删除），覆盖多分支 CASE WHEN。
    # 回归 #120：占位符缺 ::int/::uuid 转型时 PG 推断为 text → SQLSTATE 42804 恒 500。
    req PUT "/api/v1/device-groups/sort" "{\"items\":[{\"id\":\"$L1_ID\",\"sort_order\":8},{\"id\":\"$L2_ID\",\"sort_order\":9}]}"
    check_ret_ok "批量排序(PUT /device-groups/sort)"
    req GET "/api/v1/device-groups/$L2_ID"
    v=$(jget data.sort_order)
    if [ "$v" = "9" ]; then pass "排序后 L2 sort_order 回读=9"
    else fail "排序后 L2 sort_order 回读=9" "实际 '$v'"; fi
else
    skip "L2 查询/改名/排序" "L2 创建失败"
fi

# 建组后树/统计强断言（此时必有自建分组）
req GET "/api/v1/device-groups/tree"
check_list_nonempty "建组后分组树非空" "data.items"
req GET "/api/v1/device-groups/stats"
check_count_ge "建组后 total_groups ≥ 2" "data.total_groups" 2

# ---------------------------------------------------------------------------
section "legacy /groups 兼容路由"
# ---------------------------------------------------------------------------
req GET "/api/v1/groups"
check_ret_ok "legacy 分组树可查(GET /groups)"
check_list_nonempty "legacy 树含自建分组" "data.items"

if [ -n "$L2_ID" ]; then
    req GET "/api/v1/groups/$L2_ID"
    check_ret_ok "legacy 按 ID 查询分组(GET /groups/:id)"
    v=$(jget data.name)
    if [ "$v" = "${L2_NAME}-renamed" ]; then pass "legacy 路由与主路由返回同一分组"
    else fail "legacy 路由与主路由返回同一分组" "期望 ${L2_NAME}-renamed，实际 '$v'"; fi
else
    skip "legacy 按 ID 查询分组" "L2 创建失败"
fi

# ---------------------------------------------------------------------------
section "设备进出组（可逆：结束后还原设备原归属）"
# ---------------------------------------------------------------------------
DEV_ID=""
DEV_SN=""
req GET "/api/v1/devices?page=1&page_size=10"
check_ret_ok "设备列表可查（取测试设备）"
DEV_ID=$(jget data.items.0.id)
DEV_SN=$(jget data.items.0.serial_number)

ORIG_GROUP=""
if [ -n "$DEV_ID" ] && [ -n "$L2_ID" ]; then
    echo "  使用设备: sn=${DEV_SN} id=${DEV_ID}"
    # 记录设备当前归属分组（device_group_members 一设备一组），用于结束后还原
    req GET "/api/v1/device-groups/tree"
    GROUP_IDS=$(printf '%s' "$BODY" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
except Exception:
    sys.exit(0)
def walk(nodes):
    for n in nodes or []:
        print(n.get('id', ''))
        walk(n.get('children'))
walk((d.get('data') or {}).get('items'))
" 2>/dev/null)
    for gid in $GROUP_IDS; do
        [ -z "$gid" ] && continue
        req GET "/api/v1/device-groups/$gid/devices"
        case "$BODY" in
            *"$DEV_ID"*) ORIG_GROUP="$gid"; break ;;
        esac
    done
    if [ -n "$ORIG_GROUP" ]; then
        echo "  设备原归属分组: ${ORIG_GROUP}（结束后移回）"
    else
        echo "  设备原为未分组状态（结束后还原为未分组）"
    fi

    # 1) 批量加入 L2 组
    req POST "/api/v1/device-groups/$L2_ID/devices" "{\"device_ids\":[\"$DEV_ID\"]}"
    check_ret_ok "批量加入设备(POST :id/devices)"
    check_field "加入返回 affected" "data.affected"

    # 2) 验证成员
    req GET "/api/v1/device-groups/$L2_ID/devices"
    check_ret_ok "查询分组成员(GET :id/devices)"
    check_count_ge "分组成员 total ≥ 1" "data.total" 1
    case "$BODY" in
        *"$DEV_ID"*) pass "成员列表包含测试设备" ;;
        *) fail "成员列表包含测试设备" "device_ids 中未找到 ${DEV_ID}" ;;
    esac

    # 3) legacy 单设备移除
    req DELETE "/api/v1/groups/$L2_ID/devices/$DEV_ID"
    check_ret_ok "legacy 单设备移除(DELETE /groups/:id/devices/:deviceId)"
    req GET "/api/v1/device-groups/$L2_ID/devices"
    v=$(jget data.total)
    if [ "$v" = "0" ]; then pass "移除后分组成员归零"
    else fail "移除后分组成员归零" "期望 total=0，实际 '$v'"; fi

    # 4) legacy 单设备加入
    req POST "/api/v1/groups/$L2_ID/devices" "{\"device_id\":\"$DEV_ID\"}"
    check_ret_ok "legacy 单设备加入(POST /groups/:id/devices)"

    # 5) 批量移除
    req DELETE "/api/v1/device-groups/$L2_ID/devices" "{\"device_ids\":[\"$DEV_ID\"]}"
    check_ret_ok "批量移除设备(DELETE :id/devices)"

    # 6) 还原设备原归属
    if [ -n "$ORIG_GROUP" ]; then
        req POST "/api/v1/device-groups/move-devices" "{\"device_ids\":[\"$DEV_ID\"],\"target_group_id\":\"$ORIG_GROUP\"}"
        check_ret_ok "move-devices 移回原分组(POST /device-groups/move-devices)"
        req GET "/api/v1/device-groups/$ORIG_GROUP/devices"
        case "$BODY" in
            *"$DEV_ID"*) pass "设备已还原至原分组" ;;
            *) fail "设备已还原至原分组" "原分组 ${ORIG_GROUP} 成员中未找到设备" ;;
        esac
    else
        pass "设备原为未分组，批量移除后已还原为未分组"
    fi
else
    if [ -z "$DEV_ID" ]; then
        skip "设备进出组闭环" "活栈无设备可用（GET /devices 为空），无法自建真实设备"
    else
        skip "设备进出组闭环" "L2 分组创建失败，无目标分组"
    fi
fi

# move-devices 负路径（不依赖设备）
req POST "/api/v1/device-groups/move-devices" "{\"device_ids\":[\"$NOEXIST_ID\"],\"target_group_id\":\"not-a-uuid\"}"
check_ret_fail "move-devices 非法 target_group_id 被拒绝"

# ---------------------------------------------------------------------------
section "分组删除链（check-delete → DELETE → 验证 404）"
# ---------------------------------------------------------------------------
if [ -n "$L2_ID" ]; then
    req GET "/api/v1/device-groups/$L2_ID/check-delete"
    check_ret_ok "L2 删除预检(GET :id/check-delete)"
    check_field "预检返回 can_delete" "data.can_delete"

    req DELETE "/api/v1/device-groups/$L2_ID"
    check_ret_ok "删除 L2 分组"

    req GET "/api/v1/device-groups/$L2_ID"
    check_ret_fail "已删 L2 分组再查询被拒绝(404)"
else
    skip "L2 删除链" "L2 创建失败"
fi

if [ -n "$L1_ID" ]; then
    req GET "/api/v1/device-groups/$L1_ID/check-delete"
    check_ret_ok "L1 删除预检"

    req DELETE "/api/v1/device-groups/$L1_ID"
    check_ret_ok "删除 L1 分组"
else
    skip "L1 删除链" "L1 创建失败"
fi

# 分组负路径
req POST "/api/v1/device-groups" "{}"
check_ret_fail "创建分组缺 name 被拒绝"
req GET "/api/v1/device-groups/not-a-uuid"
check_ret_fail "非法分组 ID 被拒绝"
req DELETE "/api/v1/device-groups/$NOEXIST_ID"
check_ret_fail "删除不存在分组被拒绝"

# ---------------------------------------------------------------------------
section "站点 sites"
# ---------------------------------------------------------------------------
req GET "/api/v1/sites"
check_list_or_empty "站点列表可查(GET /sites)" "data.items"
SITE_ID=$(jget data.items.0.id)

if [ -n "$SITE_ID" ]; then
    req GET "/api/v1/sites/$SITE_ID"
    check_ret_ok "站点详情可查(GET /sites/:id)"
    check_field "站点详情含 name" "data.name"
else
    req GET "/api/v1/sites/$NOEXIST_ID"
    check_ret_fail "不存在站点详情被拒绝(无站点数据，打负路径覆盖 :id 路由)"
fi

# POST /sites 无 DELETE 配套路由，自建会留残留 → 只测参数校验负路径
req POST "/api/v1/sites" "{\"address\":\"smoke-no-name\"}"
check_ret_fail "创建站点缺 name 被拒绝"

# ---------------------------------------------------------------------------
section "拓扑图读链路 nodes/edges/graph/geo/statistics"
# ---------------------------------------------------------------------------
req GET "/api/v1/topology/nodes"
check_list_or_empty "拓扑节点列表可查(GET /topology/nodes)" "data.items"

req GET "/api/v1/topology/edges"
check_list_or_empty "拓扑边列表可查(GET /topology/edges)" "data.items"

req GET "/api/v1/topology/graph"
check_ret_ok "拓扑图可查(GET /topology/graph)"

req GET "/api/v1/topology/graph?layout_type=force&limit=50"
check_ret_ok "拓扑图带布局参数可查(layout_type=force&limit=50)"

req GET "/api/v1/topology/geo"
check_ret_ok "地理拓扑可查(GET /topology/geo)"

req GET "/api/v1/topology/statistics"
check_ret_ok "拓扑统计可查(GET /topology/statistics)"
check_field "拓扑统计含 total_nodes" "data.total_nodes"

# ---------------------------------------------------------------------------
section "拓扑节点/边 CRUD 闭环（自建自删）"
# ---------------------------------------------------------------------------
NODE_A=""
NODE_B=""
EDGE_ID=""

req POST "/api/v1/topology/nodes" "{\"label\":\"${SMOKE_TAG}-node-a\",\"node_type\":\"eNB\",\"x\":10,\"y\":20,\"status\":\"online\"}"
check_ret_ok "创建拓扑节点 A"
NODE_A=$(jget data.id)
check_field "节点 A 返回 id" "data.id"

req POST "/api/v1/topology/nodes" "{\"label\":\"${SMOKE_TAG}-node-b\",\"node_type\":\"gNB\",\"x\":30,\"y\":40,\"status\":\"offline\"}"
check_ret_ok "创建拓扑节点 B"
NODE_B=$(jget data.id)

if [ -n "$NODE_A" ]; then
    req GET "/api/v1/topology/nodes/$NODE_A"
    check_ret_ok "节点详情可查(GET /topology/nodes/:id)"
    v=$(jget data.label)
    if [ "$v" = "${SMOKE_TAG}-node-a" ]; then pass "节点 label 回读一致"
    else fail "节点 label 回读一致" "期望 ${SMOKE_TAG}-node-a，实际 '$v'"; fi

    req PUT "/api/v1/topology/nodes/$NODE_A" "{\"label\":\"${SMOKE_TAG}-node-a2\"}"
    check_ret_ok "PUT 更新节点 label"
    v=$(jget data.label)
    if [ "$v" = "${SMOKE_TAG}-node-a2" ]; then pass "更新后 label 回读一致"
    else fail "更新后 label 回读一致" "期望 ${SMOKE_TAG}-node-a2，实际 '$v'"; fi
else
    skip "节点详情/更新" "节点 A 创建失败"
fi

# 建节点后强断言列表与统计
req GET "/api/v1/topology/nodes"
check_list_nonempty "建节点后拓扑节点列表非空" "data.items"
req GET "/api/v1/topology/statistics"
check_count_ge "建节点后 total_nodes ≥ 2" "data.total_nodes" 2

if [ -n "$NODE_A" ] && [ -n "$NODE_B" ]; then
    req POST "/api/v1/topology/edges" "{\"source_id\":\"$NODE_A\",\"target_id\":\"$NODE_B\",\"label\":\"${SMOKE_TAG}-edge\",\"status\":\"active\"}"
    check_ret_ok "创建拓扑边 A→B"
    EDGE_ID=$(jget data.id)
    check_field "边返回 id" "data.id"
else
    skip "拓扑边 CRUD" "节点 A/B 创建失败"
fi

if [ -n "$EDGE_ID" ]; then
    req GET "/api/v1/topology/edges/$EDGE_ID"
    check_ret_ok "边详情可查(GET /topology/edges/:id)"

    req PUT "/api/v1/topology/edges/$EDGE_ID" "{\"status\":\"inactive\"}"
    check_ret_ok "PUT 更新边状态"
    v=$(jget data.status)
    if [ "$v" = "inactive" ]; then pass "边状态回读 inactive"
    else fail "边状态回读 inactive" "实际 '$v'"; fi

    req DELETE "/api/v1/topology/edges/$EDGE_ID"
    check_ret_ok "删除拓扑边"
fi

# 清理节点（边已删/或边未建成时直接删）
if [ -n "$NODE_A" ]; then
    req DELETE "/api/v1/topology/nodes/$NODE_A"
    check_ret_ok "删除拓扑节点 A"
    req GET "/api/v1/topology/nodes/$NODE_A"
    check_ret_fail "已删节点再查询被拒绝(404)"
fi
if [ -n "$NODE_B" ]; then
    req DELETE "/api/v1/topology/nodes/$NODE_B"
    check_ret_ok "删除拓扑节点 B"
fi

# ---------------------------------------------------------------------------
section "拓扑写接口参数校验负路径（不真实触发同步/批建）"
# ---------------------------------------------------------------------------
req POST "/api/v1/topology/nodes" "{\"label\":\"${SMOKE_TAG}-bad\",\"node_type\":\"bad-type\"}"
check_ret_fail "创建节点非法 node_type 被拒绝"

req POST "/api/v1/topology/edges" "{\"source_id\":\"not-a-uuid\",\"target_id\":\"$NOEXIST_ID\"}"
check_ret_fail "创建边非法 source_id 被拒绝"

req POST "/api/v1/topology/sync?domain_id=not-a-uuid"
check_ret_fail "拓扑同步非法 domain_id 被拒绝（不真实触发同步）"

req GET "/api/v1/topology/statistics?domain_id=not-a-uuid"
check_ret_fail "拓扑统计非法 domain_id 被拒绝"

smoke_summary
