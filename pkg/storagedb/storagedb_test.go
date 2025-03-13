package storagedb

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com.Vova4o/nasforhome/pkg/models"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClose проверяет закрытие соединения
func TestClose(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	mock.ExpectClose()

	// Закрываем соединение
	err = storage.Close()

	// Проверяем результат
	assert.NoError(t, err, "Закрытие соединения должно пройти без ошибок")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestCreateUser проверяет создание пользователя
func TestCreateUser(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	username := "testuser"
	passwordHash := "hashedpassword"
	email := "test@example.com"
	minioConfig := &models.MinioConfig{
		BucketName: "test-bucket",
		AccessKey:  "test-access",
		SecretKey:  "test-secret",
	}
	expectedID := 1

	// Настраиваем ожидания для транзакций и запросов
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO users").
		WithArgs(username, passwordHash, email, minioConfig.BucketName, minioConfig.AccessKey, minioConfig.SecretKey).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedID))
	mock.ExpectCommit()

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	id, err := storage.CreateUser(username, passwordHash, email, minioConfig)

	// Проверяем результаты
	assert.NoError(t, err, "Создание пользователя должно пройти без ошибок")
	assert.Equal(t, expectedID, id, "ID должен соответствовать ожидаемому")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestCreateUserError проверяет обработку ошибок при создании пользователя
