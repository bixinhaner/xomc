#!/usr/bin/env bash
# =============================================================================
# plan-resources.sh —— OMC【本地 dev / 单机 compose 栈】资源动态规划脚本
# =============================================================================
#
# 用途：自动按「本机 Docker 引擎可用资源」算出 docker-compose.yml 各服务的
#   CPU / 内存 / 缓存（Redis maxmemory）/ PostgreSQL 内部参数（shared_buffers 等）
#   / Go 运行时上限（GOMEMLIMIT / GOMAXPROCS），写入同目录 resources.env，
#   省去每次换机器 / 调 Docker Desktop VM 大小都要手改十几个限额的麻烦。
#
#   生产交付包另有一份 deployments/release/bundle/deploy/plan-resources.sh（多 compose
#   文件 + 离线安装 + 监控栈），本脚本是它在「本地单文件 dev 栈」上的对应物，关键差异：
#     · 探测口径用 `docker info`（= 容器真正的天花板：Docker Desktop 下是 VM 上限，
#       不是物理内存；Linux 下 = 宿主总量），而非物理 RAM —— Mac 上物理 24g 但 VM 仅
#       ~11.67g，按物理算会严重超分。
#     · 账上有【两个 PostgreSQL 实例】：主库 postgres + 时序库 postgres-tsdb（KPI/时序
#       物理分离后），二者都吃 shared_buffers，故内存切分与 OOM 自检按「双 PG」做。
#     · 单一自适应（不分业务 profile）：各组件 floor/ceiling/权重已内含相对重要度
#       （PG / worker 重、web 轻），按空闲资源在 floor..ceiling 间统一向上伸缩。
#
# 设计原则（与 docs/operations/OMC内存分配与容量规划-20260611.md 一致）：
#   1. 容器 limits.memory 是「安全上限/熔断」，不是预留。除 PG 外的服务实测都只用
#      几十 MB，Σcap 允许 > VM —— 真正会吃满的只有两个 PG，故 PG 的 cap + 内部参数
#      + 双 PG 合计实占是唯一关键自检（本脚本据此 clamp shared_buffers，绝不让两库
#      合计实占 + 其它栈 + OS 顶破 VM）。
#   2. floor-first：每个组件先拿 floor 下限（绝不低于，否则重演 Redis/PG OOM 事故），
#      只有「空闲余量」才按权重在 floor..ceiling 间向上伸缩。
#   3. 联动派生：limits 一变，GOMEMLIMIT / PG shared_buffers / Redis maxmemory 由
#      同一预算同步算出，结构上杜绝「限额与进程内上限不一致」。
#
# 产出：deployments/docker/resources.env（纯文本含注释，可手改）。日常用法：
#   bash deployments/docker/dc.sh up -d           # wrapper 自动带 --env-file，首次自动规划
#   # 或手工：docker compose -f deployments/docker/docker-compose.yml \
#   #           --env-file deployments/docker/resources.env up -d
#   网络变量没有默认回退：必须先由 dc.sh 按 DOCKER_BIP 规划，再启动 Compose。
#
# 用法：
#   ./plan-resources.sh                  # 探测 + 计算 + 写 resources.env
#   ./plan-resources.sh --dry-run        # 只打印规划，不写文件
#   ./plan-resources.sh --with-monitoring  # 把 prometheus/grafana/loki/... 也计入预算
#   ./plan-resources.sh --assume-dedicated # 视 Docker 引擎为 OMC 独占，不扣其它项目（如 boss 栈）
#   ./plan-resources.sh --self-project omc # 视为「本 OMC 栈」而跳过的 compose 项目名（默认 omc omcgo）
#   ./plan-resources.sh -o /path/resources.env  # 指定输出路径
#
# what-if（在别的机器上预规划目标机，或测试）：设了即覆盖探测值
#   OMC_PROBE_CPU=8 OMC_PROBE_MEM_TOTAL_MIB=8192 ./plan-resources.sh --dry-run
#
# 退出码：0 成功；1 主机内存不足以安全跑双 PG 全栈（给建议）；2 参数错误。
# =============================================================================
set -euo pipefail

# ---------------------------------------------------------------------------
# 0. 参数
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUT_FILE="$SCRIPT_DIR/resources.env"
DRY_RUN=0
WITH_MONITORING=0
ASSUME_DEDICATED=0
MAXIMIZE=0
DEVICES=""
RETENTION_DAYS=60
DISK_GIB_OVERRIDE=""
SELF_PROJECTS="omc omcgo"

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run)          DRY_RUN=1 ;;
    --with-monitoring)  WITH_MONITORING=1 ;;
    --assume-dedicated) ASSUME_DEDICATED=1 ;;
    # --maximize：独占型生产服务器（如 40C/80T、64/128G）最大化吃硬件 —— CPU 限额设满核
    # （谁抢到是谁的），内存按 PG 最佳实践（shared_buffers≈25% 物理内存）+ 大档派生，
    # 其余留作 OS page cache（TimescaleDB 解压/读全靠它）。详见 RESOURCE-PLANNING.md。
    --maximize)         MAXIMIZE=1 ;;
    # 保留期自动测算：给定本机承载设备数，按磁盘容量算 15min 原始数据可保留天数（上限 --retention-days，
    # 默认 60），写 retention.sql（UPDATE sys_configs pm.retention）。不给 --devices 则只报「本盘 60 天能撑几台」。
    --devices)          DEVICES="${2:?--devices 需要设备数}"; shift ;;
    --devices=*)        DEVICES="${1#*=}" ;;
    --retention-days)   RETENTION_DAYS="${2:?--retention-days 需要天数}"; shift ;;
    --retention-days=*) RETENTION_DAYS="${1#*=}" ;;
    --disk-gib)         DISK_GIB_OVERRIDE="${2:?--disk-gib 需要 GiB}"; shift ;;  # 覆盖磁盘探测（Mac/what-if）
    --disk-gib=*)       DISK_GIB_OVERRIDE="${1#*=}" ;;
    --self-project)     SELF_PROJECTS="${2:?--self-project 需要项目名}"; shift ;;
    --self-project=*)   SELF_PROJECTS="${1#*=}" ;;
    -o|--output)        OUT_FILE="${2:?-o 需要路径}"; shift ;;
    -o=*|--output=*)    OUT_FILE="${1#*=}" ;;
    -h|--help)          sed -n '2,70p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "未知参数：$1（--help 查看用法）" >&2; exit 2 ;;
  esac
  shift
