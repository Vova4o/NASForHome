package storagedb

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com.Vova4o/nasforhome/pkg/models"
	_ "github.com/lib/pq" // Драйвер PostgreSQL
)

// StorageDB структура для простого подключения к PostgreSQL
type StorageDB struct {
	db *sql.DB
}

// StoragerDB интерфейс для работы с базой данных
type StoragerDB interface {
	// Общие операции с БД
	Close() error

	// Операции с пользователями
	CreateUser(username, passwordHash, email string, config *models.MinioConfig) (int, error)
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id int) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(id int) error

	// Операции с MinIO для пользователя
	CreateMinIOUser(userID int, bucketName, accessKey, secretKey string) error
	GetMinIOCredentials(userID int) (string, string, string, error)

	// Управление миграциями
	InitDB() error
	GetCurrentDBVersion() (int, error)
	MigrateTo(version int) error
}

// Константы для SQL запросов
const (
	selectUserByIDSQL = `
        SELECT id, user_name, password_hash, email, minio_bucket_name, 
               minio_access_key, minio_secret_key, created_at, updated_at
        FROM users
        WHERE id = $1
    `

	selectUserByUsernameSQL = `
        SELECT id, user_name, password_hash, email, minio_bucket_name, 
               minio_access_key, minio_secret_key, created_at, updated_at
        FROM users
        WHERE user_name = $1
    `

	updateUserSQL = `
        UPDATE users
        SET user_name = $1, email = $2, minio_bucket_name = $3, 
            minio_access_key = $4, minio_secret_key = $5
        WHERE id = $6
    `

	deleteUserSQL = "DELETE FROM users WHERE id = $1"

	createUserSQL = `
        INSERT INTO users (user_name, password_hash, email, minio_bucket_name, minio_access_key, minio_secret_key) 
        VALUES ($1, $2, $3, $4, $5, $6) 
        RETURNING id
    `

	updateMinioUserSQL = `
        UPDATE users
        SET minio_bucket_name = $1, minio_access_key = $2, minio_secret_key = $3
        WHERE id = $4
    `

	getMinioCredsSQL = `
        SELECT minio_bucket_name, minio_access_key, minio_secret_key
        FROM users
        WHERE id = $1
    `
)

// New создание нового экземпляра StorageDB
func New(host, port, user, password, dbname string) (StoragerDB, error) {
	// Формируем строку подключения
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	// Открываем соединение
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &StorageDB{db: db}, nil
}

// Close закрывает соединение с базой данных
func (s *StorageDB) Close() error {
	return s.db.Close()
}

// scanUser сканирует результат запроса в структуру User
func scanUser(row *sql.Row) (*models.User, error) {
	user := &models.User{}
	err := row.Scan(
		&user.ID,
		&user.UserName,
		&user.PasswordHash,
		&user.Email,
		&user.MinioBucketName,
		&user.MinioAccessKey,
		&user.MinioSecretKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("пользователь не найден")
		}
		return nil, fmt.Errorf("ошибка сканирования данных пользователя: %w", err)
	}
	return user, nil
}

// CreateUser создает нового пользователя со всеми необходимыми данными
func (s *StorageDB) CreateUser(username, passwordHash, email string, minioConfig *models.MinioConfig) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}

	defer func() {
		if err != nil {
			err := tx.Rollback()
			if err != nil {
				fmt.Printf("ошибка отката транзакции: %v\n", err)
			}
		}
	}()

	var id int
	err = tx.QueryRow(createUserSQL, username, passwordHash, email,
		minioConfig.BucketName, minioConfig.AccessKey, minioConfig.SecretKey).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания пользователя: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return 0, fmt.Errorf("ошибка подтверждения транзакции: %w", err)
	}

	return id, nil
}

// GetUserByUsername возвращает пользователя по имени пользователя
func (s *StorageDB) GetUserByUsername(username string) (*models.User, error) {
	row := s.db.QueryRow(selectUserByUsernameSQL, username)
	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя по имени %s: %w", username, err)
	}
	return user, nil
}

// GetUserByID возвращает пользователя по ID
func (s *StorageDB) GetUserByID(id int) (*models.User, error) {
	row := s.db.QueryRow(selectUserByIDSQL, id)
	user, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения пользователя по ID %d: %w", id, err)
	}
	return user, nil
}

// UpdateUser обновляет информацию о пользователе
func (s *StorageDB) UpdateUser(user *models.User) error {
	_, err := s.db.Exec(updateUserSQL,
		user.UserName,
		user.Email,
		user.MinioBucketName,
		user.MinioAccessKey,
		user.MinioSecretKey,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("ошибка обновления пользователя: %w", err)
	}
	return nil
}

// DeleteUser удаляет пользователя по ID
func (s *StorageDB) DeleteUser(id int) error {
	result, err := s.db.Exec(deleteUserSQL, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления пользователя: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка определения количества удаленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с ID %d не найден", id)
	}

	return nil
}

// CreateMinIOUser связывает пользователя с данными MinIO
func (s *StorageDB) CreateMinIOUser(userID int, bucketName, accessKey, secretKey string) error {
	result, err := s.db.Exec(updateMinioUserSQL, bucketName, accessKey, secretKey, userID)
	if err != nil {
		return fmt.Errorf("ошибка сохранения данных MinIO: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка определения количества обновленных строк: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("пользователь с ID %d не найден", userID)
	}

	return nil
}

// GetMinIOCredentials возвращает учетные данные MinIO для пользователя
func (s *StorageDB) GetMinIOCredentials(userID int) (string, string, string, error) {
	var bucketName, accessKey, secretKey string
	err := s.db.QueryRow(getMinioCredsSQL, userID).Scan(&bucketName, &accessKey, &secretKey)
	if err != nil {
		return "", "", "", fmt.Errorf("ошибка получения данных MinIO: %w", err)
	}

	// Проверка наличия данных
	if bucketName == "" || accessKey == "" || secretKey == "" {
		return "", "", "", fmt.Errorf("для пользователя с ID %d не настроен MinIO", userID)
	}

	return bucketName, accessKey, secretKey, nil
}

// InitDB инициализирует структуру базы данных
func (s *StorageDB) InitDB() error {
	return MigrateDatabase(s.db, -1)
}

// GetCurrentDBVersion возвращает текущую версию базы данных
func (s *StorageDB) GetCurrentDBVersion() (int, error) {
	return getCurrentVersion(s.db)
}

// MigrateTo мигрирует базу данных до указанной версии
func (s *StorageDB) MigrateTo(version int) error {
	return MigrateDatabase(s.db, version)
}
