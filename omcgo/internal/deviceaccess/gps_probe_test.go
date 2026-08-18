package deviceaccess

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/config/parammodel"
	"github.com/omcgo/omcgo/internal/core/model"
	devicepkg "github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/product"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type gpsFlowRepository struct {
	context   EvaluationContext
	decisions []DecisionChange
	appended  []EvidenceBatch
}

func (r *gpsFlowRepository) LoadEvaluationContext(context.Context, string, string) (EvaluationContext, error) {
	return r.context, nil
}

func (r *gpsFlowRepository) SaveDecision(_ context.Context, change DecisionChange) (SavedDecision, error) {
	r.decisions = append(r.decisions, change)
	if change.Evidence != nil {
		r.context.Evidence = *change.Evidence
	}
	version := change.ExpectedDecisionVersion + 1
	r.context.State = &AccessStateProjection{
		Carrier:           change.Carrier,
		SerialNumber:      change.SerialNumber,
		DeviceID:          change.DeviceID,
		CandidateID:       change.CandidateID,
		State:             change.Decision.State,
		EffectiveDecision: change.Decision.EffectiveAction,
		ReasonCode:        change.Decision.ReasonCode,
		EvidenceVersion:   change.EvidenceVersion,
		DecisionVersion:   version,
		NormalTasksFrozen: change.Decision.FreezeNormal,
	}
	return SavedDecision{State: change.Decision.State, DecisionVersion: version}, nil
}

func (r *gpsFlowRepository) UpsertCandidateObservation(_ context.Context, observation Observation) (Candidate, error) {
	return Candidate{
		ID: uuid.New(), Carrier: observation.Carrier, SerialNumber: observation.SerialNumber,
		ProductClass: observation.ProductClass, SoftwareVersion: observation.SoftwareVersion,
	}, nil
}

func (r *gpsFlowRepository) AppendEvidence(_ context.Context, batch EvidenceBatch) (int64, error) {
	r.appended = append(r.appended, batch)
	r.context.Evidence = batch
	return batch.Version, nil
}

type gpsTaskEnqueuer struct {
	created *task.Task
}

func TestProbeCommandKeyBindsCompleteProbeContract(t *testing.T) {
	gpsOnly, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-001", EvidenceVersion: 2, NeedGPS: true,
	})
	require.NoError(t, err)
	withRadio, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-001", EvidenceVersion: 2,
		NeedGPS: true, NeedTAC: true, NeedECGI: true,
	})
	require.NoError(t, err)

	require.Equal(t, probeCommandKey("values", gpsOnly), probeCommandKey("values", gpsOnly))
	require.NotEqual(t, probeCommandKey("values", gpsOnly), probeCommandKey("values", withRadio))
	require.NotEqual(t, probeCommandKey("values", gpsOnly), probeCommandKey("discovery", gpsOnly))
}

func (e *gpsTaskEnqueuer) CreateTask(_ context.Context, request *task.CreateTaskRequest) (*task.Task, error) {
	e.created = task.NewTask(request)
	return e.created, nil
}

func (e *gpsTaskEnqueuer) GetQueueLength(context.Context, string) (int64, error) { return 0, nil }

