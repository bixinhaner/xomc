#!/usr/bin/env bash
# =============================================================================
# smoke_system.sh — F06 系统配置 业务冒烟
#
# 覆盖（/tmp/smoke_routes.json key ∈ {sysconfig} 全部路由）：
#   - GET    /api/v1/admin/sysConfig                      系统配置列表（data 为纯数组）
#   - POST   /api/v1/admin/sysConfig                      创建自定义键（闭环后删除）
#   - GET    /api/v1/admin/sysConfig/:id                  单条查询
#   - PUT    /api/v1/admin/sysConfig/:id                  更新 value/desc
#   - DELETE /api/v1/admin/sysConfig/:id                  删除（清理冒烟自建键）
#   - POST   /api/v1/admin/sysConfig/batch                按 category 批量 upsert
#   - GET    /api/v1/admin/sysDictionary/getSysDictionaryList   字典列表（data.list/total）
#   - GET    /api/v1/admin/sysDictionary/batch?codes=     批量取字典（注意参数名是 codes 非 types）
#   - GET    /api/v1/admin/sysDictionaryDetail/getSysDictionaryDetailList  字典明细列表
#   - GET    /api/v1/admin/logs/{login,operation,task}    登录/操作/任务日志
#   - GET    /api/v1/system/info                          版本/运行信息（断言 version）
#   - GET    /api/v1/system-license                       单例许可（200 或 404+biz_code 12113）
#   - GET    /api/v1/system-license/history               许可替换历史
#   - POST   /api/v1/system-license                       destructive — 只测无效内容校验负路径
#   - POST   /api/v1/admin/uploads/ui-asset               只测非法 kind 负路径（无删除端点，真传会残留 MinIO）
#   - POST   /api/v1/admin/dictload/reload                SKIP（热重载有全局影响，见脚本内注释）
#
# 后端契约（internal/admin/{sys_config*,dictionary_*,sys_log_handler,ui_asset_handler}.go、
#           internal/license/system_license_handler.go、internal/core/components/sysinfo.go）：
#   - sysConfig 列表 data 是 []SysConfig 纯数组（非 items 包裹）；Create 必填 category+key；
#     batch 必填 category + items(min=1)，items[].key 必填；id 非 UUID → 400，
#     不存在 → handler 统一 500 + ret=0（AbortWithError 透传 service 错误）。
#   - sysDictionary 列表返回 data{list,total}；batch 参数名是 **codes**（routes.json 注释写
#     ?types= 已过时）；明细列表嵌 model.ListRequest，必须显式 page/page_size 否则 binding 400。
#   - admin/logs/* 同样嵌 ListRequest，必须显式 page/page_size；返回 data{items,total,...}；
#     登录日志已接通写入（#122：auth Login 成功/失败异步写 sys_login_logs，硬断言非空）；
#     oper/task 日志写入链路仍未接通（#122 遗留），列表可为空。
#   - system-license：未配置时 GET → 404 + biz_code 12113（语义"未配置"非路由缺失）；
#     POST body {"raw_content":...}，解析失败 → 400 + 12111。
#
# 种子依赖（migrations/seed/000001_init_seed.sql，builtin 必有）：
#   - sys_configs：system/ui_custom/pm.retention 等 19 条 → 列表强断言非空；
#   - sys_dictionaries：gender/is_online/op_state 等 14 条 + details → 强断言非空。
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "F06 系统配置" "$@"
smoke_login

NOEXIST_ID="00000000-0000-0000-0000-00000000dead"
SMK_CATEGORY="smoke_${SMOKE_TAG}"

# ---------------------------------------------------------------------------
section "系统配置列表（/admin/sysConfig，前端启动依赖）"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/sysConfig"
check_ret_ok "系统配置全量列表可查"
check_list_nonempty "系统配置含 builtin 种子（system/ui_custom 等）" "data"

req GET "/api/v1/admin/sysConfig?category=system"
check_ret_ok "系统配置按 category=system 过滤"
check_list_nonempty "system 类配置非空（seed: system_name 等）" "data"
check_field "system 类配置首条含 key 字段" "data.0.key"

