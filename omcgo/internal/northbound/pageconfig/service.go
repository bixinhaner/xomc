package pageconfig

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	nbsnmp "github.com/omcgo/omcgo/internal/northbound/snmp"
)

type Repository interface {
	EnsureDefaults(ctx context.Context, catalog *Catalog) error
	ListFileProfiles(ctx context.Context) ([]FileProfile, error)
	CreateFileProfile(ctx context.Context, profile FileProfile) (*FileProfile, error)
	UpdateFileProfile(ctx context.Context, idOrCode string, req UpdateFileProfileRequest) (*FileProfile, error)
	ListInventoryProfiles(ctx context.Context) ([]InventoryProfile, error)
	UpdateInventoryProfile(ctx context.Context, idOrCode string, req UpdateInventoryProfileRequest) (*InventoryProfile, error)
	CreateFileRun(ctx context.Context, run FileRun) (*FileRun, error)
	ListFileRuns(ctx context.Context, filter RunFilter) (RunListResult, error)
	GetFileRun(ctx context.Context, id string) (*FileRun, error)
	UpdateFileRunLocalArchive(ctx context.Context, id string, result LocalArchiveResult, clearContent bool) (*FileRun, error)
	LoadDeviceSnapshotRows(ctx context.Context, tech string, limit int) ([]ExportDataRow, error)
	LoadOMCInventoryRows(ctx context.Context) ([]ExportDataRow, error)
	LoadPMMetricRows(ctx context.Context, req PMMetricQuery) ([]ExportDataRow, error)
	LoadMRRows(ctx context.Context, objectCode string, limit int) ([]ExportDataRow, error)
	LoadLogRows(ctx context.Context, objectCode string, windowStart time.Time, windowEnd time.Time, limit int) ([]ExportDataRow, error)
	ListPMMetricFields(ctx context.Context, filter FieldFilter) ([]FieldDefinition, error)
	ListDeviceInfoFields(ctx context.Context, filter FieldFilter) ([]FieldDefinition, error)
	ValidatePMMetricPaths(ctx context.Context, metricPaths []string) ([]string, error)
	EnsureExtendedDefaults(ctx context.Context) error
	ListDeliveryTargets(ctx context.Context, filter DeliveryTargetFilter) ([]DeliveryTarget, error)
	ListActiveDeliveryTargets(ctx context.Context, scope DeliveryScope, ownerCode string) ([]DeliveryTarget, error)
	ReplaceDeliveryTargets(ctx context.Context, req ReplaceDeliveryTargetsRequest) ([]DeliveryTarget, error)
	ListSNMPAlarmTargets(ctx context.Context) ([]SNMPAlarmTarget, error)
	GetSNMPAlarmTargetForSend(ctx context.Context, key string) (*SNMPAlarmTarget, error)
	ListActiveSNMPAlarmTargetsForSend(ctx context.Context) ([]SNMPAlarmTarget, error)
	UpdateSNMPAlarmTarget(ctx context.Context, key string, target SNMPAlarmTarget) (*SNMPAlarmTarget, error)
	ListSocketAlarmConfigs(ctx context.Context) ([]SocketAlarmConfig, error)
	ListActiveSocketAlarmConfigsForServe(ctx context.Context) ([]SocketAlarmConfig, error)
	UpdateSocketAlarmConfig(ctx context.Context, key string, config SocketAlarmConfig) (*SocketAlarmConfig, error)
	ListAPIConfigs(ctx context.Context) ([]APIConfig, error)
	UpdateAPIConfig(ctx context.Context, key string, enabled bool) (*APIConfig, error)
	ListAPIClients(ctx context.Context) ([]APIClient, error)
	ReplaceAPIClients(ctx context.Context, req ReplaceAPIClientsRequest) ([]APIClient, error)
	AuthenticateAPIClient(ctx context.Context, credential string, remoteIP string, apiKey string) (*APIClient, error)
	ListAPIUsers(ctx context.Context) ([]APIUser, error)
	ReplaceAPIUsers(ctx context.Context, req ReplaceAPIUsersRequest) ([]APIUser, error)
	LoginAPIUser(ctx context.Context, req APIUserLoginRequest) (*APIUserToken, error)
	CreateEvent(ctx context.Context, event PageConfigEvent) (*PageConfigEvent, error)
	ListEvents(ctx context.Context, filter EventFilter) (EventListResult, error)
	GetEvent(ctx context.Context, id string) (*PageConfigEvent, error)
	CleanupExpiredResults(ctx context.Context, runBefore time.Time, eventBefore time.Time) (ResultCleanupSummary, error)
}

