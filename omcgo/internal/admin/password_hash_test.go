package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func Test_clampBcryptCost(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"低于下限被抬到 floor", bcryptCostFloor - 5, bcryptCostFloor},
		{"远低于下限（含负数）也抬到 floor", -100, bcryptCostFloor},
		{"等于下限保持", bcryptCostFloor, bcryptCostFloor},
		{"区间内保持", bcryptCostFloor + 1, bcryptCostFloor + 1},
		{"默认值保持", bcryptCostDefault, bcryptCostDefault},
		{"高于上限被夹到 MaxCost", bcrypt.MaxCost + 10, bcrypt.MaxCost},
		{"等于上限保持", bcrypt.MaxCost, bcrypt.MaxCost},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, clampBcryptCost(tt.in))
		})
	}
}

func Test_resolveBcryptCost_DefaultAtLeast12(t *testing.T) {
	// 不设环境变量时，进程级 cost 必须 ≥ floor（12），且应为业务默认 13。
	got := resolveBcryptCost()
	assert.GreaterOrEqual(t, got, bcryptCostFloor, "解析出的 cost 不得低于安全下限")
	assert.Equal(t, bcryptCostDefault, got, "未配置时应采用默认 cost 13")
}

func Test_hashSecret_RoundTrip(t *testing.T) {
	// 成功路径：生成的哈希能被 bcrypt 校验通过，且 cost 落在已加固区间。
	plain := []byte("S3cure-Passw0rd!")
	hash, err := hashSecret(plain)
	require.NoError(t, err)

	cost, err := bcrypt.Cost(hash)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, cost, bcryptCostFloor, "实际写入哈希的 cost 必须 ≥12")

	assert.NoError(t, bcrypt.CompareHashAndPassword(hash, plain), "正确口令应校验通过")
}

func Test_hashSecret_WrongPasswordFails(t *testing.T) {
	// 失败路径：错误口令必须校验不通过。
	hash, err := hashSecret([]byte("correct-horse"))
	require.NoError(t, err)
	assert.Error(t, bcrypt.CompareHashAndPassword(hash, []byte("battery-staple")))
}