req GET "/api/v1/admin/sysConfig?category=system&public=true"
check_ret_ok "系统配置 public=true 过滤"
check_list_nonempty "公开配置非空（seed: system_name is_public=true）" "data"

# ---------------------------------------------------------------------------
section "sysConfig CRUD 闭环（建 ${SMOKE_TAG} 自定义键 → 查 → 改 → 删）"
# ---------------------------------------------------------------------------
req POST "/api/v1/admin/sysConfig" \
    "{\"category\":\"${SMK_CATEGORY}\",\"key\":\"${SMOKE_TAG}_k1\",\"value\":\"v1\",\"value_type\":\"string\",\"desc\":\"冒烟自建配置\",\"is_public\":false}"
check_ret_ok "创建冒烟自定义配置键"
check_field "创建返回 id" "data.id"
CFG_ID=$(jget data.id)

if [ -n "$CFG_ID" ]; then
    req GET "/api/v1/admin/sysConfig/${CFG_ID}"
    check_ret_ok "按 id 查询自建配置"
    V=$(jget data.value)
    if [ "$V" = "v1" ]; then pass "自建配置 value 回读一致 (v1)"
    else fail "自建配置 value 回读一致" "期望 v1，实际 '$V'"; fi

    req PUT "/api/v1/admin/sysConfig/${CFG_ID}" \
        "{\"value\":\"v2\",\"desc\":\"冒烟更新\"}"
    check_ret_ok "更新自建配置 value"
    V=$(jget data.value)
    if [ "$V" = "v2" ]; then pass "更新后 value=v2 已生效"
    else fail "更新后 value=v2 已生效" "实际 '$V'"; fi
else
    skip "按 id 查询/更新自建配置" "创建未返回 id，无法继续闭环"
fi

# 负路径：必填字段 / 非法 id
req POST "/api/v1/admin/sysConfig" "{\"key\":\"${SMOKE_TAG}_nocat\"}"
check_ret_fail "创建缺 category 被拒（binding required）"
req GET "/api/v1/admin/sysConfig/not-a-uuid"
check_ret_fail "非法 id 查询被拒"
req PUT "/api/v1/admin/sysConfig/not-a-uuid" "{\"value\":\"x\"}"
check_ret_fail "非法 id 更新被拒"
req DELETE "/api/v1/admin/sysConfig/${NOEXIST_ID}"
check_ret_fail "不存在 id 删除被拒"

# ---------------------------------------------------------------------------
section "sysConfig batch 批量 upsert（同 category 覆写 + 新增）"
# ---------------------------------------------------------------------------
req POST "/api/v1/admin/sysConfig/batch" \
    "{\"category\":\"${SMK_CATEGORY}\",\"items\":[{\"key\":\"${SMOKE_TAG}_k1\",\"value\":\"v3\"},{\"key\":\"${SMOKE_TAG}_k2\",\"value\":\"batch-val\",\"value_type\":\"string\"}]}"
check_ret_ok "batch upsert（k1 覆写 + k2 新增）"
check_count_ge "batch 返回 updated≥2" "data.updated" 2

req GET "/api/v1/admin/sysConfig?category=${SMK_CATEGORY}"
check_ret_ok "冒烟 category 列表回查"
N=$(jlen data)
if [ "$N" -eq 2 ]; then pass "冒烟 category 下恰好 2 条 (k1+k2)"
else fail "冒烟 category 下恰好 2 条" "实际 ${N} 条"; fi
V=$(jget data.0.value)
if [ "$V" = "v3" ]; then pass "batch 覆写 k1 value=v3 已生效"
else fail "batch 覆写 k1 value=v3 已生效" "data.0.value='$V'"; fi
CFG_ID2=$(jget data.1.id)
# 容错：若创建步骤没拿到 id，从列表补取
[ -z "$CFG_ID" ] && CFG_ID=$(jget data.0.id)