done

if [ -t 1 ]; then C_B='\033[1m'; C_G='\033[32m'; C_Y='\033[33m'; C_R='\033[31m'; C_0='\033[0m'
else C_B=''; C_G=''; C_Y=''; C_R=''; C_0=''; fi
log()  { printf '%b\n' "$*"; }
sep()  { printf '%b\n' "${C_B}── $* ─────────────────────────────────────────${C_0}"; }
warn() { printf '%b\n' "${C_Y}[warn] $*${C_0}" >&2; }
die()  { printf '%b\n' "${C_R}[FATAL] $*${C_0}" >&2; exit "${2:-1}"; }

pct()    { awk -v a="$1" -v p="$2" 'BEGIN{printf "%d", a*p/100}'; }     # a * p%
to_gib() { awk -v m="$1" 'BEGIN{printf "%.1f", m/1024}'; }             # MiB→GiB 显示
floori() { awk -v x="$1" 'BEGIN{printf "%d", int(x)}'; }

# ---------------------------------------------------------------------------
# 1. 探测「容器可用天花板」—— 优先 docker info（= VM 上限 / 宿主总量）
# ---------------------------------------------------------------------------
sep "1/4 探测 Docker 引擎可用资源"
VM_CPU=0; VM_MEM_MIB=0; DISK_FREE_GIB=0
DETECT_SRC="docker info"

if [ -n "${OMC_PROBE_CPU:-}" ] && [ -n "${OMC_PROBE_MEM_TOTAL_MIB:-}" ]; then
  # what-if / CI 探测值是完整输入，不能再依赖 docker、uname 或 sysctl。
  VM_CPU="$OMC_PROBE_CPU"
  VM_MEM_MIB="$OMC_PROBE_MEM_TOTAL_MIB"
  DETECT_SRC="OMC_PROBE_* 覆盖"
else
  OS="$(uname -s)"
  if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    VM_CPU="$(docker info --format '{{.NCPU}}' 2>/dev/null || echo 0)"
    _mb="$(docker info --format '{{.MemTotal}}' 2>/dev/null || echo 0)"
    VM_MEM_MIB="$(awk -v b="$_mb" 'BEGIN{printf "%d", b/1024/1024}')"
    # docker root 盘可用空间（Docker Desktop 下为 VM 盘；Linux 下为 /var/lib/docker 所在盘）
    DOCKER_ROOT="$(docker info --format '{{.DockerRootDir}}' 2>/dev/null || echo /var/lib/docker)"
    DISK_FREE_GIB="$(df -g "$DOCKER_ROOT" 2>/dev/null | awk 'NR==2{print $4}' || echo 0)"
    [ -z "$DISK_FREE_GIB" ] && DISK_FREE_GIB="$(df -BG "$DOCKER_ROOT" 2>/dev/null | awk 'NR==2{gsub(/G/,"");print $4}' || echo 0)"
  else
    # docker 不可用：回退到宿主探测（仅供预览；实际容器仍受 docker 引擎限制）
    DETECT_SRC="宿主探测（docker 不可用，仅预览）"
    warn "docker 引擎不可达——回退宿主探测；实际请在 docker 可用时重跑以读 VM 真实上限。"
    if [ "$OS" = "Linux" ]; then
      VM_CPU="$(nproc)"
      VM_MEM_MIB="$(awk '/^MemTotal:/ {printf "%d", $2/1024}' /proc/meminfo)"
      DISK_FREE_GIB="$(df -BG /var/lib/docker 2>/dev/null | awk 'NR==2{gsub(/G/,"");print $4}' || echo 0)"
    elif [ "$OS" = "Darwin" ]; then
      VM_CPU="$(sysctl -n hw.ncpu)"
      VM_MEM_MIB="$(sysctl -n hw.memsize | awk '{printf "%d", $1/1024/1024}')"
      warn "macOS 物理内存 ≠ Docker VM 上限；按物理算会超分，请在 docker 可用时重跑。"
    else
      die "不支持的系统：$OS" 2
    fi
  fi
fi

# what-if 覆盖
[ -n "${OMC_PROBE_CPU:-}" ]            && { VM_CPU="$OMC_PROBE_CPU"; DETECT_SRC="OMC_PROBE_* 覆盖"; }
[ -n "${OMC_PROBE_MEM_TOTAL_MIB:-}" ]  && { VM_MEM_MIB="$OMC_PROBE_MEM_TOTAL_MIB"; DETECT_SRC="OMC_PROBE_* 覆盖"; }
[ "${VM_CPU:-0}" -gt 0 ] 2>/dev/null || die "探测不到 CPU 核数。" 2
[ "${VM_MEM_MIB:-0}" -gt 0 ] 2>/dev/null || die "探测不到内存总量。" 2
[ "${DISK_FREE_GIB:-0}" -gt 0 ] 2>/dev/null || DISK_FREE_GIB=0
[ -n "$DISK_GIB_OVERRIDE" ] && DISK_FREE_GIB="$DISK_GIB_OVERRIDE"   # --disk-gib 覆盖（Mac df 读不到 VM 盘 / what-if）

log "  探测来源      : $DETECT_SRC"
log "  CPU 核数(VM)  : ${C_B}${VM_CPU}${C_0}"
log "  内存总量(VM)  : ${C_B}$(to_gib "$VM_MEM_MIB") GiB${C_0} (${VM_MEM_MIB} MiB)"
log "  Docker 盘可用 : ${DISK_FREE_GIB} GiB"

