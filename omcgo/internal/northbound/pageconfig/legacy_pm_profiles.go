package pageconfig

import (
	"fmt"
	"strings"
)

type legacyPMBaseFieldSeed struct {
	outputAlias string
	systemField string
	source      string
	dataType    string
	renderer    string
	cnName      string
}

type legacyPMMetricSeed struct {
	metricPath  string
	outputAlias string
	matchAlias  string
}

type legacyPMMetricRange struct {
	prefix     string
	width      int
	start, end int
}

type legacyPMProfileSpec struct {
	profile    string
	objectCode string
	tech       string
	baseFields []legacyPMBaseFieldSeed
	metrics    []legacyPMMetricSeed
}

var legacyPMBaseFieldsS0001PC = []legacyPMBaseFieldSeed{
	{"dn", "legacy.pm.dn", "device_info.eci / pm_metrics.object_ldn", "string", "preserve text", "Cell DN / ECI"},
	{"related_enb_dn", "legacy.pm.related_enb_dn", "devices.serial_number / pm_metrics.device_sn", "string", "quote", "Related eNB DN"},
	{"related_enb_id", "legacy.pm.related_enb_id", "device_info.enb_id or ECI >> 8", "string", "quote", "Related eNB ID"},
	{"related_enb_userlabel", "legacy.pm.related_enb_userlabel", "devices.site_name / device_info.device_name", "string", "quote", "Related eNB label"},
	{"cel_id", "legacy.pm.cel_id", "ECI & 255 or device_info.cell_id", "string", "quote", "Cell ID"},
	{"cel_id_local", "legacy.pm.cel_id_local", "device_info.cell_id", "string", "quote", "Local cell ID"},
	{"userlabel", "legacy.pm.userlabel", "device_info.device_name / devices.site_name", "string", "quote", "Cell label"},
	{"freq_mode", "legacy.pm.freq_mode", "derived from device_info.network_model", "string", "enum", "Duplex mode"},
}

var legacyPMBaseFieldsS0007PC = append([]legacyPMBaseFieldSeed{
	{"DATE_TIME", "legacy.pm.date_time", "derived from export window start", "datetime", "dd/MM/yyyy HH:mm", "Date time"},
}, legacyPMBaseFieldsS0001PC...)

var legacyS0001PCMetricRanges = []legacyPMMetricRange{
	{prefix: "C", width: 9, start: 1, end: 27},
	{prefix: "C", width: 9, start: 30, end: 166},
	{prefix: "C", width: 9, start: 10001, end: 10223},
	{prefix: "C", width: 9, start: 10228, end: 10254},
	{prefix: "C", width: 9, start: 20001, end: 20057},
	{prefix: "C", width: 9, start: 30001, end: 30053},
	{prefix: "C", width: 9, start: 30056, end: 30068},
	{prefix: "C", width: 9, start: 30071, end: 30095},
	{prefix: "C", width: 9, start: 40001, end: 40004},
	{prefix: "C", width: 9, start: 50001, end: 50004},
	{prefix: "C", width: 9, start: 60001, end: 60195},
	{prefix: "C", width: 9, start: 70001, end: 70029},
	{prefix: "C", width: 9, start: 80001, end: 80088},
	{prefix: "C", width: 9, start: 90001, end: 90092},
	{prefix: "C", width: 9, start: 90095, end: 90100},
	{prefix: "C", width: 9, start: 100001, end: 100016},
	{prefix: "C", width: 9, start: 100018, end: 100166},
	{prefix: "C", width: 9, start: 110001, end: 110016},
	{prefix: "C", width: 9, start: 120001, end: 120002},
	{prefix: "C", width: 9, start: 130001, end: 130005},
	{prefix: "C", width: 9, start: 150001, end: 150004},
	{prefix: "C", width: 9, start: 160001, end: 160004},
	{prefix: "C", width: 9, start: 170001, end: 170049},
	{prefix: "K", width: 9, start: 900010001, end: 900010037},
}

