package provider

import (
	"testing"

	"go.uber.org/zap"
)

func TestAppAdhocRepo_HasStreamingRepositoryWired(t *testing.T) {
	repo := buildPMAdhocRepo(nil, nil, nil, zap.NewNop())

	if !repo.HasStreamingRepository() {
		t.Fatal("app 端任务 CRUD 必须生成不可变在线聚合版本")
	}
}
