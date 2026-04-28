package storage

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_ObjectPath_WithCategory(t *testing.T) {
	path := ObjectPath("backup", "cmcc", "DEV-001", "config.xml")

	// {category}/{carrier}/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}
	parts := strings.Split(path, "/")
	assert.Len(t, parts, 7)
	assert.Equal(t, "backup", parts[0])
	assert.Equal(t, "cmcc", parts[1])
	// parts[2..4] are date components
	assert.Equal(t, "DEV-001", parts[5])
	assert.Equal(t, "config.xml", parts[6])
}

func Test_ObjectPath_EmptyCategory(t *testing.T) {
	path := ObjectPath("", "ctcc", "DEV-X", "log.txt")

	// {carrier}/{YYYY}/{MM}/{DD}/{deviceSN}/{filename}
	parts := strings.Split(path, "/")
	assert.Len(t, parts, 6)
	assert.Equal(t, "ctcc", parts[0])
	assert.Equal(t, "DEV-X", parts[4])
	assert.Equal(t, "log.txt", parts[5])
}

func Test_ObjectPathAt_FixedTimestamp(t *testing.T) {
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	t.Run("with category", func(t *testing.T) {
		path := ObjectPathAt("pm", "cmcc", "SN1", "f.xml", ts)
		assert.Equal(t, "pm/cmcc/2024/06/15/SN1/f.xml", path)
	})

	t.Run("empty category", func(t *testing.T) {
		path := ObjectPathAt("", "cucc", "SN2", "g.txt", ts)
		assert.Equal(t, "cucc/2024/06/15/SN2/g.txt", path)
	})
}

func Test_FirmwarePath(t *testing.T) {
	path := FirmwarePath("img", "BC8910", "v1.2.3", "fw.bin")
	assert.Equal(t, "img/BC8910/v1.2.3/fw.bin", path)
}

func Test_ExchangePath(t *testing.T) {
	path := ExchangePath("import", "user42", "devices.csv")

	parts := strings.Split(path, "/")
	// {category}/{YYYY/MM/DD}/{userID}/{filename} → 6 parts
	assert.Len(t, parts, 6)
	assert.Equal(t, "import", parts[0])
	assert.Equal(t, "user42", parts[4])
	assert.Equal(t, "devices.csv", parts[5])
}

func Test_DataModelPath(t *testing.T) {
	path := DataModelPath("cmcc", "001E5F", "WR8500", "DEV-A")

	// datamodel/cmcc/001E5F/WR8500/DEV-A_<ts>.xml
	assert.True(t, strings.HasPrefix(path, "datamodel/cmcc/001E5F/WR8500/DEV-A_"), "got %s", path)
	assert.True(t, strings.HasSuffix(path, ".xml"), "got %s", path)
}

func Test_DatePrefix_Format(t *testing.T) {
	prefix := DatePrefix()
	parts := strings.Split(prefix, "/")
	assert.Len(t, parts, 3, "expected YYYY/MM/DD format, got %s", prefix)
	assert.Len(t, parts[0], 4, "year should be 4 digits")
	assert.Len(t, parts[1], 2, "month should be 2 digits")
	assert.Len(t, parts[2], 2, "day should be 2 digits")
}