# 其它项目（非本 OMC 栈）正在占用的内存 —— 共享引擎时替它们留出余量
OTHER_RESERVE_MIB=0
if [ "$ASSUME_DEDICATED" = 0 ] && command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  while IFS=$'\t' read -r name proj; do
    [ -z "$name" ] && continue
    _skip=0
    for sp in $SELF_PROJECTS; do [ "$proj" = "$sp" ] && _skip=1; done
    [ "$_skip" = 1 ] && continue
    use_raw="$(docker stats --no-stream --format '{{.MemUsage}}' "$name" 2>/dev/null | awk '{print $1}')"
    use_mib="$(awk -v s="$use_raw" 'BEGIN{
      n=s+0; u=s; gsub(/[0-9.]/,"",u);
      if(u ~ /GiB/) printf "%d", n*1024; else if(u ~ /MiB/) printf "%d", n;
      else if(u ~ /KiB/) printf "%d", n/1024; else printf "%d", 0 }')"
    OTHER_RESERVE_MIB=$(( OTHER_RESERVE_MIB + use_mib ))
  done < <(docker ps --format '{{.Names}}\t{{.Label "com.docker.compose.project"}}' 2>/dev/null)
  [ "$OTHER_RESERVE_MIB" -gt 0 ] && \
    log "  其它项目占用  : ${C_Y}$(to_gib "$OTHER_RESERVE_MIB") GiB${C_0}（非 [$SELF_PROJECTS] 容器现用量，已从空闲预算扣除；--assume-dedicated 可忽略）"
fi

# ---------------------------------------------------------------------------
# 2. 空闲预算
# ---------------------------------------------------------------------------
sep "2/4 计算空闲预算"
# VM 内核/dockerd 保留：max(1 GiB, 总量 10%)。VM 已不含宿主 OS，10% 足够。
OS_RESERVE_MIB="$(awk -v t="$VM_MEM_MIB" 'BEGIN{r=t*0.10; if(r<1024)r=1024; printf "%d", r}')"
IDLE_MEM_MIB=$(( VM_MEM_MIB - OS_RESERVE_MIB - OTHER_RESERVE_MIB ))
[ "$IDLE_MEM_MIB" -lt 0 ] && IDLE_MEM_MIB=0
log "  内存空闲预算  : ${C_G}${C_B}$(to_gib "$IDLE_MEM_MIB") GiB${C_0}  = VM $(to_gib "$VM_MEM_MIB") − 引擎保留 $(to_gib "$OS_RESERVE_MIB") − 其它占用 $(to_gib "$OTHER_RESERVE_MIB")"

# ---------------------------------------------------------------------------
# 3. 组件 floor/ceiling/权重 + floor-first 分配
# ---------------------------------------------------------------------------
# 单位 MiB。floor=单机单副本下限（绝不低于）；ceil=单机纵向上限（再大走横向扩展）；
# weight=空闲余量分配权重（内含业务相对重要度：tsdb/worker/pg 重，web 轻）。
#   组件         floor  ceil  权重   依据
#   app           512  1536    8     运维UI+OSS轮询，非设备量驱动，最先让出
#   acs           512  2048   12     TR-069 长连接堆；>1万会话走横向多副本
#   worker        768  2560   20     PM/MR XML 解析最吃内存
#   postgres     1024  6144   14     业务主库；KPI/时序已分离到 tsdb，本库写量较小
#   postgres-tsdb 1024 8192   22     时序库：承载 PM COPY 入库 + KPI 聚合，写压力主要在此
#   redis-core   2048  6144    6     ACS 会话/任务/告警，保留 1GiB AOF COW
#   redis-pm     8192 12288   10     PM 双小时窗口/重算，保留 2GiB AOF COW
#   nats          384  1024    6     JetStream file store + PM 突发 in-flight
#   minio         512  2048    6     小内存开发机保持弹性；32GiB 及以上档位在分配后重排到 6GiB
#   web           192   512    0     nginx 静态+反代，近似固定
COMP_NAMES=(app acs worker postgres postgres-tsdb redis-core redis-pm nats minio web)
COMP_FLOOR=(512 512 768 1024 1024 2048 8192 384 512 192)
COMP_CEIL=(1536 2048 2560 6144 8192 6144 12288 1024 2048 512)
COMP_WEIGHT=(8 12 20 14 22 6 10 6 6 0)

# 监控栈（固定块，不纵向伸缩）：prometheus1024+grafana512+loki512+tempo1024+otelcol512
#   +alertmgr512+exporters(128*3)+cadvisor256 ≈ 4736 MiB。dev 本地默认不起，故默认不计入。
MON_FIXED_MIB=0
[ "$WITH_MONITORING" = 1 ] && MON_FIXED_MIB=4736

FLOOR_SUM=0; for f in "${COMP_FLOOR[@]}"; do FLOOR_SUM=$(( FLOOR_SUM + f )); done

if [ "$MAXIMIZE" = 1 ]; then
  sep "3/4 maximize 分配（独占生产机，最大化吃硬件）"
  # CPU：所有服务限额 = 全部逻辑核 = 实际不限（谁抢到是谁的，不按进程切）。
  CPU_pg=$VM_CPU; CPU_tsdb=$VM_CPU; CPU_worker=$VM_CPU; CPU_acs=$VM_CPU; CPU_app=$VM_CPU
  CPU_redis_core=$VM_CPU; CPU_redis_pm=$VM_CPU; CPU_nats=$VM_CPU; CPU_minio=$VM_CPU; CPU_web=$VM_CPU
  # 内存：PG 最佳实践 —— shared_buffers 总量 ≈ 25% 物理内存（其余留 OS page cache，
  # TimescaleDB 列存解压/读全靠页缓存，故不把内存全塞进 cap）；两库按时序库偏重切
  # （tsdb 60% / 主库 40%）。各 PG cap = 该库 shared_buffers ÷ 0.4（SB 占 cap 40%，
  # 留 60% 给 backends/work/temp）。caps 允许 > 物理内存（仅 PG/worker 真吃得多）。
  SB_TOTAL=$(pct "$VM_MEM_MIB" 25)
  PG_SB=$(awk -v s="$SB_TOTAL" 'BEGIN{printf "%d", s*0.4}')
  TSDB_SB=$(awk -v s="$SB_TOTAL" 'BEGIN{printf "%d", s*0.6}')
  PG_MEM=$(awk -v s="$PG_SB" 'BEGIN{printf "%d", s*2.5}')
  TSDB_MEM=$(awk -v s="$TSDB_SB" 'BEGIN{printf "%d", s*2.5}')
  clampm() { awk -v r="$VM_MEM_MIB" -v p="$1" -v lo="$2" -v hi="$3" 'BEGIN{v=r*p/100; if(v<lo)v=lo; if(v>hi)v=hi; printf "%d", v}'; }
  WORKER_MEM=$(clampm 10 4096 24576)   # PM/MR XML 解析最吃内存
  ACS_MEM=$(clampm 6 4096 16384)       # TR-069 长连接会话堆
  APP_MEM=$(clampm 3 2048 8192)
  MINIO_MEM=$(clampm 5 6144 8192)      # 海量小对象的可回收 slab 在 4GiB cap 下持续触发 memory.max；32GiB 档至少 6GiB
  NATS_MEM=$(clampm 3 1024 4096)
  REDIS_CORE_MEM=$(clampm 4 4096 6144)
  REDIS_PM_MEM=$(clampm 8 8192 12288)  # 12分钟关窗需同时容纳相邻两个小时窗口
  WEB_MEM=512
  log "  物理内存      : $(to_gib "$VM_MEM_MIB") GiB；shared_buffers 总量 $(to_gib "$SB_TOTAL") GiB(25%)，余量留 OS page cache"
  log "  CPU 限额      : 各服务 = 全部 ${VM_CPU} 逻辑核（谁抢到是谁的，不按进程切）"
