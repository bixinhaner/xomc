package admin

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
)

// ApiEndpointService provides CRUD and sync operations for API endpoints.
type ApiEndpointService struct {
	repo                  ApiEndpointRepository
	builtInPermReconciler BuiltInAPIPermissionReconciler
	logger                *zap.Logger
}

type BuiltInAPIPermissionReconciler interface {
	ReconcileBuiltInAPIPermissions(context.Context) (BuiltInAPIPermissionGrantResult, error)
}

type apiEndpointBatchUpserter interface {
	UpsertBatch(context.Context, []ApiEndpointUpsertInput) (created int, err error)
}

// NewApiEndpointService creates a new ApiEndpointService.
func NewApiEndpointService(repo ApiEndpointRepository, logger *zap.Logger) *ApiEndpointService {
	return &ApiEndpointService{
		repo:   repo,
		logger: logger.Named("api-endpoint-service"),
	}
}

// SetBuiltInPermissionReconciler wires the immutable built-in role baseline
// maintenance into both startup and manually triggered endpoint synchronization.
func (s *ApiEndpointService) SetBuiltInPermissionReconciler(reconciler BuiltInAPIPermissionReconciler) {
	s.builtInPermReconciler = reconciler
}

// ListApiEndpoints retrieves a paginated, filtered list of API endpoints.
func (s *ApiEndpointService) ListApiEndpoints(ctx context.Context, filter ApiEndpointFilter) (*model.ListResponse[ApiEndpointDB], error) {
	result, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list api endpoints: %w", err)
	}
	return result, nil
}

// CreateApiEndpoint inserts a new API endpoint.
func (s *ApiEndpointService) CreateApiEndpoint(ctx context.Context, req CreateApiEndpointRequest) (*ApiEndpointDB, error) {
	ep := &ApiEndpointDB{
		Path:        req.Path,
		Method:      strings.ToUpper(req.Method),
		Name:        req.Name,
		Description: req.Description,
		ApiGroup:    req.ApiGroup,
		IsAuto:      false,
	}
	if err := s.repo.Create(ctx, ep); err != nil {
		return nil, fmt.Errorf("create api endpoint: %w", err)
	}
	return ep, nil
}

// UpdateApiEndpoint modifies an existing API endpoint.
func (s *ApiEndpointService) UpdateApiEndpoint(ctx context.Context, id uuid.UUID, req UpdateApiEndpointRequest) (*ApiEndpointDB, error) {
	ep, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("update api endpoint: %w", err)
	}
	return ep, nil
}

// DeleteApiEndpoint removes a single API endpoint by ID.
func (s *ApiEndpointService) DeleteApiEndpoint(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete api endpoint: %w", err)
	}
	return nil
}

// DeleteApiEndpointsByIDs removes multiple API endpoints.
func (s *ApiEndpointService) DeleteApiEndpointsByIDs(ctx context.Context, ids []uuid.UUID) error {
	if err := s.repo.DeleteByIDs(ctx, ids); err != nil {
		return fmt.Errorf("batch delete api endpoints: %w", err)
	}
	return nil
}

// GetApiGroups returns the distinct list of api_group values.
func (s *ApiEndpointService) GetApiGroups(ctx context.Context) ([]string, error) {
	groups, err := s.repo.GetGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("get api groups: %w", err)
	}
	return groups, nil
}

