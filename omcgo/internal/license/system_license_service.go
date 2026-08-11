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
	"encoding/base64"
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
	repo                  SystemLicenseRepository
	logger                *zap.Logger
	enforcer              Enforcer // 可选；Update 成功后调 Invalidate 让 enforcer 重读
	legacyKeyStore        []byte
	legacyStorePassword   string
	legacyKeyAlias        string
	legacyVerifyIntegrity bool
	legacyFeatureMapping  *LegacyFeatureMapping
}

// NewSystemLicenseService 构造旧项目 License 业务 service。
func NewSystemLicenseService(repo SystemLicenseRepository, logger *zap.Logger) *SystemLicenseService {
	return &SystemLicenseService{
		repo:   repo,
		logger: logger.Named("system-license"),
	}
}

// SetEnforcer 注入 Enforcer，让 Update 成功后能调 Invalidate 让 enforcer 立即
// 拉新 license（避免 5min cache TTL 内 device.create 仍用旧容量裁决）。
// nil 安全：Update 路径跳过 Invalidate。
func (s *SystemLicenseService) SetEnforcer(e Enforcer) {
	s.enforcer = e
}

// SetLegacyTrueLicenseConfig configures the pure-Go decoder for legacy .lic
// files. The key store bytes are copied so callers may release their buffer.
func (s *SystemLicenseService) SetLegacyTrueLicenseConfig(keyStore []byte, password, alias string, verifyIntegrity bool) {
	s.legacyKeyStore = append([]byte(nil), keyStore...)
	s.legacyStorePassword = strings.TrimRight(password, "\r\n")
	s.legacyKeyAlias = alias
	s.legacyVerifyIntegrity = verifyIntegrity
}

// SetLegacyFeatureMapping injects the old project's ID/Code/menu mapping.
func (s *SystemLicenseService) SetLegacyFeatureMapping(mapping *LegacyFeatureMapping) {
	s.legacyFeatureMapping = mapping
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
	s.enrichLegacyFeatureList(&lic.FeatureList)
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
	for i := range resp.Items {
		s.enrichLegacyFeatureList(&resp.Items[i].FeatureList)
	}
	return resp, nil
}

// enrichLegacyFeatureList upgrades an older persisted feature snapshot at read
// time. Existing rows may predate the ID/Code mapping deployment; rewriting
// them on every read would create unrelated database writes, so the response
// receives the derived display objects without changing the stored License.
func (s *SystemLicenseService) enrichLegacyFeatureList(featureList *FeatureList) {
	if s.legacyFeatureMapping == nil || featureList == nil || len(*featureList) == 0 {
		return
	}
	var payload map[string]any
	if err := json.Unmarshal(*featureList, &payload); err != nil {
		return
	}
	ids := legacyPayloadStrings(payload["legacy_feature_ids"])
	codes := legacyPayloadStrings(payload["legacy_feature_codes"])
	if len(ids) == 0 && len(codes) == 0 {
		return
	}
	payload["features"] = s.legacyFeatureMapping.Normalize(ids, codes)
	encoded, err := json.Marshal(payload)
	if err == nil {
		*featureList = FeatureList(encoded)
	}
}

func legacyPayloadStrings(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			result = append(result, strings.TrimSpace(text))
		}
	}
	return result
}

// CheckFeature evaluates one dot-separated feature path against the current license.
func (s *SystemLicenseService) CheckFeature(ctx context.Context, path string) (bool, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return false, commonerrors.NewBusinessError(
			global.ErrCodeSystemLicenseInvalidFormat,
			"feature path is required",
			commonerrors.ErrInvalidInput,
		)
	}

	parts := strings.Split(path, ".")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return false, commonerrors.NewBusinessError(
				global.ErrCodeSystemLicenseInvalidFormat,
				"feature path contains an empty segment",
				commonerrors.ErrInvalidInput,
			)
		}
	}

	lic, err := s.GetCurrent(ctx)
	if err != nil {
		return false, err
	}
	return HasFeature(lic.FeatureList, parts...), nil
}

