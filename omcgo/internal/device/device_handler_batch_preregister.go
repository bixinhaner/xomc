package device

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/admin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
)

// ── 批量预登记 DTO ────────────────────────────────────────────────────────────
//
// 批量预登记用于"开站规划"场景：在设备通过 TR-069 Bootstrap 自动注册前，
// 运维人员先上传 SN 列表并预设名称/备注，便于开站统计。
//
// 行为语义：
//   - SN 已存在 → 只更新 device_name / remark（不改 carrier/technology 等不可变字段）
//   - SN 不存在 → 调用 CreateDevice 新建（lifecycle_state='registered', is_online=false）
//     设备不写入 device_group_members，自动落在"未分组"视图（source_type='auto'）
//   - carrier 推断：优先使用 CSV 填写值；若为空则从 SN 前6位推断 OUI → CarrierRegistry
//     推断失败 → error_code="carrier_required"，该行跳过，不阻断其他行
//   - technology 默认 "lte"（如 CSV 未填）
//   - product_class 可选；UPS 预登记填写 UPS* 后可在设备首次 Inform 前进入 UPS 设备类型口径

// BatchPreRegisterRow 单行预登记请求。
type BatchPreRegisterRow struct {
	// SerialNumber 设备 SN，必填。
	SerialNumber string `json:"serial_number" binding:"required"`
	// DeviceName 设备名称（可选）。
	DeviceName *string `json:"device_name"`
	// Remark 备注（可选）。
	Remark *string `json:"remark"`
	// Carrier 运营商（可选）。为空时从 SN 前6位推断 OUI → CarrierRegistry。
	// 取值：cmcc / ctcc / cucc
	Carrier model.CarrierCode `json:"carrier"`
	// Technology 制式（可选）。为空时默认 lte。取值：lte / nr
	Technology model.Technology `json:"technology"`
	// OUI 设备 OUI（可选）。为空时从 SN 前6位截取。
	OUI string `json:"oui"`
	// ProductClass 设备 TR-069 ProductClass（可选）。UPS 预登记可填写 UPS*，
	// 真正上线后仍由 Inform 上报值覆盖。
	ProductClass string `json:"product_class"`
}

// BatchPreRegisterRequest 批量预登记请求体。
type BatchPreRegisterRequest struct {
	Devices []BatchPreRegisterRow `json:"devices" binding:"required,min=1,max=1000,dive"`
}

// BatchPreRegisterRowResult 单行预登记结果。
type BatchPreRegisterRowResult struct {
	// Row CSV 行号（1-based，不含 header）。
	Row int `json:"row"`
	// SN 设备 SN。
	SN string `json:"sn"`
	// Action 操作类型：created（新建）/ updated（已存在则更新）/ skipped（跳过）。
	Action string `json:"action,omitempty"`
	// ErrorCode 机器可读错误码（仅失败行有值）。
	ErrorCode string `json:"error_code,omitempty"`
	// Reason 英文兜底描述。
	Reason string `json:"reason,omitempty"`
}

// BatchPreRegisterResponse 批量预登记响应。
type BatchPreRegisterResponse struct {
	Total   int                         `json:"total"`
	Created int                         `json:"created"`
	Updated int                         `json:"updated"`
	Failed  int                         `json:"failed"`
	Errors  []BatchPreRegisterRowResult `json:"errors"`
}