// SyncApiEndpoints scans gin.RoutesInfo and upserts each route into the database.
// Routes with is_auto=false (manually created) are preserved.
func (s *ApiEndpointService) SyncApiEndpoints(ctx context.Context, routes gin.RoutesInfo) (SyncResult, error) {
	var result SyncResult
	result.Total = len(routes)
	inputs := make([]ApiEndpointUpsertInput, 0, len(routes))
	seen := make(map[string]struct{}, len(routes))

	for _, route := range routes {
		apiGroup := inferApiGroup(route.Path)
		name := inferRouteName(route.Method, route.Path)
		description := inferRouteDescription(route.Method, route.Path)
		key := route.Method + "\x00" + route.Path
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		inputs = append(inputs, ApiEndpointUpsertInput{
			Path: route.Path, Method: route.Method, Name: name, Description: description, ApiGroup: apiGroup,
		})
	}

	if batchUpserter, ok := s.repo.(apiEndpointBatchUpserter); ok {
		created, err := batchUpserter.UpsertBatch(ctx, inputs)
		if err != nil {
			return result, fmt.Errorf("batch upsert API endpoints: %w", err)
		}
		result.Created = created
		result.Updated = result.Total - created
	} else {
		for _, input := range inputs {
			created, err := s.repo.Upsert(ctx, input.Path, input.Method, input.Name, input.Description, input.ApiGroup)
			if err != nil {
				return result, fmt.Errorf("upsert API endpoint %s %s: %w", input.Method, input.Path, err)
			}
			if created {
				result.Created++
			} else {
				result.Updated++
			}
		}
	}

	if s.builtInPermReconciler != nil {
		grants, err := s.builtInPermReconciler.ReconcileBuiltInAPIPermissions(ctx)
		if err != nil {
			return result, fmt.Errorf("reconcile built-in API permissions: %w", err)
		}
		s.logger.Info("reconciled built-in API permission baseline",
			zap.Int64("admin_grants_added", grants.Admin),
			zap.Int64("operator_grants_added", grants.Operator),
			zap.Int64("viewer_grants_added", grants.Viewer),
		)
	}

	s.logger.Info("synced api endpoints",
		zap.Int("total", result.Total),
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated),
	)
	return result, nil
}

// inferApiGroup extracts an api_group from a URL path.
//
// 规则：跳过 /api/v{N} 前缀后取第一个有意义的段。/admin/<sub>/... 形态再下钻
// 一级，避免所有 admin 路径都被聚成同一个 "admin" 组（粒度过粗），更贴近
// 业务模块（users/roles/menus/sysConfig/api-endpoints 等）。
//
// Examples:
//
//	/api/v1/admin/users        -> users
//	/api/v1/admin/roles/:id    -> roles
//	/api/v1/admin/sysConfig    -> sysConfig
//	/api/v1/auth/login         -> auth
//	/api/v1/devices            -> devices
//	/api/v1/device-groups/tree -> device-groups
//	/api/v1/admin              -> admin   （兜底，无下钻段）
func inferApiGroup(path string) string {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	skip := map[string]bool{"api": true, "v1": true, "v2": true, "v3": true}
	idx := 0
	for idx < len(parts) && (parts[idx] == "" || skip[parts[idx]]) {
		idx++
	}
	if idx >= len(parts) {
		return ""
	}

	first := parts[idx]
	// /admin/<sub>/... 下钻一级；仅当下一段非空时生效，避免 "/admin" 退化成 ""。
	if first == "admin" && idx+1 < len(parts) && parts[idx+1] != "" {
		return parts[idx+1]
	}
	return first
}

// inferRouteName generates a human-readable name from method and path.
func inferRouteName(method, path string) string {
	meta := inferRouteMetadata(method, path)
	if meta.Name != "" {
		return meta.Name
	}
	return strings.ToUpper(method) + " " + path
}

func inferRouteDescription(method, path string) string {
	meta := inferRouteMetadata(method, path)
	return meta.Description
}

type routeMetadata struct {
	Name        string
	Description string
}

// APIRouteMetadata exposes the same deterministic route semantics used by the
// API permission UI to other internal consumers such as the Agent handbook.
type APIRouteMetadata struct {
	Name        string
	Description string
	Group       string
}

func DescribeAPIRoute(method, path string) APIRouteMetadata {
	metadata := inferRouteMetadata(method, path)
	return APIRouteMetadata{
		Name:        metadata.Name,
		Description: metadata.Description,
		Group:       inferApiGroup(path),
	}
}

