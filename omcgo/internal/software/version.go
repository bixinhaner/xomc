package software

import (
	"strconv"
	"strings"
)

// compareFirmwareVersion 比较两个固件版本号，返回 (cmp, ok)：
//
//	cmp < 0  → a 比 b 旧
//	cmp == 0 → 相等
//	cmp > 0  → a 比 b 新
//	ok == false → 无法可靠比较（任一为空 / 含非数字段 / 解析失败），调用方应按「不确定」处理
//
// 设计取向（#59 Problem 4 回退防降级）：本仓固件版本形如 "V1.0.0" / "V2.5.7" / "V1"，
// 无统一 semver 依赖。这里做最朴素可靠的解析——去掉前导非数字前缀（V/v），按 '.' 切段
// 逐段比较数值，缺失段补 0。任一段含非数字即判定 ok=false，绝不猜测。
//
// 防降级守卫只在 ok==true 且 cmp<0 时拦截——「无法判断」一律放行（fail-open on guard），
// 因为误拦一次合法回退比漏拦一次降级更直接破坏现网 happy-path；漏拦的降级仍由 force
// 显式确认 / 运维流程兜底。授权（authz）与本守卫正交，永不 fail-open。
func compareFirmwareVersion(a, b string) (int, bool) {
	as, aok := splitVersionSegments(a)
	bs, bok := splitVersionSegments(b)
	if !aok || !bok {
		return 0, false
	}
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av != bv {
			if av < bv {
				return -1, true
			}
			return 1, true
		}
	}
	return 0, true
}

// splitVersionSegments 去掉前导非数字前缀，按 '.' 切段并解析为整数切片。
// 空串 / 任一段非纯数字 → ok=false。
func splitVersionSegments(v string) ([]int, bool) {
	v = strings.TrimSpace(v)
	// 去掉前导非数字字符（典型："V" / "v"）。
	v = strings.TrimLeftFunc(v, func(r rune) bool {
		return r < '0' || r > '9'
	})
	if v == "" {
		return nil, false
	}
	parts := strings.Split(v, ".")
	segs := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, false
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, false
		}
		segs = append(segs, n)
	}
	return segs, true
}

// isDowngrade 报告把设备从 current 回退到 target 是否构成降级（target 严格更旧）。
// 仅当两个版本都能可靠比较且 target<current 时返回 true；无法判断时返回 false（放行）。
func isDowngrade(current, target string) bool {
	cmp, ok := compareFirmwareVersion(target, current)
	return ok && cmp < 0
}
