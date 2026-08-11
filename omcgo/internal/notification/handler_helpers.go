package notification

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

var ErrPreconditionRequired = errors.New("If-Match revision is required")

func parseRuleID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return uuid.Nil, false
	}
	return id, true
}

func parseRuleMutation(c *gin.Context) (uuid.UUID, int64, bool) {
	id, ok := parseRuleID(c)
	if !ok {
		return uuid.Nil, 0, false
	}
	revision, err := parseIfMatchRevision(c.GetHeader("If-Match"))
	if errors.Is(err, ErrPreconditionRequired) {
		commonerrors.AbortWithError(c, http.StatusPreconditionRequired, err)
		return uuid.Nil, 0, false
	}
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, commonerrors.ErrInvalidInput)
		return uuid.Nil, 0, false
	}
	return id, revision, true
}

func parseIfMatchRevision(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, ErrPreconditionRequired
	}
	value = strings.TrimPrefix(value, "W/")
	value = strings.Trim(value, `"`)
	revision, err := strconv.ParseInt(value, 10, 64)
	if err != nil || revision < 1 {
		return 0, commonerrors.ErrInvalidInput
	}
	return revision, nil
}

func abortRuleError(c *gin.Context, err error) {
	status := commonerrors.HTTPStatusFromError(err)
	switch {
	case errors.Is(err, ErrRevisionMismatch):
		status = http.StatusPreconditionFailed
	case errors.Is(err, ErrDraftUnavailable), errors.Is(err, ErrVersionUnpublished),
		errors.Is(err, ErrRuleIncomplete), errors.Is(err, ErrChannelDisabled):
		status = http.StatusConflict
	}
	commonerrors.AbortWithError(c, status, err)
}

func setRevisionETag(c *gin.Context, revision int64) {
	c.Header("ETag", fmt.Sprintf(`"%d"`, revision))
}

func requestActor(c *gin.Context) string {
	actor, _ := c.Get(admin.CtxKeyUsername)
	username, _ := actor.(string)
	return username
}
