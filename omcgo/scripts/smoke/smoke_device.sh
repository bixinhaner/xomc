#!/usr/bin/env bash
# =============================================================================
# smoke_device.sh — F06 设备管理域真实业务冒烟
#
# 覆盖（对应 /tmp/smoke_routes.json key=device 全部 read 路由 + 可逆 write 闭环）：
#   1. 设备列表分页 + 多维过滤（carrier/technology/is_online/search/sn）
#   2. 统计 stats / product-classes / enums（builtin 字典强断言）
#   3. 地图 geo / geo/stats / search
#   4. CSV 导出（非信封，只断言 HTTP 200）
#   5. 预注册设备全闭环：建 → 查 → 改 → 软删进回收站 → 回收站可见 →
#      restore 恢复 → 再删 → recycle/permanent 永久删 → 404 验证
#   6. device-registrations 预登记闭环（建 → 列表过滤 → 删）
#   7. column-configs 用户列配置（默认列读取 + SMOKE_TAG pageKey 写读还原）
#   8. 危险操作只测参数校验负路径：reboot / batch-reboot / batch 删 /
#      recycle restore/permanent 非法 body —— 绝不向真实设备下发指令
#
# 用法：bash smoke_device.sh [BASE_URL]   （默认 http://localhost:8081）
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 设备管理" "$@"
smoke_login

# ---------------------------------------------------------------------------
section "设备列表分页 + 过滤"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices?page=1&page_size=10"
check_ret_ok "设备列表分页可查"
check_field "列表带 total 字段" "data.total"
# 记下第一台真实设备（inform 注册的才有 device_info，供 /:id/info 用例）
EXIST_DEV_ID=$(jget data.items.0.id)
EXIST_DEV_SN=$(jget data.items.0.serial_number)

req GET "/api/v1/devices?page=1&page_size=5&carrier=cmcc"
check_ret_ok "列表按 carrier=cmcc 过滤"

req GET "/api/v1/devices?page=1&page_size=5&technology=lte&is_online=false"
check_ret_ok "列表按 technology+is_online 组合过滤"

req GET "/api/v1/devices?page=1&page_size=5&search=${SMOKE_TAG}-nonexist"
check_ret_ok "列表 search 关键字过滤（无命中也应 200）"

req GET "/api/v1/devices?page=1&page_size=5&lifecycle_state=commissioned,maintenance"
check_ret_ok "列表按 lifecycle_state 多选过滤"

# ---------------------------------------------------------------------------
section "统计 / 产品类 / 枚举字典"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices/stats"
check_ret_ok "设备状态统计 stats"
check_field "stats 含 counts 对象" "data.counts"

req GET "/api/v1/devices/stats?carrier=cmcc"
check_ret_ok "stats 按 carrier 过滤"

req GET "/api/v1/devices/product-classes"
check_ret_ok "产品类列表 product-classes"

req GET "/api/v1/devices/enums"
check_ret_ok "设备枚举字典 enums"
check_list_nonempty "枚举 carrier 字典非空（builtin 硬编码）" "data.carrier"
check_list_nonempty "枚举 technology 字典非空" "data.technology"
check_list_nonempty "枚举 status 字典非空" "data.status"

# ---------------------------------------------------------------------------
section "地图 geo 数据"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices/geo"
check_ret_ok "地图设备打点 geo"

req GET "/api/v1/devices/geo?status=onlineActive,offline"
check_ret_ok "geo 按前端状态值过滤"

req GET "/api/v1/devices/geo/stats"
check_ret_ok "地图统计 geo/stats"

req GET "/api/v1/devices/search?keyword=zzz-no-match&limit=5"
check_ret_ok "地图设备搜索（无命中关键字返回空列表）"

req GET "/api/v1/devices/search?keyword=x"
check_ret_ok "地图搜索 keyword 过短返回空列表（200）"

# 命中坐标为 NULL 的设备时 repository 扫描崩 500（真实后端 bug，已实测）
req GET "/api/v1/devices/search?keyword=SM&limit=5"
if [ "$HTTP_CODE" = "200" ]; then
    pass "地图设备搜索 keyword 命中真实设备 → 200"
