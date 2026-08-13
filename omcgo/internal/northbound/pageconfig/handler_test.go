package pageconfig

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	nbsnmp "github.com/omcgo/omcgo/internal/northbound/snmp"
	"github.com/pkg/sftp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(NewService(NewDefaultCatalog()), zap.NewNop())
	h.RegisterRoutes(r.Group("/api/v1/northbound"))
	return r
}

type fakeRepository struct {
	fileProfiles      []FileProfile
	inventoryProfiles []InventoryProfile
	runs              []FileRun
	deliveryTargets   []DeliveryTarget
	snmpTargets       []SNMPAlarmTarget
	socketConfigs     []SocketAlarmConfig
	apiConfigs        []APIConfig
	apiClients        []APIClient
	apiTokens         map[string]string
	apiInvocationLogs []APIInvocationLog
	events            []PageConfigEvent
	deviceRows        []ExportDataRow
	pmRows            []ExportDataRow
	mrRows            []ExportDataRow
	logRows           []ExportDataRow
}

type fakeLocalArchiveStore struct {
	bucket         string
	puts           []LocalArchiveObject
	putErr         error
	cleanupReqs    []LocalArchiveCleanupRequest
	cleanupSummary LocalArchiveCleanupSummary
	cleanupErr     error
}

type fakeMRSourceStore struct {
	objects map[string][]byte
}

func (s *fakeMRSourceStore) Get(_ context.Context, objectKey string) ([]byte, error) {
	if s == nil {
		return nil, commonerrors.ErrNotFound
	}
	content, ok := s.objects[objectKey]
	if !ok {
		return nil, commonerrors.ErrNotFound
	}
	return append([]byte(nil), content...), nil
}

func (s *fakeLocalArchiveStore) Put(_ context.Context, object LocalArchiveObject) (LocalArchiveResult, error) {
	object.Content = append([]byte(nil), object.Content...)
	s.puts = append(s.puts, object)
	return LocalArchiveResult{
		Bucket:      s.bucket,
		ObjectKey:   object.Key,
		ObjectName:  object.Name,
		Bytes:       int64(len(object.Content)),
		ContentType: object.ContentType,
		RunID:       object.Run.ID,
		ProfileKind: object.Run.ProfileKind,
		ProfileCode: object.Run.ProfileCode,
	}, s.putErr
}

