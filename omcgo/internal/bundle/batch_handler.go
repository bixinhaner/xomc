package bundle

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// BatchDownloadRequest 是 POST /{module}/batch-download 的统一请求体。
// firmware 用 `ids`(UUID 列表),其它三个用 `serial_numbers`(SN 列表)。
type BatchDownloadRequest struct {
	IDs           []string `json:"ids,omitempty"`
	SerialNumbers []string `json:"serial_numbers,omitempty"`
}

// NewBatchDownloadHandler 工厂：返回 gin.HandlerFunc,流式打 zip 到响应体。
//
// 浏览器侧用 axios responseType:'blob' + Content-Disposition 自动触发下载。
// 不存任何中间产物,不开后台 goroutine,不签 presigned URL。
//
// 注意：响应必须 keep-alive,nginx proxy_read_timeout 要够大(我们 default.conf
// 已经 60s,大批量 GB 级 zip 可能需要 5min+,handler 先 Flush 0 字节让 nginx
// 不超时,后续 chunk 持续 keepalive)。
func NewBatchDownloadHandler(svc *Service, module Module, resolver *authz.Resolver) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req BatchDownloadRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			commonerrors.AbortWithError(c, http.StatusBadRequest, err)
			return
		}
		targets := req.IDs
		if len(targets) == 0 {
			targets = req.SerialNumbers
		}
		if len(targets) == 0 {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("empty target list (ids / serial_numbers)"))
			return
		}
		// #63 目标数上限：超大请求直接 400，避免拖垮 MinIO 拉取与 zip 流。
		if len(targets) > MaxBatchTargets {
			commonerrors.AbortWithError(c, http.StatusBadRequest,
				fmt.Errorf("%w: too many targets %d (max %d)", commonerrors.ErrInvalidInput, len(targets), MaxBatchTargets))
			return
		}
		// #63 设备组可见性：解析调用者可见设备组；解析失败已 abort（403/500），直接 return。
		visibleGroups, ok := resolver.FromContext(c)
		if !ok {
			return
		}

		filename := fmt.Sprintf("%s-bundle-%s.zip", module, time.Now().Format("20060102-150405"))
		c.Header("Content-Type", "application/zip")
		c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
		// 关键 1: X-Accel-Buffering: no — nginx 看到这个 header 会对当前响应禁用
		// proxy_buffering,zip 的每个 chunk 立刻透传给浏览器。否则 nginx 默认攒
		// 满 ~32KB-MB 才推一次,前端 onDownloadProgress 看到的字节数会大块大块跳
		// (实测从 0 → 22MB → 长时间不动 → 401MB)。
		c.Header("X-Accel-Buffering", "no")
		// 关键 2: 先 WriteHeader 让浏览器开始拿数据,后续 Write 就是 chunked transfer。
		c.Status(http.StatusOK)
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}

		count, err := svc.WriteZipTo(c.Request.Context(), module, targets, visibleGroups, c.Writer)
		if err != nil {
			// 已经 WriteHeader 了,这里只能记日志,后续浏览器下载到的 zip 不完整
			svc.logger.Warn("batch download stream failed",
				zap.String("module", string(module)),
				zap.Int("targets", len(targets)),
				zap.Error(err))
			return
		}
		svc.logger.Info("batch download completed",
			zap.String("module", string(module)),
			zap.Int("targets", len(targets)),
			zap.Int("zipped", count))
	}
}
