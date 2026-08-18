package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/pkg/soap"
)

// RPCHandler handles building requests and processing responses for a specific RPC method.
type RPCHandler interface {
	BuildRequest(cmd *Command) ([]byte, error)
}

// Dispatcher routes commands to the appropriate RPC handler and builds SOAP requests.
type Dispatcher struct {
	handlers map[string]RPCHandler
}

// DispatcherConfig holds optional configuration for the RPC dispatcher.
type DispatcherConfig struct {
	DownloadBaseURL         string // Download server base URL, e.g. "http://localhost:8080"
	DownloadPath            string // Download path prefix, e.g. "/smallcell/FileDownloadService"
	DownloadUser            string // Download HTTP Basic Auth username
	DownloadPass            string // Download HTTP Basic Auth password
	TransferConfigProvider  transfercfg.Provider
	TransferAddressResolver transferAddressResolver
	DownloadDeviceLookup    DownloadDeviceLookup
}

type transferAddressResolver interface {
	Resolve(context.Context, uuid.UUID, transfercfg.TransferDirection) (transfercfg.AddressDecision, error)
}

type DownloadDeviceLookup interface {
	GetBySerialNumber(context.Context, string) (*model.Device, error)
}

// NewDispatcher creates a new RPC dispatcher with all standard handlers registered.
func NewDispatcher(cfgs ...DispatcherConfig) *Dispatcher {
	d := &Dispatcher{
		handlers: make(map[string]RPCHandler),
	}

	var cfg DispatcherConfig
	if len(cfgs) > 0 {
		cfg = cfgs[0]
	}

	d.Register("GetParameterValues", &GetParameterValuesHandler{})
	d.Register("SetParameterValues", &SetParameterValuesHandler{})
	d.Register("GetParameterNames", &GetParameterNamesHandler{})
	d.Register("AddObject", &AddObjectHandler{})
	d.Register("DeleteObject", &DeleteObjectHandler{})
	d.Register("Download", &DownloadHandler{
		DownloadBaseURL: cfg.DownloadBaseURL,
		DownloadPath:    cfg.DownloadPath,
		DownloadUser:    cfg.DownloadUser,
		DownloadPass:    cfg.DownloadPass,
		ConfigProvider:  cfg.TransferConfigProvider,
		AddressResolver: cfg.TransferAddressResolver,
		DeviceLookup:    cfg.DownloadDeviceLookup,
	})
	d.Register("Upload", &UploadHandler{})
	d.Register("Reboot", &RebootHandler{})
	d.Register("FactoryReset", &FactoryResetHandler{})
	d.Register("X_BAICELLS_COM_PasswordReset", &ResetLMTPasswordHandler{})
	d.Register("X_COMMON_COM_PasswordReset", &ResetLMTPasswordHandler{})
	d.Register("GetParameterAttributes", &GetParameterAttributesHandler{})
	d.Register("SetParameterAttributes", &SetParameterAttributesHandler{})
	d.Register("GetRPCMethods", &GetRPCMethodsHandler{})

	return d
}

// Register adds a handler for the given RPC method.
func (d *Dispatcher) Register(method string, handler RPCHandler) {
	d.handlers[method] = handler
}

// BuildRequest builds a SOAP request for the given command.
// cwmpID is used as the SOAP Header cwmp:ID (distinct from CommandKey which is the RPC-level key).
func (d *Dispatcher) BuildRequest(cmd *Command, cwmpID string) ([]byte, error) {
	handler, ok := d.handlers[cmd.Method]
	if !ok {
		return nil, fmt.Errorf("unknown RPC method: %s", cmd.Method)
	}
	// Set CWMPID on command so handlers can use it for the SOAP header ID
	cmd.CWMPID = cwmpID
	return handler.BuildRequest(cmd)
}

// --- Individual RPC Handlers ---

type GetParameterValuesHandler struct{}

func (h *GetParameterValuesHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse GetParameterValues params: %w", err)
	}

	data := soap.GetParameterValuesData{ID: cmd.CWMPID}
	for _, name := range params.Names {
		data.Params = append(data.Params, soap.ParameterNameData{Name: name})
	}
	return soap.RenderResponse(soap.GetParameterValuesTmpl, data)
}

type SetParameterValuesHandler struct{}

