package storagedb

import "database/sql"

// Migration представляет собой миграцию для обновления структуры БД
type Migration struct {
	Version     int
	Description string
	Up          func(db *sql.DB) error
	Down        func(db *sql.DB) error
}

// migrations.go
var migrations = []Migration{
	{
		Version:     1,
		Description: "Создание таблицы users",
		Up: func(db *sql.DB) error {
			query := `CREATE TABLE IF NOT EXISTS users (
                id SERIAL PRIMARY KEY,
                user_name VARCHAR(100) NOT NULL UNIQUE,
                password_hash VARCHAR(255) NOT NULL,
                email VARCHAR(255) NOT NULL UNIQUE,
                minio_bucket_name VARCHAR(255) NOT NULL UNIQUE,
                created_at TIMESTAMP DEFAULT (now() AT TIME ZONE 'UTC'),
                updated_at TIMESTAMP DEFAULT (now() AT TIME ZONE 'UTC')
            );`
			_, err := db.Exec(query)
			return err
		},
		Down: func(db *sql.DB) error {
			_, err := db.Exec("DROP TABLE IF EXISTS users;")
			return err
		},
	},
	{
		Version:     2,
		Description: "Добавление полей для хранения ключей MinIO",
		Up: func(db *sql.DB) error {
			_, err := db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS minio_access_key VARCHAR(255), ADD COLUMN IF NOT EXISTS minio_secret_key VARCHAR(255);")
			return err
		},
		Down: func(db *sql.DB) error {
			_, err := db.Exec("ALTER TABLE users DROP COLUMN IF EXISTS minio_access_key, DROP COLUMN IF EXISTS minio_secret_key;")
			return err
		},
	},
}
