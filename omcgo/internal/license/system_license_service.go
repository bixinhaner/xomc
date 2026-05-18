// system_license_service.go — F06 System License 重构 P1 Step 2。
//
// PRD: docs/project/prd/F06-system-license-redesign.md §5（API）+ §3.3（业务规则）。
//
// 与老 license.Service 并存：本 service 只处理 singleton system_license
// （GetCurrent / Update / ListHistory）；老的 multi-license + Activate/Revoke
// 流程留到 Step 5 才下线。
package license

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

// SystemLicenseService — singleton system_license 业务编排：
//  1. GetCurrent → repo.GetCurrent，NotFound 翻译成 12113
//  2. Update     → 解析 raw → 验签（可选 strict）→ 重复 ID 预检 → repo.Replace
//  3. ListHistory→ 透传 repo
//
// 与老 License Service 的区别：无 Activate/Revoke 概念，singleton 不变量靠
// repo 事务 + DB partial unique index 保证。
type SystemLicenseService struct {
	repo        SystemLicenseRepository
	logger      *zap.Logger
	sigVerifier *SignatureVerifier // 可选；nil 时退化为 unverified 放过
	enforcer    Enforcer           // 可选；Update 成功后调 Invalidate 让 enforcer 重读
}

// NewSystemLicenseService 构造 service。sigVerifier 可后置 SetSignatureVerifier。
func NewSystemLicenseService(repo SystemLicenseRepository, logger *zap.Logger) *SystemLicenseService {
	return &SystemLicenseService{
		repo:   repo,
		logger: logger.Named("system-license"),
	}
}

// SetSignatureVerifier 注入 OEM 公钥验签器。
//
// nil 等价于"不验签 + 全部 unverified"；strict 模式由 verifier 自身的 strict
// 字段控制（与老 license 共用同一 verifier 实例，配置走 license.signing.strict）。
func (s *SystemLicenseService) SetSignatureVerifier(v *SignatureVerifier) {
	s.sigVerifier = v
}

// SetEnforcer 注入 Enforcer，让 Update 成功后能调 Invalidate 让 enforcer 立即
// 拉新 license（避免 5min cache TTL 内 device.create 仍用旧容量裁决）。
// nil 安全：Update 路径跳过 Invalidate。
func (s *SystemLicenseService) SetEnforcer(e Enforcer) {
	s.enforcer = e
}

// GetCurrent 返回当前生效 license。无 license 时返业务错误 12113 +
// commonerrors.ErrNotFound（→ HTTP 404）。
func (s *SystemLicenseService) GetCurrent(ctx context.Context) (*SystemLicense, error) {
	lic, err := s.repo.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, ErrSystemLicenseNotFound) {
			return nil, commonerrors.NewBusinessError(
				global.ErrCodeSystemLicenseNotConfigured,
				"no system license configured",
				commonerrors.ErrNotFound,
			)
		}
		return nil, fmt.Errorf("get current system license: %w", err)
	}
	return lic, nil
}

// ListHistory 透传 repo.ListHistory。Filter 内部走 pagination 默认值兜底。
func (s *SystemLicenseService) ListHistory(
	ctx context.Context, filter SystemLicenseHistoryFilter,
) (*model.ListResponse[SystemLicenseHistory], error) {
	resp, err := s.repo.ListHistory(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list system license history: %w", err)
	}
	return resp, nil
}

// UpdateRequest — POST /system-license 请求体。
//
// RawContent 是 OEM 签发的 license JSON 文件原文（可含 signature/signature_key_id
// 字段，verifier 会按 canonical JSON 验签）。
type UpdateRequest struct {
	RawContent       string     `json:"raw_content"`
	UploadedByUserID *uuid.UUID `json:"-"` // 由 handler 从 gin ctx 注入
}

// UpdateResult — Update 的返回值。
//
// Current 是刚生效的 license；Replaced 是被替换的旧 license（首次上传 = nil）。
type UpdateResult struct {
	Current  *SystemLicense        `json:"current"`
	Replaced *SystemLicenseHistory `json:"replaced,omitempty"`
}

