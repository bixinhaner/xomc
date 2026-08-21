package device

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/carrier"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

var ErrCandidateContainmentUnsupported = errors.New("candidate RF containment is not supported")

// CandidateRFIdentitySnapshot extracts only the RF control surface needed to
// contain an unadmitted endpoint. Reported paths take precedence over model
// defaults, and are persisted with the candidate so retries cannot drift to a
// different parameter family after the originating Inform has ended.
func CandidateRFIdentitySnapshot(observation AccessObservation) (model.Technology, []string) {
	var params []model.DeviceParameter
	var reportedCanonical []string
	var informParams []tr069.ParameterValueStruct
	if observation.Inform != nil {
		informParams = observation.Inform.ParameterList
		for _, parameter := range informParams {
			path := strings.TrimSpace(parameter.Name)
			if carrier.IsRFControlPath(path) {
				params = append(params, model.DeviceParameter{ParameterPath: path, Writable: true})
			} else if IsCandidateRFControlPath(path) {
				reportedCanonical = append(reportedCanonical, path)
			}
		}
	}
	technology := detectTechnologyForInform(
		observation.ProductClass,
		findParamValue(informParams, "Device.DeviceInfo.ModelName"),
		observation.SoftwareVersion,
		informParams,
	)
	paths := carrier.ResolveWritableRFControlPathsForProduct(observation.ProductClass, params)
	if len(paths) == 0 && !carrier.IsMBS31001ProductClass(observation.ProductClass) {
		paths = reportedCanonical
	}
	sort.Strings(paths)
	return technology, paths
}

// QueueCandidateRFSwitch queues the only permitted write for an unadmitted
// candidate: RF containment. Enabling RF through this endpoint is rejected.
func (s *DeviceService) QueueCandidateRFSwitch(
	ctx context.Context,
	target CandidateRFTarget,
	enabled bool,
	options RFSwitchTaskOptions,
) (*task.Task, error) {
	if enabled {
		return nil, fmt.Errorf("candidate RF endpoint cannot enable radio: %w", commonerrors.ErrInvalidInput)
	}
	paths, err := s.resolveCandidateRFControlTargets(target, options.TargetPaths)
	if err != nil {
		return nil, err
	}
	values := make([]map[string]string, 0, len(paths))
	for _, path := range paths {
		values = append(values, map[string]string{"name": path, "value": "0", "type": "xsd:boolean"})
	}
	params, err := json.Marshal(map[string]any{"values": values})
	if err != nil {
		return nil, fmt.Errorf("encode candidate RF containment parameters: %w", err)
	}
	return s.queueCandidateRFTask(ctx, target, "SetParameterValues", params, options)
}

// QueueCandidateRFReadback queues the GPV baseline/verification paired with a
// candidate containment action. No other candidate RPC is exposed.
func (s *DeviceService) QueueCandidateRFReadback(
	ctx context.Context,
	target CandidateRFTarget,
	options RFSwitchTaskOptions,
) (*task.Task, error) {
	paths, err := s.resolveCandidateRFControlTargets(target, options.TargetPaths)
	if err != nil {
		return nil, err
	}
	params, err := json.Marshal(map[string]any{"names": paths})
	if err != nil {
		return nil, fmt.Errorf("encode candidate RF readback parameters: %w", err)
	}
	return s.queueCandidateRFTask(ctx, target, "GetParameterValues", params, options)
}

func (s *DeviceService) resolveCandidateRFControlTargets(
	target CandidateRFTarget,
	requested []string,
) ([]string, error) {
	if target.CandidateID == uuid.Nil || strings.TrimSpace(target.SerialNumber) == "" || target.Carrier == "" {
		return nil, fmt.Errorf("candidate RF target identity is incomplete: %w", commonerrors.ErrInvalidInput)
	}
	if s.taskSvc == nil || s.carrierRegistry == nil {
		return nil, fmt.Errorf("candidate RF dispatcher is not configured")
	}
	adapter, err := s.carrierRegistry.Get(target.Carrier)
	if err != nil {
		return nil, fmt.Errorf("resolve candidate carrier=%s for RF control: %w", target.Carrier, err)
	}
	canonicalPath := adapter.RFControlPath(target.Technology)
	paths := append([]string(nil), target.RFControlPaths...)
	if len(paths) == 0 {
		if carrier.IsMBS31001ProductClass(target.ProductClass) {
			return nil, fmt.Errorf("candidate mBS31001 did not report Device.DeviceInfo.SAS.RadioEnable: %w", ErrCandidateContainmentUnsupported)
		}
		if canonicalPath == "" {
			return nil, fmt.Errorf("carrier=%s does not support candidate RF control for technology=%s: %w",
				target.Carrier, target.Technology, ErrCandidateContainmentUnsupported)
		}
		paths = []string{canonicalPath}
	}
	for _, path := range paths {
		if path != canonicalPath && !carrier.IsRFControlPath(path) && !IsCandidateRFControlPath(path) {
			return nil, fmt.Errorf("candidate RF path %q is not permitted: %w", path, ErrCandidateContainmentUnsupported)
		}
	}
	if len(requested) > 0 {
		return pinRFControlTargets(paths, requested)
	}
	sort.Strings(paths)
	return paths, nil
}

// IsCandidateRFControlPath recognizes the standard AdminState family used by
// carrier adapters without changing the broader geofence path resolver.
func IsCandidateRFControlPath(path string) bool {
	return strings.HasPrefix(path, "Device.Services.FAPService.") &&
		(strings.HasSuffix(path, ".FAPControl.LTE.AdminState") ||
			strings.HasSuffix(path, ".FAPControl.NR.AdminState"))
}

func (s *DeviceService) queueCandidateRFTask(
	ctx context.Context,
	target CandidateRFTarget,
	method string,
	params json.RawMessage,
	options RFSwitchTaskOptions,
) (*task.Task, error) {
	if options.Source != task.TaskSourceDeviceAccess ||
		options.AdmissionClass != task.AdmissionClassSecurityAction ||
		strings.TrimSpace(options.SourceID) == "" || strings.TrimSpace(options.CommandKey) == "" {
		return nil, fmt.Errorf("candidate RF task requires an audited security action: %w", commonerrors.ErrInvalidInput)
	}
	created, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
		DeviceSN: target.SerialNumber, Method: method, Params: params, Priority: 5,
		CommandKey: options.CommandKey, Source: options.Source, SourceID: options.SourceID,
		CreatorID: options.CreatorID, Description: options.Description,
		AdmissionClass: options.AdmissionClass,
	})
	if err != nil {
		return nil, fmt.Errorf("queue candidate RF security task: %w", err)
	}
	return created, nil
}
