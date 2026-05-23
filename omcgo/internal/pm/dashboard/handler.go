package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/response"
)

// Handler 是 G6 dashboard REST 入口。
//
// 用户身份从 gin context 的 "user_id" key 取（由 admin.AuthMiddleware 注入 jwt.MapClaims.UserID）。
type Handler struct {
	svc    *Service
	logger *zap.Logger
}

// NewHandler 构造 Handler。
func NewHandler(svc *Service, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{svc: svc, logger: logger.Named("pm.dashboard.handler")}
}

// RegisterRoutes 挂 8 个 REST 端点（按 plan §1.3 一致）。
//
// 路由全部在 router group 内（caller 决定 prefix + auth middleware）：
//   GET    /pm/dashboards                  列表（owner + shared）
//   POST   /pm/dashboards                  创建
//   GET    /pm/dashboards/:id              详情（含 panels）
//   PUT    /pm/dashboards/:id              更新（仅 owner）
//   DELETE /pm/dashboards/:id              删除（仅 owner）
//   POST   /pm/dashboards/:id/fork         派生
//   POST   /pm/dashboards/:id/share        分享
//   DELETE /pm/dashboards/:id/share/:user  取消单用户分享
//   POST   /pm/dashboards/:id/panels       创建 panel
//   PUT    /pm/dashboards/:id/panels/:pid  更新 panel
//   DELETE /pm/dashboards/:id/panels/:pid  删除 panel
//   GET    /pm/user-preferences/dashboard  本人偏好
//   PUT    /pm/user-preferences/dashboard  更新偏好
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	dash := rg.Group("/pm/dashboards")
	{
		dash.GET("", h.List)
		dash.POST("", h.Create)
		dash.GET("/:id", h.Get)
		dash.PUT("/:id", h.Update)
		dash.DELETE("/:id", h.Delete)

		dash.POST("/:id/fork", h.Fork)
		dash.POST("/:id/share", h.Share)
		dash.DELETE("/:id/share/:user", h.Unshare)

		dash.POST("/:id/panels", h.CreatePanel)
		dash.PUT("/:id/panels/:pid", h.UpdatePanel)
		dash.DELETE("/:id/panels/:pid", h.DeletePanel)
	}
	prefs := rg.Group("/pm/user-preferences")
	{
		prefs.GET("/dashboard", h.GetPreferences)
		prefs.PUT("/dashboard", h.PutPreferences)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────

// userID 从 context 取登录用户。未登录返 uuid.Nil + false。
func userID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return uuid.Nil, false
	}
	switch x := v.(type) {
	case uuid.UUID:
		return x, true
	case string:
		id, err := uuid.Parse(x)
		if err != nil {
			return uuid.Nil, false
		}
		return id, true
	}
	return uuid.Nil, false
}

func mapErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Fail(c, http.StatusNotFound, "not found")
	case errors.Is(err, ErrPermissionDenied):
		response.Fail(c, http.StatusForbidden, "permission denied")
	case errors.Is(err, ErrBuiltinReadonly):
		response.Fail(c, http.StatusForbidden, "builtin dashboard is read-only; fork it first")
	default:
		commonerrors.AbortWithError(c, http.StatusInternalServerError, err)
	}
}

// ── DTOs ──────────────────────────────────────────────────────────────────