func TestGPSProbeFakeCPECompletesInformGPVDecisionFlow(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{StandardPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude", PrivatePath: "Device.FAP.GPS.LockedLatitude"},
		{StandardPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude", PrivatePath: "Device.FAP.GPS.LockedLongitude"},
		{StandardPath: "Device.FAP.GPS.Height", PrivatePath: "Device.FAP.GPS.altidute"},
		{StandardPath: "Device.FAP.GPS.NumberOfSatellites", PrivatePath: "Device.DeviceInfo.X_COM_GPS_Satellite_count"},
		{StandardPath: "Device.DeviceInfo.GPS_Status", PrivatePath: "Device.DeviceInfo.X_COM_GPS_Status"},
		{StandardPath: "Device.DeviceInfo.GPS.horizontalAccuracy", PrivatePath: "Device.DeviceInfo.GPS.horizontalAccuracy"},
		{StandardPath: "Device.DeviceInfo.GPS.verticalAccuracy", PrivatePath: "Device.DeviceInfo.GPS.verticalAccuracy"},
	}}, nil, nil)
	paths := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)
	repository := &gpsFlowRepository{}
	policy := CompiledPolicy{Rules: []CompiledRule{{
		ID: "gps-rule", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll},
		Conditions: []CompiledCondition{{
			ID: "gps", Type: ConditionTypeGPS, Operator: ConditionOperatorWithinRadius,
			GeoFence: &GeoFence{Center: GeoPoint{Latitude: 25.924128, Longitude: 115.366448}, RadiusMeters: 100},
			Required: true, EvidenceTTL: time.Hour,
		}},
	}}}
	policies := accessGatePolicyStub{policy: policy}
	coordinator := NewReevaluationCoordinator(repository, accessGateAssetStub{
		evidence: AssetEvidence{Source: AssetEvidenceSourceDevice, DeviceID: ptrUUID(uuid.New())},
	}, policies)
	coordinator.SetClock(func() time.Time { return now.Add(time.Minute) })
	tasks := &gpsTaskEnqueuer{}
	probes := NewGPSProbeService(paths, tasks, repository, coordinator, zap.NewNop())
	probes.SetClock(func() time.Time { return now })
	gate := NewAccessGate(repository, accessGateAssetStub{
		evidence: AssetEvidence{Source: AssetEvidenceSourceDevice, DeviceID: ptrUUID(uuid.New())},
	}, policies)
	gate.SetClock(func() time.Time { return now })
	gate.SetGPSProbePlanner(probes)

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-GPS-FLOW", OUI: "48BF74",
		ProductClass: "FAP/BU1810", SoftwareVersion: "BM_2.0.4", EventID: "inform-1",
		Authenticated: true, CarrierIdentityResolved: true,
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionCollecting, decision.State)
	require.NotNil(t, tasks.created)
	require.Equal(t, task.TaskSourceDeviceAccess, tasks.created.Source)
	require.Equal(t, task.AdmissionClassAccessProbe, tasks.created.AdmissionClass)
	require.Equal(t, "GetParameterValues", tasks.created.Method)
	var params gpsProbeTaskParams
	require.NoError(t, json.Unmarshal(tasks.created.Params, &params))
	require.Equal(t, "cmcc", params.Carrier)
	require.Equal(t, int64(2), params.EvidenceVersion)
	require.Equal(t, []string{
		"Device.FAP.GPS.LockedLatitude",
		"Device.FAP.GPS.LockedLongitude",
		"Device.FAP.GPS.altidute",
		"Device.DeviceInfo.X_COM_GPS_Satellite_count",
		"Device.DeviceInfo.X_COM_GPS_Status",
		"Device.DeviceInfo.GPS.horizontalAccuracy",
		"Device.DeviceInfo.GPS.verticalAccuracy",
	}, params.Names)

	result, err := json.Marshal(map[string]any{"private_parameter_values": []tr069.ParameterValueStruct{
		{Name: "Device.FAP.GPS.LockedLatitude", Value: "25924128"},
		{Name: "Device.FAP.GPS.LockedLongitude", Value: "115366448"},
		{Name: "Device.FAP.GPS.altidute", Value: "169.070007"},
		{Name: "Device.DeviceInfo.X_COM_GPS_Satellite_count", Value: "0"},
		{Name: "Device.DeviceInfo.X_COM_GPS_Status", Value: "Locked"},
		{Name: "Device.DeviceInfo.GPS.horizontalAccuracy", Value: "50"},
		{Name: "Device.DeviceInfo.GPS.verticalAccuracy", Value: "3"},
	}})
	require.NoError(t, err)
	tasks.created.Status = task.TaskStatusCompleted
	tasks.created.Result = result

	require.NoError(t, probes.HandleCompleted(context.Background(), tasks.created))
	require.Len(t, repository.appended, 1)
	require.NotEmpty(t, repository.appended[0].Records)
	gpsRecord := evidenceRecordByType(t, repository.appended[0], ConditionTypeGPS)
	var normalized gpsEvidenceValue
	require.NoError(t, json.Unmarshal(gpsRecord.NormalizedValue, &normalized))
	require.Equal(t, 25.924128, normalized.Latitude)
	require.Equal(t, 115.366448, normalized.Longitude)
	require.Equal(t, "Locked", normalized.Status)
	require.Len(t, repository.decisions, 2)
	require.Equal(t, AccessStateCollecting, repository.decisions[0].Decision.State)
	require.Equal(t, AccessStateAccepted, repository.decisions[1].Decision.State)
	require.Nil(t, repository.decisions[1].Evidence, "already appended GPS evidence must not be inserted twice")
}

