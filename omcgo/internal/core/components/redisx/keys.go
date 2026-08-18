package redisx

import "fmt"

// Keys 是全项目 Redis 键的唯一构造来源。任何形如 "acs:foo:bar" 的字符串字面量
// 禁止出现在本文件之外；业务侧统一通过 `redisx.Keys.XXX(...)` 获取，便于：
//   - 搜索/审计（一个文件就能看清所有 key 命名空间）
//   - 迁移（改前缀只需改本文件）
//   - 防手写错别字
//
// 命名约定：
//   - ACS 会话/任务相关         → acs:*
//   - 数据模型缓存             → datamodel:*
//   - 告警相关                 → alarm:*
//   - 设备相关                 → device:*
//   - 自动开站 sync plan       → provision:*
//   - 异常重启滑动窗口         → reboot:*
//   - 认证 / 登录 / 验证码     → auth:*
//   - 权限 / 可见设备组        → perm:*
//   - Casbin 策略热更（历史） → casbin:*
//   - SSE 推送离线缓存         → sse:*
//   - 系统运行期开关           → system:*
//   - 文件上传会话             → upload:*
var Keys KeyBuilder

// KeyBuilder 是 Redis 键构造器。所有方法返回具体的 key；以 Pattern 结尾的返回
// SCAN/KEYS 用的通配符；以 Prefix 结尾的返回不带终止符的前缀（用于 TrimPrefix）。
type KeyBuilder struct{}

// ===== ACS: TR-069 会话 / 心跳 / 命令唤醒 / STUN =====

// ACSSession 根据 session_id 定位一条 TR069 会话 Hash。
func (KeyBuilder) ACSSession(sessionID string) string { return acsSessionPrefix + sessionID }

// ACSSessionPrefix 返回会话键前缀，供扫描/迁移代码使用。
func (KeyBuilder) ACSSessionPrefix() string { return acsSessionPrefix }

// ACSHeartbeat 设备最近一次 Inform 的心跳时间戳。
func (KeyBuilder) ACSHeartbeat(deviceSN string) string { return acsHeartbeatPrefix + deviceSN }

// ACSHeartbeatPrefix 返回心跳键前缀。
func (KeyBuilder) ACSHeartbeatPrefix() string { return acsHeartbeatPrefix }

// ACSOnlineSet 在线设备索引（Sorted Set，member=SN，score=最近 Inform unix 秒）。
// issue #397：ACS 收 Inform 当场（发 NATS 之前）同步 ZADD，作为「免 NATS」的存活信号；
// ZCOUNT 算在线总数 O(log N)、ZREMRANGEBYSCORE 周期清理。单一全局键，非 per-device。
func (KeyBuilder) ACSOnlineSet() string { return acsOnlineSetKey }

// ACSConnReqPending Connection Request 去重键（30s TTL）。
func (KeyBuilder) ACSConnReqPending(deviceSN string) string {
	return acsConnReqPendingPrefix + deviceSN
}

// ACSContinuousWake 连续唤醒计数器（反抖动 / 抑制重复 CR）。
func (KeyBuilder) ACSContinuousWake(deviceSN string) string {
	return acsContinuousWakePrefix + deviceSN
}

// ACSUECountProbe Periodic Inform 触发的 UE Count 查询原子租约。
func (KeyBuilder) ACSUECountProbe(deviceSN string) string {
	return acsUECountProbePrefix + deviceSN
}

// PMUploadSetupAdmission 设备上线自动下发 PM 配置的跨实例短租约。
func (KeyBuilder) PMUploadSetupAdmission(deviceSN string) string {
	return pmUploadSetupAdmissionPrefix + deviceSN
}

// ACSSTUN 按设备 SN 维护 UDP Connection Request 的 STUN 地址。
func (KeyBuilder) ACSSTUN(deviceSN string) string { return acsSTUNPrefix + deviceSN }

// ACSSTUNPrefix STUN 地址键前缀。
func (KeyBuilder) ACSSTUNPrefix() string { return acsSTUNPrefix }

// ACSAdmissionSlots 全局准入控制 Sorted Set（member=sessionID, score=过期 unix 秒）。
// 跨实例共享，使全局并发会话上限真正成为全局而非每实例 N×max。槽位带 TTL 过期分，
// 丢失的 Release 由 ZREMRANGEBYSCORE 自愈回收。issue #65（Option B）。
func (KeyBuilder) ACSAdmissionSlots() string { return acsAdmissionSlotsKey }

