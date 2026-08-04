#!/usr/bin/env bash
# =============================================================================
# plan-resources.sh —— OMC 部署前「资源动态规划」脚本（独立运行，install.sh 之前）
# =============================================================================
#
# 职责（对应需求 6 点中的 ①②③）：
#   ① 探测服务器硬件配置（CPU 核数 / 内存总量 / 磁盘）
#   ② 探测服务器当前状态与负荷（可用内存 MemAvailable、load average、已被
#      「其它项目」占用/预留的容器资源）—— 避免与其它项目共用时超分
#   ③ 按「空闲资源」对 compose 各组件计算资源分配，生成 resources.env
#
# 设计要点（详见同目录 RESOURCE-PLANNING.md）：
#   · 口径 = 空闲优先：以 MemAvailable 为基准，再扣除其它容器「已声明但未用」的
#     预留额（committed − used），只对真正空闲的资源分配。
#   · floor-first（关键）：每个组件先拿到 100k 基线下限（floor），下限之和构成
#     最低门槛；只有「剩余空闲」才按权重在 floor..ceiling 之间向上伸缩。
#     绝不低于 floor —— 否则重演 Redis 2.36G / Postgres 连接耗尽的 OOM 事故。
#   · 联动派生：limits.memory 一变，GOMEMLIMIT / PG shared_buffers / Redis
#     maxmemory 同步由「同一预算」算出，结构上杜绝「限额与进程内上限不一致」。
#   · 全量记账：nats/minio/web/monitoring 也计入预算，最后校验 Σ限额 ≤ 空闲预算。
#
# 产出：deploy/resources.env —— 纯文本、含注释，部署人员可查看并手改（需求 ④）。
#   install.sh 后续以 `--env-file resources.env` 传给 compose 动态部署（需求 ⑤）；
#   改 resources.env 后 `docker compose ... up -d` 即按新值重建容器（需求 ⑥）。
#
# 用法：
#   ./plan-resources.sh                 # 探测 + 计算 + 写 resources.env
#   ./plan-resources.sh --dry-run       # 只打印规划，不写文件
#   ./plan-resources.sh --skip-monitoring   # 不部署监控栈，降低门槛
#   ./plan-resources.sh --tier small|medium|large   # 手动指定档位（默认自动判定）
#   ./plan-resources.sh --assume-dedicated  # 视整机为 OMC 独占，不扣其它容器预留
#   ./plan-resources.sh --floor-tolerance-pct N  # 门禁容忍度（默认30，见下方说明）
#   ./plan-resources.sh -o /path/resources.env   # 指定输出路径
#   ./plan-resources.sh --lang cn|en          # 输出语言（默认 en）
#
# 退出码：0 成功；1 主机低于最低配置（含建议最低配）；2 参数错误。
#
# 兼容性：生产目标为 Linux（/proc、cgroup）。脚本同时支持 macOS 探测，便于在
#   本机（Mac）dry-run 预览（sysctl / vm_stat）。
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# 0. 参数解析
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$SCRIPT_DIR/storage-paths-lib.sh"
. "$SCRIPT_DIR/resource-env-lib.sh"
OUT_FILE="$SCRIPT_DIR/resources.env"
STORAGE_ENV_FILE="${OMC_STORAGE_ENV_FILE:-$SCRIPT_DIR/.env}"
OMC_LANG="${OMC_LANG:-en}"
DRY_RUN=0
SKIP_MONITORING=0
ASSUME_DEDICATED=0
TIER_OVERRIDE=""
# 门禁容忍度：空闲预算低于「下限之和」时，只要缺口不超过这个百分比就降级为 WARN
# 按下限分配（不再向上伸缩），而不是直接 FATAL 拒绝部署。压测/生产实测组件很少
# 同时打满 floor，留一点容忍度换可用性；超过该百分比说明缺口太大，仍然 die。
FLOOR_TOLERANCE_PCT=30

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run)          DRY_RUN=1 ;;
    --skip-monitoring)  SKIP_MONITORING=1 ;;
    --assume-dedicated) ASSUME_DEDICATED=1 ;;
    --tier)             TIER_OVERRIDE="${2:-}"; shift ;;
    --tier=*)           TIER_OVERRIDE="${1#*=}" ;;
    --floor-tolerance-pct) FLOOR_TOLERANCE_PCT="${2:?--floor-tolerance-pct 需要 0-99 的百分比}"; shift ;;
    --floor-tolerance-pct=*) FLOOR_TOLERANCE_PCT="${1#*=}" ;;
    -o|--output)        OUT_FILE="${2:?-o 需要路径}"; shift ;;
    -o=*|--output=*)    OUT_FILE="${1#*=}" ;;
    --lang|--language)  OMC_LANG="${2:?--lang 需要 cn 或 en}"; shift ;;
    --lang=*|--language=*) OMC_LANG="${1#*=}" ;;
    -h|--help)
      if [ "$OMC_LANG" = en ]; then
        cat <<'EOF'
plan-resources.sh - OMC dynamic resource planner

Usage:
  ./plan-resources.sh [options]

Options:
  --dry-run                    Print the plan without writing resources.env
  --skip-monitoring            Exclude the monitoring stack from the budget
  --assume-dedicated           Treat the host as dedicated to OMC
  --tier small|medium|large    Override automatic host tier selection
  --floor-tolerance-pct N      Allow a 0-99% floor budget gap (default 30)
  -o, --output <path>          Write resources.env to a specific path
  --lang, --language <cn|en>   Output language (default en)
  -h, --help                   Show this help
EOF
      else
        sed -n '2,55p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      fi
      exit 0 ;;
    *)
      if [ "$OMC_LANG" = en ]; then echo "Unknown option: $1 (see --help for usage)" >&2
      else echo "未知参数：$1（--help 查看用法）" >&2; fi
      exit 2 ;;
  esac
  shift
done

case "$OMC_LANG" in
  cn|en) ;;
  *)
    if [ "$OMC_LANG" = en ]; then echo "--lang accepts only cn or en, got: $OMC_LANG" >&2
    else echo "--lang 仅支持 cn 或 en，收到：$OMC_LANG" >&2; fi
    exit 2 ;;
esac

case "$TIER_OVERRIDE" in ""|small|medium|large) ;; *)
  if [ "$OMC_LANG" = en ]; then echo "--tier accepts only small|medium|large, got: $TIER_OVERRIDE" >&2
  else echo "--tier 仅支持 small|medium|large，收到：$TIER_OVERRIDE" >&2; fi
  exit 2 ;;
esac

case "$FLOOR_TOLERANCE_PCT" in
  ''|*[!0-9]*)
    if [ "$OMC_LANG" = en ]; then echo "--floor-tolerance-pct accepts an integer from 0 to 99, got: $FLOOR_TOLERANCE_PCT" >&2
    else echo "--floor-tolerance-pct 仅支持 0-99 的整数，收到：$FLOOR_TOLERANCE_PCT" >&2; fi
    exit 2 ;;
esac
[ "$FLOOR_TOLERANCE_PCT" -ge 100 ] && {
  if [ "$OMC_LANG" = en ]; then echo "--floor-tolerance-pct must be less than 100, got: $FLOOR_TOLERANCE_PCT" >&2
  else echo "--floor-tolerance-pct 必须 < 100，收到：$FLOOR_TOLERANCE_PCT" >&2; fi
  exit 2
}

# 颜色与日志
if [ -t 1 ]; then C_B='\033[1m'; C_G='\033[32m'; C_Y='\033[33m'; C_R='\033[31m'; C_0='\033[0m'
else C_B=''; C_G=''; C_Y=''; C_R=''; C_0=''; fi
localize() {
  local cn="$1" en="${2:-$1}"
  if [ "$OMC_LANG" = en ]; then printf '%s' "$en"; else printf '%s' "$cn"; fi
}
log()  { printf '%b\n' "$(localize "$1" "${2:-$1}")"; }
sep()  { printf '%b\n' "${C_B}── $(localize "$1" "${2:-$1}") ─────────────────────────────────────────${C_0}"; }
warn() { printf '%b\n' "${C_Y}[warn] $(localize "$1" "${2:-$1}")${C_0}" >&2; }
die()  {
  local cn="$1" en="$1" code=1
  if [[ "${2:-}" =~ ^[0-9]+$ ]]; then code="$2"; else en="${2:-$1}"; code="${3:-1}"; fi
  printf '%b\n' "${C_R}[FATAL] $(localize "$cn" "$en")${C_0}" >&2
  exit "$code"
}

