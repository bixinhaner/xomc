#!/usr/bin/env bash
# =============================================================================
# smoke_product.sh — F02 产品与参数字典（products + param-models, super_admin）
#
# 覆盖：
#   - products 列表/详情/match 正则路由/match-order/orphan-devices/枚举字典
#   - 产品 CRUD + pattern 闭环（建 ${SMOKE_TAG} 产品 → 加正则 → 改/移/删 → 删产品）
#   - cache/refresh 幂等、orphan rematch（幂等后台任务）
#   - param-models 列表(builtin 非空)/:name 详情/:name mappings/standard 树/file-content
#   - translate 双向翻译（真实 builtin mapping：standardPath → privatePath → 回翻一致）
#   - standard 参数治理闭环、custom mapping 闭环（builtin 模型上增删 custom 行）
#   - param-model PUT 元信息治理（自建 custom 模型改 description/is_active 回读一致）
#   - upload-xml 闭环（构造最小合法 parameterModel XML，name 唯一带 ${SMOKE_TAG}：
#     上传 → 列表/详情可见(source=custom, deletable=true) → PUT 改元信息 → DELETE 清理）
#   - upload-xml 负路径（缺 file 400 / 缺 paramModel 名 400 / 根元素非法 400）
#   - builtin 删除守门（param-model 403 / 产品 403 / builtin pattern 403 / builtin mapping 403）
#   - discovered 视图 + destructive 端点负路径（仅参数校验，绝不真删）
#   - 权限边界：未认证 401 + 普通角色(viewer) 403
#
# 红线说明：
#   - upload-xml 正路径用「全新唯一名 + 不带 ?force」——新模型上传后 destructiveReload
#     全量重载磁盘上仍在的全部 builtin XML（UPSERT 触 updated_at），DeleteOrphansSince
#     只删未被触及的孤儿行，故 orphans_deleted=0、不伤任何 builtin（已实测验证）；
#     绝不带 ?force=true（覆盖既有文件，真 destructive），绝不上传与 builtin 同名的 XML。
#   - PUT 元信息只打自建 custom 模型（删除即完全回滚）；builtin PUT 虽无守门（会改
#     builtin 的 description/is_active），但为保持 builtin 纯净不去触碰，仅 notes 说明。
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "F02 产品与参数字典" "$@"
smoke_login

# ---------------------------------------------------------------------------
# 本地辅助断言：精确值比较（jget 输出与期望值逐字符比对）
# ---------------------------------------------------------------------------
assert_eq() {
    local desc="$1" actual="$2" expected="$3"
    if [ "$actual" = "$expected" ]; then pass "$desc ($actual)"
    else fail "$desc" "期望 '$expected'，实际 '$actual'"; fi
}

# 全局提取变量（set -u 防御：全部先初始化）
BUILTIN_PRODUCT_ID=""
TR_PRODUCT_ID=""
TR_PM_NAME=""
BUILTIN_PATTERN_ID=""
LITERAL_CLASS=""
NEW_PRODUCT_ID=""
PATTERN2_ID=""
PM_NAME=""
PM_LOADED_FROM=""
STD_PATH=""
PRIV_PATH=""
BUILTIN_MAPPING_ID=""
STD_TREE_PATH=""
SMK_STD=""
MAPPING_ID=""
VIEWER_ROLE_ID=""
SMK_USER_ID=""
VIEWER_TOKEN=""
PM_UPLOAD_NAME=""

