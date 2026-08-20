package acs

import (
	"encoding/json"
	"fmt"
	"time"
)

// SessionState 表示 TR069/CWMP 会话的状态。
// 会话遵循状态机模式，追踪 ACS 与 CPE 之间 TR069 对话的生命周期。
//
// 状态机图示：
//
//	                          Inform
//	                            │
//	                            ▼
//	                  ┌─────────────────┐
//	                  │ INFORM_RECEIVED │←─────────────────┐
//	                  └────────┬────────┘                  │
//	                           │                           │
//	                    Empty POST                        │
//	                           │                           │
//	                           ▼                           │
//	                  ┌─────────────────┐                  │
//	   ┌─────────────│   PROCESSING    │←────────┐        │
//	   │             └────────┬────────┘         │        │
//	   │                      │                  │        │
//	   │            ┌─────────┴─────────┐        │        │
//	   │            │                   │        │        │
//	   │       有待发命令            无待发命令    │        │
//	   │            │                   │        │        │
//	   │            ▼                   ▼        │        │
//	   │   ┌─────────────────┐   ┌──────────┐   │        │
//	   │   │   RPC_PENDING   │   │ COMPLETE │   │        │
//	   │   └────────┬────────┘   └──────────┘   │        │
//	   │            │                          │        │
//	   │     CPE Response                      │        │
//	   │            │                          │        │
//	   │            ▼                          │        │
//	   │   ┌─────────────────┐                  │        │
//	   │   │   RPC_RESPONSE  │──────────────────┘        │
//	   │   └────────┬────────┘   有更多命令              │
//	   │            │                                    │
//	   │     ┌──────┴──────┐                             │
//	   │     │             │                             │
//	   │  有更多命令    无更多命令                         │
//	   │     │             │                             │
//	   │     │             ▼                             │
//	   │     │      ┌──────────┐                         │
//	   └─────┴─────→│ COMPLETE │                         │
//	                └──────────┘                         │
//	                     │                                │
//	                     │ 会话结束                        │
//	                     ▼                                │
//	                ┌──────────┐                          │
//	                │   IDLE   │──────────────────────────┘
//	                └──────────┘     新 Inform 到达
type SessionState string

const (
	// StateIdle 表示无活跃会话。这是 CPE 发送 Inform 之前的初始状态，
	// 也是会话完成后的最终状态。
	// 下一状态：INFORM_RECEIVED（新 Inform 到达时）
	StateIdle SessionState = "IDLE"

	// StateInformReceived 表示 ACS 已接收并解析了 CPE 发来的 Inform。
	// 此时创建会话、设置 Cookie、发布设备事件。
	// ACS 将发送 InformResponse 并等待 CPE 的空 POST。
	// 下一状态：PROCESSING（收到空 POST），COMPLETE（错误/会话关闭）
	StateInformReceived SessionState = "INFORM_RECEIVED"

	// StateProcessing 表示 CPE 在 InformResponse 后发送了空 POST。
	// ACS 检查命令队列中是否有待发送给 CPE 的命令。
	// 下一状态：RPC_PENDING（有命令），COMPLETE（无命令）
	StateProcessing SessionState = "PROCESSING"

	// StateRPCPending 表示 ACS 已向 CPE 发送 RPC 请求，正在等待响应。
	// LastRPC 字段被设置为待处理请求的方法名。
	// 下一状态：RPC_RESPONSE（CPE 响应时）
	StateRPCPending SessionState = "RPC_PENDING"

	// StateRPCResponse 表示 ACS 已收到 CPE 的 RPC 响应。
	// ACS 处理响应并检查队列中是否有更多命令。
	// 下一状态：PROCESSING（处理响应后检查命令），RPC_PENDING（链式命令），COMPLETE（完成）
	StateRPCResponse SessionState = "RPC_RESPONSE"

	// StateComplete 表示 TR069 会话已正常结束。
	// 资源被释放：准入槽位释放、指标记录、会话删除。
	// ACS 发送空的 HTTP 204 响应通知 CPE 会话已关闭。
	// 下一状态：IDLE（准备好接收新会话）
	StateComplete SessionState = "COMPLETE"
)

