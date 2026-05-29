package indicator

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateUploadFilename 覆盖 T-0180 PRD §3 GWT-3 文件名守门。
func TestValidateUploadFilename(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// 合法
		{"normal lowercase", "myindicator.xml", false},
		{"with digits", "ENB123.xml", false},
		{"with underscore", "MY_INDICATOR.xml", false},
		{"with dash", "MY-INDICATOR.xml", false},
		{"max length 64", strings.Repeat("a", 60) + ".xml", false},

		// 拒绝
		{"empty", "", true},
		{"path separator", "sub/MY.xml", true},
		{"backslash", "sub\\MY.xml", true},
		{"dot start", ".hidden.xml", true},
		{"no extension", "myindicator", true},
		{"wrong extension", "MY.json", true},
		{"double extension", "MY.xml.sh", true},
		{"unicode", "中文.xml", true},
		{"space", "my indicator.xml", true},
		{"too long", strings.Repeat("a", 65) + ".xml", true},
		{"traversal", "../etc/passwd.xml", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUploadFilename(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateUploadTech 覆盖 ?tech= query 参数白名单(D2 关联)。
func TestValidateUploadTech(t *testing.T) {
	assert.NoError(t, validateUploadTech("enb"))
	assert.NoError(t, validateUploadTech("gsm"))
	assert.NoError(t, validateUploadTech("gnb"))
	assert.Error(t, validateUploadTech("lte"))
	assert.Error(t, validateUploadTech("ENB")) // 大小写敏感
	assert.Error(t, validateUploadTech(""))
}

// TestValidateUploadXML 覆盖 T-0180 PRD §3 GWT-3 XML 根 + D2 deviceType 校验。
func TestValidateUploadXML(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		tech    string
		wantErr bool
		errHint string // 子串匹配,nil/空 表不检查
	}{
		// 合法
		{
			name: "valid GSM with deviceType match",
			body: `<indicatorModel platform="BSC" deviceType="GSM" indicatorCount="0"></indicatorModel>`,
			tech: "gsm",
		},
		{
			name: "valid GNB with deviceType match",
			body: `<indicatorModel platform="BaiBNQ" deviceType="GNB" indicatorCount="0"></indicatorModel>`,
			tech: "gnb",
		},
		{
			name: "valid ENB without deviceType (legacy)",
			body: `<indicatorModel platform="ENB_DEFAULT_098" indicatorCount="0"></indicatorModel>`,
			tech: "enb",
		},
		{
			name: "case-insensitive deviceType match",
			body: `<indicatorModel platform="X" deviceType="enb" indicatorCount="0"></indicatorModel>`,
			tech: "enb",
		},
		{
			name: "with xml header",
			body: `<?xml version="1.0" encoding="UTF-8"?><indicatorModel platform="X" indicatorCount="0"></indicatorModel>`,
			tech: "enb",
		},

		// 拒绝
		{
			name:    "empty body",
			body:    "",
			tech:    "enb",
			wantErr: true,
			errHint: "empty",
		},
		{
			name:    "wrong root element",
			body:    `<paramModel platform="X"></paramModel>`,
			tech:    "enb",
			wantErr: true,
			errHint: "root element must be <indicatorModel>",
		},
		{
			name:    "no platform attr",
			body:    `<indicatorModel indicatorCount="0"></indicatorModel>`,
			tech:    "enb",
			wantErr: true,
			errHint: "non-empty platform",
		},
		{
			name:    "empty platform attr",
			body:    `<indicatorModel platform="" indicatorCount="0"></indicatorModel>`,
			tech:    "enb",
			wantErr: true,
			errHint: "non-empty platform",
		},
		{
			name:    "deviceType mismatch tech",
			body:    `<indicatorModel platform="X" deviceType="GSM" indicatorCount="0"></indicatorModel>`,
			tech:    "enb",
			wantErr: true,
			errHint: "does not match upload tech",
		},
		{
			name:    "malformed XML",
			body:    `<indicatorModel platform="X" indicatorCount="0">`,
			tech:    "enb",
			wantErr: true,
			errHint: "invalid xml",
		},
		{
			name:    "no root found",
			body:    `<?xml version="1.0"?><!-- comment only -->`,
			tech:    "enb",
			wantErr: true,
			errHint: "no root element",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateUploadXML([]byte(tc.body), tc.tech)
			if tc.wantErr {
				assert.Error(t, err)
				if tc.errHint != "" && err != nil {
					assert.Contains(t, err.Error(), tc.errHint)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestPathContainedIn 路径遍历二次防御。
func TestPathContainedIn(t *testing.T) {
	base, _ := filepath.Abs(t.TempDir())
	cases := []struct {
		name   string
		base   string
		target string
		want   bool
	}{
		{"same dir", base, base, true},
		{"child", base, filepath.Join(base, "x.xml"), true},
		{"deep child", base, filepath.Join(base, "a/b/c.xml"), true},
		{"parent escape", base, filepath.Join(base, ".."), false},
		{"sibling escape", base, filepath.Join(base, "../sibling/x.xml"), false},
		{"absolute outside", base, "/etc/passwd", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pathContainedIn(tc.base, tc.target)
			assert.Equal(t, tc.want, got)
		})
	}
}