# ═══════════════════════════════════════════════════════════════════════════
section "1. 产品列表与枚举字典（builtin products.xml 装配）"
# ═══════════════════════════════════════════════════════════════════════════
req GET "/api/v1/products"
check_ret_ok "产品列表可查"
check_list_nonempty "内置产品非空（products.xml 装配）" "data.items"
EXTRACT=$(printf '%s' "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = (d.get('data') or {}).get('items') or []
builtin_id = ''
tr_id = ''
tr_pm = ''
for it in items:
    if not builtin_id and it.get('is_builtin'):
        builtin_id = str(it.get('id') or '')
    if not tr_id and it.get('param_model_name'):
        tr_id = str(it.get('id') or '')
        tr_pm = str(it.get('param_model_name') or '')
print(builtin_id + '|' + tr_id + '|' + tr_pm)
" 2>/dev/null)
IFS='|' read -r BUILTIN_PRODUCT_ID TR_PRODUCT_ID TR_PM_NAME <<< "$EXTRACT"

if [ -n "$BUILTIN_PRODUCT_ID" ]; then
    req GET "/api/v1/products/$BUILTIN_PRODUCT_ID"
    check_ret_ok "产品详情可查"
    check_field "产品详情含 product.id" "data.product.id"
    check_field "产品详情含 patterns 列表" "data.patterns"
else
    skip "产品详情" "列表中无 builtin 产品（无法取 ID）"
fi

req GET "/api/v1/products/match-order"
check_ret_ok "match-order 全局正则序可查"
check_list_nonempty "match-order 非空（builtin 正则）" "data.items"
EXTRACT=$(printf '%s' "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = (d.get('data') or {}).get('items') or []
builtin_pid = ''
literal = ''
dollar = chr(36)  # 避免在 bash 双引号串里写美元符
for it in items:
    if not builtin_pid and it.get('source') == 'builtin':
        builtin_pid = str(it.get('pattern_id') or '')
    pc = it.get('product_class') or ''
    if (not literal and it.get('is_active') and it.get('source') == 'builtin'
            and len(pc) > 2 and pc.startswith('^') and pc.endswith(dollar)):
        mid = pc[1:-1]
        if mid and all(ch.isalnum() or ch in '/_-' for ch in mid):
            literal = mid
print(builtin_pid + '|' + literal)
" 2>/dev/null)
IFS='|' read -r BUILTIN_PATTERN_ID LITERAL_CLASS <<< "$EXTRACT"

req GET "/api/v1/products/orphan-devices?page=1&page_size=20"
check_list_or_empty "孤儿设备列表可查" "data.items"

req GET "/api/v1/products/indicator-platforms?deviceType=enb"
check_ret_ok "indicator-platforms(enb) 可查"
check_list_nonempty "enb 指标平台字典非空（builtin 指标库）" "data.items"

req GET "/api/v1/products/indicator-platforms"
check_ret_fail "indicator-platforms 缺 deviceType 被拒"

req GET "/api/v1/products/alarm-ne-types"
check_ret_ok "alarm-ne-types 可查"
check_list_nonempty "告警网元类型字典非空（builtin 告警库）" "data.items"

# ═══════════════════════════════════════════════════════════════════════════
section "2. match 正则路由命中（活栈已有 product_class）"
# ═══════════════════════════════════════════════════════════════════════════
if [ -n "$LITERAL_CLASS" ]; then
    req GET "/api/v1/products/match?productClass=$LITERAL_CLASS"
    check_ret_ok "match 命中查询（productClass=${LITERAL_CLASS}）"
    assert_eq "builtin 字面正则命中 matched=true" "$(jget data.matched)" "True"
    check_field "命中返回 product.id" "data.product.id"
    check_field "命中返回 matched_pattern" "data.matched_pattern"
else
    skip "match 命中（builtin 字面正则）" "match-order 中未找到可还原成 product_class 的字面正则"
fi

# #125-product 修复：registry.MatchProductClass 未命中返回 ErrOrphan（registry.go），
# Handler.Match 现把 ErrOrphan 识别为「未命中」→ 200 + matched=false（不再当 500）。
req GET "/api/v1/products/match?productClass=SMK-NOMATCH-${SMOKE_TAG}"
check_ret_ok "match 未命中查询返回 200（productClass 无匹配）"
assert_eq "未命中返回 matched=false" "$(jget data.matched)" "False"

req GET "/api/v1/products/match"
check_ret_fail "match 缺 productClass 参数被拒"

# ═══════════════════════════════════════════════════════════════════════════
section "3. 产品 CRUD + pattern 闭环（${SMOKE_TAG} 自建实体）"
# ═══════════════════════════════════════════════════════════════════════════
req POST "/api/v1/products" "{\"name\":\"smoke-prod-${SMOKE_TAG}\",\"vendor\":\"SmokeVendor\",\"tech\":\"lte\",\"description\":\"smoke test product\",\"indicator_device_type\":\"gnb\",\"alarm_ne_type\":\"ENB\",\"patterns\":[\"^SMK-MATCH-${SMOKE_TAG}\$\"]}"
check_ret_ok "创建产品（带 1 条 custom 正则）"
NEW_PRODUCT_ID=$(jget data.id)
check_field "创建返回产品 id" "data.id"

if [ -n "$NEW_PRODUCT_ID" ]; then
    req GET "/api/v1/products/$NEW_PRODUCT_ID"
    check_ret_ok "新建产品详情可查"
    assert_eq "新建产品 name 回读一致" "$(jget data.product.name)" "smoke-prod-${SMOKE_TAG}"

    # 新正则立即参与全局路由（Create 内同步 registry.Refresh）
    req GET "/api/v1/products/match?productClass=SMK-MATCH-${SMOKE_TAG}"
    assert_eq "新建正则路由命中 matched=true" "$(jget data.matched)" "True"
    assert_eq "命中产品即新建产品" "$(jget data.product.id)" "$NEW_PRODUCT_ID"

    req PUT "/api/v1/products/$NEW_PRODUCT_ID" "{\"description\":\"updated by smoke\"}"
    check_ret_ok "更新产品描述"
    assert_eq "描述更新生效" "$(jget data.description)" "updated by smoke"

    # pattern 闭环：加第二条 → 上移 → 下移还原 → 停用 → 启用 → 删除
    req POST "/api/v1/products/$NEW_PRODUCT_ID/patterns" "{\"product_class\":\"^SMK-EXTRA-${SMOKE_TAG}\$\"}"
    check_ret_ok "新增第二条 custom 正则"
    PATTERN2_ID=$(jget data.id)
    assert_eq "新增正则 source=custom" "$(jget data.source)" "custom"

    if [ -n "$PATTERN2_ID" ]; then
        req PUT "/api/v1/products/$NEW_PRODUCT_ID/patterns/$PATTERN2_ID/move" "{\"direction\":\"up\"}"
        check_ret_ok "custom 正则上移"
        req PUT "/api/v1/products/$NEW_PRODUCT_ID/patterns/$PATTERN2_ID/move" "{\"direction\":\"down\"}"
        check_ret_ok "custom 正则下移（还原顺序）"

        req PUT "/api/v1/products/$NEW_PRODUCT_ID/patterns/$PATTERN2_ID" "{\"is_active\":false}"
        check_ret_ok "custom 正则停用"
        assert_eq "停用后 is_active=false" "$(jget data.is_active)" "False"
        req PUT "/api/v1/products/$NEW_PRODUCT_ID/patterns/$PATTERN2_ID" "{\"is_active\":true}"
        check_ret_ok "custom 正则恢复启用"

        req DELETE "/api/v1/products/$NEW_PRODUCT_ID/patterns/$PATTERN2_ID"
        check_ret_ok "删除第二条 custom 正则"
        PATTERN2_ID=""
    else
        skip "pattern 移动/启停/删除" "第二条正则创建未返回 id"
    fi
else
    skip "产品闭环后续（详情/match/更新/pattern）" "产品创建未返回 id"
fi

# builtin 守门：内置正则只读、内置产品禁删
if [ -n "$BUILTIN_PATTERN_ID" ] && [ -n "$BUILTIN_PRODUCT_ID" ]; then
    req PUT "/api/v1/products/$BUILTIN_PRODUCT_ID/patterns/$BUILTIN_PATTERN_ID" "{\"is_active\":false}"
    check_status "builtin 正则修改被拒" 403
else
    skip "builtin 正则修改守门" "未取到 builtin pattern_id"
fi
if [ -n "$BUILTIN_PRODUCT_ID" ]; then
    req DELETE "/api/v1/products/$BUILTIN_PRODUCT_ID"
    check_status "builtin 产品删除被拒" 403
else
    skip "builtin 产品删除守门" "未取到 builtin 产品 id"
fi

if [ -n "$NEW_PRODUCT_ID" ]; then
    req DELETE "/api/v1/products/$NEW_PRODUCT_ID"
    check_ret_ok "删除自建产品（pattern 随 FK CASCADE 清理）"
    req GET "/api/v1/products/$NEW_PRODUCT_ID"
    check_status "删除后产品详情 404" 404
    NEW_PRODUCT_ID=""
fi

req POST "/api/v1/products" "{}"
check_ret_fail "创建产品缺必填字段被拒"
req GET "/api/v1/products/not-a-uuid"
check_ret_fail "产品详情非法 id 被拒"

# ═══════════════════════════════════════════════════════════════════════════
section "4. cache/refresh 幂等 + 孤儿设备重匹配"
# ═══════════════════════════════════════════════════════════════════════════
req POST "/api/v1/products/cache/refresh"
check_ret_ok "cache/refresh 第一次"
assert_eq "refreshed=true" "$(jget data.refreshed)" "True"
req POST "/api/v1/products/cache/refresh"
check_ret_ok "cache/refresh 第二次（幂等）"

req POST "/api/v1/products/orphan-devices/rematch"
check_ret_ok "孤儿设备重匹配触发"
REMATCH_STATUS=$(jget data.status)
if [ "$REMATCH_STATUS" = "accepted" ] || [ "$REMATCH_STATUS" = "running" ]; then
    pass "rematch 状态合法 (status=$REMATCH_STATUS)"
else
    fail "rematch 状态合法" "期望 accepted|running，实际 '$REMATCH_STATUS'"
fi

# ═══════════════════════════════════════════════════════════════════════════
section "5. param-models 字典（builtin 种子）"
# ═══════════════════════════════════════════════════════════════════════════
req GET "/api/v1/param-models"
check_ret_ok "参数模型列表可查"
check_list_nonempty "builtin 参数模型非空" "data.items"
PM_NAME="$TR_PM_NAME"
if [ -z "$PM_NAME" ]; then
    PM_NAME=$(jget data.items.0.name)
fi
PM_LOADED_FROM=$(printf '%s' "$BODY" | PMN="$PM_NAME" python3 -c "
import sys, json, os
d = json.load(sys.stdin)
items = (d.get('data') or {}).get('items') or []
for it in items:
    if it.get('name') == os.environ['PMN']:
        print(it.get('loaded_from') or '')
        break
" 2>/dev/null)

if [ -n "$PM_NAME" ]; then
    req GET "/api/v1/param-models/$PM_NAME"
    check_ret_ok "参数模型详情可查（${PM_NAME}）"
    assert_eq "详情 name 一致" "$(jget data.name)" "$PM_NAME"
    assert_eq "builtin 模型 deletable=false" "$(jget data.deletable)" "False"

    req GET "/api/v1/param-models/$PM_NAME/mappings"
    check_ret_ok "默认映射列表可查"
    check_list_nonempty "builtin 默认映射非空" "data.items"
    EXTRACT=$(printf '%s' "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = (d.get('data') or {}).get('items') or []
sp = pp = bmid = ''
for it in items:
    s = it.get('standard_path') or ''
    p = it.get('private_path') or ''
    if not bmid and it.get('source') == 'builtin':
        bmid = str(it.get('id') or '')
    if (not sp and s and p and '{i}' not in s and '{i}' not in p
            and it.get('is_active') and it.get('is_supported')
            and it.get('entry_type') == 'parameter' and it.get('source') == 'builtin'):
        sp, pp = s, p
print(sp + '|' + pp + '|' + bmid)
" 2>/dev/null)
    IFS='|' read -r STD_PATH PRIV_PATH BUILTIN_MAPPING_ID <<< "$EXTRACT"
else
    skip "参数模型详情/映射" "参数模型列表为空，取不到 name"
fi

req GET "/api/v1/param-models/standard"
check_ret_ok "标准参数树可查"
check_list_nonempty "标准参数树非空（builtin）" "data.items"
STD_TREE_PATH=$(printf '%s' "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = (d.get('data') or {}).get('items') or []
for it in items:
    s = it.get('standard_path') or ''
    if s and '{i}' not in s:
        print(s)
        break
" 2>/dev/null)
if [ -n "$STD_TREE_PATH" ]; then
    req GET "/api/v1/param-models/standard/$STD_TREE_PATH"
    check_ret_ok "标准参数单点详情可查"
    assert_eq "标准参数 path 回读一致" "$(jget data.standard_path)" "$STD_TREE_PATH"
else
    skip "标准参数单点详情" "标准树未找到无 {i} 占位的路径"
fi

if [ -n "$PM_LOADED_FROM" ]; then
    req GET "/api/v1/param-models/file-content?loaded_from=$PM_LOADED_FROM"
    check_status "XML 原文件可下载（${PM_LOADED_FROM}）" 200
    case "$BODY" in
        *parameterModel*) pass "下载内容为 parameterModel XML" ;;
        *) fail "下载内容为 parameterModel XML" "响应中未见 parameterModel 根元素" ;;
    esac
else
    skip "XML 原文件下载" "未取到 loaded_from"
fi
req GET "/api/v1/param-models/file-content?loaded_from=../../etc/passwd"
check_ret_fail "file-content 路径逃逸被拒"

req GET "/api/v1/param-models/no-such-model-${SMOKE_TAG}"
check_status "不存在的参数模型 404" 404

# ═══════════════════════════════════════════════════════════════════════════
section "6. translate 双向翻译（builtin 真实映射往返）"
# ═══════════════════════════════════════════════════════════════════════════
if [ -n "$TR_PRODUCT_ID" ] && [ -n "$STD_PATH" ] && [ -n "$PRIV_PATH" ]; then
    req POST "/api/v1/param-models/translate" "{\"productId\":\"$TR_PRODUCT_ID\",\"direction\":\"to_private\",\"paths\":[\"$STD_PATH\"]}"
    check_ret_ok "standardPath → privatePath 翻译"
    assert_eq "to_private 命中 found=true" "$(jget data.results.0.found)" "True"
    assert_eq "to_private 译文与映射表一致" "$(jget data.results.0.translated)" "$PRIV_PATH"

    req POST "/api/v1/param-models/translate" "{\"productId\":\"$TR_PRODUCT_ID\",\"direction\":\"to_standard\",\"paths\":[\"$PRIV_PATH\"]}"
    check_ret_ok "privatePath → standardPath 回翻"
    assert_eq "to_standard 命中 found=true" "$(jget data.results.0.found)" "True"
    assert_eq "双向翻译往返一致" "$(jget data.results.0.translated)" "$STD_PATH"
else
    skip "translate 双向翻译" "缺产品/映射数据（TR_PRODUCT_ID='$TR_PRODUCT_ID' STD_PATH='$STD_PATH'）"
fi

if [ -n "$TR_PRODUCT_ID" ]; then
    req POST "/api/v1/param-models/translate" "{\"productId\":\"$TR_PRODUCT_ID\",\"direction\":\"sideways\",\"paths\":[\"Device.X\"]}"
    check_ret_fail "translate 非法 direction 被拒"
fi
req POST "/api/v1/param-models/translate" "{\"direction\":\"to_private\",\"paths\":[\"Device.X\"]}"
check_ret_fail "translate 缺 productId 被拒"

# ═══════════════════════════════════════════════════════════════════════════
section "7. standard 参数治理闭环（${SMOKE_TAG} 自建条目）"
# ═══════════════════════════════════════════════════════════════════════════
SMK_STD="X_SMK_${SMOKE_TAG}.Test.Param"
req POST "/api/v1/param-models/standard" "{\"standard_path\":\"$SMK_STD\",\"entry_type\":\"parameter\",\"access\":\"RW\",\"data_type\":\"string\"}"
check_ret_ok "新增 standard 参数"
if [ "$(jget data.standard_path)" = "$SMK_STD" ]; then
    req GET "/api/v1/param-models/standard/$SMK_STD"
    check_ret_ok "自建 standard 参数可查"

    req PUT "/api/v1/param-models/standard/$SMK_STD" "{\"standard_path\":\"$SMK_STD\",\"entry_type\":\"parameter\",\"access\":\"RW\",\"data_type\":\"int\"}"
    check_ret_ok "更新 standard 参数"
    assert_eq "data_type 更新生效" "$(jget data.data_type)" "int"

    req DELETE "/api/v1/param-models/standard/$SMK_STD"
    check_ret_ok "删除自建 standard 参数"
    req GET "/api/v1/param-models/standard/$SMK_STD"
    check_status "删除后 standard 参数 404" 404
    SMK_STD=""
else
    skip "standard 闭环后续" "新增 standard 参数未回读到 standard_path"
fi

# ═══════════════════════════════════════════════════════════════════════════
section "8. custom mapping 闭环（builtin 模型上增改删 custom 行）"
# ═══════════════════════════════════════════════════════════════════════════
if [ -n "$PM_NAME" ]; then
    req POST "/api/v1/param-models/$PM_NAME/mappings" "{\"standard_path\":\"X_SMK_${SMOKE_TAG}.Map\",\"private_path\":\"Device.X_SMK_${SMOKE_TAG}.Map\",\"entry_type\":\"parameter\",\"access\":\"RW\",\"data_type\":\"string\"}"
    check_ret_ok "新增 custom 映射"
    MAPPING_ID=$(jget data.id)
    assert_eq "新增映射 source=custom" "$(jget data.source)" "custom"
    assert_eq "custom 映射 deletable=true" "$(jget data.deletable)" "True"

    if [ -n "$MAPPING_ID" ]; then
        req PUT "/api/v1/param-models/$PM_NAME/mappings/$MAPPING_ID" "{\"private_path\":\"Device.X_SMK_${SMOKE_TAG}.MapV2\"}"
        check_ret_ok "更新 custom 映射 private_path"
        assert_eq "private_path 更新生效" "$(jget data.private_path)" "Device.X_SMK_${SMOKE_TAG}.MapV2"

        # 重复 standard_path 应 409（与 builtin 已有映射撞键）
        if [ -n "$STD_PATH" ]; then
            req POST "/api/v1/param-models/$PM_NAME/mappings" "{\"standard_path\":\"$STD_PATH\",\"private_path\":\"Device.X_SMK_${SMOKE_TAG}.Dup\",\"entry_type\":\"parameter\"}"
            check_status "重复 standard_path 撞键 409" 409
        fi

        req DELETE "/api/v1/param-models/$PM_NAME/mappings/$MAPPING_ID"
        check_ret_ok "删除 custom 映射"
        MAPPING_ID=""
    else
        skip "custom 映射更新/删除" "新增映射未返回 id"
    fi

    if [ -n "$BUILTIN_MAPPING_ID" ]; then
        req DELETE "/api/v1/param-models/$PM_NAME/mappings/$BUILTIN_MAPPING_ID"
        check_status "builtin 映射删除被拒" 403
    else
        skip "builtin 映射删除守门" "未取到 builtin 映射 id"
    fi
else
    skip "custom mapping 闭环" "未取到参数模型 name"
fi

# ═══════════════════════════════════════════════════════════════════════════
section "9. builtin 删除守门 + upload-xml 闭环 + PUT 元信息治理"
# ═══════════════════════════════════════════════════════════════════════════
if [ -n "$PM_NAME" ]; then
    req DELETE "/api/v1/param-models/$PM_NAME"
    check_status "builtin 参数模型删除被拒" 403
else
    skip "builtin 参数模型删除守门" "未取到参数模型 name"
fi
req DELETE "/api/v1/param-models/no-such-model-${SMOKE_TAG}"
check_status "删除不存在参数模型 404" 404

# ── PUT/:name 非法 id 负路径（不存在的模型改元信息 → 404）──────────────────
req PUT "/api/v1/param-models/no-such-model-${SMOKE_TAG}" "{\"description\":\"smoke-noop\"}"
check_status "更新不存在参数模型 404" 404

# ── upload-xml 负路径（destructive 端点之前，先验参数校验红线）─────────────
#    a) 缺 file multipart 字段 → 400
req_upload "/api/v1/param-models/upload-xml" "notfile=ignored"
check_ret_fail "upload-xml 缺 file 字段被拒"

#    b) XML 缺 paramModel 属性（取不到唯一名）→ 400
PM_NONAME_XML="$SMOKE_TMPDIR/pm_noname.xml"
cat > "$PM_NONAME_XML" <<'PMNOXML'
<?xml version="1.0" encoding="UTF-8"?>
<parameterModel totalEntries="0"><parameters></parameters></parameterModel>
PMNOXML
req_upload "/api/v1/param-models/upload-xml" "file=@${PM_NONAME_XML};type=text/xml"
check_ret_fail "upload-xml 缺 paramModel 名被拒"

#    c) 根元素非 <parameterModel> → 400
PM_BADROOT_XML="$SMOKE_TMPDIR/pm_badroot.xml"
cat > "$PM_BADROOT_XML" <<'PMBADXML'
<?xml version="1.0" encoding="UTF-8"?>
<wrongRoot paramModel="smk-bad"></wrongRoot>
PMBADXML
req_upload "/api/v1/param-models/upload-xml" "file=@${PM_BADROOT_XML};type=text/xml"
check_ret_fail "upload-xml 根元素非法被拒"

# ── upload-xml 正路径（全新唯一名 + 不带 ?force，安全非破坏）──────────────
#    模型名仅含 [a-z0-9]（SMOKE_TAG 形态）+ 前缀，满足文件名白名单 [A-Za-z0-9_-]{1,64}。
PM_UPLOAD_NAME="smkpm${SMOKE_TAG}"
PM_UPLOAD_XML="$SMOKE_TMPDIR/${PM_UPLOAD_NAME}.xml"
cat > "$PM_UPLOAD_XML" <<PMUPXML
<?xml version="1.0" encoding="UTF-8"?>
<parameterModel paramModel="${PM_UPLOAD_NAME}" totalEntries="1">
    <parameters>
        <param name="Device.DeviceInfo.SerialNumber" standardPath="Device.DeviceInfo.SerialNumber" access="READ_ONLY" type="STRING" changeApplies="Immediate"/>
    </parameters>
</parameterModel>
PMUPXML

req_upload "/api/v1/param-models/upload-xml" "file=@${PM_UPLOAD_XML};type=text/xml"
check_ret_ok "upload-xml 上传最小合法 parameterModel（全新唯一名）"
# 红线复核：全新名上传绝不删任何既有 builtin（orphans_deleted 必须为 0）
assert_eq "上传未覆盖既有文件 overwritten=false" "$(jget data.overwritten)" "False"
assert_eq "上传未误删任何模型 orphans_deleted=0" "$(jget data.orphans_deleted)" "0"

if [ "$(jget data.model_name)" = "$PM_UPLOAD_NAME" ]; then
    # 列表 / 详情可见，且判定为 custom 可删
    req GET "/api/v1/param-models/$PM_UPLOAD_NAME"
    check_ret_ok "上传后参数模型详情可查"
    assert_eq "上传模型 name 回读一致" "$(jget data.name)" "$PM_UPLOAD_NAME"
    assert_eq "上传模型 source=custom" "$(jget data.source)" "custom"
    assert_eq "上传 custom 模型 deletable=true" "$(jget data.deletable)" "True"

    # PUT/:name 元信息治理（自建 custom 模型，删除即完全回滚）
    req PUT "/api/v1/param-models/$PM_UPLOAD_NAME" "{\"description\":\"smk governed ${SMOKE_TAG}\",\"is_active\":false}"
    check_ret_ok "PUT 更新自建参数模型元信息（description + is_active）"
    assert_eq "description 更新生效" "$(jget data.description)" "smk governed ${SMOKE_TAG}"
    assert_eq "is_active 更新生效" "$(jget data.is_active)" "False"

    req PUT "/api/v1/param-models/$PM_UPLOAD_NAME" "{\"is_active\":true}"
    check_ret_ok "PUT 恢复 is_active=true（部分字段更新）"
    assert_eq "is_active 恢复生效" "$(jget data.is_active)" "True"

    # 清理：删除自建 custom 模型（连带 sidecar + DB 行 + XML 备份）
    req DELETE "/api/v1/param-models/$PM_UPLOAD_NAME"
    check_ret_ok "删除自建 custom 参数模型（清理）"
    req GET "/api/v1/param-models/$PM_UPLOAD_NAME"
    check_status "删除后参数模型 404" 404
    PM_UPLOAD_NAME=""
else
    skip "upload-xml 闭环后续（详情/PUT/删除）" "上传未回读到 model_name"
fi

# ═══════════════════════════════════════════════════════════════════════════
section "10. discovered 视图 + destructive 负路径"
# ═══════════════════════════════════════════════════════════════════════════
if [ -n "$BUILTIN_PRODUCT_ID" ]; then
    req GET "/api/v1/products/$BUILTIN_PRODUCT_ID/discovered?swVersion=smoke-none-${SMOKE_TAG}"
    check_list_or_empty "discovered 映射视图可查" "data.items"
    req GET "/api/v1/products/$BUILTIN_PRODUCT_ID/discovered/versions"
    check_list_or_empty "discovered swVersion 列表可查" "data.items"
    req GET "/api/v1/products/$BUILTIN_PRODUCT_ID/discovered"
    check_ret_fail "discovered 缺 swVersion 被拒"
else
    skip "discovered 视图" "未取到 builtin 产品 id"
fi
# destructive 端点仅做参数校验负路径，绝不真删
req DELETE "/api/v1/products/not-a-uuid/discovered"
check_ret_fail "重置 discovered 非法 id 被拒"
req DELETE "/api/v1/products/not-a-uuid/discovered/versions/v1"
check_ret_fail "删除 discovered 版本非法 id 被拒"

# ═══════════════════════════════════════════════════════════════════════════
section "11. 权限边界（未认证 401 + 普通角色 viewer 403）"
# ═══════════════════════════════════════════════════════════════════════════
req_noauth GET "/api/v1/param-models"
check_status "未认证访问 param-models 401" 401
req_noauth GET "/api/v1/products"
check_status "未认证访问 products 401" 401

# 自建 viewer 用户验证 super_admin 守门（结束后删除）
req GET "/api/v1/admin/roles?page=1&page_size=100"
VIEWER_ROLE_ID=$(printf '%s' "$BODY" | python3 -c "
import sys, json
d = json.load(sys.stdin)
items = (d.get('data') or {}).get('items') or []
rid = ''
for it in items:
    if it.get('name') == 'viewer':
        rid = str(it.get('id') or '')
        break
if not rid:
    for it in items:
        if it.get('name') != 'admin' and not it.get('is_system'):
            rid = str(it.get('id') or '')
            break
print(rid)
" 2>/dev/null)

if [ -n "$VIEWER_ROLE_ID" ]; then
    SMK_USER_PASS="SmkPass@123"
    ENC_PASS=$(encrypt_password "$SMK_USER_PASS")
    req POST "/api/v1/admin/users" "{\"username\":\"u${SMOKE_TAG}\",\"encrypted_password\":\"$ENC_PASS\",\"key_id\":\"$PUBLIC_KEY_ID\",\"display_name\":\"smoke viewer\",\"role_ids\":[\"$VIEWER_ROLE_ID\"]}"
    check_ret_ok "创建 viewer 普通用户"
    SMK_USER_ID=$(jget data.id)

    if [ -n "$SMK_USER_ID" ]; then
        ENC_PASS=$(encrypt_password "$SMK_USER_PASS")
        LOGIN_RESP=$(curl --max-time 10 -s -X POST "$API/auth/login" \
            -H 'Content-Type: application/json' \
            -d "{\"username\":\"u${SMOKE_TAG}\",\"encrypted_password\":\"$ENC_PASS\",\"key_id\":\"$PUBLIC_KEY_ID\"}")
        VIEWER_TOKEN=$(printf '%s' "$LOGIN_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print((d.get('data') or {}).get('access_token',''))" 2>/dev/null)
        if [ -n "$VIEWER_TOKEN" ]; then
            ADMIN_TOKEN="$TOKEN"
            TOKEN="$VIEWER_TOKEN"
            req GET "/api/v1/param-models"
            check_status "viewer 访问 param-models 403" 403
            req GET "/api/v1/products"
            check_status "viewer 访问 products 403" 403
            TOKEN="$ADMIN_TOKEN"
        else
            skip "viewer 403 边界" "viewer 用户登录失败：$(printf '%s' "$LOGIN_RESP" | head -c 160)"
        fi
        req DELETE "/api/v1/admin/users/$SMK_USER_ID"
        check_ret_ok "删除 viewer 普通用户（清理）"
        SMK_USER_ID=""
    else
        skip "viewer 403 边界" "viewer 用户创建未返回 id"
    fi
else
    skip "viewer 403 边界" "角色列表中找不到可用的非管理员角色"
fi

# ═══════════════════════════════════════════════════════════════════════════
# 兜底清理（闭环中途失败时残留实体的 best-effort 删除，不计断言）
# ═══════════════════════════════════════════════════════════════════════════
if [ -n "$NEW_PRODUCT_ID" ]; then
    req DELETE "/api/v1/products/$NEW_PRODUCT_ID"
    echo "  [cleanup] 残留产品 $NEW_PRODUCT_ID → HTTP $HTTP_CODE"
fi
if [ -n "$MAPPING_ID" ] && [ -n "$PM_NAME" ]; then
    req DELETE "/api/v1/param-models/$PM_NAME/mappings/$MAPPING_ID"
    echo "  [cleanup] 残留映射 $MAPPING_ID → HTTP $HTTP_CODE"
fi
if [ -n "$SMK_STD" ]; then
    req DELETE "/api/v1/param-models/standard/$SMK_STD"
    echo "  [cleanup] 残留 standard 参数 $SMK_STD → HTTP $HTTP_CODE"
fi
if [ -n "$PM_UPLOAD_NAME" ]; then
    req DELETE "/api/v1/param-models/$PM_UPLOAD_NAME"
    echo "  [cleanup] 残留自建参数模型 $PM_UPLOAD_NAME → HTTP $HTTP_CODE"
fi
if [ -n "$SMK_USER_ID" ]; then
    req DELETE "/api/v1/admin/users/$SMK_USER_ID"
    echo "  [cleanup] 残留用户 $SMK_USER_ID → HTTP $HTTP_CODE"
fi

smoke_summary