# 整数除法向下取整的 awk 助手（避免依赖 bc）
mul_pct() { awk -v a="$1" -v p="$2" 'BEGIN{printf "%d", a*p/100}'; }   # a * p%
to_gib()  { awk -v m="$1" 'BEGIN{printf "%.1f", m/1024}'; }            # MiB→GiB 显示

# ---------------------------------------------------------------------------
# 1. 探测主机硬件 + 当前负荷（需求 ①②）
# ---------------------------------------------------------------------------
sep "1/4 探测主机配置与当前负荷" "1/4 Probe host configuration and current load"
OS="${OMC_PROBE_OS:-$(uname -s)}"
HOST_CPU=0; MEM_TOTAL_MIB=0; MEM_AVAIL_MIB=0; LOAD1=0; LOAD5=0; LOAD15=0; DISK_FREE_GIB=0

if [ -n "${OMC_PROBE_CPU:-}" ] &&
   [ -n "${OMC_PROBE_MEM_TOTAL_MIB:-}" ] &&
   [ -n "${OMC_PROBE_MEM_AVAIL_MIB:-}" ] &&
   [ -n "${OMC_PROBE_LOAD15:-}" ]; then
  HOST_CPU="$OMC_PROBE_CPU"
  MEM_TOTAL_MIB="$OMC_PROBE_MEM_TOTAL_MIB"
  MEM_AVAIL_MIB="$OMC_PROBE_MEM_AVAIL_MIB"
  LOAD15="$OMC_PROBE_LOAD15"
  DISK_FREE_GIB="${OMC_PROBE_DISK_FREE_GIB:-0}"
elif [ "$OS" = "Linux" ]; then
  HOST_CPU="$(nproc)"
  MEM_TOTAL_MIB="$(awk '/^MemTotal:/ {printf "%d", $2/1024}' /proc/meminfo)"
  MEM_AVAIL_MIB="$(awk '/^MemAvailable:/ {printf "%d", $2/1024}' /proc/meminfo)"
  read -r LOAD1 LOAD5 LOAD15 _ < /proc/loadavg
  # Docker 数据根目录可能被 daemon.json 的 data-root 改到非默认路径（如 /home/docker-data），
  # 不能硬编码 /var/lib/docker，否则测到的是错误分区的可用空间。优先问 docker info，
  # 拿不到（docker 不可达）时才退回默认路径。
  DOCKER_ROOT_DIR="$(docker info --format '{{.DockerRootDir}}' 2>/dev/null || true)"
  [ -z "$DOCKER_ROOT_DIR" ] && DOCKER_ROOT_DIR="/var/lib/docker"
  DISK_FREE_GIB="$(df -BG --output=avail "$DOCKER_ROOT_DIR" 2>/dev/null | awk 'NR==2{gsub(/G/,"");print $1}')"
elif [ "$OS" = "Darwin" ]; then
  # macOS：仅供本机 dry-run 预览（生产是 Linux）
  HOST_CPU="$(sysctl -n hw.ncpu)"
  MEM_TOTAL_MIB="$(sysctl -n hw.memsize | awk '{printf "%d", $1/1024/1024}')"
  PAGE="$(sysctl -n hw.pagesize)"
  FREE_PAGES="$(vm_stat | awk -v p="$PAGE" '/Pages free/{gsub(/\./,"",$3); f=$3} /Pages inactive/{gsub(/\./,"",$3); i=$3} END{print f+i}')"
  MEM_AVAIL_MIB="$(awk -v fp="$FREE_PAGES" -v p="$PAGE" 'BEGIN{printf "%d", fp*p/1024/1024}')"
  read -r LOAD1 LOAD5 LOAD15 <<<"$(sysctl -n vm.loadavg | awk '{print $2, $3, $4}')"
  DISK_FREE_GIB="$(df -g / 2>/dev/null | awk 'NR==2{print $4}')"
  warn "当前在 macOS 上运行——仅供 dry-run 预览；生产请在目标 Linux 服务器执行。" "Running on macOS; this is for dry-run preview only. Run on the target Linux server for production."
else
  die "不支持的操作系统：${OS}（需 Linux 生产 / macOS 预览）" "Unsupported operating system: ${OS} (Linux is required for production; macOS is supported for preview)." 1
fi
[ "${DISK_FREE_GIB:-0}" -gt 0 ] 2>/dev/null || DISK_FREE_GIB=0

# what-if 覆盖：在构建机为「目标主机」预规划，或测试用。设了就覆盖探测值。
[ -n "${OMC_PROBE_CPU:-}" ]           && HOST_CPU="$OMC_PROBE_CPU"
[ -n "${OMC_PROBE_MEM_TOTAL_MIB:-}" ] && MEM_TOTAL_MIB="$OMC_PROBE_MEM_TOTAL_MIB"
[ -n "${OMC_PROBE_MEM_AVAIL_MIB:-}" ] && MEM_AVAIL_MIB="$OMC_PROBE_MEM_AVAIL_MIB"
[ -n "${OMC_PROBE_LOAD15:-}" ]        && LOAD15="$OMC_PROBE_LOAD15"

log "  操作系统      : $OS" "  Operating system: $OS"
log "  CPU 核数      : ${C_B}${HOST_CPU}${C_0}" "  CPU cores      : ${C_B}${HOST_CPU}${C_0}"
log "  内存总量      : ${C_B}$(to_gib "$MEM_TOTAL_MIB") GiB${C_0} (${MEM_TOTAL_MIB} MiB)" "  Total memory   : ${C_B}$(to_gib "$MEM_TOTAL_MIB") GiB${C_0} (${MEM_TOTAL_MIB} MiB)"
log "  当前可用内存  : ${C_B}$(to_gib "$MEM_AVAIL_MIB") GiB${C_0} (MemAvailable，已反映其它进程当前占用)" "  Available mem. : ${C_B}$(to_gib "$MEM_AVAIL_MIB") GiB${C_0} (MemAvailable, reflects other process usage)"
log "  负载(1/5/15)  : ${LOAD1} / ${LOAD5} / ${LOAD15}" "  Load (1/5/15) : ${LOAD1} / ${LOAD5} / ${LOAD15}"
log "  Docker 盘可用 : ${DISK_FREE_GIB} GiB" "  Docker disk    : ${DISK_FREE_GIB} GiB available"

if [ -n "${OMC_PROBE_STORAGE_MOUNTS:-}" ]; then
  STORAGE_MOUNTS="$OMC_PROBE_STORAGE_MOUNTS"
elif [ "$OS" = "Linux" ]; then
  STORAGE_MOUNTS="$(
    df -l -B1 --output=avail,target \
      -x tmpfs -x devtmpfs -x overlay -x squashfs -x proc -x sysfs -x cgroup -x cgroup2 \
      2>/dev/null |
      awk 'NR > 1 && $1 ~ /^[0-9]+$/ {
        avail=$1
        $1=""
        sub(/^[[:space:]]+/, "")
        if ($0 ~ /^\//) print avail "|" $0
      }'
  )"
else
  STORAGE_MOUNTS="$(
    df -k / 2>/dev/null |
      awk 'NR == 2 { print ($4 * 1024) "|" $NF }'
  )"
fi
RECOMMENDED_STORAGE_MOUNT="$(printf '%s\n' "$STORAGE_MOUNTS" | storage_select_largest_mount)" ||
  die "未找到可用的本地持久文件系统；请检查磁盘挂载后重试。" "No local persistent filesystem was found; check the disk mounts and retry." 1
