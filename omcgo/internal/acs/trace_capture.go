package acs

import (
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/acs/rpclog"
	"github.com/omcgo/omcgo/internal/trace"
	"github.com/omcgo/omcgo/pkg/soap"
)

// traceCaptureSink 抽象 trace.Service 中本 hook 需要的两个方法，便于单元测试注入 stub。
// 生产实现：*trace.Service。
type traceCaptureSink interface {
	EnqueueCapture(msg *trace.Message)
	ObserveCaptureLatency(d time.Duration)
}

// maybeCaptureTrace 在 protocolLogger 之后旁路投递 trace 报文。
//
// 性能要求：M1 阶段允许在请求 defer 中执行，目标 P99 < 5ms。
//
// 一次 HTTP 事务的请求体（CPE→ACS）与响应体（ACS→CPE）在 CWMP 协议层是**两条独立 RPC 消息**，
// 方法名通常不同（如请求体是 GetParameterValuesResponse，响应体是新下发的 GetParameterValues）。
// 因此 in / out 两条 trace.Message 的 RPCMethod 必须分别从各自的 XML body 解析，而不是共用
// rpclog.LogEntry.Method（后者在请求处理过程中只能保存一个值，会被覆盖）。
//
// 空 body（CPE 续会话的空 POST、ACS 会话结束的 204 NoContent）也照常入库，便于在 trace 视图
// 上看到 HTTP 事务的完整边界；此时 RPCMethod 写 "Empty"（与 handler.go 中 entry.Method="Empty"
// 的既有语义一致），与"XML 解析失败"（写空字符串）区分开。
func (h *Handler) maybeCaptureTrace(entry *rpclog.LogEntry, httpStatus int, reqXML, respXML string) {
	if h.traceWhitelist == nil || h.traceService == nil {
		return
	}
	if entry == nil || entry.DeviceSN == "" {
		return
	}
	hookStart := time.Now()
	defer func() {
		h.traceService.ObserveCaptureLatency(time.Since(hookStart))
	}()
	taskID, ok := h.traceWhitelist.Lookup(entry.DeviceSN)
	if !ok {
		return
	}

	inMethod, inCwmpID := detectXMLMeta(reqXML)
	outMethod, outCwmpID := detectXMLMeta(respXML)

	captured := time.Now()
	// 请求方向（CPE → ACS）
	h.traceService.EnqueueCapture(&trace.Message{
		CapturedAt:       captured,
		TaskID:           taskID,
		DeviceSN:         entry.DeviceSN,
		Direction:        trace.DirectionIn,
		RPCMethod:        inMethod,
		CwmpID:           firstNonEmpty(inCwmpID, entry.CwmpID),
		SessionID:        entry.SessionID,
		PayloadSizeBytes: len(reqXML),
		PayloadInline:    truncateForInline(reqXML),
	})
	// 响应方向（ACS → CPE）
	h.traceService.EnqueueCapture(&trace.Message{
		CapturedAt:       captured.Add(time.Microsecond), // 让顺序稳定
		TaskID:           taskID,
		DeviceSN:         entry.DeviceSN,
		Direction:        trace.DirectionOut,
		RPCMethod:        outMethod,
		CwmpID:           firstNonEmpty(outCwmpID, entry.CwmpID),
		SessionID:        entry.SessionID,
		HTTPStatus:       httpStatus,
		PayloadSizeBytes: len(respXML),
		PayloadInline:    truncateForInline(respXML),
	})
}

// detectXMLMeta 从一段 SOAP/CWMP XML 中流式解析方法名和 cwmp:ID。
// 空 body 返回 "Empty"（与 handler.go 中 entry.Method="Empty" 的既有语义一致）。
// 解析失败返回空字符串（区分"协议无 body"与"body 存在但无法解析"两种情况）。
// 注意：不返回 error —— trace 是诊断旁路，不能影响主流程。
func detectXMLMeta(xml string) (method, cwmpID string) {
	if len(xml) == 0 {
		return "Empty", ""
	}
	m, id, _, err := soap.DetectMethod(strings.NewReader(xml))
	if err != nil {
		return "", ""
	}
	return string(m), id
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// truncateForInline M1 阶段：超过 MaxInlinePayloadBytes 直接尾截断（M2 改为外置 MinIO）。
func truncateForInline(xml string) string {
	if len(xml) <= trace.MaxInlinePayloadBytes {
		return xml
	}
	return xml[:trace.MaxInlinePayloadBytes] + "...(truncated)"
}
