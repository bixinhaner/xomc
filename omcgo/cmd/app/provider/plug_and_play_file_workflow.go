package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
	"github.com/omcgo/omcgo/internal/ufte"
)

// plugAndPlayFileWorkflowAdapter reuses the same firmware and license UFTE
// execution services as File Management. Final parameter configuration bypasses
// this adapter and is sent by provisioning as a TR-069 Download.
type plugAndPlayFileWorkflowAdapter struct {
	software *software.SoftwareService
	ufte     *ufte.Service
}

func (a *plugAndPlayFileWorkflowAdapter) ExecuteUpgrade(
	ctx context.Context,
	policyName string,
	targetVersion string,
	preserveSetting bool,
	devices []*model.Device,
	operator string,
) (uuid.UUID, error) {
	if a == nil || a.software == nil || a.ufte == nil {
		return uuid.Nil, fmt.Errorf("%w: unified upgrade workflow is unavailable", commonerrors.ErrInvalidInput)
	}
	if len(devices) == 0 {
		return uuid.Nil, fmt.Errorf("%w: no devices selected", commonerrors.ErrInvalidInput)
	}

	firmwareID, err := a.resolveFirmwareID(ctx, devices[0], targetVersion)
	if err != nil {
		return uuid.Nil, err
	}
	deviceIDs := make([]uuid.UUID, 0, len(devices))
	for _, dev := range devices {
		deviceIDs = append(deviceIDs, dev.ID)
	}

	typeCode := "ENB_IMG_UPGRADE"
	switch software.ResolveDeviceTech(devices[0]) {
	case model.TechNR:
		typeCode = "GNB_IMG_UPGRADE"
	case model.TechGSM:
		typeCode = "GSM_IMG_UPGRADE"
	}
	task, err := a.ufte.CreateTask(ctx, ufte.CreateTaskRequest{
		TaskName:      fmt.Sprintf("即插即用-%s-软件升级", strings.TrimSpace(policyName)),
		TypeCode:      typeCode,
		ProductType:   devices[0].ProductClass,
		FirmwareID:    &firmwareID,
		IsKeepConfig:  preserveSetting,
		DeviceIDs:     deviceIDs,
		DeviceCount:   len(deviceIDs),
		ExecutionMode: "immediate",
		Note:          "由即插即用策略调用统一文件管理升级流程",
	}, operator, nil)
	if err != nil {
		return uuid.Nil, err
	}
	taskID, err := uuid.Parse(task.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse unified upgrade task id %q: %w", task.ID, err)
	}
	return taskID, nil
}

func (a *plugAndPlayFileWorkflowAdapter) ExecuteLicense(
	ctx context.Context,
	policyName string,
	devices []*model.Device,
	operator string,
) (uuid.UUID, error) {
	if a == nil || a.ufte == nil {
		return uuid.Nil, fmt.Errorf("%w: unified license workflow is unavailable", commonerrors.ErrInvalidInput)
	}
	deviceIDs := make([]uuid.UUID, 0, len(devices))
	for _, dev := range devices {
		deviceIDs = append(deviceIDs, dev.ID)
	}
	if len(deviceIDs) == 0 {
		return uuid.Nil, fmt.Errorf("%w: no devices selected", commonerrors.ErrInvalidInput)
	}
	task, err := a.ufte.CreateTask(ctx, ufte.CreateTaskRequest{
		TaskName:      fmt.Sprintf("即插即用-%s-License", strings.TrimSpace(policyName)),
		TypeCode:      "LICENSE_UPGRADE",
		ProductType:   devices[0].ProductClass,
		DeviceIDs:     deviceIDs,
		DeviceCount:   len(deviceIDs),
		ExecutionMode: "immediate",
		Note:          "由即插即用策略调用统一文件管理 License 导入/下发流程",
	}, operator, nil)
	if err != nil {
		return uuid.Nil, err
	}
	taskID, err := uuid.Parse(task.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse unified license task id %q: %w", task.ID, err)
	}
	return taskID, nil
}

func (a *plugAndPlayFileWorkflowAdapter) resolveFirmwareID(
	ctx context.Context,
	dev *model.Device,
	targetVersion string,
) (uuid.UUID, error) {
	fileType := software.FileTypeIMG
	filter := software.FirmwareFilter{
		FileType: &fileType,
		ListRequest: model.ListRequest{
			Page: 1, PageSize: 1000, SortDir: "desc",
		},
	}
	if dev.ProductID != nil {
		filter.ProductID = dev.ProductID.String()
	} else {
		productClass := dev.ProductClass
		filter.ProductClass = &productClass
	}
	result, err := a.software.ListFirmware(ctx, filter)
	if err != nil {
		return uuid.Nil, fmt.Errorf("list firmware versions: %w", err)
	}
	targetVersion = strings.TrimSpace(targetVersion)
	for _, firmware := range result.Items {
		if strings.TrimSpace(firmware.Version) == targetVersion {
			return firmware.ID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("%w: firmware version %q is not available for product %q",
		commonerrors.ErrNotFound, targetVersion, dev.ProductClass)
}
