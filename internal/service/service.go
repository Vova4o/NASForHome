package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"strings"

	intminio "github.com.Vova4o/nasforhome/pkg/minio"
	"github.com.Vova4o/nasforhome/pkg/models"
	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"golang.org/x/crypto/bcrypt"
)

// Service сервисный слой
type Service struct {
	Storagedb      StoragerDB
	MinioAdmin     *intminio.MinIO
	MinioConfig    MinioConfig // Конфигурация для пользовательских клиентов
	JWTConfig      JWTConfig   // Конфигурация для JWT токенов
	ExecFileOpFunc func(ctx context.Context, userID int, operation FileOperationFunc) (any, error)
}

// StoragerDB интерфейс для работы с базой данных
type StoragerDB interface {
	// Операции с пользователями
	CreateUser(username, passwordHash, email string, config *models.MinioConfig) (int, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id int) error

	// Операции с MinIO для пользователя
	CreateMinIOUser(userID int, bucketName, accessKey, secretKey string) error
	GetMinIOCredentials(userID int) (string, string, string, error)
}

// MinioClientInterface интерфейс для работы с MinIO
type MinioClientInterface interface {
	ListObjects(ctx context.Context, bucketName string, opts minio.ListObjectsOptions) <-chan minio.ObjectInfo
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	// Добавьте другие используемые методы
}

// MinioConfig структура для создания пользовательских клиентов
type MinioConfig struct {
	Endpoint string
	Secure   bool
}

// FileOperationFunc функция обработки файловых операций
type FileOperationFunc func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (any, error)

// New создает сервис с админским подключением
func New(storagedb StoragerDB, minioAdmin *intminio.MinIO, minioConfig MinioConfig, jwtConfig JWTConfig) *Service {
	return &Service{
		Storagedb:   storagedb,
		MinioAdmin:  minioAdmin,
		MinioConfig: minioConfig,
		JWTConfig:   jwtConfig,
	}
}

// PasswordHash возвращает хеш пароля
func (s *Service) PasswordHash(password string) (string, error) {
	// Используем стандартную сложность вычислений для bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	return string(hashedPassword), nil
}

// VerifyPassword проверяет соответствие пароля хешу
func (s *Service) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// RegisterUser регистрирует нового пользователя со всеми необходимыми данными
func (s *Service) RegisterUser(ctx context.Context, username, password, email string) (*models.User, *TokenPair, error) {
	// Хешируем пароль
	passwordHash, err := s.PasswordHash(password)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка хеширования пароля: %w", err)
	}

	// Формируем данные для MinIO
	bucketName := "user-" + username
	accessKey := "user-" + username

	// Генерируем безопасный секретный ключ
	secretKey, err := s.generateSecretKey(32)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка генерации ключа: %w", err)
	}

	// Сначала создаем пользователя в БД со всеми данными MinIO
	userID, err := s.Storagedb.CreateUser(username, passwordHash, email, &models.MinioConfig{
		BucketName: bucketName,
		AccessKey:  accessKey,
		SecretKey:  secretKey,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	// 1. Создаем пользователя в MinIO
	err = s.MinioAdmin.AdminClient.AddUser(ctx, accessKey, secretKey)
	if err != nil {
		err := s.Storagedb.DeleteUser(userID)
		if err != nil {
			log.Printf("error deleting user: %v", err)
		}
		return nil, nil, fmt.Errorf("ошибка создания пользователя в MinIO: %w", err)
	}

	// 2. Привязываем политику (AttachPolicy вместо устаревшей SetPolicy)
	_, err = s.MinioAdmin.AdminClient.AttachPolicy(ctx, madmin.PolicyAssociationReq{
		Policies: []string{"readwrite"},
		User:     accessKey, // Используем accessKey как имя пользователя в MinIO
	})
	if err != nil {
		// Откат: если не удалось привязать политику, удаляем пользователя
		err = s.MinioAdmin.AdminClient.RemoveUser(ctx, accessKey)
		if err != nil {
			log.Printf("error MinIo admin client: %v", err)
		}
		return nil, nil, fmt.Errorf("ошибка привязки политики к пользователю: %w", err)
	}

	// 2. Создаем бакет в MinIO (с правами администратора)
	err = s.MinioAdmin.Client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" {
			// Удаляем созданного пользователя MinIO
			err = s.MinioAdmin.AdminClient.RemoveUser(ctx, accessKey)
			if err != nil {
				log.Printf("error MinIo admin client: %v", err)
			}
			err := s.Storagedb.DeleteUser(userID)
			if err != nil {
				log.Printf("error deleting user: %v", err)
			}
			return nil, nil, fmt.Errorf("ошибка создания бакета: %w", err)
		}
	}

	// Сохраняем данные MinIO в БД
	err = s.Storagedb.CreateMinIOUser(userID, bucketName, accessKey, secretKey)
	if err != nil {
		// Если не удалось сохранить данные, удаляем пользователя
		err := s.Storagedb.DeleteUser(userID)
		if err != nil {
			log.Printf("error deleting user: %v", err)
		}
		return nil, nil, fmt.Errorf("ошибка создания хранилища: %w", err)
	}

	// Получаем данные пользователя для генерации токенов
	user, err := s.Storagedb.GetUserByID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка получения данных пользователя: %w", err)
	}

	// Генерируем токены
	tokens, err := s.GenerateTokenPair(user)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания токенов: %w", err)
	}

	fmt.Printf("Tokens: %-v", tokens)

	return user, tokens, nil
}