// ACSDeviceSession 设备当前活跃会话指针（值=sessionID，TTL）。跨实例孤儿会话
// 检测：新 Inform 落在任一实例都能读到设备上一个 sessionID 并跨实例清理。
func (KeyBuilder) ACSDeviceSession(deviceSN string) string { return acsDeviceSessionPrefix + deviceSN }

// ACSDeviceSessionPrefix 设备会话指针键前缀，供扫描/迁移使用。
func (KeyBuilder) ACSDeviceSessionPrefix() string { return acsDeviceSessionPrefix }

// ACSConnReqURL 设备 Inform 上报的 ConnectionRequestURL（HTTP 唤醒回退用，TTL）。
// 镜像 ACSSTUN：会话后续唤可在非 Inform 实例读到该 URL，避免跨实例缓存丢失。
func (KeyBuilder) ACSConnReqURL(deviceSN string) string { return acsConnReqURLPrefix + deviceSN }

// ACSConnReqURLPrefix ConnectionRequestURL 键前缀。
func (KeyBuilder) ACSConnReqURLPrefix() string { return acsConnReqURLPrefix }

// ACSAccessSnapshot 设备接入授权摘要（ACS 热路径读取，TTL）。
func (KeyBuilder) ACSAccessSnapshot(deviceSN string) string {
	return acsAccessSnapshotPrefix + deviceSN
}

// ACSAccessSnapshotPrefix 接入授权摘要键前缀。
func (KeyBuilder) ACSAccessSnapshotPrefix() string { return acsAccessSnapshotPrefix }

// ACSAuthNonce HTTP Digest 一次性 nonce（SETEX + GETDEL，TTL）。跨实例共享，
// 使 Challenge 与 Authenticate 可落在不同实例而不丢 nonce。
func (KeyBuilder) ACSAuthNonce(nonce string) string { return acsAuthNoncePrefix + nonce }

// ACSAuthNoncePrefix nonce 键前缀。
func (KeyBuilder) ACSAuthNoncePrefix() string { return acsAuthNoncePrefix }

// ===== ACS: 统一任务队列（taskq / task / cwmp2task） =====

// ACSTaskQueue 设备级任务队列 Sorted Set。
func (KeyBuilder) ACSTaskQueue(deviceSN string) string { return acsTaskQueuePrefix + deviceSN }

// ACSTaskQueuePrefix 队列键前缀。
func (KeyBuilder) ACSTaskQueuePrefix() string { return acsTaskQueuePrefix }

// ACSTaskQueuePattern 队列扫描通配符（供 KEYS/SCAN 使用）。
func (KeyBuilder) ACSTaskQueuePattern() string { return acsTaskQueuePrefix + "*" }

// ACSTaskTransitionPending 待完成 PG 同步或跨 slot 索引清理的任务转换索引。
func (KeyBuilder) ACSTaskTransitionPending() string { return acsTaskTransitionPendingKey }

// ACSCommandQueue returns the legacy per-device command queue sorted set.
// It is retained for observability during migration to the unified task queue.
func (KeyBuilder) ACSCommandQueue(deviceSN string) string { return acsCommandQueuePrefix + deviceSN }

// ACSCommandQueuePrefix returns the legacy per-device command queue prefix.
func (KeyBuilder) ACSCommandQueuePrefix() string { return acsCommandQueuePrefix }

// ACSCommandQueuePattern returns the legacy command queue scan pattern.
func (KeyBuilder) ACSCommandQueuePattern() string { return acsCommandQueuePrefix + "*" }

// ACSTaskDetail 任务详情 Hash（活跃态 4h；终态默认 15m、最低 10m）。
func (KeyBuilder) ACSTaskDetail(taskID string) string { return acsTaskDetailPrefix + taskID }

// ACSTaskDetailPrefix 任务详情键前缀。
func (KeyBuilder) ACSTaskDetailPrefix() string { return acsTaskDetailPrefix }

// ACSCWMP2Task CWMP ID → Task ID 反查（活跃态 4h，任务终态时立即删除）。
func (KeyBuilder) ACSCWMP2Task(hashed string) string { return acsCWMP2TaskPrefix + hashed }

// ACSCWMP2TaskPrefix CWMP 反查键前缀。
func (KeyBuilder) ACSCWMP2TaskPrefix() string { return acsCWMP2TaskPrefix }

// ===== 数据模型缓存（L2） =====