# 负路径：items 为空（binding min=1）
req POST "/api/v1/admin/sysConfig/batch" "{\"category\":\"${SMK_CATEGORY}\",\"items\":[]}"
check_ret_fail "batch 空 items 被拒（binding min=1）"

# 清理：删除两个冒烟键并验证删干净
if [ -n "$CFG_ID" ]; then
    req DELETE "/api/v1/admin/sysConfig/${CFG_ID}"
    check_ret_ok "删除冒烟键 k1"
    req GET "/api/v1/admin/sysConfig/${CFG_ID}"
    check_ret_fail "已删除 id 不可再查（ret=0）"
else
    skip "删除冒烟键 k1" "未获取到 id"
fi
if [ -n "$CFG_ID2" ]; then
    req DELETE "/api/v1/admin/sysConfig/${CFG_ID2}"
    check_ret_ok "删除冒烟键 k2"
else
    skip "删除冒烟键 k2" "未获取到 id"
fi
req GET "/api/v1/admin/sysConfig?category=${SMK_CATEGORY}"
check_ret_ok "清理后冒烟 category 回查"
N=$(jlen data)
if [ "$N" -eq 0 ]; then pass "冒烟 category 已清空（无残留）"
else fail "冒烟 category 已清空" "仍残留 ${N} 条"; fi

# ---------------------------------------------------------------------------
section "数据字典（/admin/sysDictionary getSysDictionaryList + batch）"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/sysDictionary/getSysDictionaryList"
check_ret_ok "字典列表可查"
check_list_nonempty "字典列表含 builtin 种子（gender/is_online 等）" "data.list"
check_count_ge "字典 total≥1" "data.total" 1
DICT_ID=$(jget data.list.0.id)

# 从列表取 type 值供 batch 用（跳过含 . 的 type 如 time.Time，避免 jget 点分路径歧义）
D1=""; D2=""
i=0
while [ $i -lt 14 ]; do
    t=$(jget "data.list.${i}.type")
    [ -z "$t" ] && break
    case "$t" in
        *.*) ;;
        *)
            if [ -z "$D1" ]; then D1="$t"
            elif [ -z "$D2" ] && [ "$t" != "$D1" ]; then D2="$t"; break
            fi
            ;;
    esac
    i=$((i + 1))
done
[ -z "$D1" ] && D1="is_online"
CODES="$D1"
[ -n "$D2" ] && CODES="${D1},${D2}"

req GET "/api/v1/admin/sysDictionary/batch?codes=${CODES}"
check_ret_ok "字典 batch 批量拉取（codes=${CODES}）"
check_field "batch 返回字典 ${D1}" "data.dicts.${D1}.type"
if [ -n "$D2" ]; then
    check_field "batch 返回字典 ${D2}" "data.dicts.${D2}.type"
fi

# 负路径：缺 codes 参数
req GET "/api/v1/admin/sysDictionary/batch"
check_ret_fail "batch 缺 codes 参数被拒"

# ---------------------------------------------------------------------------
section "字典明细（/admin/sysDictionaryDetail getSysDictionaryDetailList）"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/sysDictionaryDetail/getSysDictionaryDetailList?page=1&page_size=10"
check_ret_ok "字典明细列表可查（显式 page/page_size）"
check_list_nonempty "字典明细含 builtin 种子" "data.list"
check_count_ge "字典明细 total≥1" "data.total" 1

if [ -n "$DICT_ID" ]; then
    req GET "/api/v1/admin/sysDictionaryDetail/getSysDictionaryDetailList?page=1&page_size=10&sysDictionaryId=${DICT_ID}"
    check_ret_ok "字典明细按 sysDictionaryId=${DICT_ID} 过滤"
else
    skip "字典明细按 sysDictionaryId 过滤" "字典列表未返回 id"
fi

# 负路径：binding min=1
req GET "/api/v1/admin/sysDictionaryDetail/getSysDictionaryDetailList?page=0&page_size=10"
check_ret_fail "字典明细 page=0 被拒（binding min=1）"

