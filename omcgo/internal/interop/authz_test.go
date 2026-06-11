package interop

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
)

// fakeGroupReader 用固定 设备→分组 映射模拟 authz.GroupReader。
type fakeGroupReader struct{ groups []uuid.UUID }

func (f fakeGroupReader) GetDeviceGroupIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return f.groups, nil
}

// #63：RunAll/RunByCategory 在解析设备后、执行（可能破坏性）测试前按设备组校验。
func TestRunner_VisibilityAuthz(t *testing.T) {
	gVisible, gOther := uuid.New(), uuid.New()
	dev := newTestDevice()

	newRunner := func(readerGroups []uuid.UUID) *ConformanceTestRunner {
		r := newTestRunner(newMockDeviceRepo(dev), newMockParamRepo(), newMockCmdQueue())
		r.SetGroupReader(fakeGroupReader{groups: readerGroups})
		return r
	}

	t.Run("超管 nil 放行执行", func(t *testing.T) {
		r := newRunner([]uuid.UUID{gOther})
		results, err := r.RunAll(context.Background(), dev.SerialNumber, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, results)
	})

	t.Run("可见组有交集放行", func(t *testing.T) {
		r := newRunner([]uuid.UUID{gVisible})
		results, err := r.RunByCategory(context.Background(), dev.SerialNumber, CategoryProtocol, []uuid.UUID{gVisible})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("RunAll 无交集越权拒绝不执行", func(t *testing.T) {
		r := newRunner([]uuid.UUID{gOther})
		_, err := r.RunAll(context.Background(), dev.SerialNumber, []uuid.UUID{gVisible})
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})

	t.Run("RunByCategory 无交集越权拒绝不执行", func(t *testing.T) {
		r := newRunner([]uuid.UUID{gOther})
		_, err := r.RunByCategory(context.Background(), dev.SerialNumber, CategoryProtocol, []uuid.UUID{gVisible})
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})

	t.Run("空可见组 fail-closed", func(t *testing.T) {
		r := newRunner([]uuid.UUID{gVisible})
		_, err := r.RunAll(context.Background(), dev.SerialNumber, []uuid.UUID{})
		assert.ErrorIs(t, err, commonerrors.ErrForbidden)
	})
}
