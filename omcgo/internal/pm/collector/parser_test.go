package collector

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oneByteReader 每次只回 1 字节，模拟最碎的流式输入（MinIO 对象流极端情况），
// 用于验证流式解析对任意分块边界都产出一致结果（issue #14）。
type oneByteReader struct {
	data []byte
	pos  int
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

func TestPMXMLParser_Parse(t *testing.T) {
	testDeviceID := uuid.New()

	tests := []struct {
		name            string
		xml             string
		wantErr         bool
		wantErrMsg      string
		wantDeviceSN    string
		wantCounters    int
		wantGranularity int
		wantCells       []string
	}{
		{
			name: "valid PM XML with two cells",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc" vendorName="TestVendor" fileFormatVersion="32.435 V10.0"/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001" swVersion="V1.0"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-03-06T15:00:00+08:00"/>
      <measType p="1">rrc_conn_setup_att</measType>
      <measType p="2">rrc_conn_setup_succ</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">1000</r>
        <r p="2">950</r>
      </measValue>
      <measValue measObjLdn="CellId=Cell2">
        <r p="1">500</r>
        <r p="2">490</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`,
			wantErr:         false,
			wantDeviceSN:    "eNB001",
			wantCounters:    6, // 2 counters * 2 cells + ISSUE-389 注入每小区 1 条统计时长（2 cells）
			wantGranularity: 15,
			wantCells:       []string{"CellId=Cell1", "CellId=Cell2"},
		},
		{
			name: "valid PM XML with multiple measInfo blocks",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc" vendorName="TestVendor"/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=gNB002"/>
    <measInfo measInfoId="Accessibility">
      <granPeriod duration="PT15M" endTime="2026-03-06T15:15:00+08:00"/>
      <measType p="1">rrc_conn_setup_att</measType>
      <measValue measObjLdn="CellId=NRCell1">
        <r p="1">200</r>
      </measValue>
    </measInfo>
    <measInfo measInfoId="Throughput">
      <granPeriod duration="PT900S" endTime="2026-03-06T15:15:00+08:00"/>
      <measType p="1">pdcp_sdu_dl_volume</measType>
      <measType p="2">pdcp_sdu_ul_volume</measType>
      <measValue measObjLdn="CellId=NRCell1">
        <r p="1">50000000</r>
        <r p="2">10000000</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`,
			wantErr:      false,
			wantDeviceSN: "gNB002",
			wantCounters: 4, // 1 + 2 counters for 1 cell + ISSUE-389 注入 1 条统计时长（NRCell1 跨 2 measInfo 去重为 1）
			wantCells:    []string{"CellId=NRCell1"},
		},
		{
			name:       "empty file",
			xml:        "",
			wantErr:    true,
			wantErrMsg: "no counters found",
		},
		{
			name:       "malformed XML",
			xml:        `<?xml version="1.0"?><measCollecFile><unclosed`,
			wantErr:    true,
			wantErrMsg: "decode",
		},
		{
			name: "XML with no counters produces error",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc"/>
  <measData>
  </measData>
</measCollecFile>`,
			wantErr:    true,
			wantErrMsg: "no counters found",
		},
		{
			name: "hourly granularity",
			xml: `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader/>
  <measData>
    <managedElement localDn="MeContext=eNB003"/>
    <measInfo measInfoId="HourlyStats">
      <granPeriod duration="PT60M" endTime="2026-03-06T16:00:00+08:00"/>
      <measType p="1">total_traffic</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">999</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`,
			wantErr:         false,
			wantDeviceSN:    "eNB003",
			wantCounters:    2, // 1 counter + ISSUE-389 注入 1 条统计时长（Cell1）
			wantGranularity: 60,
		},
	}

	parser := NewPMXMLParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(strings.NewReader(tt.xml), testDeviceID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.wantDeviceSN, result.DeviceSN)
			assert.Equal(t, tt.wantCounters, len(result.Counters))

			if tt.wantGranularity > 0 {
				assert.Equal(t, tt.wantGranularity, result.Granularity)
			}

			if tt.wantCells != nil {
				cellSet := make(map[string]struct{})
				for _, c := range result.Counters {
					cellSet[c.CellID] = struct{}{}
				}
				for _, wantCell := range tt.wantCells {
					_, found := cellSet[wantCell]
					assert.True(t, found, "expected cell %s in counters", wantCell)
				}
			}

			// Verify device ID is set on all counters
			for _, c := range result.Counters {
				assert.Equal(t, testDeviceID, c.DeviceID)
			}
		})
	}
}