func TestGPSProbeDoesNotCreateTaskWhenLatitudeOrLongitudePathIsUnresolved(t *testing.T) {
	paramModelID := uuid.New()
	translator := parammodel.NewTranslator(&parammodel.MappingSet{Mappings: []parammodel.ParamMapping{
		{StandardPath: "Device.FAP.GPS.Height", PrivatePath: "Device.X_VENDOR.GPS.Height"},
	}}, nil, nil)
	paths := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: translator},
	)
	tasks := &gpsTaskEnqueuer{}
	probes := NewGPSProbeService(paths, tasks, &gpsFlowRepository{}, nil, zap.NewNop())

	err := probes.EnsureGPSProbe(context.Background(), GPSProbeRequest{
		Carrier: "cmcc", SerialNumber: "SN-NO-GPS", ProductClass: "FAP-TEST", SoftwareVersion: "V1",
		EvidenceVersion: 1,
	})

	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
	require.Nil(t, tasks.created)
}

func TestGPSProbeProjectsIncompleteResponseAsCollectionFailure(t *testing.T) {
	repository := &gpsFlowRepository{}
	probes := NewGPSProbeService(nil, nil, repository, nil, zap.NewNop())
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-INCOMPLETE", EvidenceVersion: 1,
		Paths: []GPSParameterPath{
			{StandardPath: "Device.FAP.GPS.LockedLatitude", PrivatePath: "Device.X.GPS.Lat"},
			{StandardPath: "Device.FAP.GPS.LockedLongitude", PrivatePath: "Device.X.GPS.Lon"},
		},
	})
	require.NoError(t, err)
	result, err := json.Marshal(map[string]any{"private_parameter_values": []tr069.ParameterValueStruct{
		{Name: "Device.X.GPS.Lat", Value: "39.9042"},
	}})
	require.NoError(t, err)

	err = probes.HandleCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), DeviceSN: "SN-INCOMPLETE", Method: "GetParameterValues",
		Source: task.TaskSourceDeviceAccess, AdmissionClass: task.AdmissionClassAccessProbe,
		Status: task.TaskStatusCompleted, Params: params, Result: result,
	})

	require.ErrorIs(t, err, ErrAccessGateDependencyMissing)
	require.Len(t, repository.appended, 1)
	require.Len(t, repository.appended[0].Records, 1)
	require.Equal(t, ConditionTypeGPS, repository.appended[0].Records[0].Type)
	require.Equal(t, EvidenceStatusCollectionFailed, repository.appended[0].Records[0].Status)
}

func TestGPSProbeProjectsTerminalTaskFailureAsCollectionFailure(t *testing.T) {
	repository := &gpsFlowRepository{}
	probes := NewGPSProbeService(nil, nil, repository, nil, zap.NewNop())
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-FAILED", EvidenceVersion: 1,
		NeedTAC: true,
		RadioPaths: []RadioParameterPath{{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
			PrivatePath:  "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC",
			FAPInstance:  1,
			Kind:         radioPathTAC,
		}},
	})
	require.NoError(t, err)

	err = probes.HandleCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), DeviceSN: "SN-FAILED", Method: "GetParameterValues",
		Source: task.TaskSourceDeviceAccess, AdmissionClass: task.AdmissionClassAccessProbe,
		Status: task.TaskStatusExpired, Params: params,
	})

	require.ErrorIs(t, err, ErrAccessGateDependencyMissing)
	require.Len(t, repository.appended, 1)
	require.Equal(t, EvidenceStatusCollectionFailed, repository.appended[0].Records[0].Status)
	require.Equal(t, ConditionTypeTAC, repository.appended[0].Records[0].Type)
}

