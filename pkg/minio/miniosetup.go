package miniointernal

import (
	"context"
	"fmt"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIO структура для хранения клиентов
type MinIO struct {
	Client      *minio.Client       // Для стандартных S3 операций
	AdminClient *madmin.AdminClient // Для административных операций
}

// NewAdminClient создает полное подключение с обоими типами клиентов
func NewAdminClient(ctx context.Context, endpoint, accessKey, secretKey string, secure bool) (*MinIO, error) {
	// Создаем обычный клиент
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиента MinIO: %w", err)
	}

	// Создаем административный клиент
	adminClient, err := madmin.NewWithOptions(endpoint, &madmin.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания административного клиента MinIO: %w", err)
	}

	return &MinIO{
		Client:      client,
		AdminClient: adminClient,
	}, nil
}

// NewUserClient создает клиент MinIO для конкретного пользователя
func NewUserClient(ctx context.Context, endpoint, accessKey, secretKey string, secure bool) (*MinIO, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пользовательского клиента MinIO: %w", err)
	}

	return &MinIO{
		Client:      client,
		AdminClient: nil,
	}, nil
}