// DataModelCacheVersion 跨实例通知的缓存版本号。
func (KeyBuilder) DataModelCacheVersion() string { return datamodelCacheVersionKey }

// DataModelPattern 扫描所有 datamodel:* 键。
func (KeyBuilder) DataModelPattern() string { return datamodelPattern }

// DataModelProduct product 级（最精确）缓存键。
func (KeyBuilder) DataModelProduct(carrier, tech, oui, productClass string) string {
	return fmt.Sprintf("datamodel:product:%s:%s:%s:%s", carrier, tech, oui, productClass)
}

// DataModelOUI 厂商级缓存键。
func (KeyBuilder) DataModelOUI(carrier, tech, oui string) string {
	return fmt.Sprintf("datamodel:oui:%s:%s:%s", carrier, tech, oui)
}

// DataModelDefault 运营商默认级缓存键。
func (KeyBuilder) DataModelDefault(carrier, tech string) string {
	return fmt.Sprintf("datamodel:default:%s:%s", carrier, tech)
}

// DataModelUnknown 非法 scope 的 fallback（仅用于日志/审计）。
func (KeyBuilder) DataModelUnknown(scope, carrier, tech, oui, productClass string) string {
	return fmt.Sprintf("datamodel:unknown:%s:%s:%s:%s:%s", scope, carrier, tech, oui, productClass)
}

// DataModelResolve 三级回退解析结果缓存（1h TTL）。
func (KeyBuilder) DataModelResolve(carrier, tech, oui, productClass string) string {
	return fmt.Sprintf("datamodel:resolve:%s:%s:%s:%s", carrier, tech, oui, productClass)
}

// ===== Product / ProductRegistry（T-0098 P2-01）=====

// ProductByID 单 product 详情缓存（24h TTL，按 UUID 索引）。
func (KeyBuilder) ProductByID(id string) string { return productByIDPrefix + id }

// ProductByIDPrefix product 详情键前缀，供扫描使用。
func (KeyBuilder) ProductByIDPrefix() string { return productByIDPrefix }

// ProductPattern 扫描所有 product:* 键。
func (KeyBuilder) ProductPattern() string { return productPattern }

// ProductCacheVersion 跨实例 ProductRegistry 缓存版本号。
func (KeyBuilder) ProductCacheVersion() string { return productCacheVersionKey }

// ProductByProductClass ProductRegistry productClass→product 路由结果缓存
// （hit TTL 1h / orphan TTL 5min，按 productClass 字面量索引）。productClass
// 含 "/" 在 Redis 中合法，不做转义。
func (KeyBuilder) ProductByProductClass(productClass string) string {
	return productByProductClassPrefix + productClass
}

// ProductByProductClassPrefix productClass 路由结果键前缀，供扫描使用。
func (KeyBuilder) ProductByProductClassPrefix() string { return productByProductClassPrefix }

// ===== ParamModel / ParamRegistry（T-0098 P2-02）=====

// ParamModelDefault 默认映射缓存（24h TTL，按 paramModelId 索引）。值为 JSON 序列化的 []ParamMapping。
func (KeyBuilder) ParamModelDefault(paramModelID string) string {
	return paramModelDefaultPrefix + paramModelID
}

// ParamModelDefaultPrefix 默认映射键前缀。
func (KeyBuilder) ParamModelDefaultPrefix() string { return paramModelDefaultPrefix }

// ParamModelDiscovered 设备发现映射缓存（24h TTL，按 productId+swVersion 索引）。
func (KeyBuilder) ParamModelDiscovered(productID, swVersion string) string {
	return fmt.Sprintf("%s%s:%s", paramModelDiscoveredPrefix, productID, swVersion)
}

// ParamModelDiscoveredPrefix 发现映射键前缀。
func (KeyBuilder) ParamModelDiscoveredPrefix() string { return paramModelDiscoveredPrefix }

// ParamModelPattern 扫描所有 parammodel:* 键。
func (KeyBuilder) ParamModelPattern() string { return paramModelPattern }

// ParamModelCacheVersion 跨实例 ParamRegistry 缓存版本号。
func (KeyBuilder) ParamModelCacheVersion() string { return paramModelCacheVersionKey }

// ===== KPI Route（T-0164-P1 KPI 路由）=====

// KPIRouteByProduct KPIRoute 缓存（24h TTL，按 product_id 索引）。值为 JSON 序列化的 router.KPIRoute。
func (KeyBuilder) KPIRouteByProduct(productID string) string {
	return kpiRouteByProductPrefix + productID
}

