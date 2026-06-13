package collector

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/compress"
)

// TestPMIngest_GzipAndPlaintextEquivalent 是 issue #321 的回归测试。
//
// 真机 CPE 与 cpe_simulator.py 按 TR-069 标准把 PM XML gzip 成 .xml.gz 上传，
// ACS 对 PM 文件原样存入 MinIO（不解压），故 MinIO 里是 gzip 字节。修复前 collector
// 直接把对象流喂给 xml.Decoder，撞 gzip 魔数解析失败 —— 真机 .gz 文件无法入库
// （合成压测用明文 .xml 掩盖了该问题）。本测试证明：经 compress.MaybeGunzip 透明
// 解压后，gzip 与明文两条路径解析结果一致。
func TestPMIngest_GzipAndPlaintextEquivalent(t *testing.T) {
	const xml = `<?xml version="1.0" encoding="UTF-8"?>
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
    </measInfo>
  </measData>
</measCollecFile>`

	deviceID := uuid.New()
	parser := NewPMXMLParser()

	// 明文路径（基线）：MaybeGunzip 原样透传。
	plainReader, applied, err := compress.MaybeGunzip(bytes.NewReader([]byte(xml)))
	require.NoError(t, err)
	assert.False(t, applied, "明文不应被识别为 gzip")
	plain, err := parser.Parse(io.LimitReader(plainReader, maxPMFileBytes), deviceID)
	require.NoError(t, err)
	require.NotEmpty(t, plain.Counters, "明文基线应解析出 counter")

	// gzip 路径（真机/模拟器格式）：MaybeGunzip 透明解压。
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err = gw.Write([]byte(xml))
	require.NoError(t, err)
	require.NoError(t, gw.Close())

	gzReader, applied, err := compress.MaybeGunzip(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	assert.True(t, applied, "gzip 流应被识别并解压")
	gzc, err := parser.Parse(io.LimitReader(gzReader, maxPMFileBytes), deviceID)
	require.NoError(t, err, "gzip PM 文件解压后应能正常解析（修复前此处失败）")

	// 两条路径解析结果一致。
	assert.Equal(t, plain.DeviceSN, gzc.DeviceSN)
	assert.Equal(t, len(plain.Counters), len(gzc.Counters))
}
