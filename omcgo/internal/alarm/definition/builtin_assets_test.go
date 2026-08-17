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
	require.Len(t, paths, 8, "内置告警库应包含独立的 GSM.xml")

	identifierOwner := make(map[string]string, 443)
	modelsByNeType := make(map[string]xmlAlarmModel, len(paths))
	total := 0

	for _, path := range paths {
		model := readBuiltinAlarmModel(t, path)
		filenameNeType := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		modelsByNeType[model.NeType] = model

		assert.Equal(t, filenameNeType, model.NeType, "%s 的文件名与 neType 必须一致", path)
		assert.Equal(t, model.TotalCount, len(model.Alarms), "%s 的 totalCount 必须与实际条数一致", path)

		for _, alarm := range model.Alarms {
			if owner, exists := identifierOwner[alarm.Identifier]; exists {
				t.Errorf("identifier %s 同时存在于 %s 和 %s", alarm.Identifier, owner, model.NeType)
				continue
			}
			identifierOwner[alarm.Identifier] = model.NeType
			total++
		}

	}

	assert.Equal(t, 443, total, "内置告警定义总数必须与当前数据资产一致")
	assert.Equal(t, 212, modelsByNeType["ENB"].TotalCount)
	gsmModel := modelsByNeType["GSM"]
	assert.Equal(t, "2", gsmModel.DeviceType)
	assert.Equal(t, 5, gsmModel.TotalCount)

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
