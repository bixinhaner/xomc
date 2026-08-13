package acs

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/acs/auth"
	"github.com/omcgo/omcgo/internal/acs/connreq"
	"github.com/omcgo/omcgo/internal/acs/rpc"
	"github.com/omcgo/omcgo/internal/acs/rpclog"
	"github.com/omcgo/omcgo/internal/acs/stun"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/middleware"
	"github.com/omcgo/omcgo/internal/core/tracing"
	"github.com/omcgo/omcgo/internal/netutil"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/trace"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ConnectionRequester 发送 Connection Request 唤醒设备。
// 用于会话结束后命令队列仍有待执行命令时的续唤机制。
type ConnectionRequester interface {
	// Send 发送 Connection Request。httpURL 为设备的 HTTP CR URL（可能为空）。
	Send(ctx context.Context, deviceSN, httpURL string) error
}

// SessionCookieName TR069 会话 ID 的 Cookie 名称
const SessionCookieName = "SESSION"

const maxStaleTaskSkipsPerDispatch = 256

// connSessionEntry 连接级会话绑定，记录创建时间用于 TTL 清理。
type connSessionEntry struct {
	DeviceSN  string
	CreatedAt time.Time
}

// Handler 处理 TR069/CWMP HTTP 请求。
type Handler struct {
	sessionStore  SessionStore
	taskService   TaskService // 统一任务管理服务
	eventBus      event.EventBus
	authenticator auth.DeviceAuthenticator
	rpcDispatcher *rpc.Dispatcher
	rateLimiter   *DeviceRateLimiter
	admission     AdmissionController
	metrics       *ACSMetrics
	// localActiveSessions 记录本进程最近五分钟跟踪过的 session ID，仅驱动本地诊断指标。
	// 共享 SessionStore 可能包含重启前或其它实例的会话，清理它们时不能递减本进程 gauge。
	localActiveSessions     sync.Map
	logger                  *zap.Logger
	requestIDPrefix         string                    // 请求 ID 前缀，如 "acs"
	enableTestTaskInjection bool                      // 启用随机测试任务注入（仅测试用）
	uploadConfig            *appconfig.UploadConfig   // 上传服务器配置，用于生成上传 URL
	downloadConfig          *appconfig.DownloadConfig // 下载服务器配置，用于生成下载 URL
	transferConfigProvider  transfercfg.Provider
	// maxRPCPerSession 限制每个 TR069 会话的 RPC 交互次数。
	// 达到上限后优雅结束会话；剩余命令留在队列中，通过会话后续唤在后续会话中下发。
	// 0 表示不限制。建议：≤15，避免触发 CPE 的单会话交互上限。
	maxRPCPerSession int
	// 会话后续唤：会话结束时如果命令队列仍有待执行命令，发送 Connection Request。
	connReqSender      ConnectionRequester
	postSessionWakeCfg appconfig.PostSessionWakeConfig
	redisClient        redis.Cmdable // 用于连续唤醒计数器
	// onlineIndex 在线设备索引（acs:online ZSET）。issue #397：收 Inform 当场、发 NATS
	// 之前同步 ZADD，作为「免 NATS」的存活信号。nil 时 Mark 安全 no-op。
	onlineIndex *redisx.OnlineIndex
	stunStore   *stun.Store // 缓存 Inform 中的设备 STUN 地址
	// connReqURLStore 共享存储设备的 ConnectionRequestURL（issue #65 Option B：取代进程内
	// sync.Map）。nil 时退化为不缓存 CR URL —— postSessionWake 的 HTTP 回退拿到空 URL，
	// 行为等价于改造前缓存 miss（STUN 设备不受影响）。
	connReqURLStore ConnReqURLStore
	// protocolLogger 独立的协议交互日志器，记录完整的原始 XML 请求/响应到专用文件。
	// nil 表示协议日志关闭。
	protocolLogger *zap.Logger
	maxBodySize    int // 协议日志 XML 截断阈值（0=不截断）
	// maxRequestBodySize 限制入站 SOAP 请求体字节数，防止超大 POST 导致 OOM。
	// 0 表示用 defaultMaxRequestBodySize。
	maxRequestBodySize int64
	// T-0137 / M1: TR069 报文跟踪 — 命中白名单时旁路投递 capture，热路径开销 < 1ms。
	// 两个字段都为 nil 表示跟踪关闭（默认）。
	traceWhitelist *trace.WhitelistCache
	traceService   traceCaptureSink
	// pathTranslator: ACS 端 standardPath → privatePath 翻译服务。
	// nil 或 !Enabled() 时 → task.Params 原样下发（与改造前行为一致）。
	pathTranslator *PathTranslationService
	// connSessions 映射 HTTP RemoteAddr → connSessionEntry，用于连接级会话追踪。
	// 条目在会话完成时或由后台清理器清除。
	connSessions sync.Map
	// deviceSessionStore 共享存储设备 → 当前活跃 sessionID 指针（issue #65 Option B：
	// 取代进程内 deviceSessions sync.Map）。用于跨实例孤儿会话检测：新 Inform 落在任一
	// 实例都能读到设备上一个 sessionID 并清理。nil 时退化为不做跨实例孤儿检测（单实例
	// 仍由 SessionStore TTL + 准入槽位 TTL 兜底，不泄漏）。
	deviceSessionStore DeviceSessionStore
	// #746: 心跳周期自动调整策略。设备 BOOTSTRAP/BOOT 时入队 GPV 查询当前心跳周期，
	// 与配置目标值比较后决定是否入队 SPV 调整。nil 时功能关闭（不影响 Inform 处理）。
	informPeriodPolicy     *InformPeriodPolicy
	ueCountPolicy          *UECountPolicy
	gpvFaultRecoverer      GPVFaultRecoverer
	durableReadbackEnabled bool
}

// sessionRPCLimitReached 判断会话是否已达到单会话 RPC 上限。
func (h *Handler) sessionRPCLimitReached(session *Session) bool {
	return h.maxRPCPerSession > 0 && session.RPCCount >= h.maxRPCPerSession
}

// startSessionReaper 启动一个后台 goroutine，定期清理 connSessions 中的过期连接级会话。
//
// issue #65（Option B）后设备级孤儿会话不再靠本地映射 + reaper 清理：
//   - 跨实例孤儿检测改在 handleInform 通过 deviceSessionStore.Swap 完成（新 Inform 落在
//     任一实例都能读到并清理设备上一个会话）；
//   - 准入槽位泄漏由准入 sorted set 的 TTL 过期分自愈回收（admitScript 的 ZREMRANGEBYSCORE）；
//   - 会话数据由 SessionStore 的 Redis TTL（5min）自动过期。
//
// 连接级 connSessions 当前没有写入方（历史遗留 sync.Map），这里保留循环作为防御，
// 实际为 no-op；释放准入槽位需 sessionID，连接级条目无 sessionID，故只更新指标。
func (h *Handler) startSessionReaper(interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			h.reapLocalActiveSessions(now, maxAge)
			h.connSessions.Range(func(key, value interface{}) bool {
				entry := value.(connSessionEntry)
				if now.Sub(entry.CreatedAt) > maxAge {
					h.connSessions.Delete(key)
					h.untrackActiveSession(key.(string))
					h.metrics.SessionDuration.Observe(now.Sub(entry.CreatedAt).Seconds())
					h.logger.Warn("reaped stale conn session",
						zap.String("device_sn", entry.DeviceSN),
						zap.String("remote_addr", key.(string)),
						zap.Duration("age", now.Sub(entry.CreatedAt)))
				}
				return true
			})
		}
	}()
}

func (h *Handler) startGlobalAdmissionMetrics(interval time.Duration) {
	if h == nil || h.admission == nil || h.metrics == nil {
		return
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	h.refreshGlobalActiveSessions(context.Background())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), interval)
			h.refreshGlobalActiveSessions(ctx)
			cancel()
		}
	}()
}

func (h *Handler) refreshGlobalActiveSessions(ctx context.Context) {
	if h == nil || h.admission == nil || h.metrics == nil {
		return
	}
	current := float64(h.admission.Current(ctx))
	h.metrics.GlobalActiveSessions.Set(current)
	h.metrics.ActiveSessions.Set(current)
}

// reapOrphanedSession 清理孤儿设备会话（跨实例）：从共享 SessionStore 加载会话，
// 若仍存在则 completeSession 释放资源并触发 postSessionWake；若已 TTL 过期则只释放
// 准入槽位、更新指标并仍触发 postSessionWake（队列可能仍有命令）。
//
// MEDIUM-17：使用带 5s 超时的 ctx（不用裸 context.Background()），避免 Redis 慢/不可达
// 时无限阻塞、拖住优雅关机。5s 对齐 postSessionWake 里已有的 5s 模式。
func (h *Handler) reapOrphanedSession(deviceSN, sessionID, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 尝试从共享 SessionStore 加载会话，获取完整会话数据用于 completeSession。
	session, err := h.sessionStore.GetByID(ctx, sessionID)
	if err != nil {
		h.logger.Warn("reap orphaned session: failed to load from store",
			zap.String("device_sn", deviceSN),
			zap.String("session_id", sessionID),
			zap.String("reason", reason),
			zap.Error(err))
	}

	if session != nil {
		h.logger.Info("reap orphaned session: completing",
			zap.String("device_sn", deviceSN),
			zap.String("session_id", sessionID),
			zap.String("old_state", string(session.State)),
			zap.String("reason", reason))
		h.completeSession(ctx, session)
	} else {
		// 会话已在共享存储中过期（TTL）。仍需释放该 sessionID 的准入槽位和更新指标。
		h.logger.Info("reap orphaned session: session expired in store, releasing resources",
			zap.String("device_sn", deviceSN),
			zap.String("session_id", sessionID),
			zap.String("reason", reason))
		h.admission.Release(ctx, sessionID)
		h.untrackActiveSession(sessionID)
		// 仍触发 postSessionWake —— 即使会话已消失，设备队列中可能仍有待执行命令。
		if h.connReqSender != nil && h.postSessionWakeCfg.Enabled && deviceSN != "" {
			go h.postSessionWake(deviceSN)
		}
	}
}

