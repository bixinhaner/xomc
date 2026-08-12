package transfercfg

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildURL_PreservesBasePrefixAndEncodesQuery(t *testing.T) {
	got, err := BuildURL(
		"https://edge.example.com:9443/omc/",
		"/smallcell/FileUploadService",
		nil,
		url.Values{"filename": {"配置 a&b.xml"}},
	)

	require.NoError(t, err)
	require.Equal(t,
		"https://edge.example.com:9443/omc/smallcell/FileUploadService?filename=%E9%85%8D%E7%BD%AE+a%26b.xml",
		got,
	)
}

func TestBuildURL_AppendsEscapedObjectSegmentsWithinServicePath(t *testing.T) {
	got, err := BuildURL(
		"https://edge.example.com/omc",
		"/smallcell/FileDownloadService/",
		[]string{"firmware", "版本 1.0+正式.bin"},
		nil,
	)

	require.NoError(t, err)
	require.Equal(t,
		"https://edge.example.com/omc/smallcell/FileDownloadService/firmware/%E7%89%88%E6%9C%AC%201.0+%E6%AD%A3%E5%BC%8F.bin",
		got,
	)
}

func TestBuildTemplateURL_PreservesBusinessQueryOrderAndEmptyFilenameTail(t *testing.T) {
	got, err := BuildTemplateURL(
		"https://edge.example.com:9443/omc/",
		"/smallcell/FileUploadService?fileType=LOG&sn=SN100&taskId=abc123&filename=",
	)

	require.NoError(t, err)
	require.Equal(t,
		"https://edge.example.com:9443/omc/smallcell/FileUploadService?fileType=LOG&sn=SN100&taskId=abc123&filename=",
		got,
	)
}

func TestBuildURL_PreservesIPLiteralPortProxyPrefixAndEncoding(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		want    string
	}{
		{
			name:    "IPv4",
			baseURL: "http://192.0.2.10:8080/reverse-proxy",
			want:    "http://192.0.2.10:8080/reverse-proxy/smallcell/FileDownloadService/device%20SN/%E5%9B%BA%E4%BB%B6.bin",
		},
		{
			name:    "bracketed IPv6",
			baseURL: "https://[2001:db8::10]:9443/reverse-proxy",
			want:    "https://[2001:db8::10]:9443/reverse-proxy/smallcell/FileDownloadService/device%20SN/%E5%9B%BA%E4%BB%B6.bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildURL(
				tt.baseURL,
				"/smallcell/FileDownloadService",
				[]string{"device SN", "固件.bin"},
				nil,
			)

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestBuildTemplateURL_PreservesBracketedIPv6AndEncodedTemplate(t *testing.T) {
	got, err := BuildTemplateURL(
		"https://[2001:db8::20]:10443/omc-proxy",
		"/smallcell/FileUploadService/%E8%AE%BE%E5%A4%87?fileType=LOG&filename=",
	)

	require.NoError(t, err)
	require.Equal(t,
		"https://[2001:db8::20]:10443/omc-proxy/smallcell/FileUploadService/%E8%AE%BE%E5%A4%87?fileType=LOG&filename=",
		got,
	)
}

func TestBuildURL_RejectsInvalidConfiguredComponents(t *testing.T) {
	for name, tc := range map[string]struct {
		baseURL     string
		servicePath string
	}{
		"base query":     {"https://edge.example.com/omc?token=x", "/smallcell/FileUploadService"},
		"base fragment":  {"https://edge.example.com/omc#section", "/smallcell/FileUploadService"},
		"empty base":     {"", "/smallcell/FileUploadService"},
		"path traversal": {"https://edge.example.com/omc", "/smallcell/../admin"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := BuildURL(tc.baseURL, tc.servicePath, nil, nil)
			require.Error(t, err)
		})
	}
}

func TestBuildURL_RejectsEscapingObjectSegment(t *testing.T) {
	_, err := BuildURL(
		"https://edge.example.com/omc",
		"/smallcell/FileDownloadService",
		[]string{"firmware", "../secret"},
		nil,
	)

	require.Error(t, err)
}