else
sep "3/4 floor-first 分配"
log "  组件 cap 下限和 : $(to_gib "$FLOOR_SUM") GiB$([ "$WITH_MONITORING" = 1 ] && echo '（另含监控固定块 '"$(to_gib "$MON_FIXED_MIB")"' GiB）')"

# 余量分配（caps 为上限，允许 Σcap > VM —— 只有 PG 会吃满，见原则1；故不在此 die）
SURPLUS=$(( IDLE_MEM_MIB - FLOOR_SUM - MON_FIXED_MIB )); [ "$SURPLUS" -lt 0 ] && SURPLUS=0
if [ $(( FLOOR_SUM + MON_FIXED_MIB )) -gt "$IDLE_MEM_MIB" ]; then
  warn "空闲预算 $(to_gib "$IDLE_MEM_MIB") GiB < 组件下限和 $(to_gib $(( FLOOR_SUM + MON_FIXED_MIB ))) GiB：按下限分配。"
  warn "  cap 是上限（仅 PG 实占），通常仍安全；但若其它项目占用很大，请先释放或加 --assume-dedicated。"
fi
WEIGHT_SUM=0; for w in "${COMP_WEIGHT[@]}"; do WEIGHT_SUM=$(( WEIGHT_SUM + w )); done

declare -a COMP_MEM
for i in "${!COMP_NAMES[@]}"; do
  floor=${COMP_FLOOR[$i]}; ceil=${COMP_CEIL[$i]}; w=${COMP_WEIGHT[$i]}; add=0
  [ "$w" -gt 0 ] && [ "$WEIGHT_SUM" -gt 0 ] && add=$(( SURPLUS * w / WEIGHT_SUM ))
  mem=$(( floor + add )); [ "$mem" -gt "$ceil" ] && mem=$ceil
  COMP_MEM[$i]=$mem
done

idx() { local n="$1"; for i in "${!COMP_NAMES[@]}"; do [ "${COMP_NAMES[$i]}" = "$n" ] && { echo "$i"; return; }; done; }

# 32GiB 开发/压测档与发布规划保持一致：从其它组件 floor 以上的 cap 余量重排，
# 给 MinIO 6GiB；小内存开发机继续使用原弹性 floor，避免本地环境被固定大 cap 挤占。
if [ "$VM_MEM_MIB" -ge 32768 ]; then
  MINIO_IDX=$(idx minio); MINIO_TARGET_MIB=6144
  if [ "${COMP_MEM[$MINIO_IDX]}" -lt "$MINIO_TARGET_MIB" ]; then
    MINIO_NEED=$(( MINIO_TARGET_MIB - COMP_MEM[$MINIO_IDX] ))
    for reclaim_name in app nats web redis-core worker postgres postgres-tsdb redis-pm; do
      [ "$MINIO_NEED" -gt 0 ] || break
      reclaim_idx=$(idx "$reclaim_name")
      reclaimable=$(( COMP_MEM[$reclaim_idx] - COMP_FLOOR[$reclaim_idx] ))
      [ "$reclaimable" -gt 0 ] || continue
      take="$reclaimable"; [ "$take" -gt "$MINIO_NEED" ] && take="$MINIO_NEED"
      COMP_MEM[$reclaim_idx]=$(( COMP_MEM[$reclaim_idx] - take ))
      MINIO_NEED=$(( MINIO_NEED - take ))
    done
    [ "$MINIO_NEED" -eq 0 ] || die "32GiB 档位无法在不削减组件 floor 的前提下为 MinIO 保留 6GiB" 1
    COMP_MEM[$MINIO_IDX]="$MINIO_TARGET_MIB"
  fi
fi

APP_MEM=${COMP_MEM[$(idx app)]};            ACS_MEM=${COMP_MEM[$(idx acs)]}
WORKER_MEM=${COMP_MEM[$(idx worker)]};      PG_MEM=${COMP_MEM[$(idx postgres)]}
TSDB_MEM=${COMP_MEM[$(idx postgres-tsdb)]}
REDIS_CORE_MEM=${COMP_MEM[$(idx redis-core)]}; REDIS_PM_MEM=${COMP_MEM[$(idx redis-pm)]}
NATS_MEM=${COMP_MEM[$(idx nats)]};          MINIO_MEM=${COMP_MEM[$(idx minio)]}
WEB_MEM=${COMP_MEM[$(idx web)]}

# ---- CPU 限额（突发可超分；按 VM 核数缩放，封顶 VM 核数）----
cpu_share() { local frac="$1" minv="$2"; awk -v c="$VM_CPU" -v f="$frac" -v m="$minv" \
  'BEGIN{v=int(c*f+0.5); if(v<m)v=m; if(v>c)v=c; if(v<1)v=1; printf "%d", v}'; }