func (h *SetParameterValuesHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params struct {
		Values []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
			Type  string `json:"type"`
		} `json:"values"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse SetParameterValues params: %w", err)
	}
	if len(params.Values) == 0 {
		return nil, fmt.Errorf("build SetParameterValues request: values must not be empty")
	}

	data := soap.SetParameterValuesData{
		ID:  cmd.CWMPID,
		Key: cmd.CommandKey,
	}
	for _, v := range params.Values {
		if strings.TrimSpace(v.Name) == "" {
			return nil, fmt.Errorf("build SetParameterValues request: parameter name must not be empty")
		}
		typ := v.Type
		if typ == "" {
			typ = "xsd:string"
		}
		data.Params = append(data.Params, soap.ParameterSetData{
			Name: v.Name, Value: v.Value, Type: typ,
		})
	}
	return soap.RenderResponse(soap.SetParameterValuesTmpl, data)
}

type GetParameterNamesHandler struct{}

func (h *GetParameterNamesHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params struct {
		Path      string `json:"path"`
		NextLevel bool   `json:"next_level"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse GetParameterNames params: %w", err)
	}

	data := soap.GetParameterNamesData{
		ID: cmd.CWMPID, Path: params.Path, NextLevel: params.NextLevel,
	}
	return soap.RenderResponse(soap.GetParameterNamesTmpl, data)
}

type AddObjectHandler struct{}

func (h *AddObjectHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params struct {
		ObjectName string `json:"object_name"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse AddObject params: %w", err)
	}
	data := soap.AddObjectData{ID: cmd.CWMPID, ObjectName: params.ObjectName, Key: cmd.CommandKey}
	return soap.RenderResponse(soap.AddObjectTmpl, data)
}

type DeleteObjectHandler struct{}

func (h *DeleteObjectHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params struct {
		ObjectName string `json:"object_name"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse DeleteObject params: %w", err)
	}
	data := soap.DeleteObjectData{ID: cmd.CWMPID, ObjectName: params.ObjectName, Key: cmd.CommandKey}
	return soap.RenderResponse(soap.DeleteObjectTmpl, data)
}

// DownloadHandler handles Download RPC requests.
// When DownloadBaseURL is set, it translates internal MinIO paths to HTTP URLs
// that CPE can access through the gateway/ACS download endpoint.
// Internal paths are plain "bucket/object/path" (no scheme), e.g. "firmware/v2.0.bin".
// URLs with a scheme (http://, https://, ftp://) are passed through unchanged.
type DownloadHandler struct {
	DownloadBaseURL string // e.g. "http://localhost:8080"
	DownloadPath    string // e.g. "/smallcell/FileDownloadService"
	DownloadUser    string // HTTP Basic Auth credentials injected into download URL
	DownloadPass    string
	ConfigProvider  transfercfg.Provider
	AddressResolver transferAddressResolver
	DeviceLookup    DownloadDeviceLookup
}

func (h *DownloadHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params soap.DownloadData
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse Download params: %w", err)
	}
	params.ID = cmd.CWMPID
	params.CommandKey = cmd.CommandKey

	// Translate internal MinIO path to HTTP download endpoint URL.
	// Plain path (no "://" scheme) is treated as MinIO bucket/object path:
	//   firmware/v2.0.bin → {BaseURL}{Path}/firmware/v2.0.bin
	// URLs with a scheme (http://, https://, ftp://) are passed through unchanged.
	//
	// Username / Password 不再自动从 transfercfg.Download 注入——产品线要求 Download
	// 不走 HTTP Basic Auth（与 Upload 对齐），CPE 拿到的 <cwmp:Username></cwmp:Username>
	// 应该是空标签。Params.URL 上层若已显式塞凭据走透传；空字符串则渲染成空标签。
	params.URL = appconfig.NormalizeConfigBackupReference(params.URL)
	current := h.currentSettings()
	if strings.TrimSpace(params.URL) != "" {
		rewrittenURL, ok, err := h.resolvePolicyDownloadURL(cmd, params.URL, current)
		if err != nil {
			return nil, err
		}
		if ok {
			params.URL = rewrittenURL
		}
	}
	if current.BaseURL != "" && params.URL != "" && !strings.Contains(params.URL, "://") {
		servicePath := current.Path
		if servicePath == "" {
			servicePath = "/smallcell/FileDownloadService"
		}
		objectPath := strings.Trim(params.URL, "/")
		if objectPath == "" {
			return nil, fmt.Errorf("build Download URL: object path is required")
		}
		builtURL, err := transfercfg.BuildURL(current.BaseURL, servicePath, strings.Split(objectPath, "/"), nil)
		if err != nil {
			return nil, fmt.Errorf("build Download URL: %w", err)
		}
		params.URL = builtURL
	}

	return soap.RenderResponse(soap.DownloadTmpl, params)
}

func (h *DownloadHandler) currentSettings() transfercfg.DownloadSettings {
	if h.ConfigProvider != nil {
		return h.ConfigProvider.Snapshot(context.Background()).Download
	}
	return transfercfg.DownloadSettings{
		BaseURL:  h.DownloadBaseURL,
		Path:     h.DownloadPath,
		Username: h.DownloadUser,
		Password: h.DownloadPass,
	}
}

