package license

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type LegacyFeatureDefinition struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	NameZH   string `json:"name_zh"`
	NameEN   string `json:"name_en"`
	ParentID string `json:"parent_id"`
	Path     string `json:"path"`
	PathEN   string `json:"path_en"`
}

type LegacyFeatureMapping struct {
	IDToCode   map[string]string
	CatalogIDs map[string]bool
	Features   map[string]LegacyFeatureDefinition
}

type LegacyFeatureDisplay struct {
	FeatureID   string   `json:"feature_id,omitempty"`
	FeatureCode string   `json:"feature_code,omitempty"`
	NameZH      string   `json:"name_zh"`
	NameEN      string   `json:"name_en"`
	Path        string   `json:"path,omitempty"`
	PathEN      string   `json:"path_en,omitempty"`
	Source      string   `json:"source"`
	Recognized  bool     `json:"recognized"`
	Licensed    bool     `json:"licensed"`
	RawIDs      []string `json:"raw_ids,omitempty"`
	RawCodes    []string `json:"raw_codes,omitempty"`
}

type legacyFeatureMappingFile struct {
	Version    int                                `json:"version"`
	IDToCode   map[string]string                  `json:"id_to_code"`
	CatalogIDs []string                           `json:"catalog_ids"`
	Features   map[string]LegacyFeatureDefinition `json:"features"`
}

// LoadLegacyFeatureMapping loads the repository-owned, generated mapping asset.
// The runtime never parses the legacy project's Java or SQL sources.
func LoadLegacyFeatureMapping(path string) (*LegacyFeatureMapping, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read license feature mapping: %w", err)
	}
	var source legacyFeatureMappingFile
	if err := json.Unmarshal(raw, &source); err != nil {
		return nil, fmt.Errorf("parse license feature mapping: %w", err)
	}
	if source.Version != 1 {
		return nil, fmt.Errorf("unsupported license feature mapping version %d", source.Version)
	}
	if len(source.IDToCode) == 0 || len(source.Features) == 0 {
		return nil, fmt.Errorf("license feature mapping is empty")
	}
	catalogIDs := make(map[string]bool, len(source.CatalogIDs))
	for _, id := range source.CatalogIDs {
		catalogIDs[id] = true
	}
	return &LegacyFeatureMapping{
		IDToCode:   source.IDToCode,
		CatalogIDs: catalogIDs,
		Features:   source.Features,
	}, nil
}

func (m *LegacyFeatureMapping) Normalize(ids, codes []string) []LegacyFeatureDisplay {
	if m == nil {
		return nil
	}
	items := make([]LegacyFeatureDisplay, 0, len(ids)+len(codes))
	seenCodes := make(map[string]bool)
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || seenCodes[code] {
			continue
		}
		seenCodes[code] = true
		items = append(items, m.display(code, "", "raw_code", ids, codes))
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		code := m.IDToCode[id]
		if code != "" && seenCodes[code] {
			continue
		}
		if code != "" {
			seenCodes[code] = true
		} else if m.CatalogIDs[id] {
			// Catalog nodes are grouping rows, not licensed leaf features.
			continue
		}
		items = append(items, m.display(code, id, "id_mapping", []string{id}, nil))
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i].NameZH
		if left == "" {
			left = items[i].FeatureCode + items[i].FeatureID
		}
		right := items[j].NameZH
		if right == "" {
			right = items[j].FeatureCode + items[j].FeatureID
		}
		return left < right
	})
	return items
}

