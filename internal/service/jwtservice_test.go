package service

import (
    "testing"
    "time"

    "github.com.Vova4o/nasforhome/pkg/models"
    "github.com/dgrijalva/jwt-go"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
)

// MockStorageDB мок для StorageDB
type MockStorageDB struct {
    mock.Mock
}

func (m *MockStorageDB) GetUserByID(id int) (*models.User, error) {
    args := m.Called(id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*models.User), args.Error(1)
}

// Реализация других необходимых методов интерфейса
func (m *MockStorageDB) Close() error { return nil }
func (m *MockStorageDB) CreateUser(username, passwordHash, email string, config *models.MinioConfig) (int, error) {
    return 0, nil
}
func (m *MockStorageDB) GetUserByUsername(username string) (*models.User, error) { return nil, nil }
func (m *MockStorageDB) UpdateUser(user *models.User) error { return nil }
func (m *MockStorageDB) DeleteUser(id int) error { return nil }
func (m *MockStorageDB) CreateMinIOUser(userID int, bucketName, accessKey, secretKey string) error {
    return nil
}
func (m *MockStorageDB) GetMinIOCredentials(userID int) (string, string, string, error) {
    return "", "", "", nil
}
func (m *MockStorageDB) InitDB() error { return nil }
func (m *MockStorageDB) GetCurrentDBVersion() (int, error) { return 0, nil }
func (m *MockStorageDB) MigrateTo(version int) error { return nil }

// TestGenerateTokenPair проверяет генерацию пары токенов
func TestGenerateTokenPair(t *testing.T) {
    // Установка
    service := &Service{
        JWTConfig: JWTConfig{
            AccessSecret:  "test-access-secret",
            RefreshSecret: "test-refresh-secret",
            AccessTTL:     900,    // 15 минут
            RefreshTTL:    604800, // 7 дней
        },
    }

    user := &models.User{
        ID:       1,
        UserName: "testuser",
    }

    // Действие
    tokens, err := service.GenerateTokenPair(user)

    // Проверки
    require.NoError(t, err, "Ошибка при генерации токенов")
    require.NotNil(t, tokens, "Токены не должны быть nil")
    require.NotEmpty(t, tokens.AccessToken, "Access-токен не должен быть пустым")
    require.NotEmpty(t, tokens.RefreshToken, "Refresh-токен не должен быть пустым")
    assert.Equal(t, 900, tokens.ExpiresIn, "Неправильное время жизни access-токена")
    assert.Equal(t, 604800, tokens.RefreshTTL, "Неправильное время жизни refresh-токена")

    // Дополнительно проверяем содержимое токенов
    claims, err := service.VerifyAccessToken(tokens.AccessToken)
    require.NoError(t, err, "Ошибка при проверке access-токена")
    assert.Equal(t, 1, claims.UserID, "Неправильный ID пользователя в токене")
    assert.Equal(t, "user", claims.Role, "Неправильная роль в токене")
    assert.Equal(t, "testuser", claims.Subject, "Неправильный subject в токене")
}

// TestVerifyAccessToken проверяет проверку access-токена
func TestVerifyAccessToken(t *testing.T) {
    service := &Service{
        JWTConfig: JWTConfig{
            AccessSecret:  "test-access-secret",
            RefreshSecret: "test-refresh-secret",
            AccessTTL:     900,
            RefreshTTL:    604800,
        },
    }

    // Создаем тестовые токены
    validToken := createTestToken(t, 1, "user", time.Now().Add(time.Hour).Unix(), service.JWTConfig.AccessSecret)
    expiredToken := createTestToken(t, 1, "user", time.Now().Add(-time.Hour).Unix(), service.JWTConfig.AccessSecret)
    invalidSignatureToken := createTestToken(t, 1, "user", time.Now().Add(time.Hour).Unix(), "wrong-secret")

    tests := []struct {
        name          string
        token         string
        expectedError bool
    }{
        {"Валидный токен", validToken, false},
        {"Истекший токен", expiredToken, true},
        {"Неверная подпись", invalidSignatureToken, true},
        {"Пустой токен", "", true},
        {"Некорректный токен", "invalid.token.format", true},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            claims, err := service.VerifyAccessToken(test.token)
            
            if test.expectedError {
                assert.Error(t, err, "Ожидалась ошибка")
                assert.Nil(t, claims, "Claims должны быть nil при ошибке")
            } else {
                assert.NoError(t, err, "Не ожидалась ошибка")
                assert.NotNil(t, claims, "Claims не должны быть nil")
                assert.Equal(t, 1, claims.UserID, "Неправильный ID пользователя")
                assert.Equal(t, "user", claims.Role, "Неправильная роль")
            }
        })
    }
}

// TestRefreshTokens проверяет обновление токенов
func TestRefreshTokens(t *testing.T) {
    mockStorage := new(MockStorageDB)
    service := &Service{
        JWTConfig: JWTConfig{
            AccessSecret:  "test-access-secret",
            RefreshSecret: "test-refresh-secret",
            AccessTTL:     900,
            RefreshTTL:    604800,
        },
        Storagedb: mockStorage,
    }

    validRefreshToken := createTestToken(t, 1, "", time.Now().Add(time.Hour).Unix(), service.JWTConfig.RefreshSecret)
    expiredRefreshToken := createTestToken(t, 1, "", time.Now().Add(-time.Hour).Unix(), service.JWTConfig.RefreshSecret)
    
    // Настраиваем мок для GetUserByID
    mockStorage.On("GetUserByID", 1).Return(&models.User{
        ID:       1,
        UserName: "testuser",
    }, nil)

    tests := []struct {
        name          string
        token         string
        expectedError bool
    }{
        {"Валидный refresh-токен", validRefreshToken, false},
        {"Истекший refresh-токен", expiredRefreshToken, true},
        {"Пустой токен", "", true},
    }

    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            tokens, err := service.RefreshTokens(test.token)
            
            if test.expectedError {
                assert.Error(t, err, "Ожидалась ошибка")
                assert.Nil(t, tokens, "Токены должны быть nil при ошибке")
            } else {
                assert.NoError(t, err, "Не ожидалась ошибка")
                assert.NotNil(t, tokens, "Токены не должны быть nil")
                assert.NotEmpty(t, tokens.AccessToken, "Access-токен не должен быть пустым")
                assert.NotEmpty(t, tokens.RefreshToken, "Refresh-токен не должен быть пустым")
            }
        })
    }

    mockStorage.AssertExpectations(t)
}

// createTestToken вспомогательная функция для создания тестовых токенов
func createTestToken(t *testing.T, userID int, role string, expiresAt int64, secret string) string {
    claims := &Claims{
        UserID: userID,
        Role:   role,
        StandardClaims: jwt.StandardClaims{
            ExpiresAt: expiresAt,
            IssuedAt:  time.Now().Unix(),
            Subject:   "testuser",
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(secret))
    require.NoError(t, err, "Ошибка при создании тестового токена")
    
    return tokenString
}