type routeAction struct {
	Verb        string
	Description string
}

var routeResourceNames = map[string]string{
	"alarms":               "告警",
	"alarm-filters":        "告警过滤规则",
	"alarm-libraries":      "告警库",
	"api-endpoints":        "API 端点",
	"api-keys":             "API 密钥",
	"audit-logs":           "审计日志",
	"auth":                 "认证",
	"backup":               "备份",
	"cell":                 "小区",
	"column-configs":       "列配置",
	"config":               "配置",
	"dashboard":            "仪表盘",
	"datamodels":           "数据模型",
	"dead-letters":         "死信队列",
	"device-groups":        "设备分组",
	"device-registrations": "设备预登记",
	"device-rules":         "设备规则",
	"devices":              "设备",
	"events":               "事件",
	"files":                "文件",
	"firmware":             "固件",
	"gnb":                  "gNB",
	"groups":               "用户组",
	"healthz":              "健康检查",
	"interop":              "互操作",
	"licenses":             "许可证",
	"logs":                 "日志",
	"menus":                "菜单",
	"mml":                  "MML",
	"mr":                   "测量报告 MR",
	"northbound":           "北向接口",
	"notifications":        "通知",
	"ops":                  "运维任务",
	"oui":                  "OUI",
	"permissions":          "权限",
	"pm":                   "性能 PM",
	"provisioning":         "自动开站",
	"readyz":               "就绪检查",
	"reports":              "报表",
	"roles":                "角色",
	"sites":                "站点",
	"software":             "软件",
	"sysConfig":            "系统配置",
	"sysDictionary":        "字典",
	"sysDictionaryDetail":  "字典明细",
	"system":               "系统信息",
	"tasks":                "任务",
	"templates":            "模板",
	"topology":             "拓扑",
	"transfer":             "文件传输",
	"upgrade-sub-tasks":    "升级子任务",
	"upgrade-tasks":        "升级任务",
	"users":                "用户",
}