log "  推荐数据盘    : ${C_B}${RECOMMENDED_STORAGE_MOUNT}${C_0}（按可用空间最大选择）" "  Recommended disk: ${C_B}${RECOMMENDED_STORAGE_MOUNT}${C_0} (largest available filesystem)"

# 其它项目（非 omcgo）容器的「已声明但未用」预留 —— 共享主机要替它们留出余量
OTHER_RESERVE_MIB=0; OTHER_CPU=0
if [ "$ASSUME_DEDICATED" = 0 ] && command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  # 列出非 omcgo project 的运行中容器，累加 (mem_limit - mem_usage) 作为待预留余量。
  # mem_limit=0 表示无上限，则用当前用量近似。
  while IFS=$'\t' read -r name proj; do
    [ -z "$name" ] && continue
    case "$proj" in omcgo|"") continue ;; esac   # 跳过本项目与无 label 的
    lim="$(docker inspect -f '{{.HostConfig.Memory}}' "$name" 2>/dev/null || echo 0)"
    use_raw="$(docker stats --no-stream --format '{{.MemUsage}}' "$name" 2>/dev/null | awk '{print $1}')"
    use_mib="$(awk -v s="$use_raw" 'BEGIN{
      n=s+0; u=s; gsub(/[0-9.]/,"",u);
      if(u ~ /GiB/) printf "%d", n*1024; else if(u ~ /MiB/) printf "%d", n;
      else if(u ~ /KiB/) printf "%d", n/1024; else printf "%d", 0 }')"
    lim_mib="$(awk -v l="$lim" 'BEGIN{printf "%d", l/1024/1024}')"
    if [ "$lim_mib" -gt 0 ]; then
      headroom=$(( lim_mib - use_mib )); [ "$headroom" -lt 0 ] && headroom=0
      OTHER_RESERVE_MIB=$(( OTHER_RESERVE_MIB + headroom ))
    else
      OTHER_RESERVE_MIB=$(( OTHER_RESERVE_MIB + use_mib ))   # 无上限：按现用量近似预留
    fi
  done < <(docker ps --format '{{.Names}}\t{{.Label "com.docker.compose.project"}}' 2>/dev/null)
  [ "$OTHER_RESERVE_MIB" -gt 0 ] && \
    log "  其它项目预留  : ${C_Y}$(to_gib "$OTHER_RESERVE_MIB") GiB${C_0}（非 omcgo 容器待用余量，已从空闲预算扣除）" "  Other projects : ${C_Y}$(to_gib "$OTHER_RESERVE_MIB") GiB${C_0} reserved (unused capacity of non-omcgo containers deducted from the idle budget)"
fi

# ---------------------------------------------------------------------------
# 2. 计算「空闲预算」（需求 ②③，空闲优先口径）
# ---------------------------------------------------------------------------
sep "2/4 计算空闲资源预算" "2/4 Calculate idle resource budget"
# OS / dockerd / 内核保留：随主机规模缩放 = 总量 / 8，下限 4 GiB，上限 16 GiB。
# 例：32核/32GiB → 保留4GiB（业务28GiB）；64核/64GiB → 保留8GiB；≥128GiB 封顶16GiB。
OS_RESERVE_MIB="$(awk -v t="$MEM_TOTAL_MIB" 'BEGIN{r=t/8; if(r<4096)r=4096; if(r>16384)r=16384; printf "%d", r}')"
IDLE_MEM_MIB=$(( MEM_AVAIL_MIB - OS_RESERVE_MIB - OTHER_RESERVE_MIB ))
[ "$IDLE_MEM_MIB" -lt 0 ] && IDLE_MEM_MIB=0

# 主机 CPU 保留：同样随规模缩放 = 核数 / 8，下限 4 核，上限 12 核。
# 例：32核 → 保留4核（业务28核）；64核 → 保留8核；≥96核 封顶12核。
CPU_RESERVE="$(awk -v c="$HOST_CPU" 'BEGIN{r=c/8; if(r<4)r=4; if(r>12)r=12; printf "%d", (r==int(r))?r:int(r)+1}')"
# 空闲 CPU：核数 − 主机保留 − max(其它容器CPU, 取整后的 load15)
LOAD15_CEIL="$(awk -v l="$LOAD15" 'BEGIN{printf "%d", (l==int(l))?l:int(l)+1}')"
IDLE_CPU=$(( HOST_CPU - CPU_RESERVE - LOAD15_CEIL )); [ "$IDLE_CPU" -lt 1 ] && IDLE_CPU=1

log "  内存空闲预算  : ${C_G}${C_B}$(to_gib "$IDLE_MEM_MIB") GiB${C_0}  = MemAvailable $(to_gib "$MEM_AVAIL_MIB") − OS保留 $(to_gib "$OS_RESERVE_MIB") − 其它预留 $(to_gib "$OTHER_RESERVE_MIB")" "  Idle memory    : ${C_G}${C_B}$(to_gib "$IDLE_MEM_MIB") GiB${C_0} = MemAvailable $(to_gib "$MEM_AVAIL_MIB") - OS reserve $(to_gib "$OS_RESERVE_MIB") - other reserve $(to_gib "$OTHER_RESERVE_MIB")"
log "  CPU 空闲预算  : ${C_G}${C_B}${IDLE_CPU} 核${C_0}  = ${HOST_CPU} − 主机保留 ${CPU_RESERVE} − 负载占用 ${LOAD15_CEIL}（CPU 限额可突发超分，仅作下限参考）" "  Idle CPU      : ${C_G}${C_B}${IDLE_CPU} cores${C_0} = ${HOST_CPU} - host reserve ${CPU_RESERVE} - load ${LOAD15_CEIL} (CPU limits may burst; reference for floors only)"

# ---------------------------------------------------------------------------
# 3. 组件 floor/ceiling 表 + floor-first 分配（需求 ③）
# ---------------------------------------------------------------------------
# 单位 MiB。下面是 32 GiB 基线机型的参考画像，不是最终申请值；最终 floor/ceiling
# 会按本机实际业务内存预算动态缩放。数据来源：资源规划分析 + 对抗评审修正
#（floor-clamp / 全量记账 / PG 按连接数定容）。
#
#   组件          floor   ceil    surplus权重(%)   说明
#   app           1536    3072        10           非设备量驱动，最先让出预算；线上巡检发现 995MiB 配额下常驻内存已到 88%，floor/ceil 一并调大留余量
#   acs           4096    6144        18           TR-069 最热堆；1M 走横向多副本；PM 上传 body 需缓冲进内存，omc78 压测实测 1-2GiB 在高并发（max_inflight 调大后）下触发 cgroup OOMKilled 循环重启，floor 提到 4GiB（2026-07-21）
#   worker        1024    2048        25           PM/MR 解析最吃内存；1M 走横向
#   postgres      7168   16384        25           业务主库；须容 max_connections=300(池+exporter+余,与 main #131 对齐)
#   postgres-tsdb 4096   12288        22           时序库(#347)：PM COPY 入库 + KPI 聚合，写压力主要在此；独立实例，计入预算防双 PG 超分 OOM
#   redis-core    4096    6144         8           会话/任务/告警；与 PM 状态物理隔离，保留 1GiB AOF COW
#   redis-pm      8192   12288        15           12分钟关窗双小时重叠；保留 2GiB AOF COW
#   nats          1024    2048         5           JetStream backlog + client buffers，避免 512MiB cgroup 临界
#   minio         3072    4096         8           对象存储；压测发现按可见CPU配额自动估算的并发上限过于保守，且线上巡检 2.5GiB 配额下已到 88%，floor/ceil 一并调大留余量
#   web            512     512         0           静态+反代，固定
# 监控栈（固定块，不纵向伸缩，但计入预算）：~4736 MiB
COMP_NAMES=(app acs worker postgres postgres-tsdb redis-core redis-pm nats minio web)
BASE_COMP_FLOOR=(1536 4096 1024 7168 4096 4096 8192 1024 3072 512)
BASE_COMP_CEIL=(3072 6144 2048 16384 12288 6144 12288 2048 4096 512)
# 低配机仍须保证能启动一套有意义的 OMC；这些是按组件职责定义的最低比例
# 约束，不是固定的最终申请值。双 ACS 接力副本在后续总账中再计一次。
COMP_MIN=(512 1024 512 2048 1536 2048 4096 256 512 128)
COMP_WEIGHT=(10 18 25 25 22 8 15 5 8 0)

