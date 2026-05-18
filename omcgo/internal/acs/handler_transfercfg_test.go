package acs

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticTransferProvider struct {
	snapshot transfercfg.Snapshot
}

func (s staticTransferProvider) Snapshot(context.Context) transfercfg.Snapshot {
	return s.snapshot
}

func TestCreatePMUploadTask_UsesRuntimeTransferConfig(t *testing.T) {
	h := &Handler{
		transferConfigProvider: staticTransferProvider{snapshot: transfercfg.Snapshot{
			Upload: transfercfg.UploadSettings{
				BaseURL:  "http://runtime.example.com",
				Path:     "/smallcell/FileUploadService",
				Username: "runtime-user",
				Password: "runtime-pass",
			},
		}},
	}
	req := httptest.NewRequest("POST", "/smallcell/AcsService", nil)

	task := h.createPMUploadTask(req, "SN0001")
	require.NotNil(t, task)
	assert.Equal(t, "Upload", task.method)

	var params map[string]any
	require.NoError(t, json.Unmarshal(task.params, &params))
	urlValue, _ := params["url"].(string)
	assert.Contains(t, urlValue, "http://runtime.example.com/smallcell/FileUploadService?fileType=PM&filename=SN0001_")
	assert.Equal(t, "runtime-user", params["username"])
	assert.Equal(t, "runtime-pass", params["password"])
}
