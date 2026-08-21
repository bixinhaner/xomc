package definition

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"os"
	"path/filepath"
	"strconv"
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
	require.Contains(t, modelsByNeType, "UPS")

	for _, identifier := range []string{"60001", "60002", "60003", "60004", "60005"} {
		assert.Equal(t, "GSM", identifierOwner[identifier], "2G 告警 %s 必须归属 GSM 库", identifier)
	}
	for i := 42000; i <= 42018; i++ {
		identifier := strconv.Itoa(i)
		assert.Equal(t, "UPS", identifierOwner[identifier], "UPS 告警 %s 必须归属 UPS 库", identifier)
	}
}

func TestBuiltinAlarmLibraries_UPSDefinitionsMatchReferenceDoc(t *testing.T) {
	path := filepath.Join("..", "..", "..", "data", "alarm-definitions", "UPS.xml")
	upsModel := readBuiltinAlarmModel(t, path)
	require.Equal(t, "UPS", upsModel.NeType)
	require.Equal(t, "5", upsModel.DeviceType)
	require.Equal(t, 19, upsModel.TotalCount)
	require.Len(t, upsModel.Alarms, 19)

	type expectedUPSAlarm struct {
		enName       string
		cnName       string
		cnSuggestion string
	}
	expected := map[string]expectedUPSAlarm{
		"42000": {"AC Power Off", "市电掉电", "检查 UPS current，>20mA 时认为 AC 掉电"},
		"42001": {"Sensor fault", "LTC 故障", "检查 UPS current"},
		"42002": {"DC Power Off", "直流无输出", "检查 AC，电压 >85V 时认为 DC 故障"},
		"42003": {"Charger Over Temperature", "电源过温", "检查 Charger 温度"},
		"42004": {"Charger Over Current", "电源过流", "检查 charger current，>20A"},
		"42005": {"Battery Under Temperature", "低温保护", "电池温度 < -20"},
		"42006": {"Over Temperature Charging", "高温充电保护", "电池温度 > 50"},
		"42007": {"Over Temperature DisCharging", "高温放电保护", "电池温度 > 55"},
		"42008": {"Cell malfunction", "电芯故障", "检查电芯 min/max 电压"},
		"42009": {"Battery over discharge alarm voltage", "Battery over discharge alarm voltage", ""},
		"42010": {"Battery Communication Error", "电池通信故障", "检查通信线缆"},
		"42011": {"Alarm When Door Opened", "机柜门开", "检查机柜"},
		"42012": {"Cabin was soggy", "机柜进水", "检查机柜"},
		"42013": {"Smoking detected in cabin", "机柜烟雾", "有物体燃烧"},
		"42014": {"SPD Out", "防雷失效", "雷击"},
		"42015": {"Fan out work", "风扇故障", "线缆/风扇损坏"},
		"42016": {"Inverter overload", "逆变器过载", "负载过大"},
		"42017": {"High temperature", "高温", "温度 > 60"},
		"42018": {"Low temperature", "低温", "温度 < -5"},
	}

	for _, alarm := range upsModel.Alarms {
		want, ok := expected[alarm.Identifier]
		require.Truef(t, ok, "unexpected UPS alarm identifier %s", alarm.Identifier)
		assert.Equal(t, "Major", alarm.Severity, "UPS 告警 %s severity 必须为 Major", alarm.Identifier)
		assert.Equal(t, "30003", alarm.EventType, "UPS 告警 %s eventType 必须为设备类 30003", alarm.Identifier)
		assert.Equal(t, want.enName, alarm.EnName, "UPS 告警 %s 英文名必须匹配旧系统文档", alarm.Identifier)
		assert.Equal(t, want.cnName, alarm.CnName, "UPS 告警 %s 中文名必须匹配旧系统文档", alarm.Identifier)
		assert.Equal(t, want.cnName, alarm.CnProbableCause, "UPS 告警 %s 当前系统展示名称口径必须稳定", alarm.Identifier)
		assert.Equal(t, want.enName, alarm.EnProbableCause, "UPS 告警 %s 英文 probable cause 必须稳定", alarm.Identifier)
		assert.Equal(t, want.cnSuggestion, alarm.CnSuggestion, "UPS 告警 %s 触发判定必须匹配旧系统文档", alarm.Identifier)
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
