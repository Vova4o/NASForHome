package service_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com.Vova4o/nasforhome/internal/service"
	"github.com.Vova4o/nasforhome/pkg/models"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockStorageDB struct {
	mock.Mock
}

func (m *MockStorageDB) CreateUser(username, passwordHash, email string, config *models.MinioConfig) (int, error) {
	args := m.Called(username, passwordHash, email, config)
	return args.Int(0), args.Error(1)
}

func (m *MockStorageDB) GetUserByUsername(username string) (*models.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockStorageDB) GetUserByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockStorageDB) UpdateUser(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockStorageDB) DeleteUser(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockStorageDB) CreateMinIOUser(userID int, bucketName, accessKey, secretKey string) error {
	args := m.Called(userID, bucketName, accessKey, secretKey)
	return args.Error(0)
}

func (m *MockStorageDB) GetMinIOCredentials(userID int) (string, string, string, error) {
	args := m.Called(userID)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

// MockMinIO мок для MinIO
type MockMinIO struct {
	mock.Mock
	Client      *MockMinioClient
	AdminClient *MockAdminClient
}

// MockMinioClient мок для minio.Client
type MockMinioClient struct {
	mock.Mock
}

func (m *MockMinioClient) PutObject(ctx context.Context, bucketName, objectName string,
	reader io.Reader, size int64, opts minio.PutObjectOptions,
) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, reader, size, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func (m *MockMinioClient) GetObject(ctx context.Context, bucketName, objectName string,
	opts minio.GetObjectOptions,
) (*minio.Object, error) {
	args := m.Called(ctx, bucketName, objectName, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*minio.Object), args.Error(1)
}

func (m *MockMinioClient) ListObjects(ctx context.Context, bucketName string,
	opts minio.ListObjectsOptions,
) <-chan minio.ObjectInfo {
	args := m.Called(ctx, bucketName, opts)
	return args.Get(0).(<-chan minio.ObjectInfo)
}

func (m *MockMinioClient) RemoveObject(ctx context.Context, bucketName, objectName string,
	opts minio.RemoveObjectOptions,
) error {
	args := m.Called(ctx, bucketName, objectName, opts)
	return args.Error(0)
}

func (m *MockMinioClient) MakeBucket(ctx context.Context, bucketName string,
	opts minio.MakeBucketOptions,
) error {
	args := m.Called(ctx, bucketName, opts)
	return args.Error(0)
}

// MockAdminClient мок для madmin.AdminClient
type MockAdminClient struct {
	mock.Mock
}

func (m *MockAdminClient) AddUser(ctx context.Context, accessKey, secretKey string) error {
	args := m.Called(ctx, accessKey, secretKey)
	return args.Error(0)
}

func (m *MockAdminClient) AttachPolicy(ctx context.Context, req madmin.PolicyAssociationReq) (madmin.PolicyAssociationResp, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(madmin.PolicyAssociationResp), args.Error(1)
}

func (m *MockAdminClient) RemoveUser(ctx context.Context, accessKey string) error {
	args := m.Called(ctx, accessKey)
	return args.Error(0)
}

// TestPasswordHash проверяет хеширование пароля
func TestPasswordHash(t *testing.T) {
	srv := &service.Service{}
	password := "testpassword"

	// Проверяем хеширование пароля
	hash, err := srv.PasswordHash(password)
	require.NoError(t, err, "Ошибка хеширования пароля")
	assert.NotEmpty(t, hash, "Хеш не должен быть пустым")
	assert.NotEqual(t, password, hash, "Хеш должен отличаться от пароля")

	// Проверяем, что разные вызовы дают разные хеши (соль)
	hash2, err := srv.PasswordHash(password)
	require.NoError(t, err)
	assert.NotEqual(t, hash, hash2, "Разные вызовы должны давать разные хеши")

	// Проверяем верификацию пароля
	assert.True(t, srv.VerifyPassword(password, hash), "Пароль должен проходить проверку")
	assert.False(t, srv.VerifyPassword("wrongpassword", hash), "Неверный пароль не должен проходить проверку")
}

// TestLoginUser проверяет вход пользователя
func TestLoginUser(t *testing.T) {
	// Создаем моки и сервис
	mockStorage := new(MockStorageDB)
	srv := &service.Service{
		Storagedb: mockStorage,
		JWTConfig: service.JWTConfig{
			AccessSecret:  "test-access-secret",
			RefreshSecret: "test-refresh-secret",
			AccessTTL:     900,
			RefreshTTL:    604800,
		},
	}

	// Создаем тестовые данные
	username := "testuser"
	password := "testpassword"

	// Хешируем пароль для теста
	hash, err := srv.PasswordHash(password)
	require.NoError(t, err)

	// Настраиваем мок для успешного входа
	mockStorage.On("GetUserByUsername", username).Return(&models.User{
		ID:           1,
		UserName:     username,
		PasswordHash: hash,
		Email:        "test@example.com",
	}, nil)

	// Проверяем успешный вход
	user, tokens, err := srv.LoginUser(context.Background(), username, password)
	assert.NoError(t, err, "Вход должен быть успешным")
	assert.NotNil(t, user, "Пользователь не должен быть nil")
	assert.NotNil(t, tokens, "Токены не должны быть nil")
	assert.NotEmpty(t, tokens.AccessToken, "Access токен не должен быть пустым")
	assert.NotEmpty(t, tokens.RefreshToken, "Refresh токен не должен быть пустым")

	// Проверяем неверный пароль
	_, _, err = srv.LoginUser(context.Background(), username, "wrongpassword")
	assert.Error(t, err, "Неверный пароль должен вызывать ошибку")

	// Настраиваем мок для несуществующего пользователя
	mockStorage.On("GetUserByUsername", "nonexistent").Return(nil, errors.New("пользователь не найден"))

	// Проверяем несуществующего пользователя
	_, _, err = srv.LoginUser(context.Background(), "nonexistent", password)
	assert.Error(t, err, "Несуществующий пользователь должен вызывать ошибку")

	// Проверяем ожидания мока
	mockStorage.AssertExpectations(t)
}

// TestUploadUserFile проверяет загрузку файла
func TestUploadUserFile(t *testing.T) {
	// Создаем моки
	mockStorage := new(MockStorageDB)
	mockMinioClient := new(MockMinioClient)

	// Настраиваем сервис с моками
	srv := &service.Service{
		Storagedb: mockStorage,
		MinioConfig: service.MinioConfig{
			Endpoint: "localhost:9000",
			Secure:   false,
		},
	}

	// Мокируем интерфейс для GetUserMinioClient
	userID := 1
	bucketName := "test-bucket"
	accessKey := "test-access"
	secretKey := "test-secret"

	// Настраиваем мок для получения учетных данных
	mockStorage.On("GetMinIOCredentials", userID).Return(bucketName, accessKey, secretKey, nil)

	// Настраиваем данные для загрузки
	objectName := "test-file.txt"
	content := "test content"
	reader := strings.NewReader(content)
	size := int64(len(content))
	contentType := "text/plain"

	// Создаем ожидаемый результат
	expectedUploadInfo := minio.UploadInfo{
		Bucket: bucketName,
		Key:    objectName,
		ETag:   "test-etag",
		Size:   size,
	}

	// Устанавливаем мок-функцию вместо сохранения старой
	srv.ExecFileOpFunc = func(ctx context.Context, userID int, operation service.FileOperationFunc) (any, error) {
		// Вызываем операцию с мок-клиентом
		mockMinioClient.On("PutObject", ctx, bucketName, objectName, reader, size,
			minio.PutObjectOptions{ContentType: contentType}).Return(expectedUploadInfo, nil)

		return operation(ctx, mockMinioClient, bucketName)
	}

	// Выполняем тест
	uploadInfo, err := srv.UploadUserFile(context.Background(), userID, objectName, reader, size, contentType)

	// Проверяем результаты
	assert.NoError(t, err, "Загрузка файла должна быть успешной")
	assert.Equal(t, expectedUploadInfo, uploadInfo, "Информация о загрузке должна совпадать")

	// Проверяем ожидания моков
	mockMinioClient.AssertExpectations(t)
}
