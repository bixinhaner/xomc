package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
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
	DownloadBaseURL        string // Download server base URL, e.g. "http://localhost:8080"
	DownloadPath           string // Download path prefix, e.g. "/smallcell/FileDownloadService"
	DownloadUser           string // Download HTTP Basic Auth username
	DownloadPass           string // Download HTTP Basic Auth password
	TransferConfigProvider transfercfg.Provider
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
	})
	d.Register("Upload", &UploadHandler{})
	d.Register("Reboot", &RebootHandler{})
	d.Register("FactoryReset", &FactoryResetHandler{})
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

	data := soap.SetParameterValuesData{
		ID:  cmd.CWMPID,
		Key: cmd.CommandKey,
	}
	for _, v := range params.Values {
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
	current := h.currentSettings()
	if current.BaseURL != "" && params.URL != "" && !strings.Contains(params.URL, "://") {
		params.URL = strings.TrimRight(current.BaseURL, "/") + current.Path + "/" + params.URL
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
