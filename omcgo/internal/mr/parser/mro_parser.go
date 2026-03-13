package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
)

// MROParser parses MRO (Measurement Report - Optimization) XML files.
// MRO contains per-UE measurement samples for handover optimization,
// including RSRP, RSRQ, SINR values for serving and neighbor cells.
type MROParser struct{}

// NewMROParser creates a new MRO parser.
func NewMROParser() *MROParser {
	return &MROParser{}
}

// MRO XML structures
type mroFileHeader struct {
	StartTime string `xml:"startTime,attr"`
}

type mroENodeB struct {
	ID string `xml:"id,attr"`
}

type mroMeasurement struct {
	MRType string     `xml:"mrType,attr"`
	Object []mroObject `xml:"object"`
}

type mroObject struct {
	ID      string   `xml:"id,attr"`
	MmeUeS1apId string `xml:"MmeUeS1apId,attr"`
	TimeStamp   string `xml:"TimeStamp,attr"`
	V      []string `xml:"v"`
}

func (p *MROParser) Parse(r io.Reader, carrier model.CarrierCode) (*MRData, error) {
	decoder := xml.NewDecoder(r)
	data := &MRData{MRType: "mro"}

	var inMeasurement bool
	var currentCellID string
	var collectTime time.Time
	var headerFields []string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode MRO XML: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			switch se.Name.Local {
			case "fileHeader":
				for _, attr := range se.Attr {
					if attr.Name.Local == "startTime" {
						if t, err := time.Parse("2006-01-02T15:04:05Z", attr.Value); err == nil {
							collectTime = t
						} else if t, err := time.Parse("2006-01-02T15:04:05.000Z", attr.Value); err == nil {
							collectTime = t
						}
					}
				}
				data.CollectTime = collectTime

			case "eNB":
				for _, attr := range se.Attr {
					if attr.Name.Local == "id" {
						data.DeviceSN = attr.Value
					}
				}

			case "measurement":
				inMeasurement = true
				headerFields = nil

			case "smr":
				if inMeasurement {
					var content string
					if err := decoder.DecodeElement(&content, &se); err == nil {
						headerFields = strings.Fields(strings.TrimSpace(content))
					}
				}

			case "object":
				if inMeasurement {
					for _, attr := range se.Attr {
						if attr.Name.Local == "id" {
							currentCellID = attr.Value
						}
					}
				}

			case "v":
				if inMeasurement {
					var content string
					if err := decoder.DecodeElement(&content, &se); err == nil {
						values := strings.Fields(strings.TrimSpace(content))
						record := MRRecord{
							Time:            collectTime,
							CellID:          currentCellID,
							MeasurementData: make(map[string]interface{}),
						}
						for i, v := range values {
							if i < len(headerFields) {
								if fv, err := strconv.ParseFloat(v, 64); err == nil {
									record.MeasurementData[headerFields[i]] = fv
								} else {
									record.MeasurementData[headerFields[i]] = v
								}
							}
						}
						if len(record.MeasurementData) > 0 {
							data.Records = append(data.Records, record)
						}
					}
				}
			}

		case xml.EndElement:
			if se.Name.Local == "measurement" {
				inMeasurement = false
			}
		}
	}

	if len(data.Records) == 0 {
		return nil, fmt.Errorf("no MRO records found")
	}

	return data, nil
}
