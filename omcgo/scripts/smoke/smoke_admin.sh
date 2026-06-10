#!/usr/bin/env bash
# =============================================================================
# smoke_admin.sh — F06 RBAC（用户/角色/菜单/权限/审计/API端点/API Key/死信）业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=admin 全部路由）：
#   - 全读：users / roles(+all) / menus(列表+tree+user-tree) / permissions /
#     audit-logs / api-endpoints(+groups) / dead-letters / api-keys
#   - 用户 CRUD 闭环：POST 建（密码 RSA-OAEP 加密传输）→ GET 详情 → PUT 改昵称
#     → lock(禁用) → unlock(启用) → DELETE → GET 404
#   - 角色闭环：建 → 绑两个菜单(PUT :id/menus) → GET :id/menus 验证 →
#     绑 API 权限(PUT :id/api-permissions) → copy → 删 copy → 删原角色
#   - API Key 闭环：POST 签发 → GET 列表含它 → DELETE 吊销 → revoked_at 置位
#   - api-endpoints sync（upsert-only 幂等，安全）
#   - 死信队列：列表 + 不存在 ID 负路径（replay 只测负路径）
#   - 负路径：缺分页 400、缺密码 400、非法 UUID 400、缺 name 400
#
# 红线遵守：绝不修改/锁定/删除 admin、system 等内置用户与内置角色；
#           所有写操作仅作用于 ${SMOKE_TAG} 前缀的自建实体，结束前清理。
#
# 用法：bash smoke_admin.sh [BASE_URL]   （默认 http://localhost:8081，本机容器栈用 :18091）
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "F06 RBAC" "$@"
smoke_login

# 在 data.items 中按 id 查找下标（找不到输出空），bash 3.2 兼容
find_item_idx_by_id() {
    local target="$1" n i v
    n=$(jlen data.items)
    i=0
    while [ "$i" -lt "$n" ]; do
        v=$(jget "data.items.$i.id")
        if [ "$v" = "$target" ]; then echo "$i"; return 0; fi
        i=$((i + 1))
    done
    echo ""
}

NIL_UUID="00000000-0000-0000-0000-000000000000"

# ---------------------------------------------------------------------------
section "用户/角色/菜单全读（builtin 种子，显式分页）"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/users?page=1&page_size=20"
check_list_nonempty "GET /admin/users 用户列表非空（admin 内置）" "data.items"
check_count_ge "用户总数 ≥ 1" "data.total" 1

req GET "/api/v1/admin/users"
check_ret_fail "GET /admin/users 缺分页参数被拒（binding min=1）"

req GET "/api/v1/admin/roles?page=1&page_size=20"
check_list_nonempty "GET /admin/roles 角色列表非空（内置角色）" "data.items"
check_field "角色首条含 id" "data.items.0.id"

req GET "/api/v1/admin/roles/all"
check_list_nonempty "GET /admin/roles/all 不分页下拉非空" "data"

req GET "/api/v1/admin/menus?page=1&page_size=50"
check_list_nonempty "GET /admin/menus 菜单列表非空（内置菜单）" "data.items"

req GET "/api/v1/admin/menus/tree"
check_ret_ok "GET /admin/menus/tree 菜单树可查"
check_list_nonempty "菜单树根节点非空" "data.data"
MENU_ID1=$(jget data.data.0.id)
MENU_ID2=$(jget data.data.1.id)
if [ -z "$MENU_ID2" ]; then
    MENU_ID2=$(jget data.data.0.children.0.id)
fi

req GET "/api/v1/admin/menus/user-tree"
check_ret_ok "GET /admin/menus/user-tree 当前用户菜单树可查"
check_list_nonempty "admin 用户菜单树非空" "data.data"

# ---------------------------------------------------------------------------
section "API 端点清单（读 + 幂等 sync）"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/api-endpoints?page=1&page_size=20"
check_list_or_empty "GET /admin/api-endpoints API 端点列表可查" "data.items"