// defaultMaxRequestBodySize 是入站 SOAP 请求体的默认上限（50MB），
// 在未配置 server.max_request_body_size 时生效。50MB 足以覆盖最大的
// Inform/GetParameterValuesResponse，同时挡住恶意/故障 CPE 的多 GB POST。
const defaultMaxRequestBodySize int64 = 50 << 20 // 50 MiB

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 生成或传播请求 ID
	requestID := r.Header.Get(middleware.RequestIDHeader)
	if requestID == "" {
		prefix := h.requestIDPrefix
		if prefix == "" {
			prefix = "acs" // 默认前缀
		}
		requestID = middleware.GenerateRequestIDWithPrefix(prefix)
	}

	// 将请求 ID 存入上下文，供日志和下游服务使用
	ctx := logger.WithRequestID(r.Context(), requestID)
	r = r.WithContext(ctx)

	// 将请求 ID 设置到响应头，便于客户端关联
	w.Header().Set(middleware.RequestIDHeader, requestID)

	// 创建携带 request_id 的上下文感知日志器
	log := logger.L(ctx)

	if r.Method != http.MethodPost {
		log.Warn("method not allowed", zap.String("method", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 限制入站请求体大小，防止恶意/故障 CPE 发送超大 POST 导致 ACS 进程 OOM。
	// 南向接口互联网可达且 Inform 为首条消息（无认证），必须在读取前设上限。
	maxBody := h.maxRequestBodySize
	if maxBody <= 0 {
		maxBody = defaultMaxRequestBodySize
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			log.Warn("request body exceeds limit, rejecting",
				zap.Int64("limit_bytes", maxBody))
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return
		}
		log.Error("read request body", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 协议日志 / 报文跟踪：用 ResponseCapturer 包装 ResponseWriter，捕获响应字节。
	//
	// issue #218（高 CPU 回归治本）：
	//   - 仅当协议日志器存在 *或* 报文跟踪启用时才包装 capturer 并注册 defer，
	//     避免无人消费时白白缓冲整段响应。
	//   - 每报文全量 XML 的 zap JSON 编码（旧热点）改由协议日志级别短路：协议日志器
	//     默认按 warn 级构建（cmd/acs 侧），下面的 .Info("rpc", ...) 在核心层被丢弃，
	//     **不做整段 XML 的 string 拷贝 + JSON 编码**，稳态零开销；排障调 level=info 即恢复。
	//   - 报文跟踪(trace)与协议日志解耦：trace 命中白名单的捕获不受日志级别影响，照常旁路落库。
	if h.protocolLogger != nil || h.traceWhitelist != nil {
		capturer := rpclog.NewResponseCapturer(w)
		w = capturer
		logEntry := &rpclog.LogEntry{StartTime: time.Now()}
		ctx = rpclog.WithEntry(ctx, logEntry)
		r = r.WithContext(ctx)

		defer func() {
			// 全量 XML 记录开销大，仅在协议日志器真正会落该级别时才构建并编码。
			logFullXML := h.protocolLogger != nil &&
				h.protocolLogger.Core().Enabled(zapcore.InfoLevel)

			// trace 命中检查很轻（白名单 map 查），且需要 XML body；与全量日志相互独立。
			// 两者都不需要时直接跳过整段拷贝/编码。
			if !logFullXML && (h.traceWhitelist == nil || h.traceService == nil) {
				return
			}

			// 原始未截断 XML：trace 捕获必须拿到全量报文（基站 GPV 响应轻易 >4KB），
			// protocol_log.max_body_size 仅约束协议日志落盘，绝不污染 trace 保真度（修 #296）。
			// trace 内部 truncateForInline 按自己的 32KB 阈值处理，与协议日志阈值解耦。
			reqXMLRaw := string(body)
			respXMLRaw := string(capturer.Body())

			if logFullXML {
				// 仅在协议日志真正落 info 级时才构建截断串，保证默认 warn 级零额外开销。
				reqXML, respXML := reqXMLRaw, respXMLRaw
				if h.maxBodySize > 0 {
					if len(reqXML) > h.maxBodySize {
						reqXML = reqXML[:h.maxBodySize] + "...(truncated)"
					}
					if len(respXML) > h.maxBodySize {
						respXML = respXML[:h.maxBodySize] + "...(truncated)"
					}
				}
				h.protocolLogger.Info("rpc",
					zap.String("session_id", logEntry.SessionID),
					zap.String("device_sn", logEntry.DeviceSN),
					zap.Int("sequence", logEntry.Sequence),
					zap.String("method", logEntry.Method),
					zap.String("task_id", logEntry.TaskID),
					zap.String("cwmp_id", logEntry.CwmpID),
					zap.Int("http_status", capturer.StatusCode()),
					zap.Int("duration_ms", int(time.Since(logEntry.StartTime).Milliseconds())),
					zap.String("request_xml", reqXML),
					zap.String("response_xml", respXML),
				)
			}

			// T-0137 / M1: 命中跟踪白名单则旁路落库（不依赖协议日志级别）。
			// 传未截断原始 XML，trace 自行按 32KB 阈值处理。
			h.maybeCaptureTrace(logEntry, capturer.StatusCode(), reqXMLRaw, respXMLRaw)
		}()
	}

	// 记录请求信息（Info 级别，生产环境可见）
	log.Info("ACS received request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("body_len", len(body)),
	)

	// 从请求体检测 RPC 方法
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		h.handleEmpty(w, r, log)
		return
	}

	method := soap.DetectRPCMethod(body)
	log.Info("ACS detected RPC method", zap.String("method", string(method)))

	// 优先检查 SOAP Fault（可能出现在任何 RPC 的响应中）
	if isFault, faultCode, soapFaultCode, faultMsg, badPath := detectSOAPFault(body); isFault {
		h.handleSOAPFault(w, r, body, faultCode, soapFaultCode, faultMsg, badPath, log)
		return
	}

	switch method {
	case soap.MethodInform:
		h.handleInform(w, r, body, log)
	case soap.MethodTransferComplete:
		h.handleTransferComplete(w, r, body, log)
	case soap.MethodAutonomousTransferComplete:
		h.handleAutonomousTransferComplete(w, r, body, log)
	case soap.MethodGetParameterValuesResp,
		soap.MethodSetParameterValuesResp,
		soap.MethodGetParameterNamesResp,
		soap.MethodGetParameterAttributesResp,
		soap.MethodSetParameterAttributesResp,
		soap.MethodAddObjectResp,
		soap.MethodDeleteObjectResp,
		soap.MethodDownloadResp,
		soap.MethodUploadResp,
		soap.MethodRebootResp,
		soap.MethodFactoryResetResp,
		soap.MethodBaicellsPasswordResetResp,
		soap.MethodCommonPasswordResetResp:
		h.handleRPCResponse(w, r, body, method, log)
	default:
		log.Warn("unknown SOAP method", zap.String("method", string(method)))
		http.Error(w, "Unknown method", http.StatusBadRequest)
	}
}

// handleInform 处理 CPE 设备发送的 Inform 消息。
// 根据 TR069 规范：Inform → InformResponse（必须）。RPC 下发在后续的空 POST 中通过 handleEmpty() 处理。
func (h *Handler) handleInform(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
	ctx, span := tracing.StartSpan(r.Context(), tracing.ACSTracerName, "ACS HandleInform",
		attribute.String("acs.remote_addr", r.RemoteAddr),
		attribute.Int("acs.body_len", len(body)),
	)
	defer span.End()
	r = r.WithContext(ctx)

	// 记录完整的请求 XML
	log.Debug("ACS received Inform request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.Int("body_len", len(body)),
		zap.String("xml", string(body)),
	)

	// 解析 Inform
	inform, cwmpID, err := soap.DecodeInform(bytes.NewReader(body))
	if err != nil {
		tracing.RecordError(span, err)
		log.Error("decode Inform", zap.Error(err),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("xml", string(body)),
		)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	deviceSN := inform.DeviceId.SerialNumber
	span.SetAttributes(
		attribute.String("acs.device_sn", deviceSN),
		attribute.String("acs.oui", inform.DeviceId.OUI),
		attribute.String("acs.product_class", inform.DeviceId.ProductClass),
		attribute.String("acs.cwmp_id", cwmpID),
	)

	// 报文跟踪被动抓（issue #186）：一旦解析出 SN 就立刻把 SN/方法/CwmpID 写入
	// protocolLogger 的 LogEntry，使 ServeHTTP 中的 maybeCaptureTrace 旁路在 SN 命中
	// 跟踪白名单时一定能落库——即使本次周期 Inform 随后被限流（377）或准入拒绝（402）
	// 提前返回、根本没建立会话。这把"是否抓到报文"从"主动呼叫/会话是否建成"解耦：
	// NAT 后设备网管呼不到，但只要设备自己周期上报 Inform，网管收到即抓。
	// 注意：下游成功路径在 line ~494 会再次写同样的字段（带 SessionID/Sequence），是幂等覆盖。
	if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
		entry.DeviceSN = deviceSN
		entry.Method = "Inform"
		entry.CwmpID = cwmpID
	}

	// 记录解析后的 Inform 详情
	log.Info("ACS parsed Inform",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("device_sn", deviceSN),
		zap.String("oui", inform.DeviceId.OUI),
		zap.String("product_class", inform.DeviceId.ProductClass),
		zap.String("manufacturer", inform.DeviceId.Manufacturer),
		zap.String("cwmp_id", cwmpID),
	)

	// 速率限制
	if !h.rateLimiter.Allow(deviceSN) {
		h.metrics.RateLimitRejected.Inc()
		log.Warn("rate limited", zap.String("device_sn", deviceSN))
		http.Error(w, "Too Many Requests", http.StatusServiceUnavailable)
		return
	}

	// 预生成新会话 ID —— 准入槽位以 sessionID 为成员（issue #65 Option B），
	// 使 Acquire/Release 跨实例严格配对。
	sessionID := generateSessionID()

	// 准入控制 —— 全局 Redis 槽位，持有直到会话完成（completeSession）或 TTL 自愈回收。
	// 在孤儿清理之前申请：若全局已满直接拒绝，不动设备会话指针，避免误清理后又被拒。
	if !h.admission.Acquire(r.Context(), sessionID) {
		h.metrics.AdmissionRejected.Inc()
		log.Warn("admission denied", zap.String("device_sn", deviceSN))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// 跨实例孤儿会话清理：把设备当前会话指针原子换成新 sessionID，拿回被顶替的旧 sessionID。
	// 当 CPE 在前一个会话仍挂起时发送新 Inform（CPE 未响应 RPC、重启或周期上报定时器触发），
	// 旧会话永不完成。新 Inform 落在任一实例都能读到旧 sessionID 并跨实例清理，以：
	// 1) 释放旧的准入槽位（防止槽位泄漏）2) 触发 postSessionWake 处理剩余命令 3) 指标准确。
	if h.deviceSessionStore != nil {
		oldSessionID, swapErr := h.deviceSessionStore.Swap(r.Context(), deviceSN, sessionID)
		if swapErr != nil {
			log.Error("device session swap failed; rejecting Inform",
				zap.String("device_sn", deviceSN), zap.Error(swapErr))
			h.admission.Release(r.Context(), sessionID)
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		} else if oldSessionID != "" {
			log.Info("cleaning orphaned session before new Inform",
				zap.String("device_sn", deviceSN),
				zap.String("old_session_id", oldSessionID),
				zap.String("new_session_id", sessionID))
			h.reapOrphanedSession(deviceSN, oldSessionID, "new_inform")
		}
	}

	// 只有新 Inform 已通过准入且设备会话指针交换成功后，才能恢复上一会话遗留的
	// sent 任务。失败/被拒绝的 Inform 不得提前删除旧 CWMP 映射或重置任务状态。
	// 新 Inform 表示上一 CWMP 会话已被替代，因此恢复同设备全部 sent 任务；失败
	// 不阻塞 InformResponse，仅告警留痕。
	if err := h.taskService.RecoverPendingTasks(r.Context(), deviceSN); err != nil {
		log.Warn("recover pending tasks failed (non-blocking)",
			zap.String("device_sn", deviceSN),
			zap.Error(err))
	}

	// 跟踪本进程活跃会话 —— 将由 completeSession() 或清理器配对递减。
	h.trackActiveSession(sessionID)

	// 记录指标
	eventCodes := tr069.EventCodes(inform.Event)
	for _, code := range eventCodes {
		h.metrics.InformTotal.WithLabelValues(code).Inc()
	}

	log.Info("ACS Inform processing",
		zap.String("device_sn", deviceSN),
		zap.String("oui", inform.DeviceId.OUI),
		zap.String("product_class", inform.DeviceId.ProductClass),
		zap.Strings("events", eventCodes),
		zap.Int("param_count", len(inform.ParameterList)),
	)

	// 缓存 Inform 中的 Connection Request 地址，用于会话后续唤。
	for _, p := range inform.ParameterList {
		if strings.HasSuffix(p.Name, ".UDPConnectionRequestAddress") && p.Value != "" && !netutil.IsUnspecifiedUDPAddress(p.Value) {
			if h.stunStore != nil {
				if err := h.stunStore.SetFromInform(ctx, deviceSN, p.Value); err != nil {
					log.Warn("cache STUN address from Inform",
						zap.String("device_sn", deviceSN),
						zap.String("udp_addr", p.Value),
						zap.Error(err))
				} else {
					log.Debug("cached STUN address from Inform",
						zap.String("device_sn", deviceSN),
						zap.String("udp_addr", p.Value))
				}
			}
		}
		if strings.HasSuffix(p.Name, ".ConnectionRequestURL") && p.Value != "" {
			if h.connReqURLStore != nil {
				if err := h.connReqURLStore.Set(ctx, deviceSN, p.Value); err != nil {
					log.Warn("cache ConnectionRequestURL from Inform",
						zap.String("device_sn", deviceSN),
						zap.String("cr_url", p.Value),
						zap.Error(err))
				} else {
					log.Debug("cached ConnectionRequestURL from Inform",
						zap.String("device_sn", deviceSN),
						zap.String("cr_url", p.Value))
				}
			}
		}
	}

	// 使用上面预生成的 Session ID 创建会话（准入槽位与设备会话指针已绑定该 ID）。
	session := &Session{
		ID:           sessionID,
		DeviceSN:     deviceSN,
		State:        StateInformReceived,
		InstanceID:   r.RemoteAddr,
		StartedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		InformEvents: eventCodes,
		CWMPId:       cwmpID,
	}

	// 按 Session ID 存储会话（基于 Cookie 的查找）
	if err := h.sessionStore.CreateWithID(r.Context(), sessionID, session); err != nil {
		log.Error("create session by id", zap.Error(err), zap.String("device_sn", deviceSN))
	}
	// 设备 → 会话指针已由上面的 deviceSessionStore.Swap 设置（用于跨实例孤儿会话检测）。

	// 协议日志元数据
	if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
		entry.SessionID = sessionID
		entry.DeviceSN = deviceSN
		entry.Method = "Inform"
		entry.CwmpID = cwmpID
		entry.Sequence = 0
	}

	// 为该设备注入随机测试任务（测试功能）
	// 跳过 TransferComplete/AutonomousTransferComplete 会话 ——
	// 这些会话有特定用途，不应被测试任务污染。
	isTC := tr069.HasEvent(inform.Event, tr069.EventTransferComplete)
	isATC := tr069.IsAutonomousTransferComplete(inform.Event)
	log.Info("ACS task injection check",
		zap.String("device_sn", deviceSN),
		zap.Bool("test_task_injection_enabled", h.enableTestTaskInjection),
		zap.Bool("task_service_available", h.taskService != nil),
		zap.Bool("is_tc", isTC),
		zap.Bool("is_atc", isATC),
	)
	if !isTC && !isATC {
		h.injectRandomTestTasks(r, deviceSN, log)
	} else {
		log.Debug("skipped test task injection for TC/ATC session",
			zap.String("device_sn", deviceSN),
			zap.Strings("events", eventCodes))
	}

	// 发布事件
	h.publishInformEvents(r.Context(), inform, eventCodes, log)

	// #746: 心跳周期自动调整 — BOOTSTRAP/BOOT 事件时入队 GPV 查询当前心跳周期。
	// GPV 响应后由 handleRPCResponse 中的 processInformPeriodGPV 比较并决定是否 SPV。
	if h.informPeriodPolicy.ShouldTrigger(eventCodes) {
		productClass := inform.DeviceId.ProductClass
		if h.durableReadbackEnabled && h.informPeriodPolicy.ShouldProbe(r.Context(), productClass) &&
			h.requestDurableReadback(r.Context(), deviceSN, "inform_period_probe",
				[]string{"Device.ManagementServer.PeriodicInformInterval"}, "inform-period:"+deviceSN+":"+sessionID, log) {
			// Durable request accepted by NATS; APP plans the GPV task.
		} else if err := h.informPeriodPolicy.EnqueueGPVTask(r.Context(), deviceSN, productClass); err != nil {
			log.Warn("enqueue inform period GPV task failed (non-blocking)",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
		}
	}

	// #220: 周期 Inform 会话中查询当前产品支持的 UE Count 参数。普通 GPV 回包
	// 复用既有 device_parameters 入库与 device_info 投影链路。
	if h.ueCountPolicy.ShouldTrigger(eventCodes) {
		if err := h.ueCountPolicy.Enqueue(r.Context(), deviceSN); err != nil {
			log.Warn("enqueue UE count GPV task failed (non-blocking)",
				zap.String("device_sn", deviceSN),
				zap.Error(err))
		}
	}

	// 重置连续唤醒计数器 —— 设备已连接，允许新的唤醒周期。
	h.resetContinuousWake(r.Context(), deviceSN)

	// 根据 TR069 规范：必须先发送 InformResponse。
	// 命令队列将在后续的空 POST 中检查。
	log.Info("ACS Inform done, sending InformResponse",
		zap.String("device_sn", deviceSN),
		zap.String("cwmp_id", cwmpID),
		zap.String("session_id", sessionID))

	// 在响应头中设置 Session Cookie
	h.setSessionCookie(w, sessionID)

	h.sendInformResponse(w, cwmpID, log)
}

// handleEmpty 处理 CPE 发送的空 POST。
// 根据 TR069 规范，InformResponse 之后 CPE 发送空 POST。
// ACS 应发送 RPC 请求或空响应来关闭会话。
func (h *Handler) handleEmpty(w http.ResponseWriter, r *http.Request, log *zap.Logger) {
	ctx, span := tracing.StartSpan(r.Context(), tracing.ACSTracerName, "ACS HandleEmpty")
	defer span.End()
	r = r.WithContext(ctx)

	// 从 Cookie 查找会话
	session, sessionID := h.getSessionFromCookie(r, log)
	if session == nil {
		// 无有效会话 —— 直接关闭。
		log.Warn("empty POST without valid session cookie", zap.String("remote_addr", r.RemoteAddr))
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deviceSN := session.DeviceSN
	span.SetAttributes(
		attribute.String("acs.device_sn", deviceSN),
		attribute.String("acs.session_state", string(session.State)),
	)

	log.Info("ACS HandleEmpty started",
		zap.String("device_sn", deviceSN),
		zap.String("session_id", sessionID),
		zap.String("session_state", string(session.State)),
	)

	// 协议日志元数据
	if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
		entry.SessionID = sessionID
		entry.DeviceSN = deviceSN
		entry.Sequence = session.RPCCount
		entry.Method = "Empty"
	}

	// 状态转换：InformReceived → Processing
	if session.State == StateInformReceived {
		session.State = StateProcessing
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)
	}

	// 在下发更多命令前检查单会话 RPC 上限。
	// 防止超出 CPE 的单会话交互限制。
	if h.sessionRPCLimitReached(session) {
		log.Info("ACS session RPC limit reached, completing session",
			zap.String("device_sn", deviceSN),
			zap.String("session_id", sessionID),
			zap.Int("rpc_count", session.RPCCount),
			zap.Int("max_rpc_per_session", h.maxRPCPerSession),
		)
		// 完成会话；会话后续唤将为队列中剩余命令触发新会话。
		h.completeSession(r.Context(), session)
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 尝试从统一任务队列获取下一个仍满足 PG pending fence 的任务。
	taskItem, cwmpID, err := h.popAndMarkNextTask(r.Context(), deviceSN, log)
	if err != nil {
		log.Error("pop sendable task from queue", zap.Error(err))
	} else if taskItem != nil {
		originalParams := append(json.RawMessage(nil), taskItem.Params...)
		// standardPath → privatePath 翻译（T-XXX：翻译职责从 App fanout 迁移到 ACS）
		h.translateTaskParamsInPlace(r.Context(), taskItem, log)

		// 更新会话状态
		session.State = StateRPCPending
		session.LastRPC = taskItem.Method
		session.LastCommandParams = originalParams
		session.LastTaskID = taskItem.ID
		session.LastTaskCWMPID = cwmpID
		session.RPCCount++
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)

		// 使用 CWMP ID 构建 RPC 请求
		cmd := &rpc.Command{
			ID:         taskItem.ID,
			DeviceSN:   deviceSN,
			Method:     taskItem.Method,
			Params:     taskItem.Params,
			CommandKey: taskItem.CommandKey,
		}
		respData, err := h.rpcDispatcher.BuildRequest(cmd, cwmpID)
		if err != nil {
			log.Error("build RPC request from task", zap.Error(err))
			h.metrics.RPCErrorsTotal.WithLabelValues(taskItem.Method).Inc()
			// 标记任务失败
			h.taskService.MarkTaskFailed(r.Context(), taskItem.ID, 0, err.Error())
		} else {
			log.Info("ACS sending RPC request from task",
				zap.String("device_sn", deviceSN),
				zap.String("method", taskItem.Method),
				zap.String("task_id", taskItem.ID),
				zap.String("cwmp_id", cwmpID),
				zap.Int("soap_size", len(respData)),
				zap.String("soap_body", string(respData)),
				zap.String("command_key", taskItem.CommandKey),
			)
			h.logOutgoingTransferRPC(log, deviceSN, taskItem.Method, cwmpID, taskItem.CommandKey, taskItem.Params, respData)
			if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
				entry.Method = taskItem.Method
				entry.TaskID = taskItem.ID
				entry.CwmpID = cwmpID
			}
			// 在响应中设置 Session Cookie
			h.setSessionCookie(w, sessionID)
			h.sendSOAPResponse(w, respData, log)
			return
		}
	} else {
		log.Info("ACS TaskService.PopTask returned nil",
			zap.String("device_sn", deviceSN),
			zap.String("reason", "no pending tasks in queue"))
	}

	// 没有更多命令 —— 完成会话。
	log.Info("ACS HandleEmpty no tasks found, completing session",
		zap.String("device_sn", deviceSN),
		zap.String("session_id", sessionID),
		zap.String("session_state", string(session.State)),
	)
	h.completeSession(r.Context(), session)

	// 发送空响应表示会话结束（根据 TR069 规范无 body）。
	// Connection: close 告知 CPE 关闭 TCP 连接。
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleRPCResponse(w http.ResponseWriter, r *http.Request, body []byte, method soap.RPCMethod, log *zap.Logger) {
	ctx, span := tracing.StartSpan(r.Context(), tracing.ACSTracerName, "ACS HandleRPCResponse",
		attribute.String("acs.rpc_method", string(method)),
	)
	defer span.End()
	r = r.WithContext(ctx)

	// 从 SOAP 响应中提取 CWMP ID。
	_, cwmpID, _, _ := soap.DetectMethod(bytes.NewReader(body))
	span.SetAttributes(attribute.String("acs.cwmp_id", cwmpID))

	// 记录完整的响应 XML
	log.Debug("ACS received RPC response",
		zap.String("method", string(method)),
		zap.String("cwmp_id", cwmpID),
		zap.String("xml", string(body)),
	)

	log.Info("RPC response received",
		zap.String("method", string(method)),
		zap.String("cwmp_id", cwmpID),
	)

	// 从 Cookie 查找会话
	session, sessionID := h.getSessionFromCookie(r, log)
	if session == nil {
		log.Warn("no valid session cookie for RPC response", zap.String("remote_addr", r.RemoteAddr))
		w.WriteHeader(http.StatusNoContent)
		return
	}

	deviceSN := session.DeviceSN

	// 协议日志元数据
	if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
		entry.SessionID = sessionID
		entry.DeviceSN = deviceSN
		entry.Method = string(method)
		entry.CwmpID = cwmpID
		entry.Sequence = session.RPCCount
	}

	// 记录 RPC 耗时（近似值：自上次状态更新以来的时间）。
	rpcDuration := time.Since(session.UpdatedAt).Seconds()
	h.metrics.RPCDuration.WithLabelValues(string(method)).Observe(rpcDuration)

	session.State = StateRPCResponse
	session.UpdatedAt = time.Now()
	h.sessionStore.UpdateByID(r.Context(), sessionID, session)

	// 检查是否为基于任务的 RPC（新任务队列系统）
	//   优先用 SOAP Header cwmp:ID 反查（spec 标准路径：response.cwmp:ID == request.cwmp:ID）。
	//   fallback 用 session.LastTaskID：实测 BAIBLQ 等厂商在 AddObjectResponse 中
	//     回填空 cwmp:ID（违反 TR-069 §3.4.1.5 EchoBack 约束），cwmp_id 反查失败导致
	//     task 永远停在 sent，前端 waitForTaskTerminal 60s 超时。
	//     TR-069 一个 session 内 RPC 严格串行，session.LastTaskID 必定就是这次响应所对应的 task。
	var taskItem *task.Task
	if cwmpID != "" {
		var err error
		taskItem, err = h.taskService.GetTaskByCWMPID(r.Context(), cwmpID)
		if err != nil {
			log.Warn("get task by cwmp_id", zap.Error(err), zap.String("cwmp_id", cwmpID))
		}
		if taskItem != nil && !rpcResponseMatchesTask(method, taskItem.Method) {
			log.Warn("RPC response cwmp_id matched task with different method; falling back to session.last_task_id",
				zap.String("device_sn", deviceSN),
				zap.String("response_method", string(method)),
				zap.String("task_method", taskItem.Method),
				zap.String("cpe_cwmp_id", cwmpID),
				zap.String("acs_cwmp_id", session.LastTaskCWMPID),
				zap.String("matched_task_id", taskItem.ID),
				zap.String("last_task_id", session.LastTaskID))
			taskItem = nil
		}
	}
	if taskItem == nil && session.LastTaskID != "" {
		t, err := h.taskService.GetTask(r.Context(), session.LastTaskID)
		if err != nil {
			log.Warn("get task by session.last_task_id (cwmp_id missing/mismatch)",
				zap.Error(err),
				zap.String("cpe_cwmp_id", cwmpID),
				zap.String("acs_cwmp_id", session.LastTaskCWMPID),
				zap.String("last_task_id", session.LastTaskID))
		} else if t != nil && !rpcResponseMatchesTask(method, t.Method) {
			log.Warn("RPC response session.last_task_id has different method; leave response unassociated",
				zap.String("device_sn", deviceSN),
				zap.String("response_method", string(method)),
				zap.String("task_method", t.Method),
				zap.String("cpe_cwmp_id", cwmpID),
				zap.String("acs_cwmp_id", session.LastTaskCWMPID),
				zap.String("last_task_id", session.LastTaskID))
		} else if t != nil {
			taskItem = t
			log.Warn("RPC response associated via session.last_task_id (CPE returned empty/mismatched cwmp:ID)",
				zap.String("device_sn", deviceSN),
				zap.String("method", string(method)),
				zap.String("cpe_cwmp_id", cwmpID),
				zap.String("acs_cwmp_id", session.LastTaskCWMPID),
				zap.String("task_id", t.ID))
		}
	}
	if taskItem != nil {
		// 检查响应中是否有 SOAP Fault
		if isFault, faultCode, soapFaultCode, faultMsg, badPath := detectSOAPFault(body); isFault {
			// 把 SOAP 1.1 outer faultcode（如 "Server.Internal"）合进 error message，
			// 否则丢失诊断信息；DB 落盘的 error_message 也能完整呈现。
			combinedMsg := faultMsg
			if soapFaultCode != "" && !strings.Contains(faultMsg, soapFaultCode) {
				combinedMsg = fmt.Sprintf("[%s] %s", soapFaultCode, faultMsg)
			}
			// 参数同步 GPV 自愈：剔除坏 path 后续查，命中即跳过 MarkTaskFailed
			if !h.tryRecoverGPVFault(r.Context(), taskItem, badPath, faultCode, log) {
				if markErr := h.taskService.MarkTaskFailed(r.Context(), taskItem.ID, faultCode, combinedMsg); markErr != nil {
					log.Error("mark task failed", zap.Error(markErr), zap.String("task_id", taskItem.ID))
				}
				log.Warn("task failed with SOAP fault",
					zap.String("task_id", taskItem.ID),
					zap.Int("fault_code", faultCode),
					zap.String("soap_fault_code", soapFaultCode),
					zap.String("fault_msg", faultMsg))
			}
		} else {
			// 任务成功完成 - 将原始响应存为结果
			resultMap := map[string]interface{}{
				"method":       string(method),
				"raw_response": string(body),
			}
			// issue #424：GPV 响应里的参数名是私有 path，回译为标准 path 一并入库
			// （raw_response 保留原文供 XmlViewer 调试）。MML 控制台据此按「执行的标准
			// path」匹配结果，不再因私有/标准不一致而显示为空。
			if method == soap.MethodGetParameterValuesResp {
				if pvs, _, decErr := soap.DecodeGetParameterValuesResponse(bytes.NewReader(body)); decErr == nil && len(pvs) > 0 {
					names := make([]string, len(pvs))
					for i, pv := range pvs {
						names[i] = pv.Name
					}
					stdNames, _ := h.pathTranslator.TranslateResponseNamesForRequest(r.Context(), taskItem.DeviceSN, names, session.LastCommandParams)
					std := make([]tr069.ParameterValueStruct, len(pvs))
					for i, pv := range pvs {
						std[i] = tr069.ParameterValueStruct{Name: stdNames[i], Value: pv.Value, Type: pv.Type}
					}
					resultMap["standard_parameter_values"] = std
					// Durable parameter-sync runs project against the mapping snapshot
					// frozen at planning time. Preserve private names so a registry
					// refresh during the run cannot change result interpretation.
					resultMap["private_parameter_values"] = pvs
				}
			}
			// AddObject 提前解析 InstanceNumber 写入 result,供 notification 渲染
			// 标题"InterFreq.Carrier.6"等场景使用,避免下游再解一次 SOAP body。
			if method == soap.MethodAddObjectResp {
				if instanceNumber, _, _, decErr := soap.DecodeAddObjectResponse(bytes.NewReader(body)); decErr == nil && instanceNumber > 0 {
					resultMap["instance_number"] = instanceNumber
				}
			}
			resultJSON, _ := json.Marshal(resultMap)
			if markErr := h.taskService.MarkTaskCompleted(r.Context(), taskItem.ID, resultJSON); markErr != nil {
				log.Error("mark task completed", zap.Error(markErr), zap.String("task_id", taskItem.ID))
			}
			log.Info("task completed", zap.String("task_id", taskItem.ID), zap.String("method", taskItem.Method))

			// T-0147:SetParameterValues 成功后,自动入队 GPV 回读改过的 path。
			// 原因:SetParameterValuesResponse 仅含 Status(无 path/value),CPE 端真已应用但
			// OMC device_parameters.current_value 未更新;期望 Inform 自动同步但
			// BaiBLQ inform_interval=300s + Inform 不全量上报 → UI 长时间看到旧值。
			// 回读由已有 RPCResponseSubscriber.handleGPVResponse 写库,本 hook 仅入队。
			if method == soap.MethodSetParameterValuesResp {
				h.queueAutoGPVAfterSPV(r.Context(), taskItem, log)
			}
			// AddObject 成功后,自动入队 GPV 回读新实例参数。
			// 原因:AddObjectResponse 仅含 InstanceNumber(无参数值),CPE 上对象已建但
			// device_parameters 表无新实例记录,导致"快速设置"等列表看不到新增项,
			// 用户须手动点"同步参数"全量 Path B 才能恢复。本 hook 针对新实例 path
			// 入队 GPV,由已有 handleGPVResponse 写库。
			if method == soap.MethodAddObjectResp {
				h.queueAutoGPVAfterAddObject(r.Context(), taskItem, body, log)
			}
			// #746: 心跳周期 GPV 响应处理 — 比较当前值与配置目标值，不一致则入队 SPV。
			if method == soap.MethodGetParameterValuesResp {
				h.processInformPeriodGPV(r.Context(), taskItem, body, log)
			}
		}
	}

	// 发布 RPC 响应事件到开通引擎（包含原始 body 用于 GPN/GPV 处理）。
	eventParams := session.LastCommandParams
	if taskItem != nil && len(taskItem.Params) > 0 {
		eventParams = taskItem.Params
	}
	h.publishRPCResponseEvent(r.Context(), deviceSN, method, body, eventParams, taskItem, log)
	// 在下发下一条命令前检查单会话 RPC 上限。
	if h.sessionRPCLimitReached(session) {
		log.Info("ACS session RPC limit reached after response, completing session",
			zap.String("device_sn", deviceSN),
			zap.String("session_id", sessionID),
			zap.Int("rpc_count", session.RPCCount),
			zap.Int("max_rpc_per_session", h.maxRPCPerSession),
		)
		h.completeSession(r.Context(), session)
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 尝试从统一任务队列获取下一个仍满足 PG pending fence 的任务。
	nextTask, newCWMPID, err := h.popAndMarkNextTask(r.Context(), deviceSN, log)
	if err != nil {
		log.Error("pop sendable task from queue", zap.Error(err))
	} else if nextTask != nil {
		h.translateTaskParamsInPlace(r.Context(), nextTask, log)

		session.State = StateRPCPending
		session.LastRPC = nextTask.Method
		session.LastCommandParams = nextTask.Params
		session.LastTaskID = nextTask.ID
		session.LastTaskCWMPID = newCWMPID
		session.RPCCount++
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)

		cmd := &rpc.Command{
			ID:         nextTask.ID,
			DeviceSN:   deviceSN,
			Method:     nextTask.Method,
			Params:     nextTask.Params,
			CommandKey: nextTask.CommandKey,
		}
		respData, err := h.rpcDispatcher.BuildRequest(cmd, newCWMPID)
		if err != nil {
			log.Error("build next RPC request from task", zap.Error(err))
			h.metrics.RPCErrorsTotal.WithLabelValues(nextTask.Method).Inc()
			h.taskService.MarkTaskFailed(r.Context(), nextTask.ID, 0, err.Error())
		} else {
			log.Info("ACS sending RPC request from task",
				zap.String("device_sn", deviceSN),
				zap.String("method", nextTask.Method),
				zap.String("task_id", nextTask.ID),
				zap.String("cwmp_id", newCWMPID),
				zap.Int("soap_size", len(respData)),
				zap.String("soap_body", string(respData)),
				zap.String("trigger", "after_rpc_response"),
				zap.String("command_key", nextTask.CommandKey),
			)
			h.logOutgoingTransferRPC(log, deviceSN, nextTask.Method, newCWMPID, nextTask.CommandKey, nextTask.Params, respData)
			h.setSessionCookie(w, sessionID)
			h.sendSOAPResponse(w, respData, log)
			return
		}
	}

	// 没有更多命令 —— 完成会话。
	log.Info("ACS RPC loop done, completing session",
		zap.String("device_sn", deviceSN),
		zap.String("session_id", sessionID),
		zap.String("session_state", string(session.State)),
	)
	h.completeSession(r.Context(), session)

	// 发送空响应表示会话结束（根据 TR069 规范无 body）。
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusNoContent)
}