// UpdateRequest — POST /system-license 请求体。
//
// RawContent 是旧项目 TrueLicense .lic 二进制原文的 Base64 编码。JSON 只作为
// HTTP 请求封装，不是 License 文件格式。
type UpdateRequest struct {
	RawContent         string     `json:"raw_content"`
	RawContentEncoding string     `json:"raw_content_encoding"`
	UploadedByUserID   *uuid.UUID `json:"-"` // 由 handler 从 gin ctx 注入
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
//  1. 解码旧项目 TrueLicense .lic → parsedLicense
//  2. PBE/GZIP/XML 解密、JKS/DSA 验签和旧字段映射
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

	if req.RawContentEncoding != "base64" {
		return nil, commonerrors.NewBusinessError(
			global.ErrCodeSystemLicenseInvalidFormat,
			"raw_content must be a Base64-encoded legacy TrueLicense .lic file",
			commonerrors.ErrInvalidInput,
		)
	}

	parsed, sigStatus, sigNote, storedRawContent, err := s.parseLegacyUpdate(req.RawContent)
	if err != nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeSystemLicenseInvalidFormat, err.Error(), commonerrors.ErrInvalidInput)
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
		RawContent:       storedRawContent,
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

func (s *SystemLicenseService) parseLegacyUpdate(encodedRaw string) (*parsedLicense, SignatureStatus, string, string, error) {
	if len(s.legacyKeyStore) == 0 || strings.TrimSpace(s.legacyStorePassword) == "" || strings.TrimSpace(s.legacyKeyAlias) == "" {
		return nil, SignatureUnverified, "", "", errors.New("legacy TrueLicense keystore is not configured")
	}
	raw, err := base64.StdEncoding.DecodeString(encodedRaw)
	if err != nil {
		return nil, SignatureInvalid, "", "", fmt.Errorf("decode legacy license base64: %w", err)
	}
	artifact, err := DecodeLegacyTrueLicense(raw, s.legacyStorePassword)
	if err != nil {
		s.logger.Warn("legacy license decrypt failed",
			zap.Int("received_size", len(raw)),
			zap.String("received_sha256", hexSHA256(raw)),
			zap.Error(err))
		return nil, SignatureInvalid, "", "", fmt.Errorf(
			"%w (received_size=%d, received_sha256=%s)",
			err,
			len(raw),
			hexSHA256(raw),
		)
	}
	if err := VerifyLegacyTrueLicenseSignatureWithOptions(
		artifact,
		s.legacyKeyStore,
		s.legacyStorePassword,
		s.legacyKeyAlias,
		LegacyJKSOptions{VerifyIntegrity: s.legacyVerifyIntegrity},
	); err != nil {
		return nil, SignatureInvalid, "", "", err
	}
	claims, err := MapLegacyTrueLicense(artifact)
	if err != nil {
		return nil, SignatureInvalid, "", "", err
	}
	featurePayloadData := map[string]any{
		"legacy_feature_ids":   claims.FeatureIDs,
		"legacy_feature_codes": claims.FeatureCodes,
	}
	if s.legacyFeatureMapping != nil {
		featurePayloadData["features"] = s.legacyFeatureMapping.Normalize(claims.FeatureIDs, claims.FeatureCodes)
	}
	featurePayload, err := json.Marshal(featurePayloadData)
	if err != nil {
		return nil, SignatureInvalid, "", "", fmt.Errorf("marshal legacy feature list: %w", err)
	}
	var expiryDate *time.Time
	if claims.OMCNotAfter != nil {
		expiryDate = claims.OMCNotAfter
	} else {
		expiryDate = claims.StandardNotAfter
	}
	return &parsedLicense{
		LicenseID:      claims.LicenseID,
		LicenseType:    string(claims.LicenseType),
		IssuedAt:       claims.IssuedAt,
		ExpiryDate:     expiryDate,
		DevicesSupport: claims.DevicesSupport,
		FeatureList:    featurePayload,
		Signature:      &artifact.Signature,
	}, SignatureVerified, "legacy TrueLicense SHA1withDSA verified", base64.StdEncoding.EncodeToString(raw), nil
}

// parsedLicense 是旧 TrueLicense 字段映射后的中间结构，仅用于业务字段提取。
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
