package imsparam

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestScanParamFileAllowsNullDeviceSN(t *testing.T) {
	file, err := scanParamFile(nullDeviceSNRow{})
	if err != nil {
		t.Fatalf("scanParamFile returned error: %v", err)
	}
	if file.DeviceSN != "" {
		t.Fatalf("DeviceSN = %q, want empty string for NULL database value", file.DeviceSN)
	}
}

type nullDeviceSNRow struct{}

func (nullDeviceSNRow) Scan(dest ...any) error {
	*dest[0].(*uuid.UUID) = uuid.New()
	*dest[1].(*string) = "FT_ImsCore_User_Setting_UD"
	*dest[2].(*string) = "params.bin"
	*dest[3].(*string) = "config-backup"
	*dest[4].(*string) = "ims-params/params.bin"
	*dest[5].(**string) = nil
	*dest[6].(*int64) = 1
	*dest[7].(**string) = nil
	*dest[8].(**string) = nil
	*dest[9].(**string) = nil
	*dest[10].(*time.Time) = time.Now()
	*dest[11].(*time.Time) = time.Now()
	return nil
}
