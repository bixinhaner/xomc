package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/omcgo/omcgo/internal/provision"
)

func p(id, path, value string) provision.ResolvedParameter {
	return provision.ResolvedParameter{ParameterID: id, TRPath: path, Value: value, Source: "test-data"}
}

func gsmParameters() []provision.ResolvedParameter {
	return []provision.ResolvedParameter{
		p("IPA + Unit ID", "Device.Services.GsmBTSCellDT.1.IPAUnitID", "6969-1"),
		p("Remote IP", "Device.Services.GsmBTSCellDT.1.OMLRemoteIP", "192.0.2.10"),
		p("Bind IP", "Device.Services.GsmBTSCellDT.1.GsmBtsBindMib", "192.0.2.20"),
		p("WAN IP", "Device.DeviceInfo.WAN_CONFIG1_IPADDR", "192.0.2.20"),
		p("Synchronization", "Device.FAP.Synchronization.PpsTimeMode", "1"),
		p("OMC", "Device.ManagementServer.URL", "http://192.0.2.100:7547"),
	}
}

func lteParameters() []provision.ResolvedParameter {
	return []provision.ResolvedParameter{
		p("BAND", "Device.Services.FAPService.1.Capabilities.LTE.BandsSupported", "42"),
		p("Bandwidth", "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth", "100"),
		p("EARFCN", "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLEARFCN", "43000"),
		p("SubFrameAssignment", "Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SubFrameAssignment", "2"),
		p("SpecialSubframePatterns", "Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns", "7"),
		p("ECI", "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.CellIdentity", "222218"),
		p("TAC", "Device.Services.FAPService.1.CellConfig.LTE.RAN.Common.TAC", "100"),
		p("PCI", "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.PhyCellID", "218"),
		p("RootSequenceIndex", "Device.Services.FAPService.1.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex", "30"),
		p("Power", "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.X_COM_MaxTxPowerExpanded", "40"),
		p("PLMN", "Device.Services.FAPService.1.CellConfig.LTE.EPC.PLMNList.1.PLMNID", "46000"),
		p("MME IP", "Device.Services.FAPService.CellConfig.LTE.MmePoolConfigParam.1.MMEIp1", "198.51.100.10"),
		p("WAN IP", "Device.DeviceInfo.WAN_CONFIG1_IPADDR", "192.0.2.30"),
		p("Synchronization", "Device.ManagementServer.tfcsManagerPrimsrc", "3"),
		p("OMC", "Device.ManagementServer.URL", "http://192.0.2.100:7547"),
		p("TunnelEnable", "Device.FAP.Ipsec.1.TUNNEL_ENABLE", "1"),
		p("Gateway", "Device.FAP.Ipsec.1.TUNNEL_GATEWAY", "198.51.100.20"),
		p("leftAuth", "Device.FAP.Ipsec.1.LEFT_AUTH", "psk"),
		p("rightAuth", "Device.FAP.Ipsec.1.RIGHT_AUTH", "psk"),
		p("Right Subnet", "Device.FAP.Ipsec.1.RIGHT_SUBNET", "10.20.0.0/24"),
		p("leftId", "Device.FAP.Ipsec.1.LEFT_IDENTIFIER", "lte-test-left"),
		p("rightId", "Device.FAP.Ipsec.1.RIGHT_IDENTIFIER", "lte-test-right"),
		p("leftSourceIp", "Device.FAP.Ipsec.1.LEFTSOURCEIP", "192.0.2.30"),
		p("leftSubnet", "Device.FAP.Ipsec.1.LEFT_SUBNET", "10.10.0.0/24"),
		p("fragmentation", "Device.FAP.Ipsec.1.FRAGMENTATION", "yes"),
		p("IKE Encryption", "Device.FAP.Ipsec.1.IKE_ENCRYPTION", "aes256"),
		p("IKE DH Group", "Device.FAP.Ipsec.1.IKE_DH_GROUP", "modp2048"),
		p("IKE Authentication", "Device.FAP.Ipsec.1.IKE_AUTHENTICATION", "sha256"),
		p("ESP Encryption", "Device.FAP.Ipsec.1.ESP_ENCRYPTION", "aes256"),
		p("ESP DH Group", "Device.FAP.Ipsec.1.ESP_DH_GROUP", "modp2048"),
		p("ESP Authentication", "Device.FAP.Ipsec.1.ESP_AUTHENTICATION", "sha256"),
		p("KeyLife", "Device.FAP.Ipsec.1.KEYLIFE", "3600"),
		p("IKELifeTime", "Device.FAP.Ipsec.1.IKELIFETIME", "86400"),
		p("RekeyMargin", "Device.FAP.Ipsec.1.REKEYMARGIN", "540"),
		p("Dpdaction", "Device.FAP.Ipsec.1.DPDACTION", "restart"),
		p("Dpddelay", "Device.FAP.Ipsec.1.DPDDELAY", "30"),
		p("Left Interface", "Device.FAP.Ipsec.1.LEFT_INTERFACE", "eth0"),
		p("Forceencaps", "Device.FAP.Ipsec.1.FORCEENCAPS", "yes"),
		p("NTP", "Device.Time.NTPServer1", "203.0.113.123"),
		p("Local Time Zone", "Device.Time.LocalTimeZoneName", "Asia/Shanghai"),
	}
}

