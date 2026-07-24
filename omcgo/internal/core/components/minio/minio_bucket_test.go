package minio

import (
	"context"
	"testing"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/stretchr/testify/require"
)

type recordingBucketClient struct {
	existsBuckets []string
	madeBuckets   []string
}

func (c *recordingBucketClient) BucketExists(_ context.Context, bucket string) (bool, error) {
	c.existsBuckets = append(c.existsBuckets, bucket)
	return false, nil
}

func (c *recordingBucketClient) MakeBucket(_ context.Context, bucket string, _ miniogo.MakeBucketOptions) error {
	c.madeBuckets = append(c.madeBuckets, bucket)
	return nil
}

func TestEnsureBuckets_NormalizesLegacyConfigBackupBeforeMinIO(t *testing.T) {
	client := &recordingBucketClient{}

	err := EnsureBuckets(context.Background(), client, appconfig.BucketConfig{
		ConfigBackup: "config_backup",
	})

	require.NoError(t, err)
	require.Equal(t, []string{"config-backup"}, client.existsBuckets)
	require.Equal(t, []string{"config-backup"}, client.madeBuckets)
	require.NoError(t, s3utils.CheckValidBucketNameStrict(client.existsBuckets[0]))
	require.NoError(t, s3utils.CheckValidBucketNameStrict(client.madeBuckets[0]))
}
