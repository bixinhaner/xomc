package datamodel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// MultiInstanceObject describes a multi-instance object node in the parameter model.
type MultiInstanceObject struct {
	TemplatePath string // "Device.Services.FAPService.{12}."
	BasePath     string // "Device.Services.FAPService."
	MaxInstances int
	MinInstances int
	Writable     bool // access == READ_WRITE
	Depth        int  // Nesting depth (0=top-level, 1=second-level, ...)
	ParentIdx    int  // Index in multiInst of parent multi-instance object, -1 if none
}

// ParameterTreeIterator analyzes a parameter model and generates device parameter fetch plans.
type ParameterTreeIterator struct {
	objects    []ObjectInfo
	parameters []Parameter
	multiInst  []MultiInstanceObject
}

// SyncPlan describes a two-phase plan for fetching all device parameters.
type SyncPlan struct {
	Phase1GPNs     []GPNRequest // GPN requests to discover multi-instance objects
	StaticPrefixes []string     // Non-multi-instance object prefixes for direct GPV
}

// GPNRequest describes a GetParameterNames request.
type GPNRequest struct {
	TemplatePath string // Template path (may contain {N})
	BasePath     string // Actual GPN request path
	Depth        int    // Nesting depth
	NextLevel    bool   // Always true for instance discovery
}

// InstanceMap holds GPN-discovered actual instances.
// key: object base path (e.g. "Device.Services.FAPService.")
// value: actual instance numbers (e.g. [1, 2, 3])
type InstanceMap map[string][]int

// NewParameterTreeIterator builds an iterator from a DataModel.
func NewParameterTreeIterator(dm *DataModel) (*ParameterTreeIterator, error) {
	it := &ParameterTreeIterator{}

	if dm.ObjectTree != nil {
		if err := json.Unmarshal(dm.ObjectTree, &it.objects); err != nil {
			return nil, fmt.Errorf("unmarshal object_tree: %w", err)
		}
	}
	if dm.ParameterTree != nil {
		if err := json.Unmarshal(dm.ParameterTree, &it.parameters); err != nil {
			return nil, fmt.Errorf("unmarshal parameter_tree: %w", err)
		}
	}

	// Apply min instances rules.
	it.objects = ApplyMinInstances(it.objects)

	// Analyze multi-instance objects.
	it.analyzeMultiInstances()

	return it, nil
}

// analyzeMultiInstances identifies multi-instance objects and their nesting relationships.
func (it *ParameterTreeIterator) analyzeMultiInstances() {
	var multiObjs []MultiInstanceObject

	for _, obj := range it.objects {
		if !ContainsPlaceholder(obj.Name) {
			continue
		}
		mo := MultiInstanceObject{
			TemplatePath: obj.Name,
			BasePath:     TemplateToBasePath(obj.Name),
			MaxInstances: obj.MaxInstances,
			MinInstances: obj.MinInstances,
			Writable:     obj.Access == "READ_WRITE",
			Depth:        CountPlaceholders(obj.Name) - 1, // depth is 0-based
			ParentIdx:    -1,
		}
		multiObjs = append(multiObjs, mo)
	}

	// Sort by depth (ascending), then by path.
	sort.Slice(multiObjs, func(i, j int) bool {
		if multiObjs[i].Depth != multiObjs[j].Depth {
			return multiObjs[i].Depth < multiObjs[j].Depth
		}
		return multiObjs[i].TemplatePath < multiObjs[j].TemplatePath
	})

	// Establish parent-child relationships.
	for i := range multiObjs {
		if multiObjs[i].Depth == 0 {
			continue
		}
		// Find parent: the multi-instance object whose template is a prefix
		// of this object's template and has depth = this.depth - 1.
		for j := range multiObjs {
			if multiObjs[j].Depth == multiObjs[i].Depth-1 &&
				strings.HasPrefix(multiObjs[i].TemplatePath, stripTrailingDot(multiObjs[j].TemplatePath)) {
				multiObjs[i].ParentIdx = j
				break
			}
		}
	}

	it.multiInst = multiObjs
}

