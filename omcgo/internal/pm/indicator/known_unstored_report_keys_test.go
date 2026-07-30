package indicator

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKnownUnstoredReportKeysENBCoversBundledBLQSampleOutsideIndicatorLibrary(t *testing.T) {
	registered := readCounterReportKeysFromIndicatorLibrary(t, "../../../data/indicator-library/enb/*.xml")
	for _, key := range KnownUnstoredReportKeys(DeviceTypeENB) {
		require.NotEmpty(t, key)
		require.NotContains(t, registered, key, "显式未入库目录不应重复登记指标库已有 report_key")
		registered[key] = struct{}{}
	}

	reported := readMeasTypeNames(t, "../../../test/pmperf/A20260611.0815+0800-0830+0800_48BF74.120299024119AA05000.xml")
	var missing []string
	for key := range reported {
		if _, ok := registered[key]; !ok {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	require.Empty(t, missing, "内置 BLQ 真机样本出现新上报名时必须登记指标定义或明确未入库处置")
}

func readCounterReportKeysFromIndicatorLibrary(t *testing.T, pattern string) map[string]struct{} {
	t.Helper()
	paths, err := filepath.Glob(pattern)
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	out := make(map[string]struct{})
	for _, path := range paths {
		f, err := os.Open(path)
		require.NoError(t, err)
		decoder := xml.NewDecoder(f)
		for {
			token, err := decoder.Token()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			start, ok := token.(xml.StartElement)
			if !ok || start.Name.Local != "indicator" {
				continue
			}
			var reportKey, isCounter string
			for _, attr := range start.Attr {
				switch attr.Name.Local {
				case "reportKey":
					reportKey = attr.Value
				case "isCounter":
					isCounter = attr.Value
				}
			}
			if isCounter == "1" && reportKey != "" {
				out[reportKey] = struct{}{}
			}
		}
		require.NoError(t, f.Close())
	}
	return out
}

func readMeasTypeNames(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	out := make(map[string]struct{})
	decoder := xml.NewDecoder(f)
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "measType" {
			continue
		}
		var key string
		require.NoError(t, decoder.DecodeElement(&key, &start))
		if key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}