MON_FIXED_MIB=4736   # prometheus1024+loki512+tempo1024+otelcol512+grafana512+alertmgr512+exporters(128*3+256)
[ "$SKIP_MONITORING" = 1 ] && MON_FIXED_MIB=0

# 32 GiB 基线只用于计算比例；业务总预算来自当前主机的实际可用内存，且先扣掉
# 监控固定成本。这样同一套交付包在 16/32/64 GiB 主机上不会申请同样的内存。
BASE_BUSINESS_FLOOR_SUM=0
for f in "${BASE_COMP_FLOOR[@]}"; do BASE_BUSINESS_FLOOR_SUM=$(( BASE_BUSINESS_FLOOR_SUM + f )); done
BASE_BUSINESS_FLOOR_SUM=$(( BASE_BUSINESS_FLOOR_SUM + BASE_COMP_FLOOR[1] ))
MIN_BUSINESS_FLOOR_SUM=0
for m in "${COMP_MIN[@]}"; do MIN_BUSINESS_FLOOR_SUM=$(( MIN_BUSINESS_FLOOR_SUM + m )); done
MIN_BUSINESS_FLOOR_SUM=$(( MIN_BUSINESS_FLOOR_SUM + COMP_MIN[1] ))
BUSINESS_BUDGET_MIB=$(( IDLE_MEM_MIB - MON_FIXED_MIB ))
if [ "$BUSINESS_BUDGET_MIB" -lt "$MIN_BUSINESS_FLOOR_SUM" ]; then
  MIN_AVAILABLE_WITH_MONITORING_MIB=$(( OS_RESERVE_MIB + MON_FIXED_MIB + MIN_BUSINESS_FLOOR_SUM ))
  die "扣除监控固定成本后业务内存预算仅 $(to_gib "$BUSINESS_BUDGET_MIB") GiB，低于最低可运行预算 $(to_gib "$MIN_BUSINESS_FLOOR_SUM") GiB；全栈至少需要 MemAvailable 约 $(to_gib "$MIN_AVAILABLE_WITH_MONITORING_MIB") GiB。\n       --floor-tolerance-pct 只对后续组件下限缺口生效，不能绕过此最低运行预算。\n       需要保留监控：请释放内存后重试，或扩容主机。\n       可以不部署监控：bash $SCRIPT_DIR/plan-resources.sh --skip-monitoring，然后使用 install.sh --skip-monitoring。" "Business memory budget after the fixed monitoring cost is only $(to_gib "$BUSINESS_BUDGET_MIB") GiB, below the minimum runnable budget of $(to_gib "$MIN_BUSINESS_FLOOR_SUM") GiB; the full stack needs about $(to_gib "$MIN_AVAILABLE_WITH_MONITORING_MIB") GiB of MemAvailable.\n       --floor-tolerance-pct applies only to later component-floor gaps and cannot bypass this minimum runtime budget.\n       To keep monitoring: release memory and retry, or resize the host.\n       To skip monitoring: bash $SCRIPT_DIR/plan-resources.sh --skip-monitoring, then use install.sh --skip-monitoring." 1
fi

# floor 先占业务预算的 70%，剩余 30% 按组件权重向上分配；低配机若按比例
# 得到的 floor 低于组件最低可运行值，则优先抬到该最低值，并继续做总账校验。
FLOOR_BUDGET_MIB=$(( BUSINESS_BUDGET_MIB * 70 / 100 ))
[ "$FLOOR_BUDGET_MIB" -lt "$MIN_BUSINESS_FLOOR_SUM" ] && FLOOR_BUDGET_MIB="$MIN_BUSINESS_FLOOR_SUM"
COMP_FLOOR=(); COMP_CEIL=()
for i in "${!COMP_NAMES[@]}"; do
  floor=$(( BASE_COMP_FLOOR[$i] * FLOOR_BUDGET_MIB / BASE_BUSINESS_FLOOR_SUM ))
  ceil=$(( BASE_COMP_CEIL[$i] * BUSINESS_BUDGET_MIB / BASE_BUSINESS_FLOOR_SUM ))
  [ "$floor" -lt "${COMP_MIN[$i]}" ] && floor="${COMP_MIN[$i]}"
  [ "$ceil" -lt "$floor" ] && ceil="$floor"
  COMP_FLOOR[$i]="$floor"
  COMP_CEIL[$i]="$ceil"
done

# 下限之和（最低门槛）。ACS 无损发布常驻 primary + candidate 两个同规格实例；
# COMP_NAMES 中的 acs 负责计算单副本规格，这里把第二副本完整计入预算。
FLOOR_SUM=0; for f in "${COMP_FLOOR[@]}"; do FLOOR_SUM=$(( FLOOR_SUM + f )); done
FLOOR_SUM=$(( FLOOR_SUM + COMP_FLOOR[1] + MON_FIXED_MIB ))

sep "3/4 floor-first 资源分配" "3/4 Floor-first resource allocation"
if [ "$SKIP_MONITORING" = 1 ]; then
  FLOOR_MONITORING_NOTE="（不含监控）"
  FLOOR_MONITORING_NOTE_EN="(monitoring excluded)"
else
  FLOOR_MONITORING_NOTE="（含监控 $(to_gib "$MON_FIXED_MIB") GiB）"
  FLOOR_MONITORING_NOTE_EN="($(to_gib "$MON_FIXED_MIB") GiB monitoring included)"
fi
log "  组件下限之和  : $(to_gib "$FLOOR_SUM") GiB${FLOOR_MONITORING_NOTE}" "  Component floors: $(to_gib "$FLOOR_SUM") GiB ${FLOOR_MONITORING_NOTE_EN}"
log "  业务内存预算  : $(to_gib "$BUSINESS_BUDGET_MIB") GiB（实际空闲预算扣除监控固定成本，按本机预算动态缩放）" "  Business budget : $(to_gib "$BUSINESS_BUDGET_MIB") GiB (idle budget minus fixed monitoring cost, scaled to this host)"

# 最低配置门禁（需求 ②：缺口超过容忍度才 fail + 给建议；容忍度内降级为 WARN 按下限分配）
FLOOR_MIN_REQUIRED=$(mul_pct "$FLOOR_SUM" $((100 - FLOOR_TOLERANCE_PCT)))
if [ "$IDLE_MEM_MIB" -lt "$FLOOR_MIN_REQUIRED" ]; then
  REC_FULL=$(( (FLOOR_SUM + OS_RESERVE_MIB) / 1024 + 2 ))
    die "空闲内存 $(to_gib "$IDLE_MEM_MIB") GiB < 组件下限之和 $(to_gib "$FLOOR_SUM") GiB 的 $((100 - FLOOR_TOLERANCE_PCT))%（容忍度 ${FLOOR_TOLERANCE_PCT}% 后仍不够）—— 无法安全部署。
      建议最低配置：整机 ≥ ${REC_FULL} GiB 内存$([ "$SKIP_MONITORING" = 0 ] && echo '（或加 --skip-monitoring 降到约 20 GiB）')；
      或释放本机其它项目占用后重试，或用 --assume-dedicated（确认本机 OMC 独占时），或调大 --floor-tolerance-pct（当前 ${FLOOR_TOLERANCE_PCT}%）。" "Idle memory $(to_gib "$IDLE_MEM_MIB") GiB is below $((100 - FLOOR_TOLERANCE_PCT))% of the component floors $(to_gib "$FLOOR_SUM") GiB (still insufficient after ${FLOOR_TOLERANCE_PCT}% tolerance); deployment is unsafe.
      Recommended minimum: host memory >= ${REC_FULL} GiB$([ "$SKIP_MONITORING" = 0 ] && echo ' (or about 20 GiB with --skip-monitoring)');
      Release other projects and retry, use --assume-dedicated only when OMC owns the host, or increase --floor-tolerance-pct (currently ${FLOOR_TOLERANCE_PCT}%)." 1
