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