// popAndMarkNextTask returns the next task that still owns the authoritative
// PostgreSQL pending-state fence. Redis can temporarily contain terminal task
// copies after recovery or cancellation. MarkTaskSent removes those copies and
// returns ErrTaskNotPending; skipping them here preserves the current CWMP
// session so later valid tasks can be sent without waiting for another Inform.
func (h *Handler) popAndMarkNextTask(ctx context.Context, deviceSN string, log *zap.Logger) (*task.Task, string, error) {
	for skipped := 0; skipped < maxStaleTaskSkipsPerDispatch; skipped++ {
		nextTask, err := h.taskService.PopTask(ctx, deviceSN)
		if err != nil || nextTask == nil {
			return nextTask, "", err
		}

		cwmpID := task.GenerateCWMPID(nextTask.Method)
		if err := h.taskService.MarkTaskSent(ctx, nextTask.ID, cwmpID); err != nil {
			if errors.Is(err, task.ErrTaskNotPending) {
				log.Warn("skipping stale terminal task in execution queue",
					zap.String("device_sn", deviceSN),
					zap.String("task_id", nextTask.ID),
					zap.Error(err))
				continue
			}
			return nil, "", fmt.Errorf("mark task %s sent: %w", nextTask.ID, err)
		}
		return nextTask, cwmpID, nil
	}
	return nil, "", fmt.Errorf("skip stale tasks for device %s: limit %d reached", deviceSN, maxStaleTaskSkipsPerDispatch)
}