func TestCreateUserError(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	username := "testuser"
	passwordHash := "hashedpassword"
	email := "test@example.com"
	minioConfig := &models.MinioConfig{
		BucketName: "test-bucket",
		AccessKey:  "test-access",
		SecretKey:  "test-secret",
	}

	// Настраиваем ожидания для ошибки при выполнении запроса
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO users").
		WithArgs(username, passwordHash, email, minioConfig.BucketName, minioConfig.AccessKey, minioConfig.SecretKey).
		WillReturnError(errors.New("ошибка создания пользователя"))
	mock.ExpectRollback()

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	_, err = storage.CreateUser(username, passwordHash, email, minioConfig)

	// Проверяем результаты
	assert.Error(t, err, "Должна быть возвращена ошибка")
	assert.Contains(t, err.Error(), "ошибка создания пользователя", "Текст ошибки должен содержать ожидаемое сообщение")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestGetUserByUsername проверяет получение пользователя по имени пользователя
func TestGetUserByUsername(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	username := "testuser"
	now := time.Now()

	// Настраиваем ожидания для запроса
	columns := []string{
		"id", "user_name", "password_hash", "email", "minio_bucket_name",
		"minio_access_key", "minio_secret_key", "created_at", "updated_at",
	}
	mock.ExpectQuery("SELECT .* FROM users WHERE user_name").
		WithArgs(username).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(1, username, "hashedpass", "test@example.com", "bucket", "access", "secret", now, now))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	user, err := storage.GetUserByUsername(username)

	// Проверяем результаты
	assert.NoError(t, err, "Получение пользователя должно пройти без ошибок")
	assert.NotNil(t, user, "Пользователь должен быть возвращен")
	assert.Equal(t, username, user.UserName, "Имя пользователя должно соответствовать ожидаемому")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestGetUserByUsernameNotFound проверяет случай, когда пользователь не найден
func TestGetUserByUsernameNotFound(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	username := "nonexistent"

	// Настраиваем ожидания для запроса с пустым результатом
	mock.ExpectQuery("SELECT .* FROM users WHERE user_name").
		WithArgs(username).
		WillReturnError(sql.ErrNoRows)

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	user, err := storage.GetUserByUsername(username)

	// Проверяем результаты
	assert.Error(t, err, "Должна быть возвращена ошибка")
	assert.Nil(t, user, "Пользователь не должен быть возвращен")
	assert.Contains(t, err.Error(), "пользователь не найден", "Текст ошибки должен содержать ожидаемое сообщение")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestGetUserByID проверяет получение пользователя по ID
func TestGetUserByID(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	userID := 1
	username := "testuser"
	now := time.Now()

	// Настраиваем ожидания для запроса
	columns := []string{
		"id", "user_name", "password_hash", "email", "minio_bucket_name",
		"minio_access_key", "minio_secret_key", "created_at", "updated_at",
	}
	mock.ExpectQuery("SELECT .* FROM users WHERE id").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows(columns).
			AddRow(userID, username, "hashedpass", "test@example.com", "bucket", "access", "secret", now, now))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	user, err := storage.GetUserByID(userID)

	// Проверяем результаты
	assert.NoError(t, err, "Получение пользователя должно пройти без ошибок")
	assert.NotNil(t, user, "Пользователь должен быть возвращен")
	assert.Equal(t, userID, user.ID, "ID пользователя должен соответствовать ожидаемому")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestUpdateUser проверяет обновление информации о пользователе
func TestUpdateUser(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	user := &models.User{
		ID:              1,
		UserName:        "updateduser",
		Email:           "updated@example.com",
		MinioBucketName: "updated-bucket",
		MinioAccessKey:  "updated-access",
		MinioSecretKey:  "updated-secret",
	}

	// Настраиваем ожидания для запроса обновления
	mock.ExpectExec("UPDATE users SET").
		WithArgs(user.UserName, user.Email, user.MinioBucketName, user.MinioAccessKey, user.MinioSecretKey, user.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	err = storage.UpdateUser(user)

	// Проверяем результаты
	assert.NoError(t, err, "Обновление пользователя должно пройти без ошибок")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestDeleteUser проверяет удаление пользователя
func TestDeleteUser(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	userID := 1

	// Настраиваем ожидания для запроса удаления
	mock.ExpectExec("DELETE FROM users").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	err = storage.DeleteUser(userID)

	// Проверяем результаты
	assert.NoError(t, err, "Удаление пользователя должно пройти без ошибок")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestDeleteUserNotFound проверяет ошибку при удалении несуществующего пользователя
func TestDeleteUserNotFound(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	userID := 999

	// Настраиваем ожидания для запроса удаления с нулевым результатом
	mock.ExpectExec("DELETE FROM users").
		WithArgs(userID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	err = storage.DeleteUser(userID)

	// Проверяем результаты
	assert.Error(t, err, "Должна быть возвращена ошибка")
	assert.Contains(t, err.Error(), "пользователь с ID 999 не найден", "Текст ошибки должен содержать ожидаемое сообщение")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestCreateMinIOUser проверяет связывание пользователя с данными MinIO
func TestCreateMinIOUser(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	userID := 1
	bucketName := "test-bucket"
	accessKey := "test-access"
	secretKey := "test-secret"

	// Настраиваем ожидания для запроса обновления
	mock.ExpectExec("UPDATE users SET minio_bucket_name").
		WithArgs(bucketName, accessKey, secretKey, userID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	err = storage.CreateMinIOUser(userID, bucketName, accessKey, secretKey)

	// Проверяем результаты
	assert.NoError(t, err, "Связывание с MinIO должно пройти без ошибок")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestGetMinIOCredentials проверяет получение учетных данных MinIO
func TestGetMinIOCredentials(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	userID := 1
	bucketName := "test-bucket"
	accessKey := "test-access"
	secretKey := "test-secret"

	// Настраиваем ожидания для запроса
	mock.ExpectQuery("SELECT minio_bucket_name, minio_access_key, minio_secret_key FROM users").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"minio_bucket_name", "minio_access_key", "minio_secret_key"}).
			AddRow(bucketName, accessKey, secretKey))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	b, a, s, err := storage.GetMinIOCredentials(userID)

	// Проверяем результаты
	assert.NoError(t, err, "Получение данных MinIO должно пройти без ошибок")
	assert.Equal(t, bucketName, b, "Имя бакета должно соответствовать ожидаемому")
	assert.Equal(t, accessKey, a, "Access key должен соответствовать ожидаемому")
	assert.Equal(t, secretKey, s, "Secret key должен соответствовать ожидаемому")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}

// TestGetMinIOCredentialsEmpty проверяет случай, когда данные MinIO не настроены
func TestGetMinIOCredentialsEmpty(t *testing.T) {
	// Создаем мок БД
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Создаем тестовые данные
	userID := 1

	// Настраиваем ожидания для запроса с пустыми значениями
	mock.ExpectQuery("SELECT minio_bucket_name, minio_access_key, minio_secret_key FROM users").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"minio_bucket_name", "minio_access_key", "minio_secret_key"}).
			AddRow("", "", ""))

	// Создаем экземпляр StorageDB с моком
	storage := &StorageDB{db: db}

	// Вызываем тестируемый метод
	_, _, _, err = storage.GetMinIOCredentials(userID)

	// Проверяем результаты
	assert.Error(t, err, "Должна быть возвращена ошибка")
	assert.Contains(t, err.Error(), "не настроен MinIO", "Текст ошибки должен содержать ожидаемое сообщение")
	assert.NoError(t, mock.ExpectationsWereMet(), "Все ожидания должны быть выполнены")
}