func TestGPSProbeCarriesForwardExistingEvidenceSnapshot(t *testing.T) {
	now := time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)
	tacJSON, err := json.Marshal([]string{"100"})
	require.NoError(t, err)
	repository := &gpsFlowRepository{context: EvaluationContext{Evidence: EvidenceBatch{
		Carrier: "cmcc", SerialNumber: "SN-SNAPSHOT", Version: 1,
		Records: []EvidenceRecord{{
			Type: ConditionTypeTAC, NormalizedValue: tacJSON, ValueHash: evidenceHash(tacJSON),
			Source: "inform", ObservedAt: now,
		}},
	}}}
	probes := NewGPSProbeService(nil, nil, repository, nil, zap.NewNop())
	probes.SetClock(func() time.Time { return now })
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-SNAPSHOT", EvidenceVersion: 2, NeedGPS: true,
		Paths: []GPSParameterPath{
			{StandardPath: gpsLatitudeStandardPath, PrivatePath: "lat"},
			{StandardPath: gpsLongitudeStandardPath, PrivatePath: "lon"},
		},
	})
	require.NoError(t, err)
	result, err := json.Marshal(map[string]any{"private_parameter_values": []tr069.ParameterValueStruct{
		{Name: "lat", Value: "25.924128"}, {Name: "lon", Value: "115.366448"},
	}})
	require.NoError(t, err)

	err = probes.HandleCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), DeviceSN: "SN-SNAPSHOT", Method: "GetParameterValues",
		Source: task.TaskSourceDeviceAccess, AdmissionClass: task.AdmissionClassAccessProbe,
		Status: task.TaskStatusCompleted, Params: params, Result: result,
	})

	require.ErrorIs(t, err, ErrAccessGateDependencyMissing)
	require.Len(t, repository.appended, 1)
	require.Len(t, repository.appended[0].Records, 2)
	require.Equal(t, []string{"100"}, evidenceTexts(t, repository.appended[0], ConditionTypeTAC))
}

func TestAccessGateDoesNotTrustInformAsCompleteCellInventory(t *testing.T) {
	now := time.Date(2026, 8, 4, 11, 0, 0, 0, time.UTC)
	repository := &gpsFlowRepository{}
	policy := radioAccessPolicy(false)
	gate := NewAccessGate(repository, accessGateAssetStub{
		evidence: AssetEvidence{Source: AssetEvidenceSourceDevice, DeviceID: ptrUUID(uuid.New())},
	}, accessGatePolicyStub{policy: policy})
	gate.SetClock(func() time.Time { return now })

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-RADIO-INFORM", OUI: "48BF74", EventID: "inform-radio",
		Authenticated: true, CarrierIdentityResolved: true,
		Inform: &tr069.InformMessage{ParameterList: servingCellResponseValues()},
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionCollecting, decision.State)
	require.Empty(t, repository.appended)
}

