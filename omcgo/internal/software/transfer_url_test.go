package software

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildTransferUploadURL_PreservesRuntimeLogFilenameTail(t *testing.T) {
	got, err := buildTransferUploadURL(
		"https://edge.example.com:9443/omc/",
		"/smallcell/FileUploadService?fileType=LOG&sn=SN100&taskId=abc123&filename=",
	)
	require.NoError(t, err)
	require.Equal(t,
		"https://edge.example.com:9443/omc/smallcell/FileUploadService?fileType=LOG&sn=SN100&taskId=abc123&filename=",
		got,
	)
}

func TestBuildTransferUploadURL_PreservesFaultLogFileNameTail(t *testing.T) {
	got, err := buildTransferUploadURL(
		"https://edge.example.com:9443/omc/",
		"/smallcell/FileUploadService?fileType=RL&id=task-1&sn=SN100&fileName=",
	)
	require.NoError(t, err)
	require.Equal(t,
		"https://edge.example.com:9443/omc/smallcell/FileUploadService?fileType=RL&id=task-1&sn=SN100&fileName=",
		got,
	)
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
