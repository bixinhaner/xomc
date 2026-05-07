package provision

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/omcgo/omcgo/internal/config/datamodel"
)

func TestAccessFromWritable(t *testing.T) {
	assert.Equal(t, "readWrite", accessFromWritable(true))
	assert.Equal(t, "readOnly", accessFromWritable(false))
}

func TestCpeMinValue_Cases(t *testing.T) {
	assert.Nil(t, cpeMinValue(nil))
	assert.Nil(t, cpeMinValue(&datamodel.Constraints{MinValue: nil}))

	v := int64(7)
	got := cpeMinValue(&datamodel.Constraints{MinValue: &v})
	require := assert.New(t)
	require.NotNil(got)
	require.EqualValues(7, *got)

	// 验证返回的是新指针（不共享底层），便于上层任意修改
	*got = 100
	assert.EqualValues(t, 7, v, "返回值不应别名输入")
}

func TestCpeMaxValue_Cases(t *testing.T) {
	assert.Nil(t, cpeMaxValue(nil))
	assert.Nil(t, cpeMaxValue(&datamodel.Constraints{MaxValue: nil}))

	v := int64(255)
	got := cpeMaxValue(&datamodel.Constraints{MaxValue: &v})
	require := assert.New(t)
	require.NotNil(got)
	require.EqualValues(255, *got)
}