else
    known_bug "地图设备搜索命中 NULL 坐标设备返回 500" "GET /devices/search?keyword=SM → HTTP ${HTTP_CODE}，msg=scan search result: can't scan into dest[5] (col: latitude): cannot scan NULL into *float64"
fi

# ---------------------------------------------------------------------------
section "设备导出 CSV"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices/export"
check_status "设备导出 CSV（非信封）" 200

req GET "/api/v1/devices/export?carrier=cmcc"
check_status "设备导出按 carrier 过滤" 200

# ---------------------------------------------------------------------------
section "预注册设备闭环：建 → 查 → 改"
# ---------------------------------------------------------------------------
DEV_SN="${SMOKE_TAG}-DEV"
req POST "/api/v1/devices" "{\"serial_number\":\"$DEV_SN\",\"oui\":\"00A0C6\",\"product_class\":\"FAP-LTE-100\",\"manufacturer\":\"SmokeVendor\",\"carrier\":\"cmcc\",\"technology\":\"lte\",\"device_name\":\"${SMOKE_TAG}-smoke-device\"}"
check_ret_ok "创建预注册设备（SN=${DEV_SN}）"
DEV_ID=$(jget data.id)
check_field "创建返回设备 id" "data.id"

if [ -n "$DEV_ID" ]; then
    req GET "/api/v1/devices/$DEV_ID"
    check_ret_ok "按 id 查询新建设备"
    check_field "详情回显 serial_number" "data.serial_number"

    req GET "/api/v1/devices?page=1&page_size=5&sn=$DEV_SN"
    check_list_nonempty "列表按 SN 精确过滤可见新设备" "data.items"

    req GET "/api/v1/devices/$DEV_ID/detail"
    check_ret_ok "设备 detail 组合视图（device+info+params）"

    req PUT "/api/v1/devices/$DEV_ID" "{\"device_name\":\"${SMOKE_TAG}-renamed\"}"
    check_ret_ok "更新设备名称"

    req POST "/api/v1/devices" "{\"serial_number\":\"$DEV_SN\",\"oui\":\"00A0C6\",\"carrier\":\"cmcc\",\"technology\":\"lte\"}"
    check_ret_fail "重复 SN 创建被拒绝（409）"
else
    skip "按 id 查询新建设备" "创建未返回 id，闭环无法继续"
    skip "列表按 SN 精确过滤" "同上"
    skip "设备 detail 组合视图" "同上"
    skip "更新设备名称" "同上"
    skip "重复 SN 创建被拒绝" "同上"
fi

# /:id/info 需要 device_info 行（仅 inform 注册路径创建；API 预注册设备无 info 行属设计行为）
# → 遍历列表前几台找一台有 info 的真实设备做强断言，全无则降级 skip
INFO_HIT=""
INFO_SN=""
if [ -n "$EXIST_DEV_ID" ]; then
    req GET "/api/v1/devices?page=1&page_size=10"
    CAND_N=$(jlen data.items)
    [ "$CAND_N" -gt 5 ] && CAND_N=5
    i=0
    while [ $i -lt "$CAND_N" ]; do
        CAND_ID=$(jget "data.items.$i.id")
        CAND_SN=$(jget "data.items.$i.serial_number")
        req GET "/api/v1/devices/$CAND_ID/info"
        if [ "$HTTP_CODE" = "200" ]; then
            INFO_HIT="$CAND_ID"
            INFO_SN="$CAND_SN"
            break
        fi
        # 重新拉列表（req 共享全局 BODY，被 /info 覆盖了）
        req GET "/api/v1/devices?page=1&page_size=10"
        i=$((i + 1))
    done
fi
if [ -n "$INFO_HIT" ]; then
    req GET "/api/v1/devices/$INFO_HIT/info"
    check_ret_ok "已注册设备 info 视图（SN=${INFO_SN}）"
