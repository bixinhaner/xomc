package metrics

import "testing"

// 真实样本（取自 PM测量对象唯一键-多制式分析-20260618.md 第二节 + 第九节）。
// 这些用例是死判依据：4G/5G/GSM 三制式 + 缺段降级 + 无法识别回退 + 制式判定优先级。

func TestParseObjectLDN_4G(t *testing.T) {
	cases := []struct {
		name       string
		ldn        string
		wantTech   Tech
		wantCellID string
		wantPlmn   string
	}{
		{"基础小区行 仅 Cellid", "Cellid=66", TechLTE, "66", ""},
		{"PLMN 行 Cellid+PLMN", "Cellid=66,PLMN=46001", TechLTE, "66", "46001"},
		{"真机大号 Cellid", "Cellid=654321", TechLTE, "654321", ""},
		{"真机大号 PLMN", "Cellid=654321,PLMN=46068", TechLTE, "654321", "46068"},
		{"大小写不敏感 CellId", "CellId=99,PLMN=1", TechLTE, "99", "1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseObjectLDN(c.ldn)
			if got.Tech != c.wantTech {
				t.Errorf("Tech: got=%q want=%q", got.Tech, c.wantTech)
			}
			if got.CellID != c.wantCellID {
				t.Errorf("CellID: got=%q want=%q", got.CellID, c.wantCellID)
			}
			if got.Plmn != c.wantPlmn {
				t.Errorf("Plmn: got=%q want=%q", got.Plmn, c.wantPlmn)
			}
			// 4G 行不应误填 5G/GSM 字段
			if got.GNBID != "" || got.NrCGI != "" || got.Uid != "" {
				t.Errorf("4G 行不该填 5G/GSM 字段：%+v", got)
			}
			// BaseCellID 应等于 CellID
			if got.BaseCellID() != c.wantCellID {
				t.Errorf("BaseCellID: got=%q want=%q", got.BaseCellID(), c.wantCellID)
			}
		})
	}
}

func TestParseObjectLDN_5G(t *testing.T) {
	cases := []struct {
		name           string
		ldn            string
		wantGNBID      string
		wantNrCGI      string
		wantCUID       string
		wantDUID       string
		wantPLMNID     string
		wantNSSAI      string
		wantSliceGroup string
	}{
		{
			name:      "设备级 仅 gNBID",
			ldn:       "Type=gNB,Mode=SA,gNBID=350251605",
			wantGNBID: "350251605",
		},
		{
			name:      "CU 小区级 gNBID+NrCGI+CUID",
			ldn:       "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1",
			wantGNBID: "350251605", wantNrCGI: "15153", wantCUID: "1",
		},
		{
			name:      "PLMN 行 +PLMNID",
			ldn:       "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1,PLMNID=00101",
			wantGNBID: "350251605", wantNrCGI: "15153", wantCUID: "1", wantPLMNID: "00101",
		},
		{
			name:      "切片 NSSAI 含斜杠",
			ldn:       "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1,NSSAI=0/1/2",
			wantGNBID: "350251605", wantNrCGI: "15153", wantCUID: "1", wantNSSAI: "0/1/2",
		},
		{
			name:           "切片组 SCLICEGROUP 含斜杠",
			ldn:            "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1,SCLICEGROUP=0/1/2",
			wantGNBID:      "350251605",
			wantNrCGI:      "15153",
			wantCUID:       "1",
			wantSliceGroup: "0/1/2",
		},
		{
			name:      "DU 小区级 DUID",
			ldn:       "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,DUID=1",
			wantGNBID: "350251605", wantNrCGI: "15153", wantDUID: "1",
		},
		{
			name:      "DU+PLMN 同时出现",
			ldn:       "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,DUID=1,PLMNID=00101",
			wantGNBID: "350251605", wantNrCGI: "15153", wantDUID: "1", wantPLMNID: "00101",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseObjectLDN(c.ldn)
			if got.Tech != TechNR {
				t.Errorf("Tech: got=%q want=nr", got.Tech)
			}
			if got.GNBID != c.wantGNBID {
				t.Errorf("GNBID: got=%q want=%q", got.GNBID, c.wantGNBID)
			}
			if got.NrCGI != c.wantNrCGI {
				t.Errorf("NrCGI: got=%q want=%q", got.NrCGI, c.wantNrCGI)
			}
			if got.CUID != c.wantCUID {
				t.Errorf("CUID: got=%q want=%q", got.CUID, c.wantCUID)
			}
			if got.DUID != c.wantDUID {
				t.Errorf("DUID: got=%q want=%q", got.DUID, c.wantDUID)
			}
			if got.PLMNID != c.wantPLMNID {
				t.Errorf("PLMNID: got=%q want=%q", got.PLMNID, c.wantPLMNID)
			}
			if got.NSSAI != c.wantNSSAI {
				t.Errorf("NSSAI: got=%q want=%q", got.NSSAI, c.wantNSSAI)
			}
			if got.SliceGroup != c.wantSliceGroup {
				t.Errorf("SliceGroup: got=%q want=%q", got.SliceGroup, c.wantSliceGroup)
			}
			// 5G 行 CellID/Plmn 必须留空（aggregator 不该误触配对）
			if got.CellID != "" || got.Plmn != "" {
				t.Errorf("5G 行不应填 CellID/Plmn：CellID=%q Plmn=%q", got.CellID, got.Plmn)
			}
			// BaseCellID = NrCGI（无 NrCGI 则空——设备级行无小区标识）
			if got.BaseCellID() != c.wantNrCGI {
				t.Errorf("BaseCellID: got=%q want=%q", got.BaseCellID(), c.wantNrCGI)
			}
		})
	}
}