// completeSession 完成 TR069 会话，释放所有相关资源。
// 根据 Session.ID 删除 Redis 中的会话数据。
func (h *Handler) completeSession(ctx context.Context, session *Session) {
	if session == nil {
		// 无会话上下文 → 无 sessionID，准入槽位无法配对释放（由 TTL 自愈回收）。
		return
	}
	h.untrackActiveSession(session.ID)

	// 释放该 sessionID 的准入槽位（issue #65 Option B：槽位以 sessionID 为成员，配对释放）。
	h.admission.Release(ctx, session.ID)

	// 记录会话时长指标
	duration := time.Since(session.StartedAt).Seconds()
	h.metrics.SessionDuration.Observe(duration)

	// 更新会话状态为 COMPLETE
	session.State = StateComplete
	session.UpdatedAt = time.Now()

	// 删除 Cookie-based Session
	if session.ID != "" {
		h.sessionStore.DeleteByID(ctx, session.ID)
	}

	// 清除设备→会话指针。CAS 删除：仅当共享存储中的指针仍等于当前 session.ID 时才删除，
	// 避免误删已被新 Inform 覆盖的指针（issue #65 Option B）。
	if h.deviceSessionStore != nil && session.DeviceSN != "" && session.ID != "" {
		if err := h.deviceSessionStore.CompareAndDelete(ctx, session.DeviceSN, session.ID); err != nil {
			h.logger.Warn("clear device session pointer failed (non-blocking)",
				zap.String("device_sn", session.DeviceSN),
				zap.String("session_id", session.ID),
				zap.Error(err))
		}
	}

	// 异步检查队列并续唤设备
	if h.connReqSender != nil && h.postSessionWakeCfg.Enabled && session.DeviceSN != "" {
		go h.postSessionWake(session.DeviceSN)
	}
}

func (h *Handler) trackActiveSession(sessionID string) {
	h.trackActiveSessionAt(sessionID, time.Now())
}

func (h *Handler) trackActiveSessionAt(sessionID string, trackedAt time.Time) {
	if h == nil || h.metrics == nil || sessionID == "" {
		return
	}
	if _, loaded := h.localActiveSessions.LoadOrStore(sessionID, trackedAt); !loaded {
		h.metrics.LocalTrackedSessions.Inc()
	}
}

// reapLocalActiveSessions bounds the process-local tracking set. A session can
// be completed by the other ACS instance through the shared Redis store; that
// instance cannot delete this process's sync.Map entry or decrement its gauge.
// The shared session/admission TTL is maxAge, so entries older than it are no
// longer active even when cross-instance cleanup prevented a local callback.
func (h *Handler) reapLocalActiveSessions(now time.Time, maxAge time.Duration) {
	if h == nil || maxAge <= 0 {
		return
	}
	h.localActiveSessions.Range(func(key, value any) bool {
		trackedAt, ok := value.(time.Time)
		if !ok || now.Sub(trackedAt) > maxAge {
			sessionID, _ := key.(string)
			h.untrackActiveSession(sessionID)
		}
		return true
	})
}

func (h *Handler) untrackActiveSession(sessionID string) {
	if h == nil || h.metrics == nil || sessionID == "" {
		return
	}
	if _, loaded := h.localActiveSessions.LoadAndDelete(sessionID); loaded {
		h.metrics.LocalTrackedSessions.Dec()
	}
}

// postSessionWake 检查会话结束后设备是否仍有待执行命令。
// 如果有，发送 Connection Request 立即触发新 Inform，
// 而非等待设备的下一次周期上报（约 60 秒以上）。
// 这大幅加速了命令队列的消耗（如参数发现）。
func (h *Handler) postSessionWake(deviceSN string) {
	// 捕获 panic，防止 goroutine 静默退出。
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("post-session wake: panic recovered",
				zap.String("device_sn", deviceSN),
				zap.Any("panic", r))
		}
	}()

	h.logger.Info("post-session wake: goroutine started",
		zap.String("device_sn", deviceSN),
		zap.Bool("has_connReqSender", h.connReqSender != nil),
		zap.Bool("enabled", h.postSessionWakeCfg.Enabled),
		zap.Bool("has_redisClient", h.redisClient != nil))

	if h.connReqSender == nil || !h.postSessionWakeCfg.Enabled {
		return
	}

	ctx := context.Background()

	// 先短暂延迟：等待 CPE 关闭上一个 TCP 连接，并让开通引擎（EventBus 订阅者）
	// 完成将 RPC 响应事件中的新命令添加到队列。若无此延迟，队列可能看起来为空，
	// 因为异步订阅者尚未处理完毕。
	delay := h.postSessionWakeCfg.DelayAfter
	if delay <= 0 {
		delay = time.Second
	}
	time.Sleep(delay)

	// 延迟后检查任务队列深度。
	var remaining int64
	if n, err := h.taskService.GetQueueLength(ctx, deviceSN); err == nil {
		remaining += n
	}
	h.logger.Debug("post-session wake: task queue check",
		zap.String("device_sn", deviceSN),
		zap.Int64("task_remaining", remaining))

	if remaining == 0 {
		h.logger.Info("post-session wake: queue empty, skip",
			zap.String("device_sn", deviceSN))
		return // 队列已空，无需续唤
	}

	// 检查连续唤醒计数器（防止无限循环）。
	maxContinuous := h.postSessionWakeCfg.MaxContinuous
	if maxContinuous <= 0 {
		maxContinuous = 200
	}
	cooldownTTL := h.postSessionWakeCfg.CooldownTTL
	if cooldownTTL <= 0 {
		cooldownTTL = 5 * time.Minute
	}

	counterKey := redisx.Keys.ACSContinuousWake(deviceSN)
	count, err := h.redisClient.Incr(ctx, counterKey).Result()
	if err != nil {
		h.logger.Warn("post-session wake: incr counter failed",
			zap.String("device_sn", deviceSN), zap.Error(err))
		return
	}
	h.redisClient.Expire(ctx, counterKey, cooldownTTL)

	if int(count) > maxContinuous {
		h.logger.Warn("post-session wake: max continuous reached, backing off",
			zap.String("device_sn", deviceSN),
			zap.Int64("remaining", remaining),
			zap.Int64("continuous_count", count),
			zap.Int("max_continuous", maxContinuous))
		return
	}

	// 使用短超时，避免在不可达设备上阻塞。
	crCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 从共享存储读取设备 ConnectionRequestURL 作为 HTTP 回退（issue #65 Option B：
	// 取代进程内缓存，使非 Inform 实例上的续唤也能拿到 URL；纯 HTTP-CR 设备不再 miss）。
	var httpURL string
	if h.connReqURLStore != nil {
		if u, err := h.connReqURLStore.Get(crCtx, deviceSN); err != nil {
			h.logger.Warn("post-session wake: load connreq url failed",
				zap.String("device_sn", deviceSN), zap.Error(err))
		} else {
			httpURL = u
		}
	}

	if err := h.connReqSender.Send(crCtx, deviceSN, httpURL); err != nil {
		if checked := h.logger.Check(postSessionWakeFailureLogLevel(err), "post-session wake: CR failed"); checked != nil {
			checked.Write(
				zap.String("device_sn", deviceSN),
				zap.Int64("remaining", remaining),
				zap.Error(err),
			)
		}
	} else {
		h.logger.Info("post-session wake: CR sent",
			zap.String("device_sn", deviceSN),
			zap.Int64("remaining", remaining),
			zap.Int64("continuous_count", count))
		h.metrics.PostSessionWakeTotal.Inc()
	}
}

func postSessionWakeFailureLogLevel(err error) zapcore.Level {
	if errors.Is(err, connreq.ErrNoConnectionMethod) {
		return zap.DebugLevel
	}
	return zap.WarnLevel
}

// resetContinuousWake 在设备发送 Inform 时重置连续唤醒计数器。
// 设备重新连接时允许计数器重新开始。
func (h *Handler) resetContinuousWake(ctx context.Context, deviceSN string) {
	if h.redisClient == nil || !h.postSessionWakeCfg.Enabled {
		return
	}
	h.redisClient.Del(ctx, redisx.Keys.ACSContinuousWake(deviceSN))
}

// handleSOAPFault 处理 CPE 返回的 SOAP Fault 响应。
// 通过 CWMP ID 查找关联任务，先尝试参数同步 GPV 自愈（剔除坏 path 续查），未命中再标失败。
//
// 参数：
//   - faultCode     — 内层 cwmp:FaultCode 数值（标准 CWMP fault，如 9005=Invalid parameter name）
//   - soapFaultCode — 外层 soap:faultcode 文本（如 "Server.Internal"），部分厂商只回这个不带 cwmp:Fault
//   - faultMsg      — 人类可读 faultstring
//   - badPath       — 从 faultMsg 抽出的坏 path（CWMP 协议未规范，厂商私有约定），空表示未识别
func (h *Handler) handleSOAPFault(w http.ResponseWriter, r *http.Request, body []byte, faultCode int, soapFaultCode, faultMsg, badPath string, log *zap.Logger) {
	// 从 SOAP Header 中提取 CWMP ID
	_, cwmpID, _, _ := soap.DetectMethod(bytes.NewReader(body))

	log.Warn("ACS received SOAP Fault",
		zap.String("cwmp_id", cwmpID),
		zap.Int("fault_code", faultCode),
		zap.String("soap_fault_code", soapFaultCode),
		zap.String("fault_msg", faultMsg),
		zap.ByteString("raw_body", body),
	)

	// 合并消息：把 SOAP 1.1 outer faultcode 前缀到 faultMsg，让下游业务层看到完整信息。
	combinedFaultMsg := faultMsg
	if soapFaultCode != "" && !strings.Contains(faultMsg, soapFaultCode) {
		combinedFaultMsg = fmt.Sprintf("[%s] %s", soapFaultCode, faultMsg)
	}

	// 从 Cookie 查找会话以获取设备上下文
	session, sessionID := h.getSessionFromCookie(r, log)
	if session == nil {
		log.Warn("SOAP Fault without valid session cookie")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 协议日志元数据
	if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
		entry.SessionID = sessionID
		entry.DeviceSN = session.DeviceSN
		entry.Method = "SOAPFault"
		entry.CwmpID = cwmpID
		entry.Sequence = session.RPCCount
	}

	// 关联 Fault 到原 task：
	//   优先用 SOAP Header cwmp:ID 反查（标准路径，spec 要求 response.cwmp:ID == request.cwmp:ID）
	//   fallback 用 session.LastTaskID（CPE 自己生成新 cwmp:ID 主动 POST Fault 的厂商行为
	//     —— 实测 baicells/MMMM 系列就是这种"另起一个 request 携带 Fault"的玩法，
	//     cwmp:ID 不匹配，必须按 TR-069 session 上下文关联到刚派发出去的 RPC 才能落地失败原因）
	var taskItem *task.Task
	if cwmpID != "" {
		t, err := h.taskService.GetTaskByCWMPID(r.Context(), cwmpID)
		if err != nil {
			log.Warn("get task by cwmp_id for fault", zap.Error(err), zap.String("cwmp_id", cwmpID))
		} else {
			taskItem = t
		}
	}
	if taskItem == nil && session.LastTaskID != "" {
		t, err := h.taskService.GetTask(r.Context(), session.LastTaskID)
		if err != nil {
			log.Warn("get task by session.last_task_id for fault",
				zap.Error(err),
				zap.String("last_task_id", session.LastTaskID))
		} else if t != nil {
			taskItem = t
			log.Info("SOAP fault associated via session.last_task_id (cwmp_id mismatch)",
				zap.String("cpe_cwmp_id", cwmpID),
				zap.String("acs_cwmp_id", session.LastTaskCWMPID),
				zap.String("task_id", t.ID))
		}
	}
	if taskItem != nil {
		// 参数同步 GPV 自愈：剔除坏 path 后续查；命中即跳过 MarkTaskFailed + Fault 事件。
		if h.tryRecoverGPVFault(r.Context(), taskItem, badPath, faultCode, log) {
			// 自愈分支已标 task completed 并入队 retry batch，继续走 PopTask 推进队列。
		} else {
			// T-0174 / T-0180 — SPV / GPV 失败时把 per-parameter 详情提取出来,
			// schema 统一为 {"param_faults":[{parameter_name,fault_code,fault_string}]},
			// terminal 日志能定位真正出错的 path,且结构化 result 落到 device_tasks
			// 让下游(MML auto-learn / devsweep prober)按 paramModel 过滤掉该 path。
			//
			// SPV: CWMP 协议返多条 <SetParameterValuesFault>,extractSPVFaults 解出
			//      per-param 全集。
			// GPV: CWMP 协议不返结构化 per-param fault,仅在 FaultString 暴露 1 条
			//      badPath 作为 hint(已被 detectSOAPFault 抽到 badPath 变量),这里
			//      合成 1 个 SPVFault 写入,保持 schema 与 SPV 一致供下游统一消费。
			//      badPath 为空时不写 result(下游兜底走 ErrorMessage 文本)。
			finalMsg := combinedFaultMsg
			// 失败也保存 CPE 返回的原始 SOAP Fault 报文（raw_response），供前端「查看 → 结果报文」
			// 展示；param_faults 仍按方法解析出 per-param 详情。service.go 把 result.raw_response
			// 透传为 raw_output → 前端结果行 raw → XmlViewer。
			resultMap := map[string]interface{}{}
			if len(body) > 0 {
				resultMap["raw_response"] = string(body)
			}
			switch taskItem.Method {
			case "SetParameterValues":
				if spvFaults := extractSPVFaults(body); len(spvFaults) > 0 {
					finalMsg = enrichFaultMsgWithSPV(combinedFaultMsg, spvFaults)
					resultMap["param_faults"] = spvFaults
				}
			case "GetParameterValues":
				if badPath != "" && faultCode > 0 {
					resultMap["param_faults"] = []SPVFault{{
						ParameterName: badPath,
						FaultCode:     faultCode,
						FaultString:   faultMsg,
					}}
				}
			}
			var resultBytes json.RawMessage
			if len(resultMap) > 0 {
				if rb, mErr := json.Marshal(resultMap); mErr == nil {
					resultBytes = rb
				} else {
					log.Warn("marshal fault result", zap.Error(mErr))
				}
			}

			if markErr := h.taskService.MarkTaskFailedWithResult(r.Context(), taskItem.ID, faultCode, finalMsg, resultBytes); markErr != nil {
				log.Error("mark task failed on fault", zap.Error(markErr), zap.String("task_id", taskItem.ID))
			}
			log.Info("task marked as failed on SOAP fault",
				zap.String("task_id", taskItem.ID),
				zap.String("method", taskItem.Method),
				zap.Int("fault_code", faultCode),
				zap.String("soap_fault_code", soapFaultCode),
				zap.String("fault_msg", faultMsg),
				zap.Int("spv_fault_paths", len(resultBytes)))

			// 同步发出对应方法的 command.*.response 事件，把 fault_code / fault_string 透传给业务订阅者。
			// 否则像 software.HandleUploadResponse / HandleSetParamsResponseForCollect 这种按
			// "RPC 失败 → 立即 fail 子任务"的 handler 永远收不到 SOAP Fault，sub_task 卡 Uploading
			// 直到 reaper 兜底（半小时级延迟）。fault_msg 用 finalMsg —— SPV fault 时含
			// per-path 详情，让业务层能完整呈现「[Client] Invalid arguments — path faults: ...」。
			h.publishRPCFaultEvent(r.Context(), session.DeviceSN, taskItem, faultCode, soapFaultCode, finalMsg, log)
		}
	} else {
		log.Warn("SOAP fault but no task could be associated (neither cwmp_id nor session.last_task_id matched)",
			zap.String("cwmp_id", cwmpID),
			zap.String("device_sn", session.DeviceSN))
	}

	// A device may expose only one invalid path per 9005 response. Recovery can
	// therefore continue several replacements in this session; enforce the same
	// RPC ceiling as the normal response path before dispatching another one.
	if h.sessionRPCLimitReached(session) {
		log.Info("ACS session RPC limit reached after fault, completing session",
			zap.String("device_sn", session.DeviceSN),
			zap.String("session_id", sessionID),
			zap.Int("rpc_count", session.RPCCount),
			zap.Int("max_rpc_per_session", h.maxRPCPerSession),
		)
		h.completeSession(r.Context(), session)
		w.Header().Set("Connection", "close")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 检查队列中是否有更多仍满足 PG pending fence 的任务。
	deviceSN := session.DeviceSN
	nextTask, newCWMPID, err := h.popAndMarkNextTask(r.Context(), deviceSN, log)
	if err != nil {
		log.Error("pop next sendable task after fault", zap.Error(err))
	} else if nextTask != nil {
		h.translateTaskParamsInPlace(r.Context(), nextTask, log)

		session.State = StateRPCPending
		session.LastRPC = nextTask.Method
		session.LastCommandParams = nextTask.Params
		session.LastTaskID = nextTask.ID
		session.LastTaskCWMPID = newCWMPID
		session.RPCCount++
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)

		cmd := &rpc.Command{
			ID:         nextTask.ID,
			DeviceSN:   deviceSN,
			Method:     nextTask.Method,
			Params:     nextTask.Params,
			CommandKey: nextTask.CommandKey,
		}
		respData, err := h.rpcDispatcher.BuildRequest(cmd, newCWMPID)
		if err != nil {
			log.Error("build next RPC request after fault", zap.Error(err))
			h.metrics.RPCErrorsTotal.WithLabelValues(nextTask.Method).Inc()
			h.taskService.MarkTaskFailed(r.Context(), nextTask.ID, 0, err.Error())
		} else {
			log.Info("ACS sending RPC request from task",
				zap.String("device_sn", deviceSN),
				zap.String("method", nextTask.Method),
				zap.String("task_id", nextTask.ID),
				zap.String("cwmp_id", newCWMPID),
				zap.Int("soap_size", len(respData)),
				zap.String("soap_body", string(respData)),
				zap.String("trigger", "after_soap_fault"),
				zap.String("command_key", nextTask.CommandKey),
			)
			h.logOutgoingTransferRPC(log, deviceSN, nextTask.Method, newCWMPID, nextTask.CommandKey, nextTask.Params, respData)
			h.setSessionCookie(w, sessionID)
			h.sendSOAPResponse(w, respData, log)
			return
		}
	}

	// 没有更多命令 —— 完成会话
	h.completeSession(r.Context(), session)
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusNoContent)
}

