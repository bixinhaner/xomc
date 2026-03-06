package collector

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			wantCounters:    4, // 2 counters * 2 cells
			wantGranularity: 15,
			wantCells:       []string{"Cell1", "Cell2"},
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
			wantCounters: 3, // 1 + 2 counters for 1 cell
			wantCells:    []string{"NRCell1"},
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
			wantCounters:    1,
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
	require.Len(t, result.Counters, 2)

	// Verify counter values
	counterMap := make(map[string]float64)
	for _, c := range result.Counters {
		counterMap[c.CounterName] = c.CounterValue
	}
	assert.Equal(t, float64(1000), counterMap["rrc_conn_setup_att"])
	assert.Equal(t, float64(950), counterMap["rrc_conn_setup_succ"])

	// Verify metadata
	for _, c := range result.Counters {
		assert.Equal(t, "Cell1", c.CellID)
		assert.Equal(t, "PM_Counters", c.CounterGroup)
		assert.Equal(t, 15, c.Granularity)
		assert.Equal(t, deviceID, c.DeviceID)
	}
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
		{"CellId=Cell1", "Cell1"},
		{"CellId=NRCell-01", "NRCell-01"},
		{"SubNetwork=1,CellId=Cell2", "Cell2"},
		// Fallback: return whole string when no known key
		{"SomeObj=value", "SomeObj=value"},
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