func TestAccessProbeFakeBMMultiCellCompletesInformGPVDecisionFlow(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	paramModelID := uuid.New()
	mappings := []parammodel.ParamMapping{
		{StandardPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLatitude", PrivatePath: "Device.FAP.GPS.LockedLatitude"},
		{StandardPath: "Device.DeviceInfo.SAS.FAP.GPS.LockedLongitude", PrivatePath: "Device.FAP.GPS.LockedLongitude"},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC", PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.TAC"},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID", PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.PLMNID"},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary", PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.IsPrimary"},
		{StandardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity", PrivatePath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Common.CellIdentity"},
	}
	paths := NewProductGPSPathResolver(
		gpsProductStub{product: &product.Product{ID: uuid.New(), ParamModelID: &paramModelID}},
		gpsTranslatorStub{translator: parammodel.NewTranslator(&parammodel.MappingSet{Mappings: mappings}, nil, nil)},
	)
	repository := &gpsFlowRepository{}
	policy := radioAccessPolicy(true)
	policies := accessGatePolicyStub{policy: policy}
	coordinator := NewReevaluationCoordinator(repository, accessGateAssetStub{
		evidence: AssetEvidence{Source: AssetEvidenceSourceDevice, DeviceID: ptrUUID(uuid.New())},
	}, policies)
	coordinator.SetClock(func() time.Time { return now.Add(time.Minute) })
	tasks := &gpsTaskEnqueuer{}
	probes := NewGPSProbeService(paths, tasks, repository, coordinator, zap.NewNop())
	probes.SetClock(func() time.Time { return now })
	gate := NewAccessGate(repository, accessGateAssetStub{
		evidence: AssetEvidence{Source: AssetEvidenceSourceDevice, DeviceID: ptrUUID(uuid.New())},
	}, policies)
	gate.SetClock(func() time.Time { return now })
	gate.SetGPSProbePlanner(probes)

	decision, err := gate.Admit(context.Background(), devicepkg.AccessObservation{
		Carrier: model.CarrierCMCC, SerialNumber: "SN-BM-RADIO-GPS", ProductClass: "FAP/BU1810",
		SoftwareVersion: "BM_2.0.4", EventID: "inform-radio-gps",
		OUI: "48BF74", Authenticated: true, CarrierIdentityResolved: true,
		Inform: &tr069.InformMessage{ParameterList: []tr069.ParameterValueStruct{
			{Name: "Device.Services.FAPService.1.CellConfig.LTE.RAN.OpState", Value: "true"},
			{Name: "Device.Services.FAPService.2.CellConfig.LTE.RAN.OpState", Value: "true"},
		}},
	})

	require.NoError(t, err)
	require.Equal(t, devicepkg.AccessDecisionCollecting, decision.State)
	require.NotNil(t, tasks.created)
	require.Equal(t, "GetParameterNames", tasks.created.Method)
	var discovery gpsProbeTaskParams
	require.NoError(t, json.Unmarshal(tasks.created.Params, &discovery))
	require.Equal(t, servingCellStandardRoot, discovery.DiscoveryStandardPath)
	discoveryTask := tasks.created
	discoveryTask.Result, err = json.Marshal(map[string]any{"parameter_infos": []tr069.ParameterInfoStruct{
		{Name: "Device.Services.FAPService.1."},
		{Name: "Device.Services.FAPService.2."},
	}})
	require.NoError(t, err)
	discoveryTask.Status = task.TaskStatusCompleted
	require.NoError(t, probes.HandleCompleted(context.Background(), discoveryTask))
	require.NotSame(t, discoveryTask, tasks.created)
	require.Equal(t, "GetParameterValues", tasks.created.Method)
	var params gpsProbeTaskParams
	require.NoError(t, json.Unmarshal(tasks.created.Params, &params))
	require.True(t, params.NeedGPS)
	require.True(t, params.NeedTAC)
	require.True(t, params.NeedECGI)
	require.Len(t, params.RadioPaths, 8)
	require.Len(t, params.Names, 10)

	values := servingCellResponseValues()
	values = append(values,
		tr069.ParameterValueStruct{Name: "Device.FAP.GPS.LockedLatitude", Value: "25924128"},
		tr069.ParameterValueStruct{Name: "Device.FAP.GPS.LockedLongitude", Value: "115366448"},
	)
	tasks.created.Result, err = json.Marshal(map[string]any{"private_parameter_values": values})
	require.NoError(t, err)
	tasks.created.Status = task.TaskStatusCompleted

	require.NoError(t, probes.HandleCompleted(context.Background(), tasks.created))
	require.Len(t, repository.appended, 1)
	require.NotEmpty(t, repository.appended[0].Records)
	require.Equal(t, []string{"100", "101"}, evidenceTexts(t, repository.appended[0], ConditionTypeTAC))
	require.Equal(t, []string{"460-00-1001", "460-00-1002"}, evidenceTexts(t, repository.appended[0], ConditionTypeECGI))
	require.Len(t, repository.decisions, 2)
	require.Equal(t, AccessStateCollecting, repository.decisions[0].Decision.State)
	require.Equal(t, AccessStateAccepted, repository.decisions[1].Decision.State)
}

