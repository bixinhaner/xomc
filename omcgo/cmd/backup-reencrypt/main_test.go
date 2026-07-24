package main

import (
	"testing"

	"github.com/minio/minio-go/v7/pkg/s3utils"
	"github.com/stretchr/testify/require"
)

func TestResolveBackupBucket_NormalizesFlagEnvAndDefault(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		env        string
		wantBucket string
	}{
		{name: "legacy flag wins", flag: "config_backup", env: "other-bucket", wantBucket: "config-backup"},
		{name: "legacy env", env: "config_backup", wantBucket: "config-backup"},
		{name: "default", wantBucket: "config-backup"},
		{name: "unrelated bucket unchanged", flag: "operator-archive", wantBucket: "operator-archive"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bucket := resolveBackupBucket(tc.flag, func(key string) string {
				if key == envMinIOBucket {
					return tc.env
				}
				return ""
			})

			require.Equal(t, tc.wantBucket, bucket)
			require.NoError(t, s3utils.CheckValidBucketNameStrict(bucket))
		})
	}
}