type MRSourceStore interface {
	Get(ctx context.Context, objectKey string) ([]byte, error)
}

type Service struct {
	catalog             *Catalog
	repo                Repository
	alarmStore          alarm.AlarmStore
	snmpSender          nbsnmp.Sender
	localArchive        LocalArchiveStore
	localArchiveOptions LocalArchiveOptions
	mrSourceStore       MRSourceStore

	socketConfigListenersMu sync.RWMutex
	socketConfigListeners   []func()
	snmpConfigListenersMu   sync.RWMutex
	snmpConfigListeners     []func()
}

const defaultResultRetentionDays = 90

func NewService(catalog *Catalog) *Service {
	if catalog == nil {
		catalog = NewDefaultCatalog()
	}
	return &Service{catalog: catalog}
}

func (s *Service) SetRepository(repo Repository) {
	s.repo = repo
}

func (s *Service) SetAlarmStore(store alarm.AlarmStore) {
	s.alarmStore = store
}

func (s *Service) SetSNMPSender(sender nbsnmp.Sender) {
	s.snmpSender = sender
}

func (s *Service) SetLocalArchive(store LocalArchiveStore, opts LocalArchiveOptions) {
	s.localArchive = store
	s.localArchiveOptions = normalizeLocalArchiveOptions(opts)
}

func (s *Service) SetMRSourceStore(store MRSourceStore) {
	s.mrSourceStore = store
}

func (s *Service) RegisterSocketConfigChangeListener(listener func()) {
	if s == nil || listener == nil {
		return
	}
	s.socketConfigListenersMu.Lock()
	defer s.socketConfigListenersMu.Unlock()
	s.socketConfigListeners = append(s.socketConfigListeners, listener)
}

func (s *Service) notifySocketConfigChanged() {
	if s == nil {
		return
	}
	s.socketConfigListenersMu.RLock()
	listeners := append([]func(){}, s.socketConfigListeners...)
	s.socketConfigListenersMu.RUnlock()
	for _, listener := range listeners {
		listener()
	}
}

func (s *Service) RegisterSNMPConfigChangeListener(listener func()) {
	if s == nil || listener == nil {
		return
	}
	s.snmpConfigListenersMu.Lock()
	defer s.snmpConfigListenersMu.Unlock()
	s.snmpConfigListeners = append(s.snmpConfigListeners, listener)
}

func (s *Service) notifySNMPConfigChanged() {
	if s == nil {
		return
	}
	s.snmpConfigListenersMu.RLock()
	listeners := append([]func(){}, s.snmpConfigListeners...)
	s.snmpConfigListenersMu.RUnlock()
	for _, listener := range listeners {
		listener()
	}
}

func NewServiceWithRepository(catalog *Catalog, repo Repository) *Service {
	svc := NewService(catalog)
	svc.SetRepository(repo)
	return svc
}

func (s *Service) Overview(ctx context.Context) (Overview, error) {
	if s.repo != nil {
		if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
			return Overview{}, err
		}
	}
	return s.catalog.Overview(), nil
}

func (s *Service) ListFileProfiles(ctx context.Context) ([]FileProfile, error) {
	if s.repo == nil {
		return s.catalog.FileProfiles(), nil
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return nil, err
	}
	return s.repo.ListFileProfiles(ctx)
}