func TestRadioProbeRejectsPartialMultiCellResponse(t *testing.T) {
	paths := []RadioParameterPath{
		{StandardPath: "tac-1", PrivatePath: "tac-1", FAPInstance: 1, Kind: radioPathTAC},
		{StandardPath: "tac-2", PrivatePath: "tac-2", FAPInstance: 2, Kind: radioPathTAC},
	}

	_, err := normalizeGPVRadioEvidence(paths, []tr069.ParameterValueStruct{
		{Name: "tac-1", Value: "100"},
	}, true, false)

	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
}

func TestRadioProbeRequiresPrimaryPLMNMarker(t *testing.T) {
	paths := []RadioParameterPath{
		{StandardPath: "plmn", PrivatePath: "plmn", FAPInstance: 1, Kind: radioPathPLMN},
		{StandardPath: "primary", PrivatePath: "primary", FAPInstance: 1, Kind: radioPathPLMNPrimary},
		{StandardPath: "eci", PrivatePath: "eci", FAPInstance: 1, Kind: radioPathECI},
	}

	_, err := normalizeGPVRadioEvidence(paths, []tr069.ParameterValueStruct{
		{Name: "plmn", Value: "46000"},
		{Name: "primary", Value: "false"},
		{Name: "eci", Value: "1001"},
	}, false, true)

	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
}

func TestRadioProbeBuildsECGIForEveryServingPLMN(t *testing.T) {
	paths := []RadioParameterPath{
		{
			StandardPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
			PrivatePath:  "serving-plmns", FAPInstance: 1, Kind: radioPathServingPLMNs,
		},
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "eci", FAPInstance: 1, Kind: radioPathECI,
		},
	}

	evidence, err := normalizeGPVRadioEvidence(paths, []tr069.ParameterValueStruct{
		{Name: "serving-plmns", Value: "00101,46000"},
		{Name: "eci", Value: "41473"},
	}, false, true)

	require.NoError(t, err)
	require.Equal(t, []string{"001-01-41473", "460-00-41473"}, evidence.ECGIs)
}

func TestRadioProbeRejectsInvalidServingPLMNList(t *testing.T) {
	paths := []RadioParameterPath{
		{
			StandardPath: "Device.Services.FAPService.1.FAPControl.LTE.Gateway.ExistPlmnidList",
			PrivatePath:  "serving-plmns", FAPInstance: 1, Kind: radioPathServingPLMNs,
		},
		{
			StandardPath: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity",
			PrivatePath:  "eci", FAPInstance: 1, Kind: radioPathECI,
		},
	}

	_, err := normalizeGPVRadioEvidence(paths, []tr069.ParameterValueStruct{
		{Name: "serving-plmns", Value: "00101,invalid"},
		{Name: "eci", Value: "41473"},
	}, false, true)

	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
}

