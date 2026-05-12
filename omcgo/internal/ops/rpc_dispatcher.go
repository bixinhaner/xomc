package ops

import (
	"fmt"
	"strings"
)

// RPCInlineKind is the discriminator for the inline-command envelope that
// T-0102-b stashes in OpsTask.Message when ExecuteRPC enqueues an ad-hoc
// RPC task. T-0102-c's dispatcher reads it back to know "this is a real
// RPC, not a template-driven task" and to extract action + params.
//
// Kept as a typed constant rather than scattered magic-string literals
// so the read side (TaskExecutor.Run) and the write side
// (ExtHandler.ExecuteRPC) can never drift.
const RPCInlineKind = "rpc"

// RPCInlineEnvelope is the JSON payload OpsTask.Message carries for
// inline RPC tasks. Stored as a string; T-0102-c json.Unmarshal's it on
// the dispatcher side. New fields can be added — the dispatcher skips
// any envelope whose Kind is not RPCInlineKind.
type RPCInlineEnvelope struct {
	Kind   string                 `json:"kind"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// actionToRPCMethod maps the operator-facing snake_case action name (as
// accepted by POST /ops/commands/rpc) to the TR-069 RPC method name the
// ACS dispatcher recognizes (CamelCase per CWMP spec).
//
// Supported actions (PRD §4.2.1):
//
//	reboot          → Reboot
//	factory_reset   → FactoryReset
//	get_param       → GetParameterValues
//	set_param       → SetParameterValues
//	get_rpc_methods → GetRPCMethods
//
// Unknown actions return a non-nil error so the caller surfaces a 4xx
// rather than enqueuing an undispatchable task.
func actionToRPCMethod(action string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "reboot":
		return "Reboot", nil
	case "factory_reset", "factoryreset":
		return "FactoryReset", nil
	case "get_param", "get_parameter_values", "getparametervalues":
		return "GetParameterValues", nil
	case "set_param", "set_parameter_values", "setparametervalues":
		return "SetParameterValues", nil
	case "get_rpc_methods", "getrpcmethods":
		return "GetRPCMethods", nil
	default:
		return "", fmt.Errorf("unsupported rpc action: %q", action)
	}
}
