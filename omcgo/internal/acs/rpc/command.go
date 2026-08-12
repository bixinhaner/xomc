package rpc

import (
	"encoding/json"
	"time"
)

// Command is the minimal input the RPC dispatcher needs to render a SOAP
// request for a device. It is intentionally a subset of task.Task so the
// dispatcher has no dependency on task lifecycle fields (Status, SentAt,
// Result, etc): it takes a snapshot of the "what to send" data.
//
// ACS handlers Pop a *task.Task from task.TaskService, copy the minimal
// fields here, and pass it to Dispatcher.BuildRequest. The dispatcher
// then sets CWMPID and renders the corresponding SOAP envelope.
type Command struct {
	ID         string          // task ID (for logging only)
	DeviceSN   string          // device serial number, used by handlers that need runtime device state
	Method     string          // TR-069 RPC method name
	Params     json.RawMessage // method-specific params
	Priority   int             // lower = higher priority (unused by handlers)
	CreatedAt  time.Time       // for logging only
	ExpiresAt  *time.Time      // optional expiry (unused by handlers)
	CommandKey string          // TR-069 CommandKey
	CWMPID     string          // SOAP Header cwmp:ID — set by Dispatcher
}
