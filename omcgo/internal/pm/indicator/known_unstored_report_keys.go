package indicator

// KnownUnstoredReportKeys returns vendor report keys that are explicitly
// recognized for a radio technology but intentionally have no Counter ID.
//
// These keys must remain outside the ingest whitelist: several are vendor-side
// aggregates without an approved OMC definition, while MR.RIPPRB and
// MR.RECEIVEDIPOWER repeat once per PRB with the dimension encoded only in
// measType.p and therefore cannot fit the current metric natural key. Keeping
// them in this explicit catalog prevents a known, intentionally filtered value
// from being reported as a newly discovered vendor/library drift.
func KnownUnstoredReportKeys(dt DeviceType) []string {
	keys := knownUnstoredReportKeys[dt]
	return append([]string(nil), keys...)
}

var knownUnstoredReportKeys = map[DeviceType][]string{
	DeviceTypeENB: {
		"CONTEXT.AttInitalSetup.Csfb",
		"CONTEXT.AttMod.Csfb",
		"CONTEXT.SuccInitalSetup.Csfb",
		"CONTEXT.SuccMod.Csfb",
		"DRB.MaxThpDLCell",
		"DRB.MaxThpULCell",
		"ERAB.EstabAttNbr.1",
		"ERAB.EstabAttNbr.2",
		"ERAB.EstabAttNbr.3",
		"ERAB.EstabAttNbr.4",
		"ERAB.EstabAttNbr.5",
		"ERAB.EstabAttNbr.6",
		"ERAB.EstabAttNbr.7",
		"ERAB.EstabAttNbr.8",
		"ERAB.EstabAttNbr.9",
		"ERAB.EstabAttNbr.sum",
		"ERAB.EstabInitFailNbr.Sum",
		"ERAB.EstabSuccNbr.1",
		"ERAB.EstabSuccNbr.2",
		"ERAB.EstabSuccNbr.3",
		"ERAB.EstabSuccNbr.4",
		"ERAB.EstabSuccNbr.5",
		"ERAB.EstabSuccNbr.6",
		"ERAB.EstabSuccNbr.7",
		"ERAB.EstabSuccNbr.8",
		"ERAB.EstabSuccNbr.9",
		"ERAB.EstabSuccNbr.sum",
		"ERAB.ModQoSAttNbr.QCI.Sum",
		"ERAB.ModQoSSuccNbr.QCI.Sum",
		"ERAB.NbrMaxEstab",
		"ERAB.NbrReqRelEnb.CauseOMIntervention",
		"ERAB.RelActNbr.QCI.Sum",
		"ERAB.RelSuccNbr.QCI.Sum",
		"HO.AttOutExecIntraFreq",
		"HO.AttOutIntraEnb",
		"HO.AvgTimeInterEnbS1",
		"HO.AvgTimeInterEnbX2",
		"HO.SuccOutIntraEnb",
		"HO.SuccOutIntraFreq",
		"HO.eSRVCCFailNum",
		"Inter.Rat.Geran.Srvcc.Fail.Sum",
		"Inter.Rat.Utra.Srvcc.Fail.Sum",
		"MR.RECEIVEDIPOWER",
		"MR.RIPPRB",
		"PDCP.ThrpTimeDl.QCI",
		"PDCP.ThrpTimeUl.QCI",
		"RI.03",
		"RRC.ConnEstabTimeMax.DelayTolerantAccess",
		"RRC.ConnEstabTimeMean.DelayTolerantAccess",
		"RRC.ConnReleaseCsfb",
		"RRC.SuccConnReestab.NonScrcell",
	},
}