func (h *DownloadHandler) resolvePolicyDownloadURL(
	cmd *Command,
	rawURL string,
	current transfercfg.DownloadSettings,
) (string, bool, error) {
	servicePath := current.Path
	if servicePath == "" {
		servicePath = "/smallcell/FileDownloadService"
	}
	objectSegments, ok, err := downloadObjectSegments(rawURL, servicePath)
	if err != nil {
		return "", false, fmt.Errorf("resolve policy-managed Download URL: %w", err)
	}
	if !ok {
		return "", false, nil
	}
	params := downloadPolicyParams(cmd)
	if !isPolicyResolvedDownload(params.policyCommandKey(cmd.CommandKey), params.FileType, params.TransferPolicyManaged, objectSegments) {
		return "", false, nil
	}
	if h.AddressResolver == nil {
		return "", false, fmt.Errorf("resolve policy-managed Download URL: transfer address resolver is required")
	}
	if h.DeviceLookup == nil {
		return "", false, fmt.Errorf("resolve policy-managed Download URL: device lookup is required")
	}
	deviceSN := strings.TrimSpace(cmd.DeviceSN)
	if deviceSN == "" {
		return "", false, fmt.Errorf("resolve policy-managed Download URL: device serial number is required")
	}
	device, err := h.DeviceLookup.GetBySerialNumber(context.Background(), deviceSN)
	if err != nil {
		return "", false, fmt.Errorf("lookup device %q for policy-managed Download URL: %w", deviceSN, err)
	}
	if device == nil {
		return "", false, fmt.Errorf("lookup device %q for policy-managed Download URL: not found", deviceSN)
	}
	decision, err := h.AddressResolver.Resolve(context.Background(), device.ID, transfercfg.TransferDirectionDownload)
	if err != nil {
		return "", false, fmt.Errorf("resolve policy-managed Download address: %w", err)
	}
	builtURL, err := transfercfg.BuildURL(decision.BaseURL, servicePath, objectSegments, nil)
	if err != nil {
		return "", false, fmt.Errorf("build policy-managed Download URL: %w", err)
	}
	return builtURL, true, nil
}

type policyDownloadParams struct {
	CommandKey            string `json:"command_key"`
	FileType              string `json:"file_type"`
	TransferPolicyManaged bool   `json:"transfer_policy_managed"`
}

func downloadPolicyParams(cmd *Command) policyDownloadParams {
	var params policyDownloadParams
	_ = json.Unmarshal(cmd.Params, &params)
	return params
}

func (p policyDownloadParams) policyCommandKey(fallback string) string {
	if strings.TrimSpace(p.CommandKey) != "" {
		return p.CommandKey
	}
	return fallback
}

func isPolicyResolvedDownload(commandKey, fileType string, transferPolicyManaged bool, objectSegments []string) bool {
	return isConfigRestoreDownloadCommand(commandKey) ||
		isLicenseDownloadCommand(commandKey) ||
		(transferPolicyManaged &&
			isSoftwareUpgradeDownloadCommand(commandKey) &&
			isSoftwareUpgradeDownloadFileType(fileType) &&
			isManagedSoftwareUpgradeObjectSegments(objectSegments))
}

func isConfigRestoreDownloadCommand(commandKey string) bool {
	commandKey = strings.TrimSpace(commandKey)
	return strings.HasPrefix(commandKey, "CONFIG_RESTORE_") ||
		strings.Contains(commandKey, "_RESTORE_") ||
		strings.HasPrefix(commandKey, "restore-")
}

func isLicenseDownloadCommand(commandKey string) bool {
	commandKey = strings.TrimSpace(commandKey)
	return strings.HasPrefix(commandKey, "LICENSE_UPGRADE_") ||
		strings.HasPrefix(commandKey, "LICENSE_PREINSTALL_")
}

func isSoftwareUpgradeDownloadCommand(commandKey string) bool {
	return strings.HasPrefix(strings.TrimSpace(commandKey), "Download Upgrade,")
}

func isSoftwareUpgradeDownloadFileType(fileType string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(fileType))
	return normalized == "1" ||
		strings.Contains(normalized, "FIRMWARE UPGRADE IMAGE") ||
		strings.Contains(normalized, "SOFTWARE UPGRADE PATCH") ||
		strings.Contains(normalized, "FIRMWARE UPGRADE FPGA")
}

func isManagedSoftwareUpgradeObjectSegments(segments []string) bool {
	if len(segments) < 2 || segments[0] != "firmware" {
		return false
	}
	switch strings.ToLower(segments[1]) {
	case "img", "patch", "fpga":
		return true
	default:
		return false
	}
}

