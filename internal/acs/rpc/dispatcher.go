package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/omcgo/omcgo/internal/acs/cmdqueue"
	"github.com/omcgo/omcgo/pkg/soap"
)

// RPCHandler handles building requests and processing responses for a specific RPC method.
type RPCHandler interface {
	BuildRequest(cmd *cmdqueue.Command) ([]byte, error)
}

// Dispatcher routes commands to the appropriate RPC handler and builds SOAP requests.
type Dispatcher struct {
	handlers map[string]RPCHandler
}

// NewDispatcher creates a new RPC dispatcher with all standard handlers registered.
func NewDispatcher() *Dispatcher {
	d := &Dispatcher{
		handlers: make(map[string]RPCHandler),
	}

	d.Register("GetParameterValues", &GetParameterValuesHandler{})
	d.Register("SetParameterValues", &SetParameterValuesHandler{})
	d.Register("GetParameterNames", &GetParameterNamesHandler{})
	d.Register("AddObject", &AddObjectHandler{})
	d.Register("DeleteObject", &DeleteObjectHandler{})
	d.Register("Download", &DownloadHandler{})
	d.Register("Upload", &UploadHandler{})
	d.Register("Reboot", &RebootHandler{})
	d.Register("FactoryReset", &FactoryResetHandler{})

	return d
}

// Register adds a handler for the given RPC method.
func (d *Dispatcher) Register(method string, handler RPCHandler) {
	d.handlers[method] = handler
}

// BuildRequest builds a SOAP request for the given command.
func (d *Dispatcher) BuildRequest(cmd *cmdqueue.Command, cwmpID string) ([]byte, error) {
	handler, ok := d.handlers[cmd.Method]
	if !ok {
		return nil, fmt.Errorf("unknown RPC method: %s", cmd.Method)
	}
	return handler.BuildRequest(cmd)
}

// --- Individual RPC Handlers ---

type GetParameterValuesHandler struct{}

func (h *GetParameterValuesHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	var params struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse GetParameterValues params: %w", err)
	}

	data := soap.GetParameterValuesData{ID: cmd.CommandKey}
	for _, name := range params.Names {
		data.Params = append(data.Params, soap.ParameterNameData{Name: name})
	}
	return soap.RenderResponse(soap.GetParameterValuesTmpl, data)
}

type SetParameterValuesHandler struct{}

func (h *SetParameterValuesHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
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
		ID:  cmd.CommandKey,
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

func (h *GetParameterNamesHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	var params struct {
		Path      string `json:"path"`
		NextLevel bool   `json:"next_level"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse GetParameterNames params: %w", err)
	}

	data := soap.GetParameterNamesData{
		ID: cmd.CommandKey, Path: params.Path, NextLevel: params.NextLevel,
	}
	return soap.RenderResponse(soap.GetParameterNamesTmpl, data)
}

type AddObjectHandler struct{}

func (h *AddObjectHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	var params struct {
		ObjectName string `json:"object_name"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse AddObject params: %w", err)
	}
	data := soap.AddObjectData{ID: cmd.CommandKey, ObjectName: params.ObjectName, Key: cmd.CommandKey}
	return soap.RenderResponse(soap.AddObjectTmpl, data)
}

type DeleteObjectHandler struct{}

func (h *DeleteObjectHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	var params struct {
		ObjectName string `json:"object_name"`
	}
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse DeleteObject params: %w", err)
	}
	data := soap.DeleteObjectData{ID: cmd.CommandKey, ObjectName: params.ObjectName, Key: cmd.CommandKey}
	return soap.RenderResponse(soap.DeleteObjectTmpl, data)
}

type DownloadHandler struct{}

func (h *DownloadHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	var params soap.DownloadData
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse Download params: %w", err)
	}
	params.ID = cmd.CommandKey
	params.CommandKey = cmd.CommandKey
	return soap.RenderResponse(soap.DownloadTmpl, params)
}

type UploadHandler struct{}

func (h *UploadHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	var params soap.UploadData
	if err := json.Unmarshal(cmd.Params, &params); err != nil {
		return nil, fmt.Errorf("parse Upload params: %w", err)
	}
	params.ID = cmd.CommandKey
	params.CommandKey = cmd.CommandKey
	return soap.RenderResponse(soap.UploadTmpl, params)
}

type RebootHandler struct{}

func (h *RebootHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	data := soap.RebootData{ID: cmd.CommandKey, CommandKey: cmd.CommandKey}
	return soap.RenderResponse(soap.RebootTmpl, data)
}

type FactoryResetHandler struct{}

func (h *FactoryResetHandler) BuildRequest(cmd *cmdqueue.Command) ([]byte, error) {
	return soap.RenderResponse(soap.FactoryResetTmpl, struct{ ID string }{ID: cmd.CommandKey})
}