elif [ "$IDLE_MEM_MIB" -lt "$FLOOR_SUM" ]; then
  SHORT_MIB=$(( FLOOR_SUM - IDLE_MEM_MIB ))
  SHORT_PCT=$(( SHORT_MIB * 100 / FLOOR_SUM ))
  warn "空闲内存 $(to_gib "$IDLE_MEM_MIB") GiB < 组件下限之和 $(to_gib "$FLOOR_SUM") GiB，缺口 ${SHORT_PCT}%（≤ 容忍度 ${FLOOR_TOLERANCE_PCT}%）：按下限分配，不再向上伸缩。各组件同时打满 limit 时仍有 OOM 风险，请尽快释放内存或调低 --floor-tolerance-pct 复核。" "Idle memory $(to_gib "$IDLE_MEM_MIB") GiB is below the component floors $(to_gib "$FLOOR_SUM") GiB by ${SHORT_PCT}% (within the ${FLOOR_TOLERANCE_PCT}% tolerance); allocating floors only, with no upward scaling. OOM remains possible if all components hit their limits; release memory or lower --floor-tolerance-pct and recheck."
fi

# 剩余空闲按权重分配（floor..ceil 之间向上伸缩）
SURPLUS=$(( IDLE_MEM_MIB - FLOOR_SUM )); [ "$SURPLUS" -lt 0 ] && SURPLUS=0
WEIGHT_SUM=0; for w in "${COMP_WEIGHT[@]}"; do WEIGHT_SUM=$(( WEIGHT_SUM + w )); done
# 发布期间 primary/candidate 两个 ACS 会同时驻留。COMP_NAMES 只保留一份 ACS
# 配置用于生成环境变量，因此余量分母必须显式计入第二个 ACS 的权重；
# 否则 ACS 获得的一份余量会在总分配中被重复计算，导致规划超过主机预算。
WEIGHT_SUM=$(( WEIGHT_SUM + COMP_WEIGHT[1] ))

declare -a COMP_MEM
for i in "${!COMP_NAMES[@]}"; do
  floor=${COMP_FLOOR[$i]}; ceil=${COMP_CEIL[$i]}; w=${COMP_WEIGHT[$i]}
  add=0
  if [ "$w" -gt 0 ] && [ "$WEIGHT_SUM" -gt 0 ]; then
    add=$(( SURPLUS * w / WEIGHT_SUM ))
  fi
  mem=$(( floor + add )); [ "$mem" -gt "$ceil" ] && mem=$ceil
  COMP_MEM[$i]=$mem
done
# acs 固定为 COMP_NAMES 第 2 项；提前取值，供双副本总预算核算。
ACS_MEM=${COMP_MEM[1]}

# 档位判定（按空闲内存）：仅用于 CPU 取档与提示
if [ -n "$TIER_OVERRIDE" ]; then TIER="$TIER_OVERRIDE"
elif [ "$IDLE_MEM_MIB" -ge 49152 ]; then TIER=large     # ≥48 GiB 空闲
elif [ "$IDLE_MEM_MIB" -ge 24576 ]; then TIER=medium    # ≥24 GiB 空闲
else TIER=small; fi

# CPU 限额（突发可超分；按档位给值）。medium/large 按 10000 基站压测验证的
# 热点比例分配：主库=总核数1/3、时序库=1/2、worker=1/4，均向下取整且最低2核。
# 三者限额之和可超过物理核数；cgroup CPU limit 是各自上限而非预留，实际由调度器共享。
CPU_DB_THIRD=$(( HOST_CPU / 3 )); [ "$CPU_DB_THIRD" -lt 2 ] && CPU_DB_THIRD=2
CPU_TSDB_HALF=$(( HOST_CPU / 2 )); [ "$CPU_TSDB_HALF" -lt 2 ] && CPU_TSDB_HALF=2
CPU_WORKER_QUARTER=$(( HOST_CPU / 4 )); [ "$CPU_WORKER_QUARTER" -lt 2 ] && CPU_WORKER_QUARTER=2
case "$TIER" in
  small)  CPU_app=1;   CPU_acs=2; CPU_worker=2; CPU_pg=2; CPU_tsdb=2; CPU_redis_core=1; CPU_redis_pm=1; CPU_nats=1; CPU_minio=1; CPU_web=1 ;;
  # 32核/32GiB medium 实测 10/16/8：worker/TSDB 吞吐被充分利用，PM 无重投/OOM。
  medium) CPU_app="1.5"; CPU_acs=5; CPU_worker=$CPU_WORKER_QUARTER; CPU_pg=$CPU_DB_THIRD; CPU_tsdb=$CPU_TSDB_HALF; CPU_redis_core=2; CPU_redis_pm=2; CPU_nats=1; CPU_minio=4; CPU_web=1 ;;
  large)  CPU_app=2;   CPU_acs=8; CPU_worker=$CPU_WORKER_QUARTER; CPU_pg=$CPU_DB_THIRD; CPU_tsdb=$CPU_TSDB_HALF; CPU_redis_core=2; CPU_redis_pm=2; CPU_nats=2; CPU_minio=6; CPU_web=1 ;;
esac
CPU_LIST=("$CPU_app" "$CPU_acs" "$CPU_worker" "$CPU_pg" "$CPU_tsdb" "$CPU_redis_core" "$CPU_redis_pm" "$CPU_nats" "$CPU_minio" "$CPU_web")
idx() { local n="$1"; for i in "${!COMP_NAMES[@]}"; do [ "${COMP_NAMES[$i]}" = "$n" ] && { echo "$i"; return; }; done; }

# 32GiB 及以上档位为 MinIO 固定保留 6GiB cap。线上海量小对象场景中，进程 RSS
# 仅约 0.45GiB，但可回收文件系统 slab 会把 4GiB cgroup working set 推至上限，
# 造成持续 memory.max 回收并放大磁盘 I/O。这里仅从其它组件 floor 以上的 cap 余量
# 重分配，既不突破整机预算，也不削减任何组件的最低可运行内存。
if [ "$TIER" != "small" ]; then
  MINIO_IDX=$(idx minio)
  MINIO_TARGET_MIB=6144
  if [ "${COMP_MEM[$MINIO_IDX]}" -lt "$MINIO_TARGET_MIB" ]; then
    MINIO_NEED=$(( MINIO_TARGET_MIB - COMP_MEM[$MINIO_IDX] ))
    for reclaim_name in app nats web redis-core worker postgres postgres-tsdb redis-pm acs; do
      [ "$MINIO_NEED" -gt 0 ] || break
      reclaim_idx=$(idx "$reclaim_name")
      reclaimable=$(( COMP_MEM[$reclaim_idx] - COMP_FLOOR[$reclaim_idx] ))
      [ "$reclaimable" -gt 0 ] || continue
      take="$reclaimable"; [ "$take" -gt "$MINIO_NEED" ] && take="$MINIO_NEED"
      COMP_MEM[$reclaim_idx]=$(( COMP_MEM[$reclaim_idx] - take ))
      MINIO_NEED=$(( MINIO_NEED - take ))
    done
    [ "$MINIO_NEED" -eq 0 ] || die "${TIER} 档位无法在不削减组件 floor 的前提下为 MinIO 保留 6GiB；请扩容或降低其它组件基线" "The ${TIER} tier cannot reserve 6 GiB for MinIO without reducing component floors; resize the host or lower other component baselines." 1
    COMP_MEM[$MINIO_IDX]="$MINIO_TARGET_MIB"
  fi
fi

