package agentruntime

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type apiParamDoc struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

type apiRequestDoc struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	OperationID string            `json:"operationId"`
	Query       map[string]string `json:"query,omitempty"`
	Body        any               `json:"body,omitempty"`
	Reason      string            `json:"reason"`
}

type apiRouteDoc struct {
	OperationID  string        `json:"operationId"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	Title        string        `json:"title"`
	Summary      string        `json:"summary"`
	Description  string        `json:"description"`
	Group        string        `json:"group"`
	Risk         string        `json:"risk"`
	Tags         []string      `json:"tags"`
	PathParams   []apiParamDoc `json:"pathParams,omitempty"`
	QueryParams  []apiParamDoc `json:"queryParams,omitempty"`
	RequestBody  any           `json:"requestBody,omitempty"`
	RequestUsage apiRequestDoc `json:"requestUsage"`
}

var apiResourceNames = map[string]string{
	"alarms":               "告警",
	"alarm-filters":        "告警过滤规则",
	"alarm-libraries":      "告警库",
	"api-endpoints":        "API 端点",
	"audit-logs":           "审计日志",
	"backup":               "备份",
	"cell":                 "小区",
	"config":               "配置",
	"dashboard":            "仪表盘",
	"device-groups":        "设备分组",
	"device-registrations": "设备预登记",
	"device-rules":         "设备规则",
	"devices":              "设备",
	"events":               "事件",
	"files":                "文件",
	"firmware":             "固件",
	"gnb":                  "gNB",
	"licenses":             "许可证",
	"logs":                 "日志",
	"menus":                "菜单",
	"mml":                  "MML",
	"mr":                   "测量报告",
	"northbound":           "北向接口",
	"notifications":        "通知",
	"ops":                  "运维任务",
	"permissions":          "权限",
	"pm":                   "性能",
	"provisioning":         "自动开站",
	"reports":              "报表",
	"roles":                "角色",
	"sites":                "站点",
	"sysConfig":            "系统配置",
	"sysDictionary":        "字典",
	"tasks":                "任务",
	"templates":            "模板",
	"topology":             "拓扑",
	"upgrade-tasks":        "升级任务",
	"users":                "用户",
}

var apiActionNames = map[string]string{
	"acknowledge":  "确认",
	"activate":     "激活",
	"apply":        "应用",
	"cancel":       "取消",
	"clear":        "清除",
	"clone":        "克隆",
	"copy":         "复制",
	"delete":       "删除",
	"describe":     "描述",
	"disable":      "禁用",
	"download":     "下载",
	"enable":       "启用",
	"execute":      "执行",
	"export":       "导出",
	"force-logout": "强制下线",
	"force-sync":   "强制同步",
	"health":       "健康检查",
	"history":      "历史记录",
	"login":        "登录",
	"logout":       "注销",
	"preview":      "预览",
	"reboot":       "重启",
	"refresh":      "刷新",
	"reset":        "重置",
	"restore":      "恢复",
	"retry":        "重试",
	"search":       "搜索",
	"stats":        "统计",
	"sync":         "同步",
	"tree":         "树形结构",
	"upload":       "上传",
	"validate":     "校验",
}

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func apiCatalogDoc(route gin.RouteInfo) apiRouteDoc {
	group := apiGroup(route.Path)
	resource := apiResourceNames[group]
	if resource == "" {
		resource = group
	}
	action := apiAction(route.Method, route.Path)
	title := fmt.Sprintf("%s - %s", resource, action)
	if resource == "" {
		title = strings.ToUpper(route.Method) + " " + route.Path
	}
	description := fmt.Sprintf("%s；通过 rest.request 调用 %s %s。", title, strings.ToUpper(route.Method), route.Path)
	pathParams := apiPathParams(route.Path)
	requestUsage := apiRequestDoc{
		Method:      strings.ToUpper(route.Method),
		Path:        route.Path,
		OperationID: operationID(route.Method, route.Path),
		Reason:      "说明为什么需要调用该接口，以及本次调用只读取还是会修改数据",
	}
	queryParams := []apiParamDoc{{
		Name:        "*",
		In:          "query",
		Required:    false,
		Type:        "string | number | boolean | string[]",
		Description: "按 Web UI 同名接口的查询参数传入；分页通常使用 page/pageSize/current/size 等字段。",
	}}
	if !isReadMethod(route.Method) {
		requestUsage.Body = map[string]any{}
	} else {
		requestUsage.Query = map[string]string{}
	}
	var requestBody any
	if !isReadMethod(route.Method) {
		requestBody = map[string]string{
			"type":        "object",
			"description": "JSON body。字段名和取值与 Web UI 调用该接口时保持一致。",
		}
	}
	return apiRouteDoc{
		OperationID:  requestUsage.OperationID,
		Method:       requestUsage.Method,
		Path:         route.Path,
		Title:        title,
		Summary:      title,
		Description:  description,
		Group:        group,
		Risk:         riskForMethod(route.Method),
		Tags:         apiTags(route.Path, group),
		PathParams:   pathParams,
		QueryParams:  queryParams,
		RequestBody:  requestBody,
		RequestUsage: requestUsage,
	}
}

func apiGroup(requestPath string) string {
	cleaned := strings.Trim(cleanAPIPath(requestPath), "/")
	parts := strings.Split(cleaned, "/")
	i := 0
	for i < len(parts) && (parts[i] == "" || parts[i] == "api" || strings.HasPrefix(parts[i], "v")) {
		i++
	}
	if i >= len(parts) {
		return ""
	}
	if parts[i] == "admin" && i+1 < len(parts) && parts[i+1] != "" {
		return parts[i+1]
	}
	return parts[i]
}

func apiAction(method, requestPath string) string {
	parts := strings.Split(strings.Trim(cleanAPIPath(requestPath), "/"), "/")
	last := ""
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] == "" || strings.HasPrefix(parts[i], ":") {
			continue
		}
		last = parts[i]
		break
	}
	if translated := apiActionNames[last]; translated != "" {
		return translated
	}
	if strings.Contains(last, "-") || strings.ContainsAny(last, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return humanizeSegment(last)
	}
	switch strings.ToUpper(method) {
	case http.MethodGet:
		if strings.HasSuffix(cleanAPIPath(requestPath), "/:id") {
			return "详情"
		}
		return "查询"
	case http.MethodPost:
		return "创建或执行"
	case http.MethodPut, http.MethodPatch:
		return "更新"
	case http.MethodDelete:
		return "删除"
	default:
		return strings.ToUpper(method)
	}
}

func apiPathParams(requestPath string) []apiParamDoc {
	parts := strings.Split(strings.Trim(cleanAPIPath(requestPath), "/"), "/")
	params := make([]apiParamDoc, 0)
	for _, part := range parts {
		if !strings.HasPrefix(part, ":") || len(part) == 1 {
			continue
		}
		name := strings.TrimPrefix(part, ":")
		params = append(params, apiParamDoc{
			Name:        name,
			In:          "path",
			Required:    true,
			Type:        "string",
			Description: "替换 URL 路径中的 :" + name,
		})
	}
	return params
}

func apiTags(requestPath, group string) []string {
	tags := []string{}
	if group != "" {
		tags = append(tags, group)
		if name := apiResourceNames[group]; name != "" {
			tags = append(tags, name)
		}
	}
	for _, part := range strings.Split(strings.Trim(cleanAPIPath(requestPath), "/"), "/") {
		if part == "" || part == "api" || strings.HasPrefix(part, "v") || strings.HasPrefix(part, ":") {
			continue
		}
		if !containsString(tags, part) {
			tags = append(tags, part)
		}
	}
	return tags
}

func humanizeSegment(value string) string {
	value = strings.ReplaceAll(value, "-", " ")
	value = camelBoundary.ReplaceAllString(value, "$1 $2")
	value = strings.TrimSpace(value)
	if value == "" {
		return "操作"
	}
	return value
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