var routeActionNames = map[string]routeAction{
	"acknowledge":     {"确认", "确认指定告警，记录处理状态"},
	"activate":        {"激活", "激活指定资源，使其进入可用状态"},
	"add":             {"新增", "新增指定资源记录"},
	"advance":         {"推进灰度阶段", "推进升级灰度流程到下一阶段"},
	"aggregated":      {"聚合", "查询聚合后的统计数据"},
	"alarms":          {"告警", "查询或维护告警相关数据"},
	"all":             {"全量", "查询全量数据集合"},
	"apply":           {"应用", "将配置或规则应用到目标对象"},
	"auto-discovery":  {"自动发现", "触发或查询自动发现结果"},
	"baseline":        {"基线", "查询或维护基线配置"},
	"batch":           {"批量", "对多个对象执行批量操作"},
	"batch-reboot":    {"批量重启", "批量下发设备重启操作"},
	"calculate":       {"计算", "触发计算并返回计算结果"},
	"cancel":          {"取消", "取消未完成的任务或操作"},
	"captcha":         {"获取验证码", "获取登录或校验所需验证码"},
	"cells":           {"小区列表", "查询设备或基站下的小区列表"},
	"change-password": {"修改密码", "修改当前用户或指定用户密码"},
	"check-delete":    {"校验是否可删除", "删除前检查资源是否仍被引用"},
	"children":        {"子节点", "查询树形结构的下级节点"},
	"circuit":         {"熔断器状态", "查询服务熔断与可靠性状态"},
	"clear":           {"清除", "清除指定状态或记录"},
	"clone":           {"克隆", "复制现有配置生成新记录"},
	"commands":        {"命令", "查询或执行命令定义"},
	"config":          {"配置", "查询或维护配置数据"},
	"confirm":         {"确认", "确认指定业务操作"},
	"copy":            {"复制", "复制现有资源或配置"},
	"counters":        {"计数器", "查询性能计数器数据"},
	"dangerous-check": {"危险命令检查", "校验命令是否属于高风险操作"},
	"deactivate":      {"停用", "停用指定资源"},
	"definitions":     {"定义", "查询或维护定义数据"},
	"delete":          {"删除", "删除指定资源记录"},
	"detail":          {"详情", "查询指定资源的详细信息"},
	"details":         {"明细", "查询指定资源的明细数据"},
	"dictionary":      {"字典", "查询或维护字典数据"},
	"diff":            {"差异", "比较并返回差异结果"},
	"discover":        {"探测", "探测设备或参数能力"},
	"download":        {"下载", "下载文件或导出结果"},
	"enable":          {"启用", "启用指定资源"},
	"enums":           {"枚举字典", "查询页面筛选和表单所需枚举"},
	"execute":         {"执行", "执行指定命令、脚本或任务"},
	"export":          {"导出", "按条件导出业务数据"},
	"fields":          {"字段", "查询可展示或可导出的字段集合"},
	"force-logout":    {"强制下线", "强制指定用户会话失效"},
	"force-sync":      {"强制同步", "忽略普通限制重新同步数据"},
	"full":            {"全量同步", "触发全量数据同步"},
	"geo":             {"地理位置数据", "查询地图展示所需地理数据"},
	"gnb-list":        {"gNB 列表", "查询 gNB 基站列表"},
	"groups":          {"分组", "查询或维护分组数据"},
	"health":          {"健康检查", "查询服务健康状态"},
	"history":         {"历史记录", "查询历史数据或操作记录"},
	"i18n":            {"国际化文案", "查询多语言文案资源"},
	"import":          {"导入", "导入外部文件或业务数据"},
	"incremental":     {"增量同步", "触发增量数据同步"},
	"info":            {"信息", "查询基础信息"},
	"kpi":             {"KPI", "查询或维护 KPI 数据"},
	"lock":            {"锁定", "锁定指定资源或账号"},
	"login":           {"登录", "提交账号凭据并获取访问令牌"},
	"logout":          {"注销", "退出当前登录会话"},
	"lookup":          {"查询映射", "按条件查询映射关系"},
	"mappings":        {"映射", "查询或维护映射关系"},
	"me":              {"当前用户信息", "查询当前登录用户资料和权限"},
	"move-devices":    {"移动设备", "将设备移动到目标分组"},
	"next-priority":   {"下一可用优先级", "查询规则可使用的下一优先级"},
	"objects":         {"多实例对象", "查询或维护 TR-069 多实例对象"},
	"param-paths":     {"参数路径", "查询参数路径集合"},
	"param-sync":      {"参数同步", "触发或查询设备参数同步"},
	"param-versions":  {"参数版本", "查询或维护参数版本"},
	"parameters":      {"参数", "查询或维护设备参数"},
	"params":          {"参数", "查询或提交参数集合"},
	"permission":      {"权限", "查询或维护权限配置"},
	"permissions":     {"权限", "查询或维护权限配置"},
	"perms":           {"权限", "查询或维护权限配置"},
	"permanent":       {"彻底删除", "从回收站永久删除资源"},
	"preview":         {"预览", "预览文件、配置或导出结果"},
	"pull":            {"拉取", "从设备或外部系统拉取数据"},
	"purge":           {"清理", "清理过期或无效数据"},
	"push":            {"推送", "向设备或外部系统推送数据"},
	"read":            {"标记已读", "将通知或消息标记为已读"},
	"reboot":          {"重启", "下发设备重启操作"},
	"recommend":       {"设为推荐", "将指定配置标记为推荐项"},
	"records":         {"记录", "查询业务记录列表"},
	"refresh":         {"刷新令牌", "刷新访问令牌并延长会话"},
	"reset":           {"重置", "重置状态或配置"},
	"reset-password":  {"重置密码", "管理员重置指定用户密码"},
	"restore":         {"恢复", "恢复备份、回收站记录或任务状态"},
	"results":         {"结果", "查询任务执行结果"},
	"retry":           {"重试", "重试失败任务或消息"},
	"revoke":          {"撤销", "撤销已授权或已发布的内容"},
	"rollback":        {"回滚", "将任务或版本回退到上一状态"},
	"rotate":          {"轮转", "轮转密钥、证书或配置"},
	"run":             {"执行", "执行指定任务"},
	"runs":            {"执行历史", "查询脚本或命令执行历史"},
	"schema":          {"架构", "查询数据结构或参数架构"},
	"scripts":         {"脚本", "查询或维护脚本内容"},
	"search":          {"搜索", "按关键字搜索业务数据"},
	"segments":        {"段", "查询分段数据"},
	"sort":            {"排序", "调整列表或树节点顺序"},
	"start":           {"启动", "启动任务、流程或服务"},
	"statistics":      {"统计", "查询统计汇总数据"},
	"stats":           {"统计", "查询统计汇总数据"},
	"stream":          {"SSE 推送流", "建立实时事件推送连接"},
	"sub-objects":     {"子对象", "查询参数模型子对象"},
	"summary":         {"汇总", "查询汇总概览数据"},
	"switch-role":     {"切换角色", "切换当前登录用户的工作角色"},
	"sync":            {"同步", "从运行态或外部系统同步最新数据"},
	"sync-status":     {"同步状态", "查询同步任务当前状态"},
	"targets":         {"推送目标", "查询通知或任务推送目标"},
	"template":        {"模板", "查询或维护模板数据"},
	"test":            {"测试", "发送测试请求验证配置"},
	"thresholds":      {"阈值", "查询或维护阈值配置"},
	"timeout":         {"超时清理", "清理超时任务或会话"},
	"toggle":          {"切换启用状态", "切换资源启用或禁用状态"},
	"tree":            {"树形结构", "查询树形层级数据"},
	"unacknowledge":   {"取消确认", "撤销告警确认状态"},
	"unlock":          {"解锁", "解除资源或账号锁定"},
	"upload":          {"上传", "上传文件或业务数据"},
	"validate":        {"校验", "校验请求参数或配置内容"},
	"widgets":         {"组件", "查询仪表盘组件配置"},
}

