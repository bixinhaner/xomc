package interop

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/errors"
	"github.com/omcgo/omcgo/internal/model"
)

// Handler provides REST API endpoints for interop testing (F10).
type Handler struct {
	runner    *ConformanceTestRunner
	validator *DataModelValidator
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

// RegisterRoutes registers interop testing routes on the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	interop := rg.Group("/interop")
	{
		interop.GET("/test-cases", h.ListTestCases)
		interop.POST("/run", h.RunTests)
		interop.POST("/run/:category", h.RunByCategory)
		interop.POST("/validate/:deviceId", h.ValidateDevice)
	}
}

// ListTestCases returns all registered conformance test cases grouped by category.
func (h *Handler) ListTestCases(c *gin.Context) {
	cases := h.runner.ListTestCases()
	c.JSON(http.StatusOK, cases)
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

	var results []TestResult
	var err error

	if len(req.Categories) == 0 {
		results, err = h.runner.RunAll(c.Request.Context(), req.DeviceSN)
	} else {
		for _, cat := range req.Categories {
			if !cat.IsValid() {
				commonerrors.AbortWithError(c, http.StatusBadRequest,
					commonerrors.NewBusinessError(10001, "invalid test category: "+string(cat), commonerrors.ErrInvalidInput))
				return
			}
			catResults, catErr := h.runner.RunByCategory(c.Request.Context(), req.DeviceSN, cat)
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

	c.JSON(http.StatusOK, gin.H{
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

	results, err := h.runner.RunByCategory(c.Request.Context(), req.DeviceSN, category)
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

	c.JSON(http.StatusOK, gin.H{
		"device_sn": req.DeviceSN,
		"category":  category,
		"total":     len(results),
		"passed":    passed,
		"failed":    len(results) - passed,
		"results":   results,
	})
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

	report, err := h.validator.ValidateDevice(c.Request.Context(), deviceID, carrier, tech)
	if err != nil {
		status := commonerrors.HTTPStatusFromError(err)
		commonerrors.AbortWithError(c, status, err)
		return
	}

	c.JSON(http.StatusOK, report)
}