func rpcResponseMatchesTask(responseMethod soap.RPCMethod, taskMethod string) bool {
	if taskMethod == "" {
		return false
	}
	switch responseMethod {
	case soap.MethodGetParameterValuesResp:
		return taskMethod == string(soap.MethodGetParameterValues)
	case soap.MethodSetParameterValuesResp:
		return taskMethod == string(soap.MethodSetParameterValues)
	case soap.MethodDownloadResp:
		return taskMethod == string(soap.MethodDownload)
	case soap.MethodUploadResp:
		return taskMethod == string(soap.MethodUpload)
	case soap.MethodGetParameterNamesResp:
		return taskMethod == string(soap.MethodGetParameterNames)
	case soap.MethodAddObjectResp:
		return taskMethod == string(soap.MethodAddObject)
	case soap.MethodDeleteObjectResp:
		return taskMethod == string(soap.MethodDeleteObject)
	case soap.MethodRebootResp:
		return taskMethod == string(soap.MethodReboot)
	case soap.MethodFactoryResetResp:
		return taskMethod == string(soap.MethodFactoryReset)
	case soap.MethodBaicellsPasswordResetResp:
		return taskMethod == string(soap.MethodBaicellsPasswordReset)
	case soap.MethodGetParameterAttributesResp:
		return taskMethod == string(soap.MethodGetParameterAttributes)
	case soap.MethodSetParameterAttributesResp:
		return taskMethod == string(soap.MethodSetParameterAttributes)
	case soap.MethodCommonPasswordResetResp:
		return taskMethod == "X_COMMON_COM_PasswordReset"
	default:
		return false
	}
}

const syncGPVRecoveryTaskExpiresIn = 1800

// tryRecoverGPVFault 在参数同步 GPV 收到 SOAP Fault 时执行自愈：从原批次剔除坏 path，
// 用剩余 path 重新入队 GPV 续查；批次空则视为该批完成（无任何参数可查）。
//
// 适用范围：commandKey 以 "sync-gpv-" 开头的 GetParameterValues 任务——
// 即 SyncService.StartSync / enqueueGPVPrefixes / EnqueueGPVBatches 三个入口
// （手动同步 / 周期同步 / Inform 触发 Path B / PullConfig 北向同步）。
// 其它 GPV（auto-gpv-after-spv-* / auto-gpv-after-addobject-* / 北向调试）不参与自愈。
//
// 返回 true 表示已进入自愈分支（原 task 已标 completed，新批已入队）——调用方应跳过
// MarkTaskFailed 与 publishRPCFaultEvent。返回 false 表示不适用，走原 fault 流程。
//
// 仅对具体叶子参数的 9005 做 recovery。对象/实例前缀（以 "." 结尾）通常表示实例不存在
// 或对象不可枚举，不能逐个实例滚动重试，否则会把一段连续缺失实例膨胀成很长的 -r 链。
func (h *Handler) tryRecoverGPVFault(ctx context.Context, taskItem *task.Task, badPath string, faultCode int, log *zap.Logger) bool {
	if taskItem == nil || badPath == "" {
		return false
	}
	isDurable := taskItem.Source == task.TaskSourceParamSync && strings.HasPrefix(taskItem.CommandKey, "param-sync-")
	// Durable runs tolerate an object-prefix 9005 as incomplete coverage. There
	// is nothing to split for a one-path object request, but failing the entire
	// run would cancel unrelated GPV batches and turn one unsupported subtree
	// into dozens of false failures.
	if !isRecoverableGPVBadPath(badPath, faultCode) && !isToleratedDurableGPVBadPath(taskItem, badPath, faultCode) {
		log.Info("gpv fault recovery skipped: non-leaf path or non-9005 fault",
			zap.String("task_id", taskItem.ID),
			zap.String("bad_path", badPath),
			zap.Int("fault_code", faultCode))
		return false
	}
	if taskItem.Method != "GetParameterValues" {
		return false
	}
	isLegacy := strings.HasPrefix(taskItem.CommandKey, "sync-gpv-")
	if !isLegacy && !isDurable {
		return false
	}
	var paramsObj struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(taskItem.Params, &paramsObj); err != nil {
		log.Warn("gpv fault recovery: unmarshal task params failed",
			zap.String("task_id", taskItem.ID), zap.Error(err))
		return false
	}
	remaining, skippedPaths, recoverable := planGPVFaultRecovery(paramsObj.Names, badPath, isDurable)
	if !recoverable {
		log.Warn("gpv fault recovery: bad path not in original batch (regex extraction may be off)",
			zap.String("task_id", taskItem.ID),
			zap.String("bad_path", badPath),
			zap.Strings("batch", paramsObj.Names))
		return false
	}
	result := map[string]interface{}{
		"recovered":     true,
		"bad_path":      badPath,
		"fault_code":    faultCode,
		"remaining_cnt": len(remaining),
	}
	if len(skippedPaths) > 1 || (len(skippedPaths) == 1 && skippedPaths[0] != badPath) {
		result["bad_paths"] = skippedPaths
		log.Info("durable gpv fault recovery: translated fault path did not match request; skipping batch",
			zap.String("task_id", taskItem.ID),
			zap.String("bad_path", badPath),
			zap.Int("skipped_path_count", len(skippedPaths)))
	}
	resultJSON, _ := json.Marshal(result)
	if isDurable {
		if h.gpvFaultRecoverer == nil {
			log.Error("durable gpv fault recovery is not configured", zap.String("task_id", taskItem.ID))
			return false
		}
		replacement, err := h.gpvFaultRecoverer.Recover(ctx, taskItem, remaining)
		if err != nil {
			if replacement == nil {
				log.Error("durable gpv fault recovery: extend run failed", zap.String("task_id", taskItem.ID), zap.Error(err))
				return false
			}
			// The replacement is durable and the pending outbox will retry its
			// release. Keep the original task recovered instead of turning a
			// transient Redis/admission error into a failed synchronization run.
			log.Warn("durable gpv fault recovery: immediate release failed, outbox fallback retained",
				zap.String("task_id", taskItem.ID), zap.String("replacement_task_id", replacement.ID), zap.Error(err))
		}
	}
	if err := h.taskService.MarkTaskCompleted(ctx, taskItem.ID, resultJSON); err != nil {
		log.Error("gpv fault recovery: mark original task completed failed",
			zap.String("task_id", taskItem.ID), zap.Error(err))
		return false
	}
	if len(remaining) == 0 {
		log.Info("gpv fault recovery: batch exhausted, no params left to query",
			zap.String("task_id", taskItem.ID), zap.String("bad_path", badPath))
		return true
	}
	if isDurable {
		log.Info("durable gpv fault recovery: replacement batch planned",
			zap.String("task_id", taskItem.ID), zap.String("bad_path", badPath), zap.Int("remaining", len(remaining)))
		return true
	}
	newParams, err := json.Marshal(map[string]interface{}{"names": remaining})
	if err != nil {
		log.Warn("gpv fault recovery: marshal remaining batch failed",
			zap.String("task_id", taskItem.ID), zap.Error(err))
		return true // 原 task 已 completed，不再降级到失败路径
	}
	if _, err := h.taskService.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:   taskItem.DeviceSN,
		Method:     "GetParameterValues",
		Params:     newParams,
		Priority:   taskItem.Priority,
		ExpiresIn:  syncGPVRecoveryTaskExpiresIn,
		CommandKey: taskItem.CommandKey + "-r",
		Source:     taskItem.Source,
		SourceID:   taskItem.SourceID,
	}); err != nil {
		log.Error("gpv fault recovery: enqueue retry batch failed",
			zap.String("task_id", taskItem.ID), zap.Error(err))
		return true // 原 task 已 completed
	}
	log.Info("gpv fault recovery: bad path removed, retry batch enqueued",
		zap.String("task_id", taskItem.ID),
		zap.String("device_sn", taskItem.DeviceSN),
		zap.String("bad_path", badPath),
		zap.Int("remaining", len(remaining)))
	return true
}

// removeFaultedGPVRequest removes the request item responsible for a 9005.
//
// Some CPEs answer an object-prefix GPV such as Device.DeviceInfo.EU. with a
// fault naming the concrete child they could not read, for example
// Device.DeviceInfo.EU.0.RouteIndex. The request and fault paths are therefore
// not textually equal even though the latter is covered by the former. Treating
// that as an unrecoverable mismatch cancels every unrelated batch in a full
// sync. Remove the covering object request and let the run continue with
// incomplete coverage instead.
func removeFaultedGPVRequest(names []string, badPath string) ([]string, bool) {
	badPath = strings.TrimSpace(badPath)
	remaining := make([]string, 0, len(names))
	removed := false
	for _, name := range names {
		requestPath := strings.TrimSpace(name)
		matchesFault := requestPath == badPath ||
			(strings.HasSuffix(requestPath, ".") && strings.HasPrefix(badPath, requestPath))
		if matchesFault {
			removed = true
			continue
		}
		remaining = append(remaining, name)
	}
	return remaining, removed
}

// planGPVFaultRecovery builds the continuation for a parameter-level 9005.
// Exact/covering paths remove only the rejected request. Durable full-sync
// tasks additionally tolerate a private-path translation mismatch by skipping
// the current request batch. The latter is deliberately conservative: those
// coverage paths are marked incomplete downstream, so stale values are kept
// instead of being mistaken for authoritative absence.
func planGPVFaultRecovery(names []string, badPath string, allowBatchSkip bool) (remaining, skipped []string, ok bool) {
	remaining, removed := removeFaultedGPVRequest(names, badPath)
	if removed {
		return remaining, []string{strings.TrimSpace(badPath)}, true
	}
	if !allowBatchSkip || len(names) == 0 {
		return nil, nil, false
	}
	skipped = append([]string(nil), names...)
	return nil, skipped, true
}

func isToleratedDurableGPVBadPath(taskItem *task.Task, badPath string, faultCode int) bool {
	return taskItem != nil && badPath != "" && faultCode == 9005 &&
		taskItem.Source == task.TaskSourceParamSync && strings.HasPrefix(taskItem.CommandKey, "param-sync-")
}

func isRecoverableGPVBadPath(badPath string, faultCode int) bool {
	if faultCode != 9005 {
		return false
	}
	badPath = strings.TrimSpace(badPath)
	if badPath == "" {
		return false
	}
	return !strings.HasSuffix(badPath, ".")
}