func (s *fakeLocalArchiveStore) Get(_ context.Context, result LocalArchiveResult) ([]byte, error) {
	for _, put := range s.puts {
		if put.Key == result.ObjectKey {
			return append([]byte(nil), put.Content...), nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (s *fakeLocalArchiveStore) CleanupBefore(_ context.Context, req LocalArchiveCleanupRequest) (LocalArchiveCleanupSummary, error) {
	s.cleanupReqs = append(s.cleanupReqs, req)
	summary := s.cleanupSummary
	summary.Bucket = s.bucket
	summary.Prefix = req.Prefix
	summary.Before = req.Before
	return summary, s.cleanupErr
}

func newFakeRepository() *fakeRepository {
	catalog := NewDefaultCatalog()
	return &fakeRepository{
		fileProfiles:      catalog.FileProfiles(),
		inventoryProfiles: catalog.InventoryProfiles(),
		deliveryTargets:   defaultDeliveryTargets(),
		snmpTargets:       defaultSNMPAlarmTargets(),
		socketConfigs:     defaultSocketAlarmConfigs(),
		apiConfigs:        defaultAPIConfigs(),
		apiTokens:         map[string]string{},
		deviceRows: []ExportDataRow{{
			"device.serial_number":         "SN0001",
			"device.manufacturer":          "Baicells",
			"device.model_name":            "Nova",
			"device.product_class":         "pBS11004",
			"device.firmware_version":      "BaiBS_RTS_1.0",
			"device.ip_address":            "10.0.0.1",
			"device.is_online":             "true",
			"device_info.hardware_version": "HW1",
			"device_info.mac":              "00:11:22:33:44:55",
			"device_info.plmn":             "46000",
			"device_info.sync_status":      "synced",
			"device_info.enb_id":           "100001",
			"device_info.eci":              "25600257",
			"device_info.cell_id":          "1",
			"device_info.bandwidth":        "20",
			"device_info.op_state":         "1",
			"device_info.ue_count":         "7",
			"device_info.device_name":      "Nova-001",
			"device.site_name":             "Site-A",
			"device.last_inform_at":        "2026-08-04 16:45:00+08",
		}},
		pmRows: []ExportDataRow{{
			"pm.device_sn":    "SN0001",
			"pm.metric_path":  "InternetGatewayDevice.Services.FAPService.1.PerfMgmt.PM.Counter.PUSCHPRBUsage",
			"pm.metric_type":  "counter",
			"pm.metric_value": "12",
			"pm.statis_type":  "avg",
			"pm.granularity":  "15min",
			"pm.end_time":     "2026-08-04 16:45:00+08",
			"pm.object_ldn":   "",
		}},
		mrRows: []ExportDataRow{{
			"mr.file_type":  "MRO",
			"mr.device_sn":  "SN0001",
			"mr.file_name":  "source-mro.xml",
			"mr.minio_path": "mr/source-mro.xml",
		}},
		logRows: []ExportDataRow{{
			"log.username":   "admin",
			"log.login_time": "2026-08-04 16:45:00+08",
		}},
	}
}

func (r *fakeRepository) EnsureDefaults(context.Context, *Catalog) error {
	return nil
}

func (r *fakeRepository) EnsureExtendedDefaults(context.Context) error {
	return nil
}

func (r *fakeRepository) ListFileProfiles(context.Context) ([]FileProfile, error) {
	return r.fileProfiles, nil
}

func (r *fakeRepository) CreateFileProfile(_ context.Context, profile FileProfile) (*FileProfile, error) {
	for _, current := range r.fileProfiles {
		if current.Code == profile.Code || current.ID == profile.Code {
			return nil, fmt.Errorf("%w: file profile code %s already exists", commonerrors.ErrInvalidInput, profile.Code)
		}
	}
	r.fileProfiles = append(r.fileProfiles, profile)
	return &r.fileProfiles[len(r.fileProfiles)-1], nil
}

func (r *fakeRepository) UpdateFileProfile(_ context.Context, idOrCode string, req UpdateFileProfileRequest) (*FileProfile, error) {
	for i := range r.fileProfiles {
		if r.fileProfiles[i].Code == idOrCode || r.fileProfiles[i].ID == idOrCode {
			r.fileProfiles[i] = mergeFileProfile(r.fileProfiles[i], req)
			return &r.fileProfiles[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) ListInventoryProfiles(context.Context) ([]InventoryProfile, error) {
	return r.inventoryProfiles, nil
}

func (r *fakeRepository) UpdateInventoryProfile(_ context.Context, idOrCode string, req UpdateInventoryProfileRequest) (*InventoryProfile, error) {
	for i := range r.inventoryProfiles {
		if r.inventoryProfiles[i].Code == idOrCode || r.inventoryProfiles[i].ID == idOrCode {
			r.inventoryProfiles[i] = mergeInventoryProfile(r.inventoryProfiles[i], req)
			return &r.inventoryProfiles[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) CreateFileRun(_ context.Context, run FileRun) (*FileRun, error) {
	now := time.Now()
	if run.ID == "" {
		run.ID = fmt.Sprintf("run-%d", len(r.runs)+1)
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = now
	}
	r.runs = append(r.runs, run)
	return &r.runs[len(r.runs)-1], nil
}

func (r *fakeRepository) ListFileRuns(_ context.Context, filter RunFilter) (RunListResult, error) {
	out := make([]FileRun, 0, len(r.runs))
	for _, run := range r.runs {
		if filter.ProfileKind != "" && run.ProfileKind != filter.ProfileKind {
			continue
		}
		if filter.ProfileCode != "" && run.ProfileCode != filter.ProfileCode {
			continue
		}
		if filter.Status != "" && run.Status != filter.Status {
			continue
		}
		run.ArtifactContent = ""
		out = append(out, run)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if filter.LatestPerProfile {
		seen := make(map[string]struct{}, len(out))
		latest := make([]FileRun, 0, len(out))
		for _, run := range out {
			key := strings.TrimSpace(run.ProfileCode)
			if key == "" {
				key = run.ID
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			latest = append(latest, run)
		}
		out = latest
	}
	total := len(out)
	offset := normalizeOffset(filter.Offset)
	limit := normalizeLimit(filter.Limit)
	if offset > len(out) {
		offset = len(out)
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return RunListResult{Items: out[offset:end], Total: total, Limit: limit, Offset: offset}, nil
}

func (r *fakeRepository) GetFileRun(_ context.Context, id string) (*FileRun, error) {
	for i := range r.runs {
		if r.runs[i].ID == id {
			return &r.runs[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) UpdateFileRunLocalArchive(_ context.Context, id string, result LocalArchiveResult, clearContent bool) (*FileRun, error) {
	for i := range r.runs {
		if r.runs[i].ID != id {
			continue
		}
		if r.runs[i].Summary == nil {
			r.runs[i].Summary = map[string]any{}
		}
		r.runs[i].Summary["local_archive"] = map[string]any{
			"bucket":       result.Bucket,
			"object_key":   result.ObjectKey,
			"object_name":  result.ObjectName,
			"bytes":        result.Bytes,
			"content_type": result.ContentType,
		}
		if clearContent {
			r.runs[i].ArtifactContent = ""
		}
		r.runs[i].UpdatedAt = time.Now()
		return &r.runs[i], nil
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) LoadDeviceSnapshotRows(context.Context, string, int) ([]ExportDataRow, error) {
	return r.deviceRows, nil
}

func (r *fakeRepository) LoadOMCInventoryRows(context.Context) ([]ExportDataRow, error) {
	return []ExportDataRow{{
		"inventory.omc.enb_online": "1",
		"inventory.omc.ue_count":   "7",
		"inventory.omc.version":    "xomc",
	}}, nil
}

func (r *fakeRepository) LoadPMMetricRows(context.Context, PMMetricQuery) ([]ExportDataRow, error) {
	return r.pmRows, nil
}

func (r *fakeRepository) LoadMRRows(context.Context, string, int) ([]ExportDataRow, error) {
	return r.mrRows, nil
}

func (r *fakeRepository) LoadLogRows(context.Context, string, time.Time, time.Time, int) ([]ExportDataRow, error) {
	return r.logRows, nil
}

func (r *fakeRepository) ListPMMetricFields(context.Context, FieldFilter) ([]FieldDefinition, error) {
	return []FieldDefinition{{
		Key:           "pm.pc.lte.C001",
		Domain:        DomainPM,
		ObjectCode:    "PC",
		Tech:          "LTE",
		OutputAlias:   "PUSCHPRBUsage",
		SystemField:   "C001",
		Source:        "perf_indicators_enb.id -> pm_metrics.metric_path",
		DataType:      "number",
		Renderer:      "number",
		MetricType:    "counter",
		StatisType:    "avg",
		Unit:          "%",
		CnName:        "PUSCH PRB 使用率",
		SupportStatus: SupportSupported,
	}}, nil
}

func (r *fakeRepository) ListDeviceInfoFields(_ context.Context, filter FieldFilter) ([]FieldDefinition, error) {
	fields := make([]FieldDefinition, 0, 3)
	for _, column := range []string{"device_name", "project_status", "last_offline_time"} {
		if field, ok := optionalDeviceInfoFieldFromColumn(filter.Domain, filter.ObjectCode, column, "text"); ok {
			fields = append(fields, field)
		}
	}
	return fields, nil
}

func (r *fakeRepository) ValidatePMMetricPaths(_ context.Context, metricPaths []string) ([]string, error) {
	missing := make([]string, 0)
	for _, path := range normalizeMetricPaths(metricPaths) {
		if path != "C001" && path != "InternetGatewayDevice.Services.FAPService.1.PerfMgmt.PM.Counter.PUSCHPRBUsage" {
			missing = append(missing, path)
		}
	}
	return missing, nil
}

func (r *fakeRepository) ListDeliveryTargets(_ context.Context, filter DeliveryTargetFilter) ([]DeliveryTarget, error) {
	out := make([]DeliveryTarget, 0, len(r.deliveryTargets))
	for _, target := range r.deliveryTargets {
		if filter.Scope != "" && target.Scope != filter.Scope {
			continue
		}
		if filter.OwnerCode != "" && target.OwnerCode != filter.OwnerCode {
			continue
		}
		target.Credential = ""
		out = append(out, target)
	}
	return out, nil
}

func (r *fakeRepository) GetDeliveryTargetForSend(_ context.Context, scope DeliveryScope, ownerCode string, key string) (*DeliveryTarget, error) {
	for _, target := range r.deliveryTargets {
		if target.Scope == scope && target.OwnerCode == ownerCode && target.Key == key {
			return &target, nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) ListActiveDeliveryTargets(_ context.Context, scope DeliveryScope, ownerCode string) ([]DeliveryTarget, error) {
	out := make([]DeliveryTarget, 0, len(r.deliveryTargets))
	for _, target := range r.deliveryTargets {
		if target.Scope != scope || target.OwnerCode != ownerCode || !target.Enabled {
			continue
		}
		out = append(out, target)
	}
	return out, nil
}

func (r *fakeRepository) ReplaceDeliveryTargets(_ context.Context, req ReplaceDeliveryTargetsRequest) ([]DeliveryTarget, error) {
	next := r.deliveryTargets[:0]
	existingSecrets := map[string]string{}
	for _, target := range r.deliveryTargets {
		if target.Scope == req.Scope && target.OwnerCode == req.OwnerCode {
			existingSecrets[target.Key] = target.Credential
			continue
		}
		next = append(next, target)
	}
	for _, target := range req.Items {
		target.Scope = req.Scope
		target.OwnerCode = req.OwnerCode
		if strings.TrimSpace(target.Credential) == "" && target.CredentialSet {
			target.Credential = existingSecrets[target.Key]
		}
		target.CredentialSet = strings.TrimSpace(target.Credential) != ""
		next = append(next, target)
	}
	r.deliveryTargets = next
	return r.ListDeliveryTargets(context.Background(), DeliveryTargetFilter{Scope: req.Scope, OwnerCode: req.OwnerCode})
}

func TestDeliveryTargetsForRunUsesTemplateOwnerOnly(t *testing.T) {
	repo := &fakeRepository{
		deliveryTargets: []DeliveryTarget{
			{Scope: DeliveryScopeFile, OwnerCode: "", Key: "global", Name: "global target", Enabled: true},
			{Scope: DeliveryScopeFile, OwnerCode: "S0001", Key: "s0001", Name: "S0001 target", Enabled: true},
			{Scope: DeliveryScopeFile, OwnerCode: "S0002", Key: "s0002", Name: "S0002 target", Enabled: true},
			{Scope: DeliveryScopeFile, OwnerCode: "S0001", Key: "s0001-disabled", Name: "disabled", Enabled: false},
		},
	}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	targets, err := svc.deliveryTargetsForRun(context.Background(), FileRun{
		ProfileKind: ProfileKindFile,
		ProfileCode: "S0001",
	})
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "s0001", targets[0].Key)

	targets, err = svc.deliveryTargetsForRun(context.Background(), FileRun{
		ProfileKind: ProfileKindFile,
		ProfileCode: "S0003",
	})
	require.NoError(t, err)
	require.Empty(t, targets)
}

func (r *fakeRepository) ListSNMPAlarmTargets(context.Context) ([]SNMPAlarmTarget, error) {
	out := make([]SNMPAlarmTarget, len(r.snmpTargets))
	copy(out, r.snmpTargets)
	return out, nil
}

func (r *fakeRepository) GetSNMPAlarmTargetForSend(_ context.Context, key string) (*SNMPAlarmTarget, error) {
	for i := range r.snmpTargets {
		if strings.EqualFold(r.snmpTargets[i].Key, key) {
			return &r.snmpTargets[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) ListActiveSNMPAlarmTargetsForSend(context.Context) ([]SNMPAlarmTarget, error) {
	out := make([]SNMPAlarmTarget, 0, len(r.snmpTargets))
	for _, target := range r.snmpTargets {
		if target.Enabled {
			out = append(out, target)
		}
	}
	return out, nil
}

func (r *fakeRepository) ListSNMPMIBTargetsForServe(context.Context) ([]SNMPAlarmTarget, error) {
	out := make([]SNMPAlarmTarget, 0, len(r.snmpTargets))
	for _, target := range r.snmpTargets {
		if target.MIBQueryEnabled {
			out = append(out, target)
		}
	}
	return out, nil
}

func (r *fakeRepository) UpdateSNMPAlarmTarget(_ context.Context, key string, target SNMPAlarmTarget) (*SNMPAlarmTarget, error) {
	target = normalizeSNMPAlarmTarget(key, target)
	for i := range r.snmpTargets {
		if r.snmpTargets[i].Key == key {
			r.snmpTargets[i] = target
			return &r.snmpTargets[i], nil
		}
	}
	r.snmpTargets = append(r.snmpTargets, target)
	return &r.snmpTargets[len(r.snmpTargets)-1], nil
}

func (r *fakeRepository) ListSocketAlarmConfigs(context.Context) ([]SocketAlarmConfig, error) {
	out := make([]SocketAlarmConfig, len(r.socketConfigs))
	copy(out, r.socketConfigs)
	return out, nil
}

func (r *fakeRepository) ListActiveSocketAlarmConfigsForServe(context.Context) ([]SocketAlarmConfig, error) {
	out := make([]SocketAlarmConfig, 0, len(r.socketConfigs))
	for _, config := range r.socketConfigs {
		if config.Enabled && config.Mode == "server" {
			out = append(out, config)
		}
	}
	return out, nil
}

func (r *fakeRepository) UpdateSocketAlarmConfig(_ context.Context, key string, config SocketAlarmConfig) (*SocketAlarmConfig, error) {
	config = normalizeSocketAlarmConfig(key, config)
	for i := range r.socketConfigs {
		if r.socketConfigs[i].Key == key {
			r.socketConfigs[i] = config
			return &r.socketConfigs[i], nil
		}
	}
	r.socketConfigs = append(r.socketConfigs, config)
	return &r.socketConfigs[len(r.socketConfigs)-1], nil
}

func (r *fakeRepository) ListAPIConfigs(context.Context) ([]APIConfig, error) {
	out := make([]APIConfig, len(r.apiConfigs))
	copy(out, r.apiConfigs)
	return out, nil
}

func (r *fakeRepository) UpdateAPIConfig(_ context.Context, key string, enabled bool) (*APIConfig, error) {
	for i := range r.apiConfigs {
		if r.apiConfigs[i].Key == key {
			r.apiConfigs[i].Enabled = enabled
			return &r.apiConfigs[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) ListAPIClients(context.Context) ([]APIClient, error) {
	out := make([]APIClient, len(r.apiClients))
	copy(out, r.apiClients)
	for i := range out {
		out[i].TokenSecret = ""
	}
	return out, nil
}

func (r *fakeRepository) ReplaceAPIClients(_ context.Context, req ReplaceAPIClientsRequest) ([]APIClient, error) {
	next := make([]APIClient, 0, len(req.Items))
	for _, client := range req.Items {
		client = normalizeAPIClient(client)
		client.TokenSet = strings.TrimSpace(client.TokenSecret) != ""
		client.TokenSecret = ""
		next = append(next, client)
	}
	r.apiClients = next
	return r.ListAPIClients(context.Background())
}

func (r *fakeRepository) ListAPIUsers(context.Context) ([]APIUser, error) {
	out := make([]APIUser, 0, len(r.apiClients))
	for _, client := range r.apiClients {
		out = append(out, APIUser{
			ID:          client.ID,
			Username:    client.ClientKey,
			Enabled:     client.Enabled,
			Password:    client.TokenSecret,
			PasswordSet: strings.TrimSpace(client.TokenSecret) != "" || client.TokenSet,
			CreatedAt:   client.CreatedAt,
			UpdatedAt:   client.UpdatedAt,
		})
	}
	return out, nil
}

func (r *fakeRepository) ReplaceAPIUsers(_ context.Context, req ReplaceAPIUsersRequest) ([]APIUser, error) {
	next := make([]APIClient, 0, len(req.Items))
	currentPasswords := map[string]string{}
	for _, client := range r.apiClients {
		currentPasswords[client.ClientKey] = client.TokenSecret
	}
	for _, user := range req.Items {
		user = normalizeAPIUser(user)
		password := credentialToPersist(user.Password, currentPasswords[user.Username])
		if err := validateAPIUser(APIUser{Username: user.Username, Enabled: user.Enabled, Password: password}); err != nil {
			return nil, err
		}
		next = append(next, APIClient{
			ClientKey:   user.Username,
			Name:        user.Username,
			Enabled:     user.Enabled,
			TokenSecret: password,
			TokenSet:    strings.TrimSpace(password) != "",
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
		})
	}
	r.apiClients = next
	return r.ListAPIUsers(context.Background())
}

func (r *fakeRepository) CreateAPIUser(_ context.Context, user APIUser) (*APIUser, error) {
	user = normalizeAPIUser(user)
	if err := validateAPIUser(user); err != nil {
		return nil, err
	}
	for _, client := range r.apiClients {
		if strings.EqualFold(client.ClientKey, user.Username) {
			return nil, fmt.Errorf("%w: USER_EXIST", commonerrors.ErrInvalidInput)
		}
	}
	if user.ID == "" {
		user.ID = fmt.Sprintf("api-user-%d", len(r.apiClients)+1)
	}
	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}
	r.apiClients = append(r.apiClients, APIClient{
		ID:          user.ID,
		ClientKey:   user.Username,
		Name:        user.Username,
		Enabled:     user.Enabled,
		TokenSecret: user.Password,
		TokenSet:    strings.TrimSpace(user.Password) != "",
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	})
	created := APIUser{
		ID:          user.ID,
		Username:    user.Username,
		Enabled:     user.Enabled,
		Password:    user.Password,
		PasswordSet: strings.TrimSpace(user.Password) != "",
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
	return &created, nil
}

func (r *fakeRepository) UpdateAPIUser(_ context.Context, idOrUsername string, req UpdateAPIUserRequest) (*APIUser, error) {
	idOrUsername = strings.TrimSpace(idOrUsername)
	for i := range r.apiClients {
		client := &r.apiClients[i]
		if client.ID != idOrUsername && client.ClientKey != idOrUsername {
			continue
		}
		nextUsername := client.ClientKey
		if strings.TrimSpace(req.Username) != "" {
			nextUsername = strings.TrimSpace(req.Username)
		}
		for j := range r.apiClients {
			if i != j && strings.EqualFold(r.apiClients[j].ClientKey, nextUsername) {
				return nil, fmt.Errorf("%w: USER_EXIST", commonerrors.ErrInvalidInput)
			}
		}
		nextEnabled := client.Enabled
		if req.Enabled != nil {
			nextEnabled = *req.Enabled
		}
		password := credentialToPersist(req.Password, client.TokenSecret)
		if err := validateAPIUser(APIUser{Username: nextUsername, Enabled: nextEnabled, Password: password}); err != nil {
			return nil, err
		}
		client.ClientKey = nextUsername
		client.Name = nextUsername
		client.Enabled = nextEnabled
		client.TokenSecret = password
		client.TokenSet = strings.TrimSpace(password) != ""
		client.UpdatedAt = time.Now()
		return &APIUser{
			ID:          client.ID,
			Username:    client.ClientKey,
			Enabled:     client.Enabled,
			Password:    client.TokenSecret,
			PasswordSet: client.TokenSet,
			CreatedAt:   client.CreatedAt,
			UpdatedAt:   client.UpdatedAt,
		}, nil
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) DeleteAPIUser(_ context.Context, idOrUsername string) error {
	idOrUsername = strings.TrimSpace(idOrUsername)
	for i := range r.apiClients {
		if r.apiClients[i].ID == idOrUsername || r.apiClients[i].ClientKey == idOrUsername {
			r.apiClients = append(r.apiClients[:i], r.apiClients[i+1:]...)
			return nil
		}
	}
	return commonerrors.ErrNotFound
}

func (r *fakeRepository) LoginAPIUser(_ context.Context, req APIUserLoginRequest) (*APIUserToken, error) {
	for _, client := range r.apiClients {
		if client.ClientKey != strings.TrimSpace(req.Username) {
			continue
		}
		if !client.Enabled {
			return nil, commonerrors.ErrUnauthorized
		}
		if client.TokenSecret != strings.TrimSpace(req.Password) {
			return nil, commonerrors.ErrUnauthorized
		}
		token := "test-token-" + client.ClientKey
		if r.apiTokens == nil {
			r.apiTokens = map[string]string{}
		}
		r.apiTokens[token] = client.ClientKey
		expiresAt := time.Now().Add(northboundAPITokenTTL)
		return &APIUserToken{
			Token:       token,
			AccessToken: token,
			Expires:     int(northboundAPITokenTTL.Seconds()),
			ExpiresAt:   expiresAt,
			TokenType:   "Bearer",
		}, nil
	}
	return nil, commonerrors.ErrUnauthorized
}

func (r *fakeRepository) CreateAPIInvocationLog(_ context.Context, item APIInvocationLog) error {
	if item.ID == "" {
		item.ID = fmt.Sprintf("api-log-%d", len(r.apiInvocationLogs)+1)
	}
	now := time.Now()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}
	r.apiInvocationLogs = append(r.apiInvocationLogs, item)
	return nil
}

func (r *fakeRepository) ListAPIInvocationLogs(_ context.Context, filter APIInvocationLogFilter) (APIInvocationLogListResult, error) {
	start, hasStart := parseAPIInvocationLogTime(filter.StartTime)
	end, hasEnd := parseAPIInvocationLogTime(filter.EndTime)
	out := make([]APIInvocationLog, 0, len(r.apiInvocationLogs))
	for _, item := range r.apiInvocationLogs {
		if filter.APIKey != "" && item.APIKey != filter.APIKey {
			continue
		}
		if filter.Name != "" && !containsFold(item.Name, filter.Name) {
			continue
		}
		if filter.Method != "" && !strings.EqualFold(item.Method, filter.Method) {
			continue
		}
		if filter.Path != "" && !containsFold(item.Path, filter.Path) {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.CreateUser != "" && item.CreateUser != filter.CreateUser {
			continue
		}
		if filter.IPAddress != "" && item.IPAddress != filter.IPAddress {
			continue
		}
		if hasStart && item.CreatedAt.Before(start) {
			continue
		}
		if hasEnd && item.CreatedAt.After(end) {
			continue
		}
		if filter.Keyword != "" && !apiInvocationLogMatchesKeyword(item, filter.Keyword) {
			continue
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	total := len(out)
	limit := normalizeLimit(filter.Limit)
	offset := normalizeOffset(filter.Offset)
	if offset > len(out) {
		out = []APIInvocationLog{}
	} else {
		endOffset := offset + limit
		if endOffset > len(out) {
			endOffset = len(out)
		}
		out = out[offset:endOffset]
	}
	return APIInvocationLogListResult{Items: out, Total: total, Limit: limit, Offset: offset}, nil
}

func apiInvocationLogMatchesKeyword(item APIInvocationLog, keyword string) bool {
	return containsFold(item.APIKey, keyword) ||
		containsFold(item.Name, keyword) ||
		containsFold(item.Method, keyword) ||
		containsFold(item.Path, keyword) ||
		containsFold(item.RequestParams, keyword) ||
		containsFold(item.ResponseBody, keyword) ||
		containsFold(item.Status, keyword) ||
		containsFold(item.CreateUser, keyword) ||
		containsFold(item.IPAddress, keyword)
}

func containsFold(value, keyword string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(strings.TrimSpace(keyword)))
}

func (r *fakeRepository) AuthenticateAPIClient(_ context.Context, credential string, remoteIP string, apiKey string) (*APIClient, error) {
	active := 0
	for _, client := range r.apiClients {
		if client.Enabled {
			active++
		}
	}
	if active == 0 {
		return nil, commonerrors.ErrUnauthorized
	}
	username := r.apiTokens[strings.TrimSpace(credential)]
	for i := range r.apiClients {
		client := r.apiClients[i]
		if !client.Enabled || client.ClientKey != username {
			continue
		}
		if !apiClientAllowsAPI(client, apiKey) {
			return nil, commonerrors.ErrForbidden
		}
		if !apiClientAllowsIP(client, remoteIP) {
			return nil, commonerrors.ErrForbidden
		}
		return &client, nil
	}
	return nil, commonerrors.ErrUnauthorized
}

func (r *fakeRepository) CreateEvent(_ context.Context, event PageConfigEvent) (*PageConfigEvent, error) {
	now := time.Now()
	event = normalizeEvent(event)
	if event.ID == "" {
		event.ID = fmt.Sprintf("event-%d", len(r.events)+1)
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	if event.UpdatedAt.IsZero() {
		event.UpdatedAt = now
	}
	r.events = append(r.events, event)
	return &r.events[len(r.events)-1], nil
}

func (r *fakeRepository) ListEvents(_ context.Context, filter EventFilter) (EventListResult, error) {
	out := make([]PageConfigEvent, 0, len(r.events))
	for _, event := range r.events {
		if filter.Capability != "" && event.Capability != filter.Capability {
			continue
		}
		if filter.OwnerCode != "" && event.OwnerCode != filter.OwnerCode {
			continue
		}
		if filter.TargetKey != "" && event.TargetKey != filter.TargetKey {
			continue
		}
		if filter.EventType != "" && event.EventType != filter.EventType {
			continue
		}
		if filter.Status != "" && event.Status != filter.Status {
			continue
		}
		out = append(out, event)
	}
	total := len(out)
	offset := normalizeOffset(filter.Offset)
	limit := normalizeLimit(filter.Limit)
	if offset > len(out) {
		offset = len(out)
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return EventListResult{Items: out[offset:end], Total: total, Limit: limit, Offset: offset}, nil
}

func (r *fakeRepository) GetEvent(_ context.Context, id string) (*PageConfigEvent, error) {
	for i := range r.events {
		if r.events[i].ID == id {
			return &r.events[i], nil
		}
	}
	return nil, commonerrors.ErrNotFound
}

func (r *fakeRepository) PruneEvents(_ context.Context, filter EventFilter, keep int) (int64, error) {
	if keep <= 0 {
		return 0, nil
	}
	matches := make([]int, 0, len(r.events))
	for i, event := range r.events {
		if filter.Capability != "" && event.Capability != filter.Capability {
			continue
		}
		if filter.OwnerCode != "" && event.OwnerCode != filter.OwnerCode {
			continue
		}
		if filter.TargetKey != "" && event.TargetKey != filter.TargetKey {
			continue
		}
		if filter.EventType != "" && event.EventType != filter.EventType {
			continue
		}
		if filter.Status != "" && event.Status != filter.Status {
			continue
		}
		matches = append(matches, i)
	}
	if len(matches) <= keep {
		return 0, nil
	}
	remove := map[int]struct{}{}
	for _, idx := range matches[:len(matches)-keep] {
		remove[idx] = struct{}{}
	}
	kept := r.events[:0]
	for i, event := range r.events {
		if _, ok := remove[i]; ok {
			continue
		}
		kept = append(kept, event)
	}
	deleted := int64(len(r.events) - len(kept))
	r.events = kept
	return deleted, nil
}

func (r *fakeRepository) CleanupExpiredResults(_ context.Context, runBefore time.Time, eventBefore time.Time) (ResultCleanupSummary, error) {
	var summary ResultCleanupSummary
	keptRuns := r.runs[:0]
	for _, run := range r.runs {
		if !runBefore.IsZero() && run.CreatedAt.Before(runBefore) {
			summary.RunsDeleted++
			continue
		}
		keptRuns = append(keptRuns, run)
	}
	r.runs = keptRuns
	keptEvents := r.events[:0]
	for _, event := range r.events {
		if !eventBefore.IsZero() && event.CreatedAt.Before(eventBefore) {
			summary.EventsDeleted++
			continue
		}
		keptEvents = append(keptEvents, event)
	}
	r.events = keptEvents
	summary.RunBefore = runBefore
	summary.EventBefore = eventBefore
	return summary, nil
}

func setupTestRouterWithRepository(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(newTestServiceWithRepository(repo), zap.NewNop())
	h.RegisterRoutes(r.Group("/api/v1/northbound"))
	return r
}

func newTestServiceWithRepository(repo Repository) *Service {
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	svc.SetMRSourceStore(&fakeMRSourceStore{objects: map[string][]byte{
		"mr/source-mro.xml": []byte(`<bulkPmMrDataFile><fileHeader/></bulkPmMrDataFile>`),
	}})
	return svc
}

func TestListFileProfilesReturnsScenarioSeeds(t *testing.T) {
	r := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/file/profiles", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Ret  int `json:"ret"`
		Data struct {
			Items []FileProfile `json:"items"`
			Total int           `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, 1, body.Ret)
	require.Equal(t, 17, body.Data.Total)
	require.False(t, body.Data.Items[0].Enabled)
	require.Equal(t, StatusTerminated, body.Data.Items[0].Status)
}

func TestFieldsCanFilterByDomainAndObject(t *testing.T) {
	r := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/fields?domain=CM&object=CP", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Data struct {
			Items []FieldDefinition `json:"items"`
			Total int               `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Greater(t, body.Data.Total, 0)
	for _, item := range body.Data.Items {
		require.Equal(t, DomainCM, item.Domain)
		require.Equal(t, "CP", item.ObjectCode)
	}
}

func TestPMFieldsComeFromRepositoryWhenConfigured(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/fields?domain=PM&object=PC&tech=LTE", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"system_field":"C001"`)
	require.Contains(t, rr.Body.String(), `perf_indicators_enb.id`)
	require.Contains(t, rr.Body.String(), `pm_metrics.metric_path`)
}

func TestDeviceInfoFieldsComeFromRepositoryWhenConfigured(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/fields?domain=CM&object=CP", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"system_field":"device_info.project_status"`)
}

func TestInventoryDeviceInfoFieldsComeFromRepositoryWhenConfigured(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/fields?domain=INVENTORY&object=gNB", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"domain":"INVENTORY"`)
	require.Contains(t, rr.Body.String(), `"system_field":"device_info.project_status"`)
}

func TestPreviewFileProfileRendersTemplates(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/file/profiles/S0001/preview", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"profile_code":"S0001"`)
	require.Contains(t, rr.Body.String(), `"preview_path":"/northupload/GD/BaiOMC/CM/20260804000000/"`)
	require.Contains(t, rr.Body.String(), `"preview_artifact_name":"Baicells-CP-127.0.0.1-1.0-20260804000000.xml.zip"`)
}

func TestRenderCustomLogRowsMatchesLegacyHeaders(t *testing.T) {
	rows := []ExportDataRow{{
		"log.log_time":       "2026-08-07 15:30:21",
		"log.login_time":     "2026-08-07 15:30:21",
		"log.operation_time": "2026-08-07 15:30:21",
		"log.account_name":   "admin",
		"log.terminal_name":  "Browser",
		"log.terminal_ip":    "10.10.10.100",
		"log.result":         "success",
		"log.main_name":      "admin",
		"log.log_name":       "Update",
		"log.detail":         "changed config",
	}}

	loginContent, count, err := renderLogRows("login", rows)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Contains(t, loginContent, "log_time|sys_source_name|account_name|terminal_name|terminal_ip|log_result|log_start_time|log_end_time\n")
	require.Contains(t, loginContent, "2026-08-07 15:30:21|baicells omc|admin|Browser|10.10.10.100|success|2026-08-07 15:30:21|2026-08-07 15:30:21\n")

	operationContent, count, err := renderLogRows("operation", rows)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Contains(t, operationContent, "log_time|sys_source_name|terminal_name|terminal_ip|main_name|sub_account|asset_name|asset_ip|asset_port|asset_attribute|log_data\n")
	require.Contains(t, operationContent, "\"{\"Update\",\"changed config\",\"success\"}\"")
}

func TestRenderFixedLogRowsQuotesAllFields(t *testing.T) {
	rows := []ExportDataRow{{
		"log.id":             "10234",
		"log.user_name":      "admin",
		"log.ip_address":     "10.10.10.100",
		"log.log_name":       "LoginLogout",
		"log.detail":         "login success",
		"log.result_text":    "Success",
		"log.failure_reason": " ",
		"log.login_time":     "2026-08-07 15:30:21",
		"log.op_start_time":  "2026-08-07 15:30:21",
		"log.op_end_time":    "2026-08-07 15:30:22",
	}}

	securityContent, count, err := renderLogRows("login_fix", rows)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, "\"ID\",\"User Name\",\"IP Address\",\"Log Name\",\"Record Detail\",\"Results\",\"Failure Reason\",\"Time\"\n\"10234\",\"admin\",\"10.10.10.100\",\"LoginLogout\",\"login success\",\"Success\",\" \",\"2026-08-07 15:30:21\t\"\n", securityContent)

	operationContent, count, err := renderLogRows("operation_fix", rows)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, "\"User Name\",\"IP Address\",\"Log Name\",\"Record Detail\",\"Results\",\"Failure Reason\",\"Op Start Time\",\"Op End Time\"\n\"admin\",\"10.10.10.100\",\"LoginLogout\",\"login success\",\"Success\",\" \",\"2026-08-07 15:30:21\t\",\"2026-08-07 15:30:22\t\"\n", operationContent)
}

func TestRenderLogArtifactNameMatchesLegacyFiles(t *testing.T) {
	group := FileGroup{
		Domain:             DomainLOG,
		Format:             FormatCSV,
		Period:             Period24H,
		PathTemplate:       pathLOG,
		FileNameTemplate:   "#Object#_#PeriodStartTime#-24H.csv",
		CompressionEnabled: true,
		CompressionFormat:  CompressionGz,
	}
	windowStart := time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	path, name := renderArtifactName(group, ScenarioObject{Code: "login_fix"}, windowStart, windowEnd, 1)
	require.Equal(t, "/northupload/LOGS/20260807/", path)
	require.Equal(t, "SecurityLogs_20260807000000-24H.csv.gz", name)

	group.Format = FormatTXT
	path, name = renderArtifactName(group, ScenarioObject{Code: "operation"}, windowStart, windowEnd, 1)
	require.Equal(t, "/northupload/LOGS/20260807/", path)
	require.Equal(t, "oper_20260807.txt.gz", name)
}

func TestValidateRejectsInventoryAsFileProfile(t *testing.T) {
	r := setupTestRouter()
	body := []byte(`{"profile_kind":"file","domain":"INVENTORY","format":"CSV","period":"24H","objects":[{"code":"ENB"}]}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "inventory must use inventory profiles")
}

func TestValidateRejectsEGWObjects(t *testing.T) {
	r := setupTestRouter()
	body := []byte(`{"profile_kind":"file","domain":"CM","format":"CSV","period":"24H","objects":[{"code":"EGW"}]}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "EGW/PEGW objects are not supported")
}

func TestValidateAcceptsCMCSVAndXMLProfile(t *testing.T) {
	r := setupTestRouter()

	for _, format := range []string{"CSV", "XML"} {
		body := []byte(fmt.Sprintf(`{"profile_kind":"file","domain":"CM","format":"%s","period":"24H","compression_enabled":true,"compression_format":"zip","objects":[{"code":"CP"}]}`, format))

		req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/validate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		require.Equal(t, http.StatusOK, rr.Code)
		require.Contains(t, rr.Body.String(), `"valid":true`)
	}
}

func TestValidateRejectsUnknownPMMetricPath(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{"profile_kind":"file","domain":"PM","format":"CSV","period":"15M","objects":[{"code":"PC"}],"metric_paths":["C001","C404"]}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), `unknown PM metric_path \"C404\"`)
	require.NotContains(t, rr.Body.String(), "pending PM indicator")
}

func TestUpdateFileProfilePersistsThroughRepository(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{"enabled":true,"status":"normal","name":"enabled scenario"}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/file/profiles/S0001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"enabled":true`)
	require.Contains(t, rr.Body.String(), `"status":"normal"`)
	require.Contains(t, rr.Body.String(), `"name":"enabled scenario"`)
}

func TestUpdateFileProfileRejectsMixedTechnologyGroup(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{
  "groups": [
    {
      "id": "pm-mixed",
      "domain": "PM",
      "format": "CSV",
      "period": "15M",
      "start_minute": 5,
      "path_template": "/#FTPRoot#/#Province#/#OMC-R#/PM/#DateTime#/",
      "file_name_template": "Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]",
      "compression_enabled": true,
      "compression_format": "zip",
      "objects": [
        {"code": "PC", "tech": "LTE"},
        {"code": "PC", "tech": "GNB"}
      ]
    }
  ]
}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/file/profiles/S0001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "mixes multiple technologies")
	require.Contains(t, rr.Body.String(), "one group row per ENB/GNB/GSM")
}

func TestUpdateFileProfileAcceptsSplitTechnologyGroups(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{
  "groups": [
    {
      "id": "pm-enb",
      "domain": "PM",
      "format": "CSV",
      "period": "15M",
      "start_minute": 5,
      "path_template": "/#FTPRoot#/#Province#/#OMC-R#/PM/#DateTime#/",
      "file_name_template": "Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]",
      "compression_enabled": true,
      "compression_format": "zip",
      "objects": [{"code": "PC", "tech": "LTE"}]
    },
    {
      "id": "pm-gnb",
      "domain": "PM",
      "format": "CSV",
      "period": "15M",
      "start_minute": 8,
      "path_template": "/#FTPRoot#/#Province#/#OMC-R#/PM/GNB/#DateTime#/",
      "file_name_template": "Baicells-#Object#-#LocalHost#-#DataVersion#-#DateTime#[-#Ri#]-#DataPeriod#[-#FileID#]",
      "compression_enabled": true,
      "compression_format": "zip",
      "objects": [{"code": "PC", "tech": "GNB"}]
    }
  ]
}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/file/profiles/S0001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"id":"pm-enb"`)
	require.Contains(t, rr.Body.String(), `"id":"pm-gnb"`)
	require.Contains(t, rr.Body.String(), `"tech":"LTE"`)
	require.Contains(t, rr.Body.String(), `"tech":"GNB"`)
}

func TestCreateFileProfilePersistsThroughRepository(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{
  "id": "S9001",
  "code": "S9001",
  "name": "自定义北向文件配置",
  "vendor": "Baicells",
  "scenario_name": "自定义场景",
  "scenario_name_en": "Custom Scenario",
  "description": "created by page config",
  "flags": ["custom"],
  "enabled": true,
  "status": "normal",
  "groups": [
    {
      "id": "cm-24h",
      "domain": "CM",
      "format": "CSV",
      "period": "24H",
      "start_minute": 5,
      "path_template": "/data/northbound/{YYYYMMDD}/cm",
      "file_name_template": "cm_{YYYYMMDD}.csv",
      "compression_enabled": false,
      "objects": [{"code": "CP"}]
    }
  ]
}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/file/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"code":"S9001"`)
	require.Contains(t, rr.Body.String(), `"enabled":true`)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/file/profiles", nil)
	listRR := httptest.NewRecorder()
	r.ServeHTTP(listRR, listReq)

	require.Equal(t, http.StatusOK, listRR.Code)
	require.Contains(t, listRR.Body.String(), `"code":"S9001"`)
}

func TestCreateFileProfileRejectsInvalidCode(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{"code":"CUSTOM","name":"bad"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/file/profiles", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "file profile code must match S0000 format")
}

func TestUpdateFileProfileRejectsUnsafePathTemplate(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{
  "groups": [
    {
      "id": "bad-cm",
      "domain": "CM",
      "format": "CSV",
      "period": "24H",
      "start_minute": 1,
      "path_template": "../bad",
      "file_name_template": "bad.csv",
      "compression_enabled": false,
      "objects": [{"code": "CP"}]
    }
  ]
}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/file/profiles/S0001", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "path template must be an absolute path")
}

func TestUpdateInventoryEnabledDoesNotResetStartMinute(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)
	body := []byte(`{"enabled":true,"status":"normal"}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/inventory/profiles/GNB", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"enabled":true`)
	require.Contains(t, rr.Body.String(), `"start_minute":5`)
}

func TestUpdateInventoryFieldsPersistsAndControlsOutput(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)
	body := []byte(`{
  "fields": [
    {"key":"enb-1","output_alias":"SN","system_field":"device.serial_number","source":"devices.serial_number","data_type":"string","renderer":"quote","enabled":true},
    {"key":"enb-2","output_alias":"Online","system_field":"device.is_online","source":"devices.is_online","data_type":"bool","renderer":"enum","enabled":false}
  ]
}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/inventory/profiles/ENB", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"output_alias":"SN"`)
	require.Contains(t, rr.Body.String(), `"enabled":false`)

	run, err := NewServiceWithRepository(NewDefaultCatalog(), repo).RunInventoryProfile(context.Background(), "ENB", RunProfileRequest{Limit: 1})
	require.NoError(t, err)
	require.Equal(t, "SN", strings.Split(strings.TrimSpace(run.ArtifactContent), "\n")[0])
}

func TestRunFileProfileCreatesRuns(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)
	body := []byte(`{"group_id":"cm-daily","limit":10}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/file/profiles/S0001/run", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"profile_code":"S0001"`)
	require.Contains(t, rr.Body.String(), `"total":4`)
	require.Len(t, repo.runs, 4)
	require.Len(t, repo.events, 4)
	require.Equal(t, RunStatusSuccess, repo.runs[0].Status)
	require.Contains(t, repo.runs[0].ArtifactContent, "Serial Number")
	require.Contains(t, repo.runs[0].ArtifactName, ".xml.zip")
	require.Equal(t, "file", repo.events[0].Capability)
	require.Equal(t, "run", repo.events[0].EventType)
}

func TestRunInventoryProfileCreatesRun(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/inventory/profiles/ENB/run", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, repo.runs, 1)
	require.Len(t, repo.events, 1)
	require.Equal(t, ProfileKindInventory, repo.runs[0].ProfileKind)
	require.Equal(t, DomainInventory, repo.runs[0].Domain)
	require.Contains(t, repo.runs[0].ArtifactContent, "Serial Number")
	require.Contains(t, repo.runs[0].ArtifactContent, "SN0001")
	require.Equal(t, "inventory", repo.events[0].Capability)
}

func TestRunFileProfileGoldenContentAndName(t *testing.T) {
	repo := newFakeRepository()
	cmGroup := group("cm-golden", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, []ScenarioObject{{Code: "CP", Tech: "LTE"}})
	cmGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9201", "CM golden", "Custom", "Custom", []string{"custom"}, []FileGroup{cmGroup}),
	}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	windowEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)

	resp, err := svc.RunFileProfile(context.Background(), "S9201", RunProfileRequest{
		GroupID:   "cm-golden",
		WindowEnd: &windowEnd,
		Limit:     1,
	})
	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	run := resp.Items[0]
	require.Equal(t, "Baicells-CP-127.0.0.1-1.0-20260804000000.csv", run.ArtifactName)
	require.Equal(t, "/northupload/GD/BaiOMC/CM/20260804000000/", run.ArtifactPath)
	require.Equal(t, strings.Join([]string{
		"Serial Number,dn,related_enb_dn,related_enb_id,related_enb_userlabel,cel_id,cel_userlabel,referenceSignalPower,Manufacturer,Model Name,Product Type,Hardware Version,Software Version,Device Name,Site Name,First Online Time,Last Inform,Device Alias,Management Status,Create Time",
		"SN0001,SN0001,SN0001,100001,Site-A,1,Nova-001,,Baicells,Nova,pBS11004,HW1,BaiBS_RTS_1.0,Nova-001,Site-A,,2026-08-04 16:45:00+08,Site-A,,",
		"",
	}, "\n"), run.ArtifactContent)
}

func TestRunFileProfileHonorsCSVSeparator(t *testing.T) {
	repo := newFakeRepository()
	pmGroup := group("pm-pipe", DomainPM, FormatCSV, Period15M, 5, pathPM, namePM, []ScenarioObject{{Code: "PC", Tech: "LTE"}})
	pmGroup.CSVSeparator = "|"
	pmGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9202", "PM pipe CSV", "Custom", "Custom", []string{"custom", "csv |"}, []FileGroup{pmGroup}),
	}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	windowEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)

	resp, err := svc.RunFileProfile(context.Background(), "S9202", RunProfileRequest{
		GroupID:   "pm-pipe",
		WindowEnd: &windowEnd,
		Limit:     1,
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	run := resp.Items[0]
	lines := strings.Split(strings.TrimSpace(run.ArtifactContent), "\n")
	require.NotEmpty(t, lines)
	require.Equal(t, "Device SN|Metric Path|Metric Type|Metric Value|Statis Type|Granularity|End Time|Object LDN", lines[0])
	require.Contains(t, lines[1], "SN0001|InternetGatewayDevice.Services.FAPService.1.PerfMgmt.PM.Counter.PUSCHPRBUsage|counter|12|avg|15min|2026-08-04 16:45:00+08|")
}

func TestCSVCommaAcceptsTabAliases(t *testing.T) {
	for _, separator := range []string{"\t", `\t`, "tab", "Tab"} {
		comma, ok := csvComma(separator)
		require.True(t, ok)
		require.Equal(t, '\t', comma)
	}
}

func TestRunFileProfileUsesMRSourceObjectContent(t *testing.T) {
	repo := newFakeRepository()
	mrGroup := group("mr-source", DomainMR, FormatXML, Period15M, 0, pathMR, nameMR, []ScenarioObject{{Code: "MRO"}})
	mrGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9203", "MR source", "Custom", "Custom", []string{"custom"}, []FileGroup{mrGroup}),
	}
	svc := newTestServiceWithRepository(repo)
	windowEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)

	resp, err := svc.RunFileProfile(context.Background(), "S9203", RunProfileRequest{
		GroupID:   "mr-source",
		WindowEnd: &windowEnd,
		Limit:     1,
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	run := resp.Items[0]
	require.Equal(t, "MRO-Baicells-MRO-127.0.0.1-100001-20260804000000.xml", run.ArtifactName)
	require.Equal(t, `<bulkPmMrDataFile><fileHeader/></bulkPmMrDataFile>`, run.ArtifactContent)
	require.NotContains(t, run.ArtifactContent, "<NorthboundExport>")
	require.Equal(t, 1, run.RowCount)
	require.Equal(t, true, run.Summary["mr_passthrough"])
	require.Equal(t, "source-mro.xml", run.Summary["source_file_name"])
}

func TestRunInventoryProfileGoldenContentAndName(t *testing.T) {
	repo := newFakeRepository()
	for i := range repo.inventoryProfiles {
		if repo.inventoryProfiles[i].Code == "ENB" {
			repo.inventoryProfiles[i].CompressionEnabled = false
			repo.inventoryProfiles[i].PathTemplate = "/northupload/GD/BaiOMC/Inventory/#DateTime#/"
			repo.inventoryProfiles[i].FileNameTemplate = "inventory_#Object#_#PeriodEndTime#.csv"
			repo.inventoryProfiles[i].Fields = []InventoryFieldConfig{
				{Key: "enb-1", OutputAlias: "Serial Number", SystemField: "device.serial_number", Enabled: true},
				{Key: "enb-2", OutputAlias: "Cell Status", SystemField: "device_info.op_state", Enabled: true},
				{Key: "enb-3", OutputAlias: "Online Status", SystemField: "device.is_online", Enabled: true},
				{Key: "enb-4", OutputAlias: "IP Address", SystemField: "device.ip_address", Enabled: true},
				{Key: "enb-5", OutputAlias: "Product Type", SystemField: "device.product_class", Enabled: true},
			}
			break
		}
	}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	windowEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)

	run, err := svc.RunInventoryProfile(context.Background(), "ENB", RunProfileRequest{
		WindowEnd: &windowEnd,
		Limit:     1,
	})
	require.NoError(t, err)
	require.Equal(t, "inventory_eNB_20260804000000.csv", run.ArtifactName)
	require.Equal(t, "/northupload/GD/BaiOMC/Inventory/20260804000000/", run.ArtifactPath)
	require.Equal(t, strings.Join([]string{
		"Serial Number,Cell Status,Online Status,IP Address,Product Type",
		"SN0001,1,true,10.0.0.1,pBS11004",
		"",
	}, "\n"), run.ArtifactContent)
}

func TestRunFileProfileArchivesArtifactLocally(t *testing.T) {
	repo := newFakeRepository()
	cmGroup := group("cm-one", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, []ScenarioObject{{Code: "CP"}})
	cmGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9301", "Local archive scenario", "Custom", "Custom", []string{"custom"}, []FileGroup{cmGroup}),
	}
	repo.deliveryTargets = nil
	archive := &fakeLocalArchiveStore{bucket: "northbound"}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	svc.SetLocalArchive(archive, LocalArchiveOptions{RetentionDays: 7})

	resp, err := svc.RunFileProfile(context.Background(), "S9301", RunProfileRequest{
		GroupID: "cm-one",
		Limit:   1,
	})

	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	require.Len(t, archive.puts, 1)
	require.Empty(t, resp.Items[0].ArtifactContent)
	require.Contains(t, string(archive.puts[0].Content), "Serial Number")
	require.Equal(t, "text/csv; charset=utf-8", archive.puts[0].ContentType)
	require.Contains(t, archive.puts[0].Key, resp.Items[0].CreatedAt.Local().Format("2006-01-02")+"/S9301/cm-one/run-1/")
	require.Contains(t, archive.puts[0].Key, ".csv")
	require.Len(t, repo.events, 2)
	require.Equal(t, "run", repo.events[0].EventType)
	require.Equal(t, "local_archive", repo.events[1].EventType)
	require.Equal(t, RunStatusSuccess, repo.events[1].Status)
	require.Equal(t, "northbound/"+archive.puts[0].Key, repo.events[1].ArtifactPath)
}

func TestDownloadRunArtifactFallsBackToLocalArchive(t *testing.T) {
	repo := newFakeRepository()
	cmGroup := group("cm-one", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, []ScenarioObject{{Code: "CP"}})
	cmGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9302", "Local archive download", "Custom", "Custom", []string{"custom"}, []FileGroup{cmGroup}),
	}
	repo.deliveryTargets = nil
	archive := &fakeLocalArchiveStore{bucket: "northbound"}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	svc.SetLocalArchive(archive, LocalArchiveOptions{RetentionDays: 7})

	resp, err := svc.RunFileProfile(context.Background(), "S9302", RunProfileRequest{
		GroupID: "cm-one",
		Limit:   1,
	})
	require.NoError(t, err)
	require.Empty(t, repo.runs[0].ArtifactContent)

	download, err := svc.DownloadRunArtifact(context.Background(), resp.Items[0].ID)

	require.NoError(t, err)
	require.Equal(t, resp.Items[0].ArtifactName, download.FileName)
	require.Equal(t, "text/csv; charset=utf-8", download.ContentType)
	require.Contains(t, string(download.Content), "Serial Number")
	require.Contains(t, string(download.Content), "SN0001")
}

func TestGetRunHydratesPreviewFromLocalArchive(t *testing.T) {
	repo := newFakeRepository()
	cmGroup := group("cm-one", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, []ScenarioObject{{Code: "CP"}})
	cmGroup.CompressionEnabled = true
	repo.fileProfiles = []FileProfile{
		fileProfile("S9303", "Local archive preview", "Custom", "Custom", []string{"custom"}, []FileGroup{cmGroup}),
	}
	repo.deliveryTargets = nil
	archive := &fakeLocalArchiveStore{bucket: "northbound"}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	svc.SetLocalArchive(archive, LocalArchiveOptions{RetentionDays: 7})

	resp, err := svc.RunFileProfile(context.Background(), "S9303", RunProfileRequest{
		GroupID: "cm-one",
		Limit:   1,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Items[0].ArtifactContent)

	hydrated, err := svc.GetRun(context.Background(), resp.Items[0].ID)
	require.NoError(t, err)
	require.Contains(t, hydrated.ArtifactContent, "Serial Number")
	require.Contains(t, hydrated.ArtifactContent, "SN0001")
}

func TestCleanupLocalArchiveUsesSevenDayRetention(t *testing.T) {
	repo := newFakeRepository()
	archive := &fakeLocalArchiveStore{
		bucket: "northbound",
		cleanupSummary: LocalArchiveCleanupSummary{
			ObjectsDeleted: 2,
			BytesDeleted:   128,
		},
	}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	svc.SetLocalArchive(archive, LocalArchiveOptions{})
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	summary, err := svc.CleanupLocalArchive(context.Background(), now)

	require.NoError(t, err)
	require.Len(t, archive.cleanupReqs, 1)
	require.Empty(t, archive.cleanupReqs[0].Prefix)
	require.Equal(t, now.AddDate(0, 0, -7), archive.cleanupReqs[0].Before)
	require.Equal(t, int64(2), summary.ObjectsDeleted)
	require.Equal(t, int64(128), summary.BytesDeleted)
	require.Equal(t, 7, summary.RetentionDays)
}

func TestRunFileProfileUploadsToEnabledFTPTarget(t *testing.T) {
	ftpServer := startTestFTPServer(t)
	repo := newFakeRepository()
	cmGroup := group("cm-one", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, []ScenarioObject{{Code: "CP"}})
	cmGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9002", "FTP delivery scenario", "Custom", "Custom", []string{"custom"}, []FileGroup{cmGroup}),
	}
	repo.deliveryTargets = []DeliveryTarget{{
		Scope:          DeliveryScopeFile,
		OwnerCode:      "S9002",
		Key:            "oss-main",
		Name:           "OSS Main",
		Enabled:        true,
		Protocol:       DeliveryProtocolFTP,
		Host:           ftpServer.host,
		Port:           ftpServer.port,
		Username:       "oss",
		Credential:     "secret",
		AuthMode:       DeliveryAuthPassword,
		RemoteRoot:     "/northupload",
		RetryTimes:     0,
		TimeoutSeconds: 3,
		PassiveMode:    true,
	}}
	r := setupTestRouterWithRepository(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/file/profiles/S9002/run", bytes.NewReader([]byte(`{"group_id":"cm-one","limit":1}`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, repo.runs, 1)
	require.Len(t, repo.events, 2)
	require.Equal(t, "run", repo.events[0].EventType)
	require.Equal(t, "delivery", repo.events[1].EventType)
	require.Equal(t, RunStatusSuccess, repo.events[1].Status)
	require.Equal(t, "S9002", repo.events[1].OwnerCode)
	require.Contains(t, repo.events[1].ArtifactPath, "/northupload/GD/BaiOMC/CM/")

	uploads := ftpServer.uploads()
	require.Len(t, uploads, 1)
	require.Contains(t, uploads[0].path, "/northupload/GD/BaiOMC/CM/")
	require.Contains(t, uploads[0].path, ".csv")
	require.Contains(t, string(uploads[0].payload), "Serial Number")
}

func TestRunDueSchedulesCreatesFileAndInventoryRunsOnce(t *testing.T) {
	repo := newFakeRepository()
	cmGroup := group("cm-daily", DomainCM, FormatCSV, Period24H, 1, pathCM, nameCM, []ScenarioObject{{Code: "CP"}})
	cmGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9101", "Scheduled File", "Custom", "Custom", []string{"custom"}, []FileGroup{cmGroup}),
	}
	repo.fileProfiles[0].Enabled = true
	repo.fileProfiles[0].Status = StatusNormal
	repo.inventoryProfiles = []InventoryProfile{{
		ID:                 "ENB",
		Code:               "ENB",
		Name:               "eNB Inventory",
		ObjectCode:         "ENB",
		Tech:               "LTE",
		Period:             Period24H,
		StartMinute:        1,
		PathTemplate:       "/northupload/GD/BaiOMC/Inventory/#DateTime#/",
		FileNameTemplate:   "inventory_#Object#_#PeriodEndTime#.csv",
		CompressionEnabled: false,
		Enabled:            true,
		Status:             StatusNormal,
	}}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	now := time.Date(2026, 8, 4, 0, 1, 0, 0, time.Local)

	summary, err := svc.RunDueSchedules(context.Background(), now, ScheduleRunOptions{MaxRuns: 10})
	require.NoError(t, err)
	require.Equal(t, 2, summary.Scanned)
	require.Equal(t, 2, summary.Due)
	require.Equal(t, 2, summary.Ran)
	require.Len(t, repo.runs, 2)
	require.Len(t, repo.events, 2)
	require.Equal(t, "20260804000000", repo.runs[0].WindowEnd.Format("20060102150405"))
	require.Equal(t, "20260804000000", repo.runs[1].WindowEnd.Format("20060102150405"))

	summary, err = svc.RunDueSchedules(context.Background(), now.Add(time.Minute), ScheduleRunOptions{MaxRuns: 10})
	require.NoError(t, err)
	require.Equal(t, 0, summary.Due)
	require.Equal(t, 0, summary.Ran)
	require.Len(t, repo.runs, 2)
}

func TestRunDueSchedulesWaitsForStartMinuteAndRetriesFailedWindow(t *testing.T) {
	repo := newFakeRepository()
	cmGroup := group("cm-15m", DomainCM, FormatCSV, Period15M, 5, pathCM, nameCM, []ScenarioObject{{Code: "CP"}})
	cmGroup.CompressionEnabled = false
	repo.fileProfiles = []FileProfile{
		fileProfile("S9102", "Scheduled Retry", "Custom", "Custom", []string{"custom"}, []FileGroup{cmGroup}),
	}
	repo.fileProfiles[0].Enabled = true
	repo.fileProfiles[0].Status = StatusNormal
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	previousEnd := time.Date(2026, 8, 4, 9, 45, 0, 0, time.Local)
	_, err := repo.CreateFileRun(context.Background(), FileRun{
		ProfileKind: ProfileKindFile,
		ProfileCode: "S9102",
		GroupID:     "cm-15m",
		Domain:      DomainCM,
		ObjectCode:  "CP",
		Status:      RunStatusSuccess,
		WindowStart: timePtr(previousEnd.Add(-15 * time.Minute)),
		WindowEnd:   timePtr(previousEnd),
	})
	require.NoError(t, err)

	beforeFire := time.Date(2026, 8, 4, 10, 4, 0, 0, time.Local)
	summary, err := svc.RunDueSchedules(context.Background(), beforeFire, ScheduleRunOptions{MaxRuns: 10})
	require.NoError(t, err)
	require.Equal(t, 0, summary.Due)
	require.Len(t, repo.runs, 1)

	failedEnd := time.Date(2026, 8, 4, 10, 0, 0, 0, time.Local)
	_, err = repo.CreateFileRun(context.Background(), FileRun{
		ProfileKind: ProfileKindFile,
		ProfileCode: "S9102",
		GroupID:     "cm-15m",
		Domain:      DomainCM,
		ObjectCode:  "CP",
		Status:      RunStatusFailed,
		WindowStart: timePtr(failedEnd.Add(-15 * time.Minute)),
		WindowEnd:   timePtr(failedEnd),
		CreatedAt:   failedEnd.Add(5 * time.Minute),
	})
	require.NoError(t, err)

	summary, err = svc.RunDueSchedules(context.Background(), failedEnd.Add(6*time.Minute), ScheduleRunOptions{RetryDelay: 10 * time.Minute})
	require.NoError(t, err)
	require.Equal(t, 0, summary.Ran)
	require.Len(t, repo.runs, 2)

	summary, err = svc.RunDueSchedules(context.Background(), failedEnd.Add(16*time.Minute), ScheduleRunOptions{RetryDelay: 10 * time.Minute})
	require.NoError(t, err)
	require.Equal(t, 1, summary.Ran)
	require.Len(t, repo.runs, 3)
	require.Equal(t, RunStatusSuccess, repo.runs[2].Status)
	require.Equal(t, "20260804100000", repo.runs[2].WindowEnd.Format("20060102150405"))
}

func TestListAndDownloadRun(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)
	created, err := repo.CreateFileRun(context.Background(), FileRun{
		ProfileKind:     ProfileKindFile,
		ProfileCode:     "S0001",
		GroupID:         "cm-daily",
		Domain:          DomainCM,
		ObjectCode:      "CP",
		Status:          RunStatusSuccess,
		ArtifactPath:    "/northupload/GD/BaiOMC/CM/20260804000000/",
		ArtifactName:    "sample.csv",
		ArtifactContent: "Serial Number\nSN0001\n",
		ArtifactSize:    21,
		RowCount:        1,
	})
	require.NoError(t, err)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/runs?profile_kind=file&profile_code=S0001", nil)
	listRR := httptest.NewRecorder()
	r.ServeHTTP(listRR, listReq)

	require.Equal(t, http.StatusOK, listRR.Code)
	require.NotContains(t, listRR.Body.String(), "SN0001")
	require.Contains(t, listRR.Body.String(), `"total":1`)

	downloadReq := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/runs/"+created.ID+"/download", nil)
	downloadRR := httptest.NewRecorder()
	r.ServeHTTP(downloadRR, downloadReq)

	require.Equal(t, http.StatusOK, downloadRR.Code)
	require.Equal(t, "text/csv; charset=utf-8", downloadRR.Header().Get("Content-Type"))
	require.Equal(t, "Serial Number\nSN0001\n", downloadRR.Body.String())
}

func TestListRunsLatestPerProfileUsesLatestSuccessfulRun(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)
	base := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	for _, run := range []FileRun{
		{ID: "s1-old", ProfileKind: ProfileKindFile, ProfileCode: "S0001", GroupID: "cm", Status: RunStatusSuccess, CreatedAt: base},
		{ID: "s1-new", ProfileKind: ProfileKindFile, ProfileCode: "S0001", GroupID: "pm", Status: RunStatusSuccess, CreatedAt: base.Add(2 * time.Hour)},
		{ID: "s2-success", ProfileKind: ProfileKindFile, ProfileCode: "S0002", GroupID: "cm", Status: RunStatusSuccess, CreatedAt: base.Add(time.Hour)},
		{ID: "s2-failed-newer", ProfileKind: ProfileKindFile, ProfileCode: "S0002", GroupID: "pm", Status: RunStatusFailed, CreatedAt: base.Add(3 * time.Hour)},
		{ID: "inv-enb", ProfileKind: ProfileKindInventory, ProfileCode: "ENB", GroupID: "ENB", Status: RunStatusSuccess, CreatedAt: base.Add(4 * time.Hour)},
	} {
		_, err := repo.CreateFileRun(context.Background(), run)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/runs?profile_kind=file&status=success&latest_per_profile=true&limit=10", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Data struct {
			Items []FileRun `json:"items"`
			Total int       `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, 2, body.Data.Total)
	require.Len(t, body.Data.Items, 2)
	require.Equal(t, "s1-new", body.Data.Items[0].ID)
	require.Equal(t, "s2-success", body.Data.Items[1].ID)
}

func TestReplaceDeliveryTargetsRedactsCredential(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)
	body := []byte(`{
  "scope": "file",
  "owner_code": "",
  "items": [
    {
      "key": "oss-main",
      "name": "OSS Main",
      "enabled": true,
      "protocol": "SFTP",
      "host": "10.10.1.10",
      "port": 22,
      "username": "oss",
      "credential": "secret",
      "auth_mode": "PASSWORD",
      "remote_root": "/northupload",
      "retry_times": 3,
      "timeout_seconds": 30,
      "passive_mode": false
    }
  ]
}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/delivery/targets", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"credential_set":true`)
	require.NotContains(t, rr.Body.String(), "secret")
}

func TestSNMPFieldsFollowMIBOrder(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/alarm/snmp/targets", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"field":"notificationID"`)
	require.Contains(t, rr.Body.String(), `"oid":"1.3.6.1.4.1.53058.1.1.1.1.1.1"`)
	require.Contains(t, rr.Body.String(), `"field":"additionalInformation"`)
}

func TestUpdateSNMPV2TargetDefaultsCommunity(t *testing.T) {
	repo := newFakeRepository()
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	target, err := svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v2-primary", SNMPAlarmTarget{
		Key:              "snmp-v2-primary",
		Name:             "SNMP v2",
		Enabled:          true,
		Version:          "v2",
		NotificationType: "Trap",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetHost:       "127.0.0.1",
		TargetPort:       162,
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})

	require.NoError(t, err)
	require.Equal(t, defaultSNMPV2Community, target.Community)
	require.True(t, target.Enabled)
}

func TestUpdateBuiltInSNMPTargetKeepsVersionByKey(t *testing.T) {
	repo := newFakeRepository()
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	target, err := svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v2-primary", SNMPAlarmTarget{
		Key:              "snmp-v2-primary",
		Name:             "SNMP V2C",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Trap",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       162,
		MIBQueryEnabled:  false,
		TimeoutSeconds:   5,
	})

	require.NoError(t, err)
	require.Equal(t, "v2", target.Version)

	target, err = svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP V3",
		Enabled:          false,
		Version:          "v2",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		AuthProtocol:     "SHA",
		PrivProtocol:     "DES",
		MIBQueryEnabled:  false,
		TimeoutSeconds:   5,
	})

	require.NoError(t, err)
	require.Equal(t, "v3", target.Version)
}

func TestSNMPMIBQueryTargetsDoNotRequireAlarmReportEnabled(t *testing.T) {
	repo := newFakeRepository()
	repo.snmpTargets = []SNMPAlarmTarget{{
		Key:             "snmp-v2-query-only",
		Name:            "SNMP v2 Query",
		Enabled:         false,
		Version:         "v2",
		ListenIP:        "0.0.0.0",
		ListenPort:      161,
		Community:       defaultSNMPV2Community,
		MIBQueryEnabled: true,
	}}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	targets, err := svc.listActiveSNMPMIBTargetsForServe(context.Background())
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.False(t, targets[0].Enabled)
	require.True(t, targets[0].MIBQueryEnabled)
}

func TestUpdateSNMPV3MIBQueryRequiresSecurityCredentials(t *testing.T) {
	repo := newFakeRepository()
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	_, err := svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP v3",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		AuthProtocol:     "SHA",
		PrivProtocol:     "DES",
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "security_name")
}

func TestUpdateSNMPV3MIBQueryAllowsNoAuthNoPriv(t *testing.T) {
	repo := newFakeRepository()
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	target, err := svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP v3",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		SecurityName:     "queryV3",
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})

	require.NoError(t, err)
	require.Empty(t, target.AuthProtocol)
	require.Empty(t, target.PrivProtocol)
	require.True(t, target.MIBQueryEnabled)
}

func TestUpdateSNMPV3MIBQueryRequiresCredentialsBySecurityLevel(t *testing.T) {
	repo := newFakeRepository()
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	_, err := svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP v3",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		SecurityName:     "queryV3",
		AuthProtocol:     "SHA",
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "auth credential")

	_, err = svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP v3",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		SecurityName:     "queryV3",
		AuthProtocol:     "SHA",
		AuthCredential:   "AuthPassword",
		PrivProtocol:     "DES",
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "privacy credential")

	_, err = svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP v3",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		SecurityName:     "queryV3",
		AuthProtocol:     "SHA",
		AuthCredential:   "short",
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "auth credential must be at least 8 characters")

	_, err = svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP v3",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		SecurityName:     "queryV3",
		AuthProtocol:     "SHA",
		AuthCredential:   "AuthPassword",
		PrivProtocol:     "DES",
		PrivCredential:   "short",
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "privacy credential must be at least 8 characters")

	_, err = svc.UpdateSNMPAlarmTarget(context.Background(), "snmp-v3-inform", SNMPAlarmTarget{
		Key:              "snmp-v3-inform",
		Name:             "SNMP v3",
		Enabled:          false,
		Version:          "v3",
		NotificationType: "Inform",
		ListenIP:         "0.0.0.0",
		ListenPort:       161,
		TargetPort:       163,
		SecurityName:     "queryV3",
		PrivProtocol:     "DES",
		MIBQueryEnabled:  true,
		TimeoutSeconds:   5,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "privacy requires auth_protocol")
}

func TestTestSNMPAlarmTargetUsesSenderAndRecordsResult(t *testing.T) {
	repo := newFakeRepository()
	repo.snmpTargets = []SNMPAlarmTarget{{
		Key:              "snmp-v2-primary",
		Name:             "SNMP v2",
		Enabled:          true,
		Version:          "v2",
		NotificationType: "Trap",
		TargetHost:       "127.0.0.1",
		TargetPort:       162,
		Community:        "public",
		TimeoutSeconds:   1,
		MIBFields:        defaultSNMPAlarmFields(),
	}}
	sender := &recordingSNMPSender{}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)
	svc.SetSNMPSender(sender)

	event, err := svc.TestSNMPAlarmTarget(context.Background(), "snmp-v2-primary")
	require.NoError(t, err)
	require.Equal(t, RunStatusSuccess, event.Status)
	require.Contains(t, event.Payload, OIDNotificationIDValue)
	require.Len(t, repo.events, 1)
	require.Equal(t, 1, sender.calls)
	require.False(t, sender.lastTarget.Inform)
	require.Equal(t, nbsnmp.VersionV2c, sender.lastTarget.Version)
	require.Len(t, sender.lastVars, 18)
	require.Equal(t, OIDNotificationIDValue, sender.lastVars[0].OID)
}

func TestUpdateSocketConfigRequiresServerMode(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{"key":"socket-ctcc-server","name":"CTCC","profile":"CTCC","mode":"client","listen_port":31232,"heartbeat_seconds":30,"idle_timeout_seconds":180}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/alarm/socket/configs/socket-ctcc-server", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	require.Contains(t, rr.Body.String(), "socket mode must be server")
}

func TestUpdateAPIConfigOnlyTogglesSupportedConfig(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{"enabled":true}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/api/configs/nb-sync-full-device", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"enabled":true`)
	require.Contains(t, rr.Body.String(), `/api/v1/northbound/v1/sync/full?data_type=device`)
	require.Contains(t, rr.Body.String(), `"data_type":"device"`)
}

func TestUpdateAllAPIConfigsTogglesEverySupportedConfig(t *testing.T) {
	repo := newFakeRepository()
	r := setupTestRouterWithRepository(repo)
	body := []byte(`{"enabled":true}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/api/configs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), fmt.Sprintf(`"total":%d`, len(repo.apiConfigs)))
	for _, config := range repo.apiConfigs {
		require.True(t, config.Enabled, config.Key)
	}
}

func TestIsAPIConfigEnabledFollowsPageSwitch(t *testing.T) {
	repo := newFakeRepository()
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	enabled, err := svc.IsAPIConfigEnabled(context.Background(), "nb-sync-full-device")
	require.NoError(t, err)
	require.False(t, enabled)

	_, err = svc.UpdateAPIConfig(context.Background(), "nb-sync-full-device", true)
	require.NoError(t, err)
	enabled, err = svc.IsAPIConfigEnabled(context.Background(), "nb-sync-full-device")
	require.NoError(t, err)
	require.True(t, enabled)

	enabled, err = svc.IsAPIConfigEnabled(context.Background(), "unknown-api")
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestAPIUserLoginIssuesNorthboundOnlyToken(t *testing.T) {
	repo := newFakeRepository()
	repo.apiClients = []APIClient{{
		ClientKey:      "oss-a",
		Name:           "OSS A",
		Enabled:        true,
		TokenSecret:    "secret-password",
		AllowedAPIKeys: []string{"nb-sync-full-device"},
		IPWhitelist:    []string{"127.0.0.1/32"},
	}}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	token, err := svc.LoginAPIUser(context.Background(), APIUserLoginRequest{
		Username: "oss-a",
		Password: "secret-password",
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.Token)
	require.Equal(t, 1800, token.Expires)

	client, err := svc.AuthenticateAPIClient(context.Background(), token.Token, "127.0.0.1", "nb-sync-full-device")
	require.NoError(t, err)
	require.Equal(t, "oss-a", client.ClientKey)

	_, err = svc.AuthenticateAPIClient(context.Background(), token.Token, "127.0.0.1", "nb-export-config")
	require.ErrorIs(t, err, commonerrors.ErrForbidden)

	_, err = svc.AuthenticateAPIClient(context.Background(), token.Token, "10.0.0.10", "nb-sync-full-device")
	require.ErrorIs(t, err, commonerrors.ErrForbidden)

	_, err = svc.AuthenticateAPIClient(context.Background(), "bad-token", "127.0.0.1", "nb-sync-full-device")
	require.ErrorIs(t, err, commonerrors.ErrUnauthorized)

	_, err = svc.LoginAPIUser(context.Background(), APIUserLoginRequest{
		Username: "oss-a",
		Password: "bad-password",
	})
	require.ErrorIs(t, err, commonerrors.ErrUnauthorized)
}

func TestReplaceAPIUsersReturnsPasswordForManagementPage(t *testing.T) {
	r := setupTestRouterWithRepository(newFakeRepository())
	body := []byte(`{"items":[{"username":"oss-a","enabled":true,"password":"secret-password"}]}`)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/northbound/page-config/api/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"username":"oss-a"`)
	require.Contains(t, rr.Body.String(), `"password_set":true`)
	require.Contains(t, rr.Body.String(), `"password":"secret-password"`)
}

func TestAPIUserSingleMutationsPreserveExistingClientScopes(t *testing.T) {
	repo := newFakeRepository()
	repo.apiClients = []APIClient{{
		ID:             "existing-id",
		ClientKey:      "oss-a",
		Name:           "OSS A",
		Enabled:        true,
		TokenSecret:    "secret-a",
		TokenSet:       true,
		AllowedAPIKeys: []string{"device-list"},
		IPWhitelist:    []string{"127.0.0.1/32"},
	}}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	created, err := svc.CreateAPIUser(context.Background(), APIUser{
		Username: "oss-b",
		Password: "secret-b",
		Enabled:  true,
	})
	require.NoError(t, err)
	require.Equal(t, "oss-b", created.Username)

	disabled := false
	updated, err := svc.UpdateAPIUser(context.Background(), "oss-b", UpdateAPIUserRequest{Enabled: &disabled})
	require.NoError(t, err)
	require.False(t, updated.Enabled)

	require.NoError(t, svc.DeleteAPIUser(context.Background(), "oss-b"))
	require.Len(t, repo.apiClients, 1)
	require.Equal(t, []string{"device-list"}, repo.apiClients[0].AllowedAPIKeys)
	require.Equal(t, []string{"127.0.0.1/32"}, repo.apiClients[0].IPWhitelist)
}

func TestCleanupExpiredResultsRemovesOldRunsAndEvents(t *testing.T) {
	repo := newFakeRepository()
	now := time.Now()
	repo.runs = []FileRun{
		{ID: "old-run", CreatedAt: now.AddDate(0, 0, -120)},
		{ID: "fresh-run", CreatedAt: now},
	}
	repo.events = []PageConfigEvent{
		{ID: "old-event", CreatedAt: now.AddDate(0, 0, -120)},
		{ID: "fresh-event", CreatedAt: now},
	}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	summary, err := svc.CleanupExpiredResults(context.Background(), ResultRetentionPolicy{
		RunRetentionDays:   90,
		EventRetentionDays: 90,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, summary.RunsDeleted)
	require.EqualValues(t, 1, summary.EventsDeleted)
	require.Len(t, repo.runs, 1)
	require.Equal(t, "fresh-run", repo.runs[0].ID)
	require.Len(t, repo.events, 1)
	require.Equal(t, "fresh-event", repo.events[0].ID)
}

func TestListAPIInvocationLogsEndpointFiltersAndPaginates(t *testing.T) {
	repo := newFakeRepository()
	base := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	repo.apiInvocationLogs = []APIInvocationLog{
		{
			ID:            "new-device-query",
			APIKey:        "device-list",
			Name:          "设备列表查询",
			Method:        http.MethodPost,
			Path:          "/api/v1/northbound/v1/device/query",
			RequestParams: `{"sn":"SN001"}`,
			ResponseBody:  `{"ret":1}`,
			StatusCode:    http.StatusOK,
			Status:        "success",
			CreateUser:    "oss",
			IPAddress:     "127.0.0.1",
			DurationMs:    12,
			CreatedAt:     base.Add(time.Hour),
			UpdatedAt:     base.Add(time.Hour),
		},
		{
			ID:            "old-token",
			APIKey:        "auth-login",
			Name:          "用户认证",
			Method:        http.MethodPost,
			Path:          "/api/v1/northbound/v1/access/token",
			RequestParams: `{"user":"oss"}`,
			ResponseBody:  `{"ret":0}`,
			StatusCode:    http.StatusUnauthorized,
			Status:        "failed",
			CreateUser:    "oss",
			IPAddress:     "127.0.0.2",
			DurationMs:    7,
			CreatedAt:     base,
			UpdatedAt:     base,
		},
	}
	r := setupTestRouterWithRepository(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/api/invocation-logs?method=POST&status=success&keyword=SN001&page=1&page_size=10", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Contains(t, rr.Body.String(), `"total":1`)
	require.Contains(t, rr.Body.String(), `"id":"new-device-query"`)
	require.NotContains(t, rr.Body.String(), `"id":"old-token"`)
	require.Contains(t, rr.Body.String(), `"page_size":10`)
}

func TestListAndGetEvents(t *testing.T) {
	repo := newFakeRepository()
	created, err := repo.CreateEvent(context.Background(), PageConfigEvent{
		Capability:   "snmp",
		OwnerCode:    "snmp-v2-primary",
		TargetKey:    "snmp-v2-primary",
		EventType:    "message_test",
		Status:       RunStatusSuccess,
		ArtifactType: EventArtifactMessage,
		ArtifactName: "omcAlarmNotification Trap",
		ArtifactPath: "snmp://10.10.41.11:162",
		Payload:      "1.3.6.1.4.1.53058.1.1.1.1.1.1 = 920188\n",
	})
	require.NoError(t, err)

	r := setupTestRouterWithRepository(repo)
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/events?capability=snmp&target_key=snmp-v2-primary", nil)
	listRR := httptest.NewRecorder()
	r.ServeHTTP(listRR, listReq)

	require.Equal(t, http.StatusOK, listRR.Code)
	require.Contains(t, listRR.Body.String(), `"total":1`)
	require.Contains(t, listRR.Body.String(), `"event_type":"message_test"`)

	listNoPayloadReq := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/events?capability=snmp&target_key=snmp-v2-primary&include_payload=false", nil)
	listNoPayloadRR := httptest.NewRecorder()
	r.ServeHTTP(listNoPayloadRR, listNoPayloadReq)

	require.Equal(t, http.StatusOK, listNoPayloadRR.Code)
	require.Contains(t, listNoPayloadRR.Body.String(), `"total":1`)
	require.Contains(t, listNoPayloadRR.Body.String(), `"event_type":"message_test"`)
	require.NotContains(t, listNoPayloadRR.Body.String(), `1.3.6.1.4.1.53058`)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/northbound/page-config/events/"+created.ID, nil)
	getRR := httptest.NewRecorder()
	r.ServeHTTP(getRR, getReq)

	require.Equal(t, http.StatusOK, getRR.Code)
	require.Contains(t, getRR.Body.String(), `"id":"`+created.ID+`"`)
	require.Contains(t, getRR.Body.String(), `1.3.6.1.4.1.53058`)
}

func TestTestSNMPAlarmTargetReturnsMIBPayloadEvent(t *testing.T) {
	repo := newFakeRepository()
	repo.snmpTargets[0].TargetHost = "10.10.41.11"
	r := setupTestRouterWithRepository(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/alarm/snmp/targets/snmp-v2-primary/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, repo.events, 1)
	require.Contains(t, rr.Body.String(), `"capability":"snmp"`)
	require.Contains(t, rr.Body.String(), `1.3.6.1.4.1.53058.1.1.1.1.1.18`)
	require.NotContains(t, rr.Body.String(), `1.3.6.1.4.1.52642`)
}

func TestTestSocketAlarmConfigReturnsMessageEvent(t *testing.T) {
	repo := newFakeRepository()
	repo.socketConfigs[0].Enabled = true
	r := setupTestRouterWithRepository(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/alarm/socket/configs/socket-ctcc-server/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, repo.events, 1)
	require.Contains(t, rr.Body.String(), `"capability":"socket"`)
	require.Contains(t, rr.Body.String(), `alarmSequenceId`)
	require.Contains(t, rr.Body.String(), `920188`)
}

func TestTestAPIConfigReturnsContractEvent(t *testing.T) {
	repo := newFakeRepository()
	repo.apiConfigs[1].Enabled = true
	r := setupTestRouterWithRepository(repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/northbound/page-config/api/configs/nb-sync-full-device/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, repo.events, 1)
	require.Contains(t, rr.Body.String(), `"capability":"api"`)
	require.Contains(t, rr.Body.String(), `/api/v1/northbound/v1/sync/full?data_type=device`)
}

type testFTPUpload struct {
	path    string
	payload []byte
}

type testFTPServer struct {
	host    string
	port    int
	ln      net.Listener
	mu      sync.Mutex
	records []testFTPUpload
}

func startTestFTPServer(t *testing.T) *testFTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	srv := &testFTPServer{host: host, port: port, ln: ln}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go srv.handleConn(conn)
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})
	return srv
}

func (s *testFTPServer) uploads() []testFTPUpload {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]testFTPUpload, len(s.records))
	copy(out, s.records)
	return out
}