CPU_pg=$(cpu_share 1.0 2);   CPU_tsdb=$(cpu_share 1.0 2)   # 两 PG 可突发到全核
CPU_worker=$(cpu_share 0.6 2); CPU_acs=$(cpu_share 0.35 2); CPU_app=$(cpu_share 0.35 2)
CPU_redis_core=$(cpu_share 0.15 1); CPU_redis_pm=$(cpu_share 0.15 1); CPU_nats=$(cpu_share 0.15 1)
CPU_minio=$(cpu_share 0.15 1); CPU_web=$(cpu_share 0.1 1)
fi

# 两种规划模式都在最终 NATS_MEM 确定后统一派生，避免 maximize 分支漏定义。
NATS_MAX_MEMORY_STORE=$(( NATS_MEM * 1024 * 1024 / 4 ))

# ---- 联动派生 ----
gomemlimit() { pct "$1" 90; }                          # GOMEMLIMIT = 0.90 × 内存限额（软限）
APP_GOMEM=$(gomemlimit "$APP_MEM"); ACS_GOMEM=$(gomemlimit "$ACS_MEM"); WORKER_GOMEM=$(gomemlimit "$WORKER_MEM")
if [ "$MAXIMIZE" = 1 ]; then
  APP_GOMAXPROCS=""; ACS_GOMAXPROCS=""; WORKER_GOMAXPROCS=""   # 不限 GOMAXPROCS，Go 用满所有核（谁抢谁的）
else
  APP_GOMAXPROCS=$(floori "$CPU_app"); ACS_GOMAXPROCS=$(floori "$CPU_acs"); WORKER_GOMAXPROCS=$(floori "$CPU_worker")
fi

# Postgres / TimescaleDB（两实例同算法，各按自己的 *_MEM 派生）
PG_MAXCONN=300; TSDB_MAXCONN=300                       # 覆盖 Go 端连接池总和 ~180 + 余量
# work_mem 按【单实例 cap】定档：双 PG 拆分后每实例 cap 较小，阈值相应下调，使现实 dev 档的
# 时序库也能到 12MB —— KPI GROUP BY/排序的吞吐杠杆，对齐容量规划 §3.2 12GB 档的 work_mem 意图。
pg_work()  { [ "$1" -ge 3072 ] && echo 16 || { [ "$1" -ge 1536 ] && echo 12 || echo 8; }; }
pg_maint() { awk -v m="$1" 'BEGIN{v=m*0.05; if(v>1024)v=1024; if(v<256)v=256; printf "%d", v}'; }
pg_wal()   { [ "$1" -ge 6144 ] && echo 4GB || echo 2GB; }
# 每实例「backends + 排序 work_mem + 临时文件」实占余量：idle/temp 基线 384 + work_mem×~32 活跃排序。
# 与 work_mem 联动 —— 抬 work_mem 同步抬余量，使下方 OOM 自检不被架空。
pg_allow() { echo $(( 384 + $1 * 32 )); }
if [ "$MAXIMIZE" = 1 ]; then
  # maximize：PG_SB/TSDB_SB/PG_MEM/TSDB_MEM 已按 25% 物理内存设定；这里补 eff/work/maint/wal（大档）。
  PG_EFF=$(pct "$VM_MEM_MIB" 50);  TSDB_EFF=$(pct "$VM_MEM_MIB" 50)   # 大页缓存提示（两库共享 OS page cache）
  PG_WORK=$([ "$VM_MEM_MIB" -ge 98304 ] && echo 48 || echo 32)        # ≥96G→48MB 否则 32MB
  TSDB_WORK=$PG_WORK
  PG_MAINT=$(awk -v m="$PG_MEM" 'BEGIN{v=m*0.05; if(v>2048)v=2048; if(v<256)v=256; printf "%d", v}')
  TSDB_MAINT=$(awk -v m="$TSDB_MEM" 'BEGIN{v=m*0.05; if(v>2048)v=2048; if(v<256)v=256; printf "%d", v}')
  PG_WAL=8GB; TSDB_WAL=8GB
else
  PG_SB=$(pct "$PG_MEM" 25);     PG_EFF=$(pct "$PG_MEM" 60)
  PG_WORK=$(pg_work "$PG_MEM");  PG_MAINT=$(pg_maint "$PG_MEM");   PG_WAL=$(pg_wal "$PG_MEM")
  TSDB_SB=$(pct "$TSDB_MEM" 25); TSDB_EFF=$(pct "$TSDB_MEM" 60)
  TSDB_WORK=$(pg_work "$TSDB_MEM"); TSDB_MAINT=$(pg_maint "$TSDB_MEM"); TSDB_WAL=$(pg_wal "$TSDB_MEM")
fi
PG_ALLOW=$(pg_allow "$PG_WORK"); TSDB_ALLOW=$(pg_allow "$TSDB_WORK")

# ---- 双 PG 合计实占 OOM 自检（关键）----
# PG 真正吃满的是 shared_buffers + maintenance + 每实例 backends/work/temp 余量。
# 非 PG 服务实测只用几十~几百 MB（按实占估，非 cap）；据此留给两 PG 的安全预算 = PG_AVAIL。
NONPG_ACTUAL_EST=$(( 400 + 350 + 600 + 200 + 350 + 60 ))  # app/acs/worker/nats/minio/web 实占估
REDIS_CORE_MAXMEM=$(( REDIS_CORE_MEM - 1024 ))
REDIS_PM_MAXMEM=$(( REDIS_PM_MEM - 2048 ))
REDIS_POLICY=noeviction
NONPG_ACTUAL_EST=$(( NONPG_ACTUAL_EST + REDIS_CORE_MAXMEM + REDIS_PM_MAXMEM ))
[ "$WITH_MONITORING" = 1 ] && NONPG_ACTUAL_EST=$(( NONPG_ACTUAL_EST + 2000 ))  # 监控实占估
PG_AVAIL=$(( VM_MEM_MIB - OS_RESERVE_MIB - OTHER_RESERVE_MIB - NONPG_ACTUAL_EST ))

