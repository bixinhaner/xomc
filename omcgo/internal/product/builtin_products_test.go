package product

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinProducts_GSMProductsReferenceGSMAlarmLibrary(t *testing.T) {
	path := filepath.Join("..", "..", "data", "param-mappings", "products.xml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc xmlProducts
	require.NoError(t, xml.Unmarshal(raw, &doc))

	productsByName := make(map[string]xmlProduct, len(doc.Products))
	for _, product := range doc.Products {
		productsByName[product.Name] = product
	}

	for _, product := range doc.Products {
		if product.Tech == "2G" {
			assert.Equal(t, "gsm", product.Indicator.DeviceType, "%s 的指标类型应为 gsm", product.Name)
			assert.Equal(t, "GSM", product.Alarm.NeType, "%s 应引用 GSM 告警库", product.Name)
			continue
		}
		assert.NotEqual(t, "GSM", product.Alarm.NeType, "%s 不是 2G 产品，不应引用 GSM 告警库", product.Name)
	}

	for _, name := range []string{"BSC 产品", "BTS 产品"} {
		_, exists := productsByName[name]
		require.True(t, exists, "内置产品 %s 不存在", name)
	}
}