const legacyS0001PCMetricAliasesCSV = `
RRC.SetupTimeMean,RRC.SetupTimeMax,RRC.ConnMean,RRC.ConnMax,RRC.AttConnEstab,RRC.AttConnEstab.Emergency,RRC.AttConnEstab.HI_PRIO_ACCESS,RRC.AttConnEstab.MT_ACCESS,RRC.AttConnEstab.MO_SIGNAL,RRC.AttConnEstab.MO_DATA,RRC.AttConnEstab.DeToAccess,RRC.SuccConnEstab,RRC.SuccConnEstab.EMERGENCY,RRC.SuccConnEstab.HIGHPRIORITYACCES,RRC.SuccConnEstab.MTACCESS,RRC.SuccConnEstab.MOSIGNALLING,RRC.SuccConnEstab.MODATA,RRC.SuccConnEstab.DE_TO_ACCESS,RRC.AttConnReestab,RRC.AttConnReestab.RECONF_FAIL,RRC.AttConnReestab.HO_FAIL,RRC.AttConnReestab.OTHER,RRC.SuccConnReestab,RRC.SuccConnReestab.NonSrccell,RRC.SuccConnReestab.RECONF_FAIL,RRC.SuccConnReestab.HO_FAIL,RRC.SuccConnReestab.OTHER,RRC.AttConnRelease.SUM,RRC.AttConnRelease.LB,RRC.AttConnRelease.Other,RRC.AttConnRelease.CSFB,RRC.AttConnRelease.SP,RRC.ConnReleaseCsfb,RRC.ConnRelease.RedirectTo2G,RRC.ConnRelease.RedirectTo3G
RRC.ConnEstabFaileNBCause.Congestion,RRC.ConnEstabFaileNBCause.Unspecified,RRC.ConnEstabTimeMean.Emergency,RRC.ConnEstabTimeMean.HighPriorityAccess,RRC.ConnEstabTimeMean.MtAccess,RRC.ConnEstabTimeMean.MoSignalling,RRC.ConnEstabTimeMean.MoData,RRC.ConnEstabTimeMax.Emergency,RRC.ConnEstabTimeMax.HighPriorityAccess,RRC.ConnEstabTimeMax.MtAccess,RRC.ConnEstabTimeMax.MoSignalling,RRC.ConnEstabTimeMax.MoData,RRC.ConnReleaseCsfb.RedirectTo2G,RRC.ConnReleaseCsfb.RedirectTo3G,RRC.Csfb.PrepAtt,RRC.Csfb.PrepSucc,Cell.UnavailableTime.manual,Cell.UnavailableTime.fault,Cell.UnavailableTime.energy,RRC.ConnAbnormalRel,RRC.SetupTimeMeanCL0,RRC.SetupTimeMeanCL1,RRC.SetupTimeMeanCL2,RRC.SetupTimeMaxCL0,RRC.SetupTimeMaxCL1,RRC.SetupTimeMaxCL2,RRC.ConnMeanCL0,RRC.ConnMeanCL1,RRC.ConnMeanCL2,RRC.ConnMaxCL0,RRC.ConnMaxCL1,RRC.ConnMaxCL2,RRC.AttConnEstab.Cause_mo_ExceptionData,RRC.AttConnEstabCL0
RRC.AttConnEstabCL0.Cause_mt_Access,RRC.AttConnEstabCL0.Cause_mo_Signalling,RRC.AttConnEstabCL0.Cause_mo_Data,RRC.AttConnEstabCL0.Cause_mo_ExceptionData,RRC.AttConnEstabCL0.Cause_delayTolerantAccess_v1330,RRC.AttConnEstabCL1,RRC.AttConnEstabCL1.Cause_mt_Access,RRC.AttConnEstabCL1.Cause_mo_Signalling,RRC.AttConnEstabCL1.Cause_mo_Data,RRC.AttConnEstabCL1.Cause_mo_ExceptionData,RRC.AttConnEstabCL1.Cause_delayTolerantAccess_v1330,RRC.AttConnEstabCL2,RRC.AttConnEstabCL2.Cause_mt_Access,RRC.AttConnEstabCL2.Cause_mo_Signalling,RRC.AttConnEstabCL2.Cause_mo_Data,RRC.AttConnEstabCL2.Cause_mo_ExceptionData,RRC.AttConnEstabCL2.Cause_delayTolerantAccess_v1330,RRC.SuccConnEstab.Cause_mo_ExceptionData,RRC.SuccConnEstabCL0,RRC.SuccConnEstabCL0.Cause_mt_Access,RRC.SuccConnEstabCL0.Cause_mo_Signalling,RRC.SuccConnEstabCL0.Cause_mo_Data,RRC.SuccConnEstabCL0.Cause_mo_ExceptionData
RRC.SuccConnEstabCL0.Cause_delayTolerantAccess_v1330,RRC.SuccConnEstabCL1,RRC.SuccConnEstabCL1.Cause_mt_Access,RRC.SuccConnEstabCL1.Cause_mo_Signalling,RRC.SuccConnEstabCL1.Cause_mo_Data,RRC.SuccConnEstabCL1.Cause_mo_ExceptionData,RRC.SuccConnEstabCL1.Cause_delayTolerantAccess_v1330,RRC.SuccConnEstabCL2,RRC.SuccConnEstabCL2.Cause_mt_Access,RRC.SuccConnEstabCL2.Cause_mo_Signalling,RRC.SuccConnEstabCL2.Cause_mo_Data,RRC.SuccConnEstabCL2.Cause_mo_ExceptionData,RRC.SuccConnEstabCL2.Cause_delayTolerantAccess_v1330,RRC.FailConnEstab,RRC.FailConnEstab.Cause_mt_Access,RRC.FailConnEstab.Cause_mo_Signalling,RRC.FailConnEstab.Cause_mo_Data,RRC.FailConnEstab.Cause_mo_ExceptionData,RRC.FailConnEstab.Cause_delayTolerantAccess_v1330,RRC.FailConnEstab.Cause_RrcConnRej,RRC.FailConnEstabCL0,RRC.FailConnEstabCL0.Cause_mt_Access,RRC.FailConnEstabCL0.Cause_mo_Signalling,RRC.FailConnEstabCL0.Cause_mo_Data
RRC.FailConnEstabCL0.Cause_mo_ExceptionData,RRC.FailConnEstabCL0.Cause_delayTolerantAccess_v1330,RRC.FailConnEstabCL0.Cause_RrcConnRej,RRC.FailConnEstabCL1,RRC.FailConnEstabCL1.Cause_mt_Access,RRC.FailConnEstabCL1.Cause_mo_Signalling,RRC.FailConnEstabCL1.Cause_mo_Data,RRC.FailConnEstabCL1.Cause_mo_ExceptionData,RRC.FailConnEstabCL1.Cause_delayTolerantAccess_v1330,RRC.FailConnEstabCL1.Cause_RrcConnRej,RRC.FailConnEstabCL2,RRC.FailConnEstabCL2.Cause_mt_Access,RRC.FailConnEstabCL2.Cause_mo_Signalling,RRC.FailConnEstabCL2.Cause_mo_Data,RRC.FailConnEstabCL2.Cause_mo_ExceptionData,RRC.FailConnEstabCL2.Cause_delayTolerantAccess_v1330,RRC.FailConnEstabCL2.Cause_RrcConnRej,RRC.EffectiveConnMean,RRC.EffectiveConnMax,RRC.RaReceivedCL0,RRC.RaReceivedCL1,RRC.RaReceivedCL2,RRC.RaResponseCL0,RRC.RaResponseCL1,RRC.RaResponseCL2,RRC.Msg3PhrCL0.Phr0,RRC.Msg3PhrCL0.Phr1,RRC.Msg3PhrCL0.Phr2
RRC.Msg3PhrCL0.Phr3,RRC.Msg3PhrCL1.Phr0,RRC.Msg3PhrCL1.Phr1,RRC.Msg3PhrCL1.Phr2,RRC.Msg3PhrCL1.Phr3,RRC.Msg3PhrCL2.Phr0,RRC.Msg3PhrCL2.Phr1,RRC.Msg3PhrCL2.Phr2,RRC.Msg3PhrCL2.Phr3,RRC.ConnAbnormalRel.ErrorInd,RRC.ConnAbnormalRel.MmeGuardTimerExpire,RRC.ConnEstabFail.Emergency,RRC.ConnEstabFail.HighPriorityAccess,RRC.ConnEstabFail.MoData,RRC.ConnEstabFail.MoSignalling,RRC.ConnEstabFail.MtAccess,RRC.ConnEstabFail.sum,RRC.ConnEstabFaileNBCause.EnergySaving,RRC.ConnMin,RRC.ConnTime.UE,ERAB.NbrMeanEstab,ERAB.NbrMeanEstab.1,ERAB.NbrMeanEstab.2,ERAB.NbrMeanEstab.3,ERAB.NbrMeanEstab.4,ERAB.NbrMeanEstab.5,ERAB.NbrMeanEstab.6,ERAB.NbrMeanEstab.7,ERAB.NbrMeanEstab.8,ERAB.NbrMeanEstab.9,ERAB.UsageNbrMax.QCI1,ERAB.UsageNbrMax.QCI2,ERAB.UsageNbrMax.QCI3,ERAB.UsageNbrMax.QCI4,ERAB.UsageNbrMax.QCI5,ERAB.UsageNbrMax.QCI6,ERAB.UsageNbrMax.QCI7,ERAB.UsageNbrMax.QCI8,ERAB.UsageNbrMax.QCI9,ERAB.EstabTimeMean
ERAB.EstabTimeMax,ERAB.EstabTimeMean.QCI1,ERAB.EstabTimeMean.QCI2,ERAB.EstabTimeMean.QCI3,ERAB.EstabTimeMean.QCI4,ERAB.EstabTimeMean.QCI5,ERAB.EstabTimeMean.QCI6,ERAB.EstabTimeMean.QCI7,ERAB.EstabTimeMean.QCI8,ERAB.EstabTimeMean.QCI9,ERAB.EstabTimeMax.QCI1,ERAB.EstabTimeMax.QCI2,ERAB.EstabTimeMax.QCI3,ERAB.EstabTimeMax.QCI4,ERAB.EstabTimeMax.QCI5,ERAB.EstabTimeMax.QCI6,ERAB.EstabTimeMax.QCI7,ERAB.EstabTimeMax.QCI8,ERAB.EstabTimeMax.QCI9,ERAB.NbrHoInc,ERAB.NbrHoInc.1,ERAB.NbrHoInc.2,ERAB.NbrHoInc.3,ERAB.NbrHoInc.4,ERAB.NbrHoInc.5,ERAB.NbrHoInc.6,ERAB.NbrHoInc.7,ERAB.NbrHoInc.8,ERAB.NbrHoInc.9,ERAB.NbrAttEstab,ERAB.NbrAttEstab.1,ERAB.NbrAttEstab.2,ERAB.NbrAttEstab.3,ERAB.NbrAttEstab.4,ERAB.NbrAttEstab.5,ERAB.NbrAttEstab.6,ERAB.NbrAttEstab.7,ERAB.NbrAttEstab.8,ERAB.NbrAttEstab.9,ERAB.NbrSuccEstab,ERAB.NbrSuccEstab.1,ERAB.NbrSuccEstab.2,ERAB.NbrSuccEstab.3,ERAB.NbrSuccEstab.4
ERAB.NbrSuccEstab.5,ERAB.NbrSuccEstab.6,ERAB.NbrSuccEstab.7,ERAB.NbrSuccEstab.8,ERAB.NbrSuccEstab.9,ERAB.EstabInitAttNbr.sum,ERAB.EstabInitAttNbr.QCI1,ERAB.EstabInitAttNbr.QCI2,ERAB.EstabInitAttNbr.QCI3,ERAB.EstabInitAttNbr.QCI4,ERAB.EstabInitAttNbr.QCI5,ERAB.EstabInitAttNbr.QCI6,ERAB.EstabInitAttNbr.QCI7,ERAB.EstabInitAttNbr.QCI8,ERAB.EstabInitAttNbr.QCI9,ERAB.EstEstabInitSuccNbr.sum,ERAB.EstabInitSuccNbr.QCI1,ERAB.EstabInitSuccNbr.QCI2,ERAB.EstabInitSuccNbr.QCI3,ERAB.EstabInitSuccNbr.QCI4,ERAB.EstabInitSuccNbr.QCI5,ERAB.EstabInitSuccNbr.QCI6,ERAB.EstabInitSuccNbr.QCI7,ERAB.EstabInitSuccNbr.QCI8,ERAB.EstabInitSuccNbr.QCI9,ERAB.NbrFailEstab,ERAB.NbrFailEstab.CauseTransport,ERAB.NbrFailEstab.CauseRadioResourcesNotAvailable,ERAB.NbrFailEstab.CauseFailureInRadioInterfaceProcedure,ERAB.EstabInitFailNbr.ControlProcessingOverload
ERAB.EstabInitFailNbr.NotEnoughUserPlaneProcessingResourcesAvailable,ERAB.EstabInitFailNbr.RadioConnectionWithUELost,ERAB.EstabAddAttNbr.QCI1,ERAB.EstabAddAttNbr.QCI2,ERAB.EstabAddAttNbr.QCI3,ERAB.EstabAddAttNbr.QCI4,ERAB.EstabAddAttNbr.QCI5,ERAB.EstabAddAttNbr.QCI6,ERAB.EstabAddAttNbr.QCI7,ERAB.EstabAddAttNbr.QCI8,ERAB.EstabAddAttNbr.QCI9,ERAB.EstabAddAttNbr.sum,ERAB.EstabAddSuccNbr.QCI1,ERAB.EstabAddSuccNbr.QCI2,ERAB.EstabAddSuccNbr.QCI3,ERAB.EstabAddSuccNbr.QCI4,ERAB.EstabAddSuccNbr.QCI5,ERAB.EstabAddSuccNbr.QCI6,ERAB.EstabAddSuccNbr.QCI7,ERAB.EstabAddSuccNbr.QCI8,ERAB.EstabAddSuccNbr.QCI9,ERAB.EstabAddSuccNbr.sum,ERAB.EstabAddFailNbr.Sum,ERAB.EstabAddFailNbr.ControlProcessingOverload,ERAB.EstabAddFailNbr.FailureInTheRadioInterfaceProcedure,ERAB.EstabAddFailNbr.NoRadioResourcesAvailableInTargetCell,ERAB.EstabAddFailNbr.NotEnoughUserPlaneProcessingResourcesAvailable
ERAB.EstabAddFailNbr.RadioConnectionWithUELost,ERAB.EstabAddFailNbr.TransportResourceUnavailable,ERAB.EstabFailNbr.RadioConnectionWithUELost,ERAB.EstabFailNbr.TransportResourceUnavailable,ERAB.NbrReqRelEnb,ERAB.NbrReqRelEnb.CauseUserInactivity,ERAB.NbrReqRelEnb.CauseRADIORESOURCESNOTAVAILABLE,ERAB.NbrReqRelEnb.CauseREDUCELOADINSERVINGCELL,ERAB.NbrReqRelEnb.CauseFAILUREINTHERADIOINTERFACEPROCEDURE,ERAB.NbrReqRelEnb.CauseRELEASEDUETOEUTRANGENERATEDREASONS,ERAB.NbrReqRelEnb.CauseRADIOCONNECTIONWITHUELOST,ERAB.NbrReqRelEnb.CauseOAMINTERVENTION,ERAB.RelEnbNbr.InterSystemRedirection,ERAB.RelEnbNbr.HOOverallExpiry,ERAB.NbrReqRelEnb.1,ERAB.NbrReqRelEnb.2,ERAB.NbrReqRelEnb.3,ERAB.NbrReqRelEnb.4,ERAB.NbrReqRelEnb.5,ERAB.NbrReqRelEnb.6,ERAB.NbrReqRelEnb.7,ERAB.NbrReqRelEnb.8,ERAB.NbrReqRelEnb.9,ERAB.NbrReqRelEnb.Normal,ERAB.NbrReqRelEnb.Normal.1,ERAB.NbrReqRelEnb.Normal.2,ERAB.NbrReqRelEnb.Normal.3
ERAB.NbrReqRelEnb.Normal.4,ERAB.NbrReqRelEnb.Normal.5,ERAB.NbrReqRelEnb.Normal.6,ERAB.NbrReqRelEnb.Normal.7,ERAB.NbrReqRelEnb.Normal.8,ERAB.NbrReqRelEnb.Normal.9,ERAB.RelSuccNbr.QCI1,ERAB.RelSuccNbr.QCI2,ERAB.RelSuccNbr.QCI3,ERAB.RelSuccNbr.QCI4,ERAB.RelSuccNbr.QCI5,ERAB.RelSuccNbr.QCI6,ERAB.RelSuccNbr.QCI7,ERAB.RelSuccNbr.QCI8,ERAB.RelSuccNbr.QCI9,ERAB.RelFailNbr.Sum,ERAB.RelFailNbr.ControlProcessingOverload,ERAB.RelActNbr.QCI1,ERAB.RelActNbr.QCI2,ERAB.RelActNbr.QCI3,ERAB.RelActNbr.QCI4,ERAB.RelActNbr.QCI5,ERAB.RelActNbr.QCI6,ERAB.RelActNbr.QCI7,ERAB.RelActNbr.QCI8,ERAB.RelActNbr.QCI9,ERAB.ModQoSAttNbr.QCI1,ERAB.ModQoSAttNbr.QCI2,ERAB.ModQoSAttNbr.QCI3,ERAB.ModQoSAttNbr.QCI4,ERAB.ModQoSAttNbr.QCI5,ERAB.ModQoSAttNbr.QCI6,ERAB.ModQoSAttNbr.QCI7,ERAB.ModQoSAttNbr.QCI8,ERAB.ModQoSAttNbr.QCI9,ERAB.ModQoSSuccNbr.QCI1,ERAB.ModQoSSuccNbr.QCI2,ERAB.ModQoSSuccNbr.QCI3,ERAB.ModQoSSuccNbr.QCI4
ERAB.ModQoSSuccNbr.QCI5,ERAB.ModQoSSuccNbr.QCI6,ERAB.ModQoSSuccNbr.QCI7,ERAB.ModQoSSuccNbr.QCI8,ERAB.ModQoSSuccNbr.QCI9,ERAB.ModQoSFailNbr.Sum,ERAB.ModQoSFailNbr.ControlProcessingOverload,ERAB.ModQoSFailNbr.NoRadioResourcesAvailableInTargetCell,ERAB.ModQoSFailNbr.NotEnoughUserPlaneProcessingResourcesAvailable,ERAB.ModQoSFailNbr.TransportResourceUnavailable,ERAB.HoFail,ERAB.HoFail.1,ERAB.HoFail.2,ERAB.HoFail.3,ERAB.HoFail.4,ERAB.HoFail.5,ERAB.HoFail.6,ERAB.HoFail.7,ERAB.HoFail.8,ERAB.HoFail.9,ERAB.NbrLeft,ERAB.NbrLeft.1,ERAB.NbrLeft.2,ERAB.NbrLeft.3,ERAB.NbrLeft.4,ERAB.NbrLeft.5,ERAB.NbrLeft.6,ERAB.NbrLeft.7,ERAB.NbrLeft.8,ERAB.NbrLeft.9,ERAB.RelEnbNbr.Abnormal,ERAB.RelMmeNbr.Abnormal,ERAB.RelMmeNbr.Sum,ERAB.SessionTime.UE,ERAB.DurationPerDrop,ERAB.NbrLeftLastPer,ERAB.RelEnbNbr.Sum,ERAB.ActiveMeanNbr.QCI1,ERAB.ActiveMeanNbr.QCI2,ERAB.AttPS.sum,ERAB.DropPS.sum,ERAB.AttVolte.sum
ERAB.DropVolte.sum,ERAB.EstabAddFailNbr.Misc,ERAB.EstabAddFailNbr.Nas,ERAB.EstabAddFailNbr.Protocol,ERAB.EstabAddFailNbr.RadioNetwork,ERAB.EstabAddFailNbr.Transport,ERAB.EstabInitFailNbr.Misc,ERAB.EstabInitFailNbr.Nas,ERAB.EstabInitFailNbr.Protocol,ERAB.EstabInitFailNbr.RadioNetwork,ERAB.EstabInitFailNbr.Transport,ERAB.RelEnbNbr.Abnormal.QCI1,ERAB.RelEnbNbr.Abnormal.QCI2,ERAB.RelEnbNbr.Abnormal.QCI3,ERAB.RelEnbNbr.Abnormal.QCI4,ERAB.RelEnbNbr.Abnormal.QCI5,ERAB.RelEnbNbr.Abnormal.QCI6,ERAB.RelEnbNbr.Abnormal.QCI7,ERAB.RelEnbNbr.Abnormal.QCI8,ERAB.RelEnbNbr.Abnormal.QCI9,ERAB.SessionTime.Volte,CONTEXT.AttInitalSetup,CONTEXT.AttInitalSetup.Csfb,CONTEXT.SuccInitalSetup,CONTEXT.SuccInitalSetup.Csfb,CONTEXT.FailInitalSetup,CONTEXT.FailInitalSetup.SecurModeFail,CONTEXT.FailInitalSetup.NoRadioRes,CONTEXT.FailInitalSetup.UeNoReply,CONTEXT.AttMod,CONTEXT.AttMod.Csfb,CONTEXT.SuccMod
CONTEXT.SuccMod.Csfb,CONTEXT.AttRelEnb,CONTEXT.AttRelEnb.CauseUserInactivity,CONTEXT.AttRelEnb.Normal,CONTEXT.AttRelMme,UECNTX.RelReq.OMIntervention,UECNTX.RelReq.HardwareFailure,UECNTX.RelSuccNbr,CONTEXT.NbrLeft,UECNTX.RelReq.AbnormalFailure,CONTEXT.NbrLeftLastPer,CONTEXT.AttRelEnb.Cause_Unspecified,CONTEXT.AttRelEnbCL0,CONTEXT.AttRelEnbCL0.Cause_UserInactivity,CONTEXT.AttRelEnbCL0.Cause_Unspecified,CONTEXT.AttRelEnbCL1,CONTEXT.AttRelEnbCL1.Cause_UserInactivity,CONTEXT.AttRelEnbCL1.Cause_Unspecified,CONTEXT.AttRelEnbCL2,CONTEXT.AttRelEnbCL2.Cause_UserInactivity,CONTEXT.AttRelEnbCL2.Cause_Unspecified,CONTEXT.AttRelEnbCL0.Normal,CONTEXT.AttRelEnbCL1.Normal,CONTEXT.AttRelEnbCL2.Normal,CONTEXT.AttRelMme.Cause_NormalRelease,CONTEXT.AttRelMme.Cause_Detach,CONTEXT.AttRelMme.Cause_Unspecified,CONTEXT.AttRelMme.Cause_AuthenticationFailure,CONTEXT.AttRelMmeCL0
CONTEXT.AttRelMmeCL0.Cause_NormalRelease,CONTEXT.AttRelMmeCL0.Cause_Detach,CONTEXT.AttRelMmeCL0.Cause_Unspecified,CONTEXT.AttRelMmeCL0.Cause_AuthenticationFailure,CONTEXT.AttRelMmeCL1,CONTEXT.AttRelMmeCL1.Cause_NormalRelease,CONTEXT.AttRelMmeCL1.Cause_Detach,CONTEXT.AttRelMmeCL1.Cause_Unspecified,CONTEXT.AttRelMmeCL1.Cause_AuthenticationFailure,CONTEXT.AttRelMmeCL2,CONTEXT.AttRelMmeCL2.Cause_NormalRelease,CONTEXT.AttRelMmeCL2.Cause_Detach,CONTEXT.AttRelMmeCL2.Cause_Unspecified,CONTEXT.AttRelMmeCL2.Cause_AuthenticationFailure,CONTEXT.NbrLeftCL0,CONTEXT.NbrLeftCL1,CONTEXT.NbrLeftCL2,HO.AttOutInterEnbS1,HO.AttOutInterEnbS1.1,HO.SuccOutPrepInterEnbS1,HO.SuccOutPrepInterEnbS1.1,HO.AttOutExecInterEnbS1,HO.AttOutExecInterEnbS1.1,HO.SuccOutInterEnbS1,HO.SuccOutInterEnbS1.1,HO.AvgTimeInterEnbS1,HO.AttOutInterEnbX2,HO.AttOutInterEnbX2.1,HO.SuccOutPrepInterEnbX2,HO.SuccOutPrepInterEnbX2.1
HO.AttOutExecInterEnbX2,HO.AttOutExecInterEnbX2.1,HO.SuccOutInterEnbX2,HO.SuccOutInterEnbX2.1,HO.AvgTimeInterEnbX2,HO.FailOut,HO.InterEnbFail.RelocPrepExpiry,HO.OutAttTarget.RadioNetwork,HO.OutAttTarget.Transport,HO.OutAttTarget.NAS,HO.OutAttTarget.Protocol,HO.OutAttTarget.Misc,HO.OutAttTarget.Sum,HO.OutSuccTarget.RadioNetwork,HO.OutSuccTarget.Transport,HO.OutSuccTarget.NAS,HO.OutSuccTarget.Protocol,HO.OutSuccTarget.Misc,HO.OutSuccTarget.Sum,HO.InterEnbOutPrepAtt,HO.InterEnbOutAtt.Sum,HO.InterEnbOutSucc.Sum,HO.IntraFreqOutAtt,HO.IntraFreqOutSucc,HO.InterFreqMeasGapOutAtt,HO.InterFreqMeasGapOutSucc,HO.InterFreqNoMeasGapOutAtt,HO.InterFreqNoMeasGapOutSucc,HO.DrxOutAtt,HO.DrxOutSucc,HO.NoDrxOutAtt,HO.NoDrxOutSucc,HO.InterRatOutAtt.Sum,HO.InterRatOutAtt.Csfb.Geran,HO.InterRatOutAtt.Csfb.Utran,HO.InterRatOutAtt.Csfb.Cco,HO.InterRatOutSucc.Sum,HO.InterRatOutSucc.Csfb.Geran
HO.InterRatOutSucc.Csfb.Utran,HO.InterRatOutSucc.Csfb.Cco,HO.Prep.FailOut.Sum,HO.InterEnb.InAtt,HO.InterEnb.InSucc,HO.InterRatOutAtt.LTEtoUTRAN,HO.InterRatOutSucc.LTEtoUTRAN,HO.InterRatOutAtt.LTEtoGERAN,HO.InterRatOutSucc.LTEtoGERAN,HO.IntraEnbOutPrepAtt.sum,HO.IntraEnbOutPrepSucc.sum,HO.IntraEnbOutAtt.sum,HO.IntraEnbOutSucc.sum,HO.InterEnbOutPrepAtt.sum,HO.InterEnbOutPrepSucc.sum,HO.InterFreqPrepAtt,HO.InterFreqPrepSucc,HO.IntraFreqPrepAtt,HO.IntraFreqPrepSucc,HO.InterEnb.InAtt.IntraFreq,HO.InterEnb.InAtt.InterFreq,HO.InterEnb.InSucc.IntraFreq,HO.InterEnb.InSucc.InterFreq,HO.InterEnb.InAtt.S1,HO.InterEnb.InAtt.X2,HO.InterEnb.InSucc.S1,HO.InterEnb.InSucc.X2,HO.InterEnbFail.sum,HO.InterEnbFail.RadioNetwork,HO.InterEnbFail.Transport,HO.InterEnbFail.Nas,HO.InterEnbFail.Protocol,HO.InterEnbFail.Misc,HO.InterEnbFail.OverallExpiry,HO.InterEnbFail.ReEstablish,HO.InterEnbFail.InterError
HO.InterEnbFail.HoProtocolError,HO.InterEnbFail.RadioUeLost,HO.InterEnbFail.HoCancel,HO.InterEnbFail.RadioLinkFailure,IRATHO.AttOutUtran,IRATHO.SuccPrepOutUtran,IRATHO.FailPrepOutUtran,IRATHO.SuccOutUtran,PAG.PagReceived,PAG.PagDiscarded,PAG.SucceedNbr,PAG.UserNbrRate,PDCP.UpOctUl,PDCP.UpOctUl.1,PDCP.UpOctUl.2,PDCP.UpOctUl.3,PDCP.UpOctUl.4,PDCP.UpOctUl.5,PDCP.UpOctUl.6,PDCP.UpOctUl.7,PDCP.UpOctUl.8,PDCP.UpOctUl.9,PDCP.UpOctDl,PDCP.UpOctDl.1,PDCP.UpOctDl.2,PDCP.UpOctDl.3,PDCP.UpOctDl.4,PDCP.UpOctDl.5,PDCP.UpOctDl.6,PDCP.UpOctDl.7,PDCP.UpOctDl.8,PDCP.UpOctDl.9,PDCP.CpOctUl,PDCP.CpOctDl,PDCP.NbrPktUl,DRB.IPThpUl.QCI1,DRB.IPThpUl.QCI2,DRB.IPThpUl.QCI3,DRB.IPThpUl.QCI4,DRB.IPThpUl.QCI5,DRB.IPThpUl.QCI6,DRB.IPThpUl.QCI7,DRB.IPThpUl.QCI8,DRB.IPThpUl.QCI9,DRB.IPThpDl.QCI1,DRB.IPThpDl.QCI2,DRB.IPThpDl.QCI3,DRB.IPThpDl.QCI4,DRB.IPThpDl.QCI5,DRB.IPThpDl.QCI6,DRB.IPThpDl.QCI7,DRB.IPThpDl.QCI8
DRB.IPThpDl.QCI9,PDCP.NbrPktUl.1,PDCP.NbrPktUl.2,PDCP.NbrPktUl.3,PDCP.NbrPktUl.4,PDCP.NbrPktUl.5,PDCP.NbrPktUl.6,PDCP.NbrPktUl.7,PDCP.NbrPktUl.8,PDCP.NbrPktUl.9,PDCP.NbrPktLossUl,PDCP.NbrPktLossUl.1,PDCP.NbrPktLossUl.2,PDCP.NbrPktLossUl.3,PDCP.NbrPktLossUl.4,PDCP.NbrPktLossUl.5,PDCP.NbrPktLossUl.6,PDCP.NbrPktLossUl.7,PDCP.NbrPktLossUl.8,PDCP.NbrPktLossUl.9,DRB.PdcpSduLossRateUl.Sum,DRB.PdcpSduLossRateUl.QCI1,DRB.PdcpSduLossRateUl.QCI2,DRB.PdcpSduLossRateUl.QCI3,DRB.PdcpSduLossRateUl.QCI4,DRB.PdcpSduLossRateUl.QCI5,DRB.PdcpSduLossRateUl.QCI6,DRB.PdcpSduLossRateUl.QCI7,DRB.PdcpSduLossRateUl.QCI8,DRB.PdcpSduLossRateUl.QCI9,PDCP.NbrPktDl,PDCP.NbrPktDl.1,PDCP.NbrPktDl.2,PDCP.NbrPktDl.3,PDCP.NbrPktDl.4,PDCP.NbrPktDl.5,PDCP.NbrPktDl.6,PDCP.NbrPktDl.7,PDCP.NbrPktDl.8,PDCP.NbrPktDl.9,PDCP.NbrPktLossDl,PDCP.NbrPktLossDl.1,PDCP.NbrPktLossDl.2,PDCP.NbrPktLossDl.3,PDCP.NbrPktLossDl.4
PDCP.NbrPktLossDl.5,PDCP.NbrPktLossDl.6,PDCP.NbrPktLossDl.7,PDCP.NbrPktLossDl.8,PDCP.NbrPktLossDl.9,DRB.PdcpSduAirLossRateDl.Sum,DRB.PdcpSduAirLossRateDl.QCI1,DRB.PdcpSduAirLossRateDl.QCI2,DRB.PdcpSduAirLossRateDl.QCI3,DRB.PdcpSduAirLossRateDl.QCI4,DRB.PdcpSduAirLossRateDl.QCI5,DRB.PdcpSduAirLossRateDl.QCI6,DRB.PdcpSduAirLossRateDl.QCI7,DRB.PdcpSduAirLossRateDl.QCI8,DRB.PdcpSduAirLossRateDl.QCI9,PDCP.UpPktDelayDl,PDCP.UpPktDelayDl.1,PDCP.UpPktDelayDl.2,PDCP.UpPktDelayDl.3,PDCP.UpPktDelayDl.4,PDCP.UpPktDelayDl.5,PDCP.UpPktDelayDl.6,PDCP.UpPktDelayDl.7,PDCP.UpPktDelayDl.8,PDCP.UpPktDelayDl.9,DRB.IpLateDL.QCI1,DRB.IpLateDL.QCI2,DRB.IpLateDL.QCI3,DRB.IpLateDL.QCI4,DRB.IpLateDL.QCI5,DRB.IpLateDL.QCI6,DRB.IpLateDL.QCI7,DRB.IpLateDL.QCI8,DRB.IpLateDL.QCI9,PDCP.UpPktDiscardDl,PDCP.UpPktDiscardDl.1,PDCP.UpPktDiscardDl.2,PDCP.UpPktDiscardDl.3,PDCP.UpPktDiscardDl.4,PDCP.UpPktDiscardDl.5
PDCP.UpPktDiscardDl.6,PDCP.UpPktDiscardDl.7,PDCP.UpPktDiscardDl.8,PDCP.UpPktDiscardDl.9,DRB.PdcpSduDropRateDl.Sum,DRB.PdcpSduDropRateDl.QCI1,DRB.PdcpSduDropRateDl.QCI2,DRB.PdcpSduDropRateDl.QCI3,DRB.PdcpSduDropRateDl.QCI4,DRB.PdcpSduDropRateDl.QCI5,DRB.PdcpSduDropRateDl.QCI6,DRB.PdcpSduDropRateDl.QCI7,DRB.PdcpSduDropRateDl.QCI8,DRB.PdcpSduDropRateDl.QCI9,SRB.PdcpSduBitrateDl,SRB.PdcpSduBitrateUl,DRB.PdcpSduBitrateDl.QCI1,DRB.PdcpSduBitrateDl.QCI2,DRB.PdcpSduBitrateDl.QCI3,DRB.PdcpSduBitrateDl.QCI4,DRB.PdcpSduBitrateDl.QCI5,DRB.PdcpSduBitrateDl.QCI6,DRB.PdcpSduBitrateDl.QCI7,DRB.PdcpSduBitrateDl.QCI8,DRB.PdcpSduBitrateDl.QCI9,DRB.PdcpSduBitrateUl.QCI1,DRB.PdcpSduBitrateUl.QCI2,DRB.PdcpSduBitrateUl.QCI3,DRB.PdcpSduBitrateUl.QCI4,DRB.PdcpSduBitrateUl.QCI5,DRB.PdcpSduBitrateUl.QCI6,DRB.PdcpSduBitrateUl.QCI7,DRB.PdcpSduBitrateUl.QCI8,DRB.PdcpSduBitrateUl.QCI9,DRB.PdcpSduBitrateDlMax
DRB.PdcpSduBitrateUlMax,DRB.PdcpSduDelayDl.Sum,DRB.PdcpSduDelayDl.QCI1,DRB.PdcpSduDelayDl.QCI2,DRB.PdcpSduDelayDl.QCI3,DRB.PdcpSduDelayDl.QCI4,DRB.PdcpSduDelayDl.QCI5,DRB.PdcpSduDelayDl.QCI6,DRB.PdcpSduDelayDl.QCI7,DRB.PdcpSduDelayDl.QCI8,DRB.PdcpSduDelayDl.QCI9,SRB1bis.OctUl,SRB1bis.OctDl,SRB1bis.RlcThrpTimeDl,SRB1bis.RlcThrpTimeUl,DRB.PdcpLastTtiVolDL.QCI1,DRB.PdcpLastTtiVolDL.QCI2,DRB.PdcpLastTtiVolDL.QCI3,DRB.PdcpLastTtiVolDL.QCI4,DRB.PdcpLastTtiVolDL.QCI5,DRB.PdcpLastTtiVolDL.QCI6,DRB.PdcpLastTtiVolDL.QCI7,DRB.PdcpLastTtiVolDL.QCI8,DRB.PdcpLastTtiVolDL.QCI9,DRB.PdcpLastTtiVolUL.QCI1,DRB.PdcpLastTtiVolUL.QCI2,DRB.PdcpLastTtiVolUL.QCI3,DRB.PdcpLastTtiVolUL.QCI4,DRB.PdcpLastTtiVolUL.QCI5,DRB.PdcpLastTtiVolUL.QCI6,DRB.PdcpLastTtiVolUL.QCI7,DRB.PdcpLastTtiVolUL.QCI8,DRB.PdcpLastTtiVolUL.QCI9,DRB.PdcpThrpTimeUl.QCI9,DRB.PdcpThrpTimeDl.QCI9,DRB.UEActiveUl.QCI1,DRB.UEActiveUl.QCI2
DRB.UEActiveUl.QCI3,DRB.UEActiveUl.QCI4,DRB.UEActiveUl.QCI5,DRB.UEActiveUl.QCI6,DRB.UEActiveUl.QCI7,DRB.UEActiveUl.QCI8,DRB.UEActiveUl.QCI9,DRB.UEActiveDl.QCI1,DRB.UEActiveDl.QCI2,DRB.UEActiveDl.QCI3,DRB.UEActiveDl.QCI4,DRB.UEActiveDl.QCI5,DRB.UEActiveDl.QCI6,DRB.UEActiveDl.QCI7,DRB.UEActiveDl.QCI8,DRB.UEActiveDl.QCI9,DRB.UEActiveUl.Sum,DRB.UEActiveDl.Sum,DRB.ActiveUENbr.Avg,DRB.ActiveUENbr.Max,DRB.AvgThpDlCell,DRB.AvgThpDlUe,DRB.AvgThpUlCell,DRB.AvgThpUlUe,DRB.ActiveUENbr.Min,RRU.DtchPrbAssnMeanDl,RRU.PuschPrbTotMeanUl,RRU.PdschPrbTotMeanDl,RRU.PuschPrbMeanTot,RRU.PdschPrbMeanTot,RRU.TtiTotUl,RRU.TtiTotDl,RRU.PrbDl.QCI1,RRU.PrbDl.QCI2,RRU.PrbDl.QCI3,RRU.PrbDl.QCI4,RRU.PrbDl.QCI5,RRU.PrbDl.QCI6,RRU.PrbDl.QCI7,RRU.PrbDl.QCI8,RRU.PrbDl.QCI9,RRU.PrbUl.QCI1,RRU.PrbUl.QCI2,RRU.PrbUl.QCI3,RRU.PrbUl.QCI4,RRU.PrbUl.QCI5,RRU.PrbUl.QCI6,RRU.PrbUl.QCI7,RRU.PrbUl.QCI8,RRU.PrbUl.QCI9,RRU.PrbTotDl
RRU.PrbTotUl,RRU.Rssi0,RRU.Rssi1,RRU.RachPreambleDedMean,RRU.RachPreambleAMean,RRU.RachPreambleBMean,RRU.CellUnavailableTime.manual,RRU.CellUnavailableTime.fault,RRU.CellUnavailableTime.sum,RRU.Vswr0,RRU.Vswr1,RRU.CellAvailableTime,RRU.RachPreambleRcvd,RRU.NpdcchCceUtil,RRU.NpdcchCceUtil.Cce1,RRU.NpdcchCceUtil.Cce2,RRU.NpdcchUsageTime,RRU.NpdcchUsageTime.UL,RRU.NpdcchUsageTime.DL,RRU.NpdschUsageTime,RRU.NprachReserveResource,RRU.NpuschUsageResource,RRU.NpuschUsageResource.x3750Hz,RRU.NprachAssnResource,RRU.NpuschUsageTime,RRU.NprachReserveResourceCL0,RRU.NprachReserveResourceCL1,RRU.NprachReserveResourceCL2,RRU.NprachAssnResourceCL0,RRU.NprachAssnResourceCL1,RRU.NprachAssnResourceCL2,RRU.PuschPrbAssn,RRU.PdschPrbAssn,RRU.PuschPrbTot,RRU.PdschPrbTot,RRU.ULInitBler,RRU.ULRetxBler,RRU.AvgULRSSI,RRU.DLInitBler,RRU.DLRetxBler,RRU.DLVolteInitBler,RRU.DLVolteRetxBler,RRU.Msg3InitBler
RRU.Msg3RetxBler,RRU.LastTtiCntDl.QCI1,RRU.LastTtiCntDl.QCI2,RRU.LastTtiCntDl.QCI3,RRU.LastTtiCntDl.QCI4,RRU.LastTtiCntDl.QCI5,RRU.LastTtiCntDl.QCI6,RRU.LastTtiCntDl.QCI7,RRU.LastTtiCntDl.QCI8,RRU.LastTtiCntDl.QCI9,RRU.LastTtiCntUl.QCI1,RRU.LastTtiCntUl.QCI2,RRU.LastTtiCntUl.QCI3,RRU.LastTtiCntUl.QCI4,RRU.LastTtiCntUl.QCI5,RRU.LastTtiCntUl.QCI6,RRU.LastTtiCntUl.QCI7,RRU.LastTtiCntUl.QCI8,RRU.LastTtiCntUl.QCI9,MAC.NbrTbUl,MAC.NbrResErrTbUl,MAC.NbrInitTbDl,MAC.NbrResErrTbDl,MAC.TxTotKbytes,MAC.TxMaxBytes,MAC.RxTotKbytes,MAC.RxMaxBytes,MAC.DLHarq.nack.0,MAC.DLHarq.nack.1,MAC.DLHarq.ack.0,MAC.DLHarq.ack.1,MAC.ULHarq.nack,MAC.ULHarq.ack,MAC.AvgUserThpDl,MAC.AvgUserThpUl,MAC.TimingAdvDistribution,MAC.RachAttempt,MAC.RachSuccess,MAC.RachSuccessRate,MAC.NbrInitTbUl,MAC.NbrSuccInitTbUl,MAC.NbrTbUlCL0,MAC.NbrTbUlCL1,MAC.NbrTbUlCL2,MAC.NbrInitTbUlCL0,MAC.NbrInitTbUlCL1,MAC.NbrInitTbUlCL2
MAC.NbrSuccInitTbUlCL0,MAC.NbrSuccInitTbUlCL1,MAC.NbrSuccInitTbUlCL2,MAC.NbrResErrTbUlCL0,MAC.NbrResErrTbUlCL1,MAC.NbrResErrTbUlCL2,MAC.NbrTbDl,MAC.NbrTbDlCL0,MAC.NbrTbDlCL1,MAC.NbrTbDlCL2,MAC.NbrInitTbDlCL0,MAC.NbrInitTbDlCL1,MAC.NbrInitTbDlCL2,MAC.NbrSuccInitTbDlCL0,MAC.NbrSuccInitTbDlCL1,MAC.NbrSuccInitTbDlCL2,MAC.NbrResErrTbDlCL0,MAC.NbrResErrTbDlCL1,MAC.NbrResErrTbDlCL2,MAC.NbrSuccInitTbDl,MAC.NpdschMcs.Mcs0,MAC.NpdschMcs.Mcs1,MAC.NpdschMcs.Mcs2,MAC.NpdschMcs.Mcs3,MAC.NpdschMcs.Mcs4,MAC.NpdschMcs.Mcs5,MAC.NpdschMcs.Mcs6,MAC.NpdschMcs.Mcs7,MAC.NpdschMcs.Mcs8,MAC.NpdschMcs.Mcs9,MAC.NpdschMcs.Mcs10,MAC.NpdschMcs.Mcs11,MAC.NpdschMcs.Mcs12,MAC.NpdschMcs.Mcs13,MAC.NpuschMcs.Mcs0,MAC.NpuschMcs.Mcs1,MAC.NpuschMcs.Mcs2,MAC.NpuschMcs.Mcs3,MAC.NpuschMcs.Mcs4,MAC.NpuschMcs.Mcs5,MAC.NpuschMcs.Mcs6,MAC.NpuschMcs.Mcs7,MAC.NpuschMcs.Mcs8,MAC.NpuschMcs.Mcs9,MAC.NpuschMcs.Mcs10,MAC.NpuschMcs.Mcs11
MAC.NpuschMcs.Mcs12,MAC.NpuschMcs.Mcs13,MAC.NpdschRepetition.Rep1,MAC.NpdschRepetition.Rep2,MAC.NpdschRepetition.Rep4,MAC.NpdschRepetition.Rep8,MAC.NpdschRepetition.Rep16,MAC.NpdschRepetition.Rep32,MAC.NpdschRepetition.Rep64,MAC.NpdschRepetition.Repother,MAC.NpuschRepetition.Rep1,MAC.NpuschRepetition.Rep2,MAC.NpuschRepetition.Rep4,MAC.NpuschRepetition.Rep8,MAC.NpuschRepetition.Rep16,MAC.NpuschRepetition.Rep32,MAC.NpuschRepetition.Rep64,MAC.NpuschRepetition.Rep128,MAC.NbrTbDlRank1,MAC.NbrTbDlRank2,MAC.DLMeanMcs,MAC.ULMeanMcs,MAC.TimingAdvanceMax,MAC.TimingAdvanceMin,PHY.NbrCqi0,PHY.NbrCqi1,PHY.NbrCqi2,PHY.NbrCqi3,PHY.NbrCqi4,PHY.NbrCqi5,PHY.NbrCqi6,PHY.NbrCqi7,PHY.NbrCqi8,PHY.NbrCqi9,PHY.NbrCqi10,PHY.NbrCqi11,PHY.NbrCqi12,PHY.NbrCqi13,PHY.NbrCqi14,PHY.NbrCqi15,PHY.ULMeanNL.SubCarrier0,PHY.ULMeanNL.SubCarrier1,PHY.ULMeanNL.SubCarrier2,PHY.ULMeanNL.SubCarrier3,PHY.ULMeanNL.SubCarrier4
PHY.ULMeanNL.SubCarrier5,PHY.ULMeanNL.SubCarrier6,PHY.ULMeanNL.SubCarrier7,PHY.ULMeanNL.SubCarrier8,PHY.ULMeanNL.SubCarrier9,PHY.ULMeanNL.SubCarrier10,PHY.ULMeanNL.SubCarrier11,PHY.ULMeanNL.SubCarrier12,PHY.ULMeanNL.SubCarrier13,PHY.ULMeanNL.SubCarrier14,PHY.ULMeanNL.SubCarrier15,PHY.ULMeanNL.SubCarrier16,PHY.ULMeanNL.SubCarrier17,PHY.ULMeanNL.SubCarrier18,PHY.ULMeanNL.SubCarrier19,PHY.ULMeanNL.SubCarrier20,PHY.ULMeanNL.SubCarrier21,PHY.ULMeanNL.SubCarrier22,PHY.ULMeanNL.SubCarrier23,PHY.ULMeanNL.SubCarrier24,PHY.ULMeanNL.SubCarrier25,PHY.ULMeanNL.SubCarrier26,PHY.ULMeanNL.SubCarrier27,PHY.ULMeanNL.SubCarrier28,PHY.ULMeanNL.SubCarrier29,PHY.ULMeanNL.SubCarrier30,PHY.ULMeanNL.SubCarrier31,PHY.ULMeanNL.SubCarrier32,PHY.ULMeanNL.SubCarrier33,PHY.ULMeanNL.SubCarrier34,PHY.ULMeanNL.SubCarrier35,PHY.ULMeanNL.SubCarrier36,PHY.ULMeanNL.SubCarrier37,PHY.ULMeanNL.SubCarrier38
PHY.ULMeanNL.SubCarrier39,PHY.ULMeanNL.SubCarrier40,PHY.ULMeanNL.SubCarrier41,PHY.ULMeanNL.SubCarrier42,PHY.ULMeanNL.SubCarrier43,PHY.ULMeanNL.SubCarrier44,PHY.ULMeanNL.SubCarrier45,PHY.ULMeanNL.SubCarrier46,PHY.ULMeanNL.SubCarrier47,PHY.NbrCqiSum,PHY.ULMeanNL.0,PHY.ULMeanNL.1,PHY.ULMeanNL.2,PHY.ULMeanNL.3,PHY.ULMeanNL.4,PHY.ULMeanNL.5,PHY.ULMeanNL.6,PHY.ULMeanNL.7,PHY.ULMeanNL.8,PHY.ULMeanNL.9,PHY.ULMeanNL.10,PHY.ULMeanNL.11,PHY.ULMeanNL.12,PHY.ULMeanNL.13,PHY.ULMeanNL.14,PHY.ULMeanNL.15,PHY.ULMeanNL.16,PHY.ULMeanNL.17,PHY.ULMeanNL.18,PHY.ULMeanNL.19,PHY.ULMeanNL.20,PHY.ULMeanNL.21,PHY.ULMeanNL.22,PHY.ULMeanNL.23,PHY.ULMeanNL.24,PHY.ULMeanNL.25,PHY.ULMeanNL.26,PHY.ULMeanNL.27,PHY.ULMeanNL.28,PHY.ULMeanNL.29,PHY.ULMeanNL.30,PHY.ULMeanNL.31,PHY.ULMeanNL.32,PHY.ULMeanNL.33,PHY.ULMeanNL.34,PHY.ULMeanNL.35,PHY.ULMeanNL.36,PHY.ULMeanNL.37,PHY.ULMeanNL.38,PHY.ULMeanNL.39,PHY.ULMeanNL.40
PHY.ULMeanNL.41,PHY.ULMeanNL.42,PHY.ULMeanNL.43,PHY.ULMeanNL.44,PHY.ULMeanNL.45,PHY.ULMeanNL.46,PHY.ULMeanNL.47,PHY.ULMeanNL.48,PHY.ULMeanNL.49,PHY.ULMeanNL.50,PHY.ULMeanNL.51,PHY.ULMeanNL.52,PHY.ULMeanNL.53,PHY.ULMeanNL.54,PHY.ULMeanNL.55,PHY.ULMeanNL.56,PHY.ULMeanNL.57,PHY.ULMeanNL.58,PHY.ULMeanNL.59,PHY.ULMeanNL.60,PHY.ULMeanNL.61,PHY.ULMeanNL.62,PHY.ULMeanNL.63,PHY.ULMeanNL.64,PHY.ULMeanNL.65,PHY.ULMeanNL.66,PHY.ULMeanNL.67,PHY.ULMeanNL.68,PHY.ULMeanNL.69,PHY.ULMeanNL.70,PHY.ULMeanNL.71,PHY.ULMeanNL.72,PHY.ULMeanNL.73,PHY.ULMeanNL.74,PHY.ULMeanNL.75,PHY.ULMeanNL.76,PHY.ULMeanNL.77,PHY.ULMeanNL.78,PHY.ULMeanNL.79,PHY.ULMeanNL.80,PHY.ULMeanNL.81,PHY.ULMeanNL.82,PHY.ULMeanNL.83,PHY.ULMeanNL.84,PHY.ULMeanNL.85,PHY.ULMeanNL.86,PHY.ULMeanNL.87,PHY.ULMeanNL.88,PHY.ULMeanNL.89,PHY.ULMeanNL.90,PHY.ULMeanNL.91,PHY.ULMeanNL.92,PHY.ULMeanNL.93,PHY.ULMeanNL.94,PHY.ULMeanNL.95,PHY.ULMeanNL.96
PHY.ULMeanNL.97,PHY.ULMeanNL.98,PHY.ULMeanNL.99,EQPT.MeanMemUsage,EQPT.MaxMemUsage,EQPT.MeanCpuUsage.0,EQPT.MeanCpuUsage.1,EQPT.MaxCpuUsage.0,EQPT.MaxCpuUsage.1,EQPT.MeanCpuUsage,EQPT.MeanCpuUsage.2,EQPT.MeanCpuUsage.3,EQPT.MeanCpuUsage.hex2,EQPT.MeanCpuUsage.hex3,EQPT.MaxCpuUsage,EQPT.MaxCpuUsage.2,EQPT.MaxCpuUsage.3,EQPT.MaxCpuUsage.hex2,EQPT.MaxCpuUsage.hex3,S1SIG.ConnEstabAtt,S1SIG.ConnEstabSucc,S1.SetupRequest,S1.SetupResponse,S1.SetupFailure,S1.LinkDown,S1.LinkDuration,DRB.RlcDlTxByte.Sum,DRB.RlcUlRxByte.Sum,DRB.RlcDlReTxByte.Sum,DRB.RlcUlReRxByte.Sum,USER.ConnMean,USER.ConnMax,USER.ActMean,USER.ActMax,MR.RSRP.00,MR.RSRP.01,MR.RSRP.02,MR.RSRP.03,MR.RSRP.04,MR.RSRP.05,MR.RSRP.06,MR.RSRP.07,MR.RSRP.08,MR.RSRP.09,MR.RSRP.10,MR.RSRP.11,MR.RSRP.12,MR.RSRP.13,MR.RSRP.14,MR.RSRP.15,MR.RSRP.16,MR.RSRP.17,MR.RSRP.18,MR.RSRP.19,MR.RSRP.20,MR.RSRP.21,MR.RSRP.22,MR.RSRP.23,MR.RSRP.24
MR.RSRP.25,MR.RSRP.26,MR.RSRP.27,MR.RSRP.28,MR.RSRP.29,MR.RSRP.30,MR.RSRP.31,MR.RSRP.32,MR.RSRP.33,MR.RSRP.34,MR.RSRP.35,MR.RSRP.36,MR.RSRP.37,MR.RSRP.38,MR.RSRP.39,MR.RSRP.40,MR.RSRP.41,MR.RSRP.42,MR.RSRP.43,MR.RSRP.44,MR.RSRP.45,MR.RSRP.46,MR.RSRP.47,MR.RSRPSUM,DRB.MaxThpDlUe,DRB.MaxThpUlUe,RACH setup success rate,RRC setup success rate,RRC ConnReEstabSucc Rate,UE S1 Signaling Connection Establish Success Rate,E-RAB setup success rate,Initial establishment success rate,E-RAB drop rate(H),RRC drop rate,DL Packet loss Rate,UL Packet loss Rate,Statistics of voice call(QCI=1),Statistics of video call(QCI=2),Uplink PRB utilization rate,Downlink PRB utilization rate,Data Volume DL,Data Volume UL,HO.IntraFreqOutSucc.Rate,HO.InterFreqOutSucc.Rate,HO InterEnbOutSucc Rate S1,HO InterEnbOutSucc RateX2,HO InterEnbOutSucc Rate,HO IntraFreqInSucc Rate,HO InterFreqInSucc Rate
HO InterEnbInSucc Rate S1,HO InterEnbInSucc Rate X2,HO InterEnbInSucc Rate,E-RAB drop rate,Handover Preparation Rate,CSFB Success Rate,CSFB Succ,Uplink PRB utilization rate(Q),Downlink PRB utilization rate(Q),VOLTE drop rate,MR Coverage,MR Poor Cell,DOWN BLER,UP BLER
`