func (s *testFTPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)
	writeFTPLine(writer, "220 test ftp ready")
	var dataLn net.Listener
	defer func() {
		if dataLn != nil {
			_ = dataLn.Close()
		}
	}()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "USER "):
			writeFTPLine(writer, "331 password required")
		case strings.HasPrefix(upper, "PASS "):
			writeFTPLine(writer, "230 login ok")
		case strings.HasPrefix(upper, "TYPE "):
			writeFTPLine(writer, "200 type ok")
		case strings.HasPrefix(upper, "MKD "):
			writeFTPLine(writer, "257 directory created")
		case strings.HasPrefix(upper, "PASV"):
			if dataLn != nil {
				_ = dataLn.Close()
			}
			var err error
			dataLn, err = net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				writeFTPLine(writer, "425 cannot open data connection")
				continue
			}
			_, portText, _ := net.SplitHostPort(dataLn.Addr().String())
			port, _ := strconv.Atoi(portText)
			writeFTPLine(writer, fmt.Sprintf("227 Entering Passive Mode (127,0,0,1,%d,%d)", port/256, port%256))
		case strings.HasPrefix(upper, "STOR "):
			remotePath := strings.TrimSpace(line[5:])
			writeFTPLine(writer, "150 opening data")
			if dataLn == nil {
				writeFTPLine(writer, "425 passive listener missing")
				continue
			}
			dataConn, err := dataLn.Accept()
			if err != nil {
				writeFTPLine(writer, "425 data accept failed")
				continue
			}
			payload, _ := io.ReadAll(dataConn)
			_ = dataConn.Close()
			_ = dataLn.Close()
			dataLn = nil
			s.mu.Lock()
			s.records = append(s.records, testFTPUpload{path: remotePath, payload: payload})
			s.mu.Unlock()
			writeFTPLine(writer, "226 transfer complete")
		case strings.HasPrefix(upper, "QUIT"):
			writeFTPLine(writer, "221 bye")
			return
		default:
			writeFTPLine(writer, "200 ok")
		}
	}
}

