#!/usr/bin/env bash
# =============================================================================
# smoke_auth.sh — 认证与会话 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key=auth 全部 12 条路由）：
#   - 健康/就绪探针：GET /healthz、GET /readyz（免认证）
#   - 免认证公开端点：captcha、public-key、admin/public/configs、
#     admin/public/ui-assets/:name（无种子资产，以 400/404 负路径覆盖读链路）
#   - RSA 加密登录（lib smoke_login）→ me / menus 会话链路
#   - refresh_token 刷新链路（二次登录取 refresh_token → refresh → 新 token 访问 me）
#   - switch-role（admin 单角色环境正路径 skip，负路径恒测）
#   - X-API-Key 直通（内部 key 文件在容器卷内，宿主机无 key 时 skip；伪造 key 401 恒测）
#   - 负路径：错密码 / 明文禁用 / 缺密码 / 伪造 refresh_token / 无 token / 伪造 Bearer
#   - change-password：按域要求整体跳过（避免影响后续套件登录）
#
# 用法：bash smoke_auth.sh [BASE_URL]   （默认 http://localhost:8081，本机容器栈用 :18091）
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"

smoke_init "认证与会话" "$@"

# ---------------------------------------------------------------------------
section "健康/就绪探针（免认证）"
# ---------------------------------------------------------------------------
req_noauth GET "/healthz"
check_status "GET /healthz 存活探针" 200
check_field "healthz 返回 status 字段" "status"

req_noauth GET "/readyz"
check_status "GET /readyz 就绪探针" 200
check_list_nonempty "readyz 返回组件健康明细" "components"

# ---------------------------------------------------------------------------
section "免认证公开端点"
# ---------------------------------------------------------------------------
req_noauth GET "/api/v1/auth/captcha"
check_ret_ok "GET /auth/captcha 生成验证码"
check_field "captcha 返回 captcha_id" "data.captcha_id"
check_field "captcha 返回 base64 图片" "data.image"

req_noauth GET "/api/v1/auth/public-key"
check_ret_ok "GET /auth/public-key 获取 RSA 公钥"
check_field "public-key 返回 key_id" "data.key_id"
check_field "public-key 返回 PEM 公钥" "data.public_key"

req_noauth GET "/api/v1/admin/public/configs"
check_ret_ok "GET /admin/public/configs 免认证公开配置"
check_list_nonempty "公开配置非空（is_public=true 种子）" "data"

# ui-assets 无内置种子资产：用合法格式不存在的 uuid.png 走 404、非法文件名走 400，
# 两条负路径足以证明免认证路由 + 文件名白名单校验链路在线。
req_noauth GET "/api/v1/admin/public/ui-assets/00000000-0000-0000-0000-000000000000.png"
check_status "GET /admin/public/ui-assets/<不存在的uuid>.png → 404" 404
req_noauth GET "/api/v1/admin/public/ui-assets/evil.txt"
check_status "GET /admin/public/ui-assets/<非法文件名> → 400" 400

# ---------------------------------------------------------------------------
section "RSA 加密登录 → me / menus 会话链路"
# ---------------------------------------------------------------------------
smoke_login

req GET "/api/v1/auth/me"
check_ret_ok "GET /auth/me 当前会话用户"
check_field "me 返回用户名" "data.username"
check_field "me 返回用户 id" "data.id"
check_list_nonempty "me 返回角色列表（admin 至少 1 个角色）" "data.roles"
ROLE_CNT=$(jlen "data.roles")
ROLE0_ID=$(jget "data.roles.0.id")

req GET "/api/v1/auth/menus"
check_ret_ok "GET /auth/menus 当前角色菜单树"
check_list_nonempty "菜单树非空（内置菜单种子）" "data"

# ---------------------------------------------------------------------------
section "refresh_token 刷新链路"
# ---------------------------------------------------------------------------
# 二次显式登录拿完整 token 对（lib smoke_login 只留 access_token）
ENC_PASS=$(encrypt_password "$ADMIN_PASS")
req_noauth POST "/api/v1/auth/login" \
    "{\"username\":\"$ADMIN_USER\",\"encrypted_password\":\"$ENC_PASS\",\"key_id\":\"$PUBLIC_KEY_ID\"}"
check_ret_ok "POST /auth/login 加密登录（取 token 对）"
check_field "登录返回 access_token" "data.access_token"
check_field "登录返回 refresh_token" "data.refresh_token"
REFRESH_TOKEN=$(jget "data.refresh_token")

if [ -n "$REFRESH_TOKEN" ]; then
    req_noauth POST "/api/v1/auth/refresh" "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
    check_ret_ok "POST /auth/refresh 刷新成功"
    check_field "refresh 返回新 access_token" "data.access_token"
    NEW_ACCESS=$(jget "data.access_token")
    if [ -n "$NEW_ACCESS" ]; then
        SAVED_TOKEN="$TOKEN"
        TOKEN="$NEW_ACCESS"
        req GET "/api/v1/auth/me"
        check_ret_ok "刷新后的 access_token 可访问 /auth/me"
        TOKEN="$SAVED_TOKEN"
    else
        skip "刷新后 token 复用验证" "refresh 未返回 access_token（前一断言已计失败）"
    fi
else
    skip "refresh 刷新链路" "登录未返回 refresh_token（前一断言已计失败）"
fi

