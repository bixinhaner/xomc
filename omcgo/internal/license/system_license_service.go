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
	usageRepo             SystemLicenseUsageRepository
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

// SetUsageRepo 注入累计使用时长仓储，启用 GetCurrent 响应里的累计状态填充
//（cumulative_used_hours / cumulative_limit_hours / is_expired）。
func (s *SystemLicenseService) SetUsageRepo(r SystemLicenseUsageRepository) {
	s.usageRepo = r
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
	s.enrichCumulativeStatus(ctx, lic)
	return lic, nil
}

// enrichCumulativeStatus 填充 is_expired / cumulative_used_hours / cumulative_limit_hours
// 三个 transient 字段。is_expired 覆盖日期过期 + 累计超限；累计值含未推进增量
// （now - last_visited）以保证展示与 enforcement 裁决口径一致。usageRepo 未注入
// 或读取失败时仅做日期过期判断（不阻塞响应）。
func (s *SystemLicenseService) enrichCumulativeStatus(ctx context.Context, lic *SystemLicense) {
	now := nowFunc()
	expired := lic.ExpiryDate != nil && now.After(*lic.ExpiryDate)

	if s.usageRepo != nil {
		timeLimit := extractTimeLimitHours(lic.FeatureList)
		if timeLimit > 0 {
			lic.CumulativeLimitHours = timeLimit
			total, lastVisited, err := s.usageRepo.CurrentUsage(ctx)
			if err == nil {
				if elapsed := now.Sub(lastVisited); elapsed > 0 {
					total += elapsed.Hours()
				}
				lic.CumulativeUsedHours = &total
				if total >= float64(timeLimit) {
					expired = true
				}
			} else {
				s.logger.Warn("enrich cumulative status: read usage failed", zap.Error(err))
			}
		}
	}
	lic.IsExpired = expired
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
	if _, ok := payload["authorization_tree"]; !ok {
		payload["authorization_tree"] = s.legacyFeatureMapping.AuthorizationTree(ids, codes)
	}
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

// HasActiveLicense reports whether an effective (configured AND non-expired)
// system license is in place. Satisfies admin.LicenseFeatureGate.
//   - No license (GetCurrent → 404 NotConfigured) → (false, nil)
//   - Expired license (IsExpired, 日期过期或累计超限) → (false, nil)：与无 license
//     同等处理，菜单只剩 /license，前端进入 licenseOnlyMode（issue #310）。
//   - Transient error → (false, err)，调用方在菜单路径 fail-open。
func (s *SystemLicenseService) HasActiveLicense(ctx context.Context) (bool, error) {
	lic, err := s.GetCurrent(ctx)
	if err != nil {
		if errors.Is(err, commonerrors.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if lic.IsExpired {
		return false, nil
	}
	return true, nil
}

// FilterAuthorized returns the subset of legacy feature codes authorized by the
// current system license. Satisfies admin.LicenseFeatureGate structurally (no
// import of admin). Fail-open: on no-license (GetCurrent 404) or transient
// error returns the input unchanged (menu visibility must not lock the UI out;
// device-level enforcement is fail-closed via the Enforcer).
//
// 过期 license（IsExpired）→ 返回空集：与无 license 的菜单裁剪口径一致，
// 菜单闸门已在 HasActiveLicense 处短路到 filterLicenseOnlyMenus，此处为防御
// 性兜底，避免未来直接调用方误把过期 license 当成全授权（issue #310）。
func (s *SystemLicenseService) FilterAuthorized(ctx context.Context, codes []string) ([]string, error) {
	lic, err := s.GetCurrent(ctx)
	if err != nil {
		return codes, nil
	}
	if lic.IsExpired {
		return nil, nil
	}
	tree := extractAuthorizationTree(lic.FeatureList)
	authorized := make([]string, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		path := codeToPath(code)
		if len(path) == 0 {
			continue
		}
		if HasFeature(FeatureList(tree), path...) {
			authorized = append(authorized, code)
		}
	}
	return authorized, nil
}

// CheckFeature evaluates one dot-separated feature path against the current license.
//
// 过期 license（IsExpired）→ (false, nil)：API 中间件 RequireFeature 据此
// fail-closed 返回 403，与设备 enforcer 的过期拦截口径一致（issue #310）。
// license 管理路由不挂 RequireFeature，上传/查看不受影响。
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
	if lic.IsExpired {
		return false, nil
	}
	return HasFeature(FeatureList(extractAuthorizationTree(lic.FeatureList)), parts...), nil
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
//  3. 硬件绑定校验（MAC / SystemUUID 不匹配 → 403）
//  4. 构造 SystemLicense + repo.Replace（事务 + SELECT FOR UPDATE，singleton 删除语义）
//
// 允许重传任意 license_id（含历史里用过的）：license 文件 license_id 由厂商固定签发，
// 传错后需能恢复。Replace 删旧行再插新行，不撞 UNIQUE；history 允许同 id 多行。
//
// 错误：
//   - JSON 解析 / 必填缺失 → 12111 (400)
//   - 签名校验失败（strict） → 12109 (400)
//   - 硬件绑定不匹配       → 12115 (403)
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

	// 硬件绑定校验（MAC / SystemUUID）：license 非空绑定且与本机不匹配 → 拒绝上传（403）。
	if hwErr := validateHardwareBinding(parsed.MACAddress, parsed.SystemUUID); hwErr != nil {
		return nil, commonerrors.NewBusinessError(global.ErrCodeSystemLicenseHardwareMismatch, hwErr.Error(), commonerrors.ErrLicenseHardwareMismatch)
	}

	// 注意：不再做 license_id 重复预检。license 文件的 license_id 由厂商固定签发，
	// 传错（如过期 license）后必须能重传原 license 恢复。Replace 采用 singleton 删除
	// 语义（DELETE 旧行再 INSERT 新行），不会撞 system_license_license_id_key UNIQUE；
	// history 表本就允许同 license_id 多行（仅非唯一索引）。Repo 内仍保留 UNIQUE 撞击
	// 翻译为 ErrSystemLicenseIDExists 作为并发兜底（SELECT FOR UPDATE 下理论不会触发）。

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
	// 复刻旧项目 FunctionCodeRelation 的 feature code 补全规则（sysList/gnbList/
	// getOrtherCode/delAdvanceControl/delAdvancePlug），使真实 .lic 的授权 code 集合与
	// 旧项目 supportFeatureCode 一致。无 mapping 时退回原始 codes。
	effectiveCodes := claims.FeatureCodes
	if s.legacyFeatureMapping != nil {
		effectiveCodes = s.legacyFeatureMapping.ExpandFeatureCodes(claims.FeatureIDs, claims.FeatureCodes, claims.IsCloud)
	}
	featurePayloadData := map[string]any{
		"legacy_feature_ids":   claims.FeatureIDs,
		"legacy_feature_codes": effectiveCodes,
		"time_limit_hours":     claims.TimeLimitHours,
	}
	if s.legacyFeatureMapping != nil {
		featurePayloadData["features"] = s.legacyFeatureMapping.Normalize(nil, effectiveCodes)
		featurePayloadData["authorization_tree"] = s.legacyFeatureMapping.AuthorizationTree(nil, effectiveCodes)
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
		MACAddress:     claims.MACAddress,
		SystemUUID:     claims.SystemUUID,
		TimeLimitHours: claims.TimeLimitHours,
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
	MACAddress     string          `json:"mac_address,omitempty"`
	SystemUUID     string          `json:"system_uuid,omitempty"`
	TimeLimitHours int             `json:"time_limit_hours,omitempty"`
}
