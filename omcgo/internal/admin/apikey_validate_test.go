package admin

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// mkCandidate 用给定明文 key 生成一条 hash 已落库的 APIKey 候选。
func mkCandidate(t *testing.T, plainKey string) *APIKey {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(plainKey), bcrypt.MinCost) // MinCost 仅为测试提速
	require.NoError(t, err)
	return &APIKey{ID: uuid.New(), KeyHash: string(hash)}
}

func Test_selectMatchingAPIKey_NoCandidates(t *testing.T) {
	// 失败路径：空候选集返回 nil。
	assert.Nil(t, selectMatchingAPIKey(nil, "omk_anything"))
	assert.Nil(t, selectMatchingAPIKey([]*APIKey{}, "omk_anything"))
}

func Test_selectMatchingAPIKey_NoMatch(t *testing.T) {
	// 失败路径：同前缀但无一命中 → nil。
	candidates := []*APIKey{
		mkCandidate(t, "omk_aaaa0000000000000000000000000000"),
		mkCandidate(t, "omk_aaaa1111111111111111111111111111"),
	}
	assert.Nil(t, selectMatchingAPIKey(candidates, "omk_aaaa9999999999999999999999999999"))
}

func Test_selectMatchingAPIKey_MatchFirst(t *testing.T) {
	// 成功路径：命中位于首位。
	want := mkCandidate(t, "omk_bbbb0000000000000000000000000000")
	candidates := []*APIKey{
		want,
		mkCandidate(t, "omk_bbbb1111111111111111111111111111"),
		mkCandidate(t, "omk_bbbb2222222222222222222222222222"),
	}
	got := selectMatchingAPIKey(candidates, "omk_bbbb0000000000000000000000000000")
	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID)
}

func Test_selectMatchingAPIKey_MatchLast(t *testing.T) {
	// 成功路径：命中位于末位 —— 验证遍历不在前面非命中项处提前退出。
	want := mkCandidate(t, "omk_cccc2222222222222222222222222222")
	candidates := []*APIKey{
		mkCandidate(t, "omk_cccc0000000000000000000000000000"),
		mkCandidate(t, "omk_cccc1111111111111111111111111111"),
		want,
	}
	got := selectMatchingAPIKey(candidates, "omk_cccc2222222222222222222222222222")
	require.NotNil(t, got)
	assert.Equal(t, want.ID, got.ID, "命中末位说明遍历未在首个非命中处短路")
}