func TestPMXMLParser_ParseCounterValues(t *testing.T) {
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-03-06T15:00:00+08:00"/>
      <measType p="1">rrc_conn_setup_att</measType>
      <measType p="2">rrc_conn_setup_succ</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">1000</r>
        <r p="2">950</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`

	deviceID := uuid.New()
	parser := NewPMXMLParser()
	result, err := parser.Parse(strings.NewReader(xml), deviceID)
	require.NoError(t, err)
	// ISSUE-389: 2 真实 counter + 1 条注入统计时长（Cell1）。
	require.Len(t, result.Counters, 3)

	// Verify counter values
	counterMap := make(map[string]float64)
	for _, c := range result.Counters {
		counterMap[c.CounterName] = c.CounterValue
	}
	assert.Equal(t, float64(1000), counterMap["rrc_conn_setup_att"])
	assert.Equal(t, float64(950), counterMap["rrc_conn_setup_succ"])
	// ISSUE-389: 注入的统计时长 = granPeriod 真实周期（PT900S → 900 秒）。
	assert.Equal(t, float64(900), counterMap[StatisDurationReportKey])

	// Verify metadata
	for _, c := range result.Counters {
		assert.Equal(t, "CellId=Cell1", c.CellID)
		assert.Equal(t, "PM_Counters", c.CounterGroup)
		assert.Equal(t, 15, c.Granularity)
		assert.Equal(t, deviceID, c.DeviceID)
	}
}

// ==================== ISSUE-389 阶段2：统计时长注入 ====================

// TestPMXMLParser_InjectsStatisDuration 守死判 statduration-injected-900：
// 喂一个性能文件，断言解析结果里每个小区多出一条「统计时长」计数器，
// 值 = 文件头声明的采集周期（PT900S → 900 秒），CounterName = report_key（下游
// collector 白名单据此改写成各制式统计时长编号）。取文件实际周期、非写死。
func TestPMXMLParser_InjectsStatisDuration(t *testing.T) {
	const xmlTwoCells = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc"/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-03-06T15:00:00+08:00"/>
      <measType p="1">OTHER.CellServiceTime</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">900</r>
      </measValue>
      <measValue measObjLdn="CellId=Cell2">
        <r p="1">900</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`

	parser := NewPMXMLParser()
	result, err := parser.Parse(strings.NewReader(xmlTwoCells), uuid.New())
	require.NoError(t, err)

	// 收集每个小区的统计时长行。
	statisByCell := make(map[string]float64)
	for _, c := range result.Counters {
		if c.CounterName == StatisDurationReportKey {
			statisByCell[c.CellID] = c.CounterValue
		}
	}
	require.Len(t, statisByCell, 2, "每个小区各注入一条统计时长")
	assert.Equal(t, float64(900), statisByCell["CellId=Cell1"], "Cell1 统计时长 = 采集周期 900s")
	assert.Equal(t, float64(900), statisByCell["CellId=Cell2"], "Cell2 统计时长 = 采集周期 900s")
}

// TestPMXMLParser_StatisDurationFollowsRealPeriod 验证统计时长取文件实际采集周期、非写死。
// 文件头声明 PT300S（5 分钟一报）→ 注入的统计时长应为 300，而非写死的 900。
func TestPMXMLParser_StatisDurationFollowsRealPeriod(t *testing.T) {
	const xml5Min = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc"/>
  <measData>
    <managedElement localDn="MeContext=eNB001"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT300S" endTime="2026-03-06T15:05:00+08:00"/>
      <measType p="1">OTHER.CellServiceTime</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">300</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`

	parser := NewPMXMLParser()
	result, err := parser.Parse(strings.NewReader(xml5Min), uuid.New())
	require.NoError(t, err)

	var statis float64
	var found bool
	for _, c := range result.Counters {
		if c.CounterName == StatisDurationReportKey {
			statis, found = c.CounterValue, true
		}
	}
	require.True(t, found, "应注入统计时长")
	assert.Equal(t, float64(300), statis, "统计时长跟随真实采集周期 300s，非写死 900")
}