# sync 是 upsert-only（不删行），幂等安全：从 Gin 路由表同步端点字典
req POST "/api/v1/admin/api-endpoints/sync"
check_ret_ok "POST /admin/api-endpoints/sync 从路由同步"
check_count_ge "sync 扫描路由数 ≥ 1" "data.total" 1

req GET "/api/v1/admin/api-endpoints?page=1&page_size=20"
check_list_nonempty "sync 后 API 端点列表非空" "data.items"
EP_ID1=$(jget data.items.0.id)
EP_ID2=$(jget data.items.1.id)

req GET "/api/v1/admin/api-endpoints/groups"
check_ret_ok "GET /admin/api-endpoints/groups 分组可查"
check_list_nonempty "API 分组非空" "data.data"

# ---------------------------------------------------------------------------
section "用户 CRUD 闭环（建→查→改→锁→解锁→删）"
# ---------------------------------------------------------------------------
SMK_USER="${SMOKE_TAG}user"
SMK_PWD="Smk@Pass123"
# POST /admin/users 密码必须 RSA-OAEP 加密传输（CreateUserHTTPRequest），复用 lib 加密
ENC_PWD=$(encrypt_password "$SMK_PWD")
if [ -z "$ENC_PWD" ] || [ -z "$PUBLIC_KEY_ID" ]; then
    fail "用户创建前置：密码加密" "encrypt_password 或 key_id 为空"
    USER_ID=""
else
    req POST "/api/v1/admin/users" "{\"username\":\"$SMK_USER\",\"encrypted_password\":\"$ENC_PWD\",\"key_id\":\"$PUBLIC_KEY_ID\",\"display_name\":\"冒烟测试用户\",\"description\":\"smoke ${SMOKE_TAG}\"}"
    check_ret_ok "POST /admin/users 创建用户 ${SMK_USER}"
    check_field "创建返回用户 id" "data.id"
    USER_ID=$(jget data.id)
fi

if [ -n "$USER_ID" ]; then
    req GET "/api/v1/admin/users/$USER_ID"
    check_ret_ok "GET /admin/users/:id 用户详情可查"
    UV=$(jget data.username)
    if [ "$UV" = "$SMK_USER" ]; then pass "详情 username 回读一致 ($UV)"
    else fail "详情 username 回读一致" "期望 ${SMK_USER}，实际 ${UV}"; fi

    req PUT "/api/v1/admin/users/$USER_ID" "{\"display_name\":\"冒烟改名${SMOKE_TAG}\"}"
    check_ret_ok "PUT /admin/users/:id 修改昵称"
    DV=$(jget data.display_name)
    if [ "$DV" = "冒烟改名${SMOKE_TAG}" ]; then pass "昵称已更新 ($DV)"
    else fail "昵称已更新" "期望 冒烟改名${SMOKE_TAG}，实际 ${DV}"; fi

    req POST "/api/v1/admin/users/$USER_ID/lock"
    check_ret_ok "POST /admin/users/:id/lock 锁定（禁用）用户"
    req GET "/api/v1/admin/users/$USER_ID"
    SV=$(jget data.status)
    if [ "$SV" = "disabled" ]; then pass "锁定后 status=disabled"
    else fail "锁定后 status=disabled" "实际 status=${SV}"; fi

    req POST "/api/v1/admin/users/$USER_ID/unlock"
    check_ret_ok "POST /admin/users/:id/unlock 解锁用户"
    req GET "/api/v1/admin/users/$USER_ID"
    SV=$(jget data.status)
    if [ "$SV" = "active" ]; then pass "解锁后 status=active"
    else fail "解锁后 status=active" "实际 status=${SV}"; fi

    req DELETE "/api/v1/admin/users/$USER_ID"
    check_ret_ok "DELETE /admin/users/:id 删除冒烟用户"
    req GET "/api/v1/admin/users/$USER_ID"
    check_ret_fail "删除后 GET 用户详情应 404"
else
    skip "用户详情/改名/锁定/解锁/删除链" "用户创建失败，无 id 可用"
fi

