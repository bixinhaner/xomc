#!/usr/bin/env bash
# 冒烟：原始 PM/MR 文件压缩回写（issue #321 + 加固）
#
# 验证「基站上传的明文 XML，解析入库后被压缩一次回写 MinIO」这条铁律在 v 真栈成立：
#   1. 用真设备 SN 把明文 PM / MR 样本上传到 ACS FileUploadService（fileType=PM/MR）；
#   2. 等 worker 入库 + 内联压缩回写；
#   3. 直接查 MinIO 对象：首 2 字节须为 gzip 魔数 1f8b、Content-Encoding=gzip、且能解压还原
#      （= 压缩态对象后续仍可被入库侧 MaybeGunzip 正确再解析）。
#
# 依赖 docker exec 进 MinIO 容器跑 mc（与 smoke_retention.sh 同范式）。无 docker / MinIO
# 容器 / ACS 不可达 / 无可用设备时整段 skip，不误报 fail。
#
# 环境变量（默认值对齐本机偏移端口栈）：
#   OMC_ACS_URL（默认 http://localhost:7557）          ACS 上传口
#   OMC_WORKER_METRICS_URL（默认 http://localhost:9092）worker Prometheus 指标
#   OMC_MINIO_CONTAINER（默认 omc-minio-1）             MinIO 容器名
#   OMC_MINIO_USER/OMC_MINIO_PASS（默认 minioadmin）    MinIO 根凭据
#   OMC_PM_BUCKET/OMC_MR_BUCKET（默认 pm-files/mr-files）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib.sh"
smoke_init "原始文件压缩回写(#321)" "$@"
smoke_login

ACS_UPLOAD_URL="${OMC_ACS_URL:-http://localhost:7557}"
WORKER_METRICS_URL="${OMC_WORKER_METRICS_URL:-http://localhost:9092}"
MINIO_CONTAINER="${OMC_MINIO_CONTAINER:-omc-minio-1}"
MINIO_USER="${OMC_MINIO_USER:-minioadmin}"
MINIO_PASS="${OMC_MINIO_PASS:-minioadmin}"
PM_BUCKET="${OMC_PM_BUCKET:-pm-files}"
MR_BUCKET="${OMC_MR_BUCKET:-mr-files}"

# 在 MinIO 容器内跑一条 mc 命令（已配 alias lo）。
mc_run() {
    docker exec "$MINIO_CONTAINER" sh -c \
        "mc alias set lo http://localhost:9000 ${MINIO_USER} ${MINIO_PASS} >/dev/null 2>&1; $1" 2>/dev/null
}
# 在 bucket 里按文件名子串找到完整对象键（host 侧 grep，容器无 grep）。
mc_find() {
    local bucket="$1" needle="$2"
    mc_run "mc ls --recursive lo/${bucket}" | awk '{print $NF}' | grep -F "$needle" | head -1
}
# 对象首 2 字节十六进制（host 侧 od，容器无 xxd）。
mc_magic() {
    mc_run "mc cat lo/$1/$2" | head -c 2 | od -An -tx1 | tr -d ' \n'
}
# 对象 Content-Encoding 元数据。
mc_encoding() {
    mc_run "mc stat lo/$1/$2" | awk -F':' '/[Cc]ontent-[Ee]ncoding/{gsub(/^[ \t]+|[ \t]+$/,"",$2);print $2}'
}
# 上传一个文件到 ACS（fileType=PM/MR），回显 HTTP 码。
acs_upload() {
    local ftype="$1" fname="$2" sn="$3" body="$4" extra="$5"
    curl -s --max-time 10 -o /dev/null -w '%{http_code}' \
        -X POST "${ACS_UPLOAD_URL}/smallcell/FileUploadService?fileType=${ftype}&filename=${fname}&sn=${sn}${extra}" \
        --data-binary "$body" -H "Content-Type: application/octet-stream" 2>/dev/null || echo "000"
}

# 一段高度可压缩的明文 PM XML（重复 measType/measValue 块，确保 gzip 有正收益）。
gen_pm_xml() {
    printf '%s' '<?xml version="1.0" encoding="UTF-8"?><measCollecFile xmlns="http://www.3gpp.org/ftp/specs/archive/32_series/32.435#measCollec"><fileHeader vendorName="SmokeVendor"/><measData><managedElement localDn="SubNetwork=1,MeContext=SMOKE"/><measInfo measInfoId="PM_Counters"><granPeriod duration="PT900S" endTime="2026-06-14T00:15:00Z"/>'
    local i=0
    while [ "$i" -lt 60 ]; do
        printf '<measType p="%d">RRC.ConnEstabAtt</measType><measValue measObjLdn="Cell=%d"><r p="%d">12345</r></measValue>' "$i" "$i" "$i"
        i=$((i + 1))
    done
    printf '%s' '</measInfo></measData></measCollecFile>'
}
# 明文 MRO XML（重复 object 块，确保可压缩）。
gen_mr_xml() {
    printf '%s' '<?xml version="1.0" encoding="UTF-8"?><bulkPmMrDataFile><fileHeader startTime="2026-06-14T00:00:00Z" reportingPeriod="PT900S"/><eNB id="SMOKE"><measurement mrType="MRO"><smr>MR.LteScRSRP MR.LteScRSRQ</smr>'
    local i=0
    while [ "$i" -lt 60 ]; do
        printf '<object id="%d"><v>-95 -10</v></object>' "$i"
        i=$((i + 1))
    done
    printf '%s' '</measurement></eNB></bulkPmMrDataFile>'
}

