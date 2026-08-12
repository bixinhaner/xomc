package paramsync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
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
	pathPlan := planStorablePaths(effectiveSet.Mappings, cmd.Scope, requestedPaths, cmd.Device)
	batches := buildBatchesWithIsolation(pathPlan.Paths, pathPlan.Isolated, p.batchSize)
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
			path = normalizeRequestedCoveragePath(path)
			if path != "" {
				subtree := requestedPathIsSubtree(path, mappings)
				if subtree {
					path = ensureObjectCoveragePath(path)
				}
				seen[path] = subtree
			}
		}
	} else {
		for _, mapping := range mappings {
			if !mapping.IsStorable || !mapping.IsSupported {
				continue
			}
			path, subtree := fullCoveragePath(mapping)
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
		complete := scope.IsFull() || (scope == SyncScopePartial && seen[path])
		item := CoverageScope{Path: path, Complete: complete, Subtree: seen[path]}
		for _, mapping := range mappings {
			if !mapping.IsSupported {
				continue
			}
			if scope.IsFull() {
				mappingPath, _ := fullCoveragePath(mapping)
				if mappingPath != path {
					continue
				}
			} else if !mappingBelongsToCoverage(mapping.StandardPath, item) {
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

func normalizeRequestedCoveragePath(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasSuffix(path, "{i}.") {
		return strings.TrimSuffix(path, "{i}.")
	}
	return path
}

func ensureObjectCoveragePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasSuffix(path, ".") {
		return path
	}
	return path + "."
}

// fullCoveragePath keeps failure and deletion coverage at the deepest repeated
// object represented by a mapping. Collapsing at the first {i} made one 9005
// under FAPService protect every unrelated table below it from reconciliation.
func fullCoveragePath(mapping parammodel.ParamMapping) (string, bool) {
	path := mapping.StandardPath
	if idx := strings.LastIndex(path, "{i}"); idx >= 0 {
		return path[:idx], true
	}
	return path, mapping.EntryType == "object"
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

const (
	mlnDCProductClass         = "FAP/MLN/DC"
	mlnDCRFStatusStandardPath = "Device.Services.FAPService.{i}.FAPControl.LTE.RFTxStatus"
	mlnDCRFStatusPrivatePath  = "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.AdminCellState"
	mlnDCCarrierCount         = 2
)

type storablePathPlan struct {
	Paths    []string
	Isolated []string
}

func planStorablePaths(
	mappings []parammodel.ParamMapping,
	scope SyncScope,
	requested []string,
	device *model.Device,
) storablePathPlan {
	if !isMLNDCDeviceModel(device) || !hasMLNDCRFStatusMapping(mappings) {
		return storablePathPlan{Paths: genericStorablePrefixes(mappings, scope, requested)}
	}

	otherMappings := make([]parammodel.ParamMapping, 0, len(mappings)-1)
	for _, mapping := range mappings {
		if !isMLNDCRFStatusMapping(mapping) {
			otherMappings = append(otherMappings, mapping)
		}
	}
	rfPaths := mlnDCRFStatusPaths(scope, requested, mlnDCCarrierCount)
	paths := append(genericStorablePrefixes(otherMappings, scope, requested), rfPaths...)
	return storablePathPlan{
		Paths:    sortedUniquePaths(paths),
		Isolated: rfPaths,
	}
}

func genericStorablePrefixes(
	mappings []parammodel.ParamMapping,
	scope SyncScope,
	requested []string,
) []string {
	if scope.IsFull() {
		return provision.PathBStorablePrefixes(mappings)
	}
	return provision.PathBStorablePrefixesForStandardPaths(mappings, requested)
}

func isMLNDCDeviceModel(device *model.Device) bool {
	return device != nil &&
		strings.EqualFold(strings.TrimSpace(device.ProductClass), mlnDCProductClass)
}

func hasMLNDCRFStatusMapping(mappings []parammodel.ParamMapping) bool {
	for _, mapping := range mappings {
		if isMLNDCRFStatusMapping(mapping) {
			return true
		}
	}
	return false
}

func isMLNDCRFStatusMapping(mapping parammodel.ParamMapping) bool {
	return mapping.IsStorable && mapping.IsSupported &&
		mapping.StandardPath == mlnDCRFStatusStandardPath &&
		mapping.PrivatePath == mlnDCRFStatusPrivatePath
}

func mlnDCRFStatusPaths(scope SyncScope, requested []string, carrierCount int) []string {
	if carrierCount <= 0 {
		return nil
	}

	paths := make([]string, 0, carrierCount)
	for instance := 1; instance <= carrierCount; instance++ {
		if !scope.IsFull() && !requestsMLNDCRFCarrier(requested, instance) {
			continue
		}
		paths = append(paths, instantiatePathTemplate(mlnDCRFStatusPrivatePath, instance))
	}
	return paths
}

func requestsMLNDCRFCarrier(requested []string, instance int) bool {
	standardPath := instantiatePathTemplate(mlnDCRFStatusStandardPath, instance)
	for _, requestedPath := range requested {
		concreteRequestedPath := instantiatePathTemplate(requestedPath, instance)
		if concreteRequestedPath == standardPath ||
			(strings.HasSuffix(concreteRequestedPath, ".") &&
				strings.HasPrefix(standardPath, concreteRequestedPath)) {
			return true
		}
	}
	return false
}

func instantiatePathTemplate(path string, instance int) string {
	return strings.Replace(path, "{i}", strconv.Itoa(instance), 1)
}

func sortedUniquePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		if path != "" {
			seen[path] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for path := range seen {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func buildBatches(paths []string, batchSize int) []TaskBatch {
	return buildBatchesWithIsolation(paths, nil, batchSize)
}

func buildBatchesWithIsolation(paths, isolatedPaths []string, batchSize int) []TaskBatch {
	isolated := make(map[string]struct{}, len(isolatedPaths))
	for _, path := range isolatedPaths {
		isolated[path] = struct{}{}
	}
	sharedPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, found := isolated[path]; !found {
			sharedPaths = append(sharedPaths, path)
		}
	}

	legacyBatches := provision.PathBGPVBatches(sharedPaths, batchSize)
	batches := make([]TaskBatch, 0, len(legacyBatches)+len(isolatedPaths))
	for _, paths := range legacyBatches {
		batchPaths := append([]string(nil), paths...)
		batches = append(batches, TaskBatch{
			Paths: batchPaths, EstimatedPayloadBytes: len(batchPaths) * defaultEstimatedPathBytes,
		})
	}
	for _, path := range isolatedPaths {
		batches = append(batches, TaskBatch{
			Paths: []string{path}, EstimatedPayloadBytes: defaultEstimatedPathBytes,
		})
	}
	return batches
}