// KPIRouteByProductPrefix KPIRoute 键前缀（供扫描/清理）。
func (KeyBuilder) KPIRouteByProductPrefix() string { return kpiRouteByProductPrefix }

// KPIRoutePattern 扫描所有 kpi-route:* 键。
func (KeyBuilder) KPIRoutePattern() string { return kpiRoutePattern }

// KPIRouteCacheVersion 跨实例 KPI Router 缓存版本号。
func (KeyBuilder) KPIRouteCacheVersion() string { return kpiRouteCacheVersionKey }

// ===== 告警 / 异常重启 =====

// AlarmActive 活跃告警 Hash（每设备一个）。
func (KeyBuilder) AlarmActive(deviceSN string) string {
	return fmt.Sprintf("alarm:active:%s", deviceSN)
}

// RebootAbnormal 异常重启滑动窗口 ZSET。
func (KeyBuilder) RebootAbnormal(deviceSN string) string {
	return fmt.Sprintf("reboot:abnormal:%s", deviceSN)
}

// ===== Device / Provision / Upload =====

// DeviceSN 按 SN 缓存 Device 对象（read-through）。
func (KeyBuilder) DeviceSN(deviceSN string) string { return deviceSNPrefix + deviceSN }

// ProvisionSyncPlan 两阶段 sync plan 状态（Redis STRING + TTL）。
func (KeyBuilder) ProvisionSyncPlan(deviceSN string) string {
	return provisionSyncPlanPrefix + deviceSN
}

// ProvisionSyncPlanPrefix sync plan 键前缀。
func (KeyBuilder) ProvisionSyncPlanPrefix() string { return provisionSyncPlanPrefix }

// UploadSession 上传会话（按 deviceSN + cwmpID 定位）。
func (KeyBuilder) UploadSession(deviceSN, cwmpID string) string {
	return fmt.Sprintf("upload:session:%s:%s", deviceSN, cwmpID)
}

// MMLScriptImportSession 返回 TXT 脚本导入校验会话键。tokenSHA256 必须是
// 原始随机令牌的 SHA-256 十六进制摘要，禁止将原始令牌写入 Redis 键空间。
func (KeyBuilder) MMLScriptImportSession(tokenSHA256 string) string {
	return fmt.Sprintf("%s{%s}:session", mmlScriptImportPrefix, tokenSHA256)
}

// MMLScriptImportConsumed 返回已消费导入令牌的短期 tombstone 键。它与会话键
// 使用相同摘要，避免最终确认成功后的网络重试被错误地视为普通过期。
func (KeyBuilder) MMLScriptImportConsumed(tokenSHA256 string) string {
	return fmt.Sprintf("%s{%s}:consumed", mmlScriptImportPrefix, tokenSHA256)
}

// MMLScriptImportClaim stores the short-lived claim owner separately from the
// immutable validation-session JSON. It uses the same hash tag as the session
// and consumed keys so Lua claim/release/finalize operations remain atomic in
// Redis Cluster.
func (KeyBuilder) MMLScriptImportClaim(tokenSHA256 string) string {
	return fmt.Sprintf("%s{%s}:claim", mmlScriptImportPrefix, tokenSHA256)
}

// ===== Admin: 认证 / 暴力破解 / 权限 / Casbin =====

// AuthCaptcha 验证码 Hash。
func (KeyBuilder) AuthCaptcha(captchaID string) string { return authCaptchaPrefix + captchaID }

// AuthCaptchaPrefix 验证码键前缀。
func (KeyBuilder) AuthCaptchaPrefix() string { return authCaptchaPrefix }

// AuthFailed 登录失败计数（暴力破解防护）。
func (KeyBuilder) AuthFailed(usernameOrIP string) string { return authFailedPrefix + usernameOrIP }

// AuthFailedPrefix 登录失败计数键前缀。
func (KeyBuilder) AuthFailedPrefix() string { return authFailedPrefix }

// PermVisibleGroups 缓存用户的可见设备分组列表。
func (KeyBuilder) PermVisibleGroups(userID string) string { return permVisibleGroupsPrefix + userID }

// PermVisibleGroupsPrefix 可见分组缓存键前缀。
func (KeyBuilder) PermVisibleGroupsPrefix() string { return permVisibleGroupsPrefix }