# 用户域负路径（不触达任何真实用户）
req POST "/api/v1/admin/users" "{\"username\":\"${SMOKE_TAG}nopwd\"}"
check_ret_fail "POST /admin/users 缺密码被拒"
req GET "/api/v1/admin/users/not-a-uuid"
check_ret_fail "GET /admin/users/<非法UUID> 被拒"
req POST "/api/v1/admin/users/not-a-uuid/lock"
check_ret_fail "POST /admin/users/<非法UUID>/lock 被拒"

# ---------------------------------------------------------------------------
section "角色闭环（建→绑菜单→验证→绑API权限→复制→删）"
# ---------------------------------------------------------------------------
req POST "/api/v1/admin/roles" "{\"name\":\"${SMOKE_TAG}-role\",\"description\":\"冒烟测试角色\"}"
check_ret_ok "POST /admin/roles 创建角色 ${SMOKE_TAG}-role"
check_field "创建返回角色 id" "data.id"
ROLE_ID=$(jget data.id)

if [ -n "$ROLE_ID" ]; then
    req GET "/api/v1/admin/roles/$ROLE_ID"
    check_ret_ok "GET /admin/roles/:id 角色详情可查"
    check_field "角色详情含 name" "data.name"

    if [ -n "$MENU_ID1" ] && [ -n "$MENU_ID2" ]; then
        req PUT "/api/v1/admin/roles/$ROLE_ID/menus" "{\"menu_ids\":[\"$MENU_ID1\",\"$MENU_ID2\"]}"
        check_ret_ok "PUT /admin/roles/:id/menus 绑定两个菜单"

        req GET "/api/v1/admin/roles/$ROLE_ID/menus"
        check_ret_ok "GET /admin/roles/:id/menus 角色菜单可查"
        MN=$(jlen data.menu_ids)
        if [ "$MN" = "2" ]; then pass "角色菜单绑定数 = 2"
        else fail "角色菜单绑定数 = 2" "实际 menu_ids 共 ${MN} 条"; fi
    else
        skip "角色绑菜单链" "菜单树取不到两个菜单 id"
    fi

    if [ -n "$EP_ID1" ] && [ -n "$EP_ID2" ]; then
        req PUT "/api/v1/admin/roles/$ROLE_ID/api-permissions" "{\"endpoint_ids\":[\"$EP_ID1\",\"$EP_ID2\"]}"
        check_ret_ok "PUT /admin/roles/:id/api-permissions 绑定两个 API 端点"

        req GET "/api/v1/admin/roles/$ROLE_ID/api-permissions"
        check_ret_ok "GET /admin/roles/:id/api-permissions 可查"
        EN=$(jlen data.endpoint_ids)
        if [ "$EN" = "2" ]; then pass "角色 API 权限绑定数 = 2"
        else fail "角色 API 权限绑定数 = 2" "实际 endpoint_ids 共 ${EN} 条"; fi
    else
        skip "角色绑 API 权限链" "api-endpoints 列表取不到两个端点 id"
    fi

    req GET "/api/v1/admin/roles/$ROLE_ID/device-groups"
    check_ret_ok "GET /admin/roles/:id/device-groups 可查"

    req GET "/api/v1/admin/roles/$ROLE_ID/users?page=1&page_size=20"
    check_ret_ok "GET /admin/roles/:id/users 角色成员可查（空亦可）"

    req POST "/api/v1/admin/roles/$ROLE_ID/copy"
    check_ret_ok "POST /admin/roles/:id/copy 复制角色"
    COPY_ID=$(jget data.id)
    CN=$(jget data.name)
    case "$CN" in
        *_copy*) pass "副本名带 _copy 后缀 ($CN)" ;;
        *) fail "副本名带 _copy 后缀" "实际 name=${CN}" ;;
    esac

    if [ -n "$COPY_ID" ]; then
        req DELETE "/api/v1/admin/roles/$COPY_ID"
        check_ret_ok "DELETE 角色副本"
    else
        skip "删除角色副本" "copy 未返回 id"
    fi

    req DELETE "/api/v1/admin/roles/$ROLE_ID"
    check_ret_ok "DELETE 冒烟角色"
    req GET "/api/v1/admin/roles/$ROLE_ID"
    check_ret_fail "删除后 GET 角色详情应 404"
