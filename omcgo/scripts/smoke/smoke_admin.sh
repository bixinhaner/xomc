#!/usr/bin/env bash
# =============================================================================
# smoke_admin.sh — F06 RBAC（用户/角色/菜单/权限/审计/API端点/API Key/死信）业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=admin 全部路由）：
#   - 全读：users / roles(+all) / menus(列表+tree+user-tree) / permissions /
#     audit-logs / api-endpoints(+groups) / dead-letters / api-keys
#   - 用户 CRUD 闭环：POST 建（密码 RSA-OAEP 加密传输）→ GET 详情 → PUT 改昵称
#     → lock(禁用) → unlock(启用) → DELETE → GET 404
#   - 密码生命周期闭环（自建用户）：reset-password → 新密码登录 →
#     change-password（旧密登录失败/新密登录成功）→ force-logout（旧 token 401）
#   - 角色闭环：建 → PUT :id 改属性 → 绑两个菜单(PUT :id/menus) → GET :id/menus 验证 →
#     绑 API 权限(PUT :id/api-permissions) → 绑数据范围(PUT :id/device-groups →
#     GET 校验 → PUT 空列表还原) → copy → 删 copy → 删原角色
#   - API Key 闭环：POST 签发 → GET 列表含它 → DELETE 吊销 → revoked_at 置位
#   - api-endpoints sync（upsert-only 幂等，安全）
#   - 死信队列：列表 + 不存在 ID 负路径（replay 只测负路径）
#   - 负路径：缺分页 400、缺密码 400、非法 UUID 400、缺 name 400、
#     缺 user_ids 400、change-password 旧密码错误 7020
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

# try_login 用户名 密码 —— 尝试登录，成功输出 access_token，失败输出空（不像
# smoke_login 那样 exit 2；公钥已在 smoke_login 时缓存，子 shell 内可安全调用）
try_login() {
    local user="$1" pass="$2" enc resp
    enc=$(encrypt_password "$pass") || { echo ""; return 0; }
    resp=$(curl --max-time 10 -s -X POST "$API/auth/login" \
        -H 'Content-Type: application/json' \
        -d "{\"username\":\"$user\",\"encrypted_password\":\"$enc\",\"key_id\":\"$PUBLIC_KEY_ID\"}")
    printf '%s' "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('access_token',''))" 2>/dev/null || echo ""
}

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

# 已知问题探针（只读）：roles 列表 search 参数被后端忽略 —— 用一个必然不存在的
# search 值查询，若返回 0 条说明过滤已修复（自动转正常断言）
req GET "/api/v1/admin/roles?page=1&page_size=100&search=${SMOKE_TAG}-no-such-role"
RN_SEARCH=$(jlen data.items)
if [ "$RN_SEARCH" = "0" ]; then pass "GET /admin/roles?search= 过滤生效（返回 0 条）"
else known_bug "GET /admin/roles?search= 过滤生效" "search 被忽略返回全量 ${RN_SEARCH} 条（pg_role_repository.ListWithPagination 未使用 RoleFilter.Search，且 count 查询不带任何过滤条件）"; fi

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
section "密码生命周期闭环（重置→登录→改密→强退，全程自建用户）"
# ---------------------------------------------------------------------------
PWD_USER="${SMOKE_TAG}pwd"
PWD_P1="Smk@Pass123"      # 创建时初始密码
PWD_P2="Smk@Reset456"     # admin 重置后的密码
PWD_P3="Smk@Change789"    # 用户自助改密后的密码

ENC_P1=$(encrypt_password "$PWD_P1")
req POST "/api/v1/admin/users" "{\"username\":\"$PWD_USER\",\"encrypted_password\":\"$ENC_P1\",\"key_id\":\"$PUBLIC_KEY_ID\",\"display_name\":\"冒烟密码闭环用户\",\"description\":\"smoke ${SMOKE_TAG}\"}"
check_ret_ok "POST /admin/users 创建密码闭环用户 ${PWD_USER}"
PWD_UID=$(jget data.id)

