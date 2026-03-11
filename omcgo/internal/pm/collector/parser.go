package collector

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// PMFileContent holds the parsed content of a PM XML file.
type PMFileContent struct {
	DeviceSN    string
	CollectTime time.Time
	Granularity int // minutes
	Counters    []model.PMCounter
}

// PMXMLParser parses 3GPP 32.435 format PM XML files using streaming XML decoder.
type PMXMLParser struct{}

// NewPMXMLParser creates a new PM XML parser.
func NewPMXMLParser() *PMXMLParser {
	return &PMXMLParser{}
}

// XML structures for 3GPP 32.435 PM file format.
type xmlMeasInfo struct {
	MeasInfoId string           `xml:"measInfoId,attr"`
	GranPeriod xmlGranPeriod    `xml:"granPeriod"`
	MeasTypes  []xmlMeasType    `xml:"measType"`
	MeasValues []xmlMeasValue   `xml:"measValue"`
}

type xmlGranPeriod struct {
	Duration string `xml:"duration,attr"`
	EndTime  string `xml:"endTime,attr"`
}

type xmlMeasType struct {
	P    string `xml:"p,attr"`
	Name string `xml:",chardata"`
}

type xmlMeasValue struct {
	MeasObjLdn string  `xml:"measObjLdn,attr"`
	Results    []xmlR  `xml:"r"`
}

type xmlR struct {
	P     string `xml:"p,attr"`
	Value string `xml:",chardata"`
}

type xmlManagedElement struct {
	LocalDn   string `xml:"localDn,attr"`
	SwVersion string `xml:"swVersion,attr"`
}

// Parse parses a PM XML file from the given reader.
func (p *PMXMLParser) Parse(r io.Reader, deviceID uuid.UUID) (*PMFileContent, error) {
	decoder := xml.NewDecoder(r)

	content := &PMFileContent{}
	var inMeasData bool

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode pm xml: %w", err)
		}

		se, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		switch se.Name.Local {
		case "managedElement":
			var me xmlManagedElement
			if err := decoder.DecodeElement(&me, &se); err != nil {
				return nil, fmt.Errorf("decode managedElement: %w", err)
			}
			content.DeviceSN = extractDeviceSN(me.LocalDn)
			inMeasData = true

		case "measInfo":
			if !inMeasData {
				continue
			}
			var mi xmlMeasInfo
			if err := decoder.DecodeElement(&mi, &se); err != nil {
				return nil, fmt.Errorf("decode measInfo: %w", err)
			}

			// Parse granularity and end time
			granSeconds := parseDuration(mi.GranPeriod.Duration)
			content.Granularity = granSeconds / 60
			if content.Granularity == 0 {
				content.Granularity = 15
			}

			collectTime, timeErr := time.Parse(time.RFC3339, mi.GranPeriod.EndTime)
			if timeErr != nil {
				collectTime = time.Now()
			}
			content.CollectTime = collectTime

			// Build counter name index
			typeIndex := make(map[string]string, len(mi.MeasTypes))
			for _, mt := range mi.MeasTypes {
				typeIndex[mt.P] = strings.TrimSpace(mt.Name)
			}

			// Extract counters from each measValue
			for _, mv := range mi.MeasValues {
				cellID := extractCellID(mv.MeasObjLdn)
				counterGroup := mi.MeasInfoId

				for _, r := range mv.Results {
					counterName, ok := typeIndex[r.P]
					if !ok {
						continue
					}
					value, parseErr := strconv.ParseFloat(strings.TrimSpace(r.Value), 64)
					if parseErr != nil {
						continue
					}
					content.Counters = append(content.Counters, model.PMCounter{
						Time:         collectTime,
						DeviceID:     deviceID,
						CellID:       cellID,
						CounterGroup: counterGroup,
						CounterName:  counterName,
						CounterValue: value,
						Granularity:  content.Granularity,
					})
				}
			}
		}
	}

	if len(content.Counters) == 0 {
		return nil, fmt.Errorf("no counters found in pm xml")
	}

	return content, nil
}

// extractDeviceSN extracts the device serial number from a localDn string.
func extractDeviceSN(localDn string) string {
	parts := strings.Split(localDn, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && kv[0] == "MeContext" {
			return kv[1]
		}
	}
	if len(parts) > 0 {
		kv := strings.SplitN(parts[len(parts)-1], "=", 2)
		if len(kv) == 2 {
			return kv[1]
		}
	}
	return localDn
}

// extractCellID extracts the cell ID from a measObjLdn string.
func extractCellID(measObjLdn string) string {
	parts := strings.Split(measObjLdn, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 && (kv[0] == "CellId" || kv[0] == "NRCellDU" || kv[0] == "NRCellCU") {
			return kv[1]
		}
	}
	return measObjLdn
}

// parseDuration parses an ISO 8601 duration like "PT900S" to seconds.
func parseDuration(d string) int {
	d = strings.TrimPrefix(d, "PT")
	d = strings.TrimPrefix(d, "pt")
	if strings.HasSuffix(d, "S") || strings.HasSuffix(d, "s") {
		d = strings.TrimSuffix(d, "S")
		d = strings.TrimSuffix(d, "s")
		v, err := strconv.Atoi(d)
		if err != nil {
			return 900
		}
		return v
	}
	if strings.HasSuffix(d, "M") || strings.HasSuffix(d, "m") {
		d = strings.TrimSuffix(d, "M")
		d = strings.TrimSuffix(d, "m")
		v, err := strconv.Atoi(d)
		if err != nil {
			return 900
		}
		return v * 60
	}
	return 900
}