func (m *LegacyFeatureMapping) display(code, id, source string, rawIDs, rawCodes []string) LegacyFeatureDisplay {
	item := LegacyFeatureDisplay{
		FeatureID:   id,
		FeatureCode: code,
		Source:      source,
		Licensed:    code != "",
		RawIDs:      uniqueSorted(rawIDs),
		RawCodes:    uniqueSorted(rawCodes),
	}
	definition, recognized := m.Features[code]
	item.NameZH = definition.NameZH
	item.NameEN = definition.NameEN
	item.Path = definition.Path
	item.PathEN = definition.PathEN
	item.Recognized = recognized || item.NameZH != "" || item.NameEN != ""
	if !item.Recognized {
		item.Source = "unknown"
		item.Licensed = false
	}
	return item
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

// codeModuleNames maps the first segment after "CODE_" to the top-level
// authorization key in the derived feature tree. Codes follow the shape
// CODE_<MODULE>[_<FEATURE>]. eNB/gNB keep mixed case to match PRD §4;
// acronyms stay uppercase; others are title-cased.
var codeModuleNames = map[string]string{
	"DASHBOARD":   "Dashboard",
	"ENB":         "eNB",
	"GNB":         "gNB",
	"CPE":         "CPE",
	"EPC":         "EPC",
	"EGW":         "EGW",
	"UPS":         "UPS",
	"ALARM":       "Alarm",
	"PERFORMANCE": "Performance",
	"SYSTEM":      "System",
	"TOPO":        "Topo",
	"TOOL":        "Tool",
	"HELP":        "Help",
	"ADVANCE":     "Advance",
	"OPERATOR":    "Operator",
}

// AuthorizationTree derives a PRD §4 nested authorization tree from the legacy
// feature codes/IDs a license grants. Each recognized CODE_<MODULE>[_<FEATURE>]
// becomes a leaf "All" under Module[.Feature]; unrecognized or non-CODE_
// values are skipped (they stay in legacy_feature_codes/features for display,
// and per D-05 unknown features must NOT be authorized). Catalog/grouping IDs
// are skipped to match Normalize semantics.
//
// The returned tree is what HasFeature traverses; it is persisted under the
// "authorization_tree" key of the feature_list envelope.
func (m *LegacyFeatureMapping) AuthorizationTree(ids, codes []string) map[string]any {
	tree := make(map[string]any)
	add := func(code string) {
		if _, recognized := m.Features[code]; !recognized {
			return
		}
		path := codeToPath(code)
		if len(path) == 0 {
			return
		}
		insertAuthorizationLeaf(tree, path)
	}

	seen := make(map[string]bool, len(codes)+len(ids))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		add(code)
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || m.CatalogIDs[id] {
			continue
		}
		code, ok := m.IDToCode[id]
		if !ok || seen[code] {
			continue
		}
		seen[code] = true
		add(code)
	}
	return tree
}

// codeToPath turns CODE_<MODULE>[_<FEATURE>] into a 1- or 2-level path.
// Returns nil for non-CODE_ values (no stable key can be derived).
func codeToPath(code string) []string {
	if !strings.HasPrefix(code, "CODE_") {
		return nil
	}
	rest := strings.TrimPrefix(code, "CODE_")
	if rest == "" {
		return nil
	}
	segments := strings.Split(rest, "_")
	module, ok := codeModuleNames[segments[0]]
	if !ok {
		module = titleFirst(segments[0])
	}
	path := []string{module}
	if len(segments) > 1 {
		var b strings.Builder
		for _, seg := range segments[1:] {
			b.WriteString(titleFirst(seg))
		}
		if b.Len() > 0 {
			path = append(path, b.String())
		}
	}
	return path
}

func titleFirst(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	return strings.ToUpper(lower[:1]) + lower[1:]
}

// legacySysList 复刻旧项目 FunctionCodeRelation.sysList()：旧格式（仅 ID）license
// 在 ID→code 后默认补全的系统基础 code，使系统类菜单即便 license 未显式列出也可见。
var legacySysList = []string{
	"CODE_SYSTEM_RESOURCES", "CODE_SYSTEM_SETTINGS", "CODE_SYSTEM_LOGS_OPERATION",
	"CODE_SYSTEM_LOGS_SECURITY", "CODE_SYSTEM_LOGS_SYSTEM", "CODE_SYSTEM_USERS",
	"CODE_SYSTEM_BACKUP_RESTORE", "CODE_SYSTEM_LICENSE", "CODE_HELP_GUIDE", "CODE_HELP_ABOUT",
}