// TestPMXMLParser_StatisDurationDedupAcrossMeasInfo 验证同一小区出现在多个 measInfo 块
// （接入性 / 吞吐率…）时，统计时长只注入一条，避免聚合后失真（否则 900*N）。
func TestPMXMLParser_StatisDurationDedupAcrossMeasInfo(t *testing.T) {
	const xmlMultiInfo = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc"/>
  <measData>
    <managedElement localDn="MeContext=eNB001"/>
    <measInfo measInfoId="Accessibility">
      <granPeriod duration="PT900S" endTime="2026-03-06T15:00:00+08:00"/>
      <measType p="1">RRC.AttConnEstab</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">100</r>
      </measValue>
    </measInfo>
    <measInfo measInfoId="Service">
      <granPeriod duration="PT900S" endTime="2026-03-06T15:00:00+08:00"/>
      <measType p="1">OTHER.CellServiceTime</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">900</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`

	parser := NewPMXMLParser()
	result, err := parser.Parse(strings.NewReader(xmlMultiInfo), uuid.New())
	require.NoError(t, err)

	count := 0
	for _, c := range result.Counters {
		if c.CounterName == StatisDurationReportKey && c.CellID == "CellId=Cell1" {
			count++
		}
	}
	assert.Equal(t, 1, count, "同小区同采集窗口跨多 measInfo 只注入一条统计时长")
}

func TestExtractDeviceSN(t *testing.T) {
	tests := []struct {
		localDn string
		want    string
	}{
		{"SubNetwork=1,MeContext=eNB001", "eNB001"},
		{"MeContext=gNB002", "gNB002"},
		{"SubNetwork=1,SubNetwork=2,MeContext=eNB003", "eNB003"},
		// When no MeContext, fall back to last key=value
		{"ManagedElement=device123", "device123"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.localDn, func(t *testing.T) {
			got := extractDeviceSN(tt.localDn)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractCellID(t *testing.T) {
	tests := []struct {
		measObjLdn string
		want       string
	}{
		// extractCellID 透传完整 LDN 原串，保留 PLMN 等层级信息
		{"CellId=Cell1", "CellId=Cell1"},
		{"CellId=NRCell-01", "CellId=NRCell-01"},
		{"SubNetwork=1,CellId=Cell2", "SubNetwork=1,CellId=Cell2"},
		{"SomeObj=value", "SomeObj=value"},
		{"Cellid=66", "Cellid=66"},
		{"Cellid=66,PLMN=46001", "Cellid=66,PLMN=46001"},
		{"Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1", "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1"},
		{"Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1,PLMNID=00101", "Type=Cell,Mode=SA,gNBID=350251605,NrCGI=15153,CUID=1,PLMNID=00101"},
		{"Uid=4002-1", "Uid=4002-1"},
		{"Uid=1110-101", "Uid=1110-101"},
	}

	for _, tt := range tests {
		t.Run(tt.measObjLdn, func(t *testing.T) {
			got := extractCellID(tt.measObjLdn)
			assert.Equal(t, tt.want, got)
		})
	}
}



func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		wantSecs int
	}{
		{"PT900S", 900},
		{"PT15M", 900},
		{"PT60M", 3600},
		{"PT3600S", 3600},
		{"PT30M", 1800},
		{"INVALID", 900}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseDuration(tt.input)
			assert.Equal(t, tt.wantSecs, got)
		})
	}
}

// ==================== issue #14: 流式分块解析一致性 ====================

