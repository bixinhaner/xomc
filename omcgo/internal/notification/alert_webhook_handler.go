package notification

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AlertWebhookOptions 配置 Alertmanager 告警 webhook 入口。
type AlertWebhookOptions struct {
	Token      string   // 非空时校验请求头 Authorization: Bearer <token>
	Recipients []string // 告警邮件收件人；为空则不发（仅记日志）
}

// AlertWebhookHandler 接收 Alertmanager 的 webhook 推送，把告警汇总成一封邮件
// 经 Mailer 投递（并记入 notification_history）。
//
// 这是「告警通知链」的最后一跳之一：Prometheus 规则触发 → Alertmanager 分组 →
// 本 webhook → 邮件。它与 Alertmanager 自带的 email_configs（直发，见 T-0149）
// 互补 —— 走本入口的告警额外获得统一的发送历史与可扩展的通知中心出口。
type AlertWebhookHandler struct {
	mailer *Mailer
	opts   AlertWebhookOptions
	logger *zap.Logger
}

// NewAlertWebhookHandler 创建 AlertWebhookHandler。
func NewAlertWebhookHandler(mailer *Mailer, opts AlertWebhookOptions, logger *zap.Logger) *AlertWebhookHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AlertWebhookHandler{mailer: mailer, opts: opts, logger: logger.Named("alert-webhook")}
}

// RegisterRoutes 注册 POST /alerts/webhook。
// 须挂在【无 JWT】的路由组上 —— Alertmanager 无法携带 JWT；可选的 Bearer token
// 校验由 AlertWebhookOptions.Token 提供。
func (h *AlertWebhookHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/alerts/webhook", h.handle)
}

// alertmanagerPayload 是 Alertmanager webhook v4 报文的子集（仅取所需字段）。
type alertmanagerPayload struct {
	Status string              `json:"status"`
	Alerts []alertmanagerAlert `json:"alerts"`
}

type alertmanagerAlert struct {
	Status      string            `json:"status"` // firing | resolved
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
}

func (h *AlertWebhookHandler) handle(c *gin.Context) {
	// 可选 Bearer token 校验（常量时间比较，防时序侧信道）。
	if h.opts.Token != "" {
		got := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(h.opts.Token)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
	}

	var payload alertmanagerPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alertmanager payload"})
		return
	}
	if len(payload.Alerts) == 0 {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "sent": 0})
		return
	}
	if len(h.opts.Recipients) == 0 {
		h.logger.Warn("alert webhook received but no recipients configured",
			zap.Int("alerts", len(payload.Alerts)),
			zap.String("hint", "set notification.alert_webhook.recipients"))
		c.JSON(http.StatusOK, gin.H{"status": "no_recipients"})
		return
	}

	subject, body := renderAlerts(payload)
	if err := h.mailer.SendRaw(c.Request.Context(), h.opts.Recipients, subject, body, nil); err != nil {
		// 返回 5xx 让 Alertmanager 按其 webhook 重试策略重投 —— 借此获得免费的
		// 重试机制；本次失败已记入 notification_history。
		h.logger.Error("alert email dispatch failed", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "dispatch failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "sent": len(payload.Alerts)})
}

// renderAlerts 把一批 Alertmanager 告警汇总成一封纯文本邮件的主题与正文。
func renderAlerts(p alertmanagerPayload) (subject, body string) {
	var firing, resolved int
	for _, a := range p.Alerts {
		if a.Status == "resolved" {
			resolved++
		} else {
			firing++
		}
	}
	switch {
	case firing > 0 && resolved > 0:
		subject = fmt.Sprintf("[OMC告警] %d 条触发 / %d 条恢复", firing, resolved)
	case resolved > 0:
		subject = fmt.Sprintf("[OMC告警] %d 条恢复", resolved)
	default:
		subject = fmt.Sprintf("[OMC告警] %d 条触发", firing)
	}

	var b strings.Builder
	b.WriteString(subject)
	b.WriteString("\n\n")
	for i, a := range p.Alerts {
		name := a.Labels["alertname"]
		if name == "" {
			name = "(unnamed)"
		}
		state := a.Status
		if state == "" {
			state = "firing"
		}
		b.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, state, name))
		if labels := formatLabels(a.Labels); labels != "" {
			b.WriteString("   标签: " + labels + "\n")
		}
		if s := a.Annotations["summary"]; s != "" {
			b.WriteString("   摘要: " + s + "\n")
		}
		if d := a.Annotations["description"]; d != "" {
			b.WriteString("   详情: " + strings.TrimSpace(d) + "\n")
		}
		if a.StartsAt != "" {
			b.WriteString("   开始: " + a.StartsAt + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("—— OMC 监控（Alertmanager → /api/v1/alerts/webhook）\n")
	return subject, b.String()
}

// formatLabels 把告警标签（除 alertname 外）按键名排序拼成 k=v 串，保证输出稳定。
func formatLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		if k == "alertname" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+labels[k])
	}
	return strings.Join(parts, ", ")
}