func nrParameters() []provision.ResolvedParameter {
	return []provision.ResolvedParameter{
		p("BAND", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoDLSIB.MultiFrequencyBandListNRSIB.1.FreqBandIndicatorNR", "78"),
		p("NRARFCNUL", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.NRARFCNUL", "636666"),
		p("NRARFCNDL", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.NRARFCNDL", "636666"),
		p("UL Subcarrier Spacing", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoULSIB.ScsSpecificCarrierList.1.SCSSpecificCarrier.SubcarrierSpacing", "1"),
		p("DL Subcarrier Spacing", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoDLSIB.ScsSpecificCarrierList.1.SCSSpecificCarrier.SubcarrierSpacing", "1"),
		p("UL Bandwidth", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoULSIB.ScsSpecificCarrierList.1.SCSSpecificCarrier.CarrierBandwidth", "273"),
		p("DL Bandwidth", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoDLSIB.ScsSpecificCarrierList.1.SCSSpecificCarrier.CarrierBandwidth", "273"),
		p("SSB Frequency", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.SsbFrequency", "633984"),
		p("Power", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PowerModify", "20"),
		p("PLMN", "Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.PLMNList.1.PLMNID", "46000"),
		p("TAC", "Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.TAC", "100"),
		p("PCI", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.RF.PhyCellID", "321"),
		p("Root Sequence", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.BWP.BWPUL.1.RACHConfigGeneric.RootSequenceIndex", "30"),
		p("NGU Local Address", "Device.FAP.NguIpBind1.BindInterface", "192.0.2.40"),
		p("AMF IP", "Device.Services.FAPService.1.FAPControl.NR.AMFPoolConfigParam.1.AmfIP1", "198.51.100.30"),
		p("WAN IP", "Device.Ethernet.Interface.1.IPv4Address.1.IPAddress", "192.0.2.40"),
		p("OffsetToPointA", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.FrequencyInfoDLSIB.OffsetToPointA", "24"),
		p("SSB Subcarrier Offset", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.SSB.SsbSubcarrierOffset", "0"),
		p("TDD Reference SCS", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.ReferenceSubcarrierSpacing", "1"),
		p("Pattern1 Periodicity", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern1.DlULTransmissionPeriodicity", "5"),
		p("Pattern1 DL Slots", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern1.NrofDownlinkSlots", "7"),
		p("Pattern1 DL Symbols", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern1.NrofDownlinkSymbols", "6"),
		p("Pattern1 UL Slots", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern1.NrofUplinkSlots", "2"),
		p("Pattern1 UL Symbols", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern1.NrofUplinkSymbols", "4"),
		p("Pattern2 Periodicity", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern2.DlULTransmissionPeriodicity", "5"),
		p("Pattern2 DL Slots", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern2.NrofDownlinkSlots", "7"),
		p("Pattern2 DL Symbols", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern2.NrofDownlinkSymbols", "6"),
		p("Pattern2 UL Slots", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern2.NrofUplinkSlots", "2"),
		p("Pattern2 UL Symbols", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.TddULDLConfigurationCommon.pattern2.NrofUplinkSymbols", "4"),
		p("Synchronization", "Device.FAP.Synchronization.PpsTimeMode", "GPS_PPS"),
		p("OMC", "Device.ManagementServer.URL", "http://192.0.2.100:7547"),
		p("NTP", "Device.Time.NTPServer1", "203.0.113.123"),
		p("Local Time Zone", "Device.Time.LocalTimeZoneName", "Asia/Shanghai"),
		p("gNB Name", "Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBName", "NR-TEST-SITE"),
		p("gNB ID", "Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBId", "6001"),
		p("gNB ID Length", "Device.Services.FAPService.1.FAPControl.NR.RAN.Common.gNBIdLength", "24"),
		p("DL Antenna Count", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.Antenna.NumOfTxAntenna", "4"),
		p("UL Antenna Count", "Device.Services.FAPService.1.CellConfig.1.NR.RAN.PHY.Antenna.NumOfRxAntenna", "4"),
		p("NCI", "Device.Services.FAPService.1.CellConfig.1.NR.CN.TA.1.NrcellIdentity", "393281"),
		p("TunnelEnable", "Device.FAP.Ipsec.1.TUNNEL_ENABLE", "1"),
		p("Gateway", "Device.FAP.Ipsec.1.TUNNEL_GATEWAY", "198.51.100.40"),
		p("leftAuth", "Device.FAP.Ipsec.1.TUNNEL_LEFT_AUTH", "psk"),
		p("rightAuth", "Device.FAP.Ipsec.1.TUNNEL_RIGHT_AUTH", "psk"),
		p("Right Subnet", "Device.FAP.Ipsec.1.RIGHT_SUBNET", "10.40.0.0/24"),
		p("leftId", "Device.FAP.Ipsec.1.LEFT_IDENTIFIER", "nr-test-left"),
		p("rightId", "Device.FAP.Ipsec.1.RIGHT_IDENTIFIER", "nr-test-right"),
		p("leftSourceIp", "Device.FAP.Ipsec.1.LEFTSOURCEIP", "192.0.2.40"),
		p("leftSubnet", "Device.FAP.Ipsec.1.LEFTSUBNET", "10.30.0.0/24"),
		p("fragmentation", "Device.FAP.Ipsec.1.TUNNEL_FRAGMENTATION", "yes"),
		p("IKE Encryption", "Device.FAP.Ipsec.1.IKE_ENCRYPTION", "aes256"),
		p("IKE DH Group", "Device.FAP.Ipsec.1.IKE_DH_GROUP", "modp2048"),
		p("IKE Authentication", "Device.FAP.Ipsec.1.IKE_AUTHENTICATION", "sha256"),
		p("ESP Encryption", "Device.FAP.Ipsec.1.ESP_ENCRYPTION", "aes256"),
		p("ESP DH Group", "Device.FAP.Ipsec.1.ESP_DH_GROUP", "modp2048"),
		p("ESP Authentication", "Device.FAP.Ipsec.1.ESP_AUTHENTICATION", "sha256"),
		p("KeyLife", "Device.FAP.Ipsec.1.KEYLIFE", "3600"),
		p("IKELifeTime", "Device.FAP.Ipsec.1.IKELIFETIME", "86400"),
		p("RekeyMargin", "Device.FAP.Ipsec.1.REKEYMARGIN", "540"),
		p("Dpdaction", "Device.FAP.Ipsec.1.DPDACTION", "restart"),
		p("Dpddelay", "Device.FAP.Ipsec.1.DPDDELAY", "30"),
		p("Left Interface", "Device.FAP.Ipsec.1.LEFT_INTERFACE", "eth0"),
		p("Forceencaps", "Device.FAP.Ipsec.1.FORCEENCAPS", "yes"),
	}
}

func writeXML(outputDir, name string, doc provision.AutoStartXMLDocument) error {
	content, err := provision.GenerateAutoStartXML(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, name), content, 0o644)
}

func main() {
	outputDir := filepath.Join("..", "..", "..", "pnp-xml-samples")
	if len(os.Args) > 1 {
		outputDir = os.Args[1]
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		panic(err)
	}
	generatedAt := time.Date(2026, 8, 5, 10, 30, 0, 0, time.UTC)
	docs := []struct {
		name string
		doc  provision.AutoStartXMLDocument
	}{
		{"auto_start_GSM_TEST_001.xml", provision.AutoStartXMLDocument{NetworkType: "GSM", Parameters: gsmParameters()}},
		{"auto_start_LTE_TEST_001.xml", provision.AutoStartXMLDocument{NetworkType: "LTE", Parameters: lteParameters()}},
		{"NR_TEST_001_BAICELLS_v1.7_20260805183000.xml", provision.AutoStartXMLDocument{NetworkType: "NR", Vendor: "BAICELLS", SerialNumber: "NR_TEST_001", GeneratedAt: generatedAt, DataModelVersion: "v1.7", Parameters: nrParameters()}},
	}
	for _, item := range docs {
		if err := writeXML(outputDir, item.name, item.doc); err != nil {
			panic(err)
		}
		fmt.Printf("%s: %d parameters\n", filepath.Join(outputDir, item.name), len(item.doc.Parameters))
	}
}
