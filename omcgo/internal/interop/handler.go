package interop

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler provides REST API endpoints for interop testing (F10).
type Handler struct {
	runner    *ConformanceTestRunner
	validator *DataModelValidator
	resolver  *authz.Resolver
	logger    *zap.Logger
}

// NewHandler creates a new interop Handler.
func NewHandler(runner *ConformanceTestRunner, validator *DataModelValidator, logger *zap.Logger) *Handler {
	return &Handler{
		runner:    runner,
		validator: validator,
		logger:    logger.Named("interop-handler"),
	}
}

// SetPermissionService 注入数据权限解析器（#63 设备组可见性强制层）。未注入时
// FromContext 走 nil-safe 退化（不过滤），与 device/alarm 模块语义一致。
func (h *Handler) SetPermissionService(perm authz.VisibleGroupsResolver) {
	h.resolver = authz.NewResolver(perm)
}

// RegisterRoutes registers interop testing routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	interop := rg.Group("/interop")
	{
		interop.GET("/test-cases", h.ListTestCases)
		interop.POST("/run", h.RunTests)
		interop.POST("/run/:category", h.RunByCategory)
		interop.POST("/run/report", h.ExportReport) // T-0115: CSV report export
		interop.POST("/validate/:deviceId", h.ValidateDevice)
	}
}

// ListTestCases returns all registered conformance test cases grouped by category.
func (h *Handler) ListTestCases(c *gin.Context) {
	cases := h.runner.ListTestCases()
	response.OK(c, cases)
}

// RunTests executes conformance tests against a device.
// If categories are specified in the request body, only those categories are run.
// Otherwise all categories are run.
func (h *Handler) RunTests(c *gin.Context) {
	var req RunTestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}

	var results []TestResult
	var err error

	if len(req.Categories) == 0 {
		results, err = h.runner.RunAll(c.Request.Context(), req.DeviceSN, visibleGroups)
	} else {
		for _, cat := range req.Categories {
			if !cat.IsValid() {
				commonerrors.AbortWithError(c, http.StatusBadRequest,
					commonerrors.NewBusinessError(10001, "invalid test category: "+string(cat), commonerrors.ErrInvalidInput))
				return
			}
			catResults, catErr := h.runner.RunByCategory(c.Request.Context(), req.DeviceSN, cat, visibleGroups)
			if catErr != nil {
				err = catErr
				break
			}
			results = append(results, catResults...)
		}
	}

	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	// Compute summary.
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}

	response.OK(c, gin.H{
		"device_sn": req.DeviceSN,
		"total":     len(results),
		"passed":    passed,
		"failed":    len(results) - passed,
		"results":   results,
	})
}

// RunByCategory executes conformance tests for a single category.
func (h *Handler) RunByCategory(c *gin.Context) {
	category := TestCategory(c.Param("category"))
	if !category.IsValid() {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(10001, "invalid test category: "+string(category), commonerrors.ErrInvalidInput))
		return
	}

	var req RunTestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}

	results, err := h.runner.RunByCategory(c.Request.Context(), req.DeviceSN, category, visibleGroups)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}

	response.OK(c, gin.H{
		"device_sn": req.DeviceSN,
		"category":  category,
		"total":     len(results),
		"passed":    passed,
		"failed":    len(results) - passed,
		"results":   results,
	})
}

// executeRunRequest is the common path between RunTests and ExportReport:
// runs the requested categories (all categories if Categories is empty) and
// returns the flat result slice. Returns ErrInvalidInput-wrapped error if a
// requested category is unknown.
func (h *Handler) executeRunRequest(ctx context.Context, req *RunTestsRequest, visibleGroups []uuid.UUID) ([]TestResult, error) {
	if len(req.Categories) == 0 {
		return h.runner.RunAll(ctx, req.DeviceSN, visibleGroups)
	}

	var results []TestResult
	for _, cat := range req.Categories {
		if !cat.IsValid() {
			return nil, commonerrors.NewBusinessError(10001, "invalid test category: "+string(cat), commonerrors.ErrInvalidInput)
		}
		catResults, err := h.runner.RunByCategory(ctx, req.DeviceSN, cat, visibleGroups)
		if err != nil {
			return nil, err
		}
		results = append(results, catResults...)
	}
	return results, nil
}

// ExportReport executes the same conformance run as RunTests but streams the
// results as a downloadable report (CSV today; PDF / markdown reserved for
// future T-0115 iterations). Format is selected via the ?format= query
// parameter and defaults to csv.
//
// T-0115 Phase 1: CSV only — gives operator QA a sign-off artefact without
// needing to scrape the JSON RunTests response.
func (h *Handler) ExportReport(c *gin.Context) {
	formatParam := c.DefaultQuery("format", string(ReportFormatCSV))
	format := ReportFormat(formatParam)
	if !format.IsValid() {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(10005, "unsupported report format: "+formatParam+" (supported: csv)", commonerrors.ErrInvalidInput))
		return
	}

	var req RunTestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}

	results, err := h.executeRunRequest(c.Request.Context(), &req, visibleGroups)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	filename := ReportFilename(req.DeviceSN, format, time.Now())
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Header("X-Interop-Report-Format", string(format))
	c.Header("X-Interop-Report-Rows", strconv.Itoa(len(results)))
	c.Status(http.StatusOK)

	if err := WriteCSVReport(c.Writer, req.DeviceSN, results); err != nil {
		h.logger.Error("write csv report failed",
			zap.String("device_sn", req.DeviceSN),
			zap.Int("results", len(results)),
			zap.Error(err),
		)
		// Headers already flushed by csv writer at this point; no clean way
		// to surface the error back to client mid-stream — log + return.
	}
}

// ValidateDevice compares a device's actual parameters against its data model definition.
func (h *Handler) ValidateDevice(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("deviceId"))
	if err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(10002, "invalid device ID", commonerrors.ErrInvalidInput))
		return
	}

	carrier := model.CarrierCode(c.Query("carrier"))
	if !carrier.IsValid() {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(10003, "invalid or missing carrier", commonerrors.ErrInvalidInput))
		return
	}

	tech := model.Technology(c.Query("tech"))
	if !tech.IsValid() {
		commonerrors.AbortWithError(c, http.StatusBadRequest,
			commonerrors.NewBusinessError(10004, "invalid or missing technology", commonerrors.ErrInvalidInput))
		return
	}

	visibleGroups, ok := h.resolver.FromContext(c)
	if !ok {
		return
	}

	report, err := h.validator.ValidateDevice(c.Request.Context(), deviceID, carrier, tech, visibleGroups)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	response.OK(c, report)
}