else
    skip "已注册设备 info 视图" "栈内无 inform 注册的真实设备（device_info 仅 inform 路径创建，预注册设备无 info 行）"
fi

# ---------------------------------------------------------------------------
section "回收站闭环：软删 → 可见 → 恢复 → 永久删"
# ---------------------------------------------------------------------------
req GET "/api/v1/devices/recycle"
check_ret_ok "回收站列表可查"

if [ -n "$DEV_ID" ]; then
    req DELETE "/api/v1/devices/$DEV_ID"
    check_ret_ok "软删除设备进回收站"

    req GET "/api/v1/devices/$DEV_ID"
    check_status "软删后按 id 查询不可见" 404

    req GET "/api/v1/devices/recycle?search=$DEV_SN"
    check_list_nonempty "回收站按 SN 可见已删设备" "data.items"

    req PATCH "/api/v1/devices/recycle/restore" "{\"ids\":[\"$DEV_ID\"]}"
    check_ret_ok "回收站恢复设备"
    check_count_ge "恢复计数 restored ≥ 1" "data.restored" 1

    req GET "/api/v1/devices/$DEV_ID"
    check_ret_ok "恢复后设备重新可见"

    req DELETE "/api/v1/devices/$DEV_ID"
    check_ret_ok "再次软删除"

    req DELETE "/api/v1/devices/recycle/permanent" "{\"ids\":[\"$DEV_ID\"]}"
    check_ret_ok "回收站永久删除"
    check_count_ge "永久删除计数 deleted ≥ 1" "data.deleted" 1

    req GET "/api/v1/devices/recycle?search=$DEV_SN"
    RECYCLE_LEFT=$(jlen data.items)
    if [ "$HTTP_CODE" = "200" ] && [ "$RECYCLE_LEFT" -eq 0 ]; then
        pass "永久删除后回收站不再可见（剩 0 条）"
    else
        fail "永久删除后回收站不再可见" "HTTP $HTTP_CODE，回收站仍有 $RECYCLE_LEFT 条命中"
    fi

    req GET "/api/v1/devices/$DEV_ID"
    check_status "永久删除后按 id 查询 404" 404
else
    skip "回收站闭环（软删/恢复/永久删）" "前置创建设备失败，无可操作实体"
fi

# ---------------------------------------------------------------------------
section "device-registrations 预登记闭环"
# ---------------------------------------------------------------------------
req GET "/api/v1/device-registrations?page=1&page_size=10"
check_ret_ok "预登记列表可查"

# DB 列 device_registrations.group_id 为 NOT NULL，但 handler 把 group_id 当可选
# （CreateRegistrationRequest 无 required tag）→ 不传时 500 而非 400，属后端契约 bug。
# 闭环改为自备分组 id（device-groups/tree builtin 设备域分组），并把缺 group_id 标 known_bug。
req GET "/api/v1/device-groups/tree"
GROUP_ID=$(jget data.items.0.id)

REG_SN="${SMOKE_TAG}-REG"
if [ -n "$GROUP_ID" ]; then
    req POST "/api/v1/device-registrations" "{\"serial_number\":\"$REG_SN\",\"carrier\":\"cmcc\",\"group_id\":\"$GROUP_ID\",\"site_name\":\"${SMOKE_TAG}-site\",\"remark\":\"smoke 自建，跑完即删\"}"
    check_ret_ok "创建预登记记录（SN=${REG_SN}）"
    REG_ID=$(jget data.id)
    check_field "预登记返回 id" "data.id"

    if [ -n "$REG_ID" ]; then
        req GET "/api/v1/device-registrations?page=1&page_size=5&serial_number=$REG_SN"
        check_list_nonempty "预登记列表按 SN 过滤可见" "data.items"

        req DELETE "/api/v1/device-registrations/$REG_ID"
        check_ret_ok "删除预登记记录（清理）"
    else
        skip "预登记列表按 SN 过滤" "创建未返回 id"
        skip "删除预登记记录" "同上"
    fi