var legacyS0001PCMetrics = buildLegacyS0001PCMetrics()

var legacyS0007PCMetrics = []legacyPMMetricSeed{
	{metricPath: "C000000012", outputAlias: "RRC_setup_suc", matchAlias: "RRC setup suc"},
	{metricPath: "C000000005", outputAlias: "RRC_setup_att", matchAlias: "RRC setup att"},
	{metricPath: "K900010002", outputAlias: "RRC_setup_suc_rate", matchAlias: "RRC setup suc rate"},
	{metricPath: "C000010060", outputAlias: "ERAB_setup_suc", matchAlias: "ERAB setup suc"},
	{metricPath: "C000010050", outputAlias: "ERAB_setup_att", matchAlias: "ERAB setup att"},
	{metricPath: "K900010005", outputAlias: "ERAB_setup_suc_rate", matchAlias: "ERAB setup suc rate"},
	{metricPath: "C000010081", outputAlias: "ERAB_setup_suc_for_QCI1", matchAlias: "ERAB setup suc for QCI1"},
	{metricPath: "C000010071", outputAlias: "ERAB_setup_att_for_QCI1", matchAlias: "ERAB setup att for QCI1"},
	{metricPath: "C000010239", outputAlias: "ERAB_SetupFail_TNL", matchAlias: "ERAB SetupFail TNL"},
	{metricPath: "C000010238", outputAlias: "ERAB_SetupFail_RNL", matchAlias: "ERAB SetupFail RNL"},
	{metricPath: "C000010092", outputAlias: "ERAB_SetupFail_RNL_NoRadioRes", matchAlias: "ERAB SetupFail RNL NoRadioRes"},
	{metricPath: "C000050002", outputAlias: "RRC_Paging_Discard", matchAlias: "RRC Paging Discard"},
	{metricPath: "C000050001", outputAlias: "RRC_Paging_Records", matchAlias: "RRC Paging Records"},
	{metricPath: "N88ILSK900000013", outputAlias: "RRC_Paging_Discard_rate", matchAlias: "RRC Paging Discard rate"},
	{metricPath: "C000000057", outputAlias: "RRC_Abnormal_Release", matchAlias: "RRC Abnormal Release"},
	{metricPath: "K900010008", outputAlias: "RRC_Drop_Rate", matchAlias: "RRC Drop Rate"},
	{metricPath: "N88ILSK900000019", outputAlias: "ERAB_Abnormal_Release", matchAlias: "ERAB Abnormal Release"},
	{metricPath: "K900010007", outputAlias: "ERAB_Drop_Rate", matchAlias: "ERAB Drop Rate"},
	{metricPath: "C000010245", outputAlias: "ERAB_Abnormal_for_QCI1", matchAlias: "ERAB Abnormal for QCI1"},
	{metricPath: "C000010136", outputAlias: "ERAB_Release_for_QCI1", matchAlias: "ERAB Release for QCI1"},
	{metricPath: "C000060162", outputAlias: "Latency_DL", matchAlias: "Latency DL"},
	{metricPath: "C000060163", outputAlias: "Latency_DL_for_QCI1", matchAlias: "Latency DL for QCI1"},
	{metricPath: "N88ILSK900000007", outputAlias: "DL_Packet_loss_Rate", matchAlias: "DL Packet loss Rate"},
	{metricPath: "N88ILSK900000008", outputAlias: "UL_Packet_loss_Rate", matchAlias: "UL Packet loss Rate"},
	{metricPath: "C000030037", outputAlias: "Intra_Freq_HO_Out_suc", matchAlias: "Intra Freq HO Out suc"},
	{metricPath: "C000030036", outputAlias: "Intra_Freq_HO_Out_att", matchAlias: "Intra Freq HO Out att"},
	{metricPath: "K900010017", outputAlias: "Intra_Freq_HO_Out_suc_Rate", matchAlias: "Intra Freq HO Out suc Rate"},
	{metricPath: "C000030039", outputAlias: "Inter_Freq_HO_Out_suc", matchAlias: "Inter Freq HO Out suc"},
	{metricPath: "C000030038", outputAlias: "Inter_Freq_HO_Out_att", matchAlias: "Inter Freq HO Out att"},
	{metricPath: "K900010018", outputAlias: "Inter_Freq_HO_Out_suc_Rate", matchAlias: "Inter Freq HO Out suc Rate"},
	{metricPath: "C000030066", outputAlias: "Intra_eNB_Out_suc", matchAlias: "Intra eNB Out suc"},
	{metricPath: "C000030065", outputAlias: "Intra_eNB_Out_att", matchAlias: "Intra eNB Out att"},
	{metricPath: "C000030035", outputAlias: "Inter_eNB_Out_suc", matchAlias: "Inter eNB Out suc"},
	{metricPath: "C000030034", outputAlias: "Inter_eNB_Out_att", matchAlias: "Inter eNB Out att"},
	{metricPath: "C000030050", outputAlias: "Inter_RAT_HO_Out_suc", matchAlias: "Inter RAT HO Out suc"},
	{metricPath: "C000030046", outputAlias: "Inter_RAT_HO_Out_att", matchAlias: "Inter RAT HO Out att"},
	{metricPath: "C000030060", outputAlias: "Inter_RAT_3G_HO_Out_suc", matchAlias: "Inter RAT 3G HO Out suc"},
	{metricPath: "C000030059", outputAlias: "Inter_RAT_3G_HO_Out_Att", matchAlias: "Inter RAT 3G HO Out Att"},
	{metricPath: "N88ILSK900000017", outputAlias: "CS_Fall_Back_by_HO_att", matchAlias: "CS Fall Back by HO att"},
	{metricPath: "N88ILSK900000014", outputAlias: "CS_Fall_Back_by_HO_suc_rate", matchAlias: "CS Fall Back by HO suc rate"},
	{metricPath: "C000030052", outputAlias: "CS_Fall_Back_3G_suc", matchAlias: "CS Fall Back 3G suc"},
	{metricPath: "C000030048", outputAlias: "CS_Fall_Back_3G_att", matchAlias: "CS Fall Back 3G att"},
	{metricPath: "N88ILSK900000015", outputAlias: "CS_Fall_Back_3G_suc_Rate", matchAlias: "CS Fall Back 3G suc Rate"},
	{metricPath: "C000030008", outputAlias: "VoLTE_HO_suc", matchAlias: "VoLTE HO suc"},
	{metricPath: "C000030006", outputAlias: "VoLTE_HO_att", matchAlias: "VoLTE HO att"},
	{metricPath: "C000160001", outputAlias: "User_connect_average", matchAlias: "User connect average"},
	{metricPath: "C000160002", outputAlias: "User_connect_max", matchAlias: "User connect max"},
	{metricPath: "C000160003", outputAlias: "User_active_average", matchAlias: "User active average"},
	{metricPath: "C000160004", outputAlias: "User_active_max", matchAlias: "User active max"},
	{metricPath: "N88ILSK900000009", outputAlias: "RB_DL_average", matchAlias: "RB DL average"},
	{metricPath: "N88ILSK900000011", outputAlias: "RB_DL_VoLTE_AVG", matchAlias: "RB DL VoLTE AVG"},
	{metricPath: "N88ILSK900000012", outputAlias: "RB_UL_VoLTE_AVG", matchAlias: "RB UL VoLTE AVG"},
	{metricPath: "N88ILSK900000018", outputAlias: "User_active_VoLTE_avg", matchAlias: "User active VoLTE avg"},
	{metricPath: "C000070021", outputAlias: "User_active_Data_avg", matchAlias: "User active Data avg"},
	{metricPath: "C000010082", outputAlias: "ERAB_setup_suc_for_QCI2", matchAlias: "ERAB setup suc for QCI2"},
	{metricPath: "C000010083", outputAlias: "ERAB_setup_suc_for_QCI3", matchAlias: "ERAB setup suc for QCI3"},
	{metricPath: "C000010084", outputAlias: "ERAB_setup_suc_for_QCI4", matchAlias: "ERAB setup suc for QCI4"},
	{metricPath: "C000010085", outputAlias: "ERAB_setup_suc_for_QCI5", matchAlias: "ERAB setup suc for QCI5"},
	{metricPath: "C000010086", outputAlias: "ERAB_setup_suc_for_QCI6", matchAlias: "ERAB setup suc for QCI6"},
	{metricPath: "C000010087", outputAlias: "ERAB_setup_suc_for_QCI7", matchAlias: "ERAB setup suc for QCI7"},
	{metricPath: "C000010088", outputAlias: "ERAB_setup_suc_for_QCI8", matchAlias: "ERAB setup suc for QCI8"},
	{metricPath: "C000010089", outputAlias: "ERAB_setup_suc_for_QCI9", matchAlias: "ERAB setup suc for QCI9"},
	{metricPath: "C000010072", outputAlias: "ERAB_setup_att_for_QCI2", matchAlias: "ERAB setup att for QCI2"},
	{metricPath: "C000010073", outputAlias: "ERAB_setup_att_for_QCI3", matchAlias: "ERAB setup att for QCI3"},
	{metricPath: "C000010074", outputAlias: "ERAB_setup_att_for_QCI4", matchAlias: "ERAB setup att for QCI4"},
	{metricPath: "C000010075", outputAlias: "ERAB_setup_att_for_QCI5", matchAlias: "ERAB setup att for QCI5"},
	{metricPath: "C000010076", outputAlias: "ERAB_setup_att_for_QCI6", matchAlias: "ERAB setup att for QCI6"},
	{metricPath: "C000010077", outputAlias: "ERAB_setup_att_for_QCI7", matchAlias: "ERAB setup att for QCI7"},
	{metricPath: "C000010078", outputAlias: "ERAB_setup_att_for_QCI8", matchAlias: "ERAB setup att for QCI8"},
	{metricPath: "C000010079", outputAlias: "ERAB_setup_att_for_QCI9", matchAlias: "ERAB setup att for QCI9"},
	{metricPath: "C000010246", outputAlias: "ERAB_Abnormal_for_QCI2", matchAlias: "ERAB Abnormal for QCI2"},
	{metricPath: "C000010247", outputAlias: "ERAB_Abnormal_for_QCI3", matchAlias: "ERAB Abnormal for QCI3"},
	{metricPath: "C000010248", outputAlias: "ERAB_Abnormal_for_QCI4", matchAlias: "ERAB Abnormal for QCI4"},
	{metricPath: "C000010249", outputAlias: "ERAB_Abnormal_for_QCI5", matchAlias: "ERAB Abnormal for QCI5"},
	{metricPath: "C000010250", outputAlias: "ERAB_Abnormal_for_QCI6", matchAlias: "ERAB Abnormal for QCI6"},
	{metricPath: "C000010251", outputAlias: "ERAB_Abnormal_for_QCI7", matchAlias: "ERAB Abnormal for QCI7"},
	{metricPath: "C000010252", outputAlias: "ERAB_Abnormal_for_QCI8", matchAlias: "ERAB Abnormal for QCI8"},
	{metricPath: "C000010253", outputAlias: "ERAB_Abnormal_for_QCI9", matchAlias: "ERAB Abnormal for QCI9"},
	{metricPath: "C000010137", outputAlias: "ERAB_Release_for_QCI2", matchAlias: "ERAB Release for QCI2"},
	{metricPath: "C000010138", outputAlias: "ERAB_Release_for_QCI3", matchAlias: "ERAB Release for QCI3"},
	{metricPath: "C000010139", outputAlias: "ERAB_Release_for_QCI4", matchAlias: "ERAB Release for QCI4"},
	{metricPath: "C000010140", outputAlias: "ERAB_Release_for_QCI5", matchAlias: "ERAB Release for QCI5"},
	{metricPath: "C000010141", outputAlias: "ERAB_Release_for_QCI6", matchAlias: "ERAB Release for QCI6"},
	{metricPath: "C000010142", outputAlias: "ERAB_Release_for_QCI7", matchAlias: "ERAB Release for QCI7"},
	{metricPath: "C000010143", outputAlias: "ERAB_Release_for_QCI8", matchAlias: "ERAB Release for QCI8"},
	{metricPath: "C000010144", outputAlias: "ERAB_Release_for_QCI9", matchAlias: "ERAB Release for QCI9"},
}

