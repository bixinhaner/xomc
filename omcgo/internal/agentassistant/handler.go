package agentassistant

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/core/response"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	g := group.Group("/agent/assistants")
	g.Use(func(c *gin.Context) {
		value, _ := c.Get(admin.CtxKeyClaims)
		claims, ok := value.(*admin.Claims)
		if !ok || claims == nil || claims.UserID == uuid.Nil {
			c.Abort()
			response.Fail(c, 401, "ASSISTANT_SIGN_IN_REQUIRED")
			return
		}
		c.Next()
	})
	g.GET("/catalog", h.catalog)
	g.GET("", h.list)
	g.POST("", h.create)
	g.GET("/runs/:runID", h.run)
	g.POST("/runs/:runID/cancel", h.cancel)
	g.POST("/runs/:runID/read", h.read)
	g.GET("/:id", h.get)
	g.POST("/:id/messages", h.plan)
	g.PUT("/:id/draft", h.save)
	g.POST("/:id/publish", h.publish)
	g.PATCH("/:id/state", h.state)
	g.GET("/:id/runs", h.runs)
	g.POST("/:id/runs", h.start)
}
func claims(c *gin.Context) *admin.Claims { return c.MustGet(admin.CtxKeyClaims).(*admin.Claims) }
func fail(c *gin.Context, err error) {
	code := "ASSISTANT_REQUEST_FAILED"
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, ErrNotFound):
		code = "ASSISTANT_NOT_FOUND"
		status = 404
	case errors.Is(err, ErrConflict):
		code = "ASSISTANT_REVISION_CONFLICT"
		status = 409
	default:
		for _, part := range strings.FieldsFunc(err.Error(), func(r rune) bool { return r == ':' || r == ' ' || r == '\n' }) {
			if strings.HasPrefix(part, "ASSISTANT_") {
				code = part
				break
			}
		}
	}
	if strings.Contains(code, "FORBIDDEN") {
		status = 403
	}
	if strings.Contains(code, "NOT_CONNECTED") {
		status = 503
	}
	// Database/upstream details stay in run diagnostics and server logs, not HTTP errors.
	response.Fail(c, status, code)
}
func bind(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 128<<10)
	if err := c.ShouldBindJSON(target); err != nil {
		response.Fail(c, 400, "ASSISTANT_INVALID_REQUEST")
		return false
	}
	return true
}
func (h *Handler) catalog(c *gin.Context) {
	v, e := h.service.Catalog(c.Request.Context(), claims(c))
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) list(c *gin.Context) {
	v, e := h.service.repo.List(c.Request.Context(), claims(c).UserID.String())
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) get(c *gin.Context) {
	if _, e := uuid.Parse(c.Param("id")); e != nil {
		response.Fail(c, 404, "ASSISTANT_NOT_FOUND")
		return
	}
	v, e := h.service.repo.Get(c.Request.Context(), claims(c).UserID.String(), c.Param("id"))
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) create(c *gin.Context) {
	var body struct {
		ID       string `json:"id"`
		Locale   string `json:"locale"`
		Timezone string `json:"timezone"`
	}
	if !bind(c, &body) {
		return
	}
	v, e := h.service.Create(c.Request.Context(), claims(c), body.ID, body.Locale, body.Timezone)
	if e != nil {
		fail(c, e)
		return
	}
	response.OKWithStatus(c, 201, v)
}
func (h *Handler) plan(c *gin.Context) {
	// Planning uses a bounded runtime call rather than the normal API timeout.
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(150 * time.Second))
	var body struct {
		Revision int    `json:"revision"`
		Message  string `json:"message"`
	}
	if !bind(c, &body) {
		return
	}
	v, e := h.service.Plan(c.Request.Context(), claims(c), c.Param("id"), body.Revision, body.Message)
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) save(c *gin.Context) {
	var body struct {
		Revision   int        `json:"revision"`
		Definition Definition `json:"definition"`
	}
	if !bind(c, &body) {
		return
	}
	v, e := h.service.SaveDefinition(c.Request.Context(), claims(c), c.Param("id"), body.Revision, body.Definition)
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) publish(c *gin.Context) {
	var body struct {
		Revision int `json:"revision"`
	}
	if !bind(c, &body) {
		return
	}
	v, e := h.service.Publish(c.Request.Context(), claims(c), c.Param("id"), body.Revision)
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) state(c *gin.Context) {
	var body struct {
		State string `json:"state"`
	}
	if !bind(c, &body) {
		return
	}
	v, e := h.service.State(c.Request.Context(), claims(c), c.Param("id"), body.State)
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) start(c *gin.Context) {
	var body struct {
		Revision  int    `json:"revision"`
		Kind      string `json:"kind"`
		RequestID string `json:"requestId"`
	}
	if !bind(c, &body) {
		return
	}
	v, e := h.service.StartRun(c.Request.Context(), claims(c), c.Param("id"), body.Revision, body.Kind, body.RequestID)
	if e != nil {
		fail(c, e)
		return
	}
	response.OKWithStatus(c, 202, v)
}
func (h *Handler) runs(c *gin.Context) {
	v, e := h.service.Runs(c.Request.Context(), claims(c), c.Param("id"))
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) run(c *gin.Context) {
	v, e := h.service.Run(c.Request.Context(), claims(c), c.Param("runID"))
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) cancel(c *gin.Context) {
	v, e := h.service.repo.Cancel(c.Request.Context(), claims(c).UserID.String(), c.Param("runID"))
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, v)
}
func (h *Handler) read(c *gin.Context) {
	e := h.service.repo.ReadRun(c.Request.Context(), claims(c).UserID.String(), c.Param("runID"))
	if e != nil {
		fail(c, e)
		return
	}
	response.OK(c, nil)
}
