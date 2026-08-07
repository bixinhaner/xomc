package notification

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/model"
)

func TestPgRuleRepository_Integration(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_RULE_TEST_DSN")
	if dsn == "" {
		t.Skip("set NOTIFICATION_RULE_TEST_DSN to a dedicated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	requireDedicatedNotificationDatabase(t, databaseName)
	templateRepository := NewPgTemplateManagementRepository(pool)
	template, err := templateRepository.Create(ctx, ManagedTemplateInput{
		Name: "rule-template-" + uuid.NewString(), Channel: "email", Language: "zh-CN",
		Subject: "告警 {{.alarm_identifier}}", TextBody: "设备 {{.device_sn}}",
		Variables: []string{"alarm_identifier", "device_sn"},
	}, "integration")
	require.NoError(t, err)
	template, err = templateRepository.Publish(ctx, template.ID, template.Revision, "integration")
	require.NoError(t, err)
	channelConfigID := uuid.New()
	_, err = pool.Exec(ctx,
		`INSERT INTO notification_channel_configs (id,channel,name,enabled,created_by) VALUES ($1,'email',$2,false,'integration')`,
		channelConfigID, "rule-email-"+uuid.NewString())
	require.NoError(t, err)
	recipientID := uuid.New()

	repository := NewPgRuleRepository(pool)
	created, err := repository.Create(ctx, RuleDraftInput{
		Name: "critical-nr-" + uuid.NewString(), Priority: 10,
		MatchConditions: RuleMatchConditions{
			Severities:   []model.AlarmSeverity{model.AlarmCritical},
			Technologies: []model.Technology{model.TechNR},
		},
		Policy: json.RawMessage(`{"repeat_limit":3}`), ChangeReason: "initial",
		Recipients: []RuleRecipientInput{{
			TargetType: RecipientTargetUser, TargetID: recipientID.String(), ChannelLimit: []string{"email"},
		}},
		Channels: []RuleChannelInput{{
			Channel: "email", ChannelConfigID: channelConfigID,
			RaisedTemplateVersionID: template.Published.ID, Policy: json.RawMessage(`{}`),
		}},
	}, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(1), created.Revision)
	require.NotNil(t, created.Draft)
	require.Equal(t, int64(1), created.Draft.VersionNo)
	version1ID := created.Draft.ID

	published, err := repository.Publish(ctx, created.ID, 1, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(2), published.Revision)
	require.NotNil(t, published.Published)
	require.Equal(t, version1ID, published.Published.ID)
	_, err = repository.Publish(ctx, created.ID, 2, "integration")
	require.ErrorIs(t, err, ErrDraftUnavailable)
	channelRepository := NewPgChannelConfigRepository(pool)
	_, err = channelRepository.Update(ctx, channelConfigID, 1, ChannelConfigUpdate{
		Name: "rule-email-enabled-" + uuid.NewString(), Enabled: true, Parameters: json.RawMessage(`{}`),
	}, "integration")
	require.NoError(t, err)

	atomicInput := RuleDraftInput{
		Name: "atomic-alarm-mail-" + uuid.NewString(), Priority: 100,
		MatchConditions: RuleMatchConditions{Severities: []model.AlarmSeverity{model.AlarmCritical}},
		Policy:          json.RawMessage(`{}`), ChangeReason: "atomic create",
		Recipients: []RuleRecipientInput{{
			TargetType: RecipientTargetUser, TargetID: uuid.NewString(), ChannelLimit: []string{"email"},
		}},
		Channels: []RuleChannelInput{{
			Channel: "email", ChannelConfigID: channelConfigID,
			RaisedTemplateVersionID: template.Published.ID, Policy: json.RawMessage(`{}`),
		}},
	}
	atomicCreated, err := repository.SavePublished(ctx, uuid.Nil, 0, atomicInput, true, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(1), atomicCreated.Revision)
	require.NotNil(t, atomicCreated.CurrentDraftVersionID)
	require.Equal(t, atomicCreated.CurrentDraftVersionID, atomicCreated.CurrentPublishedVersionID)
	require.Equal(t, atomicCreated.CurrentPublishedVersionID, atomicCreated.CurrentEnabledVersionID)

	atomicInput.ChangeReason = "atomic disable update"
	atomicUpdated, err := repository.SavePublished(ctx, atomicCreated.ID, 1, atomicInput, false, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(2), atomicUpdated.Revision)
	require.Equal(t, atomicUpdated.CurrentDraftVersionID, atomicUpdated.CurrentPublishedVersionID)
	require.Nil(t, atomicUpdated.CurrentEnabledVersionID)

	_, err = pool.Exec(ctx, `UPDATE notification_channel_configs SET enabled=false WHERE id=$1`, channelConfigID)
	require.NoError(t, err)
	atomicInput.ChangeReason = "must roll back"
	_, err = repository.SavePublished(ctx, atomicCreated.ID, 2, atomicInput, true, "integration")
	require.ErrorIs(t, err, ErrChannelDisabled)
	atomicAfterFailure, err := repository.Get(ctx, atomicCreated.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), atomicAfterFailure.Revision)
	var atomicVersionCount int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notification_rule_versions WHERE rule_id=$1`, atomicCreated.ID,
	).Scan(&atomicVersionCount))
	require.Equal(t, 2, atomicVersionCount, "failed enable must roll back its immutable version")
	_, err = pool.Exec(ctx, `UPDATE notification_channel_configs SET enabled=true WHERE id=$1`, channelConfigID)
	require.NoError(t, err)

	enabled, err := repository.Enable(ctx, created.ID, 2, version1ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), enabled.Revision)
	require.Equal(t, version1ID, enabled.Enabled.ID)

	updated, err := repository.UpdateDraft(ctx, created.ID, 3, RuleDraftInput{
		Name: created.Name, Priority: 20,
		MatchConditions: RuleMatchConditions{Severities: []model.AlarmSeverity{model.AlarmMajor}},
		Policy:          json.RawMessage(`{"repeat_limit":1}`), ChangeReason: "major pilot",
		Recipients: []RuleRecipientInput{{
			TargetType: RecipientTargetRole, TargetID: uuid.NewString(), ChannelLimit: []string{"email"},
		}},
		Channels: []RuleChannelInput{{
			Channel: "email", ChannelConfigID: channelConfigID,
			RaisedTemplateVersionID: template.Published.ID, Policy: json.RawMessage(`{}`),
		}},
	}, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(4), updated.Revision)
	require.Equal(t, int64(2), updated.Draft.VersionNo)
	require.Equal(t, version1ID, updated.Published.ID)
	require.Equal(t, version1ID, updated.Enabled.ID,
		"creating a new draft must not silently change the enabled version")
	version2ID := updated.Draft.ID
	_, err = repository.Enable(ctx, created.ID, 4, version2ID)
	require.ErrorIs(t, err, ErrVersionUnpublished)

	var immutableConditions []byte
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT match_conditions FROM notification_rule_versions WHERE id=$1`, version1ID,
	).Scan(&immutableConditions))
	require.JSONEq(t, `{"severities":[1],"technologies":["nr"]}`, string(immutableConditions))
	var immutableTargetType, immutableTargetID string
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT target_type,target_id FROM notification_rule_recipients WHERE rule_version_id=$1`, version1ID,
	).Scan(&immutableTargetType, &immutableTargetID))
	require.Equal(t, RecipientTargetUser, immutableTargetType)
	require.Equal(t, recipientID.String(), immutableTargetID)

	published, err = repository.Publish(ctx, created.ID, 4, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(5), published.Revision)
	require.Equal(t, version2ID, published.Published.ID)
	require.Equal(t, version1ID, published.Enabled.ID,
		"publishing a template/rule version must not silently enable it")
	_, err = pool.Exec(ctx, `UPDATE notification_channel_configs SET enabled=false WHERE id=$1`, channelConfigID)
	require.NoError(t, err)
	_, err = repository.Enable(ctx, created.ID, 5, version2ID)
	require.ErrorIs(t, err, ErrChannelDisabled)
	_, err = pool.Exec(ctx, `UPDATE notification_channel_configs SET enabled=true WHERE id=$1`, channelConfigID)
	require.NoError(t, err)
	enabled, err = repository.Enable(ctx, created.ID, 5, version2ID)
	require.NoError(t, err)
	require.Equal(t, int64(6), enabled.Revision)
	require.Equal(t, version2ID, enabled.Enabled.ID)
	disabled, err := repository.Disable(ctx, created.ID, 6)
	require.NoError(t, err)
	require.Equal(t, int64(7), disabled.Revision)
	require.Nil(t, disabled.CurrentEnabledVersionID)
	require.False(t, disabled.Archived, "disabling must keep the rule editable and re-enableable")
	enabled, err = repository.Enable(ctx, created.ID, 7, version2ID)
	require.NoError(t, err)
	require.Equal(t, int64(8), enabled.Revision)
	require.Equal(t, version2ID, enabled.Enabled.ID)

	_, err = repository.UpdateDraft(ctx, created.ID, 3, RuleDraftInput{Name: created.Name, Priority: 10, Policy: json.RawMessage(`{}`)}, "integration")
	require.ErrorIs(t, err, ErrRevisionMismatch)
	archived, err := repository.Archive(ctx, created.ID, 8)
	require.NoError(t, err)
	require.True(t, archived.Archived)
	require.Nil(t, archived.CurrentEnabledVersionID)
}

func TestPgContactGroupRepository_Integration(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_RULE_TEST_DSN")
	if dsn == "" {
		t.Skip("set NOTIFICATION_RULE_TEST_DSN to a dedicated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	requireDedicatedNotificationDatabase(t, databaseName)

	repository := NewPgContactGroupRepository(pool)
	userID := uuid.New()
	first, err := repository.Create(ctx, ContactGroupInput{
		Name: "default-" + uuid.NewString(), IsDefault: true,
		Members: []ContactGroupMemberInput{{
			TargetType: RecipientTargetUser, TargetID: userID.String(), ChannelLimit: []string{"email"},
		}},
	}, "integration")
	require.NoError(t, err)
	require.True(t, first.IsDefault)
	require.Len(t, first.Members, 1)

	second, err := repository.Create(ctx, ContactGroupInput{
		Name: "replacement-" + uuid.NewString(), IsDefault: true,
		Members: []ContactGroupMemberInput{{
			TargetType: RecipientTargetFixedContact, AddressCiphertext: []byte("ciphertext"),
			AddressKeyVersion: 1, RecipientFingerprint: []byte("fingerprint"), ChannelLimit: []string{"email"},
		}},
	}, "integration")
	require.NoError(t, err)
	require.True(t, second.IsDefault)
	first, err = repository.Get(ctx, first.ID)
	require.NoError(t, err)
	require.False(t, first.IsDefault)
	require.Equal(t, int64(2), first.Revision, "implicit default replacement must invalidate stale editors")
	targets, err := repository.ListMembers(ctx, second.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, []byte("ciphertext"), targets[0].AddressCiphertext)
	require.True(t, second.Members[0].AddressConfigured)

	_, err = repository.Update(ctx, second.ID, 9, ContactGroupInput{Name: second.Name}, "integration")
	require.ErrorIs(t, err, ErrRevisionMismatch)
}

func TestPgTemplateManagementRepository_Integration(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_RULE_TEST_DSN")
	if dsn == "" {
		t.Skip("set NOTIFICATION_RULE_TEST_DSN to a dedicated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	requireDedicatedNotificationDatabase(t, databaseName)

	repository := NewPgTemplateManagementRepository(pool)
	created, err := repository.Create(ctx, ManagedTemplateInput{
		Name: "email-" + uuid.NewString(), Channel: "email", Language: "zh-CN",
		Subject: "告警 {{.alarm_identifier}}", TextBody: "基站 {{.device_sn}}",
		Variables: []string{"alarm_identifier", "device_sn"}, ChangeReason: "initial",
	}, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(1), created.Revision)
	require.NotNil(t, created.Draft)
	version1ID := created.Draft.ID

	published, err := repository.Publish(ctx, created.ID, 1, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(2), published.Revision)
	require.Equal(t, version1ID, published.Published.ID)
	_, err = repository.Publish(ctx, created.ID, 2, "integration")
	require.ErrorIs(t, err, ErrDraftUnavailable)

	updated, err := repository.UpdateDraft(ctx, created.ID, 2, ManagedTemplateInput{
		Name: created.Name, Channel: "email", Language: "zh-CN", Subject: "升级 {{.alarm_identifier}}",
		TextBody: "设备 {{.device_sn}}", Variables: []string{"alarm_identifier", "device_sn"}, ChangeReason: "escalated",
	}, "integration")
	require.NoError(t, err)
	require.Equal(t, int64(3), updated.Revision)
	require.NotEqual(t, version1ID, updated.Draft.ID)
	require.Equal(t, version1ID, updated.Published.ID,
		"new draft must not silently change the published template version")
	var immutableSubject string
	require.NoError(t, pool.QueryRow(ctx, "SELECT subject FROM notification_template_versions WHERE id=$1", version1ID).Scan(&immutableSubject))
	require.Equal(t, "告警 {{.alarm_identifier}}", immutableSubject)

	_, err = repository.UpdateDraft(ctx, created.ID, 1, ManagedTemplateInput{
		Name: created.Name, Channel: "email", Language: "zh-CN", Subject: "subject", TextBody: "body",
	}, "integration")
	require.ErrorIs(t, err, ErrRevisionMismatch)
}

func TestPgChannelConfigRepository_Integration(t *testing.T) {
	dsn := os.Getenv("NOTIFICATION_RULE_TEST_DSN")
	if dsn == "" {
		t.Skip("set NOTIFICATION_RULE_TEST_DSN to a dedicated database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	var databaseName string
	require.NoError(t, pool.QueryRow(ctx, "SELECT current_database()").Scan(&databaseName))
	requireDedicatedNotificationDatabase(t, databaseName)

	firstID, secondID := uuid.New(), uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO notification_channel_configs
        (id,channel,name,enabled,parameters,created_by) VALUES
        ($1,'email',$2,false,'{}','integration'),($3,'email',$4,false,'{}','integration')`,
		firstID, "email-first-"+uuid.NewString(), secondID, "email-second-"+uuid.NewString())
	require.NoError(t, err)
	repository := NewPgChannelConfigRepository(pool)
	secretRef := "vault/notification/smtp"
	first, err := repository.Update(ctx, firstID, 1, ChannelConfigUpdate{
		Name: "email-first-enabled-" + uuid.NewString(), Enabled: true,
		Parameters: json.RawMessage(`{"host":"smtp.local","port":587}`), SecretRef: &secretRef,
	}, "integration")
	require.NoError(t, err)
	require.True(t, first.Enabled)
	require.True(t, first.SecretConfigured)
	require.Equal(t, int64(2), first.Revision)

	second, err := repository.Update(ctx, secondID, 1, ChannelConfigUpdate{
		Name: "email-second-enabled-" + uuid.NewString(), Enabled: true, Parameters: json.RawMessage(`{}`),
	}, "integration")
	require.NoError(t, err)
	require.True(t, second.Enabled)
	first, err = repository.Get(ctx, firstID)
	require.NoError(t, err)
	require.False(t, first.Enabled)
	require.Equal(t, int64(3), first.Revision,
		"enabling another email channel must invalidate stale editors of the previous default")

	_, err = repository.Update(ctx, firstID, 2, ChannelConfigUpdate{Name: first.Name, Parameters: json.RawMessage(`{}`)}, "integration")
	require.ErrorIs(t, err, ErrRevisionMismatch)
}

func requireDedicatedNotificationDatabase(t *testing.T, databaseName string) {
	t.Helper()
	require.True(t, strings.HasPrefix(databaseName, "omcgo_notification_"),
		"refusing to run notification integration tests against non-dedicated database %q", databaseName)
}