# 校验：Σ内存限额 ≤ 空闲预算（全量记账，含监控）。SURPLUS=0 时 ALLOC_SUM==FLOOR_SUM，
# 若此时仍 > IDLE_MEM_MIB 属于上面已经 warn 过的「容忍度内下限缺口」，是预期行为，不重复 die；
# 只有「明明分了 SURPLUS>0 却还超预算」才是真正的分配逻辑 bug。
ALLOC_SUM=0; for m in "${COMP_MEM[@]}"; do ALLOC_SUM=$(( ALLOC_SUM + m )); done
ALLOC_SUM=$(( ALLOC_SUM + ACS_MEM + MON_FIXED_MIB ))
if [ "$ALLOC_SUM" -gt "$MEM_TOTAL_MIB" ]; then
  die "资源计划总内存 $(to_gib "$ALLOC_SUM") GiB 超过主机物理内存 $(to_gib "$MEM_TOTAL_MIB") GiB；即使提高 --floor-tolerance-pct 也禁止生成可能导致 OOM 的 resources.env。请释放内存、跳过监控或扩容主机。" "The planned memory $(to_gib "$ALLOC_SUM") GiB exceeds physical host memory $(to_gib "$MEM_TOTAL_MIB") GiB; even a higher --floor-tolerance-pct cannot allow an OOM-prone resources.env. Release memory, skip monitoring, or resize the host." 1
fi
if [ "$ALLOC_SUM" -gt "$IDLE_MEM_MIB" ] && [ "$SURPLUS" -gt 0 ]; then
  die "内部错误：分配后 Σ限额 $(to_gib "$ALLOC_SUM") GiB > 空闲预算 $(to_gib "$IDLE_MEM_MIB") GiB。请反馈此 bug。" "Internal error: allocated limits $(to_gib "$ALLOC_SUM") GiB exceed the idle budget $(to_gib "$IDLE_MEM_MIB") GiB. Please report this bug." 1
fi

# ---- 联动派生（同一预算 → 限额 + 进程内上限，结构上锁死一致）----
gomemlimit() { mul_pct "$1" 90; }                  # GOMEMLIMIT = 0.90 × 内存限额（软限，需配合准入控制）

APP_MEM=${COMP_MEM[$(idx app)]};     ACS_MEM=${COMP_MEM[$(idx acs)]}
WORKER_MEM=${COMP_MEM[$(idx worker)]}; PG_MEM=${COMP_MEM[$(idx postgres)]}
TSDB_MEM=${COMP_MEM[$(idx postgres-tsdb)]}
REDIS_CORE_MEM=${COMP_MEM[$(idx redis-core)]}; REDIS_PM_MEM=${COMP_MEM[$(idx redis-pm)]}
NATS_MEM=${COMP_MEM[$(idx nats)]}
NATS_MAX_MEMORY_STORE=$(( NATS_MEM * 1024 * 1024 / 4 ))
MINIO_MEM=${COMP_MEM[$(idx minio)]};   WEB_MEM=${COMP_MEM[$(idx web)]}

APP_GOMEM=$(gomemlimit "$APP_MEM"); ACS_GOMEM=$(gomemlimit "$ACS_MEM"); WORKER_GOMEM=$(gomemlimit "$WORKER_MEM")
APP_GOMAXPROCS="$(awk -v c="$CPU_app" 'BEGIN{printf "%d", (c<1)?1:int(c)}')"
ACS_GOMAXPROCS="$(awk -v c="$CPU_acs" 'BEGIN{printf "%d", int(c)}')"
WORKER_GOMAXPROCS="$(awk -v c="$CPU_worker" 'BEGIN{printf "%d", int(c)}')"

# Postgres：max_connections 须覆盖 Go 端连接池总和 180（app 60+40 / acs 30 / worker 25+25）
# + exporter + 手动 psql + 余量；与 main #131 的发布默认对齐取 300。
PG_MAXCONN=300
PG_SHARED_BUFFERS=$(mul_pct "$PG_MEM" 25)          # 25%（留 OS page cache 给 Timescale 解压）
PG_EFFECTIVE_CACHE=$(mul_pct "$PG_MEM" 70)         # 70% 规划器提示
PG_WORK_MEM=$([ "$PG_MEM" -ge 8192 ] && echo 16 || { [ "$PG_MEM" -ge 6144 ] && echo 8 || echo 4; })
PG_MAINT_WORK_MEM=$(awk -v m="$PG_MEM" 'BEGIN{v=m*0.05; if(v>2048)v=2048; if(v<256)v=256; printf "%d", v}')
PG_MAX_WAL=$([ "$PG_MEM" -ge 8192 ] && echo 8GB || echo 4GB)
# 自检：shared_buffers + maint + 连接基线 + work_mem 是否舒适放进 PG 限额
PG_SAT=$(( PG_SHARED_BUFFERS + PG_MAINT_WORK_MEM + PG_MAXCONN * (10 + PG_WORK_MEM) + 512 ))
[ "$PG_SAT" -gt "$(mul_pct "$PG_MEM" 92)" ] && \
  warn "Postgres 饱和估算 $(to_gib "$PG_SAT") GiB 接近限额 $(to_gib "$PG_MEM") GiB；建议增大内存或上 pgbouncer 收敛连接。" "Postgres saturation estimate $(to_gib "$PG_SAT") GiB is close to its $(to_gib "$PG_MEM") GiB limit; increase memory or use pgbouncer to consolidate connections."

# Postgres-tsdb（#347 时序库）：与主库同源派生（shared_buffers 25% / effective_cache 70% /
# work_mem / maint / max_wal）。写压力主要在此；连接池较小（app40+worker25=65 < 主库 115），
# max_connections 取 300 与主库/compose 缺省对齐（充足余量、口径一致）。
TSDB_MAXCONN=300
TSDB_SHARED_BUFFERS=$(mul_pct "$TSDB_MEM" 25)
TSDB_EFFECTIVE_CACHE=$(mul_pct "$TSDB_MEM" 70)
TSDB_WORK_MEM=$([ "$TSDB_MEM" -ge 8192 ] && echo 16 || { [ "$TSDB_MEM" -ge 6144 ] && echo 8 || echo 4; })
TSDB_MAINT_WORK_MEM=$(awk -v m="$TSDB_MEM" 'BEGIN{v=m*0.05; if(v>2048)v=2048; if(v<256)v=256; printf "%d", v}')
TSDB_MAX_WAL=$([ "$TSDB_MEM" -ge 8192 ] && echo 8GB || echo 4GB)
TSDB_SAT=$(( TSDB_SHARED_BUFFERS + TSDB_MAINT_WORK_MEM + TSDB_MAXCONN * (10 + TSDB_WORK_MEM) + 512 ))
[ "$TSDB_SAT" -gt "$(mul_pct "$TSDB_MEM" 92)" ] && \
  warn "Postgres-tsdb 饱和估算 $(to_gib "$TSDB_SAT") GiB 接近限额 $(to_gib "$TSDB_MEM") GiB；建议增大内存或下调 TSDB_MAX_CONNECTIONS（时序库连接池仅 ~65）。" "Postgres-tsdb saturation estimate $(to_gib "$TSDB_SAT") GiB is close to its $(to_gib "$TSDB_MEM") GiB limit; increase memory or lower TSDB_MAX_CONNECTIONS (the time-series connection pool is only about 65)."

# Redis：核心状态与 PM 聚合窗口使用两个物理实例，避免 PM 关窗/重算的内存和
# AOF 写放大挤压 ACS 会话、设备任务与告警。核心实例保留 1GiB、PM 实例保留
# 2GiB 给 AOF rewrite 的 fork COW。
REDIS_CORE_MAXMEM=$(( REDIS_CORE_MEM - 1024 ))
REDIS_PM_MAXMEM=$(( REDIS_PM_MEM - 2048 ))
REDIS_POLICY=noeviction