func downloadObjectSegments(rawURL, servicePath string) ([]string, bool, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, false, nil
	}
	if !strings.Contains(rawURL, "://") {
		segments := splitObjectPath(rawURL)
		if len(segments) == 0 {
			return nil, false, fmt.Errorf("object path is required")
		}
		return segments, true, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, false, err
	}
	decodedPath, err := url.PathUnescape(parsed.EscapedPath())
	if err != nil {
		return nil, false, err
	}
	for _, marker := range downloadServicePathMarkers(servicePath) {
		prefix := strings.TrimRight(marker, "/") + "/"
		idx := strings.Index(decodedPath, prefix)
		if idx < 0 {
			continue
		}
		segments := splitObjectPath(decodedPath[idx+len(prefix):])
		if len(segments) == 0 {
			return nil, false, fmt.Errorf("object path is required")
		}
		return segments, true, nil
	}
	return nil, false, nil
}

func downloadServicePathMarkers(servicePath string) []string {
	servicePath = strings.TrimSpace(servicePath)
	if servicePath == "" {
		servicePath = "/smallcell/FileDownloadService"
	}
	markers := []string{servicePath}
	const canonical = "/smallcell/FileDownloadService"
	if servicePath != canonical {
		markers = append(markers, canonical)
	}
	return markers
}

func splitObjectPath(objectPath string) []string {
	parts := strings.Split(strings.Trim(objectPath, "/"), "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	return segments
}

type UploadHandler struct{}

func (h *UploadHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params soap.UploadData
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse Upload params: %w", err)
	}
	params.ID = cmd.CWMPID
	params.CommandKey = cmd.CommandKey
	return soap.RenderResponse(soap.UploadTmpl, params)
}

type RebootHandler struct{}

func (h *RebootHandler) BuildRequest(cmd *Command) ([]byte, error) {
	data := soap.RebootData{ID: cmd.CWMPID, CommandKey: cmd.CommandKey}
	return soap.RenderResponse(soap.RebootTmpl, data)
}

type FactoryResetHandler struct{}

func (h *FactoryResetHandler) BuildRequest(cmd *Command) ([]byte, error) {
	data := soap.FactoryResetData{ID: cmd.CWMPID}
	return soap.RenderResponse(soap.FactoryResetTmpl, data)
}

type ResetLMTPasswordHandler struct{}

func (h *ResetLMTPasswordHandler) BuildRequest(cmd *Command) ([]byte, error) {
	data := soap.ResetLMTPasswordData{ID: cmd.CWMPID, Method: cmd.Method, CommandKey: cmd.CommandKey}
	return soap.RenderResponse(soap.ResetLMTPasswordTmpl, data)
}

// GetRPCMethodsHandler 处理 GetRPCMethods RPC（TR-069 §A.3.1.2）。
// ACS 询问 CPE 支持的 RPC 方法列表 — 请求体仅 <cwmp:GetRPCMethods/> 空标签，
// 无 params；与 FactoryReset 同样的最小骨架。
//
// 由 ops 的 action="get_rpc_methods" 触发（T-0102-c actionToRPCMethod 映射），
// 用于运维侧探测设备实际支持哪些 RPC 方法，便于排查协议兼容性。
type GetRPCMethodsHandler struct{}

func (h *GetRPCMethodsHandler) BuildRequest(cmd *Command) ([]byte, error) {
	data := soap.GetRPCMethodsData{ID: cmd.CWMPID}
	return soap.RenderResponse(soap.GetRPCMethodsTmpl, data)
}

type GetParameterAttributesHandler struct{}

func (h *GetParameterAttributesHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse GetParameterAttributes params: %w", err)
	}

	data := soap.GetParameterAttributesData{ID: cmd.CWMPID}
	for _, name := range params.Names {
		data.Params = append(data.Params, soap.ParameterNameData{Name: name})
	}
	return soap.RenderResponse(soap.GetParameterAttributesTmpl, data)
}

type SetParameterAttributesHandler struct{}

func (h *SetParameterAttributesHandler) BuildRequest(cmd *Command) ([]byte, error) {
	var params struct {
		Attributes []struct {
			Name               string   `json:"name"`
			NotificationChange bool     `json:"notification_change"`
			Notification       int      `json:"notification"`
			AccessListChange   bool     `json:"access_list_change"`
			AccessList         []string `json:"access_list"`
		} `json:"attributes"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse SetParameterAttributes params: %w", err)
	}

	data := soap.SetParameterAttributesData{ID: cmd.CWMPID}
	for _, a := range params.Attributes {
		data.Params = append(data.Params, soap.SetParameterAttributeData{
			Name:               a.Name,
			NotificationChange: a.NotificationChange,
			Notification:       a.Notification,
			AccessListChange:   a.AccessListChange,
			AccessList:         a.AccessList,
		})
	}
	return soap.RenderResponse(soap.SetParameterAttributesTmpl, data)
}