// LoginUser выполняет вход пользователя и генерирует токены
func (s *Service) LoginUser(ctx context.Context, username, password string) (*models.User, *TokenPair, error) {
	// Получаем пользователя из БД
	user, err := s.Storagedb.GetUserByUsername(username)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка аутентификации: %w", err)
	}

	// Проверяем пароль
	if !s.VerifyPassword(password, user.PasswordHash) {
		return nil, nil, fmt.Errorf("неверный пароль")
	}

	// Генерируем токены
	tokens, err := s.GenerateTokenPair(user)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания токенов: %w", err)
	}

	return user, tokens, nil
}

// GetUserMinioClient создает динамическое подключение к MinIO для пользователя
func (s *Service) GetUserMinioClient(ctx context.Context, userID int) (*minio.Client, error) {
	_, accessKey, secretKey, err := s.Storagedb.GetMinIOCredentials(userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения учетных данных пользователя: %w", err)
	}

	// Создаем клиент для пользователя
	client, err := intminio.NewUserClient(ctx, s.MinioConfig.Endpoint, accessKey, secretKey, s.MinioConfig.Secure)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания клиентского подключения MinIO: %w", err)
	}

	minioClient := client.Client

	return minioClient, nil
}

// ExecuteFileOperation универсальная функция для выполнения операций с файлами
func (s *Service) ExecuteFileOperation(ctx context.Context, userID int, operation FileOperationFunc) (any, error) {
	// Если установлена тестовая функция, используем её
	if s.ExecFileOpFunc != nil {
		return s.ExecFileOpFunc(ctx, userID, operation)
	}

	// Оригинальная реализация
	minioClient, err := s.GetUserMinioClient(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к хранилищу: %w", err)
	}

	bucketName, _, _, err := s.Storagedb.GetMinIOCredentials(userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных хранилища: %w", err)
	}

	// *minio.Client автоматически подходит под интерфейс MinioClientInterface
	return operation(ctx, minioClient, bucketName)
}

// ListUserFiles возвращает список файлов пользователя
func (s *Service) ListUserFiles(ctx context.Context, userID int, prefix string, recursive bool) ([]minio.ObjectInfo, error) {
	result, err := s.ExecuteFileOperation(ctx, userID, func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (interface{}, error) {
		// Получаем список объектов
		objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: recursive,
		})

		var files []minio.ObjectInfo
		for obj := range objectCh {
			if obj.Err != nil {
				return nil, obj.Err
			}

			// Пропускаем объекты, представляющие папки
			if obj.Size == 0 && len(obj.Key) > 0 && obj.Key[len(obj.Key)-1] == '/' {
				continue
			}

			files = append(files, obj)
		}

		return files, nil
	})
	if err != nil {
		return nil, err
	}

	return result.([]minio.ObjectInfo), nil
}

// GetUserFile возвращает файл пользователя
func (s *Service) GetUserFile(ctx context.Context, userID int, filename string) (*minio.Object, *minio.ObjectInfo, error) {
	result, err := s.ExecuteFileOperation(ctx, userID, func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (interface{}, error) {
		// Получаем объект
		object, err := minioClient.GetObject(ctx, bucketName, filename, minio.GetObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("ошибка получения файла: %w", err)
		}

		// Получаем информацию об объекте
		stat, err := object.Stat()
		if err != nil {
			return nil, fmt.Errorf("ошибка получения информации о файле: %w", err)
		}

		return []interface{}{object, stat}, nil
	})
	if err != nil {
		return nil, nil, err
	}

	resultArray := result.([]interface{})
	object := resultArray[0].(*minio.Object)
	stat := resultArray[1].(minio.ObjectInfo)

	return object, &stat, nil
}