# ---------------------------------------------------------------------------
# 4. 输出规划表 + 写 resources.env（需求 ③④）
# ---------------------------------------------------------------------------
sep "4/4 资源规划结果（档位：${TIER}）" "4/4 Resource plan result (tier: ${TIER})"
if [ "$OMC_LANG" = en ]; then
  printf '%b\n' "${C_B}  Component   Memory limit    GOMEMLIMIT/key links${C_0}"
  CANDIDATE_NOTE="(zero-downtime release standby)"
  TSDB_NOTE="(time-series database #347)"
  COW_NOTE="(COW headroom=%sMiB)"
  MONITORING_NOTE="(fixed block, not scaled)"
else
  printf '%b\n' "${C_B}  组件        内存限额        GOMEMLIMIT/关键联动${C_0}"
  CANDIDATE_NOTE="（无损发布接力副本）"
  TSDB_NOTE="（时序库 #347）"
  COW_NOTE="（COW 余量=%sMiB）"
  MONITORING_NOTE="（固定块，不纵向伸缩）"
fi
printf '  %-10s  %8s MiB\n' "app"    "$APP_MEM" ; printf '              ↳ GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' "$APP_GOMEM" "$APP_GOMAXPROCS"
printf '  %-10s  %8s MiB\n' "acs"    "$ACS_MEM" ; printf '              ↳ GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' "$ACS_GOMEM" "$ACS_GOMAXPROCS"
printf '  %-13s  %8s MiB\n' "acs-candidate" "$ACS_MEM" ; printf '              ↳ GOMEMLIMIT=%sMiB GOMAXPROCS=%s%s\n' "$ACS_GOMEM" "$ACS_GOMAXPROCS" "$CANDIDATE_NOTE"
printf '  %-10s  %8s MiB\n' "worker" "$WORKER_MEM" ; printf '              ↳ GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' "$WORKER_GOMEM" "$WORKER_GOMAXPROCS"
printf '  %-10s  %8s MiB\n' "postgres" "$PG_MEM" ; printf '              ↳ shared_buffers=%sMB effective_cache=%sMB max_connections=%s work_mem=%sMB\n' "$PG_SHARED_BUFFERS" "$PG_EFFECTIVE_CACHE" "$PG_MAXCONN" "$PG_WORK_MEM"
printf '  %-13s  %8s MiB\n' "postgres-tsdb" "$TSDB_MEM" ; printf '              ↳ shared_buffers=%sMB effective_cache=%sMB max_connections=%s work_mem=%sMB%s\n' "$TSDB_SHARED_BUFFERS" "$TSDB_EFFECTIVE_CACHE" "$TSDB_MAXCONN" "$TSDB_WORK_MEM" "$TSDB_NOTE"
printf '  %-13s  %8s MiB\n' "redis-core" "$REDIS_CORE_MEM" ; printf '              ↳ maxmemory=%sMB policy=%s' "$REDIS_CORE_MAXMEM" "$REDIS_POLICY"; printf "$COW_NOTE\n" "$(( REDIS_CORE_MEM - REDIS_CORE_MAXMEM ))"
printf '  %-13s  %8s MiB\n' "redis-pm" "$REDIS_PM_MEM" ; printf '              ↳ maxmemory=%sMB policy=%s' "$REDIS_PM_MAXMEM" "$REDIS_POLICY"; printf "$COW_NOTE\n" "$(( REDIS_PM_MEM - REDIS_PM_MAXMEM ))"
printf '  %-10s  %8s MiB\n' "nats"   "$NATS_MEM"
printf '  %-10s  %8s MiB\n' "minio"  "$MINIO_MEM"
printf '  %-10s  %8s MiB\n' "web"    "$WEB_MEM"
[ "$SKIP_MONITORING" = 0 ] && printf '  %-10s  %8s MiB%s\n' "monitoring" "$MON_FIXED_MIB" "$MONITORING_NOTE"
log "  ───────────────────────────────────"
log "  Σ内存限额    : ${C_B}$(to_gib "$ALLOC_SUM") GiB${C_0} / 空闲预算 $(to_gib "$IDLE_MEM_MIB") GiB（余 $(to_gib $(( IDLE_MEM_MIB - ALLOC_SUM ))) GiB）" "  Total limits  : ${C_B}$(to_gib "$ALLOC_SUM") GiB${C_0} / idle budget $(to_gib "$IDLE_MEM_MIB") GiB ($(to_gib $(( IDLE_MEM_MIB - ALLOC_SUM ))) GiB remaining)"
[ "$TIER" = large ] && warn "large 档：1M 规模须多机拓扑（acs/worker ×6-8 + pgbouncer + Redis 拆分），本脚本仅规划单机切片。" "large tier: 1M scale requires a multi-host topology (acs/worker x6-8 + pgbouncer + split Redis); this script plans one host slice only."

sep "数据路径建议" "Data path recommendations"
log "  默认根目录    : ${RECOMMENDED_STORAGE_MOUNT%/}/omc-data" "  Default root   : ${RECOMMENDED_STORAGE_MOUNT%/}/omc-data"
log "  ${C_Y}请人工检查并按物理 SSD/NVMe 修改 ${STORAGE_ENV_FILE} 中六个 *_DATA_PATH。${C_0}" "  ${C_Y}Check and adjust the six *_DATA_PATH values in ${STORAGE_ENV_FILE} for the physical SSD/NVMe layout.${C_0}"
log "  ${C_Y}本脚本只生成启动路径，不会迁移已有 Docker volume 或目录中的数据。${C_0}" "  ${C_Y}This script only prepares startup paths; it does not migrate existing Docker volumes or directory data.${C_0}"

if [ "$DRY_RUN" = 1 ]; then
  log "\n${C_Y}--dry-run：未写入文件。${C_0}去掉 --dry-run 即生成 $OUT_FILE" "\n${C_Y}--dry-run: no file was written.${C_0} Remove --dry-run to generate $OUT_FILE"
  exit 0
fi

storage_apply_recommended_paths "$STORAGE_ENV_FILE" "$RECOMMENDED_STORAGE_MOUNT" ||
  die "写入数据路径失败：$STORAGE_ENV_FILE" "Failed to write data paths: $STORAGE_ENV_FILE" 1
log "  已补齐空值    : ${STORAGE_ENV_FILE}（已有人工配置保持不变）" "  Filled empty values: ${STORAGE_ENV_FILE} (existing operator values were preserved)"
for storage_key in $STORAGE_PATH_KEYS; do
  log "    $storage_key=$(storage_env_get "$STORAGE_ENV_FILE" "$storage_key")"
done

# 在目标文件同目录生成候选，校验成功后用 rename 原子替换。这样校验失败、磁盘写入失败
# 或中断都不会截断/删除上一份 last-good resources.env。
OUT_TMP="$(mktemp "${OUT_FILE}.tmp.XXXXXX")" ||
  die "无法在 resources.env 同目录创建临时文件：${OUT_FILE}.tmp.XXXXXX" "Unable to create a temporary file beside resources.env: ${OUT_FILE}.tmp.XXXXXX" 1
cleanup_resource_output_tmp() {
  [ -z "${OUT_TMP:-}" ] || rm -f "$OUT_TMP"
}
trap cleanup_resource_output_tmp EXIT