var routeParentNames = map[string]string{
	"active":        "活跃",
	"api":           "API",
	"batch":         "批量",
	"canary":        "灰度",
	"configs":       "配置",
	"export":        "导出",
	"ftp":           "FTP",
	"history":       "历史",
	"indicatormg":   "指标管理",
	"kpimanage":     "KPI 管理",
	"mappings":      "映射",
	"objects":       "多实例对象",
	"parameters":    "参数",
	"permissions":   "权限",
	"policy":        "策略",
	"recycle":       "回收站",
	"restore":       "恢复",
	"schedules":     "调度",
	"sub-tasks":     "子任务",
	"upgrade-tasks": "升级任务",
}

var camelRouteVerbs = []struct {
	Prefix string
	Name   string
}{
	{"add", "新增"},
	{"cancel", "取消"},
	{"clear", "清除"},
	{"confirm", "确认"},
	{"create", "创建"},
	{"delete", "删除"},
	{"del", "删除"},
	{"disable", "禁用"},
	{"enable", "启用"},
	{"export", "导出"},
	{"find", "查询"},
	{"get", "查询"},
	{"import", "导入"},
	{"list", "列表"},
	{"modify", "修改"},
	{"query", "查询"},
	{"remove", "移除"},
	{"save", "保存"},
	{"search", "搜索"},
	{"set", "设置"},
	{"start", "启动"},
	{"stop", "停止"},
	{"update", "更新"},
}

