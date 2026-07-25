package device

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDeviceInfoSyncFieldsUpdateSkipsUnchangedProjection(t *testing.T) {
	query, args, err := buildDeviceInfoSyncFieldsUpdate(uuid.New(), map[string]interface{}{
		"cell_status": "active",
		"ue_count":    3,
	})

	require.NoError(t, err)
	assert.Contains(t, query, "UPDATE device_info SET")
	assert.Contains(t, query, "cell_status IS DISTINCT FROM")
	assert.Contains(t, query, "ue_count IS DISTINCT FROM")
	assert.Contains(t, query, "WHERE device_id =")
	assert.Len(t, args, 5)
}
