package agentassistant

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/authz"
)

type PermissionChecker interface {
	CheckPermission(context.Context, uuid.UUID, string, string) (bool, error)
}
type CapabilityProvider interface {
	AssistantCapabilityCatalog() []map[string]any
	HandbookMetadata() (map[string]any, error)
}

// CurrentPrincipal resolves the stored owner against live account/role records.
// A privileged identity is never invented for an automatic event: only an
// actual built-in creator retains that creator's existing read permissions.
func (r *Repository) CurrentPrincipal(ctx context.Context, p Principal) (*admin.Claims, error) {
	uid, err := uuid.Parse(p.UserID)
	if err != nil {
		return nil, fmt.Errorf("ASSISTANT_OWNER_UNAVAILABLE")
	}
	rid, err := uuid.Parse(p.RoleID)
	if err != nil {
		return nil, fmt.Errorf("ASSISTANT_ROLE_UNAVAILABLE")
	}
	q, args, err := sql.Select("username", "source").From("users").Where(sq.Eq{"id": uid, "status": "active"}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build assistant owner lookup: %w", err)
	}
	var name, source string
	if err = r.pool.QueryRow(ctx, q, args...).Scan(&name, &source); err != nil {
		return nil, fmt.Errorf("ASSISTANT_OWNER_UNAVAILABLE: %w", err)
	}
	c := &admin.Claims{UserID: uid, Username: name, Roles: []string{}}
	if source == string(admin.UserSourceBuiltIn) {
		c.IsSuperAdmin = true
		return c, nil
	}
	q, args, err = sql.Select("r.name").From("user_roles ur").Join("roles r ON r.id=ur.role_id").Where(sq.Eq{"ur.user_id": uid, "ur.role_id": rid}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build assistant role lookup: %w", err)
	}
	var role string
	if err = r.pool.QueryRow(ctx, q, args...).Scan(&role); err != nil {
		return nil, fmt.Errorf("ASSISTANT_ROLE_UNAVAILABLE: %w", err)
	}
	c.CurrentRoleID = &rid
	c.Roles = []string{role}
	return c, nil
}
func principalFromClaims(c *admin.Claims) Principal {
	role := uuid.Nil
	if c.CurrentRoleID != nil {
		role = *c.CurrentRoleID
	}
	return Principal{UserID: c.UserID.String(), RoleID: role.String()}
}
func (s *Service) capabilities(ctx context.Context, p Principal) ([]Capability, error) {
	claims, err := s.repo.CurrentPrincipal(ctx, p)
	if err != nil {
		return nil, err
	}
	if s.provider == nil || s.permissions == nil {
		return nil, fmt.Errorf("ASSISTANT_CAPABILITIES_UNAVAILABLE")
	}
	out := []Capability{}
	for _, item := range s.provider.AssistantCapabilityCatalog() {
		id, _ := item["operationId"].(string)
		path, _ := item["path"].(string)
		title, _ := item["title"].(string)
		description, _ := item["description"].(string)
		deviceScoped, _ := item["deviceScoped"].(bool)
		if !claims.IsSuperAdmin {
			ok, e := s.permissions.CheckPermission(ctx, claims.UserID, path, "GET")
			if e != nil {
				return nil, fmt.Errorf("check assistant capability: %w", e)
			}
			if !ok {
				continue
			}
		}
		out = append(out, Capability{OperationID: id, Path: path, Title: title, Description: description, DeviceScoped: deviceScoped})
	}
	return out, nil
}
func (s *Service) checkScope(ctx context.Context, p Principal, scope Scope) error {
	c, err := s.repo.CurrentPrincipal(ctx, p)
	if err != nil {
		return err
	}
	if s.groups == nil || s.devices == nil {
		return fmt.Errorf("ASSISTANT_SCOPE_UNAVAILABLE")
	}
	groups, err := s.groups.GetUserVisibleGroupIDs(ctx, c.UserID, c.IsSuperAdmin)
	if err != nil {
		return fmt.Errorf("resolve assistant data permissions: %w", err)
	}
	if p.ScopeDigest != "" && p.ScopeDigest != scopeDigest(c, groups) {
		return fmt.Errorf("ASSISTANT_SCOPE_CHANGED")
	}
	if !c.IsSuperAdmin && len(groups) == 0 {
		return fmt.Errorf("ASSISTANT_SCOPE_FORBIDDEN")
	}
	if scope.Kind == "device" {
		id, err := uuid.Parse(scope.DeviceID)
		if err != nil {
			return fmt.Errorf("ASSISTANT_SCOPE_NOT_RESOLVED")
		}
		q, args, err := sql.Select("id").From("devices").Where(sq.Eq{"id": id, "deleted_at": nil}).ToSql()
		if err != nil {
			return fmt.Errorf("build assistant device lookup: %w", err)
		}
		var found uuid.UUID
		if err = s.repo.pool.QueryRow(ctx, q, args...).Scan(&found); err != nil {
			return fmt.Errorf("ASSISTANT_SCOPE_NOT_RESOLVED")
		}
		if err = authz.AuthorizeDeviceAccess(ctx, s.devices, id, groups); err != nil {
			return fmt.Errorf("ASSISTANT_SCOPE_FORBIDDEN: %w", err)
		}
	}
	return nil
}
func (s *Service) validate(ctx context.Context, p Principal, d *Definition) error {
	capabilities, err := s.capabilities(ctx, p)
	if err != nil {
		return err
	}
	if err = ValidateDefinition(d, capabilities); err != nil {
		return err
	}
	return s.checkScope(ctx, p, d.Scope)
}

