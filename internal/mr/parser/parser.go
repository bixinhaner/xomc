package parser

import (
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// ErrNotSupported is returned when an MR type is not supported for a carrier.
var ErrNotSupported = errors.New("MR type not supported for this carrier")

// MRRecord represents a single measurement record.
type MRRecord struct {
	Time            time.Time              `json:"time"`
	CellID          string                 `json:"cell_id"`
	MeasurementData map[string]interface{} `json:"measurement_data"`
}

// MRData is the result of parsing an MR file.
type MRData struct {
	FileID      uuid.UUID `json:"file_id"`
	DeviceSN    string    `json:"device_sn"`
	MRType      string    `json:"mr_type"`
	CollectTime time.Time `json:"collect_time"`
	Records     []MRRecord `json:"records"`
}

// MRParser parses MR XML files into structured data.
type MRParser interface {
	Parse(r io.Reader, carrier model.CarrierCode) (*MRData, error)
}