# (a) VM 级：两库合计实占（各 SB+maint+allow）须 ≤ 留给 PG 的预算，否则按比例压两库 shared_buffers。
PG_FOOT=$(( PG_SB + PG_MAINT + PG_ALLOW + TSDB_SB + TSDB_MAINT + TSDB_ALLOW ))
if [ "$PG_FOOT" -gt "$PG_AVAIL" ]; then
  ROOM=$(( PG_AVAIL - (PG_MAINT + TSDB_MAINT + PG_ALLOW + TSDB_ALLOW) ))   # 留给两库 shared_buffers 之和
  if [ "$ROOM" -lt 256 ]; then
    REC=$(( (NONPG_ACTUAL_EST + OS_RESERVE_MIB + (PG_MAINT + PG_ALLOW + 128) + (TSDB_MAINT + TSDB_ALLOW + 128)) / 1024 + 1 ))
    die "本机内存不足以安全运行【双 PostgreSQL 实例 + 全栈】。
       VM $(to_gib "$VM_MEM_MIB") GiB − 引擎/其它/非PG实占 后，仅余 $(to_gib "$PG_AVAIL") GiB 给两个 PG，
       连两库最低 shared_buffers(各128MB)+维护+backends 都放不下。
       建议：Docker Desktop VM 调到 ≥ ${REC} GiB（设置→Resources→Memory），或释放其它项目占用，
       或临时只起单库（不起 postgres-tsdb）后重试。" 1
  fi
  NEW_SB=$(( ROOM / 2 )); [ "$NEW_SB" -lt 128 ] && NEW_SB=128
  warn "双 PG 合计实占自检触发：shared_buffers 由 ${PG_SB}/${TSDB_SB}MB 同步下调到 ${NEW_SB}MB（保两库合计 + 其它栈 ≤ VM，防 OOM）。"
  warn "  如需更大 shared_buffers：调大 Docker Desktop VM 内存，或 --assume-dedicated（确认无其它项目）。"
  PG_SB=$NEW_SB; TSDB_SB=$NEW_SB
  PG_EFF=$(( PG_SB * 2 )); TSDB_EFF=$(( TSDB_SB * 2 ))   # effective_cache_size 跟随，仍 < cap
fi

# (b) 单实例级：cap 必须 ≥ 本实例实占（SB+maint+allow），否则容器 cgroup 会先于 VM OOM-kill，
# 而 (a) 的 VM 级自检看不出来（曾打印误导性的 ✓）。caps 允许 > VM（原则1，仅 PG 真吃满），
# 故这里【抬 cap】而非再压 SB —— 落实容量规划 §2「容器 cap ≥ shared_buffers+maintenance+backend 余量」。
PG_FOOT1=$(( PG_SB + PG_MAINT + PG_ALLOW ));        [ "$PG_MEM" -lt "$PG_FOOT1" ]   && PG_MEM=$PG_FOOT1
TSDB_FOOT1=$(( TSDB_SB + TSDB_MAINT + TSDB_ALLOW )); [ "$TSDB_MEM" -lt "$TSDB_FOOT1" ] && TSDB_MEM=$TSDB_FOOT1

# ---------------------------------------------------------------------------
# 4. 打印 + 写 resources.env
# ---------------------------------------------------------------------------
ALLOC_SUM=$(( APP_MEM + ACS_MEM + WORKER_MEM + PG_MEM + TSDB_MEM + REDIS_CORE_MEM + REDIS_PM_MEM + NATS_MEM + MINIO_MEM + WEB_MEM + MON_FIXED_MIB ))
sep "4/4 规划结果"
printf '%b\n' "${C_B}  组件            内存cap   CPUcap   关键联动${C_0}"
printf '  %-14s %6sMiB  %5s   GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' app    "$APP_MEM"    "$CPU_app"    "$APP_GOMEM"    "$APP_GOMAXPROCS"
printf '  %-14s %6sMiB  %5s   GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' acs    "$ACS_MEM"    "$CPU_acs"    "$ACS_GOMEM"    "$ACS_GOMAXPROCS"
printf '  %-14s %6sMiB  %5s   GOMEMLIMIT=%sMiB GOMAXPROCS=%s\n' worker "$WORKER_MEM" "$CPU_worker" "$WORKER_GOMEM" "$WORKER_GOMAXPROCS"
printf '  %-14s %6sMiB  %5s   shared_buffers=%sMB effective_cache=%sMB work_mem=%sMB maint=%sMB\n' postgres      "$PG_MEM"   "$CPU_pg"   "$PG_SB"   "$PG_EFF"   "$PG_WORK"  "$PG_MAINT"
printf '  %-14s %6sMiB  %5s   shared_buffers=%sMB effective_cache=%sMB work_mem=%sMB maint=%sMB\n' postgres-tsdb "$TSDB_MEM" "$CPU_tsdb" "$TSDB_SB" "$TSDB_EFF" "$TSDB_WORK" "$TSDB_MAINT"
printf '  %-14s %6sMiB  %5s   maxmemory=%sMB policy=%s（COW余量=%sMiB）\n' redis-core "$REDIS_CORE_MEM" "$CPU_redis_core" "$REDIS_CORE_MAXMEM" "$REDIS_POLICY" "$(( REDIS_CORE_MEM - REDIS_CORE_MAXMEM ))"
printf '  %-14s %6sMiB  %5s   maxmemory=%sMB policy=%s（COW余量=%sMiB）\n' redis-pm "$REDIS_PM_MEM" "$CPU_redis_pm" "$REDIS_PM_MAXMEM" "$REDIS_POLICY" "$(( REDIS_PM_MEM - REDIS_PM_MAXMEM ))"
printf '  %-14s %6sMiB  %5s\n' nats  "$NATS_MEM"  "$CPU_nats"
printf '  %-14s %6sMiB  %5s\n' minio "$MINIO_MEM" "$CPU_minio"
printf '  %-14s %6sMiB  %5s\n' web   "$WEB_MEM"   "$CPU_web"
[ "$WITH_MONITORING" = 1 ] && printf '  %-14s %6sMiB（固定块）\n' monitoring "$MON_FIXED_MIB"
log "  ──────────────────────────────────────"
log "  Σ内存 cap    : ${C_B}$(to_gib "$ALLOC_SUM") GiB${C_0}（含双 PG；caps 为上限，仅 PG 实占，允许 > VM）"
log "  两 PG 合计实占估：$(to_gib $(( PG_SB + TSDB_SB + PG_MAINT + TSDB_MAINT + PG_ALLOW + TSDB_ALLOW ))) GiB ≤ 留给 PG 的 $(to_gib "$PG_AVAIL") GiB ✓"
log "  单实例 cap ≥ 实占：postgres $(to_gib "$PG_MEM")≥$(to_gib "$PG_FOOT1") / tsdb $(to_gib "$TSDB_MEM")≥$(to_gib "$TSDB_FOOT1") GiB ✓"