if [ -n "$PWD_UID" ]; then
    # 1) admin 重置密码 P1 → P2（reset-password 走 RSA 加密路径）
    ENC_P2=$(encrypt_password "$PWD_P2")
    req POST "/api/v1/admin/users/$PWD_UID/reset-password" "{\"encrypted_new_password\":\"$ENC_P2\",\"key_id\":\"$PUBLIC_KEY_ID\"}"
    check_ret_ok "POST /admin/users/:id/reset-password 重置自建用户密码"

    # 2) 重置后用新密码登录（must_change_password 仅作 FE 提示，不阻断签发 token）
    U_TOKEN=$(try_login "$PWD_USER" "$PWD_P2")
    if [ -n "$U_TOKEN" ]; then pass "重置后新密码可登录"
    else fail "重置后新密码可登录" "login 未返回 access_token"; fi

    # 3) change-password 闭环：以自建用户身份 P2 → P3（旧/新密码共用一次 key_id）
    if [ -n "$U_TOKEN" ]; then
        ENC_OLD=$(encrypt_password "$PWD_P2")
        ENC_NEW=$(encrypt_password "$PWD_P3")
        ADMIN_TOKEN_SAVE="$TOKEN"; TOKEN="$U_TOKEN"
        req POST "/api/v1/auth/change-password" "{\"encrypted_old_password\":\"$ENC_OLD\",\"encrypted_new_password\":\"$ENC_NEW\",\"key_id\":\"$PUBLIC_KEY_ID\"}"
        check_ret_ok "POST /auth/change-password 自建用户自助改密"

        # 负路径：空 body 缺密码 → 400（在自建用户 token 下打，不触达 admin 密码）
        req POST "/api/v1/auth/change-password" "{}"
        check_ret_fail "change-password 空 body 缺密码被拒"

        # 负路径：旧密码错误 → biz 7020 被拒（P3 已生效，P1 已是错误旧密码）
        ENC_BAD=$(encrypt_password "$PWD_P1")
        ENC_N2=$(encrypt_password "Smk@Other000")
        req POST "/api/v1/auth/change-password" "{\"encrypted_old_password\":\"$ENC_BAD\",\"encrypted_new_password\":\"$ENC_N2\",\"key_id\":\"$PUBLIC_KEY_ID\"}"
        check_ret_fail "change-password 旧密码错误被拒"
        TOKEN="$ADMIN_TOKEN_SAVE"
    else
        skip "change-password 闭环" "自建用户登录失败，无 token 可用"
    fi

    # 4) 改密后旧密码（P2）登录必须失败
    T_OLD=$(try_login "$PWD_USER" "$PWD_P2")
    if [ -z "$T_OLD" ]; then pass "改密后旧密码登录失败（预期被拒）"
    else fail "改密后旧密码登录失败（预期被拒）" "旧密码仍能签发 token"; fi

    # 5) 改密后新密码（P3）登录成功，拿 token 供强退验证
    U_TOKEN2=$(try_login "$PWD_USER" "$PWD_P3")
    if [ -n "$U_TOKEN2" ]; then pass "改密后新密码登录成功"
    else fail "改密后新密码登录成功" "login 未返回 access_token"; fi

    # 6) force-logout 强退自建用户：其已签发 token 立即失效（撤销时间戳含同秒 <=）
    req POST "/api/v1/admin/users/force-logout" "{\"user_ids\":[\"$PWD_UID\"]}"
    check_ret_ok "POST /admin/users/force-logout 强退自建用户"
    check_field "强退返回 revoked 计数" "data.revoked"

    if [ -n "$U_TOKEN2" ]; then
        ADMIN_TOKEN_SAVE="$TOKEN"; TOKEN="$U_TOKEN2"
        req GET "/api/v1/auth/me"
        TOKEN="$ADMIN_TOKEN_SAVE"
        check_status "强退后该用户旧 token 调 /auth/me 应 401" 401
    else
        skip "强退后 token 失效验证" "无自建用户 token 可用"
    fi

    # 强退仅作用于目标用户：admin 自身 token 不受影响
    req GET "/api/v1/auth/me"
    check_ret_ok "强退他人后 admin token 仍有效（GET /auth/me）"

    # 重置密码缺密码字段负路径（用户仍存在时打，命中 missing password 分支）
    req POST "/api/v1/admin/users/$PWD_UID/reset-password" "{}"
    check_ret_fail "reset-password 缺密码字段被拒"

    # 清理
    req DELETE "/api/v1/admin/users/$PWD_UID"
    check_ret_ok "DELETE 密码闭环用户（清理）"
else
    skip "密码生命周期闭环（重置/改密/强退）" "用户创建失败，无 id 可用"
fi

