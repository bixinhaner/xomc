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

// MRTypeSupportChecker reports whether a carrier collects a given MR type.
// Implemented by *carrier.CarrierRegistry; injected so the MRE parser no longer
// hardcodes "if carrier == cucc" — the carrier divergence lives in the Carrier
// adapters (#17). Decoupled via this local interface to avoid mr/parser taking a
// hard dependency on the carrier package.
type MRTypeSupportChecker interface {
	// SupportsMRType reports whether the given carrier supports the MR type.
	// Returns false when the carrier itself is unknown to the registry.
	SupportsMRType(carrierCode model.CarrierCode, mrType model.MRType) bool
}

// MREParser parses MRE (Measurement Report - Equipment/Terminal) XML files.
// MRE contains UE capability information. Whether a carrier collects MRE is
// decided by the injected MRTypeSupportChecker (Carrier adapters), not by the
// parser — e.g. CUCC (China Unicom) does not support MRE.
type MREParser struct {
	support MRTypeSupportChecker
}

// NewMREParser creates a new MRE parser.
//
// When support is nil the parser falls back to the built-in carrier-support
// table (defaultMRESupport) so legacy callers and tests keep working; wiring
// code should pass the live *carrier.CarrierRegistry to keep the support
// decision in one place.
func NewMREParser(support ...MRTypeSupportChecker) *MREParser {
	var s MRTypeSupportChecker = defaultMRESupport{}
	if len(support) > 0 && support[0] != nil {
		s = support[0]
	}
	return &MREParser{support: s}
}

// defaultMRESupport is the fallback support table used when no carrier registry
// is injected. It mirrors the Carrier adapters: every carrier supports MRE
// except CUCC. Keeping the rule here (rather than `if carrier == cucc` inline)
// means there is a single, named place describing the divergence even for the
// dependency-free fallback path.
type defaultMRESupport struct{}

func (defaultMRESupport) SupportsMRType(carrierCode model.CarrierCode, mrType model.MRType) bool {
	if mrType != model.MRTypeMRE {
		return true
	}
	return carrierCode != model.CarrierCUCC
}

func (p *MREParser) Parse(r io.Reader, carrier model.CarrierCode) (*MRData, error) {
	if p.support != nil && !p.support.SupportsMRType(carrier, model.MRTypeMRE) {
		return nil, ErrNotSupported
	}

	decoder := xml.NewDecoder(r)
	data := &MRData{MRType: "mre"}

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
			return nil, fmt.Errorf("decode MRE XML: %w", err)
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
		return nil, fmt.Errorf("no MRE records found")
	}

	return data, nil
}
