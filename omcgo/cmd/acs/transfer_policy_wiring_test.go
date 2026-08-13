package main

import (
	"context"
	"testing"

	"github.com/omcgo/omcgo/internal/acs/transfercfg"
	"github.com/omcgo/omcgo/internal/admin"
	coreevent "github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubTransferSysConfigRepository struct {
	values map[string]string
}

type stubTransferEventSubscriber struct {
	subject string
	handler coreevent.EventHandler
}

func (s *stubTransferEventSubscriber) Subscribe(
	subject string,
	handler coreevent.EventHandler,
) (coreevent.Subscription, error) {
	s.subject = subject
	s.handler = handler
	return stubTransferSubscription{}, nil
}

type stubTransferSubscription struct{}

func (stubTransferSubscription) Unsubscribe() error { return nil }

func (r stubTransferSysConfigRepository) GetByKey(
	_ context.Context,
	category string,
	key string,
) (*admin.SysConfig, error) {
	value, ok := r.values[category+"."+key]
	if !ok {
		return nil, nil
	}
	return &admin.SysConfig{Category: category, Key: key, Value: value}, nil
}

func TestACSTransferPolicyWiringLoadsProtocolSnapshotFields(t *testing.T) {
	repository := stubTransferSysConfigRepository{values: map[string]string{
		transfercfg.Category + "." + transfercfg.KeyProtocolPolicy:       transfercfg.ProtocolPolicyPreferHTTPS,
		transfercfg.Category + "." + transfercfg.KeyHTTPSUploadBaseURL:   "https://acs-upload.example.com",
		transfercfg.Category + "." + transfercfg.KeyHTTPSDownloadBaseURL: "https://acs-download.example.com",
	}}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{}, newTransferSysConfigLookup(repository))

	snapshot := policy.Snapshot(context.Background())

	assert.Equal(t, transfercfg.ProtocolPolicyPreferHTTPS, snapshot.ProtocolPolicy)
	assert.Equal(t, "https://acs-upload.example.com", snapshot.Upload.HTTPSBaseURL)
	assert.Equal(t, "https://acs-download.example.com", snapshot.Download.HTTPSBaseURL)
}

func TestSubscribeTransferPolicyInvalidation_RefreshesACSSnapshot(t *testing.T) {
	values := map[string]string{
		transfercfg.KeyProtocolPolicy: transfercfg.ProtocolPolicyForceHTTP,
	}
	policy := transfercfg.NewPolicy(transfercfg.Snapshot{}, func(_ context.Context, category, key string) (string, bool) {
		if category != transfercfg.Category {
			return "", false
		}
		value, ok := values[key]
		return value, ok
	})
	subscriber := &stubTransferEventSubscriber{}

	_, err := subscribeTransferPolicyInvalidation(subscriber, policy, zap.NewNop())
	require.NoError(t, err)
	assert.Equal(t, coreevent.SubjectSysConfigSaved, subscriber.subject)
	require.NotNil(t, subscriber.handler)
	require.Equal(t, transfercfg.ProtocolPolicyForceHTTP, policy.Snapshot(context.Background()).ProtocolPolicy)

	values[transfercfg.KeyProtocolPolicy] = transfercfg.ProtocolPolicyPreferHTTPS
	evt, err := coreevent.NewEvent(
		coreevent.SubjectSysConfigSaved,
		coreevent.SysConfigSavedPayload{Category: transfercfg.Category},
	)
	require.NoError(t, err)
	require.NoError(t, subscriber.handler(context.Background(), evt))

	assert.Equal(t, transfercfg.ProtocolPolicyPreferHTTPS, policy.Snapshot(context.Background()).ProtocolPolicy)
}
