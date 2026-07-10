package agentruntime

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

const handbookSchemaVersion = "1.0"

type HandbookRoute struct {
	OperationID string        `json:"operationId"`
	Method      string        `json:"method"`
	Path        string        `json:"path"`
	Handler     string        `json:"handler"`
	Title       string        `json:"title"`
	Summary     string        `json:"summary"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Risk        string        `json:"risk"`
	Tags        []string      `json:"tags"`
	PathParams  []apiParamDoc `json:"pathParams,omitempty"`
}

type HandbookRouteExport struct {
	SchemaVersion  string          `json:"schemaVersion"`
	CatalogVersion string          `json:"catalogVersion"`
	TotalRoutes    int             `json:"totalRoutes"`
	Routes         []HandbookRoute `json:"routes"`
}

func (e *ToolExecutor) HandbookRouteExport() HandbookRouteExport {
	routes := e.allAPIRoutes()
	items := make([]HandbookRoute, 0, len(routes))
	for _, route := range routes {
		doc := apiCatalogDoc(route)
		items = append(items, HandbookRoute{
			OperationID: doc.OperationID,
			Method:      doc.Method,
			Path:        doc.Path,
			Handler:     route.Handler,
			Title:       doc.Title,
			Summary:     doc.Summary,
			Description: doc.Description,
			Category:    doc.Group,
			Risk:        doc.Risk,
			Tags:        doc.Tags,
			PathParams:  doc.PathParams,
		})
	}
	return HandbookRouteExport{
		SchemaVersion:  handbookSchemaVersion,
		CatalogVersion: apiCatalogVersion(routes),
		TotalRoutes:    len(items),
		Routes:         items,
	}
}

func (e *ToolExecutor) allAPIRoutes() gin.RoutesInfo {
	if e == nil || e.routes == nil {
		return gin.RoutesInfo{}
	}
	routes := e.routes.Routes()
	filtered := make(gin.RoutesInfo, 0, len(routes))
	for _, route := range routes {
		if strings.HasPrefix(route.Path, "/api/v1/") {
			filtered = append(filtered, route)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Path == filtered[j].Path {
			return filtered[i].Method < filtered[j].Method
		}
		return filtered[i].Path < filtered[j].Path
	})
	return filtered
}