# 上传 → 轮询 MinIO 找到对象 → 断言 gzip。kind=PM/MR 仅用于文案。
verify_compressed() {
    local kind="$1" bucket="$2" fname="$3"
    local key="" waited=0
    while [ "$waited" -lt 45 ]; do
        key=$(mc_find "$bucket" "$fname")
        [ -n "$key" ] && break
        sleep 3; waited=$((waited + 3))
    done
    if [ -z "$key" ]; then
        fail "${kind} 明文上传后对象落 MinIO" "45s 内未在 ${bucket} 找到 ${fname}（入库未完成或上传失败）"
        return
    fi
    pass "${kind} 对象已落 MinIO (${key})"

    # 入库后内联压缩是异步的，对象 key 不变（覆盖回写）→ 轮询直到首字节变 gzip。
    local magic="" enc="" w2=0
    while [ "$w2" -lt 45 ]; do
        magic=$(mc_magic "$bucket" "$key")
        [ "$magic" = "1f8b" ] && break
        sleep 3; w2=$((w2 + 3))
    done
    if [ "$magic" = "1f8b" ]; then
        pass "${kind} MinIO 对象首字节为 gzip 魔数 1f8b（已压缩回写）"
    else
        fail "${kind} MinIO 对象应被压缩回写" "首字节=${magic:-空} 期望=1f8b（45s 内未压缩）"
        return
    fi

    enc=$(mc_encoding "$bucket" "$key")
    if [ "$enc" = "gzip" ]; then
        pass "${kind} 对象 Content-Encoding=gzip"
    else
        fail "${kind} 对象 Content-Encoding 应为 gzip" "实际=${enc:-空}"
    fi

    # 解压还原：证明压缩态对象后续仍可被入库侧 MaybeGunzip 正确再解析。
    if mc_run "mc cat lo/${bucket}/${key}" | gunzip 2>/dev/null | head -c 64 | grep -q '<?xml'; then
        pass "${kind} 对象 gunzip 解压还原成原始 XML（可再解析）"
    else
        fail "${kind} 压缩对象应能 gunzip 还原" "解压失败或非 XML"
    fi
}

# ---------------------------------------------------------------------------
section "前置：依赖可达性"
# ---------------------------------------------------------------------------
if ! command -v docker >/dev/null 2>&1; then
    skip "原始文件压缩回写验证" "本机无 docker，无法查 MinIO 对象"
    smoke_summary
fi
if ! docker exec "$MINIO_CONTAINER" true >/dev/null 2>&1; then
    skip "原始文件压缩回写验证" "MinIO 容器 ${MINIO_CONTAINER} 不可 exec（设 OMC_MINIO_CONTAINER）"
    smoke_summary
fi
ACS_CODE=$(curl -s --max-time 5 -o /dev/null -w '%{http_code}' -X POST "${ACS_UPLOAD_URL}/smallcell/FileUploadService?fileType=PM&filename=probe.xml&sn=PROBE" --data 'x' 2>/dev/null || echo "000")
if [ "$ACS_CODE" = "000" ]; then
    skip "原始文件压缩回写验证" "ACS 上传口 ${ACS_UPLOAD_URL} 不可达"
    smoke_summary
fi
pass "依赖可达：docker + MinIO 容器 + ACS 上传口"

# 取一个真设备 SN（PM/MR 入库需设备解析；无设备则 skip）。
req GET "/api/v1/devices?page=1&page_size=5"
DEV_SN=$(jget data.items.0.serial_number)
if [ -z "$DEV_SN" ] || [ "$DEV_SN" = "null" ]; then
    skip "原始文件压缩回写验证" "无可用设备（devices 列表为空），无法触发入库"
    smoke_summary
fi
pass "取到测试设备 SN=${DEV_SN}"

# ---------------------------------------------------------------------------
section "PM 明文上传 → MinIO 压缩回写"
# ---------------------------------------------------------------------------
PM_NAME="rawarchive-${SMOKE_TAG}-pm.xml"
PM_CODE=$(acs_upload "PM" "$PM_NAME" "$DEV_SN" "$(gen_pm_xml)" "")
if [ "$PM_CODE" = "200" ] || [ "$PM_CODE" = "201" ]; then
    pass "PM 明文上传 HTTP ${PM_CODE}"
    verify_compressed "PM" "$PM_BUCKET" "$PM_NAME"
else
    fail "PM 明文上传" "HTTP ${PM_CODE}"
fi

# ---------------------------------------------------------------------------
section "MR 明文上传 → MinIO 压缩回写"
# ---------------------------------------------------------------------------
MR_NAME="MRO_rawarchive-${SMOKE_TAG}.xml"
MR_CODE=$(acs_upload "MR" "$MR_NAME" "$DEV_SN" "$(gen_mr_xml)" "&cellCode=SMOKE")
if [ "$MR_CODE" = "200" ] || [ "$MR_CODE" = "201" ]; then
    pass "MR 明文上传 HTTP ${MR_CODE}"
    verify_compressed "MR" "$MR_BUCKET" "$MR_NAME"
else
    fail "MR 明文上传" "HTTP ${MR_CODE}"
fi

# ---------------------------------------------------------------------------
section "可观测：压缩回写指标"
# ---------------------------------------------------------------------------
if curl -s --max-time 5 "${WORKER_METRICS_URL}/metrics" 2>/dev/null | grep -q 'omc_raw_archive_total'; then
    pass "worker 暴露 omc_raw_archive_total 压缩回写指标"
else
    skip "压缩回写指标可见" "worker metrics 未暴露 omc_raw_archive_total（端点不可达或未压过）"
fi

smoke_summary
