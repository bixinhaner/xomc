package paramsync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/provision"
)

const (
	NATSMaxPayloadBytes       = 5 * 1024 * 1024
	GPVNATSPayloadBudgetBytes = NATSMaxPayloadBytes * 3 / 4
	defaultEstimatedPathBytes = 12 * 1024
)

var ErrMappingUnavailable = errors.New("parameter mapping unavailable")

type MappingProvider interface {
	GetMappingSet(ctx context.Context, device *model.Device) (*parammodel.MappingSet, error)
}

// ReadUnsupportedPathProvider returns leaf paths that have been conclusively
// rejected by a product/firmware pair. It is intentionally scoped to reads:
// GPV planning must not inherit write-only restrictions such as CWMP 9008.
type ReadUnsupportedPathProvider interface {
	ListReadUnsupportedPaths(ctx context.Context, productID uuid.UUID, firmwareVersion string) ([]string, error)
}

type PlanCommand struct {
	RunID          uuid.UUID
	Device         *model.Device
	Scope          SyncScope
	RequestedPaths []string
	TriggerReason  TriggerReason
	Priority       int
}

type TaskBatch struct {
	Paths                 []string `json:"paths"`
	EstimatedPayloadBytes int      `json:"estimated_payload_bytes"`
}

type CoverageScope struct {
	Path     string          `json:"path"`
	Complete bool            `json:"complete"`
	Subtree  bool            `json:"subtree,omitempty"`
	Mappings []FrozenMapping `json:"mappings,omitempty"`
}

// FrozenMapping is the minimum immutable projection contract required to
// interpret a task result for the lifetime of a run.
type FrozenMapping struct {
	StandardPath string `json:"standard_path"`
	PrivatePath  string `json:"private_path"`
	Access       string `json:"access,omitempty"`
	DataType     string `json:"data_type,omitempty"`
	IsStorable   bool   `json:"is_storable"`
}

type Plan struct {
	MappingSource   string          `json:"mapping_source"`
	MappingVersion  string          `json:"mapping_version"`
	RequestedPaths  []string        `json:"requested_paths"`
	Batches         []TaskBatch     `json:"batches"`
	Coverage        []CoverageScope `json:"coverage"`
	TaskDescription string          `json:"task_description,omitempty"`
	Priority        int             `json:"priority"`
}

type DefaultPlanner struct {
	mappings         MappingProvider
	unsupportedPaths ReadUnsupportedPathProvider
	batchSize        int
}

func NewPlanner(mappings MappingProvider, batchSize int) *DefaultPlanner {
	if batchSize <= 0 {
		batchSize = 50
	}
	return &DefaultPlanner{mappings: mappings, batchSize: batchSize}
}

func (p *DefaultPlanner) WithUnsupportedPaths(provider ReadUnsupportedPathProvider) *DefaultPlanner {
	p.unsupportedPaths = provider
	return p
}

func (p *DefaultPlanner) Plan(ctx context.Context, cmd PlanCommand) (*Plan, error) {
	if cmd.Device == nil || p.mappings == nil {
		return nil, ErrMappingUnavailable
	}
	set, err := p.mappings.GetMappingSet(ctx, cmd.Device)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMappingUnavailable, err)
	}
	if set == nil {
		return nil, ErrMappingUnavailable
	}
	effectiveSet := *set
	effectiveSet.Mappings = set.Mappings
	if cmd.Scope.IsFull() {
		effectiveSet.Mappings = p.filterReadUnsupportedMappings(ctx, set)
	}
	requestedPaths := normalizeRequestedPaths(&effectiveSet, cmd.RequestedPaths)
	prefixes := storablePrefixes(effectiveSet.Mappings, cmd.Scope, requestedPaths)
	batches := buildBatches(prefixes, p.batchSize)
	coverage := buildCoverage(effectiveSet.Mappings, cmd.Scope, requestedPaths)
	plan := &Plan{
		MappingSource: string(set.Source), MappingVersion: set.SoftwareVersion,
		RequestedPaths: requestedPaths, Batches: batches, Coverage: coverage,
		Priority: cmd.Priority,
	}
	if cmd.TriggerReason == TriggerInformPeriodProbe {
		plan.TaskDescription = "InformPeriodPolicy:GPV:" + cmd.Device.ProductClass
	}
	return plan, nil
}

func (p *DefaultPlanner) filterReadUnsupportedMappings(ctx context.Context, set *parammodel.MappingSet) []parammodel.ParamMapping {
	if p.unsupportedPaths == nil || set == nil || set.ProductID == uuid.Nil || strings.TrimSpace(set.SoftwareVersion) == "" {
		return set.Mappings
	}
	paths, err := p.unsupportedPaths.ListReadUnsupportedPaths(ctx, set.ProductID, set.SoftwareVersion)
	if err != nil || len(paths) == 0 {
		// Capability learning is an optimisation. A temporary read failure must
		// not make an otherwise valid parameter sync unavailable.
		return set.Mappings
	}
	blocked := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			blocked[path] = struct{}{}
		}
	}
	filtered := make([]parammodel.ParamMapping, 0, len(set.Mappings))
	for _, mapping := range set.Mappings {
		if _, found := blocked[mapping.StandardPath]; found {
			continue
		}
		filtered = append(filtered, mapping)
	}
	return filtered
}

func normalizeRequestedPaths(set *parammodel.MappingSet, requested []string) []string {
	translator := parammodel.NewTranslator(set, nil, nil)
	result := make([]string, 0, len(requested))
	for _, path := range requested {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if translated := translator.ToStandard(path); translated.Found {
			path = translated.Translated
		} else if translated, ok := translateRequestedObjectPrefix(path, set.Mappings); ok {
			path = translated
		}
		result = append(result, path)
	}
	return result
}