func (h *Handler) handleTransferComplete(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
	// 记录完整的请求 XML
	log.Debug("ACS received TransferComplete request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("xml", string(body)),
	)

	tc, cwmpID, err := soap.DecodeTransferComplete(bytes.NewReader(body))
	if err != nil {
		log.Error("decode TransferComplete", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 从 Cookie 会话获取 deviceSN
	deviceSN := ""
	var sessionID string
	if session, sid := h.getSessionFromCookie(r, log); session != nil {
		deviceSN = session.DeviceSN
		sessionID = sid
	}

	log.Info("TransferComplete received",
		zap.String("device_sn", deviceSN),
		zap.String("command_key", tc.CommandKey),
		zap.String("cwmp_id", cwmpID),
		zap.Any("fault", tc.FaultStruct),
		zap.String("start_time", tc.StartTime.Format(time.RFC3339)),
		zap.String("complete_time", tc.CompleteTime.Format(time.RFC3339)))

	// 协议日志元数据
	if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
		entry.SessionID = sessionID
		entry.DeviceSN = deviceSN
		entry.Method = "TransferComplete"
		entry.CwmpID = cwmpID
	}

	// 发布事件。TransferComplete SOAP 正文没有设备标识；把会话中的序列号放入
	// metadata，供设备未正确回传 CommandKey 时做受限的设备级容错关联。
	evt, evtErr := newTransferCompleteEvent(*tc, deviceSN)
	if evtErr != nil {
		log.Error("build TransferComplete event", zap.Error(evtErr))
	} else if err := h.eventBus.Publish(r.Context(), event.SubjectDeviceTransferComplete, evt); err != nil {
		log.Error("publish TransferComplete event", zap.Error(err))
	}

	// 发送 TransferCompleteResponse
	resp, err := soap.RenderResponse(soap.TransferCompleteRespTmpl, soap.InformResponseData{ID: cwmpID})
	if err != nil {
		log.Error("render TransferCompleteResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug("ACS sending TransferCompleteResponse",
		zap.String("device_sn", deviceSN),
		zap.String("xml", string(resp)),
	)
	// 如果可用，在响应中设置 Session Cookie
	if sessionID != "" {
		h.setSessionCookie(w, sessionID)
	}
	h.sendSOAPResponse(w, resp, log)
}

func newTransferCompleteEvent(tc tr069.TransferComplete, deviceSN string) (event.Event, error) {
	evt, err := event.NewEvent(event.SubjectDeviceTransferComplete, tc)
	if err != nil {
		return event.Event{}, err
	}
	if deviceSN = strings.TrimSpace(deviceSN); deviceSN != "" {
		evt.Metadata = map[string]string{event.MetadataDeviceSN: deviceSN}
	}
	return evt, nil
}

// handleAutonomousTransferComplete 处理 CPE 设备发送的 AutonomousTransferComplete 消息
// （如 PM/MR 文件上传完成）。
func (h *Handler) handleAutonomousTransferComplete(w http.ResponseWriter, r *http.Request, body []byte, log *zap.Logger) {
	// 记录完整的请求 XML
	log.Debug("ACS received AutonomousTransferComplete request",
		zap.String("remote_addr", r.RemoteAddr),
		zap.String("xml", string(body)),
	)

	atc, cwmpID, err := soap.DecodeAutonomousTransferComplete(bytes.NewReader(body))
	if err != nil {
		log.Error("decode AutonomousTransferComplete", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// 从 Cookie 会话获取 deviceSN
	deviceSN := ""
	var sessionID string
	if session, sid := h.getSessionFromCookie(r, log); session != nil {
		deviceSN = session.DeviceSN
		sessionID = sid
	}

	log.Info("AutonomousTransferComplete received",
		zap.String("device_sn", deviceSN),
		zap.String("file_type", atc.FileType),
		zap.String("transfer_url", atc.TransferURL),
		zap.Bool("is_download", atc.IsDownload),
	)

	// 协议日志元数据
	if entry := rpclog.EntryFromContext(r.Context()); entry != nil {
		entry.SessionID = sessionID
		entry.DeviceSN = deviceSN
		entry.Method = "AutonomousTransferComplete"
		entry.CwmpID = cwmpID
	}

	// 发布事件
	payload := map[string]interface{}{
		"device_sn":       deviceSN,
		"announce_url":    atc.AnnounceURL,
		"transfer_url":    atc.TransferURL,
		"is_download":     atc.IsDownload,
		"file_type":       atc.FileType,
		"file_size":       atc.FileSize,
		"target_filename": atc.TargetFileName,
		"start_time":      atc.StartTime,
		"complete_time":   atc.CompleteTime,
	}
	if atc.FaultStruct != nil {
		payload["fault"] = atc.FaultStruct
	}
	evt, _ := event.NewEvent(event.SubjectDeviceAutonomousTransferComplete, payload)
	h.eventBus.Publish(r.Context(), event.SubjectDeviceAutonomousTransferComplete, evt)

	// 发送 AutonomousTransferCompleteResponse
	resp, err := soap.RenderResponse(soap.AutonomousTransferCompleteRespTmpl, soap.InformResponseData{ID: cwmpID})
	if err != nil {
		log.Error("render AutonomousTransferCompleteResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug("ACS sending AutonomousTransferCompleteResponse",
		zap.String("device_sn", deviceSN),
		zap.String("xml", string(resp)),
	)
	// 如果可用，在响应中设置 Session Cookie
	if sessionID != "" {
		h.setSessionCookie(w, sessionID)
	}
	h.sendSOAPResponse(w, resp, log)
}

func (h *Handler) publishInformEvents(ctx context.Context, inform *tr069.InformMessage, eventCodes []string, log *zap.Logger) {
	// issue #397：发 NATS 事件之前，先同步把设备写入在线索引（acs:online ZSET）。
	// 这是「免 NATS」的存活信号 —— PM 上报洪峰压垮 NATS 时此写入不受影响，供在线计数
	// 与后续离线判定使用。ZADD O(log N)；Redis 不可用时 Mark 内部降级 no-op，
	// 失败仅告警、绝不阻断 Inform 主流程。
	if err := h.onlineIndex.Mark(ctx, inform.DeviceId.SerialNumber, time.Now().Unix()); err != nil {
		log.Warn("mark device online (acs:online) failed",
			zap.Error(err), zap.String("device_sn", inform.DeviceId.SerialNumber))
	}

	// 构建事件载荷
	payload := map[string]interface{}{
		"device_id":      inform.DeviceId,
		"events":         eventCodes,
		"event_structs":  inform.Event,
		"parameter_list": inform.ParameterList,
		"current_time":   inform.CurrentTime,
		"retry_count":    inform.RetryCount,
	}

	// 根据事件码确定主事件主题（按优先级排序）。
	// BOOTSTRAP 优先于 BOOT：首次入网/出厂复位伴随 BOOT 也归类为 bootstrap，
	// 触发设备注册与自动开站流程；仅 BOOT（无 BOOTSTRAP）或 M Reboot 归类为
	// reboot_complete，触发重启统计与状态恢复。
	var subject string
	switch {
	case tr069.IsBootstrap(inform.Event):
		subject = event.SubjectDeviceBootstrap
	case tr069.IsAlarm(inform.Event):
		subject = event.SubjectDeviceAlarm
	case tr069.IsStartupResultReport(inform.Event):
		subject = event.SubjectDeviceStartupResultReport
	case tr069.IsStartupStageReport(inform.Event):
		subject = event.SubjectDeviceStartupStageReport
	case tr069.IsRebootComplete(inform.Event), tr069.IsBoot(inform.Event):
		subject = event.SubjectDeviceRebootComplete
	case tr069.IsUpgradeFinish(inform.Event):
		subject = event.SubjectDeviceUpgradeFinish
	case tr069.IsConnectionRequest(inform.Event):
		subject = event.SubjectDeviceConnectionRequest
	case tr069.HasEvent(inform.Event, tr069.EventTransferComplete):
		subject = event.SubjectDeviceTransferComplete
	case tr069.IsValueChange(inform.Event):
		subject = event.SubjectDeviceValueChange
	case tr069.IsPeriodic(inform.Event):
		subject = event.SubjectDevicePeriodic
	default:
		subject = event.SubjectDevicePeriodic
	}

	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		log.Error("create event failed",
			zap.Error(err),
			zap.String("device_sn", inform.DeviceId.SerialNumber),
			zap.String("subject", subject))
		return
	}

	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		log.Error("publish event failed",
			zap.Error(err),
			zap.String("device_sn", inform.DeviceId.SerialNumber),
			zap.String("subject", subject),
			zap.String("event_id", evt.ID))
		return
	}

	log.Info("event published to bus",
		zap.String("device_sn", inform.DeviceId.SerialNumber),
		zap.String("subject", subject),
		zap.String("event_id", evt.ID),
		zap.Strings("event_codes", eventCodes))

	// Additional: if VALUE CHANGE contains ExpeditedEvent parameters, publish expedited alarm event.
	if tr069.IsValueChange(inform.Event) && hasExpeditedEventParams(inform.ParameterList) {
		expPayload := map[string]interface{}{
			"device_sn":        inform.DeviceId.SerialNumber,
			"parameter_values": filterExpeditedEventParams(inform.ParameterList),
		}
		expEvt, err := event.NewEvent(event.SubjectDeviceExpeditedAlarm, expPayload)
		if err != nil {
			log.Error("create expedited alarm event failed", zap.Error(err))
		} else if err := h.eventBus.Publish(ctx, event.SubjectDeviceExpeditedAlarm, expEvt); err != nil {
			log.Error("publish expedited alarm event failed", zap.Error(err))
		} else {
			log.Info("expedited alarm event published",
				zap.String("device_sn", inform.DeviceId.SerialNumber),
				zap.String("event_id", expEvt.ID))
		}
	}
}

func (h *Handler) publishRPCResponseEvent(ctx context.Context, deviceSN string, method soap.RPCMethod, body []byte, lastCmdParams json.RawMessage, taskItem *task.Task, log *zap.Logger) {
	var subject string
	switch method {
	case soap.MethodGetParameterValuesResp:
		subject = event.SubjectCommandGetParamsResponse
	case soap.MethodSetParameterValuesResp:
		subject = event.SubjectCommandSetParamsResponse
	case soap.MethodDownloadResp:
		subject = event.SubjectCommandDownloadResponse
	case soap.MethodUploadResp:
		subject = event.SubjectCommandUploadResponse
	case soap.MethodGetParameterNamesResp:
		subject = event.SubjectCommandGetNamesResponse
	case soap.MethodAddObjectResp:
		subject = event.SubjectCommandAddObjectResponse
	case soap.MethodDeleteObjectResp:
		subject = event.SubjectCommandDeleteObjectResponse
	case soap.MethodRebootResp:
		subject = event.SubjectCommandRebootResponse
	case soap.MethodFactoryResetResp:
		subject = event.SubjectCommandFactoryResetResponse
	case soap.MethodGetParameterAttributesResp:
		subject = event.SubjectCommandGetAttrsResponse
	case soap.MethodSetParameterAttributesResp:
		subject = event.SubjectCommandSetAttrsResponse
	default:
		return
	}

	payload := map[string]interface{}{
		"device_sn": deviceSN,
		"method":    string(method),
	}
	if taskItem != nil && taskItem.ID != "" {
		payload["task_id"] = taskItem.ID
		payload["task_source"] = taskItem.Source
		payload["task_source_id"] = taskItem.SourceID
	}

	// command_key 让下游订阅者(如 Path B reconcile)按入队方约定区分触发上下文,
	// 避免局部 follow-up GPV(如 auto-gpv-after-addobject-*)被当作"全量同步"
	// 触发对象级 reconcile 误删兄弟实例。
	if taskItem != nil && taskItem.CommandKey != "" {
		payload["command_key"] = taskItem.CommandKey
	}

	// 从 lastCmdParams 提取原始命令路径（用于 GPN/GPV 关联）+ object_name（AddObject/DeleteObject）。
	if len(lastCmdParams) > 0 {
		var cmdMeta struct {
			Path       string `json:"path"`
			ObjectName string `json:"object_name"`
		}
		if err := json.Unmarshal(lastCmdParams, &cmdMeta); err == nil {
			if cmdMeta.Path != "" {
				payload["path"] = cmdMeta.Path
			}
			if cmdMeta.ObjectName != "" {
				payload["object_name"] = cmdMeta.ObjectName
			}
		}
	}

	// 对于 GPN/GPV 响应，解析 SOAP body 并包含结构化数据，
	// 而非原始 XML，避免 NATS 消息大小限制。
	switch method {
	case soap.MethodGetParameterNamesResp:
		paramInfos, _, parseErr := soap.DecodeGetParameterNamesResponse(bytes.NewReader(body))
		if parseErr != nil {
			log.Warn("parse GPN response for event", zap.Error(parseErr))
		} else {
			payload["parameter_infos"] = paramInfos
			log.Info("GPN response parsed for event",
				zap.String("device_sn", deviceSN),
				zap.Int("parameter_count", len(paramInfos)),
				zap.String("path", fmt.Sprintf("%v", payload["path"])),
			)
		}
	case soap.MethodGetParameterValuesResp:
		paramValues, _, parseErr := soap.DecodeGetParameterValuesResponse(bytes.NewReader(body))
		if parseErr != nil {
			log.Warn("parse GPV response for event", zap.Error(parseErr))
		} else {
			payload["parameter_values"] = paramValues
			log.Info("GPV response parsed for event",
				zap.String("device_sn", deviceSN),
				zap.Int("parameter_count", len(paramValues)),
			)
			// 体量防御:Path B 同步对大对象(如 DeviceGSM.Bts.,17791 项 ~1.5MB)
			// 现已允许在 NATS max_payload=5MB 内整对象返回全部实例。这里用
			// 20000 项作为兜底告警,超过后需要排查 sync_pathb_expand 的 5MB
			// 估算是否偏低,或设备是否返回了异常大对象。
			if len(paramValues) > 20000 {
				log.Warn("GPV response payload is very large; instance expand may be misconfigured",
					zap.String("device_sn", deviceSN),
					zap.Int("parameter_count", len(paramValues)),
				)
			}
		}
	}

	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		log.Error("create RPC response event", zap.Error(err))
		return
	}
	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		log.Error("publish RPC response event", zap.Error(err), zap.String("subject", subject))
	}
}

// publishRPCFaultEvent 在 CPE 回 SOAP Fault 时，按"原 RPC 方法"派发到对应的
// command.*.response 事件 subject 上，payload 带 fault_code / fault_string，
// 让业务订阅器（software.HandleUploadResponse / HandleSetParamsResponseForCollect 等）
// 跟收到正常响应一样走 fault 分支处理。
//
// 与 publishRPCResponseEvent 的区别：那个是收到合法 *Response SOAP body 时调用，
// 这里是只有 SOAP Fault（不含 Response body）时的兜底。
func (h *Handler) publishRPCFaultEvent(ctx context.Context, deviceSN string, taskItem *task.Task, faultCode int, soapFaultCode, faultMsg string, log *zap.Logger) {
	if taskItem == nil || h.eventBus == nil {
		return
	}
	var subject string
	switch taskItem.Method {
	case "GetParameterValues":
		subject = event.SubjectCommandGetParamsResponse
	case "SetParameterValues":
		subject = event.SubjectCommandSetParamsResponse
	case "Download":
		subject = event.SubjectCommandDownloadResponse
	case "Upload":
		subject = event.SubjectCommandUploadResponse
	case "GetParameterNames":
		subject = event.SubjectCommandGetNamesResponse
	case "AddObject":
		subject = event.SubjectCommandAddObjectResponse
	case "DeleteObject":
		subject = event.SubjectCommandDeleteObjectResponse
	case "Reboot":
		subject = event.SubjectCommandRebootResponse
	case "FactoryReset":
		subject = event.SubjectCommandFactoryResetResponse
	case "GetParameterAttributes":
		subject = event.SubjectCommandGetAttrsResponse
	case "SetParameterAttributes":
		subject = event.SubjectCommandSetAttrsResponse
	default:
		return
	}

	payload := map[string]interface{}{
		"device_sn":       deviceSN,
		"method":          taskItem.Method,
		"task_id":         taskItem.ID,
		"task_source":     taskItem.Source,
		"task_source_id":  taskItem.SourceID,
		"command_key":     taskItem.CommandKey,
		"fault_code":      faultCode,     // 数值：cwmp:FaultCode（标准 CWMP），无则 0
		"fault_code_text": soapFaultCode, // 字符串：soap:faultcode（SOAP 1.1 outer，如 "Server.Internal"）
		"fault_string":    faultMsg,      // 已含 [soapFaultCode] 前缀的人类可读消息
	}
	evt, err := event.NewEvent(subject, payload)
	if err != nil {
		log.Warn("build RPC fault event", zap.Error(err), zap.String("subject", subject))
		return
	}
	if err := h.eventBus.Publish(ctx, subject, evt); err != nil {
		log.Error("publish RPC fault event", zap.Error(err), zap.String("subject", subject))
		return
	}
	log.Info("RPC fault event published",
		zap.String("subject", subject),
		zap.String("device_sn", deviceSN),
		zap.String("command_key", taskItem.CommandKey),
		zap.Int("fault_code", faultCode))
}

// queueAutoGPVAfterSPV 在 SetParameterValuesResponse 成功完成后,自动入队一个
// GetParameterValues task 拉取本次改过的 path 列表,触发 device_parameters.current_value 同步。
//
// 触发条件:T-0147 在 SPV task completed 后调用本函数。
// 工作流:
//  1. 从 SPV task.Params 解析 values: [{name, value, type}, ...](与 dispatcher BuildRequest 期望对齐)
//  2. 提取 path 列表
//  3. 调 task service 入队 GPV(method=GetParameterValues, params={names: [...]});
//     command_key 关联原 SPV task_id 便于追溯
//  4. GPV 完成后由已有 RPCResponseSubscriber.handleGPVResponse 自动写 device_parameters
//
// 失败语义:解析/入队失败 仅 log Warn,不阻塞主流程(SPV 主任务已完成)。
func (h *Handler) queueAutoGPVAfterSPV(ctx context.Context, spvTask *task.Task, log *zap.Logger) {
	if h.taskService == nil || spvTask == nil || len(spvTask.Params) == 0 {
		return
	}
	// Geofence actions own a correlated readback task and must not also create
	// the generic T-0147 readback. The correlated task carries SourceID so the
	// action can distinguish "SPV accepted" from "requested values verified".
	if spvTask.Source == task.TaskSourceGeofence {
		return
	}

	var spvParams struct {
		Values []struct {
			Name string `json:"name"`
		} `json:"values"`
	}
	if err := json.Unmarshal(spvTask.Params, &spvParams); err != nil {
		log.Warn("T-0147 auto GPV after SPV: parse spv params failed",
			zap.Error(err), zap.String("spv_task_id", spvTask.ID))
		return
	}
	if len(spvParams.Values) == 0 {
		return
	}

	names := make([]string, 0, len(spvParams.Values))
	for _, v := range spvParams.Values {
		if v.Name != "" {
			names = append(names, v.Name)
		}
	}
	if len(names) == 0 {
		return
	}
	if h.requestDurableReadback(ctx, spvTask.DeviceSN, "spv_readback", names, "spv-readback:"+spvTask.ID, log) {
		return
	}

	gpvParams, err := json.Marshal(map[string]interface{}{"names": names})
	if err != nil {
		log.Warn("T-0147 auto GPV after SPV: marshal gpv params failed",
			zap.Error(err), zap.String("spv_task_id", spvTask.ID))
		return
	}

	gpvTask, err := h.taskService.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:    spvTask.DeviceSN,
		Method:      "GetParameterValues",
		Params:      gpvParams,
		Priority:    5,
		CommandKey:  fmt.Sprintf("auto-gpv-after-spv-%s", spvTask.ID[:8]),
		Source:      task.TaskSourceSystem,
		Description: "auto GPV after SPV (T-0147)",
	})
	if err != nil {
		log.Warn("T-0147 auto GPV after SPV: enqueue gpv failed",
			zap.Error(err), zap.String("spv_task_id", spvTask.ID))
		return
	}
	log.Info("T-0147 auto GPV after SPV: enqueued",
		zap.String("spv_task_id", spvTask.ID),
		zap.String("gpv_task_id", gpvTask.ID),
		zap.Int("param_count", len(names)))
}

