package miniointernal_test

import (
	"context"
	"testing"

	miniointernal "github.com.Vova4o/nasforhome/pkg/minio"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEndpoint  = "localhost:9000"
	testAccessKey = "minioadmin"
	testSecretKey = "minioadmin"
	testSecure    = false
)

// Заменяем реальные функции на тестовые заглушки
var (
	minioCreateFunc  = minio.New
	madminCreateFunc = madmin.NewWithOptions
)

// Функция для возврата оригинальных функций
func resetMocks() {
	minioCreateFunc = minio.New
	madminCreateFunc = madmin.NewWithOptions
}

// TestNewAdminClient проверяет создание административного клиента MinIO
func TestNewAdminClient(t *testing.T) {
	defer resetMocks()

	// Мокируем успешное создание клиентов
	minioCreateFunc = func(endpoint string, opts *minio.Options) (*minio.Client, error) {
		return &minio.Client{}, nil
	}

	madminCreateFunc = func(endpoint string, opts *madmin.Options) (*madmin.AdminClient, error) {
		return &madmin.AdminClient{}, nil
	}

	client, err := miniointernal.NewAdminClient(context.Background(), testEndpoint, testAccessKey, testSecretKey, testSecure)
	require.NoError(t, err, "ошибка создания admin клиента")

	assert.NotNil(t, client.Client, "клиент MinIO должен быть создан")
	assert.NotNil(t, client.AdminClient, "административный клиент MinIO должен быть создан")
}

// TestNewUserClient проверяет создание клиентского подключения MinIO
func TestNewUserClient(t *testing.T) {
	defer resetMocks()

	// Мокируем успешное создание клиента
	minioCreateFunc = func(endpoint string, opts *minio.Options) (*minio.Client, error) {
		return &minio.Client{}, nil
	}

	client, err := miniointernal.NewUserClient(context.Background(), testEndpoint, testAccessKey, testSecretKey, testSecure)
	require.NoError(t, err, "ошибка создания user клиента")

	assert.NotNil(t, client.Client, "клиент MinIO должен быть создан")
	assert.Nil(t, client.AdminClient, "у user-клиента не должно быть AdminClient")
}