// Session 表示与 CPE 设备之间的活跃 TR069/CWMP 会话。
// 会话在 Inform 时创建，在完成或超时时删除。
// 存储在 Redis 中并设置 TTL 以实现自动清理。
type Session struct {
	// ID 是唯一的会话标识符（UUID v4 格式），用作 Cookie 值。
	// 在 Inform 时生成，通过 Set-Cookie 响应头返回给 CPE。
	// CPE 必须在后续请求中通过 Cookie 请求头携带此值。
	ID string `json:"id"`

	// DeviceSN 是设备序列号，从 Inform.DeviceId.SerialNumber 提取。
	// 用作设备查找和命令队列访问的主键。
	DeviceSN      string `json:"device_sn"`
	DeviceOUI     string `json:"device_oui,omitempty"`
	ProductClass  string `json:"product_class,omitempty"`
	RequestID     string `json:"request_id,omitempty"`
	Authenticated bool   `json:"authenticated"`

	// State 是状态机中的当前会话状态。
	State SessionState `json:"state"`

	// LastRPC 是最近发送给 CPE 的 RPC 方法名。
	// 在进入 RPC_PENDING 状态时设置。用于日志记录和调试。
	// 示例："GetParameterValues"、"SetParameterValues"、"Reboot"
	LastRPC string `json:"last_rpc"`

	// InstanceID 通常是 CPE 的 RemoteAddr（IP:Port），
	// 用于连接级别的追踪和日志记录。可能作为 connSessions 的备用方案。
	InstanceID string `json:"instance_id"`

	// StartedAt 是会话创建时间（收到 Inform 时）。
	StartedAt time.Time `json:"started_at"`

	// UpdatedAt 是会话状态最后修改时间。
	UpdatedAt time.Time `json:"updated_at"`

	// InformEvents 包含 Inform 消息中的事件码。
	// 示例："0 BOOTSTRAP"、"2 PERIODIC"、"4 VALUE CHANGE"
	InformEvents []string `json:"inform_events"`

	// CWMPId 是 Inform 的 SOAP-ENV:Header 中的 CWMP ID，
	// 用于关联 SOAP 对话中的请求和响应。
	CWMPId string `json:"cwmp_id"`

	// SessionTimeout 是 CPE 建议的会话超时时间（秒），
	// 通常来自 Device.ManagementServer.SessionTimeout 参数。
	// ACS 可使用此值设置 Redis 中的会话 TTL。
	SessionTimeout int `json:"session_timeout"`

	// RPCCount 记录当前会话中已完成的 RPC 交互次数（请求+响应算一次）。
	// 用于实施单会话 RPC 次数限制，避免触发基站单会话多次交互限制。
	RPCCount int `json:"rpc_count"`

	// LastCommandParams 保存最近发送给 CPE 的 RPC 命令参数（JSON）。
	// 用于在收到响应时关联原始请求上下文（如 GPN 的查询路径）。
	LastCommandParams json.RawMessage `json:"last_command_params,omitempty"`

	// LastTaskID / LastTaskCWMPID 保存最近一次从队列派发出去的 RPC 对应的
	// device_tasks.id 和 ACS 端生成的 cwmp_id。用于 handleSOAPFault 在收到
	// CPE 主动发起的 SOAP Fault（CPE 自己生成新 cwmp_id，无法用 ACS cwmp_id 反查 task）
	// 时按会话上下文 fallback 关联到正确的 task，避免厂商 Fault 信息无处归档。
	LastTaskID      string `json:"last_task_id,omitempty"`
	LastTaskCWMPID  string `json:"last_task_cwmp_id,omitempty"`
}

// validTransitions 定义允许的状态转换。
// 这确保了 TR069 会话协议流程，防止无效的状态变更。
var validTransitions = map[SessionState][]SessionState{
	// IDLE 只能转换到 INFORM_RECEIVED（新会话开始）
	StateIdle: {StateInformReceived},

	// INFORM_RECEIVED 可转换到：
	// - PROCESSING：CPE 发送了空 POST，正在检查命令
	// - COMPLETE：会话提前关闭（错误、超时或无响应）
	StateInformReceived: {StateProcessing, StateComplete},

	// PROCESSING 可转换到：
	// - RPC_PENDING：队列中找到命令，正在发送给 CPE
	// - COMPLETE：无命令，会话正常结束
	StateProcessing: {StateRPCPending, StateComplete},

	// RPC_PENDING 只能转换到 RPC_RESPONSE（CPE 响应）
	StateRPCPending: {StateRPCResponse},

	// RPC_RESPONSE 可转换到：
	// - PROCESSING：处理响应后检查更多命令
	// - RPC_PENDING：立即链式执行下一个命令
	// - COMPLETE：无更多命令，会话结束
	StateRPCResponse: {StateProcessing, StateRPCPending, StateComplete},

	// COMPLETE 转换回 IDLE（准备好接收新会话）
	StateComplete: {StateIdle},
}

// TransitionTo 尝试将会话转换到新状态。
func (s *Session) TransitionTo(newState SessionState) error {
	allowed, ok := validTransitions[s.State]
	if !ok {
		return fmt.Errorf("状态 %s 未定义转换规则", s.State)
	}

	for _, a := range allowed {
		if a == newState {
			s.State = newState
			s.UpdatedAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("无效的会话状态转换: %s → %s", s.State, newState)
}

// MarshalJSON 将会话序列化为 JSON，用于 Redis 存储。
func (s *Session) MarshalJSON() ([]byte, error) {
	type Alias Session
	return json.Marshal((*Alias)(s))
}

// UnmarshalJSON 从 JSON 反序列化会话。
func (s *Session) UnmarshalJSON(data []byte) error {
	type Alias Session
	return json.Unmarshal(data, (*Alias)(s))
}