const parityXML = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc" vendorName="TestVendor">
    <measCollec beginTime="2026-05-22T10:00:00+08:00"/>
  </fileHeader>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001" swVersion="V1.0"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-05-22T10:15:00+08:00"/>
      <measType p="1">rrc_conn_setup_att</measType>
      <measType p="2">rrc_conn_setup_succ</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">1000</r>
        <r p="2">950</r>
      </measValue>
      <measValue measObjLdn="CellId=Cell2">
        <r p="1">500</r>
        <r p="2">490</r>
      </measValue>
    </measInfo>
  </measData>
  <fileFooter>
    <measCollec endTime="2026-05-22T10:15:00+08:00"/>
  </fileFooter>
</measCollecFile>`

// TestPMXMLParser_StreamingParity 验证：把同一份 XML 经"整串 reader"与"每次 1 字节
// 的碎块 reader"分别解析，产出的 counters / 设备 SN / 粒度 / 三时间字段完全一致。
// 证明 issue #14 的 bufio 流式化没有改变解析语义（输出 = 改造前）。
func TestPMXMLParser_StreamingParity(t *testing.T) {
	deviceID := uuid.New()
	parser := NewPMXMLParser()

	whole, err := parser.Parse(strings.NewReader(parityXML), deviceID)
	require.NoError(t, err)
	require.NotNil(t, whole)

	chunked, err := parser.Parse(&oneByteReader{data: []byte(parityXML)}, deviceID)
	require.NoError(t, err)
	require.NotNil(t, chunked)

	assert.Equal(t, whole.DeviceSN, chunked.DeviceSN)
	assert.Equal(t, whole.Granularity, chunked.Granularity)
	assert.True(t, whole.CollectTime.Equal(chunked.CollectTime), "CollectTime 应一致")
	assert.True(t, whole.FileBeginTime.Equal(chunked.FileBeginTime), "FileBeginTime 应一致")
	assert.True(t, whole.FileEndTime.Equal(chunked.FileEndTime), "FileEndTime 应一致")
	require.Equal(t, len(whole.Counters), len(chunked.Counters), "counter 数应一致")

	// 逐 (cell, name) → value 比对，顺序无关。
	type ck struct{ cell, name string }
	wm := make(map[ck]float64, len(whole.Counters))
	for _, c := range whole.Counters {
		wm[ck{c.CellID, c.CounterName}] = c.CounterValue
	}
	for _, c := range chunked.Counters {
		v, ok := wm[ck{c.CellID, c.CounterName}]
		require.True(t, ok, "碎块解析多出 counter %s/%s", c.CellID, c.CounterName)
		assert.Equal(t, v, c.CounterValue, "counter %s/%s 值不一致", c.CellID, c.CounterName)
	}
}

// ==================== T-0164-P4 / G4 新增：三时间字段测试 ====================

// TestPMXMLParser_FileHeaderFooterTime_FullExtraction 验证完整含 fileHeader/measCollec
// 与 fileFooter/measCollec 的 XML 文件解析后三字段命中。
func TestPMXMLParser_FileHeaderFooterTime_FullExtraction(t *testing.T) {
	const xmlWithFullTime = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc" vendorName="TestVendor">
    <measCollec beginTime="2026-05-22T10:00:00+08:00"/>
  </fileHeader>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001" swVersion="V1.0"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-05-22T10:15:00+08:00"/>
      <measType p="1">rrc_conn_setup_att</measType>
      <measValue measObjLdn="CellId=Cell1">
        <r p="1">100</r>
      </measValue>
    </measInfo>
  </measData>
  <fileFooter>
    <measCollec endTime="2026-05-22T10:15:00+08:00"/>
  </fileFooter>
</measCollecFile>`

	parser := NewPMXMLParser()
	content, err := parser.Parse(strings.NewReader(xmlWithFullTime), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, content)

	wantBegin, _ := time.Parse(time.RFC3339, "2026-05-22T10:00:00+08:00")
	wantEnd, _ := time.Parse(time.RFC3339, "2026-05-22T10:15:00+08:00")
	require.True(t, content.FileBeginTime.Equal(wantBegin), "FileBeginTime got %v want %v", content.FileBeginTime, wantBegin)
	require.True(t, content.FileEndTime.Equal(wantEnd), "FileEndTime got %v want %v", content.FileEndTime, wantEnd)
	require.False(t, content.IngestTime.IsZero(), "IngestTime should always be set")
}

