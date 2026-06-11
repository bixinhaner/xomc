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
#   ./plan-resources.sh -o /path/resources.env   # 指定输出路径
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
OUT_FILE="$SCRIPT_DIR/resources.env"
DRY_RUN=0
SKIP_MONITORING=0
ASSUME_DEDICATED=0
TIER_OVERRIDE=""

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run)          DRY_RUN=1 ;;
    --skip-monitoring)  SKIP_MONITORING=1 ;;
    --assume-dedicated) ASSUME_DEDICATED=1 ;;
    --tier)             TIER_OVERRIDE="${2:-}"; shift ;;
    --tier=*)           TIER_OVERRIDE="${1#*=}" ;;
    -o|--output)        OUT_FILE="${2:?-o 需要路径}"; shift ;;
    -o=*|--output=*)    OUT_FILE="${1#*=}" ;;
    -h|--help)
      sed -n '2,55p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "未知参数：$1（--help 查看用法）" >&2; exit 2 ;;
  esac
  shift
done

case "$TIER_OVERRIDE" in ""|small|medium|large) ;; *)
  echo "--tier 仅支持 small|medium|large，收到：$TIER_OVERRIDE" >&2; exit 2 ;;
esac

# 颜色与日志
if [ -t 1 ]; then C_B='\033[1m'; C_G='\033[32m'; C_Y='\033[33m'; C_R='\033[31m'; C_0='\033[0m'
else C_B=''; C_G=''; C_Y=''; C_R=''; C_0=''; fi
log()  { printf '%b\n' "$*"; }
sep()  { printf '%b\n' "${C_B}── $* ─────────────────────────────────────────${C_0}"; }
warn() { printf '%b\n' "${C_Y}[warn] $*${C_0}" >&2; }
die()  { printf '%b\n' "${C_R}[FATAL] $*${C_0}" >&2; exit "${2:-1}"; }

# 整数除法向下取整的 awk 助手（避免依赖 bc）
mul_pct() { awk -v a="$1" -v p="$2" 'BEGIN{printf "%d", a*p/100}'; }   # a * p%
to_gib()  { awk -v m="$1" 'BEGIN{printf "%.1f", m/1024}'; }            # MiB→GiB 显示

# ---------------------------------------------------------------------------
# 1. 探测主机硬件 + 当前负荷（需求 ①②）
# ---------------------------------------------------------------------------
sep "1/4 探测主机配置与当前负荷"
OS="$(uname -s)"
HOST_CPU=0; MEM_TOTAL_MIB=0; MEM_AVAIL_MIB=0; LOAD1=0; LOAD5=0; LOAD15=0; DISK_FREE_GIB=0

if [ "$OS" = "Linux" ]; then
  HOST_CPU="$(nproc)"
  MEM_TOTAL_MIB="$(awk '/^MemTotal:/ {printf "%d", $2/1024}' /proc/meminfo)"
  MEM_AVAIL_MIB="$(awk '/^MemAvailable:/ {printf "%d", $2/1024}' /proc/meminfo)"
  read -r LOAD1 LOAD5 LOAD15 _ < /proc/loadavg
  DISK_FREE_GIB="$(df -BG --output=avail /var/lib/docker 2>/dev/null | awk 'NR==2{gsub(/G/,"");print $1}')"
elif [ "$OS" = "Darwin" ]; then
  # macOS：仅供本机 dry-run 预览（生产是 Linux）
  HOST_CPU="$(sysctl -n hw.ncpu)"
  MEM_TOTAL_MIB="$(sysctl -n hw.memsize | awk '{printf "%d", $1/1024/1024}')"
  PAGE="$(sysctl -n hw.pagesize)"
  FREE_PAGES="$(vm_stat | awk -v p="$PAGE" '/Pages free/{gsub(/\./,"",$3); f=$3} /Pages inactive/{gsub(/\./,"",$3); i=$3} END{print f+i}')"
  MEM_AVAIL_MIB="$(awk -v fp="$FREE_PAGES" -v p="$PAGE" 'BEGIN{printf "%d", fp*p/1024/1024}')"
  read -r LOAD1 LOAD5 LOAD15 <<<"$(sysctl -n vm.loadavg | awk '{print $2, $3, $4}')"
  DISK_FREE_GIB="$(df -g / 2>/dev/null | awk 'NR==2{print $4}')"
  warn "当前在 macOS 上运行——仅供 dry-run 预览；生产请在目标 Linux 服务器执行。"
