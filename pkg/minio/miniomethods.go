package minio

import (
	"context"

	"github.com/minio/minio-go/v7"
)

func (m *MinIO) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return m.MinIoClient.ListBuckets(ctx)
}