// TestPMXMLParser_FileHeaderFooterTime_FallbackInference 验证缺 fileHeader/measCollec
// 与 fileFooter/measCollec 的老格式 XML，fallback 推断三字段。
func TestPMXMLParser_FileHeaderFooterTime_FallbackInference(t *testing.T) {
	const xmlOldFormat = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc" vendorName="TestVendor"/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-05-22T10:15:00+08:00"/>
      <measType p="1">x</measType>
      <measValue measObjLdn="CellId=A">
        <r p="1">1</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`

	parser := NewPMXMLParser()
	content, err := parser.Parse(strings.NewReader(xmlOldFormat), uuid.New())
	require.NoError(t, err)
	require.NotNil(t, content)

	// fileFooter 缺 → FileEndTime fallback = collectTime
	wantEnd, _ := time.Parse(time.RFC3339, "2026-05-22T10:15:00+08:00")
	require.True(t, content.FileEndTime.Equal(wantEnd), "FileEndTime fallback should equal collectTime")

	// fileHeader/measCollec 缺 → FileBeginTime fallback = FileEndTime - granularity (15min)
	wantBegin := wantEnd.Add(-15 * time.Minute)
	require.True(t, content.FileBeginTime.Equal(wantBegin), "FileBeginTime fallback should equal FileEndTime - 15min")

	require.False(t, content.IngestTime.IsZero())
}

// TestPMXMLParser_FileHeaderFooterTime_PartialPresent 验证只有 fileFooter/measCollec
// 而无 fileHeader/measCollec 的混合场景。
func TestPMXMLParser_FileHeaderFooterTime_PartialPresent(t *testing.T) {
	const xmlPartial = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader dnPrefix="DC=cmcc"/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-05-22T10:15:00+08:00"/>
      <measType p="1">x</measType>
      <measValue measObjLdn="CellId=A">
        <r p="1">1</r>
      </measValue>
    </measInfo>
  </measData>
  <fileFooter>
    <measCollec endTime="2026-05-22T10:30:00+08:00"/>
  </fileFooter>
</measCollecFile>`

	parser := NewPMXMLParser()
	content, err := parser.Parse(strings.NewReader(xmlPartial), uuid.New())
	require.NoError(t, err)

	// fileFooter/measCollec 命中 → FileEndTime = 2026-05-22T10:30:00（不走 collectTime fallback）
	wantEnd, _ := time.Parse(time.RFC3339, "2026-05-22T10:30:00+08:00")
	require.True(t, content.FileEndTime.Equal(wantEnd))

	// fileHeader/measCollec 缺 → FileBeginTime = FileEndTime - 15min（基于 footer，非 collectTime）
	wantBegin := wantEnd.Add(-15 * time.Minute)
	require.True(t, content.FileBeginTime.Equal(wantBegin))
}

// TestPMXMLParser_IngestTime_AlwaysSet 验证 IngestTime 总是非 zero（即使其他时间都缺）。
func TestPMXMLParser_IngestTime_AlwaysSet(t *testing.T) {
	const xmlNoTime = `<?xml version="1.0" encoding="UTF-8"?>
<measCollecFile>
  <fileHeader/>
  <measData>
    <managedElement localDn="SubNetwork=1,MeContext=eNB001"/>
    <measInfo measInfoId="PM_Counters">
      <granPeriod duration="PT900S" endTime="2026-05-22T10:15:00+08:00"/>
      <measType p="1">x</measType>
      <measValue measObjLdn="CellId=A">
        <r p="1">1</r>
      </measValue>
    </measInfo>
  </measData>
</measCollecFile>`

	parser := NewPMXMLParser()
	before := time.Now()
	content, err := parser.Parse(strings.NewReader(xmlNoTime), uuid.New())
	after := time.Now()
	require.NoError(t, err)

	require.False(t, content.IngestTime.IsZero(), "IngestTime must always be set")
	require.True(t, !content.IngestTime.Before(before) && !content.IngestTime.After(after),
		"IngestTime should be within Parse call window")
}