func writeFTPLine(writer *bufio.Writer, line string) {
	_, _ = writer.WriteString(line + "\r\n")
	_ = writer.Flush()
}

func TestUploadSFTPPasswordAuth(t *testing.T) {
	srv := startTestSFTPServer(t)
	artifact := []byte("sn,severity\nSN0001,major\n")
	run := FileRun{
		ID:           "run-sftp-1",
		ProfileKind:  ProfileKindFile,
		ProfileCode:  "S9003",
		Status:       RunStatusSuccess,
		ArtifactPath: "/",
		ArtifactName: "alarm.csv",
	}
	// SFTP uses username + password only; host-key verification is disabled.
	target := DeliveryTarget{
		Key:            "sftp-primary",
		Protocol:       DeliveryProtocolSFTP,
		Host:           srv.host,
		Port:           srv.port,
		Username:       "north",
		Credential:     "secret",
		AuthMode:       DeliveryAuthPassword,
		RemoteRoot:     srv.root,
		RetryTimes:     0,
		TimeoutSeconds: 5,
	}

	result := uploadRunArtifact(context.Background(), target, run, artifact)
	require.True(t, result.Success, result.Error)
	uploaded, err := os.ReadFile(filepath.Join(srv.root, "alarm.csv"))
	require.NoError(t, err)
	require.Equal(t, artifact, uploaded)
}