# ---------------------------------------------------------------------------
section "switch-role 角色切换"
# ---------------------------------------------------------------------------
if [ "$ROLE_CNT" -ge 2 ]; then
    ROLE1_ID=$(jget "data.roles.1.id" 2>/dev/null)
    # 上面的 BODY 已被覆盖，重新拉 me 取第二角色
    req GET "/api/v1/auth/me"
    ROLE1_ID=$(jget "data.roles.1.id")
    req POST "/api/v1/auth/switch-role" "{\"role_id\":\"$ROLE1_ID\"}"
    check_ret_ok "切换到第二角色"
    check_field "切换返回新 token" "data.access_token"
    SWITCH_TOKEN=$(jget "data.access_token")
    if [ -n "$SWITCH_TOKEN" ]; then
        SAVED_TOKEN="$TOKEN"
        TOKEN="$SWITCH_TOKEN"
        req GET "/api/v1/auth/menus"
        check_ret_ok "切换角色后的 token 可访问 menus"
        # 还原默认角色，避免污染后续套件
        req POST "/api/v1/auth/switch-role" "{\"role_id\":\"$ROLE0_ID\"}"
        check_ret_ok "切回原角色（还原默认角色）"
        TOKEN="$SAVED_TOKEN"
    fi
else
    skip "switch-role 正路径" "admin 仅 ${ROLE_CNT} 个角色（多角色环境才测切换，避免污染默认角色）"
fi

# 负路径恒测：未分配给当前用户的角色 id → 拒绝（不落库、无副作用）
req POST "/api/v1/auth/switch-role" "{\"role_id\":\"00000000-dead-beef-0000-000000000000\"}"
check_ret_fail "switch-role 到未分配角色被拒绝"

# ---------------------------------------------------------------------------
section "X-API-Key 直通"
# ---------------------------------------------------------------------------
API_KEY="${OMCCTL_API_KEY:-}"
if [ -z "$API_KEY" ] && [ -r "/var/lib/omcgo/secrets/.api-key" ]; then
    API_KEY=$(head -1 "/var/lib/omcgo/secrets/.api-key" 2>/dev/null)
fi
if [ -n "$API_KEY" ]; then
    # 故意把 Bearer 置为无效值：若仍 200 即证明是 X-API-Key 在生效（中间件先查 key）
    SAVED_TOKEN="$TOKEN"
    TOKEN="invalid-bearer-for-apikey-test"
    req GET "/api/v1/auth/me" "" -H "X-API-Key: $API_KEY"
    TOKEN="$SAVED_TOKEN"
    check_ret_ok "X-API-Key 直通访问受保护端点"
    check_field "X-API-Key 鉴权回填用户" "data.username"
else
    skip "X-API-Key 直通正路径" "宿主机无内部 key（OMCCTL_API_KEY 未设置且密钥文件在容器卷内不可读）"
fi
# 伪造 key 恒测：X-API-Key 优先于 Bearer 校验，伪造 key 必须 401
req GET "/api/v1/auth/me" "" -H "X-API-Key: omk_deadbeefdeadbeefdeadbeefdeadbeef"
check_status "伪造 X-API-Key 被拒绝" 401

# ---------------------------------------------------------------------------
section "登录与鉴权负路径"
# ---------------------------------------------------------------------------
# 错密码（RSA 正确加密流程 + 错误明文）→ 400/401；放在所有正路径登录之后，
# 仅 1 次失败计数，下次运行 smoke_login 成功后即被 LoginGuard/IPGuard 清零
ENC_WRONG=$(encrypt_password "definitely-wrong-password-${SMOKE_TAG}")
req_noauth POST "/api/v1/auth/login" \
    "{\"username\":\"$ADMIN_USER\",\"encrypted_password\":\"$ENC_WRONG\",\"key_id\":\"$PUBLIC_KEY_ID\"}"
check_status_in "错密码加密登录被拒绝" "400 401"

# 明文密码（错误密码）：明文禁用 → 400 biz 7004；个别栈允许明文则错密码 → 401
req_noauth POST "/api/v1/auth/login" \
    "{\"username\":\"$ADMIN_USER\",\"password\":\"wrong-plain-${SMOKE_TAG}\"}"
check_status_in "明文密码登录被拒绝（禁用 400 / 错密码 401）" "400 401"

# 缺密码字段 → 400
req_noauth POST "/api/v1/auth/login" "{\"username\":\"$ADMIN_USER\"}"
check_status "缺密码字段登录 → 400" 400

# 伪造 refresh_token → 401
req_noauth POST "/api/v1/auth/refresh" "{\"refresh_token\":\"garbage.invalid.token\"}"
check_status_in "伪造 refresh_token 被拒绝" "400 401"

# 无 token 访问受保护端点 → 401
req_noauth GET "/api/v1/auth/me"
check_status "无 token 访问 /auth/me → 401" 401
req_noauth GET "/api/v1/auth/menus"
check_status "无 token 访问 /auth/menus → 401" 401

# 伪造 Bearer → 401
SAVED_TOKEN="$TOKEN"
TOKEN="totally.invalid.jwt"
req GET "/api/v1/auth/me"
TOKEN="$SAVED_TOKEN"
check_status "伪造 Bearer token → 401" 401

# ---------------------------------------------------------------------------
section "change-password（按域要求跳过）"
# ---------------------------------------------------------------------------
skip "POST /auth/change-password" "破坏性端点按域要求整体跳过（改 admin 密码会影响后续套件登录）"

smoke_summary
