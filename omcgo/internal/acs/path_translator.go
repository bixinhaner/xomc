// Package acs path_translator.go — T-XXX ACS 端 standardPath → privatePath 翻译。
//
// 设计背景：早期实现把翻译放在 App 进程的 mml fanout 阶段，导致 device_tasks.tr069_params
// 存的是 privatePath。问题：
//   - 队列里 pending 任务无法跟进字典更新（admin 修映射后旧 task 仍用旧 privatePath）
//   - 任务表存私有 path,不利于审计/重试/排查
//
// 改造后：MML 入队存 standardPath；ACS 出队时调用 PathTranslationService.Translate
// 翻为 privatePath 再 SOAP 下发。任一步失败 fallback 原样直发（与原 fanout fallback 一致）。
//
// 翻译涉及的 RPC 与字段：
//
//	GPV / GPA          → params.names[]
//	SPV / SPA          → params.values[].name
//	GPN                → params.path
//	AddObject / Delete → params.object_name
//	Reboot / Download / Upload / FactoryReset / GetRPCMethods → 无 path 字段，透传
package acs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

// pathTranslatorDeviceLookup 按 SN 反查 *coremodel.Device，仅取 ProductClass +
// FirmwareVersion。生产由 device.PgDeviceRepository 满足；测试可 stub。
type pathTranslatorDeviceLookup interface {
	GetBySerialNumber(ctx context.Context, sn string) (*coremodel.Device, error)
}

// pathTranslatorProductMatcher 把 productClass 路由到 product。由 *product.Registry 满足。
type pathTranslatorProductMatcher interface {
	MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error)
}

// pathTranslatorFactory 取 (productID, swVersion) 的 Translator。由 *parammodel.Registry 满足。
type pathTranslatorFactory interface {
	Translator(ctx context.Context, productID uuid.UUID, swVersion string) (*parammodel.Translator, error)
}

// PathTranslationService 把 task.Params 内的 standardPath 翻译为 privatePath。
//
// 缺省语义（任一依赖未注入或任一步失败）：原样返回 task.Params，调用方继续按原流程
// BuildRequest —— 这是退化兼容（与旧的 App fanout fallback 行为一致）。
type PathTranslationService struct {
	devices    pathTranslatorDeviceLookup
	products   pathTranslatorProductMatcher
	translator pathTranslatorFactory
	logger     *zap.Logger
}

// NewPathTranslationService 构造服务。任一依赖为 nil → 后续翻译退化为透传。
func NewPathTranslationService(
	devices pathTranslatorDeviceLookup,
	products pathTranslatorProductMatcher,
	translator pathTranslatorFactory,
	logger *zap.Logger,
) *PathTranslationService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PathTranslationService{
		devices:    devices,
		products:   products,
		translator: translator,
		logger:     logger.Named("acs-path-translator"),
	}
}

// Enabled 报告服务是否具备翻译能力(依赖均已注入)。供 handler 跳过 noop 路径。
func (s *PathTranslationService) Enabled() bool {
	return s != nil && s.devices != nil && s.products != nil && s.translator != nil
}

// TranslateTaskParams 翻译 task.Params 内的 path 字段并返回新的 json.RawMessage。
//
// 返回 (新 params, true) 表示发生过翻译(可能仍有 miss,但至少一条命中);
// 返回 (原 params, false) 表示完全没翻译(透传)。
// 永不返回 error —— 任一步失败 fallback 原 params + WARN 日志,与旧 fanout 一致。
//
// 注:即使所有 path 在 Translator 内都 Found=false(完全 miss),也返回 (新 params, true)
// 但新 params 与原 params 等价(原样写回)。这种语义让上层无需感知是否真的 transform 过,
// 仅需判断"是否走过翻译路径"。
func (s *PathTranslationService) TranslateTaskParams(ctx context.Context, t *task.Task) (json.RawMessage, bool) {
	if !s.Enabled() || t == nil || len(t.Params) == 0 || !methodNeedsTranslation(t.Method) {
		return t.Params, false
	}
	if taskParamsUsePrivatePathMode(t.Params) {
		return t.Params, false
	}

	tr, ok := s.resolveTranslator(ctx, t.DeviceSN)
	if !ok {
		return t.Params, false
	}

	translated, err := translateParamsByMethod(t.Method, t.Params, tr)
	if err != nil {
		s.logger.Warn("path translation skipped: params shape mismatch",
			zap.String("device_sn", t.DeviceSN),
			zap.String("method", t.Method),
			zap.String("task_id", t.ID),
			zap.Error(err))
		return t.Params, false
	}
	return translated, true
}

