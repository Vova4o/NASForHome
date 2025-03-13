package storagedb

import (
	"database/sql"
	"fmt"
)

// Получение текущей версии
func getCurrentVersion(db *sql.DB) (int, error) {
	if err := createMigrationsTable(db); err != nil {
		return 0, err
	}

	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения версии: %w", err)
	}
	return version, nil
}

// Обновление версии
func updateVersion(db *sql.DB, version int) error {
	_, err := db.Exec("INSERT INTO migrations (version) VALUES ($1)", version)
	return err
}

// Создание таблицы миграций
func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
    CREATE TABLE IF NOT EXISTS migrations (
        id SERIAL PRIMARY KEY,
        version INT NOT NULL UNIQUE,
        applied_at TIMESTAMP DEFAULT (now() AT TIME ZONE 'UTC')
    );`)
	return err
}
