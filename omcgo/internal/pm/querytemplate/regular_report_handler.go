package querytemplate

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

func (h *Handler) GetRegularReport(c *gin.Context) {
	template, ok := h.authorizeRegularReport(c, false)
	if !ok {
		return
	}
	report, err := h.regularReports.Get(c.Request.Context(), template.ID)
	if err != nil {
		commonerrors.AbortWithError(c, commonerrors.HTTPStatusFromError(err), err)
		return
	}
	c.Header("ETag", fmt.Sprintf(`"%d"`, report.Revision))
	response.OK(c, report)
}

func (h *Handler) UpdateRegularReport(c *gin.Context) {
	template, ok := h.authorizeRegularReport(c, true)
	if !ok {
		return
	}
	revision, err := parseReportIfMatch(c.GetHeader("If-Match"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusPreconditionRequired, err)
		return
	}
	var input RegularReportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	report, err := h.regularReports.Update(c.Request.Context(), template.ID, revision, input)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		if errors.Is(err, ErrReportRevisionMismatch) {
			status = http.StatusPreconditionFailed
		} else if errors.Is(err, ErrRegularReportRunnerNotReady) {
			status = http.StatusConflict
		}
		commonerrors.AbortWithError(c, status, err)
		return
	}
	c.Header("ETag", fmt.Sprintf(`"%d"`, report.Revision))
	response.OK(c, report)
}

func (h *Handler) authorizeRegularReport(c *gin.Context, write bool) (*Template, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return nil, false
	}
	template, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrNotFound) {
			status = http.StatusNotFound
		}
		commonerrors.AbortWithError(c, status, err)
		return nil, false
	}
	callerID, superAdmin := callerInfo(c)
	allowed := canRead(template, callerID, superAdmin)
	if write {
		allowed = canWrite(template, callerID, superAdmin)
	}
	if !allowed {
		response.Fail(c, http.StatusForbidden, "no permission for this template regular report")
		return nil, false
	}
	return template, true
}

func parseReportIfMatch(value string) (int64, error) {
	value = strings.Trim(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "W/")), `"`)
	if value == "" {
		return 0, fmt.Errorf("If-Match revision is required")
	}
	revision, err := strconv.ParseInt(value, 10, 64)
	if err != nil || revision < 1 {
		return 0, fmt.Errorf("invalid If-Match revision")
	}
	return revision, nil
}
