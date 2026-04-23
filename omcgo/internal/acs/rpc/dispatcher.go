package rpc

import (
	"encoding/json"
	"fmt"
	"strings"

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
	DownloadBaseURL string // Download server base URL, e.g. "http://localhost:8080"
	DownloadPath    string // Download path prefix, e.g. "/smallcell/FileDownloadService"
	DownloadUser    string // Download HTTP Basic Auth username
	DownloadPass    string // Download HTTP Basic Auth password
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
	})
	d.Register("Upload", &UploadHandler{})
	d.Register("Reboot", &RebootHandler{})
	d.Register("FactoryReset", &FactoryResetHandler{})
	d.Register("GetParameterAttributes", &GetParameterAttributesHandler{})
	d.Register("SetParameterAttributes", &SetParameterAttributesHandler{})

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
	if h.DownloadBaseURL != "" && params.URL != "" && !strings.Contains(params.URL, "://") {
		params.URL = strings.TrimRight(h.DownloadBaseURL, "/") + h.DownloadPath + "/" + params.URL
		// Inject download credentials if not already set
		if params.Username == "" && h.DownloadUser != "" {
			params.Username = h.DownloadUser
		}
		if params.Password == "" && h.DownloadPass != "" {
			params.Password = h.DownloadPass
		}
	}

	return soap.RenderResponse(soap.DownloadTmpl, params)
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