var camelRouteObjects = map[string]string{
	"AllIndicator":        "全部指标",
	"BaseKpiCustName":     "基础 KPI 自定义名",
	"GnbIndicatorsName":   "gNB 指标名",
	"Indicator":           "指标",
	"IndicatorGroup":      "指标分组",
	"IndicatorGroupInfo":  "指标分组信息",
	"IndicatorGroupTree":  "指标分组树",
	"IndicatorInfo":       "指标信息",
	"IndicatorListByPage": "指标分页列表",
	"SysDictionary":       "字典",
	"SysDictionaryDetail": "字典明细",
}

var routeParamPattern = regexp.MustCompile(`^[:{].*`)

func inferRouteMetadata(method, path string) routeMetadata {
	method = strings.ToUpper(method)
	apiGroup := inferApiGroup(path)
	resource := routeResourceNames[apiGroup]
	if resource == "" {
		resource = apiGroup
	}
	if resource == "" {
		resource = "接口"
	}

	if apiGroup == "healthz" || apiGroup == "readyz" {
		return routeMetadata{Name: resource, Description: "检查服务实例是否处于可用状态"}
	}
	if apiGroup == "system" {
		return routeMetadata{Name: "系统运行信息", Description: "查询系统版本、运行状态或基础环境信息"}
	}

	parts := routePathParts(path)
	if len(parts) > 0 && parts[0] == "admin" {
		parts = parts[1:]
	}

	action, hasID := inferRouteAction(parts, apiGroup, method)
	if action.Verb == "" {
		action = defaultRouteAction(method, hasID, isParamSegment(lastPart(parts)))
	}

	name := resource
	if action.Verb != "" && action.Verb != resource && action.Verb != apiGroup {
		name = resource + " - " + action.Verb
	}
	description := action.Description
	if description == "" {
		description = describeRouteAction(method, resource, action.Verb, hasID)
	} else if !strings.Contains(description, resource) {
		description = resource + "：" + description
	}

	return routeMetadata{Name: name, Description: description}
}

func routePathParts(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	idx := 0
	for idx < len(parts) && (parts[idx] == "api" || parts[idx] == "v1" || parts[idx] == "v2" || parts[idx] == "v3" || parts[idx] == "") {
		idx++
	}
	return parts[idx:]
}

func inferRouteAction(parts []string, apiGroup, method string) (routeAction, bool) {
	last := lastPart(parts)
	hasID := false
	lastIDAt := -1
	for i, part := range parts {
		if isParamSegment(part) {
			hasID = true
			lastIDAt = i
		}
	}

	var candidates []string
	if lastIDAt >= 0 && lastIDAt+1 < len(parts) {
		candidates = append(candidates, parts[lastIDAt+1:]...)
	} else if !isParamSegment(last) {
		candidates = append(candidates, last)
	}
	if isParamSegment(last) && len(parts) >= 2 {
		candidates = append(candidates, parts[len(parts)-2])
	}

	for i := len(candidates) - 1; i >= 0; i-- {
		candidate := candidates[i]
		if candidate == "" || candidate == apiGroup || isParamSegment(candidate) {
			continue
		}
		action := routeActionFromSegment(candidate)
		if action.Verb == "" {
			continue
		}
		action.Verb = withRouteParent(parts, candidate, action.Verb)
		if isParamSegment(last) && method != "POST" {
			action.Verb = action.Verb + defaultObjectActionSuffix(method)
			action.Description = "查询或维护指定对象的" + action.Verb + "数据"
		}
		return action, hasID
	}

	return routeAction{}, hasID
}