func TestParseObjectLDN_GSM(t *testing.T) {
	cases := []struct {
		name    string
		ldn     string
		wantUid string
	}{
		{"GSM 标准 Uid 含连字符", "Uid=4002-1", "4002-1"},
		{"GSM 另一组", "Uid=4003-1", "4003-1"},
		{"GSM 真机大号", "Uid=1110-101", "1110-101"},
		{"GSM 简单数字", "Uid=4010-1", "4010-1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseObjectLDN(c.ldn)
			if got.Tech != TechGSM {
				t.Errorf("Tech: got=%q want=gsm", got.Tech)
			}
			if got.Uid != c.wantUid {
				t.Errorf("Uid: got=%q want=%q", got.Uid, c.wantUid)
			}
			// GSM 行 CellID/Plmn 必须留空
			if got.CellID != "" || got.Plmn != "" {
				t.Errorf("GSM 行不应填 CellID/Plmn：CellID=%q Plmn=%q", got.CellID, got.Plmn)
			}
			// GSM 行不应填 5G 字段
			if got.GNBID != "" || got.NrCGI != "" {
				t.Errorf("GSM 行不应填 5G 字段：%+v", got)
			}
			if got.BaseCellID() != c.wantUid {
				t.Errorf("BaseCellID: got=%q want=%q", got.BaseCellID(), c.wantUid)
			}
		})
	}
}

func TestParseObjectLDN_Fallback(t *testing.T) {
	cases := []struct {
		name string
		ldn  string
	}{
		{"空串", ""},
		{"完全不认识的串", "SomeUnknownObj=value"},
		{"只有 SubNetwork 段", "SubNetwork=1"},
		{"空白", "   "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseObjectLDN(c.ldn)
			if got.Tech != TechUnknown {
				t.Errorf("Tech: got=%q want=unknown(空字符串)", got.Tech)
			}
			if got.CellID != "" || got.Plmn != "" || got.GNBID != "" || got.NrCGI != "" ||
				got.CUID != "" || got.DUID != "" || got.PLMNID != "" || got.NSSAI != "" ||
				got.SliceGroup != "" || got.Uid != "" {
				t.Errorf("无法识别串应所有字段留空：%+v", got)
			}
			if got.BaseCellID() != "" {
				t.Errorf("BaseCellID 应空：got=%q", got.BaseCellID())
			}
		})
	}
}

// 不 panic：往 ParseObjectLDN 塞各种古怪输入。
func TestParseObjectLDN_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ParseObjectLDN panicked: %v", r)
		}
	}()
	weird := []string{
		"",
		",,,",
		"Cellid=",
		"Cellid",
		"gNBID=,NrCGI=",
		"=value",
		string(make([]byte, 0)),
	}
	for _, s := range weird {
		_ = ParseObjectLDN(s)
	}
}

// 制式判定优先级：gNBID > Uid > Cellid（同串里出现多个特征字段时以 5G/GSM 优先认）。
func TestParseObjectLDN_TechPriority(t *testing.T) {
	// gNBID + Cellid 同串：以 5G 为准（实际不会同时出现，防御性）
	r := ParseObjectLDN("gNBID=1,Cellid=2")
	if r.Tech != TechNR {
		t.Errorf("gNBID+Cellid 同串 → 应判定 nr，got=%q", r.Tech)
	}
	if r.CellID != "" {
		t.Errorf("判定 5G 后 CellID 应被清空，got=%q", r.CellID)
	}
	// Uid + Cellid 同串：以 GSM 为准
	r = ParseObjectLDN("Uid=4002-1,Cellid=66")
	if r.Tech != TechGSM {
		t.Errorf("Uid+Cellid 同串 → 应判定 gsm，got=%q", r.Tech)
	}
}

// PLMN 与 PLMNID 必须区分：4G 的 PLMN= 与 5G 的 PLMNID= 名字相似，正则需用 \b 边界。
func TestParseObjectLDN_PlmnVsPlmnID(t *testing.T) {
	// 单独 PLMN（4G）
	r := ParseObjectLDN("Cellid=66,PLMN=46001")
	if r.Plmn != "46001" {
		t.Errorf("4G PLMN 取错：got=%q", r.Plmn)
	}
	if r.PLMNID != "" {
		t.Errorf("4G 行不应填 PLMNID：got=%q", r.PLMNID)
	}
	// 单独 PLMNID（5G）
	r = ParseObjectLDN("Type=Cell,gNBID=1,NrCGI=2,CUID=3,PLMNID=00101")
	if r.PLMNID != "00101" {
		t.Errorf("5G PLMNID 取错：got=%q", r.PLMNID)
	}
	if r.Plmn != "" {
		t.Errorf("5G 行 Plmn 应留空：got=%q", r.Plmn)
	}
}