func TestServingCellDiscoveryFailureProjectsCollectionFailure(t *testing.T) {
	repository := &gpsFlowRepository{}
	probes := NewGPSProbeService(nil, nil, repository, nil, zap.NewNop())
	params, err := json.Marshal(gpsProbeTaskParams{
		Carrier: "cmcc", SerialNumber: "SN-DISCOVERY-FAILED", EvidenceVersion: 1,
		NeedTAC: true, ProductClass: "FAP/BU1810",
		DiscoveryPath: servingCellStandardRoot, DiscoveryStandardPath: servingCellStandardRoot,
		NextLevel: true,
	})
	require.NoError(t, err)
	result, err := json.Marshal(map[string]any{"parameter_infos": []tr069.ParameterInfoStruct{
		{Name: "Device.Services.FAPService.NotAnInstance."},
	}})
	require.NoError(t, err)

	err = probes.HandleCompleted(context.Background(), &task.Task{
		ID: uuid.NewString(), DeviceSN: "SN-DISCOVERY-FAILED", Method: "GetParameterNames",
		Source: task.TaskSourceDeviceAccess, AdmissionClass: task.AdmissionClassAccessProbe,
		Status: task.TaskStatusCompleted, Params: params, Result: result,
	})

	require.ErrorIs(t, err, ErrAccessGateDependencyMissing)
	require.Len(t, repository.appended, 1)
	require.Equal(t, ConditionTypeTAC, repository.appended[0].Records[0].Type)
	require.Equal(t, EvidenceStatusCollectionFailed, repository.appended[0].Records[0].Status)
}

func TestServingCellDiscoveryRejectsInstancesBeyondBound(t *testing.T) {
	result, err := json.Marshal(map[string]any{"parameter_infos": []tr069.ParameterInfoStruct{
		{Name: "Device.Services.FAPService.1."},
		{Name: "Device.Services.FAPService.17."},
	}})
	require.NoError(t, err)

	_, err = servingCellInstancesFromDiscovery(servingCellStandardRoot, result)

	require.ErrorIs(t, err, ErrAccessEvidenceUnavailable)
}

func radioAccessPolicy(withGPS bool) CompiledPolicy {
	conditions := []CompiledCondition{
		{ID: "tac", Type: ConditionTypeTAC, Operator: ConditionOperatorIn, ExpectedAny: []string{"100", "101"}, Required: true},
		{ID: "ecgi", Type: ConditionTypeECGI, Operator: ConditionOperatorIn, ExpectedAny: []string{"460-00-1001", "460-00-1002"}, Required: true},
	}
	if withGPS {
		conditions = append(conditions, CompiledCondition{
			ID: "gps", Type: ConditionTypeGPS, Operator: ConditionOperatorWithinRadius,
			GeoFence: &GeoFence{Center: GeoPoint{Latitude: 25.924128, Longitude: 115.366448}, RadiusMeters: 100},
			Required: true,
		})
	}
	return CompiledPolicy{Rules: []CompiledRule{{
		ID: "radio-rule", Enabled: true, SerialScope: SerialScope{Type: SerialScopeAll}, Conditions: conditions,
	}}}
}

func servingCellResponseValues() []tr069.ParameterValueStruct {
	return []tr069.ParameterValueStruct{
		{Name: "Device.Services.FAPService.2.CellConfig.LTE.EPC.TAC", Value: "101"},
		{Name: "Device.Services.FAPService.1.CellConfig.LTE.EPC.TAC", Value: "100"},
		{Name: "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID", Value: "46000"},
		{Name: "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.IsPrimary", Value: "true"},
		{Name: "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", Value: "1001"},
		{Name: "Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.1.PLMNID", Value: "46000"},
		{Name: "Device.Services.FAPService.2.CellConfig.LTE.EPC.PLMNList.1.IsPrimary", Value: "true"},
		{Name: "Device.Services.FAPService.2.CellConfig.LTE.RAN.Common.CellIdentity", Value: "1002"},
	}
}

func evidenceTexts(t *testing.T, batch EvidenceBatch, evidenceType ConditionType) []string {
	t.Helper()
	for _, record := range batch.Records {
		if record.Type != evidenceType {
			continue
		}
		values, ok := normalizedTexts(record.NormalizedValue)
		require.True(t, ok)
		return values
	}
	t.Fatalf("evidence %s not found", evidenceType)
	return nil
}

func evidenceRecordByType(t *testing.T, batch EvidenceBatch, evidenceType ConditionType) EvidenceRecord {
	t.Helper()
	for _, record := range batch.Records {
		if record.Type == evidenceType {
			return record
		}
	}
	t.Fatalf("evidence %s not found", evidenceType)
	return EvidenceRecord{}
}