# ---------------------------------------------------------------------------
section "系统日志（/admin/logs/{login,operation,task}）"
# ---------------------------------------------------------------------------
req GET "/api/v1/admin/logs/login?page=1&page_size=10"
check_ret_ok "登录日志可查"
check_list_or_empty "登录日志 data.items" "data.items"
check_field "登录日志含 total 字段" "data.total"
# #122 已修复：auth Login 成功/失败路径异步写 sys_login_logs（auth_handler 调
# CreateLoginLog）。本套件登录频繁，登录日志可硬断言非空。
# oper/task 日志写入链路仍未接通（#122 遗留），下方维持 check_list_or_empty。
check_list_nonempty "登录日志在成功登录后非空（#122）" "data.items"

req GET "/api/v1/admin/logs/login?page=1&page_size=10&username=admin"
check_ret_ok "登录日志按 username=admin 过滤"

req GET "/api/v1/admin/logs/operation?page=1&page_size=10"
check_ret_ok "操作日志可查"
check_list_or_empty "操作日志 data.items" "data.items"
check_field "操作日志含 total 字段" "data.total"

req GET "/api/v1/admin/logs/task?page=1&page_size=10"
check_ret_ok "任务日志可查"
check_list_or_empty "任务日志 data.items" "data.items"
check_field "任务日志含 total 字段" "data.total"

# 负路径：binding min=1（嵌 model.ListRequest，page 必填）
req GET "/api/v1/admin/logs/login?page=0&page_size=10"
check_ret_fail "登录日志 page=0 被拒（binding min=1）"

# ---------------------------------------------------------------------------
section "系统信息（/system/info，断言 version）"
# ---------------------------------------------------------------------------
req GET "/api/v1/system/info"
check_ret_ok "系统信息可查"
check_field "系统信息含 version" "data.version"
check_field "系统信息含 db_status" "data.db_status"
check_field "系统信息含 server_time" "data.server_time"

# ---------------------------------------------------------------------------
section "系统 License（/system-license，未配置 404+12113 合法）"
# ---------------------------------------------------------------------------
req GET "/api/v1/system-license"
check_status_in "系统 License 读取（200=已配置 / 404=未配置）" "200 404"
if [ "$HTTP_CODE" = "404" ]; then
    BIZ=$(jget biz_code)
    if [ "$BIZ" = "12113" ]; then pass "未配置语义正确（biz_code=12113）"
    else fail "未配置语义正确" "期望 biz_code=12113，实际 '$BIZ'"; fi
else
    check_field "已配置 License 含 license_id" "data.license_id"
fi

req GET "/api/v1/system-license/history?page=1&page_size=10"
check_ret_ok "License 替换历史可查"
check_list_or_empty "License 历史 data.items" "data.items"

# destructive POST — 只测无效内容校验负路径（绝不上传真实 license）
req POST "/api/v1/system-license" "{\"raw_content\":\"not-a-valid-license-${SMOKE_TAG}\"}"
check_ret_fail "无效 license 内容被拒（解析失败 400+12111）"
req POST "/api/v1/system-license" "{}"
check_ret_fail "空 raw_content 被拒"

# ---------------------------------------------------------------------------
section "UI 资产上传 / 字典热重载（负路径 + SKIP）"
# ---------------------------------------------------------------------------
# ui-asset：handler 无 DELETE 端点，真传 PNG 会在 MinIO 留下孤儿对象 → 只测非法 kind 负路径
req_upload "/api/v1/admin/uploads/ui-asset" "kind=bogus_${SMOKE_TAG}"
check_ret_fail "ui-asset 非法 kind 被拒（白名单 login_bg/logo_small/logo_large）"
skip "ui-asset 真实上传闭环" "无删除端点，上传后 MinIO 对象无法清理（internal/admin/ui_asset_handler.go 仅 Upload/Serve）"

# dictload/reload：会热重载全进程字典 Registry（param-model/indicator/alarm 等），全局影响，不在冒烟触发
skip "dictload 热重载（POST /admin/dictload/reload）" "热重载影响全进程字典状态（super_admin 维护操作），冒烟不触发"

smoke_summary
