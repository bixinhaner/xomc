package parammodel

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// T-0178 Upload XML 处理共享常量与校验器(handler 子模块)。
//
// 设计取舍:
//   - 校验链分层,handler 主体只做编排(关注事务流);具体规则集中在本文件
//     便于审计安全边界(防 XXE / 路径遍历 / 大文件 DoS / 文件名混淆)
//   - 所有 helper 均纯函数,无 IO,可独立单测

// MaxUploadXMLSize 单文件上限 1 MiB。生产环境 paramModel XML 数百 KB,
// 给 4x 余量(留给注释 / 编辑器 BOM 等)。Gin engine.MaxMultipartMemory
// 应单独配 4 MiB(P3 deploy),防多文件并发 multipart 占内存。
const MaxUploadXMLSize int64 = 1 << 20

// uploadFilenamePattern 限制文件名为 [A-Za-z0-9_-] + ".xml",长度 1-64。
//
// 拒绝清单(由该正则保证):
//   - 路径分隔符 / \   → 防路径遍历
//   - 点开头(隐藏文件) → 防 dotfile 投毒
//   - 空白 / unicode    → 防文件名混淆
//   - 多扩展名 X.xml.sh → 防 MIME 嗅探旁路
//   - 超长名 → 防 fs 名长度溢出 / 日志截断
var uploadFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}\.xml$`)

// reservedUploadFilenames 是 paramModel Upload 路径上的保留名集合,
// 这些文件由 Loader 显式承载语义(standardModel / product / 历史 routing),
// Upload 禁止落到 custom 目录覆盖它们(否则启动 Reload 时合并算法行为难以预料)。
var reservedUploadFilenames = map[string]struct{}{
	"standard-model.xml":       {},
	"products.xml":             {},
	"product-name-routing.xml": {},
	"param-model-routing.xml":  {},
}

// validateUploadFilename 是 Upload handler 入口处的文件名守门。
// 返回 nil 表示通过;否则 error message 用于审计 + 400 响应体。
//
// 输入应为已 filepath.Base 过的 basename(handler 调用前必须 Base 化,
// 防止用户传 "../X.xml" 等含目录的名字)。
func validateUploadFilename(name string) error {
	if name == "" {
		return fmt.Errorf("filename empty")
	}
	if !uploadFilenamePattern.MatchString(name) {
		return fmt.Errorf("filename %q does not match %s "+
			"(allowed: 1-64 chars of [A-Za-z0-9_-] + \".xml\")",
			name, uploadFilenamePattern.String())
	}
	if _, ok := reservedUploadFilenames[strings.ToLower(name)]; ok {
		return fmt.Errorf("filename %q is reserved (carries Loader-specific semantics, "+
			"cannot be uploaded as custom paramModel)", name)
	}
	return nil
}

// validateUploadXML 验证上传字节流是合法的 paramModel XML:
//   - encoding/xml.Decoder Strict=true:严格 well-formedness,
//     拒绝 mismatched closing tag / 半截 XML / 控制字符等异常
//   - 不处理外部实体(Go 标准库默认安全,无 XXE 风险)
//   - 第一个 StartElement 的 Local Name 必须是 "paramModel"(namespace 不限,
//     与 Loader.xml.Unmarshal 默认行为对齐,避免上传通过但 Reload 失败)
//   - 找到根元素后继续扫到 EOF,捕获后续结构错误
//
// 性能:typical paramModel XML ≤ 200 KB,完整扫描 ~ms 级,不构成瓶颈。
func validateUploadXML(raw []byte) error {
	if len(raw) == 0 {
		return fmt.Errorf("upload body empty")
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = true
	rootSeen := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			if !rootSeen {
				return fmt.Errorf("no root element found in upload")
			}
			return nil
		}
		if err != nil {
			return fmt.Errorf("invalid xml: %w", err)
		}
		if rootSeen {
			// 已确认根元素,继续扫到 EOF 验完整 XML 形态(mismatched closing tag 等)
			continue
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			// ProcessingInstruction / CharData / Comment 之类 — 继续找根元素
			continue
		}
		if start.Name.Local != "paramModel" {
			return fmt.Errorf("root element must be <paramModel>, got <%s>", start.Name.Local)
		}
		rootSeen = true
	}
}

// pathContainedIn 验证 target 物理路径在 base 目录树之下。
//
// 用于二次防御路径遍历:即使文件名通过白名单正则,handler 内拼接路径前
// 仍调用本函数检查 filepath.Join(base, name) 的最终绝对路径没有逃逸 base。
//
// 实现:filepath.Abs 规范化两端(消解 ., .., 符号链相对部分);
// 再 filepath.Rel 取相对路径,首字符为 ".." 则越界。
//
// 失败情形(返 false):
//   - filepath.Abs 解析异常
//   - rel 以 ".." 开头(target 在 base 之外)
//   - target == base(允许)→ rel="." → 返 true
func pathContainedIn(base, target string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}