# 密码域负路径（不触达内置用户）
req POST "/api/v1/admin/users/not-a-uuid/reset-password" "{}"
check_ret_fail "reset-password 非法 UUID 被拒"
req POST "/api/v1/admin/users/force-logout" "{}"
check_ret_fail "force-logout 缺 user_ids 被拒"
req POST "/api/v1/admin/users/force-logout" "{\"user_ids\":[\"$NIL_UUID\"]}"
check_ret_fail "force-logout 不存在用户被拒"

# ---------------------------------------------------------------------------
section "角色闭环（建→改属性→绑菜单→验证→绑API权限→绑数据范围→复制→删）"
# ---------------------------------------------------------------------------
req POST "/api/v1/admin/roles" "{\"name\":\"${SMOKE_TAG}-role\",\"description\":\"冒烟测试角色\"}"
check_ret_ok "POST /admin/roles 创建角色 ${SMOKE_TAG}-role"
check_field "创建返回角色 id" "data.id"
ROLE_ID=$(jget data.id)

if [ -n "$ROLE_ID" ]; then
    req GET "/api/v1/admin/roles/$ROLE_ID"
    check_ret_ok "GET /admin/roles/:id 角色详情可查"
    check_field "角色详情含 name" "data.name"

    # PUT 改自建角色属性（name/description 指针字段，仅传需变更项）
    req PUT "/api/v1/admin/roles/$ROLE_ID" "{\"name\":\"${SMOKE_TAG}-role-upd\",\"description\":\"冒烟更新描述\"}"
    check_ret_ok "PUT /admin/roles/:id 修改自建角色属性"
    RN=$(jget data.name)
    if [ "$RN" = "${SMOKE_TAG}-role-upd" ]; then pass "角色 name 已更新 ($RN)"
    else fail "角色 name 已更新" "期望 ${SMOKE_TAG}-role-upd，实际 ${RN}"; fi
    RD=$(jget data.description)
    if [ "$RD" = "冒烟更新描述" ]; then pass "角色 description 已更新"
    else fail "角色 description 已更新" "实际 description=${RD}"; fi

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

    # 数据范围闭环：取一个分组 id（树空则自建临时分组）→ 绑定 → 校验 → 空列表还原
    req GET "/api/v1/device-groups/tree"
    check_ret_ok "GET /device-groups/tree 可查（取数据范围分组）"
    GROUP_ID=$(jget data.items.0.id)
    DG_CREATED=""
    if [ -z "$GROUP_ID" ]; then
        req POST "/api/v1/device-groups" "{\"name\":\"${SMOKE_TAG}grp\"}"
        GROUP_ID=$(jget data.id)
        DG_CREATED="$GROUP_ID"
    fi
    if [ -n "$GROUP_ID" ]; then
        req PUT "/api/v1/admin/roles/$ROLE_ID/device-groups" "{\"device_group_ids\":[\"$GROUP_ID\"],\"network_types\":[]}"
        check_ret_ok "PUT /admin/roles/:id/device-groups 绑定数据范围分组"

        req GET "/api/v1/admin/roles/$ROLE_ID/device-groups"
        DGN=$(jlen data.device_group_ids)
        DG0=$(jget data.device_group_ids.0)
        if [ "$DGN" = "1" ] && [ "$DG0" = "$GROUP_ID" ]; then pass "数据范围回读一致（1 个分组 ${DG0}）"
        else fail "数据范围回读一致" "期望 [${GROUP_ID}]，实际共 ${DGN} 条 首条=${DG0}"; fi

        req PUT "/api/v1/admin/roles/$ROLE_ID/device-groups" "{\"device_group_ids\":[],\"network_types\":[]}"
        check_ret_ok "PUT 空列表还原数据范围"

        req GET "/api/v1/admin/roles/$ROLE_ID/device-groups"
        DGN=$(jlen data.device_group_ids)
        if [ "$DGN" = "0" ]; then pass "还原后数据范围为空"
        else fail "还原后数据范围为空" "实际 device_group_ids 共 ${DGN} 条"; fi

        if [ -n "$DG_CREATED" ]; then
            req DELETE "/api/v1/device-groups/$DG_CREATED"
            check_ret_ok "DELETE 自建临时分组（清理）"
        fi
    else
        skip "角色数据范围绑定闭环" "分组树为空且临时分组创建失败"
    fi

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
