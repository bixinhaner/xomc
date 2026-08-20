package deviceaccess

import (
	"context"
	"testing"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/stretchr/testify/require"
)

func TestPolicyPublishedExpanderCreatesIdempotentReevaluationOutboxes(t *testing.T) {
	db := &repositoryTestDB{}
	expander := newPolicyPublishedExpanderWithDB(db)
	evt, err := event.NewEvent(event.SubjectDeviceAccessPolicyPublished, PolicyPublishedEvent{
		Carrier: "cmcc", PolicyVersionID: "version-8", TriggerEventID: "publish-8",
	})
	require.NoError(t, err)

	err = expander.Handle(context.Background(), evt)

	require.NoError(t, err)
	require.Contains(t, db.lastSQL, "INSERT INTO device_access_outbox")
	require.Contains(t, db.lastSQL, "FROM device_access_states state")
	require.Contains(t, db.lastSQL, "ON CONFLICT (event_key) DO NOTHING")
	require.Contains(t, db.lastArgs, "cmcc")
	require.Contains(t, db.lastArgs, TriggerPolicyPublished)
	require.Contains(t, db.lastArgs, event.SubjectDeviceAccessReevaluationRequested)
}