# 磁盘提醒（dev：与容量规划 §4.3 一致）
if [ "$MAXIMIZE" = 0 ] && [ "$DISK_FREE_GIB" -gt 0 ] && [ "$DISK_FREE_GIB" -lt 40 ]; then
  warn "Docker 盘仅剩 ${DISK_FREE_GIB} GiB：PM/MR/备份/镜像层易写满。dev 建议 ≥ 40 GiB；"
  warn "  生产时序数据量大（混合制式约 150MB/设备/30天），须配独立数据盘 + 保留期/降采样。"
fi

# ---------------------------------------------------------------------------
# 保留期自动测算：磁盘容量 → 15min 原始数据在【时序库】可保留天数（上限 RETENTION_DAYS）
# ---------------------------------------------------------------------------
# 模型（透明、可校验；混合制式，实际随 RAT/压缩比浮动）：
#   每设备每天 15min 原始 PM ~17MB（容量规划 §4.3）。保留 D 天的 DB 占用 ≈
#     [min(D,7)×17 + max(0,D-7)×17/12]（7 天原始 + 其后列存压缩 ~12×）× 1.2（聚合/KPI/索引开销）。
#   MinIO 原始件若同按 D 天保留：设备多按 .gz 上传 ~3MB/设备/天 → D×3 MB。
#   固定基线（镜像层/WAL/索引/业务库/OS）：BASE_GIB；并对可用盘留 30% watchdog 余量。
PM_RAW_RETENTION_DAYS=""
RET_SQL="${OUT_FILE%/*}/retention.sql"
per_dev_mib() { awk -v D="$1" 'BEGIN{
  raw=(D<7?D:7)*17 + (D>7?(D-7):0)*17/12; db=raw*1.2; minio=D*3; printf "%d", db+minio }'; }
if [ "$DISK_FREE_GIB" -gt 0 ]; then
  [ "$RETENTION_DAYS" -ge 7 ] 2>/dev/null || die "--retention-days 需 ≥7（15min 原始至少留一个压缩窗口），收到：$RETENTION_DAYS" 2
  sep "保留期测算（磁盘 → 15min 原始数据可保留天数，目标 ${RETENTION_DAYS} 天）"
  BASE_GIB=$([ "$MAXIMIZE" = 1 ] && echo 80 || echo 20)
  DISK_BUDGET_GIB=$(awk -v d="$DISK_FREE_GIB" -v b="$BASE_GIB" 'BEGIN{v=d*0.70-b; if(v<0)v=0; printf "%d", v}')
  log "  盘可用 ${DISK_FREE_GIB} GiB → 数据预算 ${C_B}${DISK_BUDGET_GIB}${C_0} GiB（70%×盘 − 基线 ${BASE_GIB} GiB）"
  if [ -n "$DEVICES" ]; then
    [ "$DEVICES" -gt 0 ] 2>/dev/null || die "--devices 需正整数，收到：$DEVICES" 2
    NEED=$(awk -v n="$DEVICES" -v p="$(per_dev_mib "$RETENTION_DAYS")" 'BEGIN{printf "%d", n*p/1024}')
    if [ "$NEED" -le "$DISK_BUDGET_GIB" ]; then
      PM_RAW_RETENTION_DAYS="$RETENTION_DAYS"
      log "  ${DEVICES} 台 @ ${RETENTION_DAYS} 天 ≈ ${NEED} GiB ≤ 预算 ${DISK_BUDGET_GIB} GiB ✓ → 保留 ${C_G}${RETENTION_DAYS}${C_0} 天"
    else
      d="$RETENTION_DAYS"
      while [ "$d" -gt 7 ]; do
        nd=$(awk -v n="$DEVICES" -v p="$(per_dev_mib "$d")" 'BEGIN{printf "%d", n*p/1024}')
        [ "$nd" -le "$DISK_BUDGET_GIB" ] && break
        d=$(( d - 1 ))
      done
      PM_RAW_RETENTION_DAYS="$d"
      warn "${DEVICES} 台 @ ${RETENTION_DAYS} 天需 ~${NEED} GiB > 预算 ${DISK_BUDGET_GIB} GiB：自动下调 15min 原始保留期到 ${d} 天。"
      warn "  要保满 ${RETENTION_DAYS} 天：扩数据盘，或降采样（小时/天聚合保留更久、15min 原始更短）。"
    fi
  else
    MAXDEV=$(awk -v b="$DISK_BUDGET_GIB" -v p="$(per_dev_mib "$RETENTION_DAYS")" 'BEGIN{printf "%d", (p>0)?b*1024/p:0}')
    log "  未给 --devices：本盘 @ ${RETENTION_DAYS} 天约可承载 ${C_B}${MAXDEV}${C_0} 台设备的 15min 历史。"
    log "  传 --devices N 即按你的设备数算可行保留天数并写 retention.sql。"
  fi
else
  [ "$MAXIMIZE" = 1 ] && warn "未探测到磁盘可用量（Mac 读不到 VM 盘）：保留期测算跳过，请在目标 Linux 机重跑或加 --disk-gib N。"
fi

if [ "$DRY_RUN" = 1 ]; then
  log "\n${C_Y}--dry-run：未写文件。${C_0}去掉 --dry-run 即生成 $OUT_FILE"
  exit 0
fi

