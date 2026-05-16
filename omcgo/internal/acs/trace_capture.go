package acs

import (
	"time"

	"github.com/omcgo/omcgo/internal/acs/rpclog"
	"github.com/omcgo/omcgo/internal/trace"
)

// maybeCaptureTrace 在 protocolLogger 之后旁路投递 trace 报文。
//
// 性能要求：M1 阶段允许在请求 defer 中执行，但要求 < 1ms。
// 实现：sync.Map.Lookup（O(1)）+ 非阻塞 channel send；队列满时丢弃并打点。
//
// 一次 CWMP 请求/响应配对产生 2 条报文（direction=in 是 CPE→ACS 的请求，
// direction=out 是 ACS→CPE 的响应）。
func (h *Handler) maybeCaptureTrace(entry *rpclog.LogEntry, httpStatus int, reqXML, respXML string) {
	if h.traceWhitelist == nil || h.traceService == nil {
		return
	}
	if entry == nil || entry.DeviceSN == "" {
		return
	}
	// M3-01：测量 hook 总耗时（lookup + enqueue × 2），目标 P99 < 5ms。
	hookStart := time.Now()
	defer func() {
		h.traceService.ObserveCaptureLatency(time.Since(hookStart))
	}()
	taskID, ok := h.traceWhitelist.Lookup(entry.DeviceSN)
	if !ok {
		return
	}
	captured := time.Now()
	// 请求方向（CPE → ACS）
	h.traceService.EnqueueCapture(&trace.Message{
		CapturedAt:       captured,
		TaskID:           taskID,
		DeviceSN:         entry.DeviceSN,
		Direction:        trace.DirectionIn,
		RPCMethod:        entry.Method,
		CwmpID:           entry.CwmpID,
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
		RPCMethod:        entry.Method,
		CwmpID:           entry.CwmpID,
		SessionID:        entry.SessionID,
		HTTPStatus:       httpStatus,
		PayloadSizeBytes: len(respXML),
		PayloadInline:    truncateForInline(respXML),
	})
}

// truncateForInline M1 阶段：超过 MaxInlinePayloadBytes 直接尾截断（M2 改为外置 MinIO）。
func truncateForInline(xml string) string {
	if len(xml) <= trace.MaxInlinePayloadBytes {
		return xml
	}
	return xml[:trace.MaxInlinePayloadBytes] + "...(truncated)"
}
