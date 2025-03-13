package models

import (
	"time"
)

// User структура для хранения данных о пользователе
type User struct {
	ID              int       `db:"id"`
	UserName        string    `db:"user_name"`
	PasswordHash    string    `db:"password_hash"`
	Email           string    `db:"email"`
	MinioBucketName string    `db:"minio_bucket_name"`
	MinioAccessKey  string    `db:"minio_access_key"` // Зашифрованный ключ доступа к MinIO
	MinioSecretKey  string    `db:"minio_secret_key"` // Зашифрованный секретный ключ доступа к MinIO
	CreatedAt       time.Time `db:"created_at"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// MinioConfig structure of minio config
type MinioConfig struct {
	BucketName string
	AccessKey  string
	SecretKey  string
}