func (s *Service) CreateFileProfile(ctx context.Context, profile FileProfile) (*FileProfile, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return nil, err
	}

	profile.Code = strings.ToUpper(strings.TrimSpace(profile.Code))
	if profile.Code == "" {
		profile.Code = strings.ToUpper(strings.TrimSpace(profile.ID))
	}
	if !validFileProfileCode(profile.Code) {
		return nil, fmt.Errorf("%w: file profile code must match S0000 format", commonerrors.ErrInvalidInput)
	}
	profile.ID = profile.Code
	profile.Name = strings.TrimSpace(profile.Name)
	if profile.Name == "" {
		profile.Name = profile.Code
	}
	profile.Vendor = strings.TrimSpace(profile.Vendor)
	if profile.Vendor == "" {
		profile.Vendor = "Baicells"
	}
	profile.ScenarioName = strings.TrimSpace(profile.ScenarioName)
	if profile.ScenarioName == "" {
		profile.ScenarioName = profile.Name
	}
	profile.ScenarioNameEn = strings.TrimSpace(profile.ScenarioNameEn)
	if profile.ScenarioNameEn == "" {
		profile.ScenarioNameEn = profile.ScenarioName
	}
	profile.Description = strings.TrimSpace(profile.Description)
	if profile.Flags == nil {
		profile.Flags = []string{}
	}
	if profile.Groups == nil {
		profile.Groups = []FileGroup{}
	}
	if profile.Status == "" {
		profile.Status = statusFromEnabled(profile.Enabled)
	}
	if profile.Status != StatusNormal && profile.Status != StatusTerminated {
		return nil, fmt.Errorf("%w: unsupported status %q", commonerrors.ErrInvalidInput, profile.Status)
	}

	enabled := profile.Enabled
	result := s.validateFileProfile(UpdateFileProfileRequest{
		Name:           profile.Name,
		Vendor:         profile.Vendor,
		ScenarioName:   profile.ScenarioName,
		ScenarioNameEn: profile.ScenarioNameEn,
		Description:    profile.Description,
		Flags:          profile.Flags,
		Enabled:        &enabled,
		Status:         profile.Status,
		Groups:         profile.Groups,
	})
	if !result.Valid {
		return nil, fmt.Errorf("%w: invalid file profile: %v", commonerrors.ErrInvalidInput, result.Errors)
	}
	return s.repo.CreateFileProfile(ctx, profile)
}

func (s *Service) UpdateFileProfile(ctx context.Context, idOrCode string, req UpdateFileProfileRequest) (*FileProfile, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return nil, err
	}
	result := s.validateFileProfile(req)
	if !result.Valid {
		return nil, fmt.Errorf("%w: invalid file profile: %v", commonerrors.ErrInvalidInput, result.Errors)
	}
	return s.repo.UpdateFileProfile(ctx, idOrCode, req)
}

func (s *Service) PreviewFileProfile(ctx context.Context, idOrCode string) (FileProfilePreview, error) {
	profile, err := s.getFileProfile(ctx, idOrCode)
	if err != nil {
		return FileProfilePreview{}, err
	}
	return buildFileProfilePreview(*profile), nil
}