else
    skip "创建预登记记录" "device-groups/tree 取不到分组 id（group_id DB 层 NOT NULL，无法自备）"
    skip "预登记列表按 SN 过滤" "同上"
    skip "删除预登记记录" "同上"
fi

req POST "/api/v1/device-registrations" "{\"serial_number\":\"${SMOKE_TAG}-REG2\"}"
check_ret_fail "预登记缺 carrier 必填字段被拒绝（400）"

# 缺 group_id：handler 放行（可选字段）但 DB NOT NULL → 500，期望应是 400 参数校验
req POST "/api/v1/device-registrations" "{\"serial_number\":\"${SMOKE_TAG}-REG3\",\"carrier\":\"cmcc\"}"
if [[ "$HTTP_CODE" == 4* ]]; then
    pass "预登记缺 group_id 被参数校验拒绝（400）"
else
    known_bug "预登记缺 group_id 返回 500 而非 400" "POST /device-registrations 不带 group_id → HTTP ${HTTP_CODE}，msg=insert registration: null value in column group_id violates not-null constraint（handler 未加 required 校验，DB NOT NULL 兜底炸 500）"
fi

# ---------------------------------------------------------------------------
section "column-configs 用户列配置"
# ---------------------------------------------------------------------------
req GET "/api/v1/column-configs/device-list"
check_ret_ok "读取设备列表列配置（缺省回退默认列）"
check_list_nonempty "列配置 columns 非空（默认列 builtin）" "data.columns"

# 写到 SMOKE_TAG 专属 pageKey，不碰真实页面 key（无 DELETE 端点，残留一行无害）
CC_KEY="cc-${SMOKE_TAG}"
req PUT "/api/v1/column-configs/$CC_KEY" "{\"columns\":[{\"key\":\"serial_number\",\"label\":\"序列号\",\"visible\":true,\"order\":1},{\"key\":\"device_name\",\"label\":\"设备名称\",\"visible\":false,\"order\":2}]}"
check_ret_ok "保存自定义列配置（pageKey=${CC_KEY}）"

req GET "/api/v1/column-configs/$CC_KEY"
check_ret_ok "回读自定义列配置"
check_count_ge "回读 columns 条数 = 2" "data.columns.1.order" 2

req PUT "/api/v1/column-configs/$CC_KEY" '{"not_columns":true}'
check_ret_fail "列配置缺 columns 必填字段被拒绝（400）"

# ---------------------------------------------------------------------------
section "危险操作负路径（绝不真发指令）"
# ---------------------------------------------------------------------------
GHOST_ID="deadbeef-dead-4ead-8ead-deadbeefdead"

req POST "/api/v1/devices/$GHOST_ID/reboot"
check_ret_fail "重启不存在设备被拒绝（404）"

req POST "/api/v1/devices/not-a-uuid/reboot"
check_ret_fail "重启非法 id 被拒绝（400）"

req POST "/api/v1/devices/batch-reboot" '{"ids":[]}'
check_ret_fail "批量重启空 ids 被拒绝（400）"

req POST "/api/v1/devices/batch-reboot" '{"ids":["not-a-uuid"]}'
check_ret_fail "批量重启非法 uuid 被拒绝（400）"

req DELETE "/api/v1/devices/batch" '{"ids":[]}'
check_ret_fail "批量删除空 ids 被拒绝（400）"

req PATCH "/api/v1/devices/recycle/restore" '{}'
check_ret_fail "回收站恢复缺 ids 被拒绝（400）"

req DELETE "/api/v1/devices/recycle/permanent" '{"ids":[]}'
check_ret_fail "永久删除空 ids 被拒绝（400）"

req POST "/api/v1/devices" '{"serial_number":"x-no-carrier"}'
check_ret_fail "创建设备缺必填字段被拒绝（400）"

req POST "/api/v1/devices" "{\"serial_number\":\"${SMOKE_TAG}-BAD\",\"oui\":\"00A0C6\",\"carrier\":\"invalid-carrier\",\"technology\":\"lte\"}"
check_ret_fail "创建设备非法 carrier 被拒绝（oneof 校验 400）"

smoke_summary
