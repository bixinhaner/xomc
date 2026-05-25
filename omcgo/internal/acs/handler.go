package acs

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
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
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/internal/trace"
	"github.com/omcgo/omcgo/pkg/soap"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

// ConnectionRequester 发送 Connection Request 唤醒设备。
// 用于会话结束后命令队列仍有待执行命令时的续唤机制。
type ConnectionRequester interface {
	// Send 发送 Connection Request。httpURL 为设备的 HTTP CR URL（可能为空）。
	Send(ctx context.Context, deviceSN, httpURL string) error
}

// SessionCookieName TR069 会话 ID 的 Cookie 名称
const SessionCookieName = "SESSION"

// connSessionEntry 连接级会话绑定，记录创建时间用于 TTL 清理。
type connSessionEntry struct {
	DeviceSN  string
	CreatedAt time.Time
}

// deviceSessionEntry 跟踪设备的活跃会话。
// 用于检测和清理孤儿会话：当新 Inform 到达但前一个会话尚未完成时
// （如 CPE 未响应 RPC、会话中途重启、或周期上报定时器触发），清理旧会话。
type deviceSessionEntry struct {
	SessionID string
	DeviceSN  string
	CreatedAt time.Time
}

// Handler 处理 TR069/CWMP HTTP 请求。
type Handler struct {
	sessionStore            SessionStore
	taskService             TaskService // 统一任务管理服务
	eventBus                event.EventBus
	authenticator           auth.DeviceAuthenticator
	rpcDispatcher           *rpc.Dispatcher
	rateLimiter             *DeviceRateLimiter
	admission               *AdmissionController
	metrics                 *ACSMetrics
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
	stunStore          *stun.Store   // 缓存 Inform 中的设备 STUN 地址
	connReqURLCache    sync.Map      // deviceSN → ConnectionRequestURL（来自 Inform）
	// protocolLogger 独立的协议交互日志器，记录完整的原始 XML 请求/响应到专用文件。
	// nil 表示协议日志关闭。
	protocolLogger *zap.Logger
	maxBodySize    int // 协议日志 XML 截断阈值（0=不截断）
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
	// deviceSessions 映射 deviceSN → *deviceSessionEntry，用于设备级会话追踪。
	// 用于检测孤儿会话：当新 Inform 到达时，同一设备的已有会话会通过 completeSession() 清理。
	deviceSessions sync.Map
}

// sessionRPCLimitReached 判断会话是否已达到单会话 RPC 上限。
func (h *Handler) sessionRPCLimitReached(session *Session) bool {
	return h.maxRPCPerSession > 0 && session.RPCCount >= h.maxRPCPerSession
}

// startSessionReaper 启动一个后台 goroutine，定期清理
// connSessions 和 deviceSessions 映射表中的过期会话。
// 对于 connSessions：释放准入槽位（admission slots）并减少指标计数。
// 对于 deviceSessions：从 Redis 加载孤立会话，调用 completeSession()
// （如果队列中仍有剩余命令，该调用会触发 postSessionWake），然后执行清理工作。
func (h *Handler) startSessionReaper(interval, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			// 清理过期的连接级会话。
			h.connSessions.Range(func(key, value interface{}) bool {
				entry := value.(connSessionEntry)
				if now.Sub(entry.CreatedAt) > maxAge {
					h.connSessions.Delete(key)
					h.admission.Release()
					h.metrics.ActiveSessions.Dec()
					h.metrics.SessionDuration.Observe(now.Sub(entry.CreatedAt).Seconds())
					h.logger.Warn("reaped stale conn session",
						zap.String("device_sn", entry.DeviceSN),
						zap.String("remote_addr", key.(string)),
						zap.Duration("age", now.Sub(entry.CreatedAt)))
				}
				return true
			})
			// 清理过期的设备级会话（孤儿会话）。
			// 这是安全网：处理因 CPE 未响应 RPC 或会话中途重启而未完成的会话。
			h.deviceSessions.Range(func(key, value interface{}) bool {
				entry := value.(*deviceSessionEntry)
				if now.Sub(entry.CreatedAt) > maxAge {
					h.deviceSessions.Delete(key)
					h.reapOrphanedSession(entry, "reaper")
				}
				return true
			})
		}
	}()
}