// BatchPreRegisterDevices handles POST /api/v1/devices/batch-preregister.
//
// 用于批量开站规划场景：在设备 Bootstrap 到达前，预先录入 SN 列表和设备名称。
// 已存在的 SN 仅更新名称/备注；不存在的 SN 新建设备（lifecycle='registered'）。
// 新建设备不写入任何分组，自然落在"未分组"视图，source_type 显示 Auto。
func (h *Handler) BatchPreRegisterDevices(c *gin.Context) {
	var req BatchPreRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		commonerrors.AbortWithError(c, http.StatusBadRequest, err)
		return
	}

	updater := ""
	if v, ok := c.Get(admin.CtxKeyUsername); ok {
		updater, _ = v.(string)
	}

	resp := BatchPreRegisterResponse{Total: len(req.Devices)}

	for i, row := range req.Devices {
		rowNum := i + 1
		sn := strings.TrimSpace(row.SerialNumber)

		// ── 1. 按 SN 查是否已存在 ──────────────────────────────────────────
		existing, err := h.service.GetBySerialNumber(c.Request.Context(), sn)
		if err != nil {
			resp.Failed++
			resp.Errors = append(resp.Errors, BatchPreRegisterRowResult{
				Row:       rowNum,
				SN:        sn,
				ErrorCode: "lookup_failed",
				Reason:    "failed to lookup device: " + err.Error(),
			})
			continue
		}

		if existing != nil {
			// ── 2a. 已存在 → 只更新名称/备注 ─────────────────────────────
			if row.DeviceName != nil {
				if _, updErr := h.service.UpdateDevice(c.Request.Context(), existing.ID, UpdateDeviceRequest{DeviceName: row.DeviceName}); updErr != nil {
					resp.Failed++
					resp.Errors = append(resp.Errors, BatchPreRegisterRowResult{
						Row:       rowNum,
						SN:        sn,
						ErrorCode: "update_name_failed",
						Reason:    "update device name failed: " + updErr.Error(),
					})
					continue
				}
			}
			if row.Remark != nil {
				if updErr := h.service.UpdateDeviceInfo(c.Request.Context(), existing.ID, UpdateDeviceInfoRequest{Remark: row.Remark}, updater); updErr != nil {
					resp.Failed++
					resp.Errors = append(resp.Errors, BatchPreRegisterRowResult{
						Row:       rowNum,
						SN:        sn,
						ErrorCode: "update_remark_failed",
						Reason:    "update remark failed: " + updErr.Error(),
					})
					continue
				}
			}
			resp.Updated++
			continue
		}

		// ── 2b. 不存在 → 新建预登记设备 ──────────────────────────────────

		// 推断 OUI：优先使用 row.OUI，否则取 SN 前6位
		oui := strings.TrimSpace(row.OUI)
		if oui == "" && len(sn) >= 6 {
			oui = strings.ToUpper(sn[:6])
		}

		// 推断 carrier：优先使用 row.Carrier，否则通过 OUI 查 CarrierRegistry
		carrier := row.Carrier
		if carrier == "" && oui != "" {
			carrier = h.service.ResolveCarrierByOUI(oui)
		}
		if carrier == "" {
			resp.Failed++
			resp.Errors = append(resp.Errors, BatchPreRegisterRowResult{
				Row:       rowNum,
				SN:        sn,
				ErrorCode: "carrier_required",
				Reason:    "carrier cannot be inferred from OUI; please specify carrier field (cmcc/ctcc/cucc)",
			})
			continue
		}

		// technology 默认 lte
		technology := row.Technology
		if technology == "" {
			technology = model.TechLTE
		}

		// 设备名称：使用 CSV 填写值，不填则空（后续可由运维补充）
		deviceName := ""
		if row.DeviceName != nil {
			deviceName = *row.DeviceName
		}

		createReq := CreateDeviceRequest{
			SerialNumber: sn,
			OUI:          oui,
			ProductClass: strings.TrimSpace(row.ProductClass),
			Carrier:      carrier,
			Technology:   technology,
			DeviceName:   deviceName,
		}

		if _, createErr := h.service.CreateDevice(c.Request.Context(), createReq); createErr != nil {
			if createErr == commonerrors.ErrAlreadyExists {
				// 极小的并发窗口：刚才查不存在但现在已被其他请求创建；当 updated 处理
				resp.Updated++
				continue
			}
			resp.Failed++
			resp.Errors = append(resp.Errors, BatchPreRegisterRowResult{
				Row:       rowNum,
				SN:        sn,
				ErrorCode: "create_failed",
				Reason:    "create device failed: " + createErr.Error(),
			})
			continue
		}

		// 备注单独写入 device_info（CreateDevice 不含 remark 字段）
		if row.Remark != nil {
			created, lookupErr := h.service.GetBySerialNumber(c.Request.Context(), sn)
			if lookupErr == nil && created != nil {
				// UpdateDeviceInfo 若 device_info 行尚未创建会返回 ErrNotFound，静默忽略
				// （Bootstrap Inform 到达后 device_info 会由 CreateDeviceInfo 补全）
				_ = h.service.UpdateDeviceInfo(c.Request.Context(), created.ID, UpdateDeviceInfoRequest{Remark: row.Remark}, updater)
			}
		}

		resp.Created++
	}

	response.OK(c, resp)
}