var legacyPMProfileSpecs = append([]legacyPMProfileSpec{
	{
		profile:    legacyPMS0001PCProfile,
		objectCode: "PC",
		tech:       "LTE",
		baseFields: legacyPMBaseFieldsS0001PC,
		metrics:    legacyS0001PCMetrics,
	},
	{
		profile:    legacyPMS0007PCProfile,
		objectCode: "PC",
		tech:       "LTE",
		baseFields: legacyPMBaseFieldsS0007PC,
		metrics:    legacyS0007PCMetrics,
	},
}, legacyPMAdditionalProfileSpecs...)

func buildLegacyS0001PCMetrics() []legacyPMMetricSeed {
	metricPaths := expandLegacyPMMetricRanges(legacyS0001PCMetricRanges)
	aliases := splitLegacyPMMetricAliases(legacyS0001PCMetricAliasesCSV)
	if len(metricPaths) != len(aliases) {
		panic(fmt.Sprintf("legacy S0001 PM metric map mismatch: %d ids vs %d aliases", len(metricPaths), len(aliases)))
	}
	metrics := make([]legacyPMMetricSeed, 0, len(metricPaths))
	for i, metricPath := range metricPaths {
		metrics = append(metrics, legacyPMMetricSeed{metricPath: metricPath, outputAlias: aliases[i], matchAlias: aliases[i]})
	}
	return metrics
}