// DeleteUserFile удаляет файл пользователя
func (s *Service) DeleteUserFile(ctx context.Context, userID int, filename string) error {
	_, err := s.ExecuteFileOperation(ctx, userID, func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (interface{}, error) {
		// Удаляем объект
		return nil, minioClient.RemoveObject(ctx, bucketName, filename, minio.RemoveObjectOptions{})
	})

	return err
}

// UploadUserFile function to uplad files to bucket.
func (s *Service) UploadUserFile(ctx context.Context, userID int, objectName string, reader io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	result, err := s.ExecuteFileOperation(ctx, userID, func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (any, error) {
		// Загрузка файла в MinIO
		uploadInfo, err := minioClient.PutObject(ctx, bucketName, objectName, reader, size, minio.PutObjectOptions{
			ContentType: contentType,
		})
		if err != nil {
			return nil, fmt.Errorf("ошибка загрузки файла: %w", err)
		}
		return uploadInfo, nil
	})
	if err != nil {
		return minio.UploadInfo{}, err
	}

	// Безопасное приведение типа
	uploadInfo, ok := result.(minio.UploadInfo)
	if !ok {
		return minio.UploadInfo{}, fmt.Errorf("не удалось преобразовать результат в minio.UploadInfo")
	}

	return uploadInfo, nil
}

// CreateUserFolder создает папку пользователя
func (s *Service) CreateUserFolder(ctx context.Context, userID int, folderName string) error {
	_, err := s.ExecuteFileOperation(ctx, userID, func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (interface{}, error) {
		// Убеждаемся, что folderName заканчивается на "/"
		if folderName[len(folderName)-1] != '/' {
			folderName += "/"
		}

		// Создаем папку
		_, err := minioClient.PutObject(ctx, bucketName, folderName, nil, 0, minio.PutObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("ошибка создания папки: %w", err)
		}

		return nil, nil
	})

	return err
}

// DeleteUserFolder удаляет папку пользователя
func (s *Service) DeleteUserFolder(ctx context.Context, userID int, folderName string) error {
	_, err := s.ExecuteFileOperation(ctx, userID, func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (interface{}, error) {
		// Убеждаемся, что folderName заканчивается на "/"
		if folderName[len(folderName)-1] != '/' {
			folderName += "/"
		}

		// Для рекурсивного удаления папки:
		objectsCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
			Prefix:    folderName,
			Recursive: true,
		})

		for obj := range objectsCh {
			if obj.Err != nil {
				return nil, obj.Err
			}
			err := minioClient.RemoveObject(ctx, bucketName, obj.Key, minio.RemoveObjectOptions{})
			if err != nil {
				return nil, err
			}
		}

		// Удаляем саму папку (без параметра Recursive)
		err := minioClient.RemoveObject(ctx, bucketName, folderName, minio.RemoveObjectOptions{})
		return nil, err
	})

	return err
}

// ListUserFolders возвращает список папок пользователя
func (s *Service) ListUserFolders(ctx context.Context, userID int, prefix string) ([]string, error) {
	result, err := s.ExecuteFileOperation(ctx, userID, func(ctx context.Context, minioClient MinioClientInterface, bucketName string) (interface{}, error) {
		// Получаем список объектов
		objectCh := minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: false,
		})

		var folders []string
		for obj := range objectCh {
			if obj.Err != nil {
				return nil, obj.Err
			}

			// Проверяем, является ли объект папкой (размер 0 и заканчивается на '/')
			if obj.Size == 0 && len(obj.Key) > 0 && obj.Key[len(obj.Key)-1] == '/' {
				// Удаляем trailing slash для красивого отображения
				folderName := obj.Key[:len(obj.Key)-1]

				// Если указан префикс, удаляем его из имени папки
				if prefix != "" && strings.HasPrefix(folderName, prefix) {
					folderName = folderName[len(prefix):]
				}

				// Если в результате получилась непустая строка, добавляем в список
				if folderName != "" {
					folders = append(folders, folderName)
				}
			}
		}

		return folders, nil
	})
	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// generateSecretKey генерирует криптографически стойкий случайный ключ
func (s *Service) generateSecretKey(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Используем URLEncoding для получения строки, безопасной для URL
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}
