package service

import (
	"fmt"
	"time"

	"github.com.Vova4o/nasforhome/pkg/models"
	"github.com/dgrijalva/jwt-go"
)

// JWTConfig конфигурация для JWT токенов
type JWTConfig struct {
	AccessSecret  string // Секретный ключ для подписи access токенов
	RefreshSecret string // Секретный ключ для подписи refresh токенов
	AccessTTL     int    // Время жизни access токена в секундах (например, 15 минут)
	RefreshTTL    int    // Время жизни refresh токена в секундах (например, 7 дней)
}

// TokenPair представляет пару access и refresh токенов
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`  // Время жизни access токена в секундах
	RefreshTTL   int    `json:"refresh_ttl"` // Время жизни refresh токена в секундах
}

// Claims стандартные данные для JWT токена
type Claims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.StandardClaims
}

// GenerateTokenPair создает новую пару токенов для пользователя
func (s *Service) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	// Текущее время для расчета времени истечения токенов
	now := time.Now()

	// Claims для access токена
	accessClaims := &Claims{
		UserID: user.ID,
		Role:   "user", // Можно добавить роли для разграничения прав
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: now.Add(time.Duration(s.JWTConfig.AccessTTL) * time.Second).Unix(),
			IssuedAt:  now.Unix(),
			Subject:   user.UserName,
		},
	}

	// Создаем access токен
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.JWTConfig.AccessSecret))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания access токена: %w", err)
	}

	// Claims для refresh токена
	refreshClaims := &Claims{
		UserID: user.ID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: now.Add(time.Duration(s.JWTConfig.RefreshTTL) * time.Second).Unix(),
			IssuedAt:  now.Unix(),
			Subject:   user.UserName,
		},
	}

	// Создаем refresh токен
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(s.JWTConfig.RefreshSecret))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания refresh токена: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    s.JWTConfig.AccessTTL,
		RefreshTTL:   s.JWTConfig.RefreshTTL,
	}, nil
}

// VerifyAccessToken проверяет валидность access токена
func (s *Service) VerifyAccessToken(tokenString string) (*Claims, error) {
	// Парсим токен
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Проверяем алгоритм подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(s.JWTConfig.AccessSecret), nil
	})
	// Проверяем ошибки и валидность токена
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки токена: %w", err)
	}

	// Проверяем, что токен валидный и получаем claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("недействительный токен")
}

// RefreshTokens обновляет пару токенов с помощью refresh токена
func (s *Service) RefreshTokens(refreshTokenString string) (*TokenPair, error) {
	// Парсим refresh токен
	token, err := jwt.ParseWithClaims(refreshTokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("неожиданный метод подписи: %v", token.Header["alg"])
		}
		return []byte(s.JWTConfig.RefreshSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки refresh токена: %w", err)
	}

	// Проверяем валидность токена и получаем claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Получаем пользователя по ID из токена
		user, err := s.Storagedb.GetUserByID(claims.UserID)
		if err != nil {
			return nil, fmt.Errorf("пользователь не найден: %w", err)
		}

		// Генерируем новую пару токенов
		return s.GenerateTokenPair(user)
	}

	return nil, fmt.Errorf("недействительный refresh токен")
}
