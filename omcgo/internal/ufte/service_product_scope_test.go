package ufte

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// #492：product_scope（适用产品=产品英文名列表）非空时，deviceMatchesTaskType 走产品目录
// 精确匹配：设备 productClass → productNameLookup → 命中模板.Products 才放行。
func TestDeviceMatchesTaskType_ProductScope(t *testing.T) {
	item := TaskType{
		TypeCode: "ENB_IMG_UPGRADE",
		Products: []string{"BAIBLQ 产品", "BAIQAFA 产品"},
		// 故意给一个会命中旧关键字的 PlatformScope，验证 product_scope 非空时不再走旧口径。
		PlatformScope: []string{"4G eNB"},
	}

	svc := NewService(nil, nil, nil, nil, nil, zap.NewNop())
	svc.productNameLookup = func(_ context.Context, pc string) (string, bool) {
		switch pc {
		case "FAP/BAIBLQ/SC":
			return "BAIBLQ 产品", true
		case "FAP/BAIQAFA/CA":
			return "BAIQAFA 产品", true
		case "FAP/OTHER":
			return "其它产品", true
		default:
			return "", false
		}
	}

	assert.True(t, svc.deviceMatchesTaskType(context.Background(), item, "FAP/BAIBLQ/SC"),
		"产品名命中 product_scope 应放行")
	assert.True(t, svc.deviceMatchesTaskType(context.Background(), item, "FAP/BAIQAFA/CA"),
		"列表内第二个产品名也应放行")
	assert.False(t, svc.deviceMatchesTaskType(context.Background(), item, "FAP/OTHER"),
		"产品名不在 product_scope 列表内不应放行")
	assert.False(t, svc.deviceMatchesTaskType(context.Background(), item, "FAP/UNKNOWN"),
		"productClass 查不到产品名不应放行")
	assert.False(t, svc.deviceMatchesTaskType(context.Background(), item, ""),
		"空 productClass 不应放行")
}

// product_scope 非空但 productNameLookup 未注入 → 不放行（不退回旧 PlatformScope 误放行）。
func TestDeviceMatchesTaskType_ProductScope_NoLookup(t *testing.T) {
	item := TaskType{TypeCode: "ENB_IMG_UPGRADE", Products: []string{"BAIBLQ 产品"}, PlatformScope: []string{"4G eNB"}}
	svc := NewService(nil, nil, nil, nil, nil, zap.NewNop())
	assert.False(t, svc.deviceMatchesTaskType(context.Background(), item, "4G eNB"),
		"product_scope 非空但无 lookup 时不应回退旧 PlatformScope 放行")
}

// product_scope 为空 → 回退旧 PlatformScope/关键字口径（灰度兼容）。
func TestDeviceMatchesTaskType_EmptyProductScope_FallsBack(t *testing.T) {
	item := TaskType{TypeCode: "ENB_IMG_UPGRADE", Products: nil, PlatformScope: []string{"4G eNB", "QAFA"}}
	svc := NewService(nil, nil, nil, nil, nil, zap.NewNop())
	assert.True(t, svc.deviceMatchesTaskType(context.Background(), item, "QAFA"),
		"product_scope 为空时应回退 PlatformScope 子串匹配")
}