func TestDeliveryTargetConnectionTestUsesStoredCredential(t *testing.T) {
	srv := startTestSFTPServer(t)
	repo := newFakeRepository()
	repo.deliveryTargets = []DeliveryTarget{{
		Scope:          DeliveryScopeFile,
		OwnerCode:      "S0312",
		Key:            "sftp-stored",
		Name:           "SFTP Stored",
		Enabled:        true,
		Protocol:       DeliveryProtocolSFTP,
		Host:           srv.host,
		Port:           srv.port,
		Username:       "north",
		Credential:     "secret",
		CredentialSet:  true,
		AuthMode:       DeliveryAuthPassword,
		RemoteRoot:     "/northupload",
		RetryTimes:     0,
		TimeoutSeconds: 5,
	}}
	svc := NewServiceWithRepository(NewDefaultCatalog(), repo)

	event, err := svc.TestDeliveryTarget(context.Background(), DeliveryTarget{
		Scope:          DeliveryScopeFile,
		OwnerCode:      "S0312",
		Key:            "sftp-stored",
		Name:           "SFTP Stored",
		Protocol:       DeliveryProtocolSFTP,
		Host:           srv.host,
		Port:           srv.port,
		Username:       "north",
		CredentialSet:  true,
		AuthMode:       DeliveryAuthPassword,
		RemoteRoot:     "/northupload",
		RetryTimes:     0,
		TimeoutSeconds: 5,
	})

	require.NoError(t, err)
	require.Equal(t, RunStatusSuccess, event.Status)
	passed, ok := event.Summary["auth_probe_passed"].(*bool)
	require.True(t, ok)
	require.NotNil(t, passed)
	require.True(t, *passed)
}