// BindOperation rejects traversal, query-in-path and operation/path mismatches.
// Dynamic route parameters are still checked by the authoritative API handbook.
func BindOperation(template, requested string) (string, error) {
	parsed, err := url.ParseRequestURI(requested)
	if err != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host != "" || parsed.Path != requested || strings.Contains(requested, "\\") {
		return "", fmt.Errorf("ASSISTANT_OPERATION_PATH_MISMATCH")
	}
	want, got := strings.Split(template, "/"), strings.Split(requested, "/")
	if len(want) != len(got) {
		return "", fmt.Errorf("ASSISTANT_OPERATION_PATH_MISMATCH")
	}
	for i, part := range want {
		if strings.HasPrefix(part, ":") {
			if got[i] == "" || got[i] == "." || got[i] == ".." || strings.ContainsAny(got[i], "?#%") {
				return "", fmt.Errorf("ASSISTANT_OPERATION_PATH_MISMATCH")
			}
		} else if part != got[i] {
			return "", fmt.Errorf("ASSISTANT_OPERATION_PATH_MISMATCH")
		}
	}
	return requested, nil
}

// AuthorizeTool is the local trust boundary. Neither remote resourceScope nor a
// model-proposed owner is used. The run snapshot must already exist in xOMC.
func (s *Service) AuthorizeTool(ctx context.Context, runID, scenarioKey, definitionDigest, handbookDigest, operationID, requested string, query map[string]any) (*admin.Claims, string, map[string]any, error) {
	run, err := s.repo.GetRun(ctx, "", runID)
	if err != nil {
		return nil, "", nil, err
	}
	if run.Status != "QUEUED" && run.Status != "RUNNING" {
		return nil, "", nil, fmt.Errorf("ASSISTANT_RUN_NOT_ACTIVE")
	}
	if scenarioKey != "assistant:"+run.AssistantID || definitionDigest != run.Request.DefinitionDigest || handbookDigest != run.Request.HandbookDigest {
		return nil, "", nil, fmt.Errorf("ASSISTANT_EXECUTION_SNAPSHOT_MISMATCH")
	}
	metadata, err := s.provider.HandbookMetadata()
	if err != nil {
		return nil, "", nil, err
	}
	if metadata["handbookDigest"] != handbookDigest {
		return nil, "", nil, fmt.Errorf("ASSISTANT_CAPABILITIES_CHANGED")
	}
	if err = s.checkScope(ctx, run.Principal, run.Request.Definition.Scope); err != nil {
		return nil, "", nil, err
	}
	claims, err := s.repo.CurrentPrincipal(ctx, run.Principal)
	if err != nil {
		return nil, "", nil, err
	}
	claims.Scopes = []string{"agent-background:run:" + runID, "agent-background:assistant:" + run.AssistantID, "agent-background:operation:" + operationID}
	if operationID == "get.agent.handbook.manifest" {
		return claims, "/api/v1/agent/handbook/manifest", map[string]any{}, nil
	}
	if operationID == "get.agent.handbook.chunks.by_index" {
		path, e := BindOperation("/api/v1/agent/handbook/chunks/:index", requested)
		if e != nil {
			return nil, "", nil, e
		}
		index := strings.TrimPrefix(path, "/api/v1/agent/handbook/chunks/")
		if index == "" || strings.Trim(index, "0123456789") != "" {
			return nil, "", nil, fmt.Errorf("ASSISTANT_OPERATION_PATH_MISMATCH")
		}
		return claims, path, map[string]any{}, nil
	}
	if !slices.Contains(run.Request.Definition.Operations, operationID) {
		return nil, "", nil, fmt.Errorf("ASSISTANT_OPERATION_NOT_ALLOWED")
	}
	catalog, err := s.capabilities(ctx, run.Principal)
	if err != nil {
		return nil, "", nil, err
	}
	var cap *Capability
	for i := range catalog {
		if catalog[i].OperationID == operationID {
			cap = &catalog[i]
			break
		}
	}
	if cap == nil {
		return nil, "", nil, fmt.Errorf("ASSISTANT_OPERATION_NOT_ALLOWED")
	}
	path, err := BindOperation(cap.Path, requested)
	if err != nil {
		return nil, "", nil, err
	}
	bound := map[string]any{}
	for k, v := range query {
		bound[k] = v
	}
	scope := run.Request.Definition.Scope
	if scope.Kind == "device" {
		switch operationID {
		case "get.devices.by_id":
			path = "/api/v1/devices/" + scope.DeviceID
		case "get.alarms.active", "get.alarms.history":
			bound["device_id"] = scope.DeviceID
		default:
			return nil, "", nil, fmt.Errorf("ASSISTANT_CAPABILITY_SCOPE_MISMATCH")
		}
	}
	return claims, path, bound, nil
}

func scopeDigest(c *admin.Claims, groups []uuid.UUID) string {
	ids := make([]string, 0, len(groups))
	for _, g := range groups {
		ids = append(ids, g.String())
	}
	sort.Strings(ids)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s:%t:%s", c.UserID, c.IsSuperAdmin, strings.Join(ids, ",")))))
}
func (s *Service) bindPrincipal(ctx context.Context, p Principal) (Principal, error) {
	c, err := s.repo.CurrentPrincipal(ctx, p)
	if err != nil {
		return p, err
	}
	if s.groups == nil {
		return p, fmt.Errorf("ASSISTANT_SCOPE_UNAVAILABLE")
	}
	groups, err := s.groups.GetUserVisibleGroupIDs(ctx, c.UserID, c.IsSuperAdmin)
	if err != nil {
		return p, fmt.Errorf("bind assistant data grant: %w", err)
	}
	p.ScopeDigest = scopeDigest(c, groups)
	return p, nil
}