else
    skip "角色绑菜单/API权限/复制/删除链" "角色创建失败，无 id 可用"
fi

# 角色域负路径（不触达内置角色）
req POST "/api/v1/admin/roles" "{}"
check_ret_fail "POST /admin/roles 缺 name 被拒"
req POST "/api/v1/admin/roles/$NIL_UUID/copy"
check_ret_fail "POST /admin/roles/<不存在ID>/copy 被拒"

# ---------------------------------------------------------------------------
section "API Key 闭环（签发→列表→吊销）"
# ---------------------------------------------------------------------------
req GET "/api/v1/api-keys"
check_ret_ok "GET /api-keys 列表可查"

req POST "/api/v1/api-keys" "{\"name\":\"${SMOKE_TAG}-key\",\"scopes\":[\"read\"]}"
check_ret_ok "POST /api-keys 签发 API Key"
check_field "签发返回明文 key（仅此一次）" "data.key"
check_field "签发返回 key_prefix" "data.key_prefix"
KEY_ID=$(jget data.id)

if [ -n "$KEY_ID" ]; then
    req GET "/api/v1/api-keys"
    check_list_nonempty "签发后 API Key 列表非空" "data.items"
    IDX=$(find_item_idx_by_id "$KEY_ID")
    if [ -n "$IDX" ]; then pass "列表包含新签发 key (items.$IDX)"
    else fail "列表包含新签发 key" "id=${KEY_ID} 未出现在 data.items"; fi

    req DELETE "/api/v1/api-keys/$KEY_ID"
    check_ret_ok "DELETE /api-keys/:id 吊销"

    req GET "/api/v1/api-keys"
    IDX=$(find_item_idx_by_id "$KEY_ID")
    if [ -n "$IDX" ]; then
        RV=$(jget "data.items.$IDX.revoked_at")
        if [ -n "$RV" ] && [ "$RV" != "None" ] && [ "$RV" != "null" ]; then
            pass "吊销后 revoked_at 已置位 ($RV)"
        else
            fail "吊销后 revoked_at 已置位" "revoked_at=${RV}"
        fi
    else
        # 若列表过滤了已吊销 key，同样视为吊销生效
        pass "吊销后 key 不再出现在列表（等效吊销生效）"
    fi
else
    skip "API Key 列表验证/吊销链" "签发未返回 id"
fi

req POST "/api/v1/api-keys" "{\"name\":\"\"}"
check_ret_fail "POST /api-keys 空 name 被拒"
req DELETE "/api/v1/api-keys/not-a-uuid"
check_ret_fail "DELETE /api-keys/<非法UUID> 被拒"

# ---------------------------------------------------------------------------
section "死信队列（读 + 负路径，replay 只测负路径）"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/dead-letters?page=1&page_size=20"
check_list_or_empty "GET /admin/dead-letters 死信列表可查" "data.items"

req GET "/api/v1/admin/dead-letters/$NIL_UUID"
check_ret_fail "GET /admin/dead-letters/<不存在ID> 被拒"

req POST "/api/v1/admin/dead-letters/$NIL_UUID/replay"
check_ret_fail "POST /admin/dead-letters/<不存在ID>/replay 被拒"

# ---------------------------------------------------------------------------
section "权限清单与审计日志可查"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/permissions"
check_ret_ok "GET /admin/permissions 权限清单可查"

# 放在所有写操作之后：本套件自身的写动作保证审计日志至少有记录
req GET "/api/v1/admin/audit-logs?page=1&page_size=20"
check_list_nonempty "GET /admin/audit-logs 审计日志非空" "data.items"
check_field "审计日志首条含 id" "data.items.0.id"
check_count_ge "审计日志总数 ≥ 1" "data.total" 1

smoke_summary
