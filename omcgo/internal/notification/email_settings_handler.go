package notification

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

type EmailSettingsHandler struct {
	mailer *Mailer
}

func NewEmailSettingsHandler(mailer *Mailer) *EmailSettingsHandler {
	return &EmailSettingsHandler{mailer: mailer}
}

// RegisterRoutes mounts the administrator-only SMTP test endpoint. Settings
// themselves continue to use the audited sys_config batch endpoint.
func (h *EmailSettingsHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/notification/email/test", h.SendTest)
}

type sendTestEmailRequest struct {
	Recipient string `json:"recipient" binding:"required"`
}

func (h *EmailSettingsHandler) SendTest(c *gin.Context) {
	var req sendTestEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	recipients, err := normalizeRecipients([]string{strings.TrimSpace(req.Recipient)})
	if err != nil || len(recipients) != 1 {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			fmt.Errorf("%w: exactly one valid recipient is required", commonerrors.ErrInvalidInput))
		return
	}

	subject := "OMC 邮件通道测试 / Email Channel Test"
	body := fmt.Sprintf("这是一封 OMC 邮件通道测试邮件。\nThis is an OMC email channel test.\n\n发送时间 / Sent at: %s",
		time.Now().Format(time.RFC3339))
	if err := h.mailer.SendRawRecorded(c.Request.Context(), recipients, subject, body, nil); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadGateway, fmt.Errorf("send test email: %w", err))
		return
	}
	response.OKWithMsg(c, gin.H{"recipient": recipients[0]}, "测试邮件发送成功")
}