else
  die "不支持的操作系统：${OS}（需 Linux 生产 / macOS 预览）" 1
fi
[ "${DISK_FREE_GIB:-0}" -gt 0 ] 2>/dev/null || DISK_FREE_GIB=0

# what-if 覆盖：在构建机为「目标主机」预规划，或测试用。设了就覆盖探测值。
[ -n "${OMC_PROBE_CPU:-}" ]            && HOST_CPU="$OMC_PROBE_CPU"
[ -n "${OMC_PROBE_MEM_TOTAL_MIB:-}" ] && MEM_TOTAL_MIB="$OMC_PROBE_MEM_TOTAL_MIB"
[ -n "${OMC_PROBE_MEM_AVAIL_MIB:-}" ] && MEM_AVAIL_MIB="$OMC_PROBE_MEM_AVAIL_MIB"
[ -n "${OMC_PROBE_LOAD15:-}" ]        && LOAD15="$OMC_PROBE_LOAD15"

log "  操作系统      : $OS"
log "  CPU 核数      : ${C_B}${HOST_CPU}${C_0}"
log "  内存总量      : ${C_B}$(to_gib "$MEM_TOTAL_MIB") GiB${C_0} (${MEM_TOTAL_MIB} MiB)"
log "  当前可用内存  : ${C_B}$(to_gib "$MEM_AVAIL_MIB") GiB${C_0} (MemAvailable，已反映其它进程当前占用)"
log "  负载(1/5/15)  : ${LOAD1} / ${LOAD5} / ${LOAD15}"
log "  Docker 盘可用 : ${DISK_FREE_GIB} GiB"

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
    log "  其它项目预留  : ${C_Y}$(to_gib "$OTHER_RESERVE_MIB") GiB${C_0}（非 omcgo 容器待用余量，已从空闲预算扣除）"
fi

# ---------------------------------------------------------------------------
# 2. 计算「空闲预算」（需求 ②③，空闲优先口径）
# ---------------------------------------------------------------------------
sep "2/4 计算空闲资源预算"
# OS / dockerd / 内核保留：max(2 GiB, 总量的 15%)
OS_RESERVE_MIB="$(awk -v t="$MEM_TOTAL_MIB" 'BEGIN{r=t*0.15; if(r<2048)r=2048; printf "%d", r}')"
IDLE_MEM_MIB=$(( MEM_AVAIL_MIB - OS_RESERVE_MIB - OTHER_RESERVE_MIB ))
[ "$IDLE_MEM_MIB" -lt 0 ] && IDLE_MEM_MIB=0

# 空闲 CPU：核数 − 主机保留(1) − max(其它容器CPU, 取整后的 load15)
LOAD15_CEIL="$(awk -v l="$LOAD15" 'BEGIN{printf "%d", (l==int(l))?l:int(l)+1}')"
IDLE_CPU=$(( HOST_CPU - 1 - LOAD15_CEIL )); [ "$IDLE_CPU" -lt 1 ] && IDLE_CPU=1

log "  内存空闲预算  : ${C_G}${C_B}$(to_gib "$IDLE_MEM_MIB") GiB${C_0}  = MemAvailable $(to_gib "$MEM_AVAIL_MIB") − OS保留 $(to_gib "$OS_RESERVE_MIB") − 其它预留 $(to_gib "$OTHER_RESERVE_MIB")"
log "  CPU 空闲预算  : ${C_G}${C_B}${IDLE_CPU} 核${C_0}  = ${HOST_CPU} − 主机保留 1 − 负载占用 ${LOAD15_CEIL}（CPU 限额可突发超分，仅作下限参考）"

# ---------------------------------------------------------------------------
# 3. 组件 floor/ceiling 表 + floor-first 分配（需求 ③）
# ---------------------------------------------------------------------------
# 单位 MiB。floor = 100k 基线下限（绝不低于）；ceil = 单机纵向上限（再大走横向扩展）。
# 数据来源：资源规划分析 + 对抗评审修正（floor-clamp / 全量记账 / PG按连接数定容）。
#
#   组件        floor   ceil    surplus权重(%)   说明
#   app          768    1536        10           非设备量驱动，最先让出预算
#   acs         1024    2048        15           TR-069 最热堆；1M 走横向多副本
#   worker      1024    2048        25           PM/MR 解析最吃内存；1M 走横向
#   postgres    7168   16384        25           须容 max_connections=300(180池+exporter+余,与 main #131 对齐)
#   redis       3072    8192        15           appendonly，限额需≥1.5×maxmemory
#   nats         512    2048         5
#   minio       1024    2048         5
#   web          512     512         0           静态+反代，固定
# 监控栈（固定块，不纵向伸缩，但计入预算）：~4224 MiB
COMP_NAMES=(app acs worker postgres redis nats minio web)
COMP_FLOOR=(768 1024 1024 7168 3072 512 1024 512)
COMP_CEIL=(1536 2048 2048 16384 8192 2048 2048 512)
COMP_WEIGHT=(10 15 25 25 15 5 5 0)

