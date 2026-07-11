package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

func TestStreamCSVToObject_Success(t *testing.T) {
	src := &sliceSource{batches: [][]ExportRow{
		{{Device: "d1", MetricCode: "K1", Value: 1}},
		{{Device: "d2", MetricCode: "K2", Value: 2}, {Device: "d3", MetricCode: "K3", Value: 3}},
	}}
	up := &stubUploader{}
	cols := []WideColumn{{Code: "K1", Type: "kpi", Name: "K1"}, {Code: "K2", Type: "kpi", Name: "K2"}, {Code: "K3", Type: "kpi", Name: "K3"}}

	// 三设备各一指标、时间同（零值）→ 三个行键摊成三横行。
	res, err := streamCSVToObject(context.Background(), up, "bkt", "obj.csv", src, cols, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(3), res.RowCount) // 3 横行
	assert.Greater(t, res.FileSize, int64(0))

	// 上传体头三字节 BOM。
	require.GreaterOrEqual(t, len(up.gotBody), 3)
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, up.gotBody[:3])

	// CSV 解析：表头 + 3 横行。
	body := strings.TrimPrefix(string(up.gotBody), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	assert.Len(t, recs, 4)
}

func TestStreamCSVToObject_EnglishFixedHeaders(t *testing.T) {
	src := &sliceSource{batches: nil}
	up := &stubUploader{}
	layout := csvLayout{
		FirstColHeader: "Product",
		Locale:         appcontext.LocaleEN,
	}

	_, err := streamCSVToObject(context.Background(), up, "bkt", "obj.csv", src, nil, layout, nil)
	require.NoError(t, err)

	body := strings.TrimPrefix(string(up.gotBody), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 1)
	assert.Equal(t, []string{"Start Time", "End Time", "Product"}, recs[0])
}

func TestStreamCSVToObject_MissingMetricCellUsesPlaceholder(t *testing.T) {
	src := &sliceSource{batches: [][]ExportRow{{
		{Device: "d1", MetricCode: "K1", Value: 1},
	}}}
	up := &stubUploader{}
	cols := []WideColumn{{Code: "K1", Type: "kpi", Name: "K1"}, {Code: "K2", Type: "kpi", Name: "K2"}}

	_, err := streamCSVToObject(context.Background(), up, "bkt", "obj.csv", src, cols, csvLayout{FirstColHeader: "设备", IncludeCell: true, MissingMetricValuePlaceholder: "-"}, nil)
	require.NoError(t, err)

	body := strings.TrimPrefix(string(up.gotBody), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 2)
	assert.Equal(t, "-", recs[1][6])
}

func TestStreamCSVToObject_NullMetricValueUsesPlaceholder(t *testing.T) {
	src := &sliceSource{batches: [][]ExportRow{{
		{Device: "d1", MetricCode: "K1", Value: math.NaN()},
	}}}
	up := &stubUploader{}
	cols := []WideColumn{{Code: "K1", Type: "kpi", Name: "K1"}}

	_, err := streamCSVToObject(context.Background(), up, "bkt", "obj.csv", src, cols, csvLayout{FirstColHeader: "设备", IncludeCell: true, MissingMetricValuePlaceholder: "-"}, nil)
	require.NoError(t, err)

	body := strings.TrimPrefix(string(up.gotBody), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	require.Len(t, recs, 2)
	assert.Equal(t, "-", recs[1][5])
}

func TestStreamCSVToObject_SourceError_Propagates(t *testing.T) {
	src := &sliceSource{err: errors.New("query boom")}
	up := &stubUploader{}
	_, err := streamCSVToObject(context.Background(), up, "bkt", "obj.csv", src, nil, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query boom")
}

func TestStreamCSVToObject_UploadError_Propagates(t *testing.T) {
	src := &sliceSource{batches: [][]ExportRow{{{Device: "d", MetricCode: "K"}}}}
	up := &stubUploader{uploadErr: errors.New("upload boom")}
	_, err := streamCSVToObject(context.Background(), up, "bkt", "obj.csv", src, nil, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil)
	require.Error(t, err)
}

func TestStreamCSVToObject_EmptySource(t *testing.T) {
	// 无数据：只写 BOM + 表头，行数 0，仍上传成功（空结果合法）。
	src := &sliceSource{batches: nil}
	up := &stubUploader{}
	res, err := streamCSVToObject(context.Background(), up, "bkt", "obj.csv", src, nil, csvLayout{FirstColHeader: "设备", IncludeCell: true}, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(0), res.RowCount)
	body := strings.TrimPrefix(string(up.gotBody), string(utf8BOM))
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	require.NoError(t, err)
	assert.Len(t, recs, 1) // 仅表头
}

// 确保 BOM 字节序常量没漂。
func TestUTF8BOM_Bytes(t *testing.T) {
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, utf8BOM)
	var buf bytes.Buffer
	_, err := NewWideCSVWriter(&buf, "设备", false, true, nil)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(buf.Bytes(), utf8BOM))
}
