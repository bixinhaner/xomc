package product

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadBuiltinProductsForTest(t *testing.T) xmlProducts {
	t.Helper()
	path := filepath.Join("..", "..", "data", "param-mappings", "products.xml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc xmlProducts
	require.NoError(t, xml.Unmarshal(raw, &doc))
	return doc
}

func TestBuiltinProducts_GSMProductsReferenceGSMAlarmLibrary(t *testing.T) {
	doc := loadBuiltinProductsForTest(t)

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

	for _, name := range []string{"BSC", "BTS"} {
		_, exists := productsByName[name]
		require.True(t, exists, "内置产品 %s 不存在", name)
	}
}

func TestBuiltinProducts_DisplayMetadataIsEnglishSafe(t *testing.T) {
	doc := loadBuiltinProductsForTest(t)
	han := regexp.MustCompile(`\p{Han}`)

	for _, product := range doc.Products {
		assert.Falsef(t, han.MatchString(product.Name), "product name %q must not contain Chinese characters", product.Name)
		assert.Falsef(t, han.MatchString(product.Vendor), "vendor %q for %q must not contain Chinese characters", product.Vendor, product.Name)
		assert.Falsef(t, han.MatchString(product.Description), "description %q for %q must not contain Chinese characters", product.Description, product.Name)
	}
}

func TestBuiltinProducts_HeaderCountsAndPatternsAreValid(t *testing.T) {
	doc := loadBuiltinProductsForTest(t)
	assert.Len(t, doc.Products, doc.TotalProducts)

	orders := make(map[int]string)
	patternCount := 0
	for _, product := range doc.Products {
		for _, pattern := range product.Patterns {
			patternCount++
			_, err := regexp.Compile(pattern.Value)
			require.NoErrorf(t, err, "product %q pattern %q must compile", product.Name, pattern.Value)

			if previous, exists := orders[pattern.GlobalOrder]; exists {
				t.Fatalf("globalOrder %d reused by %q and %q", pattern.GlobalOrder, previous, product.Name)
			}
			orders[pattern.GlobalOrder] = product.Name
		}
	}
	assert.Equal(t, doc.TotalPatterns, patternCount)
}

func TestBuiltinProducts_ParamModelsReferenceLoadableXML(t *testing.T) {
	doc := loadBuiltinProductsForTest(t)

	type paramModelXML struct {
		XMLName    xml.Name `xml:"parameterModel"`
		ParamModel string   `xml:"paramModel,attr"`
		Objects    []struct {
			Name string `xml:"name,attr"`
		} `xml:"objects>object"`
		Params []struct {
			Name string `xml:"name,attr"`
		} `xml:"parameters>param"`
	}

	for _, product := range doc.Products {
		if product.ParamModel == "" {
			continue
		}

		path := filepath.Join("..", "..", "data", "param-mappings", product.ParamModel+".xml")
		raw, err := os.ReadFile(path)
		require.NoErrorf(t, err, "%s 引用的参数模型文件不存在: %s", product.Name, path)

		var model paramModelXML
		require.NoErrorf(t, xml.Unmarshal(raw, &model), "%s 参数模型 XML 无法解析", product.ParamModel)
		assert.Equalf(t, "parameterModel", model.XMLName.Local, "%s 参数模型根节点必须是 parameterModel", product.ParamModel)
		assert.Equalf(t, product.ParamModel, model.ParamModel, "%s 参数模型 XML 的 paramModel 属性必须和 products.xml 一致", product.Name)
		assert.NotZerof(t, len(model.Objects)+len(model.Params), "%s 参数模型不应为空", product.ParamModel)
	}
}

func TestBuiltinProducts_UPSIsInformOnlyProduct(t *testing.T) {
	doc := loadBuiltinProductsForTest(t)

	var ups *xmlProduct
	for i := range doc.Products {
		if doc.Products[i].Name == "UPS" {
			ups = &doc.Products[i]
			break
		}
	}
	require.NotNil(t, ups, "products.xml 应内置 UPS 产品")
	assert.Equal(t, "UPS", ups.ParamModel, "UPS 产品目录应绑定 UPS 参数模型名称用于标准化展示")
	assert.Equal(t, "false", ups.EnableFileType11, "UPS 不支持 FileType=11 参数模型上传")
	assert.Empty(t, ups.Indicator.DeviceType, "UPS 不接入 KPI 指标 deviceType")
	assert.Empty(t, ups.Indicator.Platform, "UPS 不接入 KPI 指标平台")
	assert.Equal(t, "UPS", ups.Alarm.NeType, "UPS 仍需绑定 UPS 告警库")
	require.Len(t, ups.Patterns, 1)
	assert.Equal(t, "^UPS.*", ups.Patterns[0].Value)
}

func TestBuiltinProducts_BLNProductClassRules(t *testing.T) {
	doc := loadBuiltinProductsForTest(t)

	var bln *xmlProduct
	for i := range doc.Products {
		if doc.Products[i].Name == "BLN" {
			bln = &doc.Products[i]
			break
		}
	}
	require.NotNil(t, bln, "products.xml 应内置 BLN 产品")
	assert.Equal(t, "BLN", bln.ParamModel)
	assert.Equal(t, "enb", bln.Indicator.DeviceType)
	assert.Equal(t, "BLN", bln.Indicator.Platform)
	assert.Equal(t, "ENB", bln.Alarm.NeType)
	require.Len(t, bln.Patterns, 8)

	compiled := make([]*regexp.Regexp, 0, len(bln.Patterns))
	for _, pattern := range bln.Patterns {
		compiled = append(compiled, regexp.MustCompile(pattern.Value))
	}

	for _, productClass := range []string{
		"FAP/pCRB2000/SC",
		"FAP/xxxCRBxxx/SC",
		"FAP/xxxCRBxxx/CA",
		"FAP/xxxCRBxxx/DC",
		"FAP/xxxCRBxxx/TC",
		"FAP/CR-B4860/SC",
		"FAP/CR-B4860/CA",
		"FAP/CR-B4860/DC",
		"FAP/CR-B4860/TC",
	} {
		assert.Truef(t, matchesAny(compiled, productClass), "BLN ProductClass %q 应命中 BLN 规则", productClass)
	}

	for _, productClass := range []string{
		"fap/xxxcrbxxx/sc",
		"FAP/xxxPMBxxx/SC",
		"FAP/MLN/SC",
		"FAP/MLN/CA",
		"FAP/MLN/DC",
	} {
		assert.Falsef(t, matchesAny(compiled, productClass), "非 BLN ProductClass %q 不应命中 BLN 规则", productClass)
	}
}

func TestBuiltinProducts_BLNParamModelXMLMetadataIsConsistent(t *testing.T) {
	type paramModelXML struct {
		TotalEntries int `xml:"totalEntries,attr"`
		Objects      []struct {
			Name         string `xml:"name,attr"`
			StandardPath string `xml:"standardPath,attr"`
		} `xml:"objects>object"`
		Params []struct {
			Name         string `xml:"name,attr"`
			StandardPath string `xml:"standardPath,attr"`
		} `xml:"parameters>param"`
	}

	files := []string{
		filepath.Join("..", "..", "data", "param-mappings", "BLN.xml"),
		filepath.Join("..", "..", "docs", "param-model-delivery", "xml", "tr069-param-mapping", "BLN.xml"),
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			raw, err := os.ReadFile(file)
			require.NoError(t, err)

			var model paramModelXML
			require.NoError(t, xml.Unmarshal(raw, &model))
			assert.Equal(t, len(model.Objects)+len(model.Params), model.TotalEntries)

			for _, object := range model.Objects {
				assert.LessOrEqualf(t, strings.Count(object.Name, "{i}"), strings.Count(object.StandardPath, "{i}"), "%s privatePath has more placeholders than standardPath", object.Name)
			}
			for _, param := range model.Params {
				assert.LessOrEqualf(t, strings.Count(param.Name, "{i}"), strings.Count(param.StandardPath, "{i}"), "%s privatePath has more placeholders than standardPath", param.Name)
			}
		})
	}
}

func matchesAny(patterns []*regexp.Regexp, value string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}
