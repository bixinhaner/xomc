package storage

import (
	"testing"

	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/pkg/tr069"
	"github.com/stretchr/testify/require"
)

func TestBucketAndCategory_NormalizesLegacyConfigBackupBeforeUpload(t *testing.T) {
	buckets := appconfig.BucketConfig{ConfigBackup: "config_backup"}

	for _, fileType := range []tr069.FileType{tr069.FileTypeConfig, tr069.FileTypeSSLCert} {
		bucket, _ := BucketAndCategory(fileType, buckets)
		require.Equal(t, "config-backup", bucket)
		require.NoError(t, s3utils.CheckValidBucketNameStrict(bucket))
	}
}