// BuildSyncPlan generates the two-phase sync plan.
func (it *ParameterTreeIterator) BuildSyncPlan() *SyncPlan {
	plan := &SyncPlan{}

	// Build GPN requests for multi-instance objects.
	for _, mo := range it.multiInst {
		gpn := GPNRequest{
			TemplatePath: mo.TemplatePath,
			BasePath:     mo.BasePath,
			Depth:        mo.Depth,
			NextLevel:    true,
		}
		plan.Phase1GPNs = append(plan.Phase1GPNs, gpn)
	}

	// Identify static (non-multi-instance) top-level object prefixes.
	multiBasePaths := make(map[string]bool)
	for _, mo := range it.multiInst {
		multiBasePaths[mo.BasePath] = true
	}

	for _, obj := range it.objects {
		if ContainsPlaceholder(obj.Name) {
			continue
		}
		// Skip if this object is a child of a multi-instance object.
		isChildOfMulti := false
		for basePath := range multiBasePaths {
			if strings.HasPrefix(obj.Name, basePath) && obj.Name != basePath {
				isChildOfMulti = true
				break
			}
		}
		if isChildOfMulti {
			continue
		}
		// Only include objects that are direct children of root-level objects
		// (second-level prefixes for efficient GPV).
		dotCount := strings.Count(obj.Name, ".")
		if dotCount >= 2 && dotCount <= 3 {
			plan.StaticPrefixes = append(plan.StaticPrefixes, obj.Name)
		}
	}

	// If no static prefixes found but we have parameters, use "Device." as fallback.
	if len(plan.StaticPrefixes) == 0 && len(it.parameters) > 0 && len(it.multiInst) == 0 {
		plan.StaticPrefixes = []string{"Device."}
	}

	return plan
}

// ExpandGPNsForDepth returns GPN requests for the given depth, with placeholders
// replaced using discovered instance numbers from previous depths.
func (it *ParameterTreeIterator) ExpandGPNsForDepth(plan *SyncPlan, depth int, instances InstanceMap) []GPNRequest {
	var result []GPNRequest

	for _, gpn := range plan.Phase1GPNs {
		if gpn.Depth != depth {
			continue
		}

		if depth == 0 {
			// Top-level: no placeholder replacement needed.
			result = append(result, gpn)
			continue
		}

		// Need to replace placeholders from parent instances.
		// Find the parent multi-instance object for this GPN.
		var parentMO *MultiInstanceObject
		for i, mo := range it.multiInst {
			if mo.TemplatePath == gpn.TemplatePath {
				if mo.ParentIdx >= 0 {
					parentMO = &it.multiInst[mo.ParentIdx]
				}
				_ = i
				break
			}
		}

		if parentMO == nil {
			continue
		}

		// For each discovered parent instance, create an expanded GPN.
		expandedPaths := it.expandTemplatePath(gpn.TemplatePath, depth-1, instances)
		for _, expandedPath := range expandedPaths {
			basePath := TemplateToBasePath(expandedPath)
			result = append(result, GPNRequest{
				TemplatePath: expandedPath,
				BasePath:     basePath,
				Depth:        depth,
				NextLevel:    true,
			})
		}
	}

	return result
}

// expandTemplatePath replaces placeholders up to targetDepth with actual instances.
func (it *ParameterTreeIterator) expandTemplatePath(templatePath string, targetDepth int, instances InstanceMap) []string {
	if targetDepth < 0 {
		return []string{templatePath}
	}

	// Find the multi-instance object at depth 0 that is a prefix of this template.
	var paths []string
	paths = append(paths, templatePath)

	for d := 0; d <= targetDepth; d++ {
		var newPaths []string
		for _, p := range paths {
			// Find the base path of the first remaining placeholder.
			basePath := TemplateToBasePath(p)
			insts, ok := instances[basePath]
			if !ok || len(insts) == 0 {
				continue
			}
			for _, inst := range insts {
				expanded := ReplaceInstance(p, 0, inst)
				newPaths = append(newPaths, expanded)
			}
		}
		if len(newPaths) > 0 {
			paths = newPaths
		}
	}

	return paths
}

// BuildGPVPrefixes generates GPV partial path prefixes after all GPN phases are complete.
func (it *ParameterTreeIterator) BuildGPVPrefixes(plan *SyncPlan, instances InstanceMap) []string {
	var prefixes []string

	// Add static prefixes.
	prefixes = append(prefixes, plan.StaticPrefixes...)

	// Collect base paths that correspond to top-level (depth 0) multi-instance objects.
	topLevelBasePaths := make(map[string]bool)
	for _, mo := range it.multiInst {
		if mo.Depth == 0 {
			topLevelBasePaths[mo.BasePath] = true
		}
	}

	// Add discovered instance prefixes for top-level multi-instance objects.
	for basePath, insts := range instances {
		if !topLevelBasePaths[basePath] {
			continue
		}
		for _, inst := range insts {
			prefix := fmt.Sprintf("%s%d.", basePath, inst)
			prefixes = append(prefixes, prefix)
		}
	}

	sort.Strings(prefixes)
	return prefixes
}

// GetMultiInstanceObjects returns the analyzed multi-instance objects.
func (it *ParameterTreeIterator) GetMultiInstanceObjects() []MultiInstanceObject {
	return it.multiInst
}

// MaxDepth returns the maximum nesting depth of multi-instance objects.
func (it *ParameterTreeIterator) MaxDepth() int {
	maxD := -1
	for _, mo := range it.multiInst {
		if mo.Depth > maxD {
			maxD = mo.Depth
		}
	}
	return maxD
}

// stripTrailingDot removes the trailing dot from a path.
func stripTrailingDot(path string) string {
	return strings.TrimSuffix(path, ".")
}
