package minio

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectInfo структура для хранения информации об объекте
type ObjectInfo struct {
	Key          string
	LastModified string
	Size         int64
	ContentType  string
}

// Storager интерфейс для работы с MinIO
type Storager interface {
	ListBuckets(ctx context.Context) ([]minio.BucketInfo, error)
	// MakeBucket(ctx context.Context, bucketName, location string) error
	// RemoveBucket(ctx context.Context, bucketName string) error
	// ListObjects(ctx context.Context, bucketName, prefix string, recursive bool) ([]minio.ObjectInfo, error)
	// GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	// PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (n int64, err error)
	// RemoveObject(ctx context.Context, bucketName, objectName string) error
	// PutFolder(ctx context.Context, bucketName, folderName string) error
	// RemoveFolder(ctx context.Context, bucketName, folderName string) error
	// GetPresignedURL(ctx context.Context, bucketName, objectName string, expires int64, reqParams map[string]string) (string, error)
}

// MinIO структура, реализующая интерфейс MinioLocal
type MinIO struct {
	MinIoClient *minio.Client
}

// New создает новый экземпляр, реализующий интерфейс MinioLocal
func New(ctx context.Context, endpoint, user, password string, secure bool) (Storager, error) {
	// Создаем клиент MinIO
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(user, password, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента MinIO: %w", err)
	}

	// Проверяем подключение
	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к MinIO: %w", err)
	}

	return &MinIO{
		MinIoClient: minioClient,
	}, nil
}