MON_FIXED_MIB=4224   # prometheus1024+loki512+tempo512+otelcol512+grafana512+alertmgr512+exporters(128*3+256)
[ "$SKIP_MONITORING" = 1 ] && MON_FIXED_MIB=0

# 下限之和（最低门槛）
FLOOR_SUM=0; for f in "${COMP_FLOOR[@]}"; do FLOOR_SUM=$(( FLOOR_SUM + f )); done
FLOOR_SUM=$(( FLOOR_SUM + MON_FIXED_MIB ))

sep "3/4 floor-first 资源分配"
log "  组件下限之和  : $(to_gib "$FLOOR_SUM") GiB$([ "$SKIP_MONITORING" = 1 ] && echo '（不含监控）' || echo '（含监控 '"$(to_gib "$MON_FIXED_MIB")"' GiB）')"

# 最低配置门禁（需求 ②：不够就 fail + 给建议）
if [ "$IDLE_MEM_MIB" -lt "$FLOOR_SUM" ]; then
  REC_FULL=$(( (FLOOR_SUM + OS_RESERVE_MIB) / 1024 + 2 ))
  die "空闲内存 $(to_gib "$IDLE_MEM_MIB") GiB < 组件下限之和 $(to_gib "$FLOOR_SUM") GiB —— 无法安全部署。
       建议最低配置：整机 ≥ ${REC_FULL} GiB 内存$([ "$SKIP_MONITORING" = 0 ] && echo '（或加 --skip-monitoring 降到约 16 GiB）')；
       或释放本机其它项目占用后重试，或用 --assume-dedicated（确认本机 OMC 独占时）。" 1
fi

# 剩余空闲按权重分配（floor..ceil 之间向上伸缩）
SURPLUS=$(( IDLE_MEM_MIB - FLOOR_SUM )); [ "$SURPLUS" -lt 0 ] && SURPLUS=0
WEIGHT_SUM=0; for w in "${COMP_WEIGHT[@]}"; do WEIGHT_SUM=$(( WEIGHT_SUM + w )); done

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

# 档位判定（按空闲内存）：仅用于 CPU 取档与提示
if [ -n "$TIER_OVERRIDE" ]; then TIER="$TIER_OVERRIDE"
elif [ "$IDLE_MEM_MIB" -ge 49152 ]; then TIER=large     # ≥48 GiB 空闲
elif [ "$IDLE_MEM_MIB" -ge 24576 ]; then TIER=medium    # ≥24 GiB 空闲
else TIER=small; fi

# CPU 限额（突发可超分；按档位给值）
case "$TIER" in
  small)  CPU_app=1;   CPU_acs=2; CPU_worker=2; CPU_pg=2; CPU_redis=1; CPU_nats=1; CPU_minio=1; CPU_web=1 ;;
  medium) CPU_app="1.5"; CPU_acs=3; CPU_worker=3; CPU_pg=4; CPU_redis=2; CPU_nats=1; CPU_minio=1; CPU_web=1 ;;
  large)  CPU_app=2;   CPU_acs=4; CPU_worker=4; CPU_pg=6; CPU_redis=2; CPU_nats=2; CPU_minio=1; CPU_web=1 ;;
esac
CPU_LIST=("$CPU_app" "$CPU_acs" "$CPU_worker" "$CPU_pg" "$CPU_redis" "$CPU_nats" "$CPU_minio" "$CPU_web")

# 校验：Σ内存限额 ≤ 空闲预算（全量记账，含监控）
ALLOC_SUM=0; for m in "${COMP_MEM[@]}"; do ALLOC_SUM=$(( ALLOC_SUM + m )); done
ALLOC_SUM=$(( ALLOC_SUM + MON_FIXED_MIB ))
if [ "$ALLOC_SUM" -gt "$IDLE_MEM_MIB" ]; then
  die "内部错误：分配后 Σ限额 $(to_gib "$ALLOC_SUM") GiB > 空闲预算 $(to_gib "$IDLE_MEM_MIB") GiB。请反馈此 bug。" 1
