package definition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readBuiltinAlarmModel(t *testing.T, path string) xmlAlarmModel {
	t.Helper()

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var model xmlAlarmModel
	require.NoError(t, xml.Unmarshal(raw, &model))
	return model
}

func TestBuiltinAlarmLibraries_GSMDefinitionsAreOwnedByGSM(t *testing.T) {
	dataDir := filepath.Join("..", "..", "..", "data", "alarm-definitions")
	paths, err := filepath.Glob(filepath.Join(dataDir, "*.xml"))
	require.NoError(t, err)
	expectedLibraries := map[string]string{
		"CPE":     "CPE",
		"EGW":     "EGW",
		"ENB":     "ENB",
		"EPC":     "EPC",
		"GNB":     "GNB",
		"GSM":     "GSM",
		"IMSCORE": "IMSCORE",
		"OMC":     "OMC",
		"UPS":     "UPS",
	}
	require.Len(t, paths, len(expectedLibraries), "内置告警库文件集合必须与明确契约一致")

	identifierOwner := make(map[string]string)
	modelsByNeType := make(map[string]xmlAlarmModel, len(paths))
	actualLibraries := make(map[string]string, len(paths))

	for _, path := range paths {
		model := readBuiltinAlarmModel(t, path)
		filenameNeType := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		expectedOwner, exists := expectedLibraries[filenameNeType]
		require.True(t, exists, "发现未登记的内置告警库 %s", filepath.Base(path))
		actualLibraries[filenameNeType] = model.NeType
		modelsByNeType[model.NeType] = model

		assert.Equal(t, expectedOwner, model.NeType, "%s 必须归属已登记的 neType", path)
		assert.Equal(t, model.TotalCount, len(model.Alarms), "%s 的 totalCount 必须与实际条数一致", path)

		for _, alarm := range model.Alarms {
			if owner, exists := identifierOwner[alarm.Identifier]; exists {
				t.Errorf("identifier %s 同时存在于 %s 和 %s", alarm.Identifier, owner, model.NeType)
				continue
			}
			identifierOwner[alarm.Identifier] = model.NeType
		}

	}

	assert.Equal(t, expectedLibraries, actualLibraries, "内置告警库文件及 neType 归属必须与明确契约一致")
	require.Contains(t, modelsByNeType, "GSM")
	require.Contains(t, modelsByNeType, "IMSCORE")

	for _, identifier := range []string{"60001", "60002", "60003", "60004", "60005"} {
		assert.Equal(t, "GSM", identifierOwner[identifier], "2G 告警 %s 必须归属 GSM 库", identifier)
	}
}

func TestBuiltinAlarmLibraries_AcceptUnchangedLegacyENBOnUpgrade(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "alarm-definitions", "GSM.xml")
	gsmModel := readBuiltinAlarmModel(t, path)

	seen := make(map[string]seenAlarmIdentifier, len(gsmModel.Alarms))
	for _, alarm := range gsmModel.Alarms {
		seen[alarm.Identifier] = seenAlarmIdentifier{
			Filename: "ENB.xml",
			NeType:   "ENB",
			Alarm:    alarm,
		}
	}

	require.NoError(t, validateAlarmIdentifiers(gsmModel.Alarms, seen, "GSM.xml", "GSM"))
}

func TestBuiltinAlarmLibraries_GSMPayloadFingerprint(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "alarm-definitions", "GSM.xml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	sum := sha256.Sum256(raw)
	assert.Equal(t,
		"01cbc5a041e2e4714f25948bd2d34680e3d16b78685db682c130ee5d6f1fdcbe",
		hex.EncodeToString(sum[:]),
		"GSM.xml 的五条迁移 payload 不应被无意改写",
	)
}