// legacyGNBList 复刻旧项目 FunctionCodeRelation.gnbList()：新格式 license 的 code 集合
// 含 CODE_GNB 时，5G 基础能力整组开启。
var legacyGNBList = []string{
	"CODE_GNB_MONITOR", "CODE_GNB_SYNCHRONIZE", "CODE_GNB_SETTINGS", "CODE_GNB_ACTIVE",
	"CODE_GNB_RF_ENABLE", "CODE_GNB_TR069_MSG_EXCHANGE", "CODE_GNB_MML", "CODE_GNB_CHANGE_PASSWORD",
	"CODE_GNB_REBOOT", "CODE_GNB_LOGS", "CODE_GNB_SIGNALING_TRACE", "CODE_GNB_BACKUP_RESTORE",
	"CODE_GNB_UPGRADE_IMAGE", "CODE_GNB_ROLLBACK", "CODE_GNB_IPSEC_CERT", "CODE_GNB_DEVICE_REGISTER",
	"CODE_GNB_LICENSE", "CODE_GNB_DATA_MODEL",
}

// ExpandFeatureCodes 复刻旧项目 LicenseVerifyServiceImpl.checkLicenseInfo() 的 feature code
// 补全流程（FunctionCodeRelation），使真实 .lic 解析出的授权 code 集合与旧项目 supportFeatureCode
// 一致。规则：
//   - featureCodes 为空（旧格式仅 ID）：ID→code，补 sysList，再按特定 ID 补 RF_ENABLE/UPGRADE_FILE
//   - featureCodes 非空（新格式）：原样保留；若含 CODE_GNB 则补 gnbList
//   - 公共：Cloud 版剔 CODE_ADVANCE_ACCESS_CONTROL；有 SELFSTART 无 PNP 则补 PNP
//
// 注：旧 Java 的 delAlarmCode 结果被下一行覆盖（实际不生效），且其 alarmCode 常量不在 ID 映射中，
// 故此处不复刻（等价 no-op）。
func (m *LegacyFeatureMapping) ExpandFeatureCodes(featureIDs, featureCodes []string, isCloud bool) []string {
	seen := make(map[string]bool)
	add := func(c string) {
		if c = strings.TrimSpace(c); c != "" {
			seen[c] = true
		}
	}
	addAll := func(cs []string) {
		for _, c := range cs {
			add(c)
		}
	}

	if len(featureCodes) == 0 {
		idSet := make(map[string]bool)
		for _, id := range featureIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			idSet[id] = true
			if c, ok := m.IDToCode[id]; ok {
				add(c)
			}
		}
		addAll(legacySysList)
		if idSet["6"] {
			add("CODE_ENB_RF_ENABLE")
		}
		if idSet["11"] || idSet["14"] || idSet["72"] {
			add("CODE_ENB_UPGRADE_FILE")
		}
		if idSet["36"] {
			add("CODE_CPE_UPGRADE_FILE")
		}
		if idSet["1081"] {
			add("CODE_GNB_RF_ENABLE")
		}
		if idSet["10831"] || idSet["10833"] || idSet["10834"] {
			add("CODE_GNB_UPGRADE_FILE")
		}
	} else {
		addAll(featureCodes)
		if seen["CODE_GNB"] {
			addAll(legacyGNBList)
		}
	}

	if isCloud {
		delete(seen, "CODE_ADVANCE_ACCESS_CONTROL")
	}
	if seen["CODE_ADVANCE_SELFSTART"] && !seen["CODE_PLUG_AND_PLAY"] {
		add("CODE_PLUG_AND_PLAY")
	}

	result := make([]string, 0, len(seen))
	for c := range seen {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

// insertAuthorizationLeaf sets tree[path...] = "All", nesting one level when
// the path has a feature segment. A module-level "All" wins over per-feature
// leaves (the whole module is already authorized).
func insertAuthorizationLeaf(tree map[string]any, path []string) {
	if len(path) == 1 {
		tree[path[0]] = "All"
		return
	}
	if existing, ok := tree[path[0]]; ok && existing == "All" {
		return
	}
	sub, ok := tree[path[0]].(map[string]any)
	if !ok {
		sub = make(map[string]any)
		tree[path[0]] = sub
	}
	sub[path[1]] = "All"
}