func taskParamsUsePrivatePathMode(raw json.RawMessage) bool {
	var payload struct {
		PathMode string `json:"path_mode"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(payload.PathMode), "private")
}

// resolveTranslator 按设备 SN 解析出对应的 Translator（SN→product→param_model）。
// 出站（TranslateTaskParams）与入站（TranslateResponseNames）共用同一解析链。
// 任一步失败 → (nil, false)，调用方退化为透传（与改造前 fallback 一致）。
func (s *PathTranslationService) resolveTranslator(ctx context.Context, deviceSN string) (*parammodel.Translator, bool) {
	device, err := s.devices.GetBySerialNumber(ctx, deviceSN)
	if err != nil || device == nil {
		s.logger.Warn("path translation skipped: device lookup failed",
			zap.String("device_sn", deviceSN), zap.Error(err))
		return nil, false
	}

	matchRes, err := s.products.MatchProductClass(ctx, device.ProductClass)
	if err != nil || matchRes == nil || matchRes.Product == nil || matchRes.Product.ParamModelID == nil {
		// orphan device 不算严重错误（与 App fanout orphan_passthrough 一致）：WARN 不阻断。
		if !errors.Is(err, product.ErrOrphan) && err != nil {
			s.logger.Warn("path translation skipped: product match failed",
				zap.String("device_sn", deviceSN),
				zap.String("product_class", device.ProductClass), zap.Error(err))
		}
		return nil, false
	}

	tr, err := s.translator.Translator(ctx, matchRes.Product.ID, device.FirmwareVersion)
	if err != nil || tr == nil {
		if err != nil && !errors.Is(err, parammodel.ErrNoMapping) {
			s.logger.Warn("path translation skipped: translator unavailable",
				zap.String("device_sn", deviceSN),
				zap.String("product_class", device.ProductClass),
				zap.String("software_version", device.FirmwareVersion), zap.Error(err))
		}
		return nil, false
	}
	return tr, true
}

// ResolveUECountPaths returns the concrete standard UE Count paths supported
// by the device's current product mapping. The root path represents physical
// cell 1; if an explicit ".1" alias also exists, the root path wins.
func (s *PathTranslationService) ResolveUECountPaths(ctx context.Context, deviceSN string) ([]string, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("ACS path translator is disabled")
	}
	tr, ok := s.resolveTranslator(ctx, deviceSN)
	if !ok {
		return nil, fmt.Errorf("translator unavailable for device %s", deviceSN)
	}
	return supportedUECountPaths(tr.Mappings()), nil
}

func supportedUECountPaths(mappings []parammodel.ParamMapping) []string {
	pathsByCell := make(map[int]string)
	for _, mapping := range mappings {
		if !mapping.IsActive ||
			!mapping.IsSupported ||
			!strings.EqualFold(mapping.EntryType, "parameter") {
			continue
		}
		cellIndex, root, ok := ueCountCellIndex(mapping.StandardPath)
		if !ok {
			continue
		}
		if current, exists := pathsByCell[cellIndex]; !exists || root || current == "" {
			pathsByCell[cellIndex] = mapping.StandardPath
		}
	}

	indices := make([]int, 0, len(pathsByCell))
	for index := range pathsByCell {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	paths := make([]string, 0, len(indices))
	for _, index := range indices {
		paths = append(paths, pathsByCell[index])
	}
	return paths
}

func ueCountCellIndex(path string) (index int, root bool, ok bool) {
	const (
		rootPath = "Device.DeviceInfo.UE_Count"
		prefix   = "Device.DeviceInfo."
		suffix   = ".UE_Count"
	)
	if path == rootPath {
		return 1, true, true
	}
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return 0, false, false
	}
	rawIndex := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	index, err := strconv.Atoi(rawIndex)
	if err != nil || index <= 0 {
		return 0, false, false
	}
	return index, false, true
}

// TranslateResponseNames 把基站响应里的参数名（私有 path）回译为标准 path（issue #424）。
//
// 与出站 TranslateTaskParams 对称、共用 resolveTranslator；用于 GPV/GPA 响应的
// parameter_values[].name。返回**新切片**（不修改入参，遵不可变约定）；
// (translated, true) 表示走过翻译路径，(原值拷贝, false) 表示透传（依赖缺失/解析失败）。
// 单个 name 在 Translator 内 Found=false 时原样保留（私有 path 兜底，不丢值）。
func (s *PathTranslationService) TranslateResponseNames(ctx context.Context, deviceSN string, names []string) ([]string, bool) {
	out := make([]string, len(names))
	copy(out, names)
	if !s.Enabled() || len(names) == 0 {
		return out, false
	}
	tr, ok := s.resolveTranslator(ctx, deviceSN)
	if !ok {
		return out, false
	}
	for i, n := range out {
		if n == "" {
			continue
		}
		if res := tr.ToStandard(n); res.Found {
			out[i] = res.Translated
		}
	}
	return out, true
}

// TranslateResponseNamesForRequest 回译 GPV 响应时优先使用本次请求的 standard names 作为锚点。
// 这能保留调用方已填好的上层实例号：例如请求
// Device.Services.FAPService.1.FAPControl.LTE.LICENSE.Author 被 ACS 翻译成
// Device.FAP.License.Author 下发后，响应里的私有 path 必须回到带 ".1." 的请求 path，
// 不能退化成映射模板里的 "{i}"。
func (s *PathTranslationService) TranslateResponseNamesForRequest(
	ctx context.Context,
	deviceSN string,
	names []string,
	requestParams json.RawMessage,
) ([]string, bool) {
	out := make([]string, len(names))
	copy(out, names)
	if !s.Enabled() || len(names) == 0 {
		return out, false
	}
	tr, ok := s.resolveTranslator(ctx, deviceSN)
	if !ok {
		return out, false
	}
	requestedStandards := requestStandardNames(requestParams)
	anchors := responseNameAnchors(requestParams, tr)
	for i, n := range out {
		if n == "" {
			continue
		}
		if std, ok := matchResponseNameAnchor(n, anchors); ok {
			out[i] = std
			continue
		}
		if res := tr.ToStandard(n); res.Found {
			if !strings.Contains(res.Translated, "{i}") {
				out[i] = res.Translated
				continue
			}
			if concrete, ok := instantiateStandardTemplateFromRequests(res.Translated, requestedStandards); ok {
				out[i] = concrete
			}
		}
	}
	return out, true
}

type responseNameAnchor struct {
	private  string
	standard string
}

func responseNameAnchors(requestParams json.RawMessage, tr *parammodel.Translator) []responseNameAnchor {
	if len(requestParams) == 0 || taskParamsUsePrivatePathMode(requestParams) {
		return nil
	}
	requested := requestStandardNames(requestParams)
	if len(requested) == 0 {
		return nil
	}
	anchors := make([]responseNameAnchor, 0, len(requested))
	for _, standard := range requested {
		standard = strings.TrimSpace(standard)
		if standard == "" {
			continue
		}
		candidates := tr.ToPrivateCandidates(standard)
		if len(candidates) == 0 {
			anchors = appendResponseNameAnchor(anchors, standard, standard)
			continue
		}
		for _, candidate := range candidates {
			anchors = appendResponseNameAnchor(anchors, candidate.Translated, standard)
		}
	}
	sortResponseNameAnchors(anchors)
	return anchors
}

func requestStandardNames(requestParams json.RawMessage) []string {
	if len(requestParams) == 0 || taskParamsUsePrivatePathMode(requestParams) {
		return nil
	}
	var p struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(requestParams, &p); err != nil || len(p.Names) == 0 {
		return nil
	}
	return p.Names
}

func appendResponseNameAnchor(anchors []responseNameAnchor, private, standard string) []responseNameAnchor {
	private = strings.TrimSpace(private)
	standard = strings.TrimSpace(standard)
	if private == "" || standard == "" {
		return anchors
	}
	for _, anchor := range anchors {
		if anchor.private == private && anchor.standard == standard {
			return anchors
		}
	}
	return append(anchors, responseNameAnchor{private: private, standard: standard})
}

func sortResponseNameAnchors(anchors []responseNameAnchor) {
	sort.SliceStable(anchors, func(i, j int) bool {
		return len(anchors[i].private) > len(anchors[j].private)
	})
}

func matchResponseNameAnchor(name string, anchors []responseNameAnchor) (string, bool) {
	for _, anchor := range anchors {
		if name == anchor.private {
			return anchor.standard, true
		}
		if strings.HasSuffix(anchor.private, ".") && strings.HasSuffix(anchor.standard, ".") &&
			strings.HasPrefix(name, anchor.private) {
			return anchor.standard + strings.TrimPrefix(name, anchor.private), true
		}
	}
	return "", false
}

func instantiateStandardTemplateFromRequests(template string, requests []string) (string, bool) {
	for _, request := range requests {
		if concrete, ok := instantiateStandardTemplateFromRequest(template, request); ok {
			return concrete, true
		}
	}
	return "", false
}

func instantiateStandardTemplateFromRequest(template, request string) (string, bool) {
	template = strings.TrimSpace(template)
	request = strings.TrimSpace(request)
	if template == "" || request == "" {
		return "", false
	}
	templateParts := strings.Split(strings.TrimSuffix(template, "."), ".")
	requestParts := strings.Split(strings.TrimSuffix(request, "."), ".")
	if len(requestParts) > len(templateParts) {
		return "", false
	}
	values := make([]string, 0, 4)
	for i, reqPart := range requestParts {
		tplPart := templateParts[i]
		if tplPart == "{i}" {
			if !isDecimalSegment(reqPart) {
				return "", false
			}
			values = append(values, reqPart)
			continue
		}
		if tplPart != reqPart {
			return "", false
		}
	}
	if len(values) == 0 {
		return "", false
	}
	out := make([]string, len(templateParts))
	copy(out, templateParts)
	next := 0
	for i, part := range out {
		if part != "{i}" {
			continue
		}
		if next >= len(values) {
			return "", false
		}
		out[i] = values[next]
		next++
	}
	return strings.Join(out, "."), true
}

func isDecimalSegment(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// methodNeedsTranslation 报告哪些 RPC 方法的 params 含 path 字段。
func methodNeedsTranslation(method string) bool {
	switch method {
	case "GetParameterValues", "GetParameterAttributes",
		"SetParameterValues", "SetParameterAttributes",
		"GetParameterNames",
		"AddObject", "DeleteObject":
		return true
	default:
		return false
	}
}

// translateParamsByMethod 按 RPC 类型选择性翻译 params 内的 path 字段。
func translateParamsByMethod(method string, raw json.RawMessage, tr *parammodel.Translator) (json.RawMessage, error) {
	switch method {
	case "GetParameterValues", "GetParameterAttributes":
		return translateNamesArray(raw, tr)
	case "SetParameterValues", "SetParameterAttributes":
		return translateValuesArray(raw, tr)
	case "GetParameterNames":
		return translateSinglePath(raw, tr)
	case "AddObject", "DeleteObject":
		return translateObjectName(raw, tr)
	default:
		return raw, nil
	}
}

// translateNamesArray 翻译 {names: [string]} 形态(GPV / GPA)。
func translateNamesArray(raw json.RawMessage, tr *parammodel.Translator) (json.RawMessage, error) {
	var p struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("unmarshal names: %w", err)
	}
	names := make([]string, 0, len(p.Names))
	seen := make(map[string]struct{}, len(p.Names))
	appendUnique := func(name string) {
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	for _, n := range p.Names {
		candidates := tr.ToPrivateCandidates(n)
		if len(candidates) == 0 {
			appendUnique(n)
			continue
		}
		for _, candidate := range candidates {
			appendUnique(candidate.Translated)
		}
	}
	p.Names = names
	return json.Marshal(p)
}

// translateValuesArray 翻译 {values: [{name, value, type}]} 形态(SPV / SPA)。
// 用 map[string]interface{} 解析以容纳厂商扩展字段(如 access)。
func translateValuesArray(raw json.RawMessage, tr *parammodel.Translator) (json.RawMessage, error) {
	var p struct {
		Values []map[string]interface{} `json:"values"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("unmarshal values: %w", err)
	}
	for _, v := range p.Values {
		nameRaw, ok := v["name"]
		if !ok {
			continue
		}
		name, ok := nameRaw.(string)
		if !ok || name == "" {
			continue
		}
		if res := tr.ToPrivate(name); res.Found {
			if res.Mapping != nil && strings.TrimSpace(res.Mapping.DataType) != "" {
				if value, valueOK := v["value"].(string); valueOK {
					v["value"] = tr069.NormalizeValueForPath(name, value, res.Mapping.DataType)
				}
				v["type"] = tr069.XSDTypeForPath(name, res.Mapping.DataType)
			}
			v["name"] = res.Translated
		}
	}
	return json.Marshal(p)
}

// translateSinglePath 翻译 {path, next_level} 形态(GPN)。
func translateSinglePath(raw json.RawMessage, tr *parammodel.Translator) (json.RawMessage, error) {
	var p struct {
		Path      string `json:"path"`
		NextLevel bool   `json:"next_level"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("unmarshal path: %w", err)
	}
	if p.Path != "" {
		if res := tr.ToPrivate(p.Path); res.Found {
			p.Path = res.Translated
		}
	}
	return json.Marshal(p)
}

// translateObjectName 翻译 {object_name} 形态(AddObject / DeleteObject)。
func translateObjectName(raw json.RawMessage, tr *parammodel.Translator) (json.RawMessage, error) {
	var p map[string]interface{}
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("unmarshal object_name: %w", err)
	}
	nameRaw, ok := p["object_name"]
	if !ok {
		return raw, nil
	}
	name, ok := nameRaw.(string)
	if !ok || name == "" {
		return raw, nil
	}
	if res := tr.ToPrivate(name); res.Found {
		p["object_name"] = res.Translated
	}
	return json.Marshal(p)
}