fi

# ---- 联动派生（同一预算 → 限额 + 进程内上限，结构上锁死一致）----
gomemlimit() { mul_pct "$1" 90; }                  # GOMEMLIMIT = 0.90 × 内存限额（软限，需配合准入控制）
idx() { local n="$1"; for i in "${!COMP_NAMES[@]}"; do [ "${COMP_NAMES[$i]}" = "$n" ] && { echo "$i"; return; }; done; }

APP_MEM=${COMP_MEM[$(idx app)]};     ACS_MEM=${COMP_MEM[$(idx acs)]}
WORKER_MEM=${COMP_MEM[$(idx worker)]}; PG_MEM=${COMP_MEM[$(idx postgres)]}
REDIS_MEM=${COMP_MEM[$(idx redis)]};   NATS_MEM=${COMP_MEM[$(idx nats)]}
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
  warn "Postgres 饱和估算 $(to_gib "$PG_SAT") GiB 接近限额 $(to_gib "$PG_MEM") GiB；建议增大内存或上 pgbouncer 收敛连接。"

# Redis：maxmemory = 0.66 × 限额，且保证 限额 − maxmemory ≥ 1 GiB（AOF rewrite 的 COW 余量）
REDIS_MAXMEM=$(mul_pct "$REDIS_MEM" 66)
[ $(( REDIS_MEM - REDIS_MAXMEM )) -lt 1024 ] && REDIS_MAXMEM=$(( REDIS_MEM - 1024 ))
REDIS_POLICY=volatile-lru                           # 仅淘汰带 TTL 的键，保护无 TTL 的队列

# ---------------------------------------------------------------------------
# 4. 输出规划表 + 写 resources.env（需求 ③④）
# ---------------------------------------------------------------------------
sep "4/4 资源规划结果（档位：${TIER}）"
printf '%b\n' "${C_B}  组件        内存限额        GOMEMLIMIT/关键联动${C_0}"
printf '  %-10s  %8s MiB\n' "app"    "$APP_MEM" ; printf '              ↳ GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' "$APP_GOMEM" "$APP_GOMAXPROCS"
printf '  %-10s  %8s MiB\n' "acs"    "$ACS_MEM" ; printf '              ↳ GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' "$ACS_GOMEM" "$ACS_GOMAXPROCS"
printf '  %-10s  %8s MiB\n' "worker" "$WORKER_MEM" ; printf '              ↳ GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' "$WORKER_GOMEM" "$WORKER_GOMAXPROCS"
printf '  %-10s  %8s MiB\n' "postgres" "$PG_MEM" ; printf '              ↳ shared_buffers=%sMB effective_cache=%sMB max_connections=%s work_mem=%sMB\n' "$PG_SHARED_BUFFERS" "$PG_EFFECTIVE_CACHE" "$PG_MAXCONN" "$PG_WORK_MEM"
printf '  %-10s  %8s MiB\n' "redis"  "$REDIS_MEM" ; printf '              ↳ maxmemory=%sMB policy=%s（限额−maxmemory=%sMiB COW 余量）\n' "$REDIS_MAXMEM" "$REDIS_POLICY" "$(( REDIS_MEM - REDIS_MAXMEM ))"
printf '  %-10s  %8s MiB\n' "nats"   "$NATS_MEM"
printf '  %-10s  %8s MiB\n' "minio"  "$MINIO_MEM"
printf '  %-10s  %8s MiB\n' "web"    "$WEB_MEM"
[ "$SKIP_MONITORING" = 0 ] && printf '  %-10s  %8s MiB（固定块，不纵向伸缩）\n' "monitoring" "$MON_FIXED_MIB"
log "  ───────────────────────────────────"
log "  Σ内存限额    : ${C_B}$(to_gib "$ALLOC_SUM") GiB${C_0} / 空闲预算 $(to_gib "$IDLE_MEM_MIB") GiB（余 $(to_gib $(( IDLE_MEM_MIB - ALLOC_SUM ))) GiB）"
[ "$TIER" = large ] && warn "large 档：1M 规模须多机拓扑（acs/worker ×6-8 + pgbouncer + Redis 拆分），本脚本仅规划单机切片。"

if [ "$DRY_RUN" = 1 ]; then
  log "\n${C_Y}--dry-run：未写入文件。${C_0}去掉 --dry-run 即生成 $OUT_FILE"
  exit 0