// queueAutoGPVAfterAddObject 在 AddObjectResponse 成功完成后,针对新实例 path
// 入队一个 GetParameterValues 拉取该对象下所有参数,触发 device_parameters 写入。
//
// 工作流:
//  1. 解析 AddObjectResponse SOAP body 取 InstanceNumber
//  2. 从 addObjTask.Params 解析 object_name(父对象 path,形如 "Device.X.Y.Z.")
//  3. 拼接新实例 path: object_name + "{N}."
//  4. 入队 GPV(names=[新实例 path]);TR-069 协议:"." 结尾的 name 拉对象下所有参数
//  5. GPV 完成后由已有 RPCResponseSubscriber.handleGPVResponse 自动写 device_parameters
//
// 失败语义:解析/入队失败仅 log Warn,不阻塞主流程(AddObject 主任务已完成,实例已建)。
func (h *Handler) queueAutoGPVAfterAddObject(ctx context.Context, addObjTask *task.Task, body []byte, log *zap.Logger) {
	if h.taskService == nil || addObjTask == nil || len(addObjTask.Params) == 0 {
		return
	}

	instanceNumber, _, _, err := soap.DecodeAddObjectResponse(bytes.NewReader(body))
	if err != nil {
		log.Warn("auto GPV after AddObject: decode response failed",
			zap.Error(err), zap.String("add_obj_task_id", addObjTask.ID))
		return
	}
	if instanceNumber <= 0 {
		log.Warn("auto GPV after AddObject: invalid instance number",
			zap.Int("instance_number", instanceNumber),
			zap.String("add_obj_task_id", addObjTask.ID))
		return
	}

	var addParams struct {
		ObjectName string `json:"object_name"`
	}
	if err := json.Unmarshal(addObjTask.Params, &addParams); err != nil {
		log.Warn("auto GPV after AddObject: parse add_object params failed",
			zap.Error(err), zap.String("add_obj_task_id", addObjTask.ID))
		return
	}
	if addParams.ObjectName == "" {
		return
	}

	newInstancePath := addParams.ObjectName + strconv.Itoa(instanceNumber) + "."
	if h.requestDurableReadback(ctx, addObjTask.DeviceSN, "add_object_readback", []string{newInstancePath}, "add-object-readback:"+addObjTask.ID, log) {
		return
	}

	gpvParams, err := json.Marshal(map[string]interface{}{"names": []string{newInstancePath}})
	if err != nil {
		log.Warn("auto GPV after AddObject: marshal gpv params failed",
			zap.Error(err), zap.String("add_obj_task_id", addObjTask.ID))
		return
	}

	gpvTask, err := h.taskService.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN:    addObjTask.DeviceSN,
		Method:      "GetParameterValues",
		Params:      gpvParams,
		Priority:    5,
		CommandKey:  fmt.Sprintf("auto-gpv-after-addobject-%s", addObjTask.ID[:8]),
		Source:      task.TaskSourceSystem,
		Description: "auto GPV after AddObject",
	})
	if err != nil {
		log.Warn("auto GPV after AddObject: enqueue gpv failed",
			zap.Error(err), zap.String("add_obj_task_id", addObjTask.ID))
		return
	}
	log.Info("auto GPV after AddObject: enqueued",
		zap.String("add_obj_task_id", addObjTask.ID),
		zap.String("gpv_task_id", gpvTask.ID),
		zap.String("new_instance_path", newInstancePath))
}

func (h *Handler) requestDurableReadback(ctx context.Context, deviceSN, reason string, paths []string, key string, log *zap.Logger) bool {
	if !h.durableReadbackEnabled || h.eventBus == nil || deviceSN == "" || len(paths) == 0 {
		return false
	}
	evt, err := event.NewEvent(event.SubjectParamSyncRequested, event.ParamSyncRequestedPayload{
		DeviceSN: deviceSN, TriggerReason: reason, RequestedPaths: paths, IdempotencyKey: key,
	})
	if err == nil {
		err = h.eventBus.Publish(ctx, event.SubjectParamSyncRequested, evt)
	}
	if err != nil {
		log.Warn("publish durable parameter readback request; falling back to legacy task", zap.String("reason", reason), zap.Error(err))
		return false
	}
	return true
}

// processInformPeriodGPV 处理心跳周期 GPV 响应。
//
// #746: 识别心跳周期 GPV 任务（通过 Description 标识），解析响应中的 PeriodicInformInterval，
// 与配置目标值比较，不一致则入队 SPV 调整。
func (h *Handler) processInformPeriodGPV(ctx context.Context, gpvTask *task.Task, body []byte, log *zap.Logger) {
	if h.informPeriodPolicy == nil || !h.informPeriodPolicy.Enabled() {
		return
	}
	if gpvTask == nil {
		return
	}

	// 检查是否是心跳周期 GPV 任务
	if !IsInformPeriodGPVTask(gpvTask.Description) {
		return
	}

	// 从 Description 中提取 productClass
	productClass := ExtractProductClassFromDescription(gpvTask.Description)
	if productClass == "" {
		log.Warn("inform period GPV: cannot extract product class from description",
			zap.String("task_id", gpvTask.ID),
			zap.String("description", gpvTask.Description))
		return
	}

	// 解析 GPV 响应
	pvs, _, err := soap.DecodeGetParameterValuesResponse(bytes.NewReader(body))
	if err != nil {
		log.Warn("inform period GPV: decode response failed",
			zap.String("task_id", gpvTask.ID),
			zap.Error(err))
		return
	}
	if len(pvs) == 0 {
		log.Debug("inform period GPV: empty response",
			zap.String("task_id", gpvTask.ID))
		return
	}

	// 转换为 ParameterValue 类型
	paramValues := make([]ParameterValue, len(pvs))
	for i, pv := range pvs {
		paramValues[i] = ParameterValue{Name: pv.Name, Value: pv.Value}
	}

	// 调用策略处理响应
	enqueuedSPV := h.informPeriodPolicy.ProcessGPVResponse(ctx, gpvTask.DeviceSN, productClass, paramValues)
	if enqueuedSPV {
		log.Info("inform period GPV processed: SPV enqueued to adjust",
			zap.String("device_sn", gpvTask.DeviceSN),
			zap.String("product_class", productClass))
	}
}

