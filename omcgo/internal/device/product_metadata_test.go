package device

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/product"
)

// stubProductMatcher 是 ProductClassMatcher 的最小测试 stub。
type stubProductMatcher struct {
	res *product.MatchResult
	err error
}

func (s *stubProductMatcher) MatchProductClass(_ context.Context, _ string) (*product.MatchResult, error) {
	return s.res, s.err
}

// TestApplyProductMetadata_BackfillsModelName 覆盖 happy path：
//   - productMatcher 已注入
//   - device.ModelName 为空
//   - MatchProductClass 命中 → device.ModelName = product.Name
func TestApplyProductMetadata_BackfillsModelName(t *testing.T) {
	matcher := &stubProductMatcher{
		res: &product.MatchResult{
			Product: &product.Product{ID: uuid.New(), Name: "mBS31001"},
		},
	}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{
		SerialNumber: "SN001",
		ProductClass: "FAP/mBS31001/SC",
		ModelName:    "",
	}
	svc.applyProductMetadata(context.Background(), device)

	assert.Equal(t, "mBS31001", device.ModelName)
}

// TestApplyProductMetadata_DoesNotOverwriteExisting 验证 helper 不会覆盖已有
// ModelName（保留手填值或前次回填值的尊重）。
func TestApplyProductMetadata_DoesNotOverwriteExisting(t *testing.T) {
	matcher := &stubProductMatcher{
		res: &product.MatchResult{
			Product: &product.Product{Name: "mBS31001"},
		},
	}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{
		SerialNumber: "SN001",
		ProductClass: "FAP/mBS31001/SC",
		ModelName:    "ManuallyEdited",
	}
	svc.applyProductMetadata(context.Background(), device)

	assert.Equal(t, "ManuallyEdited", device.ModelName)
}

// TestApplyProductMetadata_OrphanFailsSoft 验证 ErrOrphan 不阻塞 Inform：
// ModelName 保持原值，无 panic / error 抛出。
func TestApplyProductMetadata_OrphanFailsSoft(t *testing.T) {
	matcher := &stubProductMatcher{err: product.ErrOrphan}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{
		SerialNumber: "SN001",
		ProductClass: "UNKNOWN",
		ModelName:    "",
	}
	svc.applyProductMetadata(context.Background(), device)

	assert.Equal(t, "", device.ModelName)
}

// TestApplyProductMetadata_RegistryErrorFailsSoft 验证 Registry IO 错误同样 soft-fail。
func TestApplyProductMetadata_RegistryErrorFailsSoft(t *testing.T) {
	matcher := &stubProductMatcher{err: errors.New("db connection refused")}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{
		SerialNumber: "SN001",
		ProductClass: "FAP/mBS31001/SC",
		ModelName:    "",
	}
	svc.applyProductMetadata(context.Background(), device)

	assert.Equal(t, "", device.ModelName)
}

// TestApplyProductMetadata_NoMatcherIsNoop 验证未注入 matcher 时 helper 安全退出。
func TestApplyProductMetadata_NoMatcherIsNoop(t *testing.T) {
	svc := &DeviceService{productMatcher: nil, logger: zap.NewNop()}
	device := &model.Device{ProductClass: "FAP/mBS31001/SC", ModelName: ""}
	svc.applyProductMetadata(context.Background(), device)
	assert.Equal(t, "", device.ModelName)
}

// TestApplyProductMetadata_EmptyProductClass 验证 ProductClass 为空时不查 matcher。
func TestApplyProductMetadata_EmptyProductClass(t *testing.T) {
	called := false
	matcher := &counterMatcher{inner: &stubProductMatcher{}, called: &called}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{ProductClass: "", ModelName: ""}
	svc.applyProductMetadata(context.Background(), device)

	assert.False(t, called, "MatchProductClass 不应被调用")
	assert.Equal(t, "", device.ModelName)
}

// TestApplyProductMetadata_NilProductInResult 验证 res.Product == nil 时不写入。
func TestApplyProductMetadata_NilProductInResult(t *testing.T) {
	matcher := &stubProductMatcher{res: &product.MatchResult{Product: nil}}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}
	device := &model.Device{ProductClass: "X", ModelName: ""}
	svc.applyProductMetadata(context.Background(), device)
	assert.Equal(t, "", device.ModelName)
}

// TestApplyProductMetadata_EmptyProductName 验证 product.Name 为空时不写入。
func TestApplyProductMetadata_EmptyProductName(t *testing.T) {
	matcher := &stubProductMatcher{
		res: &product.MatchResult{Product: &product.Product{Name: ""}},
	}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}
	device := &model.Device{ProductClass: "X", ModelName: ""}
	svc.applyProductMetadata(context.Background(), device)
	assert.Equal(t, "", device.ModelName)
}

// TestApplyProductMetadata_TechlessProductClearsTechnology 覆盖非无线产品语义：
// 产品已登记但 tech 为空（如核心网 ImsCore）→ 清空设备 Technology，
// 不保留 detectTechnology 兜底推断的 lte（设备列表会错误显示 eNB(LTE)）。
func TestApplyProductMetadata_TechlessProductClearsTechnology(t *testing.T) {
	matcher := &stubProductMatcher{
		res: &product.MatchResult{
			Product: &product.Product{ID: uuid.New(), Name: "ImsCore", Tech: ""},
		},
	}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{
		SerialNumber: "SN001",
		ProductClass: "ImsCore",
		Technology:   model.TechLTE, // 首次 Inform 被 detectTechnology 默认推断
	}
	svc.applyProductMetadata(context.Background(), device)

	assert.Equal(t, model.Technology(""), device.Technology)
}

// TestApplyProductMetadata_TechlessProductKeepsEmptyTechnology 验证已为空的
// Technology 不产生额外写入路径（幂等）。
func TestApplyProductMetadata_TechlessProductKeepsEmptyTechnology(t *testing.T) {
	matcher := &stubProductMatcher{
		res: &product.MatchResult{
			Product: &product.Product{ID: uuid.New(), Name: "ImsCore", Tech: ""},
		},
	}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{ProductClass: "ImsCore", Technology: ""}
	svc.applyProductMetadata(context.Background(), device)

	assert.Equal(t, model.Technology(""), device.Technology)
}

// TestApplyProductMetadata_ProductTechOverridesInferred 验证登记了 tech 的产品
// 仍以字典覆盖错误推断（既有行为不回归）。
func TestApplyProductMetadata_ProductTechOverridesInferred(t *testing.T) {
	matcher := &stubProductMatcher{
		res: &product.MatchResult{
			Product: &product.Product{ID: uuid.New(), Name: "BNQ", Tech: "nr"},
		},
	}
	svc := &DeviceService{productMatcher: matcher, logger: zap.NewNop()}

	device := &model.Device{ProductClass: "FAP/BSC7041C243", Technology: model.TechLTE}
	svc.applyProductMetadata(context.Background(), device)

	assert.Equal(t, model.TechNR, device.Technology)
}

// counterMatcher 包装 stub 用于断言 MatchProductClass 是否被调用过。
type counterMatcher struct {
	inner  ProductClassMatcher
	called *bool
}

func (m *counterMatcher) MatchProductClass(ctx context.Context, productClass string) (*product.MatchResult, error) {
	*m.called = true
	return m.inner.MatchProductClass(ctx, productClass)
}
