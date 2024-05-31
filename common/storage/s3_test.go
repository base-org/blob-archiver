package storage

import (
	"context"
	"os"
	"testing"

	"github.com/base-org/blob-archiver/common/flags"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

// Prior to running these tests, a local Minio server must be running.
// You can accomplish this with:
// docker compose down # shut down any running services
// docker compose up minio create-buckets # start the minio service
func setupS3(t *testing.T) *S3Storage {
	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration tests: set RUN_INTEGRATION_TESTS environment variable")
	}

	l := testlog.Logger(t, log.LvlInfo)

	s3, err := NewS3Storage(flags.S3Config{
		Endpoint:         "localhost:9000",
		AccessKey:        "admin",
		SecretAccessKey:  "password",
		UseHttps:         false,
		Bucket:           "blobs",
		S3CredentialType: flags.S3CredentialStatic,
	}, l)

	require.NoError(t, err)

	for object := range s3.s3.ListObjects(context.Background(), "blobs", minio.ListObjectsOptions{}) {
		err = s3.s3.RemoveObject(context.Background(), "blobs", object.Key, minio.RemoveObjectOptions{})
		require.NoError(t, err)
	}

	require.NoError(t, err)
	return s3
}

func TestS3Exists(t *testing.T) {
	s3 := setupS3(t)

	runTestExists(t, s3)
}

func TestS3Read(t *testing.T) {
	s3 := setupS3(t)

	runTestRead(t, s3)
}