// Update 上传新 license 覆盖当前：
//
//  1. 解析 RawContent JSON → parsedLicense
//  2. 必填字段校验（license_id / license_type / issued_at）
//  3. 验签（sigVerifier 注入时）→ strict 模式失败拒
//  4. license_id 重复预检（current 表或 history 表存在 → 12110）
//  5. 构造 SystemLicense + repo.Replace（事务 + SELECT FOR UPDATE）
//
// 错误：
//   - JSON 解析 / 必填缺失 → 12111 (400)
//   - 签名校验失败（strict） → 12109 (400)
//   - license_id 已存在     → 12110 (409)
//   - 其他 DB 错误          → 500
func (s *SystemLicenseService) Update(ctx context.Context, req UpdateRequest) (*UpdateResult, error) {
	if strings.TrimSpace(req.RawContent) == "" {
		return nil, commonerrors.NewBusinessError(
			global.ErrCodeSystemLicenseInvalidFormat,
			"raw_content is required",
			commonerrors.ErrInvalidInput,
		)
	}

	parsed, err := parseSystemLicenseJSON([]byte(req.RawContent))
	if err != nil {
		return nil, commonerrors.NewBusinessError(
			global.ErrCodeSystemLicenseInvalidFormat,
			err.Error(),
			commonerrors.ErrInvalidInput,
		)
	}

	// 签名验证（可选）。verifier=nil → unverified 放过；strict=true 时未签/失败拒。
	sigStatus := SignatureUnverified
	sigNote := "signature verifier not configured"
	if s.sigVerifier != nil {
		var sigErr error
		sigStatus, sigNote, sigErr = s.sigVerifier.VerifyLicenseJSON([]byte(req.RawContent))
		if sigErr != nil {
			s.logger.Warn("system license signature rejected (strict mode)",
				zap.String("license_id", parsed.LicenseID),
				zap.String("signature_status", string(sigStatus)),
				zap.String("signature_note", sigNote))
			return nil, commonerrors.NewBusinessError(
				global.ErrCodeLicenseSignatureVerifyFailed,
				"system license signature verification failed: "+sigNote,
				commonerrors.ErrInvalidInput,
			)
		}
	}

	// license_id 重复预检：current 行 + history 表都查一遍。Replace 内部
	// 还会撞 UNIQUE 兜底，但这里前置可以给出更友好的 409。
	exists, err := s.repo.ExistsByLicenseID(ctx, parsed.LicenseID)
	if err != nil {
		return nil, fmt.Errorf("pre-check license_id exists: %w", err)
	}
	if exists {
		return nil, commonerrors.NewBusinessError(
			global.ErrCodeSystemLicenseIDExists,
			fmt.Sprintf("license_id %q already exists (current or history)", parsed.LicenseID),
			commonerrors.ErrAlreadyExists,
		)
	}

	now := nowFunc()
	lic := &SystemLicense{
		LicenseID:        parsed.LicenseID,
		LicenseType:      SystemLicenseType(parsed.LicenseType),
		Issuer:           parsed.Issuer,
		Licensee:         parsed.Licensee,
		IssuedAt:         parsed.IssuedAt,
		ExpiryDate:       parsed.ExpiryDate,
		DevicesSupport:   parsed.DevicesSupport,
		FeatureList:      FeatureList(parsed.FeatureList),
		RawContent:       req.RawContent,
		SignatureKeyID:   parsed.SignatureKeyID,
		Signature:        parsed.Signature,
		SignatureStatus:  sigStatus,
		UploadedAt:       now,
		UploadedByUserID: req.UploadedByUserID,
		IsCurrent:        true,
	}

	replaced, err := s.repo.Replace(ctx, lic)
	if err != nil {
		if errors.Is(err, ErrSystemLicenseIDExists) {
			// repo 兜底也会撞 UNIQUE — 翻译成业务码。
			return nil, commonerrors.NewBusinessError(
				global.ErrCodeSystemLicenseIDExists,
				fmt.Sprintf("license_id %q already exists", parsed.LicenseID),
				commonerrors.ErrAlreadyExists,
			)
		}
		return nil, fmt.Errorf("replace system license: %w", err)
	}

	// Step 3：Update 成功立即让 enforcer 失效缓存，避免 5min TTL 内 device.create
	// 仍走旧容量裁决。nil-safe（enforcer 未注入时 noop）。
	if s.enforcer != nil {
		s.enforcer.Invalidate()
	}

	s.logger.Info("system license updated",
		zap.String("license_id", lic.LicenseID),
		zap.String("license_type", string(lic.LicenseType)),
		zap.String("signature_status", string(sigStatus)),
		zap.String("signature_note", sigNote),
		zap.Bool("first_install", replaced == nil))

	return &UpdateResult{Current: lic, Replaced: replaced}, nil
}

// parsedLicense 是 RawContent JSON 的中间结构，仅用于解析 → 业务字段提取。
//
// 注意：不和 SystemLicense 共用结构体——SystemLicense 有 DB 元字段（id / is_current
// / created_at 等），不应允许 license 文件控制；这里只接受文件级字段。
type parsedLicense struct {
	LicenseID      string          `json:"license_id"`
	LicenseType    string          `json:"license_type"`
	Issuer         *string         `json:"issuer,omitempty"`
	Licensee       *string         `json:"licensee,omitempty"`
	IssuedAt       time.Time       `json:"issued_at"`
	ExpiryDate     *time.Time      `json:"expiry_date,omitempty"`
	DevicesSupport DevicesSupport  `json:"devices_support"`
	FeatureList    json.RawMessage `json:"feature_list"`
	Signature      *string         `json:"signature,omitempty"`
	SignatureKeyID *string         `json:"signature_key_id,omitempty"`
}

// parseSystemLicenseJSON 解析 raw bytes 并做基本字段校验。
//
// 必填：license_id / license_type / issued_at。
// license_type 必须是 SystemLicenseType 枚举之一（DB CHECK 也会兜底）。
func parseSystemLicenseJSON(raw []byte) (*parsedLicense, error) {
	var p parsedLicense
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("license JSON parse failed: %w", err)
	}
	if strings.TrimSpace(p.LicenseID) == "" {
		return nil, errors.New("license_id is required")
	}
	if strings.TrimSpace(p.LicenseType) == "" {
		return nil, errors.New("license_type is required")
	}
	if !isValidSystemLicenseType(p.LicenseType) {
		return nil, fmt.Errorf("license_type %q is invalid (must be Commercial/Trial/Evaluation/Internal)", p.LicenseType)
	}
	if p.IssuedAt.IsZero() {
		return nil, errors.New("issued_at is required")
	}
	if p.DevicesSupport == nil {
		p.DevicesSupport = DevicesSupport{}
	}
	if len(p.FeatureList) == 0 {
		p.FeatureList = json.RawMessage("{}")
	}
	return &p, nil
}

// isValidSystemLicenseType 检查 license_type 是否在枚举内。
func isValidSystemLicenseType(t string) bool {
	switch SystemLicenseType(t) {
	case SystemLicenseTypeCommercial,
		SystemLicenseTypeTrial,
		SystemLicenseTypeEvaluation,
		SystemLicenseTypeInternal:
		return true
	}
	return false
}