func (h *Handler) sendInformResponse(w http.ResponseWriter, cwmpID string, log *zap.Logger) {
	// TR-069 规范：InformResponse 仅包含 MaxEnvelopes。
	// CurrentTime 非标准字段，已移除以符合协议规范。
	resp, err := soap.RenderResponse(soap.InformResponseTmpl, soap.InformResponseData{
		ID: cwmpID,
	})
	if err != nil {
		log.Error("render InformResponse", zap.Error(err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Debug("ACS sending InformResponse",
		zap.String("cwmp_id", cwmpID),
		zap.String("xml", string(resp)),
	)
	h.sendSOAPResponse(w, resp, log)
}

func (h *Handler) sendSOAPResponse(w http.ResponseWriter, data []byte, log *zap.Logger) {
	log.Debug("ACS sent SOAP",
		zap.Int("soap_size", len(data)),
		zap.String("soap_body", string(data)),
	)
	// T-0105: 显式 Content-Length 强制 identity transfer encoding。
	// 否则 Go 在 WriteHeader 后看不到 Content-Length 会用 Transfer-Encoding: chunked。
	// 部分 CPE 固件（BAICELLS BaiBLQ_5.0.16.1_1229 实测）在 POST 自己 RPC 响应后
	// 收到 chunked-encoded 的 piggyback RPC 请求会停止后续 POST，断了 RPC chaining。
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// getSessionFromCookie 从 Cookie 头中获取会话。
// 未找到有效 Session Cookie 时返回 nil。
// log 参数应为携带 request_id 的上下文感知日志器。
func (h *Handler) getSessionFromCookie(r *http.Request, log *zap.Logger) (*Session, string) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		log.Debug("no session cookie", zap.Error(err), zap.String("remote_addr", r.RemoteAddr))
		return nil, ""
	}

	sessionID := cookie.Value
	session, err := h.sessionStore.GetByID(r.Context(), sessionID)
	if err != nil {
		log.Error("get session by cookie", zap.Error(err), zap.String("session_id", sessionID))
		return nil, sessionID
	}
	if session != nil && h.deviceSessionStore != nil && session.DeviceSN != "" {
		currentSessionID, err := h.deviceSessionStore.Get(r.Context(), session.DeviceSN)
		if err != nil {
			log.Warn("get current device session",
				zap.Error(err),
				zap.String("device_sn", session.DeviceSN),
				zap.String("session_id", sessionID))
		} else if currentSessionID != "" && currentSessionID != sessionID {
			log.Warn("stale session cookie rejected",
				zap.String("device_sn", session.DeviceSN),
				zap.String("session_id", sessionID),
				zap.String("current_session_id", currentSessionID))
			return nil, sessionID
		}
	}

	return session, sessionID
}

// setSessionCookie 在响应头中设置 SESSION Cookie。
func (h *Handler) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// truncateString 将字符串截断到 maxLen 字符，用于日志输出。
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// generateSessionID 生成基于 UUID 的 Session ID，用于 Cookie。
func generateSessionID() string {
	uuidBytes := make([]byte, 16)
	cryptorand.Read(uuidBytes)
	// 按 RFC 4122 设置版本（4）和变体位
	uuidBytes[6] = (uuidBytes[6] & 0x0f) | 0x40
	uuidBytes[8] = (uuidBytes[8] & 0x3f) | 0x80
	return hex.EncodeToString(uuidBytes)
}

// =====================================================================// 测试功能：随机任务注入
// =====================================================================// 以下方法仅用于测试目的。
// CPE 发送 Inform 时注入随机 RPC 任务。
// 禁用方法：注释掉 handleInform 中对 injectRandomTestTasks 的调用。
// =====================================================================
// rpcTaskTemplate 定义随机 RPC 任务生成模板
type rpcTaskTemplate struct {
	method   string
	params   json.RawMessage
	priority int
}

// testRPCTaskTemplates 包含所有可用的随机生成 RPC 任务模板
var testRPCTaskTemplates = []rpcTaskTemplate{
	{method: "GetParameterValues", params: json.RawMessage(`{"names":["Device.DeviceInfo.SoftwareVersion","Device.DeviceInfo.HardwareVersion"]}`), priority: 10},
	{method: "GetParameterValues", params: json.RawMessage(`{"names":["Device.X_0000B9_Config.AntennaConfig"]}`), priority: 10},
	{method: "SetParameterValues", params: json.RawMessage(`{"values":[{"name":"Device.X_0000B9_Config.TestParam","value":"test_value_123","type":"xsd:string"}]}`), priority: 10},
	{method: "GetParameterNames", params: json.RawMessage(`{"path":"Device.DeviceInfo.","next_level":false}`), priority: 10},
	{method: "GetParameterNames", params: json.RawMessage(`{"path":"Device.X_0000B9_Config.","next_level":true}`), priority: 10},
	{method: "GetParameterAttributes", params: json.RawMessage(`{"names":["Device.DeviceInfo.SoftwareVersion"]}`), priority: 10},
	{method: "SetParameterAttributes", params: json.RawMessage(`{"attributes":[{"name":"Device.X_Test.Param","notification_change":true,"notification":1}]}`), priority: 10},
	{method: "Download", params: json.RawMessage(`{"file_type":"1 Firmware Upgrade Image","url":"http://acs.example.com/firmware/v1.0.0.bin","file_size":10485760}`), priority: 5},
	{method: "Upload", params: json.RawMessage(`{"file_type":"2 Vendor Log File","url":"http://acs.example.com/upload/logs","delay_seconds":0}`), priority: 10},
	{method: "Reboot", params: json.RawMessage(`{}`), priority: 100}, // 高优先级但应排在最后
	{method: "FactoryReset", params: json.RawMessage(`{}`), priority: 100},
}

// injectRandomTestTasks 为测试目的注入随机 RPC 任务。
// 为设备生成 1-3 个随机任务，外加一个固定的 PM 上传任务。
// 如果生成了 Reboot 任务，会移到队列末尾。
// 注意：这是测试功能，通过设置 enableTestTaskInjection 为 false 禁用。
func (h *Handler) injectRandomTestTasks(r *http.Request, deviceSN string, log *zap.Logger) {
	// 如果测试任务注入已禁用则跳过
	if !h.enableTestTaskInjection {
		return
	}

	// 如果 TaskService 不可用则跳过
	if h.taskService == nil {
		return
	}

	ctx := r.Context()

	// 生成随机数量的任务（1-3）
	numTasks := 1 + rand.Intn(3) // 1 + 0-3 = 1-3

	// 随机选择任务
	selectedIndices := make(map[int]bool)
	var tasks []rpcTaskTemplate

	for len(tasks) < numTasks {
		idx := rand.Intn(len(testRPCTaskTemplates))
		if selectedIndices[idx] {
			continue
		}
		selectedIndices[idx] = true
		tasks = append(tasks, testRPCTaskTemplates[idx])
	}

	// 固定：始终注入一个使用真实上传 URL 的 PM 上传任务
	pmUploadTask := h.createPMUploadTask(r, deviceSN)
	if pmUploadTask != nil {
		tasks = append(tasks, *pmUploadTask)
	}

	// 排序任务：Reboot 应放在最后
	var normalTasks []rpcTaskTemplate
	var rebootTask *rpcTaskTemplate

	for _, t := range tasks {
		if t.method == "Reboot" {
			rebootTask = &t
		} else {
			normalTasks = append(normalTasks, t)
		}
	}

	// 组合：普通任务在前，Reboot 在后（如果存在）
	finalTasks := normalTasks
	if rebootTask != nil {
		finalTasks = append(finalTasks, *rebootTask)
	}

	// 在 TaskService 中创建任务
	createdCount := 0
	for _, t := range finalTasks {
		req := &task.CreateTaskRequest{
			DeviceSN:    deviceSN,
			Method:      t.method,
			Params:      t.params,
			Priority:    t.priority,
			Source:      task.TaskSourceSystem,
			Description: fmt.Sprintf("Test task: %s", t.method),
		}

		_, err := h.taskService.CreateTask(ctx, req)
		if err != nil {
			log.Warn("failed to create test task",
				zap.String("device_sn", deviceSN),
				zap.String("method", t.method),
				zap.Error(err))
			continue
		}
		createdCount++
	}

	log.Info("injected random test tasks",
		zap.String("device_sn", deviceSN),
		zap.Int("total_selected", len(tasks)),
		zap.Int("created", createdCount),
		zap.Bool("has_reboot", rebootTask != nil),
		zap.Bool("has_pm_upload", pmUploadTask != nil))
}

// createPMUploadTask 使用配置中的真实上传 URL 创建 PM 文件上传任务。
// 如果 base_url 是 localhost，会替换为请求的 host。
// 上传配置不可用时返回 nil。
func (h *Handler) createPMUploadTask(r *http.Request, deviceSN string) *rpcTaskTemplate {
	uploadCfg := h.currentUploadSettings(r.Context())
	if uploadCfg.BaseURL == "" {
		return nil
	}

	// 生成上传 URL：{BaseURL}{Path}?fileType=PM&filename={deviceSN}_{timestamp}.xml.gz
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.xml.gz", deviceSN, timestamp)

	// 获取基础 URL，如需要则将 localhost 替换为请求的 host
	baseURL := uploadCfg.BaseURL
	// if isLocalhost(baseURL) {
	// 	// 使用请求的 host 替代 localhost
	// 	scheme := "http"
	// 	if r.TLS != nil {
	// 		scheme = "https"
	// 	}
	// 	baseURL = fmt.Sprintf("%s://%s", scheme, r.Host)
	// }

	// 构建上传 URL
	uploadURL := fmt.Sprintf("%s%s?fileType=PM&filename=%s",
		baseURL,
		uploadCfg.Path,
		filename,
	)

	// 创建 Upload 任务参数
	// TR069 Upload RPC 参数：
	// - FileType: "1 Vendor Configuration File" (1) 或 "2 Vendor Log File" (2) 等
	// - URL: CPE 上传文件的目标 URL
	// - Username/Password: HTTP Basic Auth 凭据（可选）
	params := map[string]interface{}{
		"file_type":     "1 Vendor Configuration File", // PM 数据作为厂商配置文件
		"url":           uploadURL,
		"delay_seconds": 0,
	}
	if uploadCfg.Username != "" {
		params["username"] = uploadCfg.Username
		params["password"] = uploadCfg.Password
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil
	}

	return &rpcTaskTemplate{
		method:   "Upload",
		params:   json.RawMessage(paramsJSON),
		priority: 10,
	}
}

func (h *Handler) currentUploadSettings(ctx context.Context) transfercfg.UploadSettings {
	if h.transferConfigProvider != nil {
		return h.transferConfigProvider.Snapshot(ctx).Upload
	}
	if h.uploadConfig == nil {
		return transfercfg.UploadSettings{}
	}
	return transfercfg.UploadSettings{
		BaseURL:     strings.TrimRight(strings.TrimSpace(h.uploadConfig.BaseURL), "/"),
		Path:        h.uploadConfig.Path,
		Username:    h.uploadConfig.Username,
		Password:    h.uploadConfig.Password,
		MaxFileSize: h.uploadConfig.MaxFileSize,
	}
}

// =====================================================================// 测试功能结束
// =====================================================================

// logOutgoingTransferRPC 在 ACS 把 Upload / Download SOAP 写回 CPE 之前，
// 把渲染好的关键字段（FileType / URL / CommandKey / fileName / username / md5）+ 完整 SOAP
// 单独打一条 INFO 日志，便于排障时直接 grep。
//
// 触发原因：通用日志 "ACS sending RPC request from task" 里其实 soap_body 字段已经带
// 完整报文，但混在所有 RPC 一起。Upload / Download 是排查文件传输问题的核心，
// 拆出来打一条带解析字段的专属日志，让 grep 一步到位。
//
// 仅对 method == "Upload" / "Download" 触发；其它方法 no-op。失败（params JSON 不合法）
// 时只记录原始 soap_body，不抛错——日志是辅助手段，不能影响 RPC 下发主链路。
func (h *Handler) logOutgoingTransferRPC(log *zap.Logger, deviceSN, method, cwmpID, commandKey string, params []byte, soapBody []byte) {
	if method != "Upload" && method != "Download" {
		return
	}
	// Upload / Download params JSON 是 SoftwareService / backup.Executor 序列化后塞进
	// task 行的，结构和 soap.UploadData / soap.DownloadData 对齐。这里用最小子集解析，
	// 避免对 soap 包反向依赖（acs/rpc 包已经依赖 soap，但 handler 层不直接关心字段细节）。
	var detail struct {
		FileType       string `json:"file_type"`
		URL            string `json:"url"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		FileName       string `json:"file_name"`
		TargetFileName string `json:"target_file_name"`
		FileSize       int64  `json:"file_size"`
		MD5            string `json:"md5"`
		RawMode        string `json:"raw_mode"`
	}
	_ = json.Unmarshal(params, &detail) // best-effort; 失败时各字段为零值

	fields := []zap.Field{
		zap.String("device_sn", deviceSN),
		zap.String("method", method),
		zap.String("cwmp_id", cwmpID),
		zap.String("command_key", commandKey),
		zap.String("file_type", detail.FileType),
		zap.String("url", detail.URL),
		zap.String("username", detail.Username),
		// password 不打——按安全合规要求避免日志泄露
		zap.String("file_name", detail.FileName),
		zap.String("target_file_name", detail.TargetFileName),
		zap.Int64("file_size", detail.FileSize),
		zap.String("md5", detail.MD5),
		zap.String("raw_mode", detail.RawMode),
		zap.Int("soap_size", len(soapBody)),
		zap.String("soap_body", string(soapBody)),
	}
	log.Info("ACS sending TR-069 "+method+" SOAP to CPE", fields...)
}

// faultEnvRegex matches the SOAP <Fault> element regardless of namespace
// prefix (soap-env / SOAP-ENV / soapenv / env / s / cwmp / none) and case.
// Examples that must match: <Fault>, <soap-env:Fault>, <SOAP-ENV:Fault>.
var faultEnvRegex = regexp.MustCompile(`(?i)<(?:[a-z][\w-]*:)?fault[ >]`)

// cwmpFaultCodeRegex / cwmpFaultStringRegex extract the inner CWMP-level
// FaultCode / FaultString carried under <detail><cwmp:Fault>. CWMP spec uses
// CamelCase tag names, while SOAP-level <faultcode>/<faultstring> are
// lowercase; matching case-sensitively keeps the two layers distinct.
var (
	cwmpFaultCodeRegex   = regexp.MustCompile(`(?s)<(?:[a-zA-Z][\w-]*:)?FaultCode>\s*(\d+)\s*</(?:[a-zA-Z][\w-]*:)?FaultCode>`)
	cwmpFaultStringRegex = regexp.MustCompile(`(?s)<(?:[a-zA-Z][\w-]*:)?FaultString>([^<]*)</(?:[a-zA-Z][\w-]*:)?FaultString>`)
	soapFaultStringRegex = regexp.MustCompile(`(?s)<faultstring>([^<]*)</faultstring>`)
	soapFaultCodeRegex   = regexp.MustCompile(`(?s)<faultcode>([^<]*)</faultcode>`)

	// faultPathKeywordRegex 匹配带关键字标记的坏 path（厂商实测格式）：
	//   "Invalid Parameter Names [1], including: Device.X.Y.Z."
	//   "Invalid parameter name: Device.X"
	//   "Parameter 'Device.X' is invalid"
	// 抓关键字后的第一个 dot-separated 标识符（末尾可有 "."，对象前缀语义需保留）。
	faultPathKeywordRegex = regexp.MustCompile(`(?i)(?:including|parameter|name)[\s:'"\x60]+([A-Za-z_]\w*(?:\.[A-Za-z0-9_]+)+\.?)`)
	// faultPathGenericRegex 兜底：找任何 dot-separated 标识符（>=2 段）。部分
	// GSM 顶层参数只有 DeviceGSM.NriNullDel 两段，仍是合法 CWMP path。
	faultPathGenericRegex = regexp.MustCompile(`[A-Za-z_]\w*(?:\.[A-Za-z0-9_]+){1,}\.?`)

	// T-0174 — extract per-parameter SetParameterValuesFault detail blocks.
	// CPE returns one block per offending path; outer cwmp:FaultCode is always 9003
	// for atomic SPV failure, but the inner per-path codes carry the real cause
	// (9005=name not found, 9007=value out of range, 9008=read-only, ...).
	spvFaultBlockRegex = regexp.MustCompile(`(?s)<(?:[a-zA-Z][\w-]*:)?SetParameterValuesFault>(.*?)</(?:[a-zA-Z][\w-]*:)?SetParameterValuesFault>`)
	spvFaultNameRegex  = regexp.MustCompile(`(?s)<(?:[a-zA-Z][\w-]*:)?ParameterName>([^<]*)</(?:[a-zA-Z][\w-]*:)?ParameterName>`)
)

// SPVFault 携带 SetParameterValues 失败时 CPE 返回的 per-parameter 详情。
// outer cwmp:Fault.FaultCode 一律 9003（SPV 是原子操作，任何一个 param 出错
// 整条 RPC 失败），真实原因在每个 <SetParameterValuesFault> 块里。
//
// JSON tag 与 device_tasks.result.param_faults[] 形态对齐，下游 MML
// ResultAggregator 直接反序列化消费做 auto-learn。
type SPVFault struct {
	ParameterName string `json:"parameter_name"`
	FaultCode     int    `json:"fault_code"`
	FaultString   string `json:"fault_string"`
}

// enrichFaultMsgWithSPV 把每个 per-path fault 拼到外层 fault msg 上，让
// device_tasks.error_message / MML 终端日志能定位真正出错的 path。
//
// 输入:
//   - base: 已经合并过 [soapFaultCode] 前缀的 outer fault msg
//     （如 "[Client] Invalid arguments"）
//   - faults: extractSPVFaults 解出的 per-path 详情
//
// 输出形如:
//
//	"[Client] Invalid arguments — path faults: [Device.DeviceInfo.UserLabel: 9005 AttributeIdNotFound : Device.DeviceInfo.UserLabel]"
func enrichFaultMsgWithSPV(base string, faults []SPVFault) string {
	if len(faults) == 0 {
		return base
	}
	var b strings.Builder
	b.WriteString(base)
	b.WriteString(" — path faults:")
	for _, f := range faults {
		fmt.Fprintf(&b, " [%s: %d %s]", f.ParameterName, f.FaultCode, f.FaultString)
	}
	return b.String()
}

// extractSPVFaults 从 SOAP Fault body 抽取所有 <SetParameterValuesFault> 详情块。
// 非 SPV fault（如 GPV / Download 的 fault）→ 返 nil。
func extractSPVFaults(body []byte) []SPVFault {
	blocks := spvFaultBlockRegex.FindAllSubmatch(body, -1)
	if len(blocks) == 0 {
		return nil
	}
	out := make([]SPVFault, 0, len(blocks))
	for _, b := range blocks {
		inner := b[1]
		f := SPVFault{}
		if m := spvFaultNameRegex.FindSubmatch(inner); m != nil {
			f.ParameterName = strings.TrimSpace(string(m[1]))
		}
		if m := cwmpFaultCodeRegex.FindSubmatch(inner); m != nil {
			if n, err := strconv.Atoi(strings.TrimSpace(string(m[1]))); err == nil {
				f.FaultCode = n
			}
		}
		if m := cwmpFaultStringRegex.FindSubmatch(inner); m != nil {
			f.FaultString = strings.TrimSpace(string(m[1]))
		}
		if f.ParameterName != "" {
			out = append(out, f)
		}
	}
	return out
}

// detectSOAPFault 解析 CPE 返回的 SOAP Fault，分别返回四层信息：
//
//   - cwmpCode：内层 <detail><cwmp:Fault><FaultCode>9xxx</...> 数值（CWMP 标准）
//   - soapCode：外层 <faultcode>...</faultcode> 文本（SOAP 1.1 标准，如 "Server.Internal" /
//     "Client" / "Client.InvalidParameter"，部分厂商如 baicells 只返回这个不带 cwmp:FaultCode）
//   - faultString：人类可读描述（优先 cwmp:FaultString，退化到 soap:faultstring，再退化到 soapCode）
//   - badPath：从 FaultString 文本里抽出的"坏 path"（CWMP 协议未规范该字段，厂商私有
//     约定，如 BLQ 把 path 拼在 "Invalid Parameter Names [1], including: ..."）。空表示未识别。
//
// 历史教训：原版只返回 (bool, int, string)，丢弃外层 soapCode。当 CPE 返回
//
//	<soap:Fault>
//	  <faultcode>Server.Internal</faultcode>
//	  <faultstring>RPC handler failed: Empty parameter list</faultstring>
//	</soap:Fault>
//
// 这种 SOAP 1.1 fault（无 cwmp:Fault 块），cwmpCode 解出 0，调用方误判成功，sub_task
// 卡 in_progress；现在把 soapCode 单独返回，调用方可以放进 error_message / event payload，
// 业务层看到 "Server.Internal" 立即知道 CPE 内部错而非协议错。
func detectSOAPFault(body []byte) (found bool, cwmpCode int, soapCode string, faultString string, badPath string) {
	if !faultEnvRegex.Match(body) {
		return false, 0, "", "", ""
	}

	faultString = "SOAP fault"

	if m := cwmpFaultCodeRegex.FindSubmatch(body); m != nil {
		if n, err := strconv.Atoi(strings.TrimSpace(string(m[1]))); err == nil {
			cwmpCode = n
		}
	}
	if m := soapFaultCodeRegex.FindSubmatch(body); m != nil {
		soapCode = strings.TrimSpace(string(m[1]))
	}
	if m := cwmpFaultStringRegex.FindSubmatch(body); m != nil {
		faultString = strings.TrimSpace(string(m[1]))
	} else if m := soapFaultStringRegex.FindSubmatch(body); m != nil {
		faultString = strings.TrimSpace(string(m[1]))
	} else if soapCode != "" {
		// 没有任何 faultstring 时，至少把 soap 外层 code 文本作为 message 返回
		faultString = soapCode
	}

	badPath = extractBadPathFromFaultString(faultString)
	return true, cwmpCode, soapCode, faultString, badPath
}

// extractBadPathFromFaultString 从 CWMP FaultString 文本里抽取 CPE 不支持的参数 path。
//
// 厂商实测格式（CWMP 协议未规范该字段，是私有约定）：
//
//	"Invalid Parameter Names [1], including: Device.Services.FAPService.2....LTECell."
//	"Invalid parameter name: Device.X.Y"
//	"Parameter 'Device.X.Y' is not supported"
//
// 抽取策略：
//  1. 优先抓 "including:" / "parameter:" / "name:" 关键字后第一个 path-like token
//  2. 兜底找最长 dot-separated 标识符（>=3 段，可能尾点）
//
// 末尾 "." 表示对象前缀（CPE 让该对象枚举实例），与 SyncService 入队的格式一致，
// 必须保留以便后续从 batch 准确剔除。
func extractBadPathFromFaultString(faultString string) string {
	if faultString == "" {
		return ""
	}
	if m := faultPathKeywordRegex.FindStringSubmatch(faultString); m != nil {
		return strings.TrimRight(strings.TrimSpace(m[1]), ",;")
	}
	matches := faultPathGenericRegex.FindAllString(faultString, -1)
	var longest string
	for _, c := range matches {
		if len(c) > len(longest) {
			longest = c
		}
	}
	return strings.TrimRight(longest, ",;")
}

// isLocalhost 检查 URL 是否包含 localhost 或 127.0.0.1
func isLocalhost(url string) bool {
	return strings.Contains(url, "localhost") ||
		strings.Contains(url, "127.0.0.1") ||
		strings.Contains(url, "[::1]") ||
		strings.Contains(url, "::1")
}

// hasExpeditedEventParams returns true if any parameter belongs to the
// Device.FaultMgmt.ExpeditedEvent.* subtree.
func hasExpeditedEventParams(params []tr069.ParameterValueStruct) bool {
	for _, p := range params {
		if isExpeditedEventParamName(p.Name) {
			return true
		}
	}
	return false
}

// filterExpeditedEventParams returns only parameters belonging to the
// Device.FaultMgmt.ExpeditedEvent.* subtree.
func filterExpeditedEventParams(params []tr069.ParameterValueStruct) []tr069.ParameterValueStruct {
	filtered := make([]tr069.ParameterValueStruct, 0, len(params))
	for _, p := range params {
		if isExpeditedEventParamName(p.Name) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func isExpeditedEventParamName(name string) bool {
	return strings.HasPrefix(name, "Device.FaultMgmt.ExpeditedEvent.") ||
		strings.HasPrefix(name, "InternetGatewayDevice.FaultMgmt.ExpeditedEvent.")
}

// translateTaskParamsInPlace 在 ACS 出队时把 task.Params 内的 standardPath 翻译为 privatePath。
//
// T-XXX 改造：翻译职责从 App 端 mml fanout 迁移到 ACS。device_tasks.tr069_params 现在
// 入队时存的是 standardPath；ACS 出队时按 device.product_class → product_class_patterns →
// (discovered|default)_param_mappings 链路翻译为 privatePath 后下发。
//
// 失败语义（任一前置依赖未注入或任一步失败）→ task.Params 原样保留，与改造前行为一致。
// helper 设计为 nil-safe + 副作用本地化：直接覆写 t.Params。
func (h *Handler) translateTaskParamsInPlace(ctx context.Context, t *task.Task, log *zap.Logger) {
	if h.pathTranslator == nil || !h.pathTranslator.Enabled() || t == nil {
		return
	}
	translated, changed := h.pathTranslator.TranslateTaskParams(ctx, t)
	if changed {
		t.Params = translated
		log.Debug("ACS path translation applied",
			zap.String("task_id", t.ID),
			zap.String("device_sn", t.DeviceSN),
			zap.String("method", t.Method))
	}
}
