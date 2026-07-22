package software

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildTransferUploadURL_PreservesPrefixAndEncodesResolvedQuery(t *testing.T) {
	got, err := buildTransferUploadURL(
		"https://edge.example.com:9443/omc/",
		"/smallcell/FileUploadService?fileType=LOG&filename=%E9%85%8D%E7%BD%AE+a%26b.log&sn=SN+100",
	)
	require.NoError(t, err)

	parsed, err := url.Parse(got)
	require.NoError(t, err)
	require.Equal(t, "/omc/smallcell/FileUploadService", parsed.Path)
	require.Equal(t, "LOG", parsed.Query().Get("fileType"))
	require.Equal(t, "配置 a&b.log", parsed.Query().Get("filename"))
	require.Equal(t, "SN 100", parsed.Query().Get("sn"))
}

func TestBuildTransferUploadURL_ProductionAllowsDeviceReachablePrivateBase(t *testing.T) {
	t.Setenv("OMCGO_ENV", "production")

	got, err := buildTransferUploadURL(
		"http://172.17.9.239:8081",
		"/smallcell/FileUploadService?fileType=LOG",
	)

	require.NoError(t, err)
	require.Equal(t, "http://172.17.9.239:8081/smallcell/FileUploadService?fileType=LOG", got)
}

func TestBuildTransferUploadURL_RejectsAbsoluteOrEscapingTransportPath(t *testing.T) {
	for _, raw := range []string{
		"https://attacker.example/upload",
		"/smallcell/../admin?token=x",
		"/smallcell/FileUploadService#fragment",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := buildTransferUploadURL("https://edge.example.com/omc", raw)
			require.Error(t, err)
		})
	}
}