// reapOrphanedSession 清理孤儿设备会话。
// 从 Redis 加载会话（如果仍存在），调用 completeSession 释放资源并触发 postSessionWake。
func (h *Handler) reapOrphanedSession(entry *deviceSessionEntry, reason string) {
	ctx := context.Background()

	// 尝试从 Redis 加载会话，获取完整会话数据用于 completeSession。
	session, err := h.sessionStore.GetByID(ctx, entry.SessionID)
	if err != nil {
		h.logger.Warn("reap orphaned session: failed to load from store",
			zap.String("device_sn", entry.DeviceSN),
			zap.String("session_id", entry.SessionID),
			zap.String("reason", reason),
			zap.Error(err))
	}

	if session != nil {
		h.logger.Info("reap orphaned session: completing",
			zap.String("device_sn", entry.DeviceSN),
			zap.String("session_id", entry.SessionID),
			zap.String("old_state", string(session.State)),
			zap.Duration("age", time.Since(entry.CreatedAt)),
			zap.String("reason", reason))
		h.completeSession(ctx, session)
	} else {
		// 会话已在 Redis 中过期（TTL）。仍需释放准入槽位和更新指标。
		h.logger.Info("reap orphaned session: session expired in store, releasing resources",
			zap.String("device_sn", entry.DeviceSN),
			zap.String("session_id", entry.SessionID),
			zap.Duration("age", time.Since(entry.CreatedAt)),
			zap.String("reason", reason))
		h.admission.Release()
		h.metrics.ActiveSessions.Dec()
		h.metrics.SessionDuration.Observe(time.Since(entry.CreatedAt).Seconds())
		// 仍触发 postSessionWake —— 即使会话已消失，设备队列中可能仍有待执行命令。
		if h.connReqSender != nil && h.postSessionWakeCfg.Enabled && entry.DeviceSN != "" {
			go h.postSessionWake(entry.DeviceSN)
		}
	}
}

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

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("read request body", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 协议日志：用 ResponseCapturer 包装 ResponseWriter，捕获响应字节
	if h.protocolLogger != nil {
		capturer := rpclog.NewResponseCapturer(w)
		w = capturer
		logEntry := &rpclog.LogEntry{StartTime: time.Now()}
		ctx = rpclog.WithEntry(ctx, logEntry)
		r = r.WithContext(ctx)

		defer func() {
			reqXML := string(body)
			respXML := string(capturer.Body())
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

			// T-0137 / M1: 命中跟踪白名单则旁路落库
			h.maybeCaptureTrace(logEntry, capturer.StatusCode(), reqXML, respXML)
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
	if isFault, faultCode, soapFaultCode, faultMsg := detectSOAPFault(body); isFault {
		h.handleSOAPFault(w, r, body, faultCode, soapFaultCode, faultMsg, log)
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
		soap.MethodFactoryResetResp:
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

	// 清理该设备的孤儿会话（如果存在）。
	// 当 CPE 在前一个会话仍挂起时发送新 Inform（如 CPE 未响应 RPC、重启或周期上报定时器触发），
	// 旧会话永远不会完成。在此检测并清理，以：
	// 1) 释放旧的准入槽位（防止槽位泄漏）
	// 2) 触发 postSessionWake 处理队列中剩余命令
	// 3) 保持 ActiveSessions 指标准确
	if old, loaded := h.deviceSessions.LoadAndDelete(deviceSN); loaded {
		oldEntry := old.(*deviceSessionEntry)
		log.Info("cleaning orphaned session before new Inform",
			zap.String("device_sn", deviceSN),
			zap.String("old_session_id", oldEntry.SessionID),
			zap.Duration("old_session_age", time.Since(oldEntry.CreatedAt)))
		h.reapOrphanedSession(oldEntry, "new_inform")
	}

	// 僵死任务恢复（docs/消息队列全流程流转说明书.md §4.3.2）。
	// CPE 重连即视为"在线信号"——把该设备上 status=sent 且 sent_at>5min 的任务
	// 按 CanRetry() 重置 pending 重入队 / 或标记 failed；否则这些任务会因
	// CPE 网络波动 / RPC 丢包 / 设备重启而永久悬挂在 sent 状态（v1.1 §6 P0 短板）。
	// 同步执行：单设备 indexed query (device_sn + status + sent_at)，亚毫秒级；
	// 失败不阻塞 InformResponse，仅 warn 留痕。
	if err := h.taskService.RecoverPendingTasks(r.Context(), deviceSN); err != nil {
		log.Warn("recover pending tasks failed (non-blocking)",
			zap.String("device_sn", deviceSN),
			zap.Error(err))
	}

	// 准入控制 —— 槽位持有直到会话完成（通过 completeSession）或后台清理器回收。
	if !h.admission.Acquire() {
		log.Warn("admission denied", zap.String("device_sn", deviceSN))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	// 跟踪活跃会话 —— 将由 completeSession() 或清理器递减。
	h.metrics.ActiveSessions.Inc()

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
		if strings.HasSuffix(p.Name, ".UDPConnectionRequestAddress") && p.Value != "" {
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
			h.connReqURLCache.Store(deviceSN, p.Value)
			log.Debug("cached ConnectionRequestURL from Inform",
				zap.String("device_sn", deviceSN),
				zap.String("cr_url", p.Value))
		}
	}

	// 使用新 Session ID 创建/更新会话
	sessionID := generateSessionID()
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

	// 注册设备 → 会话映射，用于孤儿会话检测。
	h.deviceSessions.Store(deviceSN, &deviceSessionEntry{
		SessionID: sessionID,
		DeviceSN:  deviceSN,
		CreatedAt: time.Now(),
	})

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

	// 尝试从统一任务队列获取下一个任务
	taskItem, err := h.taskService.PopTask(r.Context(), deviceSN)
	if err != nil {
		log.Error("pop task from queue", zap.Error(err))
	} else if taskItem != nil {
		// standardPath → privatePath 翻译（T-XXX：翻译职责从 App fanout 迁移到 ACS）
		h.translateTaskParamsInPlace(r.Context(), taskItem, log)
		// 为此任务生成 CWMP ID
		cwmpID := task.GenerateCWMPID(taskItem.Method)

		// 标记任务已发送
		if err := h.taskService.MarkTaskSent(r.Context(), taskItem.ID, cwmpID); err != nil {
			log.Error("mark task sent", zap.Error(err), zap.String("task_id", taskItem.ID))
		}

		// 更新会话状态
		session.State = StateRPCPending
		session.LastRPC = taskItem.Method
		session.LastCommandParams = taskItem.Params
		session.LastTaskID = taskItem.ID
		session.LastTaskCWMPID = cwmpID
		session.RPCCount++
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)

		// 使用 CWMP ID 构建 RPC 请求
		cmd := &rpc.Command{
			ID:         taskItem.ID,
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
	var taskItem *task.Task
	if cwmpID != "" {
		var err error
		taskItem, err = h.taskService.GetTaskByCWMPID(r.Context(), cwmpID)
		if err != nil {
			log.Warn("get task by cwmp_id", zap.Error(err), zap.String("cwmp_id", cwmpID))
		} else if taskItem != nil {
			// 检查响应中是否有 SOAP Fault
			if isFault, faultCode, soapFaultCode, faultMsg := detectSOAPFault(body); isFault {
				// 把 SOAP 1.1 outer faultcode（如 "Server.Internal"）合进 error message，
				// 否则丢失诊断信息；DB 落盘的 error_message 也能完整呈现。
				combinedMsg := faultMsg
				if soapFaultCode != "" && !strings.Contains(faultMsg, soapFaultCode) {
					combinedMsg = fmt.Sprintf("[%s] %s", soapFaultCode, faultMsg)
				}
				if markErr := h.taskService.MarkTaskFailed(r.Context(), taskItem.ID, faultCode, combinedMsg); markErr != nil {
					log.Error("mark task failed", zap.Error(markErr), zap.String("task_id", taskItem.ID))
				}
				log.Warn("task failed with SOAP fault",
					zap.String("task_id", taskItem.ID),
					zap.Int("fault_code", faultCode),
					zap.String("soap_fault_code", soapFaultCode),
					zap.String("fault_msg", faultMsg))
			} else {
				// 任务成功完成 - 将原始响应存为结果
				resultMap := map[string]interface{}{
					"method":       string(method),
					"raw_response": string(body),
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
			}
		}
	}

	// 发布 RPC 响应事件到开通引擎（包含原始 body 用于 GPN/GPV 处理）。
	h.publishRPCResponseEvent(r.Context(), deviceSN, method, body, session.LastCommandParams, taskItem, log)

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

	// 尝试从统一任务队列获取下一个任务
	nextTask, err := h.taskService.PopTask(r.Context(), deviceSN)
	if err != nil {
		log.Error("pop task from queue", zap.Error(err))
	} else if nextTask != nil {
		h.translateTaskParamsInPlace(r.Context(), nextTask, log)
		// 为此任务生成 CWMP ID
		newCWMPID := task.GenerateCWMPID(nextTask.Method)

		// 标记任务已发送
		if err := h.taskService.MarkTaskSent(r.Context(), nextTask.ID, newCWMPID); err != nil {
			log.Error("mark task sent", zap.Error(err), zap.String("task_id", nextTask.ID))
		}

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

// completeSession 完成 TR069 会话，释放所有相关资源。
// 根据 Session.ID 删除 Redis 中的会话数据。
func (h *Handler) completeSession(ctx context.Context, session *Session) {
	// 释放准入槽位
	h.admission.Release()

	// 递减活跃会话计数
	h.metrics.ActiveSessions.Dec()

	if session == nil {
		return
	}

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

	// 清除设备→会话映射。仅当映射的 SessionID 与当前 session 一致时才删除，
	// 避免误删已被新 Inform 覆盖的映射。
	if session.DeviceSN != "" && session.ID != "" {
		if v, ok := h.deviceSessions.Load(session.DeviceSN); ok {
			if entry := v.(*deviceSessionEntry); entry.SessionID == session.ID {
				h.deviceSessions.Delete(session.DeviceSN)
			}
		}
	}

	// 异步检查队列并续唤设备
	if h.connReqSender != nil && h.postSessionWakeCfg.Enabled && session.DeviceSN != "" {
		go h.postSessionWake(session.DeviceSN)
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

	// 发送 Connection Request。
	// 查找缓存的 ConnectionRequestURL 作为 HTTP 回退。
	var httpURL string
	if v, ok := h.connReqURLCache.Load(deviceSN); ok {
		httpURL, _ = v.(string)
	}

	// 使用短超时，避免在不可达设备上阻塞。
	crCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.connReqSender.Send(crCtx, deviceSN, httpURL); err != nil {
		h.logger.Warn("post-session wake: CR failed",
			zap.String("device_sn", deviceSN),
			zap.Int64("remaining", remaining),
			zap.Error(err))
	} else {
		h.logger.Info("post-session wake: CR sent",
			zap.String("device_sn", deviceSN),
			zap.Int64("remaining", remaining),
			zap.Int64("continuous_count", count))
		h.metrics.PostSessionWakeTotal.Inc()
	}
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
// 通过 CWMP ID 查找关联任务并标记为失败。
//
// 参数：
//   - faultCode     — 内层 cwmp:FaultCode 数值（标准 CWMP fault，如 9005=Invalid parameter name）
//   - soapFaultCode — 外层 soap:faultcode 文本（如 "Server.Internal"），部分厂商只回这个不带 cwmp:Fault
//   - faultMsg      — 人类可读 faultstring
func (h *Handler) handleSOAPFault(w http.ResponseWriter, r *http.Request, body []byte, faultCode int, soapFaultCode, faultMsg string, log *zap.Logger) {
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
		if markErr := h.taskService.MarkTaskFailed(r.Context(), taskItem.ID, faultCode, combinedFaultMsg); markErr != nil {
			log.Error("mark task failed on fault", zap.Error(markErr), zap.String("task_id", taskItem.ID))
		}
		log.Info("task marked as failed on SOAP fault",
			zap.String("task_id", taskItem.ID),
			zap.String("method", taskItem.Method),
			zap.Int("fault_code", faultCode),
			zap.String("soap_fault_code", soapFaultCode),
			zap.String("fault_msg", faultMsg))

		// 同步发出对应方法的 command.*.response 事件，把 fault_code / fault_string 透传给业务订阅者。
		// 否则像 software.HandleUploadResponse / HandleSetParamsResponseForCollect 这种按
		// "RPC 失败 → 立即 fail 子任务"的 handler 永远收不到 SOAP Fault，sub_task 卡 Uploading
		// 直到 reaper 兜底（半小时级延迟）。fault_msg 用 combinedFaultMsg —— 即包含外层
		// soap:faultcode 前缀的版本，让业务层能完整呈现「[Server.Internal] RPC handler failed: ...」。
		h.publishRPCFaultEvent(r.Context(), session.DeviceSN, taskItem, faultCode, soapFaultCode, combinedFaultMsg, log)
	} else {
		log.Warn("SOAP fault but no task could be associated (neither cwmp_id nor session.last_task_id matched)",
			zap.String("cwmp_id", cwmpID),
			zap.String("device_sn", session.DeviceSN))
	}

	// 检查队列中是否有更多任务
	deviceSN := session.DeviceSN
	nextTask, err := h.taskService.PopTask(r.Context(), deviceSN)
	if err != nil {
		log.Error("pop next task after fault", zap.Error(err))
	} else if nextTask != nil {
		h.translateTaskParamsInPlace(r.Context(), nextTask, log)
		newCWMPID := task.GenerateCWMPID(nextTask.Method)
		if err := h.taskService.MarkTaskSent(r.Context(), nextTask.ID, newCWMPID); err != nil {
			log.Error("mark next task sent", zap.Error(err), zap.String("task_id", nextTask.ID))
		}

		session.State = StateRPCPending
		session.LastRPC = nextTask.Method
		session.LastTaskID = nextTask.ID
		session.LastTaskCWMPID = newCWMPID
		session.UpdatedAt = time.Now()
		h.sessionStore.UpdateByID(r.Context(), sessionID, session)

		cmd := &rpc.Command{
			ID:         nextTask.ID,
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

	// 发布事件
	evt, _ := event.NewEvent(event.SubjectDeviceTransferComplete, tc)
	h.eventBus.Publish(r.Context(), event.SubjectDeviceTransferComplete, evt)

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
	// 构建事件载荷
	payload := map[string]interface{}{
		"device_id":      inform.DeviceId,
		"events":         eventCodes,
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
		"command_key":     taskItem.CommandKey,
		"fault_code":      faultCode,        // 数值：cwmp:FaultCode（标准 CWMP），无则 0
		"fault_code_text": soapFaultCode,    // 字符串：soap:faultcode（SOAP 1.1 outer，如 "Server.Internal"）
		"fault_string":    faultMsg,         // 已含 [soapFaultCode] 前缀的人类可读消息
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
)

// detectSOAPFault 解析 CPE 返回的 SOAP Fault，分别返回三层信息：
//
//   - cwmpCode：内层 <detail><cwmp:Fault><FaultCode>9xxx</...> 数值（CWMP 标准）
//   - soapCode：外层 <faultcode>...</faultcode> 文本（SOAP 1.1 标准，如 "Server.Internal" /
//     "Client" / "Client.InvalidParameter"，部分厂商如 baicells 只返回这个不带 cwmp:FaultCode）
//   - faultString：人类可读描述（优先 cwmp:FaultString，退化到 soap:faultstring，再退化到 soapCode）
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
func detectSOAPFault(body []byte) (found bool, cwmpCode int, soapCode string, faultString string) {
	if !faultEnvRegex.Match(body) {
		return false, 0, "", ""
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

	return true, cwmpCode, soapCode, faultString
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
		if strings.HasPrefix(p.Name, "Device.FaultMgmt.ExpeditedEvent.") {
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
		if strings.HasPrefix(p.Name, "Device.FaultMgmt.ExpeditedEvent.") {
			filtered = append(filtered, p)
		}
	}
	return filtered
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