# 写 resources.env 候选（带注释，可手改）
{
  echo "# =============================================================================="
  echo "# $(localize 'resources.env —— OMC 容器资源限额（由 plan-resources.sh 生成，可手动调整）' 'resources.env - OMC container resource limits (generated by plan-resources.sh; editable)')"
  echo "# $(localize "生成档位: $TIER   主机: ${HOST_CPU}核/$(to_gib "$MEM_TOTAL_MIB")GiB   空闲预算: $(to_gib "$IDLE_MEM_MIB")GiB" "Tier: $TIER   Host: ${HOST_CPU} cores/$(to_gib "$MEM_TOTAL_MIB") GiB   Idle budget: $(to_gib "$IDLE_MEM_MIB") GiB")"
  echo "# $(localize 'compose 以 \${VAR:-默认} 读取本文件；改完任选其一即生效（需求 ⑥）：' 'Compose reads this file with \${VAR:-default}; apply changes with either:')"
  echo "#   bash svc.sh restart       # $(localize '推荐：按 depends_on 有序重建，经健康门控' 'recommended: ordered rebuild with depends_on health gates')"
  echo "#   # $(localize '或手工：' 'or manually:') docker compose -p omcgo --env-file .env --env-file resources.env -f ... up -d"
  echo "# $(localize '约束（改值时务必遵守，否则重演 OOM 事故）：' 'Constraints (must be followed when editing values to avoid OOM incidents):')"
  echo "#   · GOMEMLIMIT $(localize '必须 < 对应 *_MEM（软限，建议 0.90×）；ACS 还须配合准入控制+SOAP体上限' 'must be < the corresponding *_MEM (soft limit, 0.90x recommended); ACS also requires admission control and a SOAP body limit')"
  echo "#   · REDIS_CORE_MEM $(localize '必须 ≥ REDIS_CORE_MAXMEMORY + 1GiB；REDIS_PM_MEM 须保留 2GiB AOF COW 余量' 'must be >= REDIS_CORE_MAXMEMORY + 1GiB; REDIS_PM_MEM must retain 2 GiB AOF COW headroom')"
  echo "#   · PG_MAX_CONNECTIONS $(localize '必须 ≥ Go 端连接池总和(当前 180)；增大须同步增大 POSTGRES_MEM' 'must be >= the Go connection-pool total (currently 180); increase POSTGRES_MEM when increasing it')"
  echo "#   · $(localize '时序库(#347)：TSDB_* 同理；TSDB_MEM 变则 TSDB_SHARED_BUFFERS 联动；两 PG 内存合计须 ≤ 空闲预算' 'time-series database (#347): the same applies to TSDB_*; TSDB_SHARED_BUFFERS follows TSDB_MEM; both PG memory budgets must fit within the idle budget')"
  echo "# =============================================================================="
  echo ""
  echo "# ── $(localize '业务（Go）' 'Business (Go)') ── *_MEM $(localize '是 cgroup 硬限；*_GOMEMLIMIT 是 Go 堆软限（0.90×）' 'is the cgroup hard limit; *_GOMEMLIMIT is the Go heap soft limit (0.90x)')"
  echo "APP_CPUS=$CPU_app";       echo "APP_MEM=${APP_MEM}m";       echo "APP_GOMEMLIMIT=${APP_GOMEM}MiB";       echo "APP_GOMAXPROCS=$APP_GOMAXPROCS"
  echo "ACS_CPUS=$CPU_acs";       echo "ACS_MEM=${ACS_MEM}m";       echo "ACS_GOMEMLIMIT=${ACS_GOMEM}MiB";       echo "ACS_GOMAXPROCS=$ACS_GOMAXPROCS"
  echo "WORKER_CPUS=$CPU_worker"; echo "WORKER_MEM=${WORKER_MEM}m"; echo "WORKER_GOMEMLIMIT=${WORKER_GOMEM}MiB"; echo "WORKER_GOMAXPROCS=$WORKER_GOMAXPROCS"
  echo ""
  echo "# ── Postgres $(localize '主库 ── 业务数据（时序已分离到 tsdb）；限额与 -c 调优参数同源派生' 'primary ── business data (time-series data is separated to tsdb); limits and -c tuning are derived from the same budget')"
  echo "POSTGRES_CPUS=$CPU_pg";   echo "POSTGRES_MEM=${PG_MEM}m"
  echo "PG_SHARED_BUFFERS=${PG_SHARED_BUFFERS}MB"; echo "PG_EFFECTIVE_CACHE_SIZE=${PG_EFFECTIVE_CACHE}MB"
  echo "PG_MAX_CONNECTIONS=$PG_MAXCONN"; echo "PG_WORK_MEM=${PG_WORK_MEM}MB"
  echo "PG_MAINTENANCE_WORK_MEM=${PG_MAINT_WORK_MEM}MB"; echo "PG_MAX_WAL_SIZE=$PG_MAX_WAL"
  echo ""
  echo "# ── Postgres postgres-tsdb (#347) ── $(localize '独立 TimescaleDB 实例，PM COPY/KPI 聚合写主要在此' 'independent TimescaleDB instance; PM COPY/KPI aggregation writes primarily land here')"
  echo "TSDB_CPUS=$CPU_tsdb";     echo "TSDB_MEM=${TSDB_MEM}m"
  echo "TSDB_SHARED_BUFFERS=${TSDB_SHARED_BUFFERS}MB"; echo "TSDB_EFFECTIVE_CACHE_SIZE=${TSDB_EFFECTIVE_CACHE}MB"
  echo "TSDB_MAX_CONNECTIONS=$TSDB_MAXCONN"; echo "TSDB_WORK_MEM=${TSDB_WORK_MEM}MB"
  echo "TSDB_MAINTENANCE_WORK_MEM=${TSDB_MAINT_WORK_MEM}MB"; echo "TSDB_MAX_WAL_SIZE=$TSDB_MAX_WAL"
  echo ""
  echo "# ── Redis Core / PM ── $(localize '两个物理实例，PM 关窗与重算不再争抢核心业务资源' 'two physical instances; PM window closing and recomputation no longer compete with core business resources')"
  echo "REDIS_CORE_CPUS=$CPU_redis_core"; echo "REDIS_CORE_MEM=${REDIS_CORE_MEM}m"
  echo "REDIS_CORE_MAXMEMORY=${REDIS_CORE_MAXMEM}mb"; echo "REDIS_CORE_MAXMEMORY_POLICY=$REDIS_POLICY"
  echo "REDIS_PM_CPUS=$CPU_redis_pm"; echo "REDIS_PM_MEM=${REDIS_PM_MEM}m"
  echo "REDIS_PM_MAXMEMORY=${REDIS_PM_MAXMEM}mb"; echo "REDIS_PM_MAXMEMORY_POLICY=$REDIS_POLICY"
  echo ""
  echo "# ── NATS / MinIO / Web ──"
  echo "NATS_CPUS=$CPU_nats";     echo "NATS_MEM=${NATS_MEM}m"; echo "NATS_MAX_MEMORY_STORE=$NATS_MAX_MEMORY_STORE"
  echo "MINIO_CPUS=$CPU_minio";   echo "MINIO_MEM=${MINIO_MEM}m"
  echo "WEB_CPUS=$CPU_web";       echo "WEB_MEM=${WEB_MEM}m"
  echo ""
  echo "# ── $(localize '规划元信息（仅记录，compose 不读取）' 'Plan metadata (record only; not read by Compose)') ──"
  echo "OMC_RESOURCE_SCHEMA_VERSION=3"
  echo "OMC_RESOURCE_PLAN_HOST_CPU=$HOST_CPU"
  echo "OMC_RESOURCE_PLAN_HOST_MEM_MIB=$MEM_TOTAL_MIB"
  echo "OMC_PLAN_TIER=$TIER"
  echo "OMC_PLAN_SKIP_MONITORING=$SKIP_MONITORING"
} > "$OUT_TMP"

if ! resource_env_validate "$OUT_TMP"; then
  die "生成的 resources.env 候选未通过完整资源契约验证；已保留上一份有效文件" "Generated resources.env candidate failed the complete resource contract validation; the previous valid file was preserved." 1
fi
chmod 0644 "$OUT_TMP" ||
  die "无法设置 resources.env 候选权限；已保留上一份有效文件" "Unable to set permissions on the resources.env candidate; the previous valid file was preserved." 1
mv -f "$OUT_TMP" "$OUT_FILE" ||
  die "无法原子替换 resources.env；已保留上一份有效文件" "Unable to atomically replace resources.env; the previous valid file was preserved." 1
OUT_TMP=""

log "\n${C_G}✓ 已写入：$OUT_FILE${C_0}" "\n${C_G}✓ Written: $OUT_FILE${C_0}"
log "  下一步：检视/调整该文件 → 运行 install.sh（将以 --env-file resources.env 动态部署）。" "  Next: review/adjust this file, then run install.sh (it will deploy dynamically with --env-file resources.env)."