fi

# 写 resources.env（带注释，可手改）
{
  echo "# =============================================================================="
  echo "# resources.env —— OMC 容器资源限额（由 plan-resources.sh 生成，可手动调整）"
  echo "# 生成档位: $TIER   主机: ${HOST_CPU}核/$(to_gib "$MEM_TOTAL_MIB")GiB   空闲预算: $(to_gib "$IDLE_MEM_MIB")GiB"
  echo "# compose 以 \${VAR:-默认} 读取本文件；改完任选其一即生效（需求 ⑥）："
  echo "#   bash svc.sh restart       # 推荐：按 depends_on 有序重建，经健康门控"
  echo "#   # 或手工：docker compose -p omcgo --env-file .env --env-file resources.env -f ... up -d"
  echo "# 约束（改值时务必遵守，否则重演 OOM 事故）："
  echo "#   · GOMEMLIMIT 必须 < 对应 *_MEM（软限，建议 0.90×）；ACS 还须配合准入控制+SOAP体上限"
  echo "#   · REDIS_MEM 必须 ≥ REDIS_MAXMEMORY + 1GiB（AOF rewrite 的 fork COW 余量）"
  echo "#   · PG_MAX_CONNECTIONS 必须 ≥ Go 端连接池总和(当前 180)；增大须同步增大 POSTGRES_MEM"
  echo "# =============================================================================="
  echo ""
  echo "# ── 业务（Go）── *_MEM 是 cgroup 硬限；*_GOMEMLIMIT 是 Go 堆软限（0.90×）"
  echo "APP_CPUS=$CPU_app";       echo "APP_MEM=${APP_MEM}m";       echo "APP_GOMEMLIMIT=${APP_GOMEM}MiB";       echo "APP_GOMAXPROCS=$APP_GOMAXPROCS"
  echo "ACS_CPUS=$CPU_acs";       echo "ACS_MEM=${ACS_MEM}m";       echo "ACS_GOMEMLIMIT=${ACS_GOMEM}MiB";       echo "ACS_GOMAXPROCS=$ACS_GOMAXPROCS"
  echo "WORKER_CPUS=$CPU_worker"; echo "WORKER_MEM=${WORKER_MEM}m"; echo "WORKER_GOMEMLIMIT=${WORKER_GOMEM}MiB"; echo "WORKER_GOMAXPROCS=$WORKER_GOMAXPROCS"
  echo ""
  echo "# ── Postgres / TimescaleDB ── 限额与 -c 调优参数同源派生"
  echo "POSTGRES_CPUS=$CPU_pg";   echo "POSTGRES_MEM=${PG_MEM}m"
  echo "PG_SHARED_BUFFERS=${PG_SHARED_BUFFERS}MB"; echo "PG_EFFECTIVE_CACHE_SIZE=${PG_EFFECTIVE_CACHE}MB"
  echo "PG_MAX_CONNECTIONS=$PG_MAXCONN"; echo "PG_WORK_MEM=${PG_WORK_MEM}MB"
  echo "PG_MAINTENANCE_WORK_MEM=${PG_MAINT_WORK_MEM}MB"; echo "PG_MAX_WAL_SIZE=$PG_MAX_WAL"
  echo ""
  echo "# ── Redis ── 限额 ≥ maxmemory + 1GiB"
  echo "REDIS_CPUS=$CPU_redis";   echo "REDIS_MEM=${REDIS_MEM}m"
  echo "REDIS_MAXMEMORY=${REDIS_MAXMEM}mb"; echo "REDIS_MAXMEMORY_POLICY=$REDIS_POLICY"
  echo ""
  echo "# ── NATS / MinIO / Web ──"
  echo "NATS_CPUS=$CPU_nats";     echo "NATS_MEM=${NATS_MEM}m"
  echo "MINIO_CPUS=$CPU_minio";   echo "MINIO_MEM=${MINIO_MEM}m"
  echo "WEB_CPUS=$CPU_web";       echo "WEB_MEM=${WEB_MEM}m"
  echo ""
  echo "# ── 规划元信息（仅记录，compose 不读取）──"
  echo "OMC_PLAN_TIER=$TIER"
  echo "OMC_PLAN_SKIP_MONITORING=$SKIP_MONITORING"
} > "$OUT_FILE"

log "\n${C_G}✓ 已写入：$OUT_FILE${C_0}"
log "  下一步：检视/调整该文件 → 运行 install.sh（将以 --env-file resources.env 动态部署）。"