func buildCoverage(mappings []parammodel.ParamMapping, scope SyncScope, requested []string) []CoverageScope {
	seen := make(map[string]bool)
	if !scope.IsFull() {
		for _, path := range requested {
			path = strings.TrimSpace(path)
			if path != "" {
				seen[path] = requestedPathIsSubtree(path, mappings)
			}
		}
	} else {
		for _, mapping := range mappings {
			if !mapping.IsStorable || !mapping.IsSupported {
				continue
			}
			path := mapping.StandardPath
			subtree := mapping.EntryType == "object"
			if idx := strings.Index(path, "{i}"); idx >= 0 {
				path = path[:idx]
				subtree = true
			}
			if path != "" {
				seen[path] = seen[path] || subtree
			}
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	coverage := make([]CoverageScope, 0, len(paths))
	for _, path := range paths {
		item := CoverageScope{Path: path, Complete: scope.IsFull(), Subtree: seen[path]}
		for _, mapping := range mappings {
			if !mapping.IsSupported || !mappingBelongsToCoverage(mapping.StandardPath, item) {
				continue
			}
			item.Mappings = append(item.Mappings, FrozenMapping{
				StandardPath: mapping.StandardPath, PrivatePath: mapping.PrivatePath,
				Access: mapping.Access, DataType: mapping.DataType, IsStorable: mapping.IsStorable,
			})
		}
		sort.Slice(item.Mappings, func(i, j int) bool {
			return item.Mappings[i].StandardPath < item.Mappings[j].StandardPath
		})
		coverage = append(coverage, item)
	}
	return coverage
}

func requestedPathIsSubtree(path string, mappings []parammodel.ParamMapping) bool {
	if strings.HasSuffix(path, ".") {
		return true
	}
	normalized := normalizeRuntimePath(path)
	for _, mapping := range mappings {
		if mapping.EntryType == "object" && strings.TrimSuffix(mapping.StandardPath, ".") == strings.TrimSuffix(normalized, ".") {
			return true
		}
	}
	return false
}

func translateRequestedObjectPrefix(path string, mappings []parammodel.ParamMapping) (string, bool) {
	hasTrailingDot := strings.HasSuffix(path, ".")
	requestedParts := strings.Split(strings.TrimSuffix(path, "."), ".")
	for _, mapping := range mappings {
		privateParts := strings.Split(strings.TrimSuffix(mapping.PrivatePath, "."), ".")
		standardParts := strings.Split(strings.TrimSuffix(mapping.StandardPath, "."), ".")
		if len(requestedParts) > len(privateParts) {
			continue
		}
		instances := make([]string, 0, 2)
		matched := true
		for i, requestedPart := range requestedParts {
			if privateParts[i] != requestedPart && !(privateParts[i] == "{i}" && isRuntimeInstanceSegment(requestedPart)) {
				matched = false
				break
			}
			if privateParts[i] == "{i}" {
				instances = append(instances, requestedPart)
			}
		}
		if matched {
			end := correspondingTemplateEnd(privateParts, standardParts, len(requestedParts)-1)
			if end < 0 {
				continue
			}
			translated := append([]string(nil), standardParts[:end+1]...)
			instanceIndex := 0
			for i := range translated {
				if translated[i] == "{i}" && instanceIndex < len(instances) {
					translated[i] = instances[instanceIndex]
					instanceIndex++
				}
			}
			result := strings.Join(translated, ".")
			if hasTrailingDot {
				result += "."
			}
			return result, true
		}
	}
	return path, false
}

func correspondingTemplateEnd(from, to []string, fromEnd int) int {
	if fromEnd < 0 || fromEnd >= len(from) {
		return -1
	}
	if from[fromEnd] == "{i}" {
		ordinal := 0
		for i := 0; i <= fromEnd; i++ {
			if from[i] == "{i}" {
				ordinal++
			}
		}
		seen := 0
		for i, part := range to {
			if part == "{i}" {
				seen++
				if seen == ordinal {
					return i
				}
			}
		}
		return -1
	}
	for i := len(to) - 1; i >= 0; i-- {
		if to[i] == from[fromEnd] {
			return i
		}
	}
	return -1
}

func isRuntimeInstanceSegment(segment string) bool {
	if segment == "" {
		return false
	}
	for _, r := range segment {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func mappingBelongsToCoverage(path string, coverage CoverageScope) bool {
	normalizedCoverage := normalizeRuntimePath(coverage.Path)
	if coverage.Subtree {
		return strings.HasPrefix(path, normalizedCoverage)
	}
	return path == normalizedCoverage
}

func normalizeRuntimePath(path string) string {
	parts := strings.Split(path, ".")
	for i, part := range parts {
		if part == "" {
			continue
		}
		allDigits := true
		for _, r := range part {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			parts[i] = "{i}"
		}
	}
	return strings.Join(parts, ".")
}

func storablePrefixes(mappings []parammodel.ParamMapping, scope SyncScope, requested []string) []string {
	if scope.IsFull() {
		return provision.PathBStorablePrefixes(mappings)
	}
	return provision.PathBStorablePrefixesForStandardPaths(mappings, requested)
}

func buildBatches(paths []string, batchSize int) []TaskBatch {
	legacyBatches := provision.PathBGPVBatches(paths, batchSize)
	batches := make([]TaskBatch, 0, len(legacyBatches))
	for _, paths := range legacyBatches {
		batchPaths := append([]string(nil), paths...)
		batches = append(batches, TaskBatch{
			Paths: batchPaths, EstimatedPayloadBytes: len(batchPaths) * defaultEstimatedPathBytes,
		})
	}
	return batches
}