// CasbinPolicyChannel Casbin 策略热更新的 Pub/Sub 频道
// （历史兼容：生产广播已迁到 NATS sys.casbin.policy.reload）。
func (KeyBuilder) CasbinPolicyChannel() string { return casbinPolicyChannel }

// ===== SSE / System =====

// SSEPending 用户离线消息缓冲队列。
func (KeyBuilder) SSEPending(userID string) string {
	return fmt.Sprintf("sse:pending:%s", userID)
}

// ===== 外部接口限流（北向 / 网元直连）=====

// RateLimitFixedWindow 返回一个固定窗口限流计数器键。scope 标识限流维度
// （如 "nb:endpoint"、"ne:device"），key 为该维度内的具体标识（端点名 / 设备 SN），
// window 为窗口起始秒（unix / windowSeconds 后取整，由调用方传入）。
// 采用 INCR + EXPIRE 的固定窗口算法，跨实例共享（Redis 后端），适合北向/网元直连
// 这类低频外部接口的防滥用，无需 token bucket 的平滑性。
func (KeyBuilder) RateLimitFixedWindow(scope, key string, window int64) string {
	return fmt.Sprintf("%s%s:%s:%d", rateLimitFixedWindowPrefix, scope, key, window)
}

// RateLimitFixedWindowPrefix 固定窗口限流键前缀（供扫描 / 审计）。
func (KeyBuilder) RateLimitFixedWindowPrefix() string { return rateLimitFixedWindowPrefix }

// ---------------------------------------------------------------------------
// private constants（对外不暴露；所有公共 API 均从本文件拼装）
// ---------------------------------------------------------------------------

const (
	// ACS
	acsSessionPrefix            = "acs:session:id:"
	acsHeartbeatPrefix          = "acs:heartbeat:"
	acsOnlineSetKey             = "acs:online"
	acsConnReqPendingPrefix     = "acs:connreq:pending:"
	acsContinuousWakePrefix     = "acs:continuous_wake:"
	acsUECountProbePrefix       = "acs:ue_count:probe:"
	acsSTUNPrefix               = "acs:stun:"
	acsTaskQueuePrefix          = "acs:taskq:"
	acsCommandQueuePrefix       = "acs:cmdq:"
	acsTaskDetailPrefix         = "acs:task:"
	acsCWMP2TaskPrefix          = "acs:cwmp2task:"
	acsTaskTransitionPendingKey = "acs:task:transition:pending"
	// issue #65（Option B）— ACS 横扩去进程态：准入槽位 / 设备会话指针 / CR URL / nonce
	acsAdmissionSlotsKey    = "acs:admission:slots"
	acsDeviceSessionPrefix  = "acs:device:session:"
	acsConnReqURLPrefix     = "acs:connreq:url:"
	acsAccessSnapshotPrefix = "acs:access:snapshot:"
	acsAuthNoncePrefix      = "acs:auth:nonce:"

	// pm
	pmUploadSetupAdmissionPrefix = "pm:upload_setup:admission:"

	// datamodel
	datamodelCacheVersionKey = "datamodel:cache_version"
	datamodelPattern         = "datamodel:*"

	// product (T-0098 P2-01 ProductRegistry)
	productByIDPrefix           = "product:byID:"
	productByProductClassPrefix = "product:byProductClass:"
	productPattern              = "product:*"
	productCacheVersionKey      = "product:cache_version"

	// parammodel (T-0098 P2-02 ParamRegistry)
	paramModelDefaultPrefix    = "parammodel:default:"
	paramModelDiscoveredPrefix = "parammodel:discovered:"
	paramModelPattern          = "parammodel:*"
	paramModelCacheVersionKey  = "parammodel:cache_version"

	// kpi-route (T-0164-P1 KPI 路由)
	kpiRouteByProductPrefix = "kpi-route:product:"
	kpiRoutePattern         = "kpi-route:*"
	kpiRouteCacheVersionKey = "kpi-route:cache_version"

	// device / provision / upload
	deviceSNPrefix          = "device:sn:"
	provisionSyncPlanPrefix = "provision:sync_plan:"
	mmlScriptImportPrefix   = "omc:mml:script-import:"

	// admin
	authCaptchaPrefix       = "auth:captcha:"
	authFailedPrefix        = "auth:failed:"
	permVisibleGroupsPrefix = "perm:visible_groups:"
	casbinPolicyChannel     = "casbin:policy:reload"

	// 外部接口限流（北向 / 网元直连固定窗口计数器）
	rateLimitFixedWindowPrefix = "ratelimit:fw:"
)