type testSFTPServer struct {
	host        string
	port        int
	root        string
	fingerprint string
	ln          net.Listener
}

func startTestSFTPServer(t *testing.T) *testSFTPServer {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := ssh.NewSignerFromKey(privateKey)
	require.NoError(t, err)
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if conn.User() == "north" && string(password) == "secret" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, portText, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portText)
	require.NoError(t, err)
	srv := &testSFTPServer{
		host:        host,
		port:        port,
		root:        t.TempDir(),
		fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
		ln:          ln,
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveTestSFTPConn(conn, cfg)
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		<-done
	})
	return srv
}

func serveTestSFTPConn(conn net.Conn, cfg *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(requests)
	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		ch, reqs, err := channel.Accept()
		if err != nil {
			continue
		}
		go serveTestSFTPChannel(ch, reqs)
	}
}

func serveTestSFTPChannel(ch ssh.Channel, reqs <-chan *ssh.Request) {
	defer ch.Close()
	for req := range reqs {
		if req.Type != "subsystem" {
			_ = req.Reply(false, nil)
			continue
		}
		var payload struct {
			Name string
		}
		ssh.Unmarshal(req.Payload, &payload)
		if payload.Name != "sftp" {
			_ = req.Reply(false, nil)
			continue
		}
		_ = req.Reply(true, nil)
		server, err := sftp.NewServer(ch)
		if err != nil {
			return
		}
		_ = server.Serve()
		_ = server.Close()
		return
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

type recordingSNMPSender struct {
	calls      int
	lastTarget *nbsnmp.TrapTarget
	lastVars   []nbsnmp.Variable
	err        error
}

func (s *recordingSNMPSender) Send(_ context.Context, target *nbsnmp.TrapTarget, vars []nbsnmp.Variable) error {
	s.calls++
	s.lastTarget = target
	s.lastVars = append([]nbsnmp.Variable(nil), vars...)
	return s.err
}
