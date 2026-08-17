package parammodel

import (
	"encoding/csv"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type expectedReferenceObject struct {
	privatePath  string
	standardPath string
}

func TestWritableMultiInstanceObjectsMatchDeviceReferences(t *testing.T) {
	referenceFiles := map[string]string{
		"BLN": "LTE全量参数集.csv", "BLQ": "LTE全量参数集.csv",
		"BM":              "BM全量参数集.csv",
		"ENB_DEFAULT_098": "LTE全量参数集.csv", "ENB_DEFAULT_181": "LTE全量参数集.csv",
		"MLN": "LTE全量参数集.csv", "MLQ": "LTE全量参数集.csv",
		"BaiBNQ": "NR全量参数集.csv",
	}
	models := map[string][]expectedReferenceObject{
		"BLN": {
			sameReferenceObject("Device.FAP.Ipsec.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."),
			{privatePath: "Device.Services.FAPService.MmePoolConfigParam.{i}.", standardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}."},
		},
		"BLQ": {
			sameReferenceObject("Device.FAP.Ipsec.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."),
		},
		"BM": {
			sameReferenceObject("Device.FAP.Ipsec.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."),
		},
		"ENB_DEFAULT_098": {
			{privatePath: "InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}.", standardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}."},
			{privatePath: "InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}.", standardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}."},
			{privatePath: "InternetGatewayDevice.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}.", standardPath: "Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."},
		},
		"ENB_DEFAULT_181": {
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."),
		},
		"MLN": {
			sameReferenceObject("Device.FAP.Ipsec.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.5GCell.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."),
		},
		"MLQ": {
			sameReferenceObject("Device.FAP.Ipsec.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.EPC.PLMNList.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.MmePoolConfigParam.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.MocnConfigParam.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.LTE.RAN.NeighborList.LTECell.{i}."),
		},
		"BaiBNQ": {
			sameReferenceObject("Device.Ethernet.Interface.{i}.IPv4Address.{i}."),
			sameReferenceObject("Device.Ethernet.Interface.{i}.IPv6Address.{i}."),
			sameReferenceObject("Device.Ethernet.Interface.{i}.PppoeAddress.{i}."),
			sameReferenceObject("Device.Ethernet.Interface.{i}.VlanInterface.{i}.VlanPppoeAddress.{i}."),
			sameReferenceObject("Device.Ethernet.IpRoute.{i}."),
			sameReferenceObject("Device.FAP.Ipsec.{i}."),
			{privatePath: "Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.EUTRA.Carrier.{i}.", standardPath: "Device.Services.FAPService.{i}.CellConfig.{i}.LTE.RAN.Mobility.IdleMode.InterFreq.Carrier.{i}."},
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.InterFreq.Carrier.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.IdleMode.EUTRA.Carrier.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.QOS.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.FAPControl.NR.AMFPoolConfigParam.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.FAPControl.NR.DscpList.{i}."),
			sameReferenceObject("Device.Services.FAPService.{i}.FAPControl.NR.XnIpAddrMapInfo.{i}."),
		},
	}

	baseDir := filepath.Join("..", "..", "..", "data", "param-mappings")
	deliveryDir := filepath.Join("..", "..", "..", "docs", "param-model-delivery", "xml", "tr069-param-mapping")
	for model, expectedObjects := range models {
		model := model
		expectedObjects := expectedObjects
		t.Run(model, func(t *testing.T) {
			referencePaths := loadDynamicReferencePaths(t, filepath.Join(baseDir, "references", referenceFiles[model]))
			for _, expected := range expectedObjects {
				require.True(t, referenceSupportsObject(referencePaths, expected),
					"%s device reference must declare dynamic object %s", model, expected.standardPath)
			}

			for _, dir := range []string{baseDir, deliveryDir} {
				body, err := os.ReadFile(filepath.Join(dir, model+".xml"))
				require.NoError(t, err)

				var doc xmlParameterModel
				require.NoError(t, xml.Unmarshal(body, &doc))
				objects := make(map[string]xmlParamEntry, len(doc.Objects))
				for _, object := range doc.Objects {
					objects[object.StandardPath] = object
				}

				for _, expected := range expectedObjects {
					object, ok := objects[expected.standardPath]
					if !assert.True(t, ok, "%s must declare writable instance object %s", model, expected.standardPath) {
						continue
					}
					assert.Equal(t, expected.privatePath, object.Name, expected.standardPath)
					assert.Equal(t, "READ_WRITE", object.Access, expected.standardPath)
				}
			}
		})
	}
}

func sameReferenceObject(path string) expectedReferenceObject {
	return expectedReferenceObject{privatePath: path, standardPath: path}
}

func loadDynamicReferencePaths(t *testing.T, path string) []string {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	require.NoError(t, err)
	pathColumn := csvColumnIndex(t, header, "trpath.name")
	minColumn := csvColumnIndex(t, header, "obj.mininstances")
	maxColumn := csvColumnIndex(t, header, "obj.maxinstances")

	var result []string
	for {
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		require.NoError(t, readErr)
		if pathColumn >= len(record) || minColumn >= len(record) || maxColumn >= len(record) {
			continue
		}
		maxInstances, parseErr := strconv.Atoi(strings.TrimSpace(record[maxColumn]))
		if parseErr != nil {
			continue
		}
		minInstances, _ := strconv.Atoi(strings.TrimSpace(record[minColumn]))
		if maxInstances > minInstances {
			result = append(result, normalizeReferencePath(record[pathColumn]))
		}
	}
	return result
}

func referenceSupportsObject(referencePaths []string, expected expectedReferenceObject) bool {
	for _, candidate := range []string{expected.privatePath, expected.standardPath} {
		for _, path := range referencePaths {
			if strings.HasPrefix(path, candidate) {
				return true
			}
		}
	}
	return false
}
