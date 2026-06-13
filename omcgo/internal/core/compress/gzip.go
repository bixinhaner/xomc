// Package compress 提供入库链路通用的流式压缩/解压工具。
//
// 背景（issue #321）：真机 CPE 与 cpe_simulator.py 按 TR-069 标准把 PM/MR 文件
// gzip 压缩成 .xml.gz 上传，ACS 上传 handler 对 PM/MR 原样存入 MinIO（不解压）。
// 但 collector 解析侧此前直接把 MinIO 对象流喂给 xml.Decoder，遇到 gzip 魔数
// (0x1f 0x8b) 必然解析失败 —— 真机 .gz 文件无法入库。本包提供按魔数嗅探的
// 透明解压，明文 .xml 原样透传，使两种格式都能正确解析。
package compress

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
)

// gzipMagic 是 gzip 流的前两个魔数字节（RFC 1952 §2.3.1）。
var gzipMagic = [2]byte{0x1f, 0x8b}

// MaybeGunzip 嗅探 r 头部的 gzip 魔数（0x1f 0x8b）。命中则用 gzip.Reader 包装，
// 返回解压后的明文流且 applied=true；否则原样透传（含已缓冲的嗅探字节）且
// applied=false。流不足 2 字节（空/极短文件）一律按非 gzip 透传，交由下游解析器
// 自行报错，本函数不视其为错误。
//
// 本函数不持有 r 的关闭权——调用方负责关闭底层 reader（如 MinIO 对象）。
// 解压发生在调用方施加体积上限（io.LimitReader）之前，故上限作用于解压后内容，
// 同时为 gzip 炸弹提供兜底（解压超限会被截断 → 解析报错被捕获）。
func MaybeGunzip(r io.Reader) (out io.Reader, applied bool, err error) {
	br := bufio.NewReader(r)
	magic, _ := br.Peek(2)
	if len(magic) < 2 || magic[0] != gzipMagic[0] || magic[1] != gzipMagic[1] {
		// 非 gzip（或流过短）：原样透传已缓冲的 reader。
		return br, false, nil
	}
	zr, zerr := gzip.NewReader(br)
	if zerr != nil {
		return nil, false, fmt.Errorf("new gzip reader: %w", zerr)
	}
	return zr, true, nil
}

// IsGzip 报告 magic 头是否为 gzip 魔数。供需要先于解压做格式判定的调用方
// （如 issue #321 压缩回写时判断对象是否已压缩、避免二次压缩）复用。
func IsGzip(magic []byte) bool {
	return len(magic) >= 2 && magic[0] == gzipMagic[0] && magic[1] == gzipMagic[1]
}