func routeActionFromSegment(segment string) routeAction {
	if action, ok := routeActionNames[segment]; ok {
		return action
	}
	if strings.Contains(segment, "-") || strings.Contains(segment, "_") {
		words := strings.FieldsFunc(segment, func(r rune) bool { return r == '-' || r == '_' })
		translated := make([]string, 0, len(words))
		for _, word := range words {
			if action, ok := routeActionNames[word]; ok {
				translated = append(translated, action.Verb)
			} else if word != "" {
				translated = append(translated, word)
			}
		}
		if len(translated) > 0 {
			return routeAction{Verb: strings.Join(translated, ""), Description: "处理" + strings.Join(translated, "") + "相关数据"}
		}
	}
	if strings.IndexFunc(segment, unicode.IsUpper) >= 0 {
		if cn := camelRouteAction(segment); cn != "" {
			return routeAction{Verb: cn, Description: "处理" + cn + "相关数据"}
		}
	}
	return routeAction{}
}

func camelRouteAction(segment string) string {
	compoundVerbs := []struct {
		prefix string
		name   string
	}{
		{prefix: "addOrModify", name: "新增或修改"},
		{prefix: "getOrCreate", name: "查询或创建"},
	}
	for _, compoundVerb := range compoundVerbs {
		if strings.HasPrefix(segment, compoundVerb.prefix) {
			tail := strings.TrimPrefix(segment, compoundVerb.prefix)
			if tail == "" {
				return compoundVerb.name
			}
			if objectCN, ok := camelRouteObjects[tail]; ok {
				return compoundVerb.name + objectCN
			}
			return compoundVerb.name + splitCamelWords(tail)
		}
	}
	for _, verb := range camelRouteVerbs {
		if strings.HasPrefix(segment, verb.Prefix) {
			tail := strings.TrimPrefix(segment, verb.Prefix)
			if tail == "" {
				return verb.Name
			}
			if objectCN, ok := camelRouteObjects[tail]; ok {
				return verb.Name + objectCN
			}
			return verb.Name + splitCamelWords(tail)
		}
	}
	return ""
}

func splitCamelWords(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range value {
		if i > 0 && unicode.IsUpper(r) {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func withRouteParent(parts []string, segment, verb string) string {
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != segment || i == 0 {
			continue
		}
		parent := parts[i-1]
		if parentCN := routeParentNames[parent]; parentCN != "" && !strings.Contains(verb, parentCN) {
			return parentCN + verb
		}
	}
	return verb
}

func defaultRouteAction(method string, hasID, idLast bool) routeAction {
	if idLast {
		switch method {
		case "GET":
			return routeAction{Verb: "详情", Description: "查询指定资源的详细信息"}
		case "PUT":
			return routeAction{Verb: "更新", Description: "更新指定资源的完整配置"}
		case "PATCH":
			return routeAction{Verb: "部分更新", Description: "更新指定资源的部分字段"}
		case "DELETE":
			return routeAction{Verb: "删除", Description: "删除指定资源记录"}
		}
	}
	if !hasID {
		switch method {
		case "GET":
			return routeAction{Verb: "列表", Description: "按筛选条件查询资源列表"}
		case "POST":
			return routeAction{Verb: "创建", Description: "创建新的资源记录"}
		case "PUT":
			return routeAction{Verb: "批量更新", Description: "批量更新资源数据"}
		case "DELETE":
			return routeAction{Verb: "批量删除", Description: "批量删除资源记录"}
		}
	}
	return routeAction{Verb: method, Description: "执行 " + method + " 请求对应的业务操作"}
}

func describeRouteAction(method, resource, verb string, hasID bool) string {
	if verb == "" {
		verb = method
	}
	target := ""
	if hasID {
		target = "指定"
	}
	return fmt.Sprintf("%s%s：%s，用于页面操作、权限控制或系统集成调用", target, resource, verb)
}

func defaultObjectActionSuffix(method string) string {
	switch method {
	case "GET":
		return "详情"
	case "PUT":
		return "更新"
	case "PATCH":
		return "部分更新"
	case "DELETE":
		return "删除"
	default:
		return ""
	}
}

func lastPart(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func isParamSegment(segment string) bool {
	return routeParamPattern.MatchString(segment)
}