func expandLegacyPMMetricRanges(ranges []legacyPMMetricRange) []string {
	metricPaths := make([]string, 0)
	for _, item := range ranges {
		for value := item.start; value <= item.end; value++ {
			metricPaths = append(metricPaths, fmt.Sprintf("%s%0*d", item.prefix, item.width, value))
		}
	}
	return metricPaths
}

func splitLegacyPMMetricAliases(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\n", ",")
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func legacyPMProfile(profile string) (legacyPMProfileSpec, bool) {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return legacyPMProfileSpec{}, false
	}
	for _, spec := range legacyPMProfileSpecs {
		if strings.EqualFold(profile, spec.profile) {
			return spec, true
		}
	}
	return legacyPMProfileSpec{}, false
}

func legacyPMBaseFields(spec legacyPMProfileSpec) []FieldDefinition {
	fields := make([]FieldDefinition, 0, len(spec.baseFields))
	for _, seed := range spec.baseFields {
		definition := profileField(spec.profile, DomainPM, spec.objectCode, seed.outputAlias, seed.systemField, seed.source, seed.dataType, seed.renderer, seed.cnName)
		definition.Tech = spec.tech
		fields = append(fields, definition)
	}
	return fields
}

func appendLegacyPMFields(fields []FieldDefinition) []FieldDefinition {
	for _, spec := range legacyPMProfileSpecs {
		fields = append(fields, legacyPMBaseFields(spec)...)
	}
	return fields
}