type dashboardDTO struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Description       string         `json:"description,omitempty"`
	OwnerID           string         `json:"owner_id"`
	SharedWith        []string       `json:"shared_with"`
	ParentDashboardID *string        `json:"parent_dashboard_id,omitempty"`
	Technology        string         `json:"technology"`
	Layout            json.RawMessage `json:"layout"`
	IsBuiltin         bool           `json:"is_builtin"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

func dashboardToDTO(d *Dashboard) dashboardDTO {
	shared := make([]string, 0, len(d.SharedWith))
	for _, u := range d.SharedWith {
		shared = append(shared, u.String())
	}
	var parent *string
	if d.ParentDashboardID != nil {
		s := d.ParentDashboardID.String()
		parent = &s
	}
	return dashboardDTO{
		ID: d.ID.String(), Name: d.Name, Description: d.Description,
		OwnerID: d.OwnerID.String(), SharedWith: shared, ParentDashboardID: parent,
		Technology: string(d.Technology), Layout: d.Layout,
		IsBuiltin: d.IsBuiltin,
		CreatedAt: d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: d.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

type panelDTO struct {
	ID             string          `json:"id"`
	DashboardID    string          `json:"dashboard_id"`
	PanelType      string          `json:"panel_type"`
	Title          string          `json:"title"`
	MetricPaths    []string        `json:"metric_paths"`
	Granularity    string          `json:"granularity"`
	Dimension      string          `json:"dimension"`
	DeviceSNs      []string        `json:"device_sns,omitempty"`
	DeviceGroupIDs []string        `json:"device_group_ids,omitempty"`
	TimeRange      json.RawMessage `json:"time_range"`
	CompareMode    *string         `json:"compare_mode,omitempty"`
	AdhocTaskID    *string         `json:"adhoc_task_id,omitempty"`
	Config         json.RawMessage `json:"config"`
}

func panelToDTO(p *Panel) panelDTO {
	dto := panelDTO{
		ID: p.ID.String(), DashboardID: p.DashboardID.String(),
		PanelType: string(p.PanelType), Title: p.Title,
		MetricPaths: p.MetricPaths, Granularity: p.Granularity,
		Dimension: string(p.Dimension), DeviceSNs: p.DeviceSNs,
		TimeRange: p.TimeRange, Config: p.Config,
	}
	if len(p.DeviceGroupIDs) > 0 {
		ids := make([]string, 0, len(p.DeviceGroupIDs))
		for _, g := range p.DeviceGroupIDs {
			ids = append(ids, g.String())
		}
		dto.DeviceGroupIDs = ids
	}
	if p.CompareMode != nil {
		s := string(*p.CompareMode)
		dto.CompareMode = &s
	}
	if p.AdhocTaskID != nil {
		s := p.AdhocTaskID.String()
		dto.AdhocTaskID = &s
	}
	return dto
}

// ── Dashboard handlers ──────────────────────────────────────────────────

type createDashboardDTO struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Technology  string          `json:"technology" binding:"required,oneof=lte nr gsm"`
	Layout      json.RawMessage `json:"layout"`
}

func (h *Handler) Create(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	var req createDashboardDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	d, err := h.svc.Create(c.Request.Context(), uid, CreateDashboardRequest{
		Name: req.Name, Description: req.Description,
		Technology: Technology(req.Technology), Layout: req.Layout,
	})
	if err != nil {
		mapErr(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, dashboardToDTO(d))
}

func (h *Handler) List(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	list, err := h.svc.List(c.Request.Context(), uid)
	if err != nil {
		mapErr(c, err)
		return
	}
	dtos := make([]dashboardDTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, dashboardToDTO(&list[i]))
	}
	response.OK(c, gin.H{"items": dtos, "total": len(dtos)})
}

func (h *Handler) Get(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	d, err := h.svc.Get(c.Request.Context(), uid, id)
	if err != nil {
		mapErr(c, err)
		return
	}
	panels, _ := h.svc.ListPanels(c.Request.Context(), uid, id)
	panelDTOs := make([]panelDTO, 0, len(panels))
	for i := range panels {
		panelDTOs = append(panelDTOs, panelToDTO(&panels[i]))
	}
	response.OK(c, gin.H{
		"dashboard": dashboardToDTO(d),
		"panels":    panelDTOs,
	})
}

type updateDashboardDTO struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Technology  *string         `json:"technology"`
	Layout      json.RawMessage `json:"layout"`
}

func (h *Handler) Update(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateDashboardDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	var techPtr *Technology
	if req.Technology != nil {
		t := Technology(*req.Technology)
		techPtr = &t
	}
	if err := h.svc.Update(c.Request.Context(), uid, id, UpdateDashboardRequest{
		Name: req.Name, Description: req.Description, Technology: techPtr, Layout: req.Layout,
	}); err != nil {
		mapErr(c, err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) Delete(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), uid, id); err != nil {
		mapErr(c, err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

type forkDTO struct {
	NewName string `json:"new_name" binding:"required"`
}

func (h *Handler) Fork(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req forkDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	d, err := h.svc.Fork(c.Request.Context(), uid, id, req.NewName)
	if err != nil {
		mapErr(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, dashboardToDTO(d))
}

type shareDTO struct {
	UserIDs []string `json:"user_ids" binding:"required,min=1"`
}

func (h *Handler) Share(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req shareDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	uids := make([]uuid.UUID, 0, len(req.UserIDs))
	for _, s := range req.UserIDs {
		u, err := uuid.Parse(s)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid user_id")
			return
		}
		uids = append(uids, u)
	}
	if err := h.svc.Share(c.Request.Context(), uid, ShareRequest{DashboardID: id, UserIDs: uids}); err != nil {
		mapErr(c, err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

func (h *Handler) Unshare(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	target, err := uuid.Parse(c.Param("user"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid user")
		return
	}
	if err := h.svc.Unshare(c.Request.Context(), uid, UnshareRequest{DashboardID: id, UserID: target}); err != nil {
		mapErr(c, err)
		return
	}
	response.OK(c, gin.H{"id": id.String()})
}

// ── Panel handlers ───────────────────────────────────────────────────────

type panelInputDTO struct {
	PanelType      string          `json:"panel_type" binding:"required"`
	Title          string          `json:"title" binding:"required"`
	MetricPaths    []string        `json:"metric_paths" binding:"required,min=1"`
	Granularity    string          `json:"granularity" binding:"required"`
	Dimension      string          `json:"dimension" binding:"required,oneof=device device_group"`
	DeviceSNs      []string        `json:"device_sns"`
	DeviceGroupIDs []string        `json:"device_group_ids"`
	TimeRange      json.RawMessage `json:"time_range"`
	CompareMode    *string         `json:"compare_mode"`
	AdhocTaskID    *string         `json:"adhoc_task_id"`
	Config         json.RawMessage `json:"config"`
}

func (d panelInputDTO) toReq(dashID uuid.UUID) (CreatePanelRequest, error) {
	req := CreatePanelRequest{
		DashboardID: dashID,
		PanelType:   PanelType(d.PanelType),
		Title:       d.Title,
		MetricPaths: d.MetricPaths,
		Granularity: d.Granularity,
		Dimension:   Dimension(d.Dimension),
		DeviceSNs:   d.DeviceSNs,
		TimeRange:   d.TimeRange,
		Config:      d.Config,
	}
	if len(d.DeviceGroupIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(d.DeviceGroupIDs))
		for _, s := range d.DeviceGroupIDs {
			u, err := uuid.Parse(s)
			if err != nil {
				return req, err
			}
			ids = append(ids, u)
		}
		req.DeviceGroupIDs = ids
	}
	if d.CompareMode != nil {
		m := CompareMode(*d.CompareMode)
		req.CompareMode = &m
	}
	if d.AdhocTaskID != nil {
		u, err := uuid.Parse(*d.AdhocTaskID)
		if err == nil {
			req.AdhocTaskID = &u
		}
	}
	return req, nil
}

func (h *Handler) CreatePanel(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	dashID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var dto panelInputDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req, err := dto.toReq(dashID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid uuid in input")
		return
	}
	p, err := h.svc.CreatePanel(c.Request.Context(), uid, req)
	if err != nil {
		mapErr(c, err)
		return
	}
	response.OKWithStatus(c, http.StatusCreated, panelToDTO(p))
}

func (h *Handler) UpdatePanel(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	panelID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid pid")
		return
	}
	var dto panelInputDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	req, err := dto.toReq(uuid.Nil) // service 内会用 panel 现存的 dashboard_id 防越权
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid uuid in input")
		return
	}
	if err := h.svc.UpdatePanel(c.Request.Context(), uid, panelID, req); err != nil {
		mapErr(c, err)
		return
	}
	response.OK(c, gin.H{"id": panelID.String()})
}

func (h *Handler) DeletePanel(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	panelID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid pid")
		return
	}
	if err := h.svc.DeletePanel(c.Request.Context(), uid, panelID); err != nil {
		mapErr(c, err)
		return
	}
	response.OK(c, gin.H{"id": panelID.String()})
}

// ── UserPreferences handlers ────────────────────────────────────────────

// parseTechnologyParam 从 query string 取 technology；缺省返 lte（默认制式）。
func parseTechnologyParam(c *gin.Context) (Technology, bool) {
	v := c.Query("technology")
	if v == "" {
		return TechLTE, true // 默认 lte（向前兼容老 client 不传 technology 的请求）
	}
	switch v {
	case "lte", "nr", "gsm":
		return Technology(v), true
	}
	response.Fail(c, http.StatusBadRequest, "invalid technology (lte/nr/gsm)")
	return "", false
}

// GetPreferences GET /pm/user-preferences/dashboard?technology=lte|nr|gsm
//
// 不传 technology 时默认 lte。返回当前用户在该制式下的 KPI 卡片 layout +
// current_dashboard_id + shared_filters。
func (h *Handler) GetPreferences(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	tech, ok := parseTechnologyParam(c)
	if !ok {
		return
	}
	p, err := h.svc.GetUserPreferences(c.Request.Context(), uid, tech)
	if err != nil {
		mapErr(c, err)
		return
	}
	resp := gin.H{
		"user_id":              p.UserID.String(),
		"technology":           string(p.Technology),
		"kpi_card_layout":      p.KPICardLayout,
		"shared_filters":       p.SharedFilters,
	}
	if p.CurrentDashboardID != nil {
		resp["current_dashboard_id"] = p.CurrentDashboardID.String()
	}
	response.OK(c, resp)
}

type prefsDTO struct {
	Technology         string          `json:"technology" binding:"required,oneof=lte nr gsm"`
	KPICardLayout      json.RawMessage `json:"kpi_card_layout"`
	CurrentDashboardID *string         `json:"current_dashboard_id"`
	SharedFilters      json.RawMessage `json:"shared_filters"`
}

func (h *Handler) PutPreferences(c *gin.Context) {
	uid, ok := userID(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, "missing user_id")
		return
	}
	var req prefsDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}
	prefs := &UserPreferences{
		UserID:        uid,
		Technology:    Technology(req.Technology),
		KPICardLayout: req.KPICardLayout,
		SharedFilters: req.SharedFilters,
	}
	if req.CurrentDashboardID != nil {
		id, err := uuid.Parse(*req.CurrentDashboardID)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, "invalid current_dashboard_id")
			return
		}
		prefs.CurrentDashboardID = &id
	}
	if err := h.svc.UpsertUserPreferences(c.Request.Context(), prefs); err != nil {
		mapErr(c, err)
		return
	}
	response.OK(c, gin.H{"user_id": uid.String(), "technology": req.Technology})
}
