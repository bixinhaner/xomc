package indicator

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

// T-0180 P1.4 Upload XML 处理共享常量与校验器(file_handler 子模块)。
//
// 设计取舍(沿用 T-0178 parammodel/upload.go 范式):
//   - 校验链分层,handler 主体只做编排;具体规则集中在本文件便于审计安全边界
//     (防 XXE / 路径遍历 / 大文件 DoS / 文件名混淆 / tech mismatch)
//   - 所有 helper 均纯函数,无 IO,可独立单测

// MaxUploadXMLSize 单文件上限 1 MiB。生产环境 indicator XML 最大约 200 KB
// (GSM.xml 73 条 + GNB.xml 282 条 + ENB 多文件单个 < 100KB),给 5x 余量。
// Gin engine.MaxMultipartMemory 应单独配 4 MiB(P3 deploy)。
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

// 注意:T-0180 决策 D1 = "不设保留名,custom 可覆盖内置同名",
// 因此 indicator Upload 不设 reservedUploadFilenames(与 T-0178 不同)。
// 同名 builtin 由 Loader.resolveENBSources 在合并阶段按 CustomOverrides 决定胜负,
// 不在 Upload 层硬拦。

// validateUploadFilename 是 Upload handler 入口处的文件名守门。
// 输入应为已 filepath.Base 过的 basename。
// 返 nil 表示通过;否则 error message 用于审计 + 400 响应体。
func validateUploadFilename(name string) error {
	if name == "" {
		return fmt.Errorf("filename empty")
	}
	if !uploadFilenamePattern.MatchString(name) {
		return fmt.Errorf("filename %q does not match %s "+
			"(allowed: 1-64 chars of [A-Za-z0-9_-] + \".xml\")",
			name, uploadFilenamePattern.String())
	}
	return nil
}

// validateUploadTech 校验 ?tech= query 参数在合法集合内。
// 合法值:enb / gsm / gnb(对齐 source.go 与 file_repository.go::validateTech)。
func validateUploadTech(tech string) error {
	if _, ok := allowedFileTechs[tech]; !ok {
		return fmt.Errorf("invalid tech %q: must be one of enb/gsm/gnb", tech)
	}
	return nil
}

// validateUploadXML 验证上传字节流是合法的 indicator XML:
//   - encoding/xml.Decoder Strict=true:well-formedness 严格,
//     拒绝 mismatched closing tag / 半截 XML / 控制字符
//   - 不处理外部实体(Go 标准库默认安全,无 XXE 风险)
//   - 根元素必须 <indicatorModel>(namespace 不限,与 Loader 对齐)
//   - 决策 D2:platform 属性必填,非空字符串
//   - 决策 D2:deviceType 属性若 present 必须(大小写不敏感)匹配 ?tech=
//     若 absent → 容忍(ENB builtin XML 历史无 deviceType,custom 同样允许省略)
//
// 性能:typical indicator XML ≤ 200 KB,完整扫描 ~ms 级。
func validateUploadXML(raw []byte, tech string) error {
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
			// 已校验根元素,继续扫到 EOF 验完整 XML 形态
			continue
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "indicatorModel" {
			return fmt.Errorf("root element must be <indicatorModel>, got <%s>", start.Name.Local)
		}
		// D2: platform 必填
		platform := attrValue(start.Attr, "platform")
		if strings.TrimSpace(platform) == "" {
			return fmt.Errorf("<indicatorModel> requires non-empty platform attribute")
		}
		// D2: deviceType 若 present 必须匹配 tech(大小写不敏感)
		if dt := attrValue(start.Attr, "deviceType"); dt != "" {
			if !strings.EqualFold(dt, tech) {
				return fmt.Errorf("deviceType=%q does not match upload tech=%q "+
					"(case-insensitive); upload to /upload-xml?tech=%s instead",
					dt, tech, strings.ToLower(dt))
			}
		}
		rootSeen = true
	}
}

// attrValue 从 XML StartElement.Attr 列表查 local 名匹配的属性值;未找到返空。
// namespace 不敏感(handler 接受所有 namespace 前缀,与 Loader 对齐)。
func attrValue(attrs []xml.Attr, local string) string {
	for _, a := range attrs {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// pathContainedIn 验证 target 物理路径在 base 目录树之下。
//
// 用于二次防御路径遍历:即使文件名通过白名单正则,handler 内拼接路径前
// 仍调用本函数检查 filepath.Join(base, name) 的最终绝对路径没有逃逸 base。
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