{
  echo "# =============================================================================="
  echo "# resources.env —— OMC 容器资源限额（plan-resources.sh 生成，可手改）"
  echo "# 模式: $([ "$MAXIMIZE" = 1 ] && echo 'maximize（独占生产机，最大化吃硬件）' || echo 'dev（floor-first 自适应）')"
  echo "# 探测: ${VM_CPU}核 / $(to_gib "$VM_MEM_MIB")GiB   空闲预算: $(to_gib "$IDLE_MEM_MIB")GiB   来源: $DETECT_SRC"
  echo "# 用法：docker-compose.yml 以 \${VAR:-默认} 读取本文件。日常："
  echo "#   bash deployments/docker/dc.sh up -d --build       # wrapper 自动带 --env-file"
  echo "#   # 或手工 docker compose -f deployments/docker/docker-compose.yml \\"
  echo "#   #          --env-file deployments/docker/resources.env up -d"
  echo "# 不传 env-file 时取 compose 内 :- 默认（= 历史 PM 写吞吐档），行为不变。"
  echo "# 约束（手改时务必遵守，否则重演 OOM 事故）："
  echo "#   · *_GOMEMLIMIT 必须 < 对应 *_MEM（软限，建议 0.90×）"
  echo "#   · REDIS_CORE_MEM 须保留 1GiB、REDIS_PM_MEM 须保留 2GiB AOF rewrite COW 余量"
  echo "#   · 两 PG 的 shared_buffers 之和 + maintenance + backends 余量 须 < VM 内存（防双库 OOM）"
  echo "#   · PG/TSDB_MAX_CONNECTIONS 必须 ≥ Go 端连接池总和（当前 ~180）"
  echo "# =============================================================================="
  echo ""
  echo "# ── 业务（Go）── *_MEM 是 cgroup 硬限；*_GOMEMLIMIT 是 Go 堆软限（0.90×）"
  echo "APP_CPUS=$CPU_app";       echo "APP_MEM=${APP_MEM}m";       echo "APP_GOMEMLIMIT=${APP_GOMEM}MiB";       echo "APP_GOMAXPROCS=$APP_GOMAXPROCS"
  echo "ACS_CPUS=$CPU_acs";       echo "ACS_MEM=${ACS_MEM}m";       echo "ACS_GOMEMLIMIT=${ACS_GOMEM}MiB";       echo "ACS_GOMAXPROCS=$ACS_GOMAXPROCS"
  echo "WORKER_CPUS=$CPU_worker"; echo "WORKER_MEM=${WORKER_MEM}m"; echo "WORKER_GOMEMLIMIT=${WORKER_GOMEM}MiB"; echo "WORKER_GOMAXPROCS=$WORKER_GOMAXPROCS"
  echo ""
  echo "# ── PostgreSQL 主库（业务数据）── 限额与 -c 调优同源派生"
  echo "PG_CPUS=$CPU_pg";         echo "PG_MEM=${PG_MEM}m"
  echo "PG_SHARED_BUFFERS=${PG_SB}MB"; echo "PG_EFFECTIVE_CACHE_SIZE=${PG_EFF}MB"
  echo "PG_MAX_CONNECTIONS=$PG_MAXCONN"; echo "PG_WORK_MEM=${PG_WORK}MB"
  echo "PG_MAINTENANCE_WORK_MEM=${PG_MAINT}MB"; echo "PG_MAX_WAL_SIZE=$PG_WAL"
  echo ""
  echo "# ── PostgreSQL 时序库 postgres-tsdb（PM/KPI 时序，写压力主要在此）──"
  echo "TSDB_CPUS=$CPU_tsdb";     echo "TSDB_MEM=${TSDB_MEM}m"
  echo "TSDB_SHARED_BUFFERS=${TSDB_SB}MB"; echo "TSDB_EFFECTIVE_CACHE_SIZE=${TSDB_EFF}MB"
  echo "TSDB_MAX_CONNECTIONS=$TSDB_MAXCONN"; echo "TSDB_WORK_MEM=${TSDB_WORK}MB"
  echo "TSDB_MAINTENANCE_WORK_MEM=${TSDB_MAINT}MB"; echo "TSDB_MAX_WAL_SIZE=$TSDB_WAL"
  echo ""
  echo "# ── Redis Core / PM ── 物理隔离，均禁止淘汰"
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
  echo "# ── 规划元信息（compose 不读取，仅记录）──"
  echo "OMC_RESOURCE_SCHEMA_VERSION=3"
  echo "OMC_PLAN_MODE=$([ "$MAXIMIZE" = 1 ] && echo maximize || echo dev)"
  echo "OMC_PLAN_VM_CPU=$VM_CPU"
  echo "OMC_PLAN_VM_MEM_MIB=$VM_MEM_MIB"
  echo "OMC_PLAN_WITH_MONITORING=$WITH_MONITORING"
  [ -n "$PM_RAW_RETENTION_DAYS" ] && echo "OMC_PLAN_PM_RAW_RETENTION_DAYS=$PM_RAW_RETENTION_DAYS  # 见 retention.sql；改 sys_configs 热加载"
} > "$OUT_FILE"

log "\n${C_G}✓ 已写入：$OUT_FILE${C_0}"

# 写 retention.sql（仅当测算出了可行保留天数）—— sys_configs 改后服务热加载，无需重启
if [ -n "$PM_RAW_RETENTION_DAYS" ]; then
  {
    echo "-- retention.sql —— plan-resources.sh 据磁盘容量测算的 PM 15min 原始数据保留天数（${PM_RAW_RETENTION_DAYS} 天）"
    echo "-- 应用：docker compose ... exec -T postgres-tsdb psql -U omcgo -d omcgo -f - < retention.sql"
    echo "--   或：psql \"postgres://omcgo:omcgo123@<host>:5433/omcgo\" -f retention.sql"
    echo "-- sys_configs 改后由 retention.Service 经 SysConfigSavedHook 热加载，无需重启。"
    echo "-- ⚠️ 仅设 DB 侧 15min 原始保留；MinIO 原始件 ILM（现硬编码 14 天）与基站日志按时间保留 = 后端特性，见对应 issue。"
    echo "UPDATE sys_configs SET value = '${PM_RAW_RETENTION_DAYS}' WHERE category = 'pm.retention' AND key = 'raw_15min_days';"
  } > "$RET_SQL"
  log "${C_G}✓ 已写入：$RET_SQL${C_0}（15min 原始保留 ${PM_RAW_RETENTION_DAYS} 天，应用后热加载）"
fi

log "  下一步：检视/微调 → ${C_B}bash deployments/docker/dc.sh up -d --build${C_0}（或手工带 --env-file）$([ "$MAXIMIZE" = 1 ] && echo '；生产记得 --assume-dedicated 重跑' )"