func (s *Service) getFileProfile(ctx context.Context, idOrCode string) (*FileProfile, error) {
	items, err := s.ListFileProfiles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if strings.EqualFold(items[i].Code, idOrCode) || strings.EqualFold(items[i].ID, idOrCode) {
			return &items[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (s *Service) ListInventoryProfiles(ctx context.Context) ([]InventoryProfile, error) {
	if s.repo == nil {
		return s.catalog.InventoryProfiles(), nil
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return nil, err
	}
	return s.repo.ListInventoryProfiles(ctx)
}

func (s *Service) UpdateInventoryProfile(ctx context.Context, idOrCode string, req UpdateInventoryProfileRequest) (*InventoryProfile, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureDefaults(ctx, s.catalog); err != nil {
		return nil, err
	}
	result := s.validateInventoryProfile(req)
	if !result.Valid {
		return nil, fmt.Errorf("%w: invalid inventory profile: %v", commonerrors.ErrInvalidInput, result.Errors)
	}
	return s.repo.UpdateInventoryProfile(ctx, idOrCode, req)
}

func (s *Service) ListFields(ctx context.Context, filter FieldFilter) []FieldDefinition {
	if s.repo != nil && filter.Domain == DomainPM {
		if fields, err := s.repo.ListPMMetricFields(ctx, filter); err == nil && len(fields) > 0 {
			return fields
		}
	}
	fields := s.catalog.Fields(filter)
	if s.repo != nil && (filter.Domain == DomainCM || filter.Domain == DomainInventory) {
		if dynamicFields, err := s.repo.ListDeviceInfoFields(ctx, filter); err == nil && len(dynamicFields) > 0 {
			return appendMissingFieldDefinitions(fields, dynamicFields)
		}
	}
	return fields
}

func appendMissingFieldDefinitions(base []FieldDefinition, extra []FieldDefinition) []FieldDefinition {
	seen := make(map[string]struct{}, len(base)+len(extra))
	for _, field := range base {
		seen[strings.ToLower(field.Key)] = struct{}{}
	}
	out := make([]FieldDefinition, 0, len(base)+len(extra))
	out = append(out, base...)
	for _, field := range extra {
		key := strings.ToLower(field.Key)
		if _, ok := seen[key]; ok {
			continue
		}
		out = append(out, field)
		seen[key] = struct{}{}
	}
	return out
}

func (s *Service) Validate(ctx context.Context, req ValidateRequest) ValidationResult {
	result := s.catalog.Validate(req)
	if req.Domain == DomainPM && len(req.MetricPaths) > 0 {
		if s.repo == nil {
			result.Warnings = append(result.Warnings, "PM metric_path validation requires a configured repository")
			result.Valid = len(result.Errors) == 0
			return result
		}
		missing, err := s.repo.ValidatePMMetricPaths(ctx, req.MetricPaths)
		if err != nil {
			result.Errors = append(result.Errors, "PM metric_path validation failed: "+err.Error())
		}
		for _, path := range missing {
			result.Errors = append(result.Errors, fmt.Sprintf("unknown PM metric_path %q", path))
		}
	}
	result.Valid = len(result.Errors) == 0
	return result
}

func validFileProfileCode(code string) bool {
	if len(code) != 5 || code[0] != 'S' {
		return false
	}
	for _, ch := range code[1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func (s *Service) ListDeliveryTargets(ctx context.Context, filter DeliveryTargetFilter) ([]DeliveryTarget, error) {
	if s.repo == nil {
		return defaultDeliveryTargets(), nil
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListDeliveryTargets(ctx, filter)
}

func (s *Service) ReplaceDeliveryTargets(ctx context.Context, req ReplaceDeliveryTargetsRequest) ([]DeliveryTarget, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	for i := range req.Items {
		req.Items[i].Scope = req.Scope
		req.Items[i].OwnerCode = req.OwnerCode
		req.Items[i] = normalizeDeliveryTarget(req.Items[i])
	}
	if err := validateDeliveryTargets(req); err != nil {
		return nil, err
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ReplaceDeliveryTargets(ctx, req)
}

func (s *Service) ListSNMPAlarmTargets(ctx context.Context) ([]SNMPAlarmTarget, error) {
	if s.repo == nil {
		return defaultSNMPAlarmTargets(), nil
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListSNMPAlarmTargets(ctx)
}

func (s *Service) UpdateSNMPAlarmTarget(ctx context.Context, key string, target SNMPAlarmTarget) (*SNMPAlarmTarget, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	target = normalizeSNMPAlarmTarget(key, target)
	if err := validateSNMPAlarmTarget(target); err != nil {
		return nil, err
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateSNMPAlarmTarget(ctx, key, target)
	if err != nil {
		return nil, err
	}
	s.notifySNMPConfigChanged()
	return updated, nil
}

func (s *Service) listActiveSNMPMIBTargetsForServe(ctx context.Context) ([]SNMPAlarmTarget, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListActiveSNMPAlarmTargetsForSend(ctx)
}

func (s *Service) ListSocketAlarmConfigs(ctx context.Context) ([]SocketAlarmConfig, error) {
	if s.repo == nil {
		return defaultSocketAlarmConfigs(), nil
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListSocketAlarmConfigs(ctx)
}

func (s *Service) ListActiveSocketAlarmConfigsForServe(ctx context.Context) ([]SocketAlarmConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListActiveSocketAlarmConfigsForServe(ctx)
}

func (s *Service) UpdateSocketAlarmConfig(ctx context.Context, key string, config SocketAlarmConfig) (*SocketAlarmConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	config = normalizeSocketAlarmConfig(key, config)
	if err := validateSocketAlarmConfig(config); err != nil {
		return nil, err
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	updated, err := s.repo.UpdateSocketAlarmConfig(ctx, key, config)
	if err != nil {
		return nil, err
	}
	s.notifySocketConfigChanged()
	return updated, nil
}

func (s *Service) ListAPIConfigs(ctx context.Context) ([]APIConfig, error) {
	if s.repo == nil {
		return defaultAPIConfigs(), nil
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListAPIConfigs(ctx)
}

func (s *Service) UpdateAPIConfig(ctx context.Context, key string, enabled bool) (*APIConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.UpdateAPIConfig(ctx, key, enabled)
}

func (s *Service) UpdateAllAPIConfigs(ctx context.Context, enabled bool) ([]APIConfig, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	configs, err := s.repo.ListAPIConfigs(ctx)
	if err != nil {
		return nil, err
	}
	updated := make([]APIConfig, 0, len(configs))
	for _, config := range configs {
		item, err := s.repo.UpdateAPIConfig(ctx, config.Key, enabled)
		if err != nil {
			return nil, err
		}
		updated = append(updated, *item)
	}
	return updated, nil
}

func (s *Service) IsAPIConfigEnabled(ctx context.Context, key string) (bool, error) {
	if s == nil || s.repo == nil {
		return true, nil
	}
	configs, err := s.ListAPIConfigs(ctx)
	if err != nil {
		return false, err
	}
	for _, config := range configs {
		if strings.EqualFold(config.Key, key) {
			return config.Enabled, nil
		}
	}
	return false, nil
}

func (s *Service) ListAPIClients(ctx context.Context) ([]APIClient, error) {
	if s.repo == nil {
		return []APIClient{}, nil
	}
	return s.repo.ListAPIClients(ctx)
}

func (s *Service) ReplaceAPIClients(ctx context.Context, req ReplaceAPIClientsRequest) ([]APIClient, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	seen := map[string]struct{}{}
	for i := range req.Items {
		req.Items[i] = normalizeAPIClient(req.Items[i])
		if _, ok := seen[req.Items[i].ClientKey]; ok {
			return nil, fmt.Errorf("%w: duplicate API client_key %s", commonerrors.ErrInvalidInput, req.Items[i].ClientKey)
		}
		seen[req.Items[i].ClientKey] = struct{}{}
	}
	return s.repo.ReplaceAPIClients(ctx, req)
}

func (s *Service) AuthenticateAPIClient(ctx context.Context, credential string, remoteIP string, apiKey string) (*APIClient, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: northbound API user token is required", commonerrors.ErrUnauthorized)
	}
	return s.repo.AuthenticateAPIClient(ctx, credential, remoteIP, apiKey)
}

func (s *Service) ListAPIUsers(ctx context.Context) ([]APIUser, error) {
	if s.repo == nil {
		return []APIUser{}, nil
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListAPIUsers(ctx)
}

func (s *Service) ReplaceAPIUsers(ctx context.Context, req ReplaceAPIUsersRequest) ([]APIUser, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for i := range req.Items {
		req.Items[i] = normalizeAPIUser(req.Items[i])
		if _, ok := seen[req.Items[i].Username]; ok {
			return nil, fmt.Errorf("%w: duplicate northbound API username %s", commonerrors.ErrInvalidInput, req.Items[i].Username)
		}
		seen[req.Items[i].Username] = struct{}{}
	}
	return s.repo.ReplaceAPIUsers(ctx, req)
}

func (s *Service) LoginAPIUser(ctx context.Context, req APIUserLoginRequest) (*APIUserToken, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.LoginAPIUser(ctx, req)
}

func (s *Service) ListEvents(ctx context.Context, filter EventFilter) (EventListResult, error) {
	if s.repo == nil {
		return EventListResult{}, fmt.Errorf("northbound page-config repository is not configured")
	}
	return s.repo.ListEvents(ctx, filter)
}

func (s *Service) GetEvent(ctx context.Context, id string) (*PageConfigEvent, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	return s.repo.GetEvent(ctx, id)
}

func (s *Service) CleanupExpiredResults(ctx context.Context, policy ResultRetentionPolicy) (ResultCleanupSummary, error) {
	if s.repo == nil {
		return ResultCleanupSummary{}, fmt.Errorf("northbound page-config repository is not configured")
	}
	runDays := policy.RunRetentionDays
	if runDays <= 0 {
		runDays = defaultResultRetentionDays
	}
	eventDays := policy.EventRetentionDays
	if eventDays <= 0 {
		eventDays = defaultResultRetentionDays
	}
	now := time.Now()
	runBefore := now.AddDate(0, 0, -runDays)
	eventBefore := now.AddDate(0, 0, -eventDays)
	summary, err := s.repo.CleanupExpiredResults(ctx, runBefore, eventBefore)
	if err != nil {
		return ResultCleanupSummary{}, err
	}
	summary.RunRetentionDays = runDays
	summary.EventRetentionDays = eventDays
	summary.RunBefore = runBefore
	summary.EventBefore = eventBefore
	return summary, nil
}

func (s *Service) CleanupLocalArchive(ctx context.Context, now time.Time) (LocalArchiveCleanupSummary, error) {
	if s.localArchive == nil {
		return LocalArchiveCleanupSummary{}, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	opts := normalizeLocalArchiveOptions(s.localArchiveOptions)
	before := now.AddDate(0, 0, -opts.RetentionDays)
	summary, err := s.localArchive.CleanupBefore(ctx, LocalArchiveCleanupRequest{
		Prefix: opts.Prefix,
		Before: before,
	})
	summary.RetentionDays = opts.RetentionDays
	summary.Prefix = opts.Prefix
	summary.Before = before
	return summary, err
}

func (s *Service) TestDeliveryTarget(ctx context.Context, target DeliveryTarget) (*PageConfigEvent, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	target = normalizeDeliveryTarget(target)
	if err := validateDeliveryTargets(ReplaceDeliveryTargetsRequest{
		Scope:     target.Scope,
		OwnerCode: target.OwnerCode,
		Items:     []DeliveryTarget{target},
	}); err != nil {
		return nil, err
	}
	result := probeDeliveryTarget(ctx, target)
	event := eventFromDeliveryTest(target, result)
	return s.repo.CreateEvent(ctx, event)
}

func (s *Service) TestSNMPAlarmTarget(ctx context.Context, key string) (*PageConfigEvent, error) {
	target, err := s.snmpTargetForSend(ctx, key)
	if err != nil {
		return nil, err
	}
	if s.snmpSender == nil {
		return s.repo.CreateEvent(ctx, eventFromSNMPTest(*target))
	}
	event := s.sendSNMPAlarmToTarget(ctx, *target, sampleSNMPAlarm(), "message_test")
	return s.repo.CreateEvent(ctx, event)
}

func (s *Service) SendSNMPAlarm(ctx context.Context, alarm *nbsnmp.AlarmEvent) ([]PageConfigEvent, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	targets, err := s.repo.ListActiveSNMPAlarmTargetsForSend(ctx)
	if err != nil {
		return nil, err
	}
	events := make([]PageConfigEvent, 0, len(targets))
	for _, target := range targets {
		event := s.sendSNMPAlarmToTarget(ctx, target, alarm, "alarm_send")
		created, err := s.repo.CreateEvent(ctx, event)
		if err != nil {
			return events, err
		}
		events = append(events, *created)
	}
	return events, nil
}

func (s *Service) snmpTargetForSend(ctx context.Context, key string) (*SNMPAlarmTarget, error) {
	if s.repo == nil {
		targets := defaultSNMPAlarmTargets()
		for i := range targets {
			if strings.EqualFold(targets[i].Key, key) {
				return &targets[i], nil
			}
		}
		return nil, commonerrors.ErrNotFound
	}
	if err := s.repo.EnsureExtendedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetSNMPAlarmTargetForSend(ctx, key)
}

func (s *Service) TestSocketAlarmConfig(ctx context.Context, key string) (*PageConfigEvent, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	configs, err := s.ListSocketAlarmConfigs(ctx)
	if err != nil {
		return nil, err
	}
	for _, config := range configs {
		if strings.EqualFold(config.Key, key) {
			return s.repo.CreateEvent(ctx, eventFromSocketTest(config))
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (s *Service) TestAPIConfig(ctx context.Context, key string) (*PageConfigEvent, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("northbound page-config repository is not configured")
	}
	configs, err := s.ListAPIConfigs(ctx)
	if err != nil {
		return nil, err
	}
	for _, config := range configs {
		if strings.EqualFold(config.Key, key) {
			return s.repo.CreateEvent(ctx, eventFromAPITest(config))
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (s *Service) validateFileProfile(req UpdateFileProfileRequest) ValidationResult {
	result := ValidationResult{Valid: true}
	if req.Status == "" {
		req.Status = StatusTerminated
	}
	if req.Status != StatusNormal && req.Status != StatusTerminated {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported status %q", req.Status))
	}
	for _, group := range req.Groups {
		if errMessage := validateSingleTechFileGroup(group); errMessage != "" {
			result.Errors = append(result.Errors, errMessage)
		}
		vr := s.catalog.Validate(ValidateRequest{
			ProfileKind:        "file",
			Domain:             group.Domain,
			Format:             group.Format,
			Period:             group.Period,
			CompressionEnabled: group.CompressionEnabled,
			CompressionFormat:  group.CompressionFormat,
			Objects:            group.Objects,
		})
		result.Errors = append(result.Errors, vr.Errors...)
		result.Warnings = append(result.Warnings, vr.Warnings...)
		result.Errors = append(result.Errors, validateFileGroupTemplates(group)...)
	}
	result.Valid = len(result.Errors) == 0
	return result
}

func validateSingleTechFileGroup(group FileGroup) string {
	if group.Domain != DomainCM && group.Domain != DomainPM {
		return ""
	}
	techs := make(map[string]struct{}, len(group.Objects))
	for _, object := range group.Objects {
		tech := normalizeTech(object.Tech)
		if tech == "" {
			tech = "LTE"
		}
		switch tech {
		case "LTE", "GNB", "GSM":
			techs[tech] = struct{}{}
		default:
			return fmt.Sprintf("group %q uses unsupported technology %q; choose ENB, GNB or GSM", group.ID, object.Tech)
		}
	}
	if len(techs) > 1 {
		return fmt.Sprintf("group %q mixes multiple technologies; create one group row per ENB/GNB/GSM technology", group.ID)
	}
	return ""
}

func (s *Service) validateInventoryProfile(req UpdateInventoryProfileRequest) ValidationResult {
	var result ValidationResult
	if req.Status != "" && req.Status != StatusNormal && req.Status != StatusTerminated {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported status %q", req.Status))
	}
	if req.Period != "" {
		if _, ok := supportedPeriods[req.Period]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("unsupported period %q", req.Period))
		}
	}
	if req.StartMinute != nil && (*req.StartMinute < 0 || *req.StartMinute > 59) {
		result.Errors = append(result.Errors, fmt.Sprintf("start_minute %d must be between 0 and 59", *req.StartMinute))
	}
	if req.ObjectCode != "" {
		if _, ok := supportedObjects[DomainInventory][req.ObjectCode]; !ok {
			result.Errors = append(result.Errors, fmt.Sprintf("object %q is not supported by domain %q", req.ObjectCode, DomainInventory))
		}
	}
	if req.CompressionEnabled != nil && *req.CompressionEnabled {
		switch req.CompressionFormat {
		case "", CompressionZip, CompressionGz:
		default:
			result.Errors = append(result.Errors, fmt.Sprintf("unsupported compression format %q", req.CompressionFormat))
		}
	}
	result.Valid = len(result.Errors) == 0
	return result
}
