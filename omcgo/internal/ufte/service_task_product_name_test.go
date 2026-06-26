package ufte

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	coremodel "github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/software"
)

// productNameLookupStub: 按 productClass → name 的固定映射模拟 ProductRegistry.MatchProductClass。
type productNameLookupStub struct {
	byClass map[string]string
}

func (p *productNameLookupStub) Lookup(_ context.Context, class string) (string, bool) {
	name, ok := p.byClass[class]
	return name, ok
}

// stubDeviceRepoByID 给 mapTask fallback 路径用 —— 内嵌 stubDeviceRepo（已实现 List）再补 GetByID。
type stubDeviceRepoByID struct {
	stubDeviceRepo
	byID map[uuid.UUID]*coremodel.Device
	err  error
}

func (s *stubDeviceRepoByID) GetByID(_ context.Context, id uuid.UUID) (*coremodel.Device, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.byID[id], nil
}

// 这组测试锁定 #682 修复：UFTE 任务列表「产品名称」列以前直接透传 task.product_class
// （升级类那是 firmware.product_class = "4G eNB" 这种宽泛类别），与设备列表「产品名称」
// （走 ProductRegistry 解析为「甲产品」）口径不一致。mapTask 现在产出 ProductName 字段，
// 双层 lookup：先 task.product_class，miss 时 fallback 任意子任务 device.product_class。

func TestMapTask_ProductName_DirectLookupHit_RollbackPath(t *testing.T) {
	taskID := uuid.New()
	lookup := &productNameLookupStub{byClass: map[string]string{"QAFA": "甲产品"}}
	svc := NewService(nil, nil, nil, nil, &stubDeviceRepo{}, zap.NewNop())
	svc.productNameLookup = lookup.Lookup

	catalog := mustCatalog(t, "VERSION_ROLLBACK")
	task := &software.UpgradeTask{
		ID:           taskID,
		TaskName:     "rollback-task",
		TaskType:     software.TaskTypeRollback,
		ProductClass: "QAFA", // 回滚类 task.product_class 是 device.product_class，直接 lookup 命中
	}
	got, err := svc.mapTask(context.Background(), catalog, task)
	require.NoError(t, err)
	assert.Equal(t, "甲产品", got.ProductName, "task.product_class lookup 命中 → 直接返回产品名")
	assert.Equal(t, "QAFA", got.ProductType, "ProductType 仍保留原 task.product_class（前端老路径兼容）")
}

func TestMapTask_ProductName_FallbackToSubtaskDevice_UpgradePath(t *testing.T) {
	taskID := uuid.New()
	deviceID := uuid.New()
	lookup := &productNameLookupStub{byClass: map[string]string{"QAFA": "甲产品"}}
	subRepo := &stubSubTaskRepo{byTask: map[uuid.UUID][]software.UpgradeSubTaskWithTaskName{
		taskID: {
			{UpgradeSubTask: software.UpgradeSubTask{ID: uuid.New(), TaskID: taskID, DeviceID: deviceID}},
		},
	}}
	devRepo := &stubDeviceRepoByID{byID: map[uuid.UUID]*coremodel.Device{
		deviceID: {ID: deviceID, ProductClass: "QAFA"},
	}}
	svc := NewService(nil, nil, nil, subRepo, devRepo, zap.NewNop())
	svc.productNameLookup = lookup.Lookup

	catalog := mustCatalog(t, "ENB_IMG_UPGRADE")
	task := &software.UpgradeTask{
		ID:           taskID,
		TaskName:     "upgrade-task",
		TaskType:     software.TaskTypeUpgrade,
		ProductClass: "4G eNB", // 升级类 task.product_class 是 firmware.product_class，lookup miss
	}
	got, err := svc.mapTask(context.Background(), catalog, task)
	require.NoError(t, err)
	assert.Equal(t, "甲产品", got.ProductName, "task.product_class lookup miss → fallback 子任务设备 productClass 再解析命中")
	assert.Equal(t, "4G eNB", got.ProductType, "ProductType 不变（前端 fallback 链已有 productType）")
}

func TestMapTask_ProductName_BothPathsMiss_ReturnsEmpty(t *testing.T) {
	taskID := uuid.New()
	deviceID := uuid.New()
	lookup := &productNameLookupStub{byClass: map[string]string{}} // 全 miss
	subRepo := &stubSubTaskRepo{byTask: map[uuid.UUID][]software.UpgradeSubTaskWithTaskName{
		taskID: {
			{UpgradeSubTask: software.UpgradeSubTask{ID: uuid.New(), TaskID: taskID, DeviceID: deviceID}},
		},
	}}
	devRepo := &stubDeviceRepoByID{byID: map[uuid.UUID]*coremodel.Device{
		deviceID: {ID: deviceID, ProductClass: "UNKNOWN"},
	}}
	svc := NewService(nil, nil, nil, subRepo, devRepo, zap.NewNop())
	svc.productNameLookup = lookup.Lookup

	catalog := mustCatalog(t, "ENB_IMG_UPGRADE")
	task := &software.UpgradeTask{
		ID:           taskID,
		TaskName:     "upgrade-task",
		TaskType:     software.TaskTypeUpgrade,
		ProductClass: "4G eNB",
	}
	got, err := svc.mapTask(context.Background(), catalog, task)
	require.NoError(t, err)
	assert.Empty(t, got.ProductName, "两条路径都 miss → ProductName 留空（前端 fallback productType）")
	assert.Equal(t, "4G eNB", got.ProductType)
}

func TestMapTask_ProductName_LookupNotInjected_ReturnsEmpty(t *testing.T) {
	svc := NewService(nil, nil, nil, nil, &stubDeviceRepo{}, zap.NewNop()) // productNameLookup nil

	catalog := mustCatalog(t, "VERSION_ROLLBACK")
	task := &software.UpgradeTask{
		ID:           uuid.New(),
		TaskName:     "rollback-task",
		TaskType:     software.TaskTypeRollback,
		ProductClass: "QAFA",
	}
	got, err := svc.mapTask(context.Background(), catalog, task)
	require.NoError(t, err)
	assert.Empty(t, got.ProductName, "未注入 productNameLookup → ProductName 留空，不阻断 mapTask")
}

func TestMapTask_ProductName_SubTaskRepoError_FallsThroughEmpty(t *testing.T) {
	lookup := &productNameLookupStub{byClass: map[string]string{}}
	subRepo := &stubSubTaskRepo{err: errors.New("db down")}
	svc := NewService(nil, nil, nil, subRepo, &stubDeviceRepoByID{}, zap.NewNop())
	svc.productNameLookup = lookup.Lookup

	catalog := mustCatalog(t, "ENB_IMG_UPGRADE")
	task := &software.UpgradeTask{
		ID:           uuid.New(),
		TaskType:     software.TaskTypeUpgrade,
		ProductClass: "4G eNB",
	}
	got, err := svc.mapTask(context.Background(), catalog, task)
	require.NoError(t, err, "subTaskRepo 故障不应阻断 mapTask，ProductName 降级为空")
	assert.Empty(t, got.ProductName)
}
