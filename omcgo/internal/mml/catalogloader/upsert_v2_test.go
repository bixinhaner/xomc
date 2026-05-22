package catalogloader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizeChapterLTreePath 校验 ":" → "_"，符合 ltree label 字符集。
func TestNormalizeChapterLTreePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"chapter:SA", "chapter_SA"},
		{"chapter:SF", "chapter_SF"},
		{"chapter:SR", "chapter_SR"},
		// 已是合法 label → 原样返回
		{"chapter_SA", "chapter_SA"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := normalizeChapterLTreePath(c.in)
			assert.Equal(t, c.want, got)
		})
	}
}